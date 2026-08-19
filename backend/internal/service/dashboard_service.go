package service

import (
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
)

/*
DashboardStats 是首页看板统计响应结构。
包含工单、查询、数据源、用户总量，以及最近趋势和资产类型分布。
*/
type DashboardStats struct {
	TotalTickets          int64              `json:"total_tickets"`
	PendingTickets        int64              `json:"pending_tickets"`
	TotalQueries          int64              `json:"total_queries"`
	TotalDataSources      int64              `json:"total_datasources"`
	TotalUsers            int64              `json:"total_users"`
	SQLQueryTrend         []model.DailyTrend `json:"sql_query_trend"`
	TicketTrend           []model.DailyTrend `json:"ticket_trend"`
	AssetTypeDistribution []model.DailyTrend `json:"asset_type_distribution"`
}

/*
DashboardService 定义首页看板业务能力。
当前首页只暴露概览统计数据，不返回最近工单明细，避免冗余字段和敏感信息泄露到前端。
*/
type DashboardService interface {
	GetStats() (*DashboardStats, error)
}

/*
dashboardService 是 DashboardService 的默认实现。
看板统计会复用工单、数据源、用户和审计日志 repository。
*/
type dashboardService struct {
	ticketRepo   repository.TicketRepository
	dsRepo       repository.DataSourceRepository
	userRepo     repository.UserRepository
	auditLogRepo repository.AuditLogRepository
}

/*
NewDashboardService 创建首页看板业务服务。
各 repository 由 main.go 注入，用于聚合不同模块的统计数据。
*/
func NewDashboardService(
	ticketRepo repository.TicketRepository,
	dsRepo repository.DataSourceRepository,
	userRepo repository.UserRepository,
	auditLogRepo repository.AuditLogRepository,
) DashboardService {
	return &dashboardService{
		ticketRepo:   ticketRepo,
		dsRepo:       dsRepo,
		userRepo:     userRepo,
		auditLogRepo: auditLogRepo,
	}
}

/*
GetStats 聚合首页看板统计数据。
核心计数失败会直接返回错误，趋势和分布查询失败时降级为空列表，避免影响看板主体展示。
*/
func (s *dashboardService) GetStats() (*DashboardStats, error) {
	totalTickets, err := s.ticketRepo.CountAll()
	if err != nil {
		return nil, err
	}

	pendingTickets, err := s.ticketRepo.CountPending()
	if err != nil {
		return nil, err
	}

	totalQueries, err := s.auditLogRepo.CountByAction("Query")
	if err != nil {
		return nil, err
	}

	totalDataSources, err := s.dsRepo.CountAll()
	if err != nil {
		return nil, err
	}

	totalUsers, err := s.userRepo.CountAll()
	if err != nil {
		return nil, err
	}

	// 趋势图默认读取最近 7 天数据，失败时用空列表兜底。
	sqlTrend, err := s.auditLogRepo.GetDailyCount(7)
	if err != nil {
		sqlTrend = []model.DailyTrend{}
	}

	ticketTrend, err := s.ticketRepo.GetDailyCount(7)
	if err != nil {
		ticketTrend = []model.DailyTrend{}
	}

	// 资产类型分布失败时也降级为空列表。
	assetDist, err := s.dsRepo.CountByType()
	if err != nil {
		assetDist = []model.DailyTrend{}
	}

	return &DashboardStats{
		TotalTickets:          totalTickets,
		PendingTickets:        pendingTickets,
		TotalQueries:          totalQueries,
		TotalDataSources:      totalDataSources,
		TotalUsers:            totalUsers,
		SQLQueryTrend:         sqlTrend,
		TicketTrend:           ticketTrend,
		AssetTypeDistribution: assetDist,
	}, nil
}
