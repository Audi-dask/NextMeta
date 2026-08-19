package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/logger"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"vitess.io/vitess/go/vt/sqlparser"
)

/*
TicketApproverResponse 是工单提交时返回给前端的可选审批人结构。
它只暴露用户 ID、账号和真实姓名，用于审批人下拉选择。
*/
type TicketApproverResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
}

/*
targetGroupApprover 表示某个可用用户组和审批人的组合。
同一个审批人可能来自多个用户组，后续会按审批人 ID 去重展示。
*/
type targetGroupApprover struct {
	groupID  uint
	approver model.User
}

/*
ticketExecutionResult 保存工单执行后的摘要信息。
异步执行完成后会用这些字段更新工单状态、影响行数和执行耗时。
*/
type ticketExecutionResult struct {
	Message             string
	Status              string
	AffectedRows        int64
	ExecutionDurationMS int64
	StatementResults    []dto.StatementExecutionResult
}

/*
TicketService 定义 SQL 工单核心业务能力。
它负责工单创建、审批流、异步执行、导出、工单列表、详情和审计记录查询。
*/
type TicketService interface {
	CreateTicket(creatorID uint, title, sqlContent, ticketType string, datasourceID uint, database string, approverID uint, isForce bool) (*model.SQLTicket, error)
	ApproveTicket(ticketID, approverID uint, action, comment, clientIP string) error
	WithdrawTicket(ticketID, creatorID uint) error
	GetMyTickets(userID uint) ([]model.SQLTicket, error)
	GetPendingTickets(approverID uint) ([]model.SQLTicket, error)
	GetTicketDetail(ticketID, currentUserID uint, currentUserRole string) (*model.SQLTicket, error)
	ExportTicketResult(userID, ticketID uint) (string, error)
	CheckSyntax(datasourceID uint, database, sqlContent, ticketType string) (*dto.ExplainResult, error)
	GetApprovalHistory(approverID uint) ([]model.SQLTicket, error)
	GetTicketsForApprover(approverID uint, status string, creatorName string) ([]model.SQLTicket, error)
	GetTicketAuditLogs(ticketType string) ([]model.SQLTicket, error)
	GetDatasourceApprovers(userID, datasourceID uint) ([]TicketApproverResponse, error)
}

/*
ticketService 是 TicketService 的默认实现。
它组合工单、用户、权限、数据源、审核、审计日志、系统设置和通知服务完成工单生命周期处理。
*/
type ticketService struct {
	ticketRepo      repository.TicketRepository
	userRepo        repository.UserRepository
	permRepo        repository.PermissionRepository
	permSvc         PermissionService
	dsService       DataSourceService
	auditLogSvc     AuditLogService
	settingsRepo    repository.SystemSettingRepository
	auditService    AuditService
	notificationSvc NotificationService
}

/*
NewTicketService 创建工单业务服务。
依赖由 main.go 注入，便于工单流程在 service 层统一编排。
*/
func NewTicketService(
	repo repository.TicketRepository,
	userRepo repository.UserRepository,
	permRepo repository.PermissionRepository,
	permService PermissionService,
	dsService DataSourceService,
	auditLogSvc AuditLogService,
	settingsRepo repository.SystemSettingRepository,
	auditService AuditService,
	notificationSvc NotificationService,
) TicketService {
	return &ticketService{
		ticketRepo:      repo,
		userRepo:        userRepo,
		permRepo:        permRepo,
		permSvc:         permService,
		dsService:       dsService,
		auditLogSvc:     auditLogSvc,
		settingsRepo:    settingsRepo,
		auditService:    auditService,
		notificationSvc: notificationSvc,
	}
}

/*
resolveTargetGroupApprovers 查询用户在指定数据源下可用的审批人。
逻辑会先找用户所属组，再筛选已授权目标数据源的组，最后收集这些组的审批人。
*/
func (s *ticketService) resolveTargetGroupApprovers(userID, datasourceID uint) ([]targetGroupApprover, error) {
	groups, err := s.permRepo.GetUserGroups(userID)
	if err != nil {
		return nil, err
	}

	matched := make([]targetGroupApprover, 0)
	for _, group := range groups {
		datasources, err := s.permRepo.GetGroupDataSources(group.ID)
		if err != nil {
			continue
		}

		matchedDataSource := false
		for _, ds := range datasources {
			if ds.ID == datasourceID {
				matchedDataSource = true
				break
			}
		}
		if !matchedDataSource {
			continue
		}

		approvers, err := s.permRepo.GetGroupApprovers(group.ID)
		if err != nil {
			continue
		}
		for _, approver := range approvers {
			matched = append(matched, targetGroupApprover{
				groupID:  group.ID,
				approver: approver,
			})
		}
	}

	if len(matched) == 0 {
		return nil, errors.New("您没有该数据源的访问权限或当前组未配置审批人")
	}

	return matched, nil
}

/*
resolveTargetGroupID 根据提交人、数据源和所选审批人确定目标用户组。
只有所选审批人属于当前数据源可用审批范围时，才允许创建工单。
*/
func (s *ticketService) resolveTargetGroupID(userID, datasourceID, approverID uint) (uint, error) {
	matched, err := s.resolveTargetGroupApprovers(userID, datasourceID)
	if err != nil {
		return 0, err
	}

	for _, item := range matched {
		if item.approver.ID == approverID {
			return item.groupID, nil
		}
	}

	return 0, errors.New("所选审核人不属于当前数据源可用审核范围")
}

/*
GetDatasourceApprovers 返回用户提交指定数据源工单时可选的审批人列表。
多个用户组包含同一审批人时，会按审批人 ID 去重。
*/
func (s *ticketService) GetDatasourceApprovers(userID, datasourceID uint) ([]TicketApproverResponse, error) {
	matched, err := s.resolveTargetGroupApprovers(userID, datasourceID)
	if err != nil {
		return nil, err
	}

	responses := make([]TicketApproverResponse, 0, len(matched))
	seen := make(map[uint]bool, len(matched))
	for _, item := range matched {
		approver := item.approver
		if seen[approver.ID] {
			continue
		}
		seen[approver.ID] = true
		responses = append(responses, TicketApproverResponse{
			ID:       approver.ID,
			Username: approver.Username,
			RealName: approver.RealName,
		})
	}

	return responses, nil
}

/*
CreateTicket 创建待审批 SQL 工单。
创建前会校验导出工单必须为 SELECT，并根据提交人与数据源解析目标审批组。
*/
func (s *ticketService) CreateTicket(creatorID uint, title, sqlContent, ticketType string, datasourceID uint, database string, approverID uint, isForce bool) (*model.SQLTicket, error) {
	// 导出工单只能保存 SELECT，避免通过导出流程执行变更类 SQL。
	if ticketType == "export" {
		parser := sqlparser.NewTestParser()
		stmt, err := parser.Parse(sqlContent)
		if err != nil {
			return nil, fmt.Errorf("SQL parse failed: %v", err)
		}
		if _, ok := stmt.(*sqlparser.Select); !ok {
			return nil, errors.New("导出工单仅支持 SELECT 语句")
		}
	}

	// 只读库禁止参与 DDL/DML 变更工单，只读访问仅用于查询窗口。
	upperTicketType := strings.ToUpper(strings.TrimSpace(ticketType))
	if upperTicketType == "DDL" || upperTicketType == "DML" {
		ds, err := s.dsService.Get(datasourceID)
		if err != nil {
			return nil, err
		}
		if normalizeDataSourceAccessMode(ds.AccessMode) == "read_only" {
			return nil, errors.New("只读库不允许提交 DDL/DML 工单")
		}
	}

	// 获取用户提交该数据源工单时对应的目标组。
	targetGroupID, err := s.resolveTargetGroupID(creatorID, datasourceID, approverID)
	if err != nil {
		return nil, err
	}

	// 所有工单类型都需要审批，目标组必须至少配置一个审批人。
	hasApprovers, err := s.permSvc.HasGroupApprovers(targetGroupID)
	if err != nil {
		return nil, err
	}
	if !hasApprovers {
		return nil, errors.New("当前组未配置审批人,无法提交工单,请联系管理员")
	}

	// 新工单统一进入 pending 状态，等待指定审批人处理。
	ticket := &model.SQLTicket{
		CreatorID:    creatorID,
		ApproverID:   approverID,
		GroupID:      targetGroupID,
		DataSourceID: datasourceID,
		Database:     database,
		Title:        title,
		SQLContent:   sqlContent,
		TicketType:   ticketType,
		IsForce:      isForce,
		Status:       "pending",
	}

	err = s.ticketRepo.Create(ticket)
	if err != nil {
		return nil, err
	}

	s.notifyTicketCreatedByID(ticket.ID)

	return ticket, nil
}

/*
ApproveTicket 处理工单审批动作。
通过审批后会把工单置为 executing 并启动后台执行；拒绝时记录审计日志并通知提交人。
*/
func (s *ticketService) ApproveTicket(ticketID, approverID uint, action, comment, clientIP string) error {
	// 获取工单并确认当前状态仍为待审批。
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return err
	}

	if ticket.Status != "pending" {
		return errors.New("工单状态不是待审批")
	}

	// 只有工单指定审批人且仍属于目标组审批人时，才允许处理该工单。
	if ticket.ApproverID != approverID {
		return errors.New("您不是该工单指定审核人")
	}

	isApprover, err := s.permSvc.IsGroupApprover(approverID, ticket.GroupID)
	if err != nil {
		return err
	}
	if !isApprover {
		return errors.New("您不是该组的审批人")
	}

	// 当前产品允许审批自己的工单，保留入口注释便于后续恢复限制。
	// if ticket.CreatorID == approverID {
	// 	return errors.New("不能审批自己的工单")
	// }

	// 获取审批人名称用于审计日志和通知内容，优先使用真实姓名。
	approverName := "Approver"
	if user, err := s.userRepo.FindByID(approverID); err == nil && user != nil {
		approverName = ticketUserDisplayName(user, approverID)
	}

	nextStatus := "rejected"
	executeResult := ""
	if action == "approve" {
		nextStatus = "executing"
		executeResult = "正在后台执行中..."
	}
	claimed, err := s.ticketRepo.ClaimApproval(ticketID, approverID, action, comment, nextStatus, executeResult)
	if err != nil {
		return err
	}
	if !claimed {
		return errors.New("工单状态不是待审批")
	}

	if action == "approve" {
		ticket.Status = "executing"
		ticket.ExecuteResult = executeResult

		var logAction string
		var logMsg string

		if ticket.TicketType == "export" {
			logAction = "SubmitAsyncExport"
			logMsg = fmt.Sprintf("导出工单已提交后台执行: %s", ticket.Title)
		} else {
			logAction = "SubmitAsyncExecution"
			dsInfo, _ := s.dsService.Get(ticket.DataSourceID)
			dsName := "Unknown"
			if dsInfo != nil {
				dsName = dsInfo.Name
			}
			logMsg = fmt.Sprintf("[%s] 工单已提交后台执行: %s", dsName, ticket.Title)
		}

		// 记录工单已提交后台执行的审计日志。
		_ = s.auditLogSvc.Log(
			approverID,
			approverName,
			logAction,
			clientIP,
			logMsg,
			true,
		)

		// 后台执行使用独立 goroutine，不依赖 HTTP 请求上下文生命周期。
		go func(ticketID uint, approverName string, approverID uint, isExport bool) {

			if isExport {
				// 导出工单走流式查询并写入本地 CSV 文件。
				s.executeExportAsync(ticketID, approverName, clientIP, approverID)
				return
			}

			// 后台执行必须兜底 recover，避免单个工单异常影响进程。
			defer func() {
				if r := recover(); r != nil {
					logger.Log.Error("Recovered panic while executing ticket", zap.Uint("ticket_id", ticketID), zap.Any("panic", r))
					if err := s.updateTicketStatusAsync(ticketID, "failed", fmt.Sprintf("System Panic: %v", r)); err == nil {
						s.notifyTicketResultByID(ticketID, approverName, fmt.Sprintf("System Panic: %v", r))
					}
				}
			}()

			// 后台任务重新读取工单，避免使用审批时的旧内存对象。
			t, err := s.ticketRepo.FindByID(ticketID)
			if err != nil {
				logger.Log.Error("Failed to load ticket for async execution",
					zap.Uint("ticket_id", ticketID),
					zap.Error(err),
				)
				return
			}

			execRes, execErr := s.executeSQL(t, approverName)

			// 根据执行结果更新最终状态、执行摘要、影响行数和耗时。
			var updateErr error
			if execRes != nil {
				status := execRes.Status
				if status == "" {
					if execErr != nil {
						status = "failed"
					} else {
						status = "executed"
					}
				}
				message := execRes.Message
				if message == "" && execErr != nil {
					message = fmt.Sprintf("执行失败: %v", execErr)
				}
				updateErr = s.updateTicketExecutionAsync(ticketID, status, message, approverID, approverName, execRes.AffectedRows, execRes.ExecutionDurationMS, execRes.StatementResults)
			} else {
				updateErr = s.updateTicketExecutionAsync(ticketID, "failed", fmt.Sprintf("执行失败: %v", execErr), approverID, approverName, 0, 0, nil)
			}

			if updateErr != nil {
				return
			}

			finalStatus := "failed"
			if execRes != nil {
				finalStatus = execRes.Status
				if finalStatus == "" {
					if execErr != nil {
						finalStatus = "failed"
					} else {
						finalStatus = "executed"
					}
				}
			}

			// 只有最终状态可靠落库后，才发送执行完成或失败通知。
			s.notifyTicketResultByID(ticketID, approverName, ticketExecutionRemark(finalStatus))

		}(ticket.ID, approverName, approverID, ticket.TicketType == "export")

		// 审批接口立即返回成功，真实执行结果后续通过工单状态查看。
		return nil
	} else if action == "reject" {
		ticket.Status = "rejected"
		if err := s.ticketRepo.Update(ticket); err != nil {
			return err
		}

		// 拒绝工单时记录数据源名称、工单标题和拒绝理由。
		dsInfo, _ := s.dsService.Get(ticket.DataSourceID)
		dsName := "Unknown"
		if dsInfo != nil {
			dsName = dsInfo.Name
		}
		_ = s.auditLogSvc.Log(approverID, approverName, "RejectTicket", clientIP, fmt.Sprintf("[%s] 拒绝工单: %s, 理由: %s", dsName, ticket.Title, comment), true)

		s.notifyTicketResultByID(ticketID, approverName, comment)
		return nil
	}

	return nil
}

const ticketStatusUpdateMaxAttempts = 3

func ticketUserDisplayName(user *model.User, fallbackID uint) string {
	if user == nil {
		return fmt.Sprintf("%d", fallbackID)
	}
	if strings.TrimSpace(user.RealName) != "" {
		return user.RealName
	}
	if strings.TrimSpace(user.Username) != "" {
		return user.Username
	}
	return fmt.Sprintf("%d", fallbackID)
}

func ticketExecutionRemark(status string) string {
	switch status {
	case "executed":
		return "工单已执行成功"
	case "partial_success":
		return "工单部分执行成功"
	case "failed":
		return "工单执行失败"
	default:
		return "工单执行完成"
	}
}

func (s *ticketService) notifyTicketCreatedByID(ticketID uint) {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		logger.Log.Error("Failed to load ticket for created notification", zap.Uint("ticket_id", ticketID), zap.Error(err))
		return
	}
	s.notificationSvc.NotifyTicketCreated(ticket)
}

func (s *ticketService) notifyTicketResultByID(ticketID uint, operator string, remark string) {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		logger.Log.Error("Failed to load ticket for result notification", zap.Uint("ticket_id", ticketID), zap.Error(err))
		return
	}
	s.notificationSvc.NotifyResult(ticket, operator, remark)
}

func retryTicketUpdate(update func() error) (int, error) {
	var lastErr error
	for attempt := 1; attempt <= ticketStatusUpdateMaxAttempts; attempt++ {
		if err := update(); err == nil {
			return attempt, nil
		} else {
			lastErr = err
		}
	}
	return ticketStatusUpdateMaxAttempts, lastErr
}

/*
updateTicketStatusAsync 是后台任务更新工单状态的简化入口。
它不记录执行人和统计信息，只更新状态和结果文本。
*/
func (s *ticketService) updateTicketStatusAsync(ticketID uint, status string, result string) error {
	return s.updateTicketExecutionAsync(ticketID, status, result, 0, "", 0, 0, nil)
}

/*
updateTicketExecutionAsync 更新异步执行后的工单状态和执行信息。
状态保存失败时有限重试；最终失败会记录高优先级日志并返回错误，由调用方阻止发送成功通知。
*/
func (s *ticketService) updateTicketExecutionAsync(ticketID uint, status string, result string, executorID uint, executorName string, affectedRows int64, durationMS int64, statementResults []dto.StatementExecutionResult) error {
	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		logger.Log.Error("Failed to load ticket for execution status update", zap.Uint("ticket_id", ticketID), zap.Error(err))
		return err
	}
	executedAt := time.Now()
	t.Status = status
	t.ExecuteResult = result
	if len(statementResults) > 0 {
		encodedResults, err := json.Marshal(statementResults)
		if err != nil {
			logger.Log.Error("Failed to encode ticket statement results", zap.Uint("ticket_id", ticketID), zap.Error(err))
			return err
		}
		t.StatementResults = string(encodedResults)
	}
	if executorID > 0 {
		t.ExecutorID = executorID
	}
	if executorName != "" {
		t.ExecutorName = executorName
	}
	t.ExecutedAt = &executedAt
	t.AffectedRows = affectedRows
	t.ExecutionDurationMS = durationMS

	attempts, err := retryTicketUpdate(func() error {
		return s.ticketRepo.Update(t)
	})
	if err != nil {
		logger.Log.Error("Failed to persist final ticket execution status",
			zap.Uint("ticket_id", ticketID),
			zap.String("target_status", status),
			zap.Int("attempts", attempts),
			zap.Error(err),
		)
		return err
	}
	if attempts > 1 {
		logger.Log.Warn("Final ticket execution status persisted after retry",
			zap.Uint("ticket_id", ticketID),
			zap.String("target_status", status),
			zap.Int("attempts", attempts),
		)
	}
	return nil
}

/*
WithdrawTicket 允许提交人在工单仍待审批时主动撤回。
只有创建人本人且状态为 pending 的工单可以撤回，避免已审批或已执行工单被回滚状态。
*/
func (s *ticketService) WithdrawTicket(ticketID, creatorID uint) error {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return err
	}
	if ticket.CreatorID != creatorID {
		return errors.New("只能撤回自己提交的工单")
	}
	if ticket.Status != "pending" {
		return errors.New("只有审核中的工单可以撤回")
	}

	withdrawn, err := s.ticketRepo.ClaimWithdrawal(ticketID, creatorID)
	if err != nil {
		return err
	}
	if !withdrawn {
		return errors.New("只有审核中的工单可以撤回")
	}
	return nil
}

/*
GetMyTickets 返回用户自己提交的工单。
该方法用于“我的工单”列表。
*/
func (s *ticketService) GetMyTickets(userID uint) ([]model.SQLTicket, error) {
	return s.ticketRepo.FindByCreator(userID)
}

/*
GetPendingTickets 返回指定审批人的待审批工单。
该方法用于审批工作台的待办列表。
*/
func (s *ticketService) GetPendingTickets(approverID uint) ([]model.SQLTicket, error) {
	return s.ticketRepo.FindPendingByApprover(approverID)
}

/*
GetApprovalHistory 返回指定审批人的审批历史。
repository 层负责筛选已处理过的工单记录。
*/
func (s *ticketService) GetApprovalHistory(approverID uint) ([]model.SQLTicket, error) {
	return s.ticketRepo.FindHistoryByApprover(approverID)
}

/*
GetTicketsForApprover 返回审批人相关工单列表。
status 和 creatorName 用于前端筛选条件。
*/
func (s *ticketService) GetTicketsForApprover(approverID uint, status string, creatorName string) ([]model.SQLTicket, error) {
	return s.ticketRepo.FindTicketsForApprover(approverID, status, creatorName)
}

/*
GetTicketAuditLogs 返回已执行工单列表。
ticketType 为空或 all 时由 repository 层决定是否不过滤类型。
*/
func (s *ticketService) GetTicketAuditLogs(ticketType string) ([]model.SQLTicket, error) {
	return s.ticketRepo.FindExecutedTickets(ticketType)
}

/*
GetTicketDetail 返回当前用户可访问的工单详情。
管理员可查看全部工单；普通用户只能查看自己创建、当前待审批或曾经审批过的工单。
*/
func (s *ticketService) GetTicketDetail(ticketID, currentUserID uint, currentUserRole string) (*model.SQLTicket, error) {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, errors.New("工单不存在")
	}

	if model.IsAdminRole(currentUserRole) || ticket.CreatorID == currentUserID || ticket.ApproverID == currentUserID {
		return ticket, nil
	}

	hasApproved, err := s.ticketRepo.HasApproval(ticketID, currentUserID)
	if err != nil {
		return nil, err
	}
	if hasApproved {
		return ticket, nil
	}

	return nil, errors.New("无权查看该工单")
}

/*
ExportTicketResult 返回导出工单生成的本地文件路径。
调用者必须是工单创建人，或当前仍拥有该数据源访问权限。
*/
func (s *ticketService) ExportTicketResult(userID, ticketID uint) (string, error) {
	ticket, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		return "", err
	}
	if ticket == nil {
		return "", errors.New("工单不存在")
	}

	if ticket.TicketType != "export" {
		return "", errors.New("只有导出类型工单支持下载结果")
	}

	// 创建人可直接下载；非创建人必须仍然拥有该数据源访问权限。
	if ticket.CreatorID != userID {
		canAccess, err := s.permSvc.CanAccessDataSource(userID, ticket.DataSourceID)
		if err != nil {
			return "", err
		}
		if !canAccess {
			return "", errors.New("无权查看该工单结果")
		}
	}

	// 只有执行完成的导出工单才允许下载结果文件。
	if ticket.Status != "executed" {
		return "", errors.New("工单未执行完成，无法下载")
	}

	// ExecuteResult 支持新版 JSON 元信息，也兼容旧版 FILE:path 格式。
	res := ticket.ExecuteResult
	var filePath string

	if strings.HasPrefix(res, "FILE:") {
		filePath = strings.TrimPrefix(res, "FILE:")
	} else {
		var meta struct {
			File string `json:"file"`
		}
		if err := json.Unmarshal([]byte(res), &meta); err == nil && meta.File != "" {
			filePath = meta.File
		}
	}

	if filePath == "" {
		return "", fmt.Errorf("无法解析导出文件路径。结果内容: %s", res)
	}

	// 返回文件前确认路径存在且不是目录。
	if info, err := os.Stat(filePath); os.IsNotExist(err) {
		workingDirectory, getwdErr := os.Getwd()
		fields := []zap.Field{
			zap.Uint("ticket_id", ticket.ID),
			zap.String("file_path", filePath),
			zap.NamedError("stat_error", err),
		}
		if getwdErr != nil {
			fields = append(fields, zap.NamedError("working_directory_error", getwdErr))
		} else {
			fields = append(fields, zap.String("working_directory", workingDirectory))
		}
		logger.Log.Error("Export file not found", fields...)
		return "", fmt.Errorf("导出文件不存在: %s", filePath)
	} else if info.IsDir() {
		return "", fmt.Errorf("路径是一个目录: %s", filePath)
	}

	return filePath, nil
}

func buildTicketExecutionSummary(statementResults []dto.StatementExecutionResult, affectedRows int64) (string, string) {
	total := len(statementResults)
	succeeded := 0
	failed := 0
	skipped := 0
	failedIndex := 0
	failedMessage := ""
	for _, result := range statementResults {
		switch result.Status {
		case "success":
			succeeded++
		case "failed":
			failed++
			if failedIndex == 0 {
				failedIndex = result.Index
				failedMessage = result.Message
			}
		case "skipped":
			skipped++
		}
	}

	status := "executed"
	if failed > 0 {
		if succeeded > 0 {
			status = "partial_success"
		} else {
			status = "failed"
		}
	}

	summary := fmt.Sprintf("共 %d 条，成功 %d，失败 %d，未执行 %d，总影响行数 %d", total, succeeded, failed, skipped, affectedRows)
	if failedIndex > 0 {
		summary = fmt.Sprintf("%s。第 %d 条执行失败：%s", summary, failedIndex, failedMessage)
	}
	if status == "partial_success" {
		summary = "部分执行失败：" + summary
	}
	return status, summary
}

func stripTicketAuditComment(statementResults []dto.StatementExecutionResult, originalSQL string) []dto.StatementExecutionResult {
	if len(statementResults) == 0 {
		return statementResults
	}
	parser := sqlparser.NewTestParser()
	pieces, err := parser.SplitStatementToPieces(originalSQL)
	if err != nil {
		return statementResults
	}
	originalStatements := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		if trimmed := strings.TrimSpace(piece); trimmed != "" {
			originalStatements = append(originalStatements, trimmed)
		}
	}
	for i := range statementResults {
		if i < len(originalStatements) {
			statementResults[i].SQL = originalStatements[i]
		}
	}
	return statementResults
}

/*
executeSQL 执行非导出类工单 SQL。
执行时注入工单和执行人注释用于数据库侧审计；风险阻断统一由工单创建前的规则引擎负责。
*/
func (s *ticketService) executeSQL(ticket *model.SQLTicket, executorName string) (*ticketExecutionResult, error) {
	// 注入工单和执行人信息到 SQL 注释，便于数据库审计日志追踪来源。
	comment := fmt.Sprintf("/* TicketID: %d, Title: %s, Executor: %s */", ticket.ID, ticket.Title, executorName)
	finalSQL := fmt.Sprintf("%s %s", comment, ticket.SQLContent)

	// 真实执行由 DataSourceService 负责，包括超时、连接池、脱敏和结果封装。
	res, err := s.dsService.ExecuteSQL(ticket.DataSourceID, finalSQL, ticket.Database)
	if res != nil {
		res.StatementResults = stripTicketAuditComment(res.StatementResults, ticket.SQLContent)
	}
	if err != nil {
		if res == nil {
			return nil, err
		}
		status, summary := buildTicketExecutionSummary(res.StatementResults, res.AffectedRows)
		return &ticketExecutionResult{
			Message:             summary,
			Status:              status,
			AffectedRows:        res.AffectedRows,
			ExecutionDurationMS: res.ExecutionTime,
			StatementResults:    res.StatementResults,
		}, err
	}

	// 多语句执行保存结构化摘要；单语句保留原有成功文案。
	if len(res.StatementResults) > 0 {
		status, summary := buildTicketExecutionSummary(res.StatementResults, res.AffectedRows)
		return &ticketExecutionResult{
			Message:             summary,
			Status:              status,
			AffectedRows:        res.AffectedRows,
			ExecutionDurationMS: res.ExecutionTime,
			StatementResults:    res.StatementResults,
		}, nil
	}

	// 只保存执行摘要，避免把完整结果集写入工单表导致记录过大。
	var summary string
	isResultSet := len(res.Rows) > 0 && res.AffectedRows == 0 && strings.EqualFold(ticket.TicketType, "query")
	if isResultSet {
		summary = fmt.Sprintf(
			"SQL执行成功。\n返回列数: %d, 返回行数(预览): %d, 执行耗时: %d ms",
			len(res.Columns),
			len(res.Rows),
			res.ExecutionTime,
		)
	} else {
		summary = fmt.Sprintf(
			"SQL执行成功。\n影响行数: %d, 执行耗时: %d ms",
			res.AffectedRows,
			res.ExecutionTime,
		)
	}
	return &ticketExecutionResult{
		Message:             summary,
		AffectedRows:        res.AffectedRows,
		ExecutionDurationMS: res.ExecutionTime,
		StatementResults:    res.StatementResults,
	}, nil
}

/*
CheckSyntax 对工单 SQL 执行提交前检查。
它会校验 SQL 非空、导出工单必须为 SELECT，并调用数据源 ExplainSQL 返回预估影响行数。
*/
func (s *ticketService) CheckSyntax(datasourceID uint, database, sqlContent, ticketType string) (*dto.ExplainResult, error) {
	// SQL 内容为空时直接拒绝，避免后续解析和 Explain 出现误导性错误。
	if strings.TrimSpace(sqlContent) == "" {
		return nil, errors.New("SQL content cannot be empty")
	}

	// 导出工单只允许 SELECT，避免通过导出入口绕过变更工单流程。
	if ticketType == "export" {
		parser := sqlparser.NewTestParser()
		stmt, err := parser.Parse(sqlContent)
		if err != nil {
			return nil, fmt.Errorf("SQL parse failed: %v", err)
		}
		if _, ok := stmt.(*sqlparser.Select); !ok {
			return nil, errors.New("导出工单仅支持 SELECT 语句")
		}
	}

	// ExplainSQL 负责连接目标数据源并返回执行计划估算结果。
	explainRes, err := s.dsService.ExplainSQL(datasourceID, sqlContent, database)
	if err != nil {
		return nil, fmt.Errorf("syntax check failed: %v", err)
	}

	return explainRes, nil
}

/*
executeExportAsync 异步执行导出工单并写入本地 CSV 文件。
它通过 QueryStream 逐行读取结果，避免一次性加载大量数据到内存。
*/
func (s *ticketService) executeExportAsync(ticketID uint, approverName string, clientIP string, approverID uint) {
	start := time.Now()

	defer func() {
		if r := recover(); r != nil {
			logger.Log.Error("Recovered panic while executing export ticket",
				zap.Uint("ticket_id", ticketID),
				zap.Any("panic", r),
			)
			panicMessage := fmt.Sprintf("System Panic during Export: %v", r)
			if err := s.updateTicketStatusAsync(ticketID, "failed", panicMessage); err == nil {
				s.notifyTicketResultByID(ticketID, approverName, panicMessage)
			}
		}
	}()

	t, err := s.ticketRepo.FindByID(ticketID)
	if err != nil {
		logger.Log.Error("Failed to load ticket for export execution",
			zap.Uint("ticket_id", ticketID),
			zap.Error(err),
		)
		return
	}

	// 导出文件统一写入 storage/exports 目录。
	exportDir := "storage/exports"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		message := fmt.Sprintf("Failed to create export dir: %v", err)
		if updateErr := s.updateTicketStatusAsync(ticketID, "failed", message); updateErr == nil {
			s.notifyTicketResultByID(ticketID, approverName, message)
		}
		return
	}

	fileName := fmt.Sprintf("export_%d_%d.csv", t.ID, time.Now().Unix())
	filePath := filepath.Join(exportDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		message := fmt.Sprintf("Failed to create export file: %v", err)
		if updateErr := s.updateTicketStatusAsync(ticketID, "failed", message); updateErr == nil {
			s.notifyTicketResultByID(ticketID, approverName, message)
		}
		return
	}
	defer file.Close()

	// 写入 UTF-8 BOM，提升中文内容在 Excel 中打开时的兼容性。
	_, _ = file.Write([]byte("\xEF\xBB\xBF"))
	writer := csv.NewWriter(file)
	defer writer.Flush()

	rowCount := 0
	dataSize := 0

	// QueryStream 每返回一行就写入 CSV，第一行额外写入表头。
	onRow := func(columns []string, row []interface{}) error {
		if rowCount == 0 {
			if err := writer.Write(columns); err != nil {
				return err
			}
		}

		// CSV 中统一写字符串，nil 显示为 NULL。
		record := make([]string, len(row))
		for i, v := range row {
			if v == nil {
				record[i] = "NULL"
			} else {
				record[i] = fmt.Sprintf("%v", v)
				// 记录文本长度作为导出大小的近似值。
				dataSize += len(record[i])
			}
		}

		if err := writer.Write(record); err != nil {
			return err
		}
		rowCount++

		return nil
	}

	// 导出 SQL 同样注入工单和执行人注释，保持审计来源一致。
	comment := fmt.Sprintf("/* AsyncExport TicketID: %d, Executor: %s */", t.ID, approverName)
	finalSQL := fmt.Sprintf("%s %s", comment, t.SQLContent)

	// 流式执行查询并写入 CSV。
	err = s.dsService.QueryStream(t.DataSourceID, finalSQL, t.Database, onRow)

	duration := time.Since(start)

	// 执行失败时更新工单状态并记录审计日志。
	if err != nil {
		message := fmt.Sprintf("Export Failed: %v", err)
		if updateErr := s.updateTicketStatusAsync(ticketID, "failed", message); updateErr != nil {
			return
		}

		_ = s.auditLogSvc.Log(
			approverID,
			approverName,
			"AsyncExecComplete",
			"System",
			fmt.Sprintf("Title: %s\nAction: Export Failed\nError: %v\nTotal Duration: %v", t.Title, err, duration),
			false,
		)
		s.notifyTicketResultByID(ticketID, approverName, message)
		return
	}
	resJSON := fmt.Sprintf(`{"file":"%s","rows":%d,"size":%d}`, filePath, rowCount, dataSize)

	// 更新工单为 executed，并记录导出文件元信息。
	if err := s.updateTicketStatusAsync(ticketID, "executed", resJSON); err != nil {
		return
	}

	// 只有最终状态可靠落库后，才记录成功审计并发送通知。
	_ = s.auditLogSvc.Log(
		approverID,
		approverName,
		"AsyncExecComplete",
		"System",
		fmt.Sprintf("Title: %s\nAction: Export Complete\nFile: %s\nRows: %d\nTotal Duration: %v", t.Title, fileName, rowCount, duration),
		true,
	)

	s.notifyTicketResultByID(ticketID, approverName, "导出任务已完成")
}
