package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/audit"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/service"
	"nextmeta-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type failingAuditService struct{}

func (failingAuditService) Audit(audit.Request) (*service.AuditReport, error) {
	return nil, errors.New("audit unavailable")
}

type ticketServiceSpy struct {
	createCalled bool
}

func (s *ticketServiceSpy) CreateTicket(uint, string, string, string, uint, string, uint, bool) (*model.SQLTicket, error) {
	s.createCalled = true
	return &model.SQLTicket{}, nil
}
func (*ticketServiceSpy) ApproveTicket(uint, uint, string, string, string) error { return nil }
func (*ticketServiceSpy) WithdrawTicket(uint, uint) error                        { return nil }
func (*ticketServiceSpy) GetMyTickets(uint) ([]model.SQLTicket, error)           { return nil, nil }
func (*ticketServiceSpy) GetPendingTickets(uint) ([]model.SQLTicket, error)      { return nil, nil }
func (*ticketServiceSpy) GetTicketDetail(uint, uint, string) (*model.SQLTicket, error) {
	return nil, nil
}
func (*ticketServiceSpy) ExportTicketResult(uint, uint) (string, error) { return "", nil }
func (*ticketServiceSpy) CheckSyntax(uint, string, string, string) (*dto.ExplainResult, error) {
	return nil, nil
}
func (*ticketServiceSpy) GetApprovalHistory(uint) ([]model.SQLTicket, error) { return nil, nil }
func (*ticketServiceSpy) GetTicketsForApprover(uint, string, string) ([]model.SQLTicket, error) {
	return nil, nil
}
func (*ticketServiceSpy) GetTicketAuditLogs(string) ([]model.SQLTicket, error) { return nil, nil }
func (*ticketServiceSpy) GetDatasourceApprovers(uint, uint) ([]service.TicketApproverResponse, error) {
	return nil, nil
}

func newTicketRequestContext(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("userID", uint(15))
	ctx.Set("role", "developer")
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tickets", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
} {
	t.Helper()
	var responseBody struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &responseBody))
	return responseBody
}

func TestCreateTicketRejectsForceSubmission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewTicketHandler(nil, nil)
	ctx, recorder := newTicketRequestContext([]byte(`{
		"title":"Force测试",
		"sql_content":"UPDATE nm_repro_force SET execute_count = execute_count + 1;",
		"ticket_type":"DML",
		"datasource_id":4,
		"database":"nm_audit_test",
		"approver_id":1,
		"force":true
	}`))

	handler.CreateTicket(ctx)

	responseBody := decodeResponse(t, recorder)
	assert.Equal(t, 2000, responseBody.Code)
	assert.Equal(t, "当前不允许强制提交工单", responseBody.Msg)
}

func TestCreateTicketRejectsAuditServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()

	ticketSvc := &ticketServiceSpy{}
	handler := NewTicketHandler(ticketSvc, failingAuditService{})
	ctx, recorder := newTicketRequestContext([]byte(`{
		"title":"审核异常测试",
		"sql_content":"UPDATE nm_repro_force SET status = 'inactive' WHERE id = 1;",
		"ticket_type":"DML",
		"datasource_id":4,
		"database":"nm_audit_test",
		"approver_id":1,
		"force":false
	}`))

	handler.CreateTicket(ctx)

	responseBody := decodeResponse(t, recorder)
	assert.Equal(t, 2000, responseBody.Code)
	assert.Equal(t, "SQL审计失败，请稍后重试", responseBody.Msg)
	assert.False(t, ticketSvc.createCalled)
}
