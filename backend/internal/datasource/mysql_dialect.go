package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nextmeta-backend/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MySQLDialect implements Dialect for MySQL data sources.
type MySQLDialect struct{}

// NewMySQLDialect returns the MySQL dialect.
func NewMySQLDialect() Dialect {
	return &MySQLDialect{}
}

// Type returns the dialect identifier.
func (m *MySQLDialect) Type() string {
	return "mysql"
}

// BuildDSN builds a MySQL DSN for the given database.
func (m *MySQLDialect) BuildDSN(info ConnInfo, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		info.User, info.Password, info.Host, info.Port, database, info.ConnectTimeout)
}

// Open opens a GORM connection using the project logger.
func (m *MySQLDialect) Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), logger.GormConfig())
}

// SystemSchemas lists MySQL system schemas that should be hidden from structure trees.
func (m *MySQLDialect) SystemSchemas() []string {
	return []string{"information_schema", "mysql", "performance_schema", "sys"}
}

// ListDatabases reads all databases on the instance.
func (m *MySQLDialect) ListDatabases(ctx context.Context, db *gorm.DB) ([]string, error) {
	var databases []string
	if err := db.Raw("SHOW DATABASES").Scan(&databases).Error; err != nil {
		return nil, err
	}
	return databases, nil
}

// ListTables reads tables and views for a database from information_schema.
func (m *MySQLDialect) ListTables(ctx context.Context, db *gorm.DB, database string) ([]TableMeta, error) {
	var tables []TableMeta
	if err := db.Raw("SELECT TABLE_NAME, TABLE_TYPE FROM TABLES WHERE TABLE_SCHEMA = ?", database).Scan(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

// ListTableSizes reads table storage sizes from information_schema.
func (m *MySQLDialect) ListTableSizes(ctx context.Context, db *gorm.DB) ([]TableSize, error) {
	var rows []TableSize
	if err := db.Raw(`
		SELECT TABLE_SCHEMA, TABLE_NAME, DATA_LENGTH, INDEX_LENGTH
		FROM TABLES
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListColumns reads column metadata for a table from information_schema.
func (m *MySQLDialect) ListColumns(ctx context.Context, db *gorm.DB, database, table string) ([]ColumnMeta, error) {
	var columns []ColumnMeta
	if err := db.Raw("SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_KEY FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION", database, table).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// CheckSyntax prepares the statement to validate syntax without executing it.
func (m *MySQLDialect) CheckSyntax(ctx context.Context, db *sql.DB, sql string) error {
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("syntax error: %v", err)
	}
	defer stmt.Close()

	return nil
}

// Explain runs EXPLAIN and returns the maximum estimated rows across plan rows.
func (m *MySQLDialect) Explain(ctx context.Context, db *sql.DB, sql string) (int64, error) {
	explainSQL := "EXPLAIN " + sql
	rows, err := db.QueryContext(ctx, explainSQL)
	if err != nil {
		return 0, fmt.Errorf("EXPLAIN failed for '%s': %v", sql, err)
	}

	columns, err := rows.Columns()
	if err != nil {
		rows.Close()
		return 0, err
	}

	rowsIdx := -1
	for i, col := range columns {
		if strings.ToLower(col) == "rows" {
			rowsIdx = i
			break
		}
	}

	if rowsIdx == -1 {
		rows.Close()
		return 0, nil
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var maxRowsInPlan int64 = 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			rows.Close()
			return 0, err
		}
		val := values[rowsIdx]
		var rowCount int64
		switch v := val.(type) {
		case int64:
			rowCount = v
		case []byte:
			parsed, _ := strconv.ParseInt(string(v), 10, 64)
			rowCount = parsed
		default:
			str := fmt.Sprintf("%v", v)
			parsed, _ := strconv.ParseInt(str, 10, 64)
			rowCount = parsed
		}
		if rowCount > maxRowsInPlan {
			maxRowsInPlan = rowCount
		}
	}
	rows.Close()

	return maxRowsInPlan, nil
}

// SessionTimeoutStatement returns the MySQL session statement limiting execution time.
func (m *MySQLDialect) SessionTimeoutStatement(d time.Duration) string {
	timeoutMs := int(d / time.Millisecond)
	return fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutMs)
}

// TestConnection connects with a short-lived connection and returns SELECT VERSION().
func (m *MySQLDialect) TestConnection(info ConnInfo) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=5s",
		info.User, info.Password, info.Host, info.Port, info.Database)

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
	if err != nil {
		return "", err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return "", err
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var version string
	err = sqlDB.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", err
	}

	return version, nil
}
