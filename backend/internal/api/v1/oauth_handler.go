package v1

import (
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OAuthHandler struct {
	feishuOAuthSvc service.FeishuOAuthService
	auditRepo      repository.LoginAuditRepository
	settingsRepo   repository.SystemSettingRepository
	userService    service.UserService
}

func NewOAuthHandler(feishuOAuthSvc service.FeishuOAuthService, auditRepo repository.LoginAuditRepository, settingsRepo repository.SystemSettingRepository, userService service.UserService) *OAuthHandler {
	return &OAuthHandler{feishuOAuthSvc: feishuOAuthSvc, auditRepo: auditRepo, settingsRepo: settingsRepo, userService: userService}
}

/*
FeishuAuthorize 返回飞书授权 URL。
前端点击飞书登录按钮后调用此接口，拿到 URL 后跳转到飞书授权页。
*/
func (h *OAuthHandler) FeishuAuthorize(c *gin.Context) {
	clientIP := c.ClientIP()
	authURL, _, err := h.feishuOAuthSvc.BuildAuthorizeURL(clientIP)
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, gin.H{"authorize_url": authURL})
}

/*
FeishuCallback 处理飞书授权回调。
飞书授权完成后浏览器携带 code 和 state 跳转到此接口。
验证通过后生成一次性 login ticket，302 重定向到前端回调页。
*/
func (h *OAuthHandler) FeishuCallback(c *gin.Context) {
	code := c.Query("code")
	stateRaw := c.Query("state")

	if code == "" || stateRaw == "" {
		response.Fail(c, response.CodeInvalidParam, "缺少 code 或 state 参数")
		return
	}

	ticket, err := h.feishuOAuthSvc.HandleCallback(code, stateRaw)
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 302 重定向到前端回调页
	c.Redirect(302, "/auth/feishu/callback?oauth_ticket="+ticket)
}

/*
FeishuExchange 用一次性 login ticket 换取 JWT token pair。
前端回调页拿到 ticket 后调用此接口，成功后获得与密码登录完全一致的 token pair。
*/
func (h *OAuthHandler) FeishuExchange(c *gin.Context) {
	var req struct {
		OAuthTicket string `json:"oauth_ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "缺少 oauth_ticket 参数")
		return
	}

	tokenPair, err := h.feishuOAuthSvc.ExchangeTicket(req.OAuthTicket)
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 记录登录审计
	if h.auditRepo != nil {
		claims, parseErr := jwt.ParseToken(tokenPair.AccessToken)
		if parseErr == nil && claims != nil {
			realName := ""
			if u, err := h.userService.GetByID(claims.UserID); err == nil && u != nil {
				realName = u.RealName
			}
			audit := &model.LoginAudit{
				UserID:       claims.UserID,
				Username:     claims.Username,
				RealName:     realName,
				LoginMethod:  "feishu",
				ClientIP:     c.ClientIP(),
				UserAgent:    c.GetHeader("User-Agent"),
				Success:      true,
				ErrorMessage: "",
			}
			if err := h.auditRepo.Create(audit); err != nil {
				logger.Log.Warn("record feishu login audit failed", zap.Error(err))
			}
		}
	}

	response.Success(c, jwt.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	})
}

/*
LoginStatus 返回登录页需要的前置信息（无需认证）。
*/
func (h *OAuthHandler) LoginStatus(c *gin.Context) {
	localEnabled := "true"
	if v, err := h.settingsRepo.Get("local_enabled"); err == nil && v != "" {
		localEnabled = v
	}
	response.Success(c, gin.H{
		"local_enabled": localEnabled,
	})
}
