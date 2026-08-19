package v1

import (
	"fmt"
	"strconv"
	"strings"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
DataSourceHandler 承接数据源相关 HTTP 请求。
它组合数据源服务、权限服务和审计日志服务，覆盖数据源管理、连接测试、元数据读取和查询窗口执行。
*/
type DataSourceHandler struct {
	service     service.DataSourceService
	permSvc     service.PermissionService
	auditSvc    service.AuditService
	auditLogSvc service.AuditLogService
}

/*
NewDataSourceHandler 创建数据源接口 Handler。
各 service 由 main.go 注入，Handler 层负责组织接口入口和统一响应。
*/
func NewDataSourceHandler(service service.DataSourceService, permSvc service.PermissionService, auditSvc service.AuditService, auditLogSvc service.AuditLogService) *DataSourceHandler {
	return &DataSourceHandler{service: service, permSvc: permSvc, auditSvc: auditSvc, auditLogSvc: auditLogSvc}
}

/*
Create 处理管理员创建数据源请求。
请求体包含连接信息、超时配置和脱敏规则，实际校验与落库由 service 层完成。
*/
func (h *DataSourceHandler) Create(c *gin.Context) {
	var req dto.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.service.Create(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Update 处理管理员更新数据源请求。
Password 为空时是否保留原密码等细节由 service 层处理。
*/
func (h *DataSourceHandler) Update(c *gin.Context) {
	var req dto.UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if err := h.service.Update(&req); err != nil {
		response.Fail(c, response.CodeError, "Failed to update data source: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Delete 处理管理员删除数据源请求。
数据源 ID 来自路径参数，解析成功后交给 service 层执行删除。
*/
func (h *DataSourceHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.Fail(c, response.CodeError, "Failed to delete data source: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
Copy 处理管理员复制数据源请求。(这个前端好像没用上，待定位)
它基于路径中的数据源 ID 调用 service 层复制现有配置。
*/
func (h *DataSourceHandler) Copy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	if err := h.service.Copy(uint(id)); err != nil {
		response.Fail(c, response.CodeError, "Failed to copy data source: "+err.Error())
		return
	}

	response.Success(c, nil)
}

/*
List 返回数据源列表。
service 层会转换响应结构，避免直接向前端返回真实数据库密码。
*/
func (h *DataSourceHandler) List(c *gin.Context) {
	list, err := h.service.List()
	if err != nil {
		response.Fail(c, response.CodeError, "Failed to fetch data sources: "+err.Error())
		return
	}

	response.Success(c, list)
}

/*
TestConnection 测试已保存数据源的连接可用性。
成功时返回数据库版本信息，失败时返回连接错误。
*/
func (h *DataSourceHandler) TestConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	version, err := h.service.TestConnection(uint(id))
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{"version": version})
}

/*
TestConnectionConfig 测试尚未保存的数据源连接配置。
该接口只使用请求体中的临时连接参数，不会创建或更新数据源记录。
*/
func (h *DataSourceHandler) TestConnectionConfig(c *gin.Context) {
	var req dto.TestDataSourceConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	version, err := h.service.TestConnectionConfig(&req)
	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, gin.H{"version": version})
}

/*
FetchSchemas 获取数据源下的库表结构树。
refresh=true 时由 service 层决定是否绕过缓存重新读取元数据。
*/
func (h *DataSourceHandler) FetchSchemas(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	refresh := c.Query("refresh") == "true"
	nodes, err := h.service.FetchSchemas(uint(id), refresh)
	if err != nil {
		response.Fail(c, response.CodeError, "Failed to fetch schemas: "+err.Error())
		return
	}

	response.Success(c, nodes)
}

/*
FetchColumns 获取指定库表的列信息。
db 和 table 查询参数都必填，返回结构用于前端 SQL 编辑和工单页面展示字段树。
*/
func (h *DataSourceHandler) FetchColumns(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	dbName := c.Query("db")
	tableName := c.Query("table")

	if dbName == "" || tableName == "" {
		response.Fail(c, response.CodeError, "db and table params are required")
		return
	}

	nodes, err := h.service.FetchColumns(uint(id), dbName, tableName)
	if err != nil {
		response.Fail(c, response.CodeError, "Failed to fetch columns: "+err.Error())
		return
	}

	response.Success(c, nodes)
}

/*
ExecuteSQL 处理查询窗口的 SQL 执行请求。
它先校验当前用户是否有数据源访问权限，再限制只能执行 SELECT，最后注入执行人注释并记录查询审计日志。
*/
func (h *DataSourceHandler) ExecuteSQL(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, response.CodeError, "Invalid ID")
		return
	}

	var req dto.ExecuteSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 查询窗口必须先校验当前用户是否有该数据源访问权限。
	userID := jwt.GetUserIDFromContext(c)
	canAccess, err := h.permSvc.CanAccessDataSource(userID, uint(id))
	if err != nil {
		response.Fail(c, response.CodeError, "权限校验失败: "+err.Error())
		return
	}
	if !canAccess {
		response.Fail(c, response.CodeError, "您没有该数据源的访问权限")
		return
	}

	// 查询窗口只允许只读 SELECT / EXPLAIN，变更类 SQL 必须走工单流程。
	sqlLower := strings.ToLower(strings.TrimSpace(req.SQL))
	if !strings.HasPrefix(sqlLower, "select") && !strings.HasPrefix(sqlLower, "explain") {
		response.Fail(c, response.CodeError, "只允许在查询窗口执行 SELECT 或 EXPLAIN 语句, 请检查当前 SQL合法性")
		return
	}

	// 注入执行人信息到 SQL 注释，便于数据库侧日志和审计追踪。
	username := jwt.GetUsernameFromContext(c)
	comment := "/* Executor: " + username + " */"
	finalSQL := comment + " " + req.SQL

	// 获取数据源名称用于审计展示，查询失败时使用 Unknown 兜底。
	dsInfo, _ := h.service.Get(uint(id))
	dsName := "Unknown"
	if dsInfo != nil {
		dsName = dsInfo.Name
	}

	res, err := h.service.ExecuteSQL(uint(id), finalSQL, req.Database)

	// 无论 SQL 执行成功或失败，都记录查询审计日志。
	status := 1
	description := strings.TrimSpace(req.Description)
	details := description
	if details == "" {
		details = fmt.Sprintf("[%s] 执行SQL: %s", dsName, req.SQL)
	}
	if err != nil {
		status = 0
		details += fmt.Sprintf("\n执行失败: %v", err)
	} else if description == "" {
		details += "\n执行成功"
	}

	details += "\n\n(查询窗口不执行DDL/DML工单静态审核)"

	rowCount := int64(0)
	durationMS := int64(0)
	if res != nil {
		rowCount = int64(len(res.Rows))
		durationMS = res.ExecutionTime
	}

	if auditErr := h.auditLogSvc.LogQueryAudit(&model.AuditLog{
		UserID:         userID,
		Username:       username,
		IP:             c.ClientIP(),
		Details:        details,
		Status:         status,
		DataSourceID:   uint(id),
		DataSource:     dsName,
		Database:       req.Database,
		QuerySessionID: req.QuerySessionID,
		SQLContent:     req.SQL,
		DurationMS:     durationMS,
		RowCount:       rowCount,
		Exported:       false,
	}); auditErr != nil {
		logger.Log.Error("Failed to write query audit log", zap.Error(auditErr))
	}

	if err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, res)
}
