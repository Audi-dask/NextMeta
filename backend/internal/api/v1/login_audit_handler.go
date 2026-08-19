package v1

import (
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

/*
LoginAuditHandler 处理登录审计记录相关的 HTTP 请求。
*/
type LoginAuditHandler struct {
	repo repository.LoginAuditRepository
}

func NewLoginAuditHandler(repo repository.LoginAuditRepository) *LoginAuditHandler {
	return &LoginAuditHandler{repo: repo}
}

/*
List 分页查询登录审计记录。
*/
func (h *LoginAuditHandler) List(c *gin.Context) {
	page := 1
	size := 20

	var req struct {
		Page        int    `form:"page"`
		Size        int    `form:"size"`
		LoginMethod string `form:"login_method"`
		Status      string `form:"status"`
		ClientIP    string `form:"client_ip"`
	}
	if err := c.ShouldBindQuery(&req); err == nil {
		if req.Page > 0 {
			page = req.Page
		}
		if req.Size > 0 && req.Size <= 200 {
			size = req.Size
		}
	}

	offset := (page - 1) * size
	audits, total, err := h.repo.FindAll(offset, size, req.LoginMethod, req.Status, req.ClientIP)
	if err != nil {
		response.Fail(c, response.CodeError, "查询登录审计记录失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": audits,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
