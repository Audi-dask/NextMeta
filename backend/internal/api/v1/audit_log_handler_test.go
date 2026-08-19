package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nextmeta-backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAggregateQueryAuditLogsGroupsByUserAndSession(t *testing.T) {
	baseTime := time.Date(2026, 7, 24, 10, 0, 0, 0, time.Local)
	logs := []model.AuditLog{
		newQueryAuditLog(2, 7, "session-a", baseTime.Add(time.Minute), "SELECT 2", false),
		newQueryAuditLog(1, 7, "session-a", baseTime, "SELECT 1", true),
		newQueryAuditLog(3, 8, "session-a", baseTime.Add(2*time.Minute), "SELECT 3", false),
	}

	items := aggregateQueryAuditLogs(logs)

	require.Len(t, items, 2)
	assert.Equal(t, uint(3), items[0].ID)
	assert.Equal(t, uint(1), items[1].ID)
	require.Len(t, items[1].Records, 2)
	assert.Equal(t, "SELECT 1", items[1].Records[0].SQL)
	assert.Equal(t, "SELECT 2", items[1].Records[1].SQL)
	assert.True(t, items[1].Exported)
}

func TestExecuteSQLRejectsMissingOrInvalidQuerySessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testCases := []struct {
		name string
		body string
	}{
		{name: "missing", body: `{"sql":"SELECT 1","description":"检查数据"}`},
		{name: "invalid", body: `{"sql":"SELECT 1","description":"检查数据","query_session_id":"not-a-uuid"}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Params = gin.Params{{Key: "id", Value: "1"}}
			ctx.Request = httptest.NewRequest(http.MethodPost, "/datasources/1/query", bytes.NewBufferString(testCase.body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			(&DataSourceHandler{}).ExecuteSQL(ctx)

			assert.Contains(t, recorder.Body.String(), `"code":2000`)
		})
	}
}

func newQueryAuditLog(id, userID uint, sessionID string, createdAt time.Time, sql string, exported bool) model.AuditLog {
	return model.AuditLog{
		Model:          gorm.Model{ID: id, CreatedAt: createdAt},
		UserID:         userID,
		Username:       "tester",
		QuerySessionID: sessionID,
		Details:        "检查数据\n\n(查询窗口不执行DDL/DML工单静态审核)",
		SQLContent:     sql,
		Status:         1,
		Exported:       exported,
	}
}
