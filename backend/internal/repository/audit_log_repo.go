package repository

import (
	"nextmeta-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

/*
AuditLogRepository 定义审计日志表的数据访问能力。
它支持日志写入、查询窗口审计列表、按动作统计和每日趋势统计。
*/
type AuditLogRepository interface {
	Create(log *model.AuditLog) error
	FindQueryLogs() ([]model.AuditLog, error)
	CountByAction(action string) (int64, error)
	GetDailyCount(days int) ([]model.DailyTrend, error)
}

/*
auditLogRepository 是 AuditLogRepository 的 GORM 实现。
所有审计日志读写都通过注入的 *gorm.DB 执行。
*/
type auditLogRepository struct {
	db *gorm.DB
}

/*
NewAuditLogRepository 创建审计日志仓储。
db 由 main.go 初始化并注入。
*/
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

/*
Create 写入审计日志记录。
调用方负责组装用户、动作、IP、详情和状态等字段。
*/
func (r *auditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

/*
FindQueryLogs 返回带查询会话 ID 的查询窗口审计日志。
结果按创建时间和 ID 升序，便于上层稳定聚合并保持 SQL 执行顺序。
*/
func (r *auditLogRepository) FindQueryLogs() ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.db.Preload("User").
		Where("action = ? AND query_session_id <> ''", "Query").
		Order("created_at ASC").
		Order("id ASC").
		Find(&logs).Error
	return logs, err
}

/*
CountByAction 按动作类型统计审计日志数量。
action 为空时统计全部审计日志，首页看板当前传入 Query 统计查询次数。
*/
func (r *auditLogRepository) CountByAction(action string) (int64, error) {
	var count int64
	db := r.db.Model(&model.AuditLog{})
	if action != "" {
		db = db.Where("action = ?", action)
	}
	err := db.Count(&count).Error
	return count, err
}

/*
GetDailyCount 统计最近指定天数内每天产生的审计日志数量。
结果按日期升序返回，用于首页 SQL 查询趋势图。
*/
func (r *auditLogRepository) GetDailyCount(days int) ([]model.DailyTrend, error) {
	var results []model.DailyTrend
	startDate := time.Now().AddDate(0, 0, -days)

	err := r.db.Model(&model.AuditLog{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') as date, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("date").
		Order("date").
		Scan(&results).Error

	return results, err
}
