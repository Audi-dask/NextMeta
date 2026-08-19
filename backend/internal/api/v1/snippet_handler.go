package v1

import (
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

/*
SnippetHandler 承接 SQL 片段相关 HTTP 请求。
所有片段操作都基于当前登录用户，只允许用户管理自己的 SQL 片段。
*/
type SnippetHandler struct {
	service service.SnippetService
}

/*
NewSnippetHandler 创建 SQL 片段接口 Handler。
SnippetService 由 main.go 注入，负责片段的持久化和归属校验。
*/
func NewSnippetHandler(service service.SnippetService) *SnippetHandler {
	return &SnippetHandler{service: service}
}

/*
CreateSnippetRequest 是创建 SQL 片段时提交的请求体。
Title 用于列表展示，Content 保存具体 SQL 内容。
*/
type CreateSnippetRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

/*
Create 创建当前登录用户的 SQL 片段。
用户 ID 来自 JWTAuth 写入的上下文，不依赖客户端传入。
*/
func (h *SnippetHandler) Create(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Fail(c, response.CodeUnauthorized, "Unauthorized")
		return
	}

	var req CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	if err := h.service.CreateSnippet(userID.(uint), req.Title, req.Content); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
List 返回当前登录用户的 SQL 片段列表。
service 层只查询当前用户拥有的片段，避免跨用户读取。
*/
func (h *SnippetHandler) List(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Fail(c, response.CodeUnauthorized, "Unauthorized")
		return
	}

	snippets, err := h.service.GetMySnippets(userID.(uint))
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, snippets)
}

/*
UpdateSnippetRequest 是更新 SQL 片段时提交的请求体。
ID 指定要更新的片段，Title 和 Content 覆盖原有内容。
*/
type UpdateSnippetRequest struct {
	ID      uint   `json:"id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

/*
Update 更新当前登录用户的 SQL 片段。
service 层会校验片段归属，防止修改其他用户的片段。
*/
func (h *SnippetHandler) Update(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Fail(c, response.CodeUnauthorized, "Unauthorized")
		return
	}

	var req UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	if err := h.service.UpdateSnippet(userID.(uint), req.ID, req.Title, req.Content); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Delete 删除当前登录用户的 SQL 片段。
片段 ID 来自路径参数，归属校验由 service 层处理。
*/
func (h *SnippetHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Fail(c, response.CodeUnauthorized, "Unauthorized")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "Invalid ID")
		return
	}

	if err := h.service.DeleteSnippet(userID.(uint), uint(id)); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}
