package v1

import (
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
GroupHandler 承接用户组相关 HTTP 请求。
Handler 层只负责参数绑定、路径参数解析、调用 groupService 和输出统一响应。
*/
type GroupHandler struct {
	groupService service.GroupService
}

/*
NewGroupHandler 创建用户组接口 Handler。
groupService 由 main.go 注入，实际用户组业务规则保留在 service 层。
*/
func NewGroupHandler(groupService service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

/*
ListAll 返回全部用户组列表。
该接口用于用户组管理页面展示，由路由层负责登录鉴权。
*/
func (h *GroupHandler) ListAll(c *gin.Context) {
	groups, err := h.groupService.ListAll()
	if err != nil {
		logger.Log.Error("Failed to list groups", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to fetch groups")
		return
	}
	response.Success(c, groups)
}

/*
Create 处理管理员创建用户组请求。
请求体中的 reviewerIds 和 memberIds 会交给 service 层同步维护审批人和成员关联。
*/
func (h *GroupHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code"`
		Status      string `json:"status"`
		Description string `json:"description"`
		ReviewerIDs []uint `json:"reviewerIds"`
		MemberIDs   []uint `json:"memberIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.groupService.Create(req.Name, req.Code, req.Status, req.Description, req.ReviewerIDs, req.MemberIDs); err != nil {
		logger.Log.Error("Failed to create group", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to create group: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Update 处理管理员更新用户组请求。
除基础信息外，成员和审批人列表也会按请求体内容交给 service 层更新。
*/
func (h *GroupHandler) Update(c *gin.Context) {
	var req struct {
		ID          uint   `json:"id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code"`
		Status      string `json:"status"`
		Description string `json:"description"`
		ReviewerIDs []uint `json:"reviewerIds"`
		MemberIDs   []uint `json:"memberIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.groupService.Update(req.ID, req.Name, req.Code, req.Status, req.Description, req.ReviewerIDs, req.MemberIDs); err != nil {
		logger.Log.Error("Failed to update group", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to update group: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Delete 处理管理员删除用户组请求。
用户组 ID 来自路径参数，解析成功后交给 service 层处理删除和关联清理。
*/
func (h *GroupHandler) Delete(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, response.CodeError, "invalid group id")
		return
	}

	if err := h.groupService.Delete(uint(groupID)); err != nil {
		logger.Log.Error("Failed to delete group", zap.Error(err))
		response.Fail(c, response.CodeError, "Failed to delete group: "+err.Error())
		return
	}

	response.Success(c, nil)
}
