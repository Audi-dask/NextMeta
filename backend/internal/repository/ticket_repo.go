package repository

import (
	"nextmeta-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

/*
TicketRepository 定义 SQL 工单和审批记录的数据访问能力。
它支撑工单创建、审批、列表查询、详情查询、审计列表和看板统计。
*/
type TicketRepository interface {
	Create(ticket *model.SQLTicket) error
	FindByID(id uint) (*model.SQLTicket, error)
	FindByCreator(creatorID uint) ([]model.SQLTicket, error)
	FindPendingByApprover(approverID uint) ([]model.SQLTicket, error)
	Update(ticket *model.SQLTicket) error
	ClaimApproval(ticketID, approverID uint, action, comment, nextStatus, executeResult string) (bool, error)
	ClaimWithdrawal(ticketID, creatorID uint) (bool, error)
	FailExecutingTickets(reason string) (int64, error)

	CreateApproval(approval *model.TicketApproval) error
	FindApprovalsByTicket(ticketID uint) ([]model.TicketApproval, error)
	HasApproval(ticketID, approverID uint) (bool, error)

	CountAll() (int64, error)
	CountPending() (int64, error)
	FindRecent(limit int) ([]model.SQLTicket, error)
	FindHistoryByApprover(approverID uint) ([]model.SQLTicket, error)
	FindTicketsForApprover(approverID uint, status string, creatorName string) ([]model.SQLTicket, error)
	FindExecutedTickets(ticketType string) ([]model.SQLTicket, error)
	GetDailyCount(days int) ([]model.DailyTrend, error)
}

/*
ticketRepository 是 TicketRepository 的 GORM 实现。
工单查询通常会预加载创建人、审批人、审批记录、数据源和执行人信息。
*/
type ticketRepository struct {
	db *gorm.DB
}

/*
NewTicketRepository 创建工单仓储。
db 由 main.go 初始化并注入。
*/
func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

/*
Create 创建 SQL 工单记录。
审批记录和执行结果会在后续流程中单独写入。
*/
func (r *ticketRepository) Create(ticket *model.SQLTicket) error {
	return r.db.Create(ticket).Error
}

/*
FindByID 按主键查询工单详情。
查询会预加载工单展示需要的用户、审批、数据源和执行人关联。
*/
func (r *ticketRepository) FindByID(id uint) (*model.SQLTicket, error) {
	var ticket model.SQLTicket
	err := r.db.Preload("Creator").Preload("Approver").Preload("Approvals.Approver").Preload("DataSource").Preload("Executor").First(&ticket, id).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) listQuery() *gorm.DB {
	return r.db.Model(&model.SQLTicket{}).
		Select("sql_tickets.id", "sql_tickets.created_at", "sql_tickets.updated_at", "sql_tickets.creator_id", "sql_tickets.approver_id", "sql_tickets.data_source_id", "sql_tickets.database", "sql_tickets.title", "sql_tickets.ticket_type", "sql_tickets.is_force", "sql_tickets.status", "sql_tickets.executor_id", "sql_tickets.executor_name", "sql_tickets.executed_at").
		Preload("Creator", func(db *gorm.DB) *gorm.DB { return db.Select("id", "username", "real_name") }).
		Preload("Approver", func(db *gorm.DB) *gorm.DB { return db.Select("id", "username", "real_name") }).
		Preload("DataSource", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name", "environment") }).
		Preload("Executor", func(db *gorm.DB) *gorm.DB { return db.Select("id", "username", "real_name") })
}

/*
FindByCreator 返回指定创建人的工单列表。
结果按创建时间倒序排列，用于“我的工单”页面。
*/
func (r *ticketRepository) FindByCreator(creatorID uint) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket
	err := r.listQuery().
		Where("creator_id = ?", creatorID).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

/*
FindPendingByApprover 返回指定审批人的待审批工单。
只查询状态为 pending 且 approver_id 匹配的记录。
*/
func (r *ticketRepository) FindPendingByApprover(approverID uint) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket
	err := r.listQuery().
		Where("approver_id = ? AND status = ?", approverID, "pending").
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

/*
FindHistoryByApprover 返回指定审批人已经处理过的工单。
通过 ticket_approvals 关联表查询，并按审批记录创建时间倒序排列。
*/
func (r *ticketRepository) FindHistoryByApprover(approverID uint) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket
	err := r.listQuery().
		Joins("JOIN ticket_approvals ON ticket_approvals.ticket_id = sql_tickets.id").
		Where("ticket_approvals.approver_id = ?", approverID).
		Order("ticket_approvals.created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

/*
FindTicketsForApprover 返回审批人相关工单列表。
包含当前待审批工单，以及该审批人曾经审批过的工单，并支持状态和创建人名称筛选。
*/
func (r *ticketRepository) FindTicketsForApprover(approverID uint, status string, creatorName string) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket

	query := r.listQuery().
		Joins("LEFT JOIN users ON users.id = sql_tickets.creator_id")

	// 子查询用于找出该审批人曾经处理过的工单。
	approvedSubQuery := r.db.Model(&model.TicketApproval{}).Select("ticket_id").Where("approver_id = ?", approverID)

	query = query.Where(
		r.db.Where("sql_tickets.status = ? AND sql_tickets.approver_id = ?", "pending", approverID).
			Or("sql_tickets.id IN (?)", approvedSubQuery),
	)

	// 按前端传入的状态和创建人关键字追加过滤条件。
	if status != "" && status != "all" {
		query = query.Where("sql_tickets.status = ?", status)
	}
	if creatorName != "" {
		query = query.Where("users.username LIKE ? OR users.real_name LIKE ?", "%"+creatorName+"%", "%"+creatorName+"%")
	}

	err := query.Order("sql_tickets.created_at DESC").Find(&tickets).Error
	return tickets, err
}

/*
FindExecutedTickets 返回已执行完成的工单列表。
ticketType 不为空且不为 all 时，会按工单类型继续过滤。
*/
func (r *ticketRepository) FindExecutedTickets(ticketType string) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket
	query := r.listQuery().
		Where("status IN ?", []string{"executed", "partial_success"})
	if ticketType != "" && ticketType != "all" {
		query = query.Where("ticket_type = ?", ticketType)
	}
	err := query.Order("updated_at DESC").Find(&tickets).Error
	return tickets, err
}

/*
Update 保存工单模型变更。
审批流和异步执行会通过该方法更新状态、结果和执行信息。
*/
func (r *ticketRepository) Update(ticket *model.SQLTicket) error {
	return r.db.Save(ticket).Error
}

/*
ClaimApproval 原子领取待审批工单并写入审批记录。
只有状态仍为 pending 的请求能够完成状态切换，避免并发请求重复启动执行任务。
*/
func (r *ticketRepository) ClaimApproval(ticketID, approverID uint, action, comment, nextStatus, executeResult string) (bool, error) {
	claimed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.SQLTicket{}).
			Where("id = ? AND status = ?", ticketID, "pending").
			Updates(map[string]interface{}{
				"status":         nextStatus,
				"execute_result": executeResult,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}

		approval := &model.TicketApproval{
			TicketID:   ticketID,
			ApproverID: approverID,
			Action:     action,
			Comment:    comment,
		}
		if err := tx.Create(approval).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return claimed, err
}

/*
ClaimWithdrawal 原子撤回提交人仍处于待审批状态的工单。
条件更新确保撤回与审批只能有一个请求成功取得状态变更权。
*/
func (r *ticketRepository) ClaimWithdrawal(ticketID, creatorID uint) (bool, error) {
	result := r.db.Model(&model.SQLTicket{}).
		Where("id = ? AND creator_id = ? AND status = ?", ticketID, creatorID, "pending").
		Updates(map[string]interface{}{
			"status":         "withdrawn",
			"execute_result": "工单已由提交人撤回",
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

/*
FailExecutingTickets 将服务启动前遗留的执行中工单统一标记为失败。
这些工单的数据库执行结果无法可靠判断，因此只记录结果未知，不进行自动重试。
*/
func (r *ticketRepository) FailExecutingTickets(reason string) (int64, error) {
	result := r.db.Model(&model.SQLTicket{}).
		Where("status = ?", "executing").
		Updates(map[string]interface{}{
			"status":         "failed",
			"execute_result": reason,
		})
	return result.RowsAffected, result.Error
}

/*
CreateApproval 创建工单审批记录。
每次审批动作都会写入一条 TicketApproval。
*/
func (r *ticketRepository) CreateApproval(approval *model.TicketApproval) error {
	return r.db.Create(approval).Error
}

/*
FindApprovalsByTicket 返回指定工单的审批记录列表。
结果按审批记录创建时间倒序排列。
*/
func (r *ticketRepository) FindApprovalsByTicket(ticketID uint) ([]model.TicketApproval, error) {
	var approvals []model.TicketApproval
	err := r.db.Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&approvals).Error
	return approvals, err
}

/*
HasApproval 判断审批人是否已经处理过指定工单。
审批接口会使用该方法避免重复审批。
*/
func (r *ticketRepository) HasApproval(ticketID, approverID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.TicketApproval{}).
		Where("ticket_id = ? AND approver_id = ?", ticketID, approverID).
		Count(&count).Error
	return count > 0, err
}

/*
CountAll 统计工单总数。
首页看板会通过该方法展示工单概览数量。
*/
func (r *ticketRepository) CountAll() (int64, error) {
	var count int64
	err := r.db.Model(&model.SQLTicket{}).Count(&count).Error
	return count, err
}

/*
CountPending 统计待审批工单数量。
首页看板会通过该方法展示待处理规模。
*/
func (r *ticketRepository) CountPending() (int64, error) {
	var count int64
	err := r.db.Model(&model.SQLTicket{}).Where("status = ?", "pending").Count(&count).Error
	return count, err
}

/*
FindRecent 返回最近创建的工单列表。
limit 控制返回数量，首页看板当前使用该方法展示最近工单。
*/
func (r *ticketRepository) FindRecent(limit int) ([]model.SQLTicket, error) {
	var tickets []model.SQLTicket
	err := r.db.Preload("Creator").Preload("Approver").Preload("DataSource").Preload("Executor").
		Order("created_at DESC").
		Limit(limit).
		Find(&tickets).Error
	return tickets, err
}

/*
GetDailyCount 统计最近指定天数内每天创建的工单数量。
结果按日期升序返回，用于首页工单趋势图。
*/
func (r *ticketRepository) GetDailyCount(days int) ([]model.DailyTrend, error) {
	var results []model.DailyTrend
	startDate := time.Now().AddDate(0, 0, -days)

	err := r.db.Model(&model.SQLTicket{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') as date, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("date").
		Order("date").
		Scan(&results).Error

	return results, err
}
