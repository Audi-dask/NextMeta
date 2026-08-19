package v1

import (
	"fmt"
	"io"
	"strings"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/license"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
SystemSettingHandler 承接系统设置相关 HTTP 请求。
当前通过 repository 读写配置项，并通过 notificationSvc 发送测试通知。
*/
type SystemSettingHandler struct {
	repo             repository.SystemSettingRepository
	ldapSvc          service.LDAPService
	ldapSyncSvc      service.LDAPSyncService
	notificationSvc  service.NotificationService
	licenseSvc       service.LicenseService
	ldapConfigRepo   repository.LdapConfigRepository
	feishuConfigRepo repository.FeishuConfigRepository
	userRepo         repository.UserRepository
}

/*
NewSystemSettingHandler 创建系统设置接口 Handler。
repo 负责配置持久化，notificationSvc 负责通知相关配置测试，
licenseSvc 用于授权管理区块的状态查询与上传替换。
*/
func NewSystemSettingHandler(
	repo repository.SystemSettingRepository,
	ldapSvc service.LDAPService,
	ldapSyncSvc service.LDAPSyncService,
	notificationSvc service.NotificationService,
	licenseSvc service.LicenseService,
	ldapConfigRepo repository.LdapConfigRepository,
	feishuConfigRepo repository.FeishuConfigRepository,
	userRepo repository.UserRepository,
) *SystemSettingHandler {
	return &SystemSettingHandler{
		repo:             repo,
		ldapSvc:          ldapSvc,
		ldapSyncSvc:      ldapSyncSvc,
		notificationSvc:  notificationSvc,
		licenseSvc:       licenseSvc,
		ldapConfigRepo:   ldapConfigRepo,
		feishuConfigRepo: feishuConfigRepo,
		userRepo:         userRepo,
	}
}

/*
licenseSnapshotResponse 把 license.Snapshot 转成更适合前端展示的响应体。
时间统一使用本地时区的 RFC3339 字符串，剩余天数和状态字段保持原样直接返回。
*/
type licenseSnapshotResponse struct {
	State         license.State `json:"state"`
	Holder        string        `json:"holder"`
	IssuedAt      string        `json:"issued_at"`
	ExpiresAt     string        `json:"expires_at"`
	RemainingDays int           `json:"remaining_days"`
	Message       string        `json:"message"`
}

func toLicenseSnapshotResponse(snapshot license.Snapshot) licenseSnapshotResponse {
	resp := licenseSnapshotResponse{
		State:         snapshot.State,
		Holder:        snapshot.Holder,
		RemainingDays: snapshot.RemainingDays,
		Message:       snapshot.Message,
	}
	if !snapshot.IssuedAt.IsZero() {
		resp.IssuedAt = snapshot.IssuedAt.Format("2006-01-02 15:04:05")
	}
	if !snapshot.ExpiresAt.IsZero() {
		resp.ExpiresAt = snapshot.ExpiresAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

/*
GetLicense 返回当前 license 状态快照，供前端授权管理区块展示。
该接口不受 license 中间件保护，但要求管理员身份，便于授权失效时管理员仍能查看状态。
*/
func (h *SystemSettingHandler) GetLicense(c *gin.Context) {
	if h.licenseSvc == nil {
		response.Fail(c, response.CodeError, "license service is not initialized")
		return
	}
	response.Success(c, toLicenseSnapshotResponse(h.licenseSvc.Snapshot()))
}

/*
UploadLicense 接收前端通过 multipart/form-data 上传的 license 文件并验签替换。
字段名固定为 file，最大 64KB；验签失败不会覆盖磁盘上的现有 license 文件。
*/
func (h *SystemSettingHandler) UploadLicense(c *gin.Context) {
	if h.licenseSvc == nil {
		response.Fail(c, response.CodeError, "license service is not initialized")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "缺少上传文件，请选择有效的 license 文件")
		return
	}
	// 限制单个 license 文件最大 64KB，正常签发结果通常在数 KB 量级，更大基本是误传。
	if fileHeader.Size > 64*1024 {
		response.Fail(c, response.CodeInvalidParam, "license 文件过大，请确认上传的是 license.lic")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, response.CodeError, "读取上传文件失败："+err.Error())
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, response.CodeError, "读取上传文件失败："+err.Error())
		return
	}

	snapshot, err := h.licenseSvc.Reload(raw)
	if err != nil {
		logger.Log.Warn("upload license rejected", zap.Error(err))
		response.FailWithData(c, response.CodeInvalidParam, "license 校验失败："+err.Error(), toLicenseSnapshotResponse(snapshot))
		return
	}

	response.Success(c, toLicenseSnapshotResponse(snapshot))
}

/*
List 返回全部系统设置。
结果会从配置项列表转换为 key-value map，方便前端按配置 key 直接读取。
*/
func (h *SystemSettingHandler) List(c *gin.Context) {
	settings, err := h.repo.GetAll()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 只返回系统设置页面需要的配置 key
	allowedKeys := map[string]bool{
		"global_sql_limit": true,
		"local_enabled":    true,
	}
	result := make(map[string]interface{})
	for _, s := range settings {
		if allowedKeys[s.Key] {
			result[s.Key] = s.Value
		}
	}

	// 未显式存储时默认为启用
	if _, ok := result["local_enabled"]; !ok {
		result["local_enabled"] = "true"
	}

	// 追加 LDAP/飞书启用状态（实际配置在独立表中）
	if ldapCfg, err := h.ldapConfigRepo.Get(); err == nil {
		result["ldap_enabled"] = fmt.Sprintf("%v", ldapCfg.Enabled)
	}
	if feishuCfg, err := h.feishuConfigRepo.Get(); err == nil {
		result["feishu_enabled"] = fmt.Sprintf("%v", feishuCfg.Enabled)
	}

	response.Success(c, result)
}

/*
ListNotifications 返回通知相关配置。
仅包含 notification_ 前缀的 key，避免把系统设置、LDAP/飞书等无关数据返回给通知页面。
*/
func (h *SystemSettingHandler) ListNotifications(c *gin.Context) {
	settings, err := h.repo.GetAll()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	result := make(map[string]interface{})
	for _, s := range settings {
		if strings.HasPrefix(s.Key, "notification_") {
			result[s.Key] = s.Value
		}
	}

	response.Success(c, result)
}

/*
Update 批量更新系统设置。
请求体按 key-value 接收，所有值最终转换为字符串后写入配置表。
禁用本地登录时校验至少有一个外部登录方式已启用。
*/
func (h *SystemSettingHandler) Update(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	for k, v := range req {
		var strVal string
		switch val := v.(type) {
		case string:
			strVal = val
		case float64:
			strVal = fmt.Sprintf("%.0f", val)
		default:
			strVal = fmt.Sprintf("%v", val)
		}

		if k == "local_enabled" && strVal == "false" {
			// 校验：关闭本地登录前确保至少有一个外部超级管理员
			users, err := h.userRepo.FindAll()
			if err != nil {
				response.Fail(c, response.CodeError, "无法查询用户列表: "+err.Error())
				return
			}
			hasExternalSuperAdmin := false
			for _, u := range users {
				if u.Role == "super_admin" && u.Source != "local" {
					hasExternalSuperAdmin = true
					break
				}
			}
			if !hasExternalSuperAdmin {
				response.Fail(c, response.CodeInvalidParam, "没有通过 AD 或飞书登录的超级管理员用户，无法关闭本地登录。请先将某个外部用户提升为超级管理员。")
				return
			}
		}

		if err := h.repo.Set(k, strVal); err != nil {
			response.Fail(c, response.CodeError, err.Error())
			return
		}
	}

	response.Success(c, nil)
}

/*
TestNotify 向请求体中的 webhook 发送一条测试通知。
该接口用于系统设置页面验证通知配置是否可用，不会修改配置表。
*/
func (h *SystemSettingHandler) TestNotify(c *gin.Context) {
	var req struct {
		Webhook string `json:"webhook" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	err := h.notificationSvc.SendDirectNotification(req.Webhook, "【NextMeta】系统通知测试\n如果你收到这条消息，说明当前 Webhook 配置可用。")
	if err != nil {
		response.Fail(c, response.CodeError, fmt.Sprintf("Failed to send notification: %v", err))
		return
	}

	response.Success(c, nil)
}

/*
TestLDAP 使用当前表单中的过滤规则和字段映射测试 LDAP 查询效果。
固定连接参数来自 config.yaml，成功时返回用户列表以及这些用户所属的组。
*/
func (h *SystemSettingHandler) TestLDAP(c *gin.Context) {
	var req dto.LDAPTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	users, err := h.ldapSvc.Test(&req)
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, service.MapLDAPPreviewUsers(users))
}

/*
SyncLDAP 立即执行一次 LDAP 到本地缓存的同步。
同步只处理 LDAP 来源的用户和组，本地用户不受影响。
*/
func (h *SystemSettingHandler) SyncLDAP(c *gin.Context) {
	result, err := h.ldapSyncSvc.SyncNow()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, dto.LDAPSyncResponse{
		CreatedUsers:      result.CreatedUsers,
		UpdatedUsers:      result.UpdatedUsers,
		DisabledUsers:     result.DisabledUsers,
		CreatedGroups:     result.CreatedGroups,
		UpdatedGroups:     result.UpdatedGroups,
		DisabledGroups:    result.DisabledGroups,
		SyncedMemberships: result.SyncedMemberships,
	})
}

/*
GetLdapConfig 返回 ldap_config 表的当前配置。
BindPass 不会返回给前端。
*/
func (h *SystemSettingHandler) GetLdapConfig(c *gin.Context) {
	cfg, err := h.ldapConfigRepo.Get()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}
	cfg.BindPassConfigured = cfg.BindPass != ""
	response.Success(c, cfg)
}

/*
UpdateLdapConfig 更新 ldap_config 表。
若前端未传入 bind_pass（留空），则保留数据库中的现有值不覆盖。
*/
func (h *SystemSettingHandler) UpdateLdapConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	cfg, err := h.ldapConfigRepo.Get()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if v, ok := req["enabled"]; ok {
		if s, ok2 := v.(string); ok2 {
			cfg.Enabled = s == "true"
		} else if b, ok2 := v.(bool); ok2 {
			cfg.Enabled = b
		}
	}
	if v, ok := req["url"]; ok {
		cfg.URL = fmt.Sprintf("%v", v)
	}
	if v, ok := req["base_dn"]; ok {
		cfg.BaseDN = fmt.Sprintf("%v", v)
	}
	if v, ok := req["group_base_dn"]; ok {
		cfg.GroupBaseDN = fmt.Sprintf("%v", v)
	}
	if v, ok := req["bind_dn"]; ok {
		cfg.BindDN = fmt.Sprintf("%v", v)
	}
	if v, ok := req["bind_pass"]; ok {
		s := fmt.Sprintf("%v", v)
		if strings.TrimSpace(s) != "" {
			cfg.BindPass = s
		}
	}
	if v, ok := req["user_filter"]; ok {
		cfg.UserFilter = fmt.Sprintf("%v", v)
	}
	if v, ok := req["group_filter"]; ok {
		cfg.GroupFilter = fmt.Sprintf("%v", v)
	}
	if v, ok := req["mapping_username"]; ok {
		cfg.MappingUsername = fmt.Sprintf("%v", v)
	}
	if v, ok := req["mapping_real_name"]; ok {
		cfg.MappingRealName = fmt.Sprintf("%v", v)
	}
	if v, ok := req["mapping_email"]; ok {
		cfg.MappingEmail = fmt.Sprintf("%v", v)
	}
	if v, ok := req["sync_interval"]; ok {
		switch val := v.(type) {
		case float64:
			cfg.SyncInterval = int(val)
		case string:
			cfg.SyncInterval = parseToInt(val)
		}
	}
	if v, ok := req["exclude_keywords"]; ok {
		cfg.ExcludeKeywords = fmt.Sprintf("%v", v)
	}

	if err := h.ldapConfigRepo.Save(cfg); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

/*
GetFeishuConfig 返回 feishu_config 表的当前配置。
AppSecret 不会返回给前端。
*/
func (h *SystemSettingHandler) GetFeishuConfig(c *gin.Context) {
	cfg, err := h.feishuConfigRepo.Get()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}
	cfg.AppSecretConfigured = cfg.AppSecret != ""
	response.Success(c, cfg)
}

/*
UpdateFeishuConfig 更新 feishu_config 表。
若前端未传入 app_secret（留空），则保留数据库中的现有值不覆盖。
*/
func (h *SystemSettingHandler) UpdateFeishuConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	cfg, err := h.feishuConfigRepo.Get()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if v, ok := req["enabled"]; ok {
		if s, ok2 := v.(string); ok2 {
			cfg.Enabled = s == "true"
		} else if b, ok2 := v.(bool); ok2 {
			cfg.Enabled = b
		}
	}
	if v, ok := req["app_id"]; ok {
		cfg.AppID = fmt.Sprintf("%v", v)
	}
	if v, ok := req["app_secret"]; ok {
		s := fmt.Sprintf("%v", v)
		if strings.TrimSpace(s) != "" {
			cfg.AppSecret = s
		}
	}
	if v, ok := req["redirect_uri"]; ok {
		cfg.RedirectURI = fmt.Sprintf("%v", v)
	}
	if v, ok := req["default_role"]; ok {
		cfg.DefaultRole = fmt.Sprintf("%v", v)
	}

	if err := h.feishuConfigRepo.Save(cfg); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}
	response.Success(c, nil)
}

func parseToInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
