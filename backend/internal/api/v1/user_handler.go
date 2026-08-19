package v1

import (
	"net/http"
	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
UserHandler 承接用户相关 HTTP 请求。
它只负责参数绑定、上下文用户信息读取、调用 userService 和统一响应输出。
*/
type UserHandler struct {
	userService    service.UserService
	licenseService service.LicenseService
	auditRepo      repository.LoginAuditRepository
	settingsRepo   repository.SystemSettingRepository
}

type userResponse struct {
	ID          uint        `json:"ID"`
	Username    string      `json:"Username"`
	RealName    string      `json:"RealName"`
	Email       string      `json:"Email,omitempty"`
	AvatarURL   string      `json:"avatar_url,omitempty"`
	Role        string      `json:"role,omitempty"`
	Status      string      `json:"status,omitempty"`
	Source      string      `json:"Source,omitempty"`
	CreatedAt   interface{} `json:"CreatedAt,omitempty"`
	LastLoginAt interface{} `json:"lastLoginAt,omitempty"`
	IsApprover  bool        `json:"is_approver"`
}

func toUserResponse(user model.User) userResponse {
	return userResponse{
		ID:          user.ID,
		Username:    user.Username,
		RealName:    user.RealName,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		Role:        user.Role,
		Status:      user.Status,
		Source:      user.Source,
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
		IsApprover:  user.IsApprover,
	}
}

func toUserResponses(users []model.User) []userResponse {
	items := make([]userResponse, 0, len(users))
	for _, user := range users {
		items = append(items, toUserResponse(user))
	}
	return items
}

/*
NewUserHandler 创建用户接口 Handler。
userService 由 main.go 完成依赖注入，便于 Handler 层保持轻量；
licenseService 在登录成功后判定当前授权状态是否允许该角色登录。
*/
func NewUserHandler(userService service.UserService, licenseService service.LicenseService, auditRepo repository.LoginAuditRepository, settingsRepo repository.SystemSettingRepository) *UserHandler {
	return &UserHandler{userService: userService, licenseService: licenseService, auditRepo: auditRepo, settingsRepo: settingsRepo}
}

/*
Register 处理管理员创建注册用户请求。
请求体校验通过后交给 userService.Register 完成用户创建。
*/
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.userService.Register(&req); err != nil {
		logger.Log.Error("Register failed", zap.Error(err))
		response.Fail(c, response.CodeError, "Registration failed: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Login 处理用户登录请求。
认证成功后返回 access token 和 refresh token，失败时不向前端暴露具体认证细节。
*/
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 本地登录已停用时拒绝
	if local, err := h.settingsRepo.Get("local_enabled"); err == nil && local == "false" {
		response.Fail(c, response.CodeInvalidParam, "本地登录已停用，请使用 AD 或飞书登录")
		return
	}

	tokens, role, userID, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		h.recordLoginAudit(c, 0, req.Username, "local", false, err.Error())
		logger.Log.Warn("Login failed", zap.String("username", req.Username), zap.Error(err))
		response.Fail(c, response.CodeError, "Login failed")
		return
	}

	if h.licenseService != nil && !h.licenseService.AllowLogin(role) {
		snapshot := h.licenseService.Snapshot()
		h.recordLoginAudit(c, userID, req.Username, "local", false, "license: "+string(snapshot.State))
		logger.Log.Warn("login blocked by license",
			zap.String("username", req.Username),
			zap.String("role", role),
			zap.String("state", string(snapshot.State)),
		)
		response.FailWithStatus(c, http.StatusForbidden, response.CodeLicenseInvalid,
			"系统授权无效（"+string(snapshot.State)+"），请联系管理员上传授权文件")
		return
	}

	h.recordLoginAudit(c, userID, req.Username, "local", true, "")
	response.Success(c, dto.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *UserHandler) recordLoginAudit(c *gin.Context, userID uint, username, method string, success bool, errMsg string) {
	if h.auditRepo == nil {
		return
	}
	audit := &model.LoginAudit{
		UserID:       userID,
		Username:     username,
		LoginMethod:  method,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Success:      success,
		ErrorMessage: errMsg,
	}
	if success && userID > 0 {
		if u, err := h.userService.GetByID(userID); err == nil && u != nil {
			audit.RealName = u.RealName
		}
	}
	if err := h.auditRepo.Create(audit); err != nil {
		logger.Log.Warn("record login audit failed", zap.Error(err))
	}
}

/*
RefreshToken 使用 refresh token 换取新的 token pair。
refresh token 无效或过期时返回 401，前端应重新登录。
*/
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	tokens, err := jwt.RefreshToken(req.RefreshToken)
	if err != nil {
		response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Invalid or expired refresh token")
		return
	}

	response.Success(c, dto.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

// @Summary Get user profile
// @Description Get the profile of the currently logged-in user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /users/profile [get]
/*
GetProfile 返回当前登录用户资料。
用户 ID 来自 JWTAuth 写入的上下文，返回前会清空密码字段，并尽量补充是否审批人标记。
*/
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Unauthorized")
		return
	}

	// JWT 解析后通常写入 uint，这里兼容历史链路中可能出现的 float64。
	uid, ok := userID.(uint)
	if !ok {
		if fId, ok := userID.(float64); ok {
			uid = uint(fId)
		} else {
			response.Fail(c, response.CodeError, "Invalid user ID")
			return
		}
	}

	user, err := h.userService.GetByID(uid)
	if err != nil {
		logger.Log.Error("Failed to get user profile", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to fetch profile")
		return
	}

	// 审批人标记查询失败不阻断个人资料返回。
	isApprover, err := h.userService.IsApprover(user.ID)
	if err == nil {
		user.IsApprover = isApprover
	}

	response.Success(c, toUserResponse(*user))
}

/*
ChangePassword 修改当前登录用户的密码。
用户 ID 取自 JWT 写入的上下文，请求体只携带旧密码与新密码，从协议层封死越权改他人密码的可能性。
*/
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		response.FailWithStatus(c, http.StatusUnauthorized, response.CodeError, "Unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	if err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		logger.Log.Warn("Change password failed", zap.Uint("user_id", userID), zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
UpdateProfile 更新当前登录用户的个人资料。
用户 ID 来自 JWTAuth 写入的上下文，避免客户端通过请求体修改其他用户。
*/
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	if err := h.userService.UpdateProfile(userID, &req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
ListLocalUsers 返回本地用户列表。
该接口由路由层限制为管理员访问，用于用户管理页面展示。
*/
func (h *UserHandler) ListLocalUsers(c *gin.Context) {
	users, err := h.userService.ListLocalUsers()
	if err != nil {
		logger.Log.Error("Failed to list local users", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to fetch users: "+err.Error())
		return
	}
	response.Success(c, toUserResponses(users))
}

/*
CreateUser 处理管理员创建本地用户请求。
请求体校验通过后由 service 负责账号唯一性、密码处理和落库。
*/
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.userService.CreateLocalUser(&req); err != nil {
		logger.Log.Error("Failed to create user", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
UpdateUser 处理用户资料、角色、状态或密码更新请求。
具体权限边界由路由层和 service 层共同保证。
*/
func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	currentUserID := jwt.GetUserIDFromContext(c)
	currentRole := jwt.GetRoleFromContext(c)

	// 只有超级管理员才能授予超级管理员角色
	if req.Role == string(model.UserRoleSuperAdmin) && currentRole != string(model.UserRoleSuperAdmin) {
		response.Fail(c, response.CodeError, "仅超级管理员可授予超级管理员角色")
		return
	}

	// 禁止修改自己的角色（防止提权）
	if req.ID == currentUserID && req.Role != "" {
		response.Fail(c, response.CodeError, "不允许修改自己的角色，请联系其他超级管理员操作")
		return
	}

	if err := h.userService.UpdateUser(&req); err != nil {
		logger.Log.Error("Failed to update user", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to update user: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
DeleteUser 处理管理员删除本地用户请求。
这里只绑定用户 ID，实际删除限制和关联检查交给 userService 处理。
*/
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.userService.DeleteUser(req.ID); err != nil {
		logger.Log.Error("Failed to delete user", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to delete user: "+err.Error())
		return
	}

	response.Success(c, nil)
}
