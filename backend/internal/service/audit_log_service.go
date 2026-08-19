package service

import (
	"errors"
	"strings"

	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
)

/*
AuditLogService 定义审计日志业务能力。
它同时支持通用操作日志写入、查询窗口审计日志写入和查询审计列表读取。
*/
type AuditLogService interface {
	Log(userID uint, username, action, ip, details string, success bool) error
	LogQueryAudit(entry *model.AuditLog) error
	QueryLogs() ([]model.AuditLog, error)
}

/*
auditLogService 是 AuditLogService 的默认实现。
实际持久化和查询由 AuditLogRepository 负责。
*/
type auditLogService struct {
	repo repository.AuditLogRepository
}

/*
NewAuditLogService 创建审计日志服务。
repo 由 main.go 注入，用于写入和查询 audit_logs 表。
*/
func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{repo: repo}
}

/*
Log 写入一条通用审计日志。
success 会转换为数据库中的状态值，成功为 1，失败为 0。
*/
func (s *auditLogService) Log(userID uint, username, action, ip, details string, success bool) error {
	status := 0
	if success {
		status = 1
	}
	log := &model.AuditLog{
		UserID:   userID,
		Username: username,
		Action:   action,
		IP:       ip,
		Details:  details,
		Status:   status,
	}
	return s.repo.Create(log)
}

/*
LogQueryAudit 写入查询窗口审计日志。
该方法固定 Action 为 Query，并把非 0 状态归一为成功状态 1。
*/
func (s *auditLogService) LogQueryAudit(entry *model.AuditLog) error {
	if strings.TrimSpace(entry.QuerySessionID) == "" {
		return errors.New("query session ID is required")
	}
	entry.Action = "Query"
	if entry.Status != 0 {
		entry.Status = 1
	}
	return s.repo.Create(entry)
}

/*
QueryLogs 返回查询窗口审计日志列表。
列表排序、关联预加载和过滤逻辑由 repository 层处理。
*/
func (s *auditLogService) QueryLogs() ([]model.AuditLog, error) {
	return s.repo.FindQueryLogs()
}
