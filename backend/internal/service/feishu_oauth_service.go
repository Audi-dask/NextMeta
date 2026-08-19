package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	feishuAppAccessTokenURL  = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	feishuUserAccessTokenURL = "https://open.feishu.cn/open-apis/authen/v1/oidc/access_token"
	feishuUserInfoURL        = "https://open.feishu.cn/open-apis/authen/v1/user_info"
	feishuAuthorizeURL       = "https://open.feishu.cn/open-apis/authen/v1/authorize"

	stateTTL      = 10 * time.Minute
	ticketTTL     = 2 * time.Minute
	stateByteLen  = 32
	ticketByteLen = 32

	providerFeishu = "feishu"
	purposeLogin   = "login"
)

func init() {
	// Ensure crypto/rand is available
	buf := make([]byte, 1)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
}

type FeishuOAuthService interface {
	BuildAuthorizeURL(clientIP string) (authURL string, stateRaw string, err error)
	HandleCallback(code, stateRaw string) (ticket string, err error)
	ExchangeTicket(ticketRaw string) (*jwt.TokenPair, error)
}

type feishuOAuthService struct {
	feishuConfigRepo repository.FeishuConfigRepository
	stateRepo        repository.OAuthStateRepository
	ticketRepo       repository.OAuthLoginTicketRepository
	bindingRepo      repository.UserOAuthBindingRepository
	userRepo         repository.UserRepository
	httpClient       *http.Client
}

func NewFeishuOAuthService(
	feishuConfigRepo repository.FeishuConfigRepository,
	stateRepo repository.OAuthStateRepository,
	ticketRepo repository.OAuthLoginTicketRepository,
	bindingRepo repository.UserOAuthBindingRepository,
	userRepo repository.UserRepository,
) FeishuOAuthService {
	return &feishuOAuthService{
		feishuConfigRepo: feishuConfigRepo,
		stateRepo:        stateRepo,
		ticketRepo:       ticketRepo,
		bindingRepo:      bindingRepo,
		userRepo:         userRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

/*
BuildAuthorizeURL 生成飞书授权 URL 并保存 state。
返回授权 URL 和明文 state（前端需保存以便回调时使用）。
*/
func (s *feishuOAuthService) BuildAuthorizeURL(clientIP string) (string, string, error) {
	cfg, err := s.feishuConfigRepo.Get()
	if err != nil {
		return "", "", fmt.Errorf("读取飞书配置失败: %w", err)
	}
	if !cfg.Enabled {
		return "", "", errors.New("飞书登录未启用")
	}
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return "", "", errors.New("飞书应用配置未完成，缺少 App ID 或 App Secret")
	}

	stateRaw, err := generateRandom(stateByteLen)
	if err != nil {
		return "", "", fmt.Errorf("生成 state 失败: %w", err)
	}

	state := &model.OAuthState{
		StateHash:   model.HashState(stateRaw),
		Provider:    providerFeishu,
		Purpose:     purposeLogin,
		RedirectURI: "",
		ClientIP:    clientIP,
		ExpiresAt:   time.Now().Add(stateTTL),
	}
	if err := s.stateRepo.Create(state); err != nil {
		return "", "", fmt.Errorf("保存 state 失败: %w", err)
	}

	// GCS：按概率清理过期 state 和 ticket
	if time.Now().UnixNano()%100 < 5 {
		_ = s.stateRepo.DeleteExpired()
		_ = s.ticketRepo.DeleteExpired()
	}

	authURL := fmt.Sprintf("%s?app_id=%s&redirect_uri=%s&state=%s",
		feishuAuthorizeURL, url.QueryEscape(cfg.AppID), url.QueryEscape(cfg.RedirectURI), url.QueryEscape(stateRaw))
	return authURL, stateRaw, nil
}

/*
HandleCallback 处理飞书回调。
1. 消费 state（防 CSRF / 重放）
2. 用 code 换 tokens
3. 获取飞书用户信息
4. 按绑定或邮箱匹配 NextMeta 用户
5. 创建一次性 login ticket
返回明文 ticket。
*/
func (s *feishuOAuthService) HandleCallback(code, stateRaw string) (string, error) {
	// 1. 原子消费 state
	stateHash := model.HashState(stateRaw)
	state, err := s.stateRepo.Consume(stateHash, providerFeishu)
	if err != nil {
		logger.Log.Warn("oauth state consume failed",
			zap.String("state_hash", stateHash[:8]+"..."),
			zap.Error(err))
		return "", errors.New("登录已过期或链接无效，请重新发起飞书登录")
	}
	logger.Log.Info("oauth state consumed",
		zap.Uint("state_id", state.ID),
		zap.String("client_ip", state.ClientIP))

	// 2. 获取 app_access_token
	cfg, err := s.feishuConfigRepo.Get()
	if err != nil {
		return "", fmt.Errorf("读取飞书配置失败: %w", err)
	}

	appToken, err := s.getAppAccessToken(cfg.AppID, cfg.AppSecret)
	if err != nil {
		logger.Log.Error("fetch feishu app token failed", zap.Error(err))
		return "", errors.New("飞书服务暂不可用，请稍后重试")
	}

	// 3. 用 code 换 user_access_token
	userToken, err := s.exchangeCodeForToken(appToken, code)
	if err != nil {
		logger.Log.Error("exchange feishu code for token failed", zap.Error(err))
		return "", errors.New("飞书授权失败，请重新登录")
	}

	// 4. 获取用户信息
	feishuUser, err := s.getUserInfo(userToken)
	if err != nil {
		logger.Log.Error("fetch feishu user info failed", zap.Error(err))
		return "", errors.New("获取飞书用户信息失败")
	}
	logger.Log.Info("feishu user fetched",
		zap.String("union_id", feishuUser.UnionID),
		zap.String("email", feishuUser.Email))

	// 5. 解析 NextMeta 用户
	nextmetaUser, err := s.resolveUser(feishuUser)
	if err != nil {
		logger.Log.Warn("resolve feishu user failed",
			zap.String("email", feishuUser.Email),
			zap.String("union_id", feishuUser.UnionID),
			zap.Error(err))
		return "", err
	}

	// 6. 创建一次性 login ticket
	ticketRaw, err := generateRandom(ticketByteLen)
	if err != nil {
		return "", fmt.Errorf("生成登录票据失败: %w", err)
	}
	ticket := &model.OAuthLoginTicket{
		TicketHash: model.HashTicket(ticketRaw),
		UserID:     nextmetaUser.ID,
		Provider:   providerFeishu,
		ExpiresAt:  time.Now().Add(ticketTTL),
	}
	if err := s.ticketRepo.Create(ticket); err != nil {
		return "", fmt.Errorf("保存登录票据失败: %w", err)
	}

	logger.Log.Info("oauth login ticket created",
		zap.Uint("user_id", nextmetaUser.ID),
		zap.String("username", nextmetaUser.Username))

	return ticketRaw, nil
}

/*
ExchangeTicket 消费一次性 login ticket，签发 JWT token pair。
*/
func (s *feishuOAuthService) ExchangeTicket(ticketRaw string) (*jwt.TokenPair, error) {
	ticketHash := model.HashTicket(ticketRaw)
	ticket, err := s.ticketRepo.Consume(ticketHash)
	if err != nil {
		logger.Log.Warn("oauth ticket consume failed",
			zap.String("ticket_hash", ticketHash[:8]+"..."),
			zap.Error(err))
		return nil, errors.New("登录票据无效或已过期，请重新登录")
	}

	// 校验用户仍有效
	user, err := s.userRepo.FindByID(ticket.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}
	if user.Status != "enabled" {
		logger.Log.Warn("disabled user attempted oauth login",
			zap.Uint("user_id", user.ID),
			zap.String("username", user.Username))
		return nil, errors.New("用户已被禁用，无法登录")
	}

	// 签发 JWT
	tokenPair, err := jwt.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("签发 JWT 失败: %w", err)
	}

	logger.Log.Info("oauth login success",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username))

	return tokenPair, nil
}

// ---- 飞书 API 调用 ----

type feishuAppTokenResponse struct {
	Code           int    `json:"code"`
	Msg            string `json:"msg"`
	AppAccessToken string `json:"app_access_token"`
	Expire         int    `json:"expire"`
}

func (s *feishuOAuthService) getAppAccessToken(appID, appSecret string) (string, error) {
	body := fmt.Sprintf(`{"app_id":"%s","app_secret":"%s"}`, appID, appSecret)
	resp, err := s.httpClient.Post(feishuAppAccessTokenURL, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request feishu app token: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read feishu app token response: %w", err)
	}

	var result feishuAppTokenResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse feishu app token: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu app token error: code=%d msg=%s", result.Code, result.Msg)
	}
	if result.AppAccessToken == "" {
		return "", errors.New("feishu app token empty")
	}
	return result.AppAccessToken, nil
}

type feishuUserAccessTokenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	} `json:"data"`
}

func (s *feishuOAuthService) exchangeCodeForToken(appToken, code string) (string, error) {
	body := fmt.Sprintf(`{"grant_type":"authorization_code","code":"%s"}`, code)
	req, err := http.NewRequest("POST", feishuUserAccessTokenURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var result feishuUserAccessTokenResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("exchange code error: code=%d msg=%s", result.Code, result.Msg)
	}
	if result.Data.AccessToken == "" {
		return "", errors.New("user access token empty")
	}
	return result.Data.AccessToken, nil
}

// FeishuUserInfo 飞书用户信息标准化结构。
type FeishuUserInfo struct {
	UnionID    string `json:"union_id"`
	OpenID     string `json:"open_id"`
	Name       string `json:"name"`
	Nickname   string `json:"nickname"`
	Email      string `json:"email"`
	AvatarURL  string `json:"avatar_url"`
	RawProfile string `json:"raw_profile"`
	Mobile     string `json:"mobile"`
}

type feishuUserInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UnionID         string `json:"union_id"`
		OpenID          string `json:"open_id"`
		Name            string `json:"name"`
		EnName          string `json:"en_name"`
		Nickname        string `json:"nickname"`
		Email           string `json:"email"`
		EnterpriseEmail string `json:"enterprise_email"`
		AvatarURL       string `json:"avatar_url"`
		AvatarThumb     string `json:"avatar_thumb"`
		AvatarMiddle    string `json:"avatar_middle"`
		AvatarBig       string `json:"avatar_big"`
		Mobile          string `json:"mobile"`
	} `json:"data"`
}

func (s *feishuOAuthService) getUserInfo(userAccessToken string) (*FeishuUserInfo, error) {
	req, err := http.NewRequest("GET", feishuUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+userAccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request feishu user info: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read user info response: %w", err)
	}

	var result feishuUserInfoResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("feishu user info error: code=%d msg=%s", result.Code, result.Msg)
	}

	email := result.Data.EnterpriseEmail
	if email == "" {
		email = result.Data.Email
	}

	info := &FeishuUserInfo{
		UnionID:   result.Data.UnionID,
		OpenID:    result.Data.OpenID,
		Name:      result.Data.Name,
		Nickname:  result.Data.Nickname,
		Email:     email,
		AvatarURL: result.Data.AvatarURL,
		Mobile:    result.Data.Mobile,
	}

	// 保存完整原始响应用于审计
	rawJSON, _ := json.Marshal(result.Data)
	info.RawProfile = string(rawJSON)

	return info, nil
}

// ---- 用户解析 ----

func (s *feishuOAuthService) resolveUser(feishuUser *FeishuUserInfo) (*model.User, error) {
	// 1. 已绑定 → 同步资料后登录
	binding, err := s.bindingRepo.FindByProvider(providerFeishu, feishuUser.UnionID)
	if err == nil {
		user, err := s.userRepo.FindByID(binding.UserID)
		if err != nil {
			return nil, errors.New("绑定的用户不存在，请联系管理员")
		}
		if user.Status != "enabled" {
			return nil, errors.New("用户已被禁用，无法登录")
		}
		// 每次登录同步飞书资料（头像、姓名、邮箱）
		s.syncProfile(user, feishuUser)
		return user, nil
	}

	// 2. 未绑定 → 自动创建新用户
	return s.createFeishuUser(feishuUser)
}

/*
syncProfile 同步飞书资料到已有用户（仅填充空字段，不覆盖已有值）。
*/
func (s *feishuOAuthService) syncProfile(user *model.User, feishuUser *FeishuUserInfo) {
	updated := false
	if user.AvatarURL == "" && feishuUser.AvatarURL != "" {
		user.AvatarURL = feishuUser.AvatarURL
		updated = true
	}
	if user.RealName == "" && feishuUser.Name != "" {
		user.RealName = feishuUser.Name
		updated = true
	}
	if user.Email == "" && feishuUser.Email != "" {
		user.Email = feishuUser.Email
		updated = true
	}
	// 旧飞书用户密码为空 → 补生成随机 bcrypt 哈希
	if user.Password == "" || len(user.Password) < 10 {
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err == nil {
			if hashed, err := bcrypt.GenerateFromPassword(randomBytes, bcrypt.DefaultCost); err == nil {
				user.Password = string(hashed)
				updated = true
			}
		}
	}
	if updated {
		if err := s.userRepo.Update(user); err != nil {
			logger.Log.Warn("sync feishu profile failed",
				zap.Uint("user_id", user.ID),
				zap.Error(err))
		}
	}
}

/*
createFeishuUser 用飞书身份创建 NextMeta 用户并自动绑定。
*/
func (s *feishuOAuthService) createFeishuUser(feishuUser *FeishuUserInfo) (*model.User, error) {
	cfg, err := s.feishuConfigRepo.Get()
	if err != nil {
		return nil, fmt.Errorf("读取飞书配置失败: %w", err)
	}

	now := time.Now()

	// 生成 32 字节随机密码再 bcrypt 哈希，飞书用户不通过密码登录
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("生成随机密码失败: %w", err)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(randomBytes, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	user := &model.User{
		Username:    feishuUser.UnionID[:16],
		RealName:    feishuUser.Name,
		Email:       feishuUser.Email,
		AvatarURL:   feishuUser.AvatarURL,
		Password:    string(hashedPassword),
		Role:        cfg.DefaultRole,
		Status:      "enabled",
		Source:      "feishu",
		LastLoginAt: &now,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	logger.Log.Info("feishu user auto-created",
		zap.Uint("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("union_id", feishuUser.UnionID))

	s.autoBind(user, feishuUser)
	return user, nil
}

/*
autoBind 创建 OAuth 绑定记录（幂等，失败不影响登录）。
*/
func (s *feishuOAuthService) autoBind(user *model.User, feishuUser *FeishuUserInfo) {
	binding := &model.UserOAuthBinding{
		UserID:         user.ID,
		Provider:       providerFeishu,
		ProviderUserID: feishuUser.UnionID,
		UnionID:        feishuUser.UnionID,
		OpenID:         feishuUser.OpenID,
		Nickname:       feishuUser.Nickname,
		AvatarURL:      feishuUser.AvatarURL,
		Email:          feishuUser.Email,
		RawProfile:     feishuUser.RawProfile,
	}
	if err := s.bindingRepo.Create(binding); err != nil {
		logger.Log.Warn("auto create binding failed",
			zap.Uint("user_id", user.ID),
			zap.String("union_id", feishuUser.UnionID),
			zap.Error(err))
	}
}

// ---- 工具 ----

func generateRandom(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
