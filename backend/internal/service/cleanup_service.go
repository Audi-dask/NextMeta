package service

import (
	"time"

	"nextmeta-backend/internal/repository"
)

/*
cleanupBatchSize 是单批物理删除的行数上限。
分批删除并逐批提交，避免一次性删除海量历史数据造成长事务和锁表。
*/
const cleanupBatchSize = 1000

/*
CleanupResult 汇总一次历史清理删除的各表行数，返回给前端展示。
*/
type CleanupResult struct {
	TicketApprovals int64 `json:"ticket_approvals"`
	SQLTickets      int64 `json:"sql_tickets"`
	AuditLogs       int64 `json:"audit_logs"`
}

/*
CleanupService 定义历史数据清理能力。
按截止时间物理删除工单审批、工单和查询审计三类历史数据。
*/
type CleanupService interface {
	PurgeBefore(cutoff time.Time) (*CleanupResult, error)
}

/*
cleanupService 是 CleanupService 的默认实现。
通过 CleanupRepository 分批物理删除，删除顺序先子表后父表，避免触发外键约束。
*/
type cleanupService struct {
	repo repository.CleanupRepository
}

/*
NewCleanupService 创建清理服务。
repo 由 main.go 注入，负责具体表的分批 DELETE。
*/
func NewCleanupService(repo repository.CleanupRepository) CleanupService {
	return &cleanupService{repo: repo}
}

/*
PurgeBefore 删除 created_at 早于 cutoff 的历史数据。
删除顺序为 ticket_approvals（子表）→ sql_tickets（父表）→ audit_logs（独立表）。
任一步失败立即返回已删除的累计行数和错误，不继续后续表，避免破坏数据一致性预期。
*/
func (s *cleanupService) PurgeBefore(cutoff time.Time) (*CleanupResult, error) {
	result := &CleanupResult{}

	approvals, err := s.repo.PurgeBefore("ticket_approvals", cutoff, cleanupBatchSize)
	if err != nil {
		return result, err
	}
	result.TicketApprovals = approvals

	tickets, err := s.repo.PurgeBefore("sql_tickets", cutoff, cleanupBatchSize)
	if err != nil {
		return result, err
	}
	result.SQLTickets = tickets

	logs, err := s.repo.PurgeBefore("audit_logs", cutoff, cleanupBatchSize)
	if err != nil {
		return result, err
	}
	result.AuditLogs = logs

	return result, nil
}
