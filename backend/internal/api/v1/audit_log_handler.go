package v1

import (
	"fmt"
	"sort"
	"strings"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

/*
AuditLogHandler 承接审计日志相关 HTTP 请求。
当前主要提供查询窗口 SQL 审计日志列表接口。
*/
type AuditLogHandler struct {
	auditService service.AuditLogService
}

/*
NewAuditLogHandler 创建审计日志接口 Handler。
auditService 由 main.go 注入，负责审计日志查询和写入业务。
*/
func NewAuditLogHandler(auditService service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{auditService: auditService}
}

/*
GetQueryAuditLogs 返回 SQL 查询窗口产生的审计日志列表。
同一用户在同一查询说明会话内执行的 SQL 会聚合为一个查询单。
*/
func (h *AuditLogHandler) GetQueryAuditLogs(c *gin.Context) {
	logs, err := h.auditService.QueryLogs()
	if err != nil {
		response.Fail(c, response.CodeError, "获取查询审计日志失败")
		return
	}

	response.Success(c, aggregateQueryAuditLogs(logs))
}

type queryAuditGroupKey struct {
	userID         uint
	querySessionID string
}

type queryAuditGroup struct {
	first    model.AuditLog
	records  []model.AuditLog
	exported bool
}

/*
aggregateQueryAuditLogs 以用户 ID 和查询会话 ID 作为严格复合键聚合查询日志。
组内按执行时间升序，查询单按首次提交时间倒序展示。
*/
func aggregateQueryAuditLogs(logs []model.AuditLog) []dto.QueryAuditLogResponse {
	sortedLogs := append([]model.AuditLog(nil), logs...)
	sort.SliceStable(sortedLogs, func(i, j int) bool {
		if sortedLogs[i].CreatedAt.Equal(sortedLogs[j].CreatedAt) {
			return sortedLogs[i].ID < sortedLogs[j].ID
		}
		return sortedLogs[i].CreatedAt.Before(sortedLogs[j].CreatedAt)
	})

	groupsByKey := make(map[queryAuditGroupKey]*queryAuditGroup)
	groups := make([]*queryAuditGroup, 0)
	for _, log := range sortedLogs {
		if strings.TrimSpace(log.QuerySessionID) == "" {
			continue
		}
		key := queryAuditGroupKey{userID: log.UserID, querySessionID: log.QuerySessionID}
		group, exists := groupsByKey[key]
		if !exists {
			group = &queryAuditGroup{first: log}
			groupsByKey[key] = group
			groups = append(groups, group)
		}
		group.records = append(group.records, log)
		group.exported = group.exported || log.Exported
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].first.CreatedAt.Equal(groups[j].first.CreatedAt) {
			return groups[i].first.ID > groups[j].first.ID
		}
		return groups[i].first.CreatedAt.After(groups[j].first.CreatedAt)
	})

	items := make([]dto.QueryAuditLogResponse, 0, len(groups))
	for _, group := range groups {
		items = append(items, mapQueryAuditGroupResponse(group))
	}
	return items
}

func mapQueryAuditGroupResponse(group *queryAuditGroup) dto.QueryAuditLogResponse {
	first := group.first
	description := truncateQueryAuditDescription(parseQueryAuditDescription(first.Details))
	records := make([]dto.QueryAuditRecordResponse, 0, len(group.records))
	for _, log := range group.records {
		records = append(records, mapQueryAuditRecordResponse(log))
	}

	return dto.QueryAuditLogResponse{
		ID:          first.ID,
		TicketNo:    fmt.Sprintf("QRY-%06d", first.ID),
		Querier:     first.Username,
		QuerierName: getAuditLogUserRealName(first),
		SubmittedAt: formatQueryAuditTime(first),
		Description: description,
		Exported:    group.exported,
		Records:     records,
	}
}

func mapQueryAuditRecordResponse(log model.AuditLog) dto.QueryAuditRecordResponse {
	status := "failed"
	if log.Status == 1 {
		status = "success"
	}

	database := log.Database
	if strings.TrimSpace(database) == "" {
		database = "-"
	}

	return dto.QueryAuditRecordResponse{
		ID:          log.ID,
		DataSource:  log.DataSource,
		Database:    database,
		SQL:         log.SQLContent,
		SubmittedAt: formatQueryAuditTime(log),
		Duration:    log.DurationMS,
		Status:      status,
		Rows:        log.RowCount,
	}
}

func formatQueryAuditTime(log model.AuditLog) string {
	return log.CreatedAt.Format("2006-01-02 15:04:05")
}

/*
getAuditLogUserRealName 返回审计日志展示用的用户名称。
优先使用关联用户的真实姓名，缺失时回退到日志里的 username。
*/
func getAuditLogUserRealName(log model.AuditLog) string {
	if strings.TrimSpace(log.User.RealName) != "" {
		return log.User.RealName
	}
	return log.Username
}

/*
parseQueryAuditDescription 从 Details 中提取查询说明，并移除执行结果附加信息。
*/
func parseQueryAuditDescription(details string) string {
	value := strings.TrimSpace(details)
	if index := strings.Index(value, "\n\n(查询窗口不执行DDL/DML工单静态审核)"); index >= 0 {
		value = value[:index]
	}
	if index := strings.Index(value, "\n执行失败:"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

/*
truncateQueryAuditDescription 截断审计列表中的描述文本。
*/
func truncateQueryAuditDescription(value string) string {
	const maxLength = 96
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= maxLength {
		return value
	}
	return string([]rune(value)[:maxLength]) + "..."
}
