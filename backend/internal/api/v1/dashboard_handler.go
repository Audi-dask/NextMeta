package v1

import (
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

/*
DashboardHandler 承接首页看板相关 HTTP 请求。
它通过 DashboardService 聚合系统概览统计和最近工单数据。
*/
type DashboardHandler struct {
	dashboardService service.DashboardService
}

/*
NewDashboardHandler 创建首页看板接口 Handler。
dashboardService 由 main.go 注入，负责具体统计数据聚合。
*/
func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

/*
GetStats 返回首页看板统计数据。
当前首页只需要 stats 概览数据，不返回最近工单明细，避免把无用字段和敏感信息暴露给前端。
*/
func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.dashboardService.GetStats()
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{"stats": stats})
}
