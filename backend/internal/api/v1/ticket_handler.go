package v1

import (
	"encoding/json"
	"net/http"
	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/audit"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/jwt"
	"nextmeta-backend/pkg/logger"
	"nextmeta-backend/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

/*
TicketHandler 承接 SQL 工单相关 HTTP 请求。
它组合工单服务和审核服务，负责工单提交、语法检测、审批流、列表查询、详情和结果导出入口。
*/
type TicketHandler struct {
	ticketService service.TicketService
	auditSvc      service.AuditService
}

type ticketUserResponse struct {
	ID       uint   `json:"ID"`
	Username string `json:"Username"`
	RealName string `json:"RealName"`
	Role     string `json:"role,omitempty"`
}

type ticketDataSourceResponse struct {
	ID          uint   `json:"ID"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type ticketApprovalResponse struct {
	ID         uint               `json:"ID"`
	CreatedAt  interface{}        `json:"CreatedAt,omitempty"`
	ApproverID uint               `json:"ApproverID"`
	Approver   ticketUserResponse `json:"Approver"`
	Action     string             `json:"Action"`
	Comment    string             `json:"Comment,omitempty"`
}

type ticketResponse struct {
	ID                  uint                           `json:"ID"`
	CreatedAt           interface{}                    `json:"CreatedAt"`
	UpdatedAt           interface{}                    `json:"UpdatedAt,omitempty"`
	CreatorID           uint                           `json:"CreatorID"`
	Creator             ticketUserResponse             `json:"Creator"`
	ApproverID          uint                           `json:"ApproverID"`
	Approver            ticketUserResponse             `json:"Approver"`
	GroupID             uint                           `json:"GroupID"`
	DataSourceID        uint                           `json:"DataSourceID"`
	DataSource          ticketDataSourceResponse       `json:"DataSource"`
	Database            string                         `json:"Database"`
	Title               string                         `json:"Title"`
	SQLContent          string                         `json:"SQLContent"`
	TicketType          string                         `json:"TicketType"`
	IsForce             bool                           `json:"IsForce"`
	Status              string                         `json:"Status"`
	ExecuteResult       string                         `json:"ExecuteResult"`
	StatementResults    []dto.StatementExecutionResult `json:"StatementResults,omitempty"`
	ExecutorID          uint                           `json:"ExecutorID"`
	Executor            ticketUserResponse             `json:"Executor"`
	ExecutorName        string                         `json:"ExecutorName"`
	ExecutedAt          interface{}                    `json:"ExecutedAt"`
	AffectedRows        int64                          `json:"AffectedRows"`
	ExecutionDurationMS int64                          `json:"ExecutionDurationMS"`
	Approvals           []ticketApprovalResponse       `json:"Approvals"`
}

func toTicketUserResponse(user model.User) ticketUserResponse {
	return ticketUserResponse{ID: user.ID, Username: user.Username, RealName: user.RealName, Role: user.Role}
}

func toTicketDataSourceResponse(ds model.DataSource) ticketDataSourceResponse {
	return ticketDataSourceResponse{ID: ds.ID, Name: ds.Name, Environment: ds.Environment}
}

func toTicketApprovalResponse(approval model.TicketApproval) ticketApprovalResponse {
	return ticketApprovalResponse{
		ID:         approval.ID,
		CreatedAt:  approval.CreatedAt,
		ApproverID: approval.ApproverID,
		Approver:   toTicketUserResponse(approval.Approver),
		Action:     approval.Action,
		Comment:    approval.Comment,
	}
}

func toTicketResponse(ticket model.SQLTicket) ticketResponse {
	approvals := make([]ticketApprovalResponse, 0, len(ticket.Approvals))
	for _, approval := range ticket.Approvals {
		approvals = append(approvals, toTicketApprovalResponse(approval))
	}

	statementResults := make([]dto.StatementExecutionResult, 0)
	if ticket.StatementResults != "" {
		_ = json.Unmarshal([]byte(ticket.StatementResults), &statementResults)
	}

	return ticketResponse{
		ID:                  ticket.ID,
		CreatedAt:           ticket.CreatedAt,
		UpdatedAt:           ticket.UpdatedAt,
		CreatorID:           ticket.CreatorID,
		Creator:             toTicketUserResponse(ticket.Creator),
		ApproverID:          ticket.ApproverID,
		Approver:            toTicketUserResponse(ticket.Approver),
		GroupID:             ticket.GroupID,
		DataSourceID:        ticket.DataSourceID,
		DataSource:          toTicketDataSourceResponse(ticket.DataSource),
		Database:            ticket.Database,
		Title:               ticket.Title,
		SQLContent:          ticket.SQLContent,
		TicketType:          ticket.TicketType,
		IsForce:             ticket.IsForce,
		Status:              ticket.Status,
		ExecuteResult:       ticket.ExecuteResult,
		StatementResults:    statementResults,
		ExecutorID:          ticket.ExecutorID,
		Executor:            toTicketUserResponse(ticket.Executor),
		ExecutorName:        ticket.ExecutorName,
		ExecutedAt:          ticket.ExecutedAt,
		AffectedRows:        ticket.AffectedRows,
		ExecutionDurationMS: ticket.ExecutionDurationMS,
		Approvals:           approvals,
	}
}

func toTicketListResponses(tickets []model.SQLTicket) []dto.TicketListResponse {
	responses := make([]dto.TicketListResponse, 0, len(tickets))
	for _, ticket := range tickets {
		responses = append(responses, dto.TicketListResponse{
			ID:         ticket.ID,
			CreatedAt:  ticket.CreatedAt,
			UpdatedAt:  ticket.UpdatedAt,
			Title:      ticket.Title,
			TicketType: ticket.TicketType,
			Status:     ticket.Status,
			Database:   ticket.Database,
			IsForce:    ticket.IsForce,
			Creator: dto.TicketListUserResponse{
				ID: ticket.Creator.ID, Username: ticket.Creator.Username, RealName: ticket.Creator.RealName,
			},
			Approver: dto.TicketListUserResponse{
				ID: ticket.Approver.ID, Username: ticket.Approver.Username, RealName: ticket.Approver.RealName,
			},
			DataSource: dto.TicketListDataSourceResponse{
				ID: ticket.DataSource.ID, Name: ticket.DataSource.Name, Environment: ticket.DataSource.Environment,
			},
			Executor: dto.TicketListUserResponse{
				ID: ticket.Executor.ID, Username: ticket.Executor.Username, RealName: ticket.Executor.RealName,
			},
			ExecutorName: ticket.ExecutorName,
			ExecutedAt:   ticket.ExecutedAt,
		})
	}
	return responses
}

/*
isSupportedChangeTicketType 判断工单类型是否属于当前支持的变更类型。
目前工单流程只允许 DDL 和 DML，查询类 SQL 走数据源查询窗口。
*/
func isSupportedChangeTicketType(ticketType string) bool {
	switch strings.ToUpper(strings.TrimSpace(ticketType)) {
	case "DDL", "DML":
		return true
	default:
		return false
	}
}

/*
NewTicketHandler 创建工单接口 Handler。
ticketService 负责工单状态流转，auditSvc 负责提交前和语法检测时的 SQL 静态审核。
*/
func NewTicketHandler(ticketService service.TicketService, auditSvc service.AuditService) *TicketHandler {
	return &TicketHandler{ticketService: ticketService, auditSvc: auditSvc}
}

/*
CreateTicket 处理 DDL/DML 工单创建请求。
创建前会校验当前角色是否允许提交工单，并执行 SQL 静态审核；阻断级违规需要 force 才能继续提交。
*/
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	role := jwt.GetRoleFromContext(c)
	if !model.CanSubmitTicket(role) {
		response.Fail(c, response.CodeError, "当前角色不允许提交 DDL/DML 工单")
		return
	}

	var req struct {
		Title        string `json:"title" binding:"required,max=15"`
		SQLContent   string `json:"sql_content" binding:"required"`
		TicketType   string `json:"ticket_type" binding:"required"`
		DataSourceID uint   `json:"datasource_id" binding:"required"`
		Database     string `json:"database"`
		ApproverID   uint   `json:"approver_id" binding:"required"`
		Force        bool   `json:"force"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "请完整填写工单信息，并选择有效的审核人")
		return
	}

	// 前端固定提交 force=false；这里用于拒绝绕过前端直接构造的强制提交请求。
	if req.Force {
		response.Fail(c, response.CodeError, "当前不允许强制提交工单")
		return
	}

	if !isSupportedChangeTicketType(req.TicketType) {
		response.Fail(c, response.CodeInvalidParam, "工单类型仅支持 DDL 或 DML")
		return
	}

	// 创建工单前先执行 SQL 静态审核，阻断级结果默认不允许继续提交。
	auditReport, err := h.auditSvc.Audit(audit.Request{
		SQLContent:   req.SQLContent,
		TicketType:   req.TicketType,
		DataSourceID: req.DataSourceID,
		Database:     req.Database,
	})
	if err != nil {
		logger.Log.Error("Audit failed", zap.Error(err))
		response.Fail(c, response.CodeError, "SQL审计失败，请稍后重试")
		return
	} else if auditReport.HasBlocker {
		response.FailWithData(c, response.CodeError, "SQL审计未通过，存在高危操作", gin.H{
			"audit_report": auditReport,
		})
		return
	}

	ticket, err := h.ticketService.CreateTicket(
		userID,
		req.Title,
		req.SQLContent,
		req.TicketType,
		req.DataSourceID,
		req.Database,
		req.ApproverID,
		req.Force,
	)

	if err != nil {
		logger.Log.Error("Failed to create ticket", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, toTicketResponse(*ticket))
}

/*
ApproveTicket 处理工单审批请求。
审批动作只允许 approve 或 reject，审批人、审批意见和客户端 IP 会交给 service 层记录。
*/
func (h *TicketHandler) ApproveTicket(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	ticketID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Action  string `json:"action" binding:"required,oneof=approve reject"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	err := h.ticketService.ApproveTicket(uint(ticketID), userID, req.Action, req.Comment, c.ClientIP())
	if err != nil {
		logger.Log.Error("Failed to approve ticket", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
WithdrawTicket 处理提交人主动撤回待审批工单。
撤回权限和状态校验由 service 层保证。
*/
func (h *TicketHandler) WithdrawTicket(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	ticketID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.ticketService.WithdrawTicket(uint(ticketID), userID); err != nil {
		logger.Log.Error("Failed to withdraw ticket", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, nil)
}

/*
GetDatasourceApprovers 根据数据源返回当前用户可选择的审批人。
service 层会结合用户权限和数据源所属用户组决定可用审批人列表。
*/
func (h *TicketHandler) GetDatasourceApprovers(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	datasourceID, err := strconv.ParseUint(c.Query("datasource_id"), 10, 32)
	if err != nil || datasourceID == 0 {
		response.Fail(c, response.CodeInvalidParam, "请选择有效的数据源")
		return
	}

	approvers, err := h.ticketService.GetDatasourceApprovers(userID, uint(datasourceID))
	if err != nil {
		logger.Log.Error("Failed to get datasource approvers", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	response.Success(c, approvers)
}

/*
GetMyTickets 返回当前登录用户提交的工单列表。
用户 ID 来自 JWT 上下文，不依赖客户端传参。
*/
func (h *TicketHandler) GetMyTickets(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)

	tickets, err := h.ticketService.GetMyTickets(userID)
	if err != nil {
		logger.Log.Error("Failed to get my tickets", zap.Error(err))
		response.Fail(c, response.CodeError, "获取工单失败")
		return
	}

	response.Success(c, toTicketListResponses(tickets))
}

/*
GetPendingTickets 返回当前登录用户待审批的工单列表。
该接口用于审批工作台展示未处理工单。
*/
func (h *TicketHandler) GetPendingTickets(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)

	tickets, err := h.ticketService.GetPendingTickets(userID)
	if err != nil {
		logger.Log.Error("Failed to get pending tickets", zap.Error(err))
		response.Fail(c, response.CodeError, "获取待审批工单失败")
		return
	}

	response.Success(c, toTicketListResponses(tickets))
}

/*
GetApprovalHistory 返回当前登录用户的审批历史。
用于展示审批人已经处理过的工单记录。
*/
func (h *TicketHandler) GetApprovalHistory(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)

	tickets, err := h.ticketService.GetApprovalHistory(userID)
	if err != nil {
		logger.Log.Error("Failed to get approval history", zap.Error(err))
		response.Fail(c, response.CodeError, "获取审批历史失败")
		return
	}

	response.Success(c, toTicketListResponses(tickets))
}

/*
GetApproverTickets 返回审批人相关工单列表。
支持按 status 和 creator 查询参数筛选，具体过滤逻辑由 service 层处理。
*/
func (h *TicketHandler) GetApproverTickets(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	status := c.Query("status")
	creator := c.Query("creator")

	tickets, err := h.ticketService.GetTicketsForApprover(userID, status, creator)
	if err != nil {
		logger.Log.Error("Failed to get approver tickets", zap.Error(err))
		response.Fail(c, response.CodeError, "获取工单失败")
		return
	}

	response.Success(c, toTicketListResponses(tickets))
}

/*
GetTicketAuditLogs 返回工单审计日志列表。
ticket_type 查询参数为空或 all 时不过滤类型，指定类型时只允许 DDL 或 DML。
*/
func (h *TicketHandler) GetTicketAuditLogs(c *gin.Context) {
	ticketType := strings.ToUpper(strings.TrimSpace(c.Query("ticket_type")))
	if ticketType != "" && ticketType != "all" && !isSupportedChangeTicketType(ticketType) {
		response.Fail(c, response.CodeInvalidParam, "工单类型仅支持 DDL 或 DML")
		return
	}

	tickets, err := h.ticketService.GetTicketAuditLogs(ticketType)
	if err != nil {
		logger.Log.Error("Failed to get ticket audit logs", zap.Error(err))
		response.Fail(c, response.CodeError, "获取工单审计日志失败")
		return
	}

	response.Success(c, toTicketListResponses(tickets))
}

/*
GetTicketDetail 返回指定工单详情。
工单 ID 来自路径参数，详情聚合逻辑由 service 层处理。
*/
func (h *TicketHandler) GetTicketDetail(c *gin.Context) {
	ticketID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := jwt.GetUserIDFromContext(c)
	role := jwt.GetRoleFromContext(c)

	ticket, err := h.ticketService.GetTicketDetail(uint(ticketID), userID, role)
	if err != nil {
		if err.Error() == "无权查看该工单" {
			response.FailWithStatus(c, http.StatusForbidden, response.CodeError, err.Error())
			return
		}
		logger.Log.Error("Failed to get ticket detail", zap.Error(err))
		response.Fail(c, response.CodeError, "获取工单详情失败")
		return
	}

	response.Success(c, toTicketResponse(*ticket))
}

/*
ExportTicketResult 导出当前用户可访问的工单执行结果。
service 层负责权限校验和生成文件，Handler 负责把生成的 CSV 作为附件返回。
*/
func (h *TicketHandler) ExportTicketResult(c *gin.Context) {
	userID := jwt.GetUserIDFromContext(c)
	ticketID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	filePath, err := h.ticketService.ExportTicketResult(userID, uint(ticketID))
	if err != nil {
		logger.Log.Error("Failed to export ticket result", zap.Error(err))
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	// 将 service 生成的导出文件作为附件返回给浏览器。
	c.FileAttachment(filePath, "export_result.csv") // TODO: cleaner filename if needed
}

/*
CheckSyntax 对工单 SQL 执行提交前静态审核。
该接口不创建工单，只返回 audit_report 供前端展示审核结果。
*/
func (h *TicketHandler) CheckSyntax(c *gin.Context) {
	role := jwt.GetRoleFromContext(c)
	if !model.CanSubmitTicket(role) {
		response.Fail(c, response.CodeError, "当前角色不允许检测 DDL/DML 工单")
		return
	}

	var req struct {
		SQLContent   string `json:"sql_content" binding:"required"`
		DataSourceID uint   `json:"datasource_id"`
		Database     string `json:"database"`
		TicketType   string `json:"ticket_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeError, err.Error())
		return
	}

	if !isSupportedChangeTicketType(req.TicketType) {
		response.Fail(c, response.CodeInvalidParam, "工单类型仅支持 DDL 或 DML")
		return
	}

	auditReport, err := h.auditSvc.Audit(audit.Request{
		SQLContent:   req.SQLContent,
		TicketType:   req.TicketType,
		DataSourceID: req.DataSourceID,
		Database:     req.Database,
	})
	if err != nil {
		response.Fail(c, response.CodeError, "审计失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"audit_report": auditReport,
	})
}
