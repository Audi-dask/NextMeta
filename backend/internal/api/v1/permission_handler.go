package v1

import (
	"encoding/json"
	"fmt"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
PermissionHandler 承接用户组权限关系相关 HTTP 请求。
当前直接依赖 PermissionRepository 维护组成员、组数据源和组审批人关联。
*/
type PermissionHandler struct {
	permRepo repository.PermissionRepository
}

type groupDataSourceResponse struct {
	ID          uint   `json:"ID"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func toGroupDataSourceResponses(datasources []model.DataSource) []groupDataSourceResponse {
	items := make([]groupDataSourceResponse, 0, len(datasources))
	for _, ds := range datasources {
		items = append(items, groupDataSourceResponse{
			ID:          ds.ID,
			Name:        ds.Name,
			Type:        ds.Type,
			Environment: ds.Environment,
			Status:      ds.Status,
			Description: ds.Description,
		})
	}
	return items
}

/*
flexibleUintIDs 用于兼容前端批量提交的数据源 ID 列表。
它允许数组元素是 JSON number 或 string，统一转换为 uint 切片。
*/
type flexibleUintIDs []uint

/*
UnmarshalJSON 自定义解析数据源 ID 列表。
解析时会拒绝非正整数、无法转成 uint 的字符串，以及非数字/字符串类型。
*/
func (ids *flexibleUintIDs) UnmarshalJSON(data []byte) error {
	var values []any
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	result := make([]uint, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case float64:
			if v <= 0 || v != float64(uint(v)) {
				return fmt.Errorf("invalid datasourceIds value %v", v)
			}
			result = append(result, uint(v))
		case string:
			id, err := strconv.ParseUint(v, 10, 32)
			if err != nil || id == 0 {
				return fmt.Errorf("invalid datasourceIds value %q", v)
			}
			result = append(result, uint(id))
		default:
			return fmt.Errorf("invalid datasourceIds value type")
		}
	}

	*ids = result
	return nil
}

/*
NewPermissionHandler 创建权限关系接口 Handler。
permRepo 由 main.go 注入，用于直接操作用户组关联关系。
*/
func NewPermissionHandler(permRepo repository.PermissionRepository) *PermissionHandler {
	return &PermissionHandler{permRepo: permRepo}
}

/*
AddGroupMember 向指定用户组添加成员。
用户组 ID 来自路径参数，用户 ID 来自请求体。
*/
func (h *PermissionHandler) AddGroupMember(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.permRepo.AddUserToGroup(req.UserID, uint(groupID)); err != nil {
		logger.Log.Error("Failed to add group member", zap.Error(err))
		response.Fail(c, response.CodeError, "添加成员失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
RemoveGroupMember 从指定用户组移除成员。
用户组 ID 和用户 ID 都来自路径参数。
*/
func (h *PermissionHandler) RemoveGroupMember(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID, _ := strconv.ParseUint(c.Param("userId"), 10, 32)

	if err := h.permRepo.RemoveUserFromGroup(uint(userID), uint(groupID)); err != nil {
		logger.Log.Error("Failed to remove group member", zap.Error(err))
		response.Fail(c, response.CodeError, "移除成员失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
GetGroupMembers 返回指定用户组的成员列表。
该接口供用户组详情或权限配置页面展示成员关系。
*/
func (h *PermissionHandler) GetGroupMembers(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	members, err := h.permRepo.GetGroupMembers(uint(groupID))
	if err != nil {
		logger.Log.Error("Failed to get group members", zap.Error(err))
		response.Fail(c, response.CodeError, "获取成员失败")
		return
	}

	response.Success(c, toUserResponses(members))
}

/*
AddGroupDataSource 为用户组授权数据源。
请求体支持单个 datasource_id 追加授权，也支持 datasourceIds 批量替换该组的数据源授权。
*/
func (h *PermissionHandler) AddGroupDataSource(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, response.CodeError, "invalid group id")
		return
	}

	var req struct {
		DataSourceID  uint            `json:"datasource_id"`
		DataSourceIDs flexibleUintIDs `json:"datasourceIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if req.DataSourceIDs != nil {
		if err := h.permRepo.ReplaceGroupDataSources(uint(groupID), req.DataSourceIDs); err != nil {
			logger.Log.Error("Failed to replace group datasources", zap.Error(err))
			response.Fail(c, response.CodeError, "授权数据源失败: "+err.Error())
			return
		}
		response.Success(c, nil)
		return
	}

	if req.DataSourceID == 0 {
		response.Fail(c, response.CodeError, "datasource_id or datasourceIds is required")
		return
	}

	if err := h.permRepo.AddDataSourceToGroup(uint(groupID), req.DataSourceID); err != nil {
		logger.Log.Error("Failed to add datasource to group", zap.Error(err))
		response.Fail(c, response.CodeError, "授权数据源失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
RemoveGroupDataSource 取消用户组对指定数据源的授权。
用户组 ID 和数据源 ID 都来自路径参数。
*/
func (h *PermissionHandler) RemoveGroupDataSource(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	dsID, _ := strconv.ParseUint(c.Param("dsId"), 10, 32)

	if err := h.permRepo.RemoveDataSourceFromGroup(uint(groupID), uint(dsID)); err != nil {
		logger.Log.Error("Failed to remove datasource from group", zap.Error(err))
		response.Fail(c, response.CodeError, "取消授权失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
GetGroupDataSources 返回指定用户组已授权的数据源列表。
失败时会记录用户组 ID 和底层错误，便于排查权限配置问题。
*/
func (h *PermissionHandler) GetGroupDataSources(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	datasources, err := h.permRepo.GetGroupDataSources(uint(groupID))
	if err != nil {
		logger.Log.Error("Failed to get group datasources",
			zap.Error(err),
			zap.Uint("groupID", uint(groupID)),
			zap.String("error_detail", err.Error()))
		response.Fail(c, response.CodeError, "获取数据源失败: "+err.Error())
		return
	}

	response.Success(c, toGroupDataSourceResponses(datasources))
}

/*
AddGroupApprover 为指定用户组添加审批人。
审批人用于工单审批流，用户 ID 来自请求体。
*/
func (h *PermissionHandler) AddGroupApprover(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.permRepo.AddApprover(uint(groupID), req.UserID); err != nil {
		logger.Log.Error("Failed to add approver", zap.Error(err))
		response.Fail(c, response.CodeError, "添加审批人失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
RemoveGroupApprover 从指定用户组移除审批人。
移除前会先查询当前审批人数量，确保用户组至少保留一个审核人。
*/
func (h *PermissionHandler) RemoveGroupApprover(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID, _ := strconv.ParseUint(c.Param("userId"), 10, 32)
	approvers, err := h.permRepo.GetGroupApprovers(uint(groupID))
	if err != nil {
		logger.Log.Error("Failed to get group approvers", zap.Error(err))
		response.Fail(c, response.CodeError, "获取审批人失败")
		return
	}
	if len(approvers) <= 1 {
		response.Fail(c, response.CodeError, "用户组至少需要保留一个审核人")
		return
	}

	if err := h.permRepo.RemoveApprover(uint(groupID), uint(userID)); err != nil {
		logger.Log.Error("Failed to remove approver", zap.Error(err))
		response.Fail(c, response.CodeError, "移除审批人失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
GetUserAccessibleDataSources 返回当前登录用户可访问的数据源列表。
它先读取用户所属用户组，再聚合这些组已授权的数据源，并按数据源 ID 去重。
*/
func (h *PermissionHandler) GetUserAccessibleDataSources(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)

	// 先获取当前用户所属用户组，后续权限范围来自这些用户组。
	groups, err := h.permRepo.GetUserGroups(userID)
	if err != nil {
		logger.Log.Error("Failed to get user groups", zap.Error(err))
		response.Fail(c, response.CodeError, "获取用户组失败")
		return
	}

	// 聚合所有用户组的数据源授权，并用 seenIDs 避免重复返回同一数据源。
	type DataSourceDTO struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Environment string `json:"environment"`
		AccessMode  string `json:"accessMode"`
	}

	finalDataSources := make([]DataSourceDTO, 0)
	seenIDs := make(map[uint]bool)

	for _, group := range groups {
		datasources, err := h.permRepo.GetGroupDataSources(group.ID)
		if err != nil {
			continue
		}
		for _, ds := range datasources {
			if !seenIDs[ds.ID] {
				seenIDs[ds.ID] = true
				finalDataSources = append(finalDataSources, DataSourceDTO{
					ID:          ds.ID,
					Name:        ds.Name,
					Type:        ds.Type,
					Environment: ds.Environment,
					AccessMode:  ds.AccessMode,
				})
			}
		}
	}

	response.Success(c, finalDataSources)
}

/*
GetGroupApprovers 返回指定用户组的审批人列表。
该接口供权限配置页面展示和维护工单审批人使用。
*/
func (h *PermissionHandler) GetGroupApprovers(c *gin.Context) {
	groupID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	approvers, err := h.permRepo.GetGroupApprovers(uint(groupID))
	if err != nil {
		logger.Log.Error("Failed to get group approvers", zap.Error(err))
		response.Fail(c, response.CodeError, "获取审批人失败")
		return
	}

	type groupApproverResponse struct {
		ID       uint   `json:"ID"`
		Username string `json:"Username"`
		RealName string `json:"RealName"`
		Role     string `json:"role,omitempty"`
	}

	items := make([]groupApproverResponse, 0, len(approvers))
	for _, approver := range approvers {
		items = append(items, groupApproverResponse{
			ID:       approver.ID,
			Username: approver.Username,
			RealName: approver.RealName,
			Role:     approver.Role,
		})
	}

	response.Success(c, items)
}
