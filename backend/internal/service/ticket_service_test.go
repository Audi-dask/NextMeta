package service

import (
	"errors"
	"testing"

	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"

	"github.com/stretchr/testify/assert"
)

func TestRetryTicketUpdate(t *testing.T) {
	t.Run("succeeds after retry", func(t *testing.T) {
		calls := 0
		attempts, err := retryTicketUpdate(func() error {
			calls++
			if calls < 3 {
				return errors.New("temporary failure")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
		assert.Equal(t, 3, calls)
	})

	t.Run("returns final error", func(t *testing.T) {
		calls := 0
		expectedErr := errors.New("persistent failure")
		attempts, err := retryTicketUpdate(func() error {
			calls++
			return expectedErr
		})

		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, ticketStatusUpdateMaxAttempts, attempts)
		assert.Equal(t, ticketStatusUpdateMaxAttempts, calls)
	})
}

func TestBuildTicketExecutionSummary(t *testing.T) {
	tests := []struct {
		name           string
		results        []dto.StatementExecutionResult
		affectedRows   int64
		expectedStatus string
		expectedText   string
	}{
		{
			name: "partial success",
			results: []dto.StatementExecutionResult{
				{Index: 1, Status: "success", AffectedRows: 3},
				{Index: 2, Status: "failed", Message: "table not found"},
				{Index: 3, Status: "skipped"},
			},
			affectedRows: 3, expectedStatus: "partial_success",
			expectedText: "部分执行失败：共 3 条，成功 1，失败 1，未执行 1，总影响行数 3。第 2 条执行失败：table not found",
		},
		{
			name:           "all failed",
			results:        []dto.StatementExecutionResult{{Index: 1, Status: "failed", Message: "syntax error"}},
			expectedStatus: "failed",
			expectedText:   "共 1 条，成功 0，失败 1，未执行 0，总影响行数 0。第 1 条执行失败：syntax error",
		},
		{
			name:         "all success",
			results:      []dto.StatementExecutionResult{{Index: 1, Status: "success", AffectedRows: 2}},
			affectedRows: 2, expectedStatus: "executed",
			expectedText: "共 1 条，成功 1，失败 0，未执行 0，总影响行数 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, summary := buildTicketExecutionSummary(tt.results, tt.affectedRows)
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedText, summary)
		})
	}
}

func TestStripTicketAuditCommentUsesOriginalStatements(t *testing.T) {
	results := []dto.StatementExecutionResult{
		{Index: 1, SQL: "/* TicketID: 1 */ UPDATE users SET enabled = 1", Status: "success"},
		{Index: 2, SQL: "UPDATE users SET enabled = 0", Status: "failed"},
	}

	actual := stripTicketAuditComment(results, "UPDATE users SET enabled = 1;\nUPDATE users SET enabled = 0;")

	assert.Equal(t, "UPDATE users SET enabled = 1", actual[0].SQL)
	assert.Equal(t, "UPDATE users SET enabled = 0", actual[1].SQL)
}

func TestExecuteSQLDoesNotBlockDropKeywordsInsideStringLiteral(t *testing.T) {
	dsService := &recordingDataSourceService{
		executeResult: &dto.ExecuteSQLResponse{AffectedRows: 0, ExecutionTime: 1},
	}
	service := &ticketService{dsService: dsService}
	ticket := &model.SQLTicket{
		DataSourceID: 1,
		Database:     "audit_test",
		Title:        "guard probe",
		SQLContent:   "CREATE TEMPORARY TABLE audit_guard_probe (note VARCHAR(64) NOT NULL DEFAULT 'drop table');",
		TicketType:   "DDL",
	}

	_, err := service.executeSQL(ticket, "admin")

	assert.NoError(t, err)
	assert.Contains(t, dsService.executedSQL, "DEFAULT 'drop table'")
}

type recordingDataSourceService struct {
	executedSQL   string
	executeResult *dto.ExecuteSQLResponse
}

func (s *recordingDataSourceService) Create(*dto.CreateDataSourceRequest) error { return nil }
func (s *recordingDataSourceService) Update(*dto.UpdateDataSourceRequest) error { return nil }
func (s *recordingDataSourceService) Delete(uint) error                         { return nil }
func (s *recordingDataSourceService) List() ([]dto.DataSourceResponse, error)   { return nil, nil }
func (s *recordingDataSourceService) TestConnection(uint) (string, error)       { return "", nil }
func (s *recordingDataSourceService) TestConnectionConfig(*dto.TestDataSourceConnectionRequest) (string, error) {
	return "", nil
}
func (s *recordingDataSourceService) FetchSchemas(uint, bool) ([]*dto.SchemaNode, error) {
	return nil, nil
}
func (s *recordingDataSourceService) FetchColumns(uint, string, string) ([]*dto.SchemaNode, error) {
	return nil, nil
}
func (s *recordingDataSourceService) ExecuteSQL(_ uint, sql, _ string) (*dto.ExecuteSQLResponse, error) {
	s.executedSQL = sql
	return s.executeResult, nil
}
func (s *recordingDataSourceService) CheckSyntax(uint, string) error { return nil }
func (s *recordingDataSourceService) ExplainSQL(uint, string, string) (*dto.ExplainResult, error) {
	return nil, nil
}
func (s *recordingDataSourceService) QueryStream(uint, string, string, func([]string, []interface{}) error) error {
	return nil
}
func (s *recordingDataSourceService) Copy(uint) error { return nil }
func (s *recordingDataSourceService) Get(uint) (*model.DataSource, error) {
	return nil, nil
}
