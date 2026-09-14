package datasource

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"nextmeta-backend/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresDialect 实现 PostgreSQL 数据源的方言行为。
// 结构树读取保持与 MySQL 一致的“库 -> 表”两层，表固定来自 public schema。
type PostgresDialect struct{}

// NewPostgresDialect 返回 PostgreSQL 方言。
func NewPostgresDialect() Dialect {
	return &PostgresDialect{}
}

// Type 返回方言标识。
func (p *PostgresDialect) Type() string {
	return "postgres"
}

// BuildDSN 构建 PostgreSQL 连接 URL。
// 使用 net/url 对用户名和密码转义，database 为空时回退到 postgres 默认库。
func (p *PostgresDialect) BuildDSN(info ConnInfo, database string) string {
	if database == "" {
		database = "postgres"
	}

	timeout := info.ConnectTimeout
	if timeout <= 0 {
		timeout = 10
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(info.User, info.Password),
		Host:   fmt.Sprintf("%s:%d", info.Host, info.Port),
		Path:   "/" + database,
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	q.Set("connect_timeout", fmt.Sprintf("%d", timeout))
	u.RawQuery = q.Encode()
	return u.String()
}

// Open 使用项目日志配置打开 GORM 连接。
func (p *PostgresDialect) Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), logger.GormConfig())
}

// SystemSchemas 返回结构树中需要隐藏的系统库。
// PostgreSQL 的系统库已在 ListDatabases 中通过 datistemplate 过滤，这里无需额外屏蔽。
func (p *PostgresDialect) SystemSchemas() []string {
	return []string{}
}

// ListDatabases 读取实例下所有可连接的非模板数据库。
func (p *PostgresDialect) ListDatabases(ctx context.Context, db *gorm.DB) ([]string, error) {
	var databases []string
	if err := db.Raw(`
		SELECT datname
		FROM pg_catalog.pg_database
		WHERE datallowconn = true AND NOT datistemplate
		ORDER BY datname
	`).Scan(&databases).Error; err != nil {
		return nil, err
	}
	return databases, nil
}

// ListTables 读取当前连接数据库 public schema 下的表与视图。
// database 参数在此处是冗余的：PostgreSQL 的表列表由连接所指向的数据库决定，逐库连接由 service 层完成。
func (p *PostgresDialect) ListTables(ctx context.Context, db *gorm.DB, database string) ([]TableMeta, error) {
	var tables []TableMeta
	if err := db.Raw(`
		SELECT table_name AS "TABLE_NAME", table_type AS "TABLE_TYPE"
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`).Scan(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

// ListTableSizes 暂不计算 PostgreSQL 表容量，返回空列表以兼容 MySQL 结构树构建流程。
func (p *PostgresDialect) ListTableSizes(ctx context.Context, db *gorm.DB) ([]TableSize, error) {
	return []TableSize{}, nil
}

// ListColumns 读取 public schema 下指定表的字段列表，主键字段标记为 PRI。
func (p *PostgresDialect) ListColumns(ctx context.Context, db *gorm.DB, database, table string) ([]ColumnMeta, error) {
	var columns []ColumnMeta
	query := `
		SELECT a.attname AS "COLUMN_NAME",
		       pg_catalog.format_type(a.atttypid, a.atttypmod) AS "COLUMN_TYPE",
		       CASE WHEN i.indisprimary THEN 'PRI' ELSE '' END AS "COLUMN_KEY"
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_catalog.pg_index i ON i.indrelid = c.oid AND i.indisprimary AND a.attnum = ANY(i.indkey)
		WHERE n.nspname = 'public' AND c.relname = ? AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum
	`
	if err := db.Raw(query, table).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// CheckSyntax 通过 prepare 校验语句语法，不实际执行。
func (p *PostgresDialect) CheckSyntax(ctx context.Context, db *sql.DB, sql string) error {
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("syntax error: %v", err)
	}
	defer stmt.Close()

	return nil
}

// Explain 使用 JSON 执行计划，返回计划树中预估扫描行数的最大值。
func (p *PostgresDialect) Explain(ctx context.Context, db *sql.DB, sql string) (int64, error) {
	explainSQL := "EXPLAIN (FORMAT JSON) " + sql
	var plan string
	if err := db.QueryRowContext(ctx, explainSQL).Scan(&plan); err != nil {
		return 0, fmt.Errorf("EXPLAIN failed for '%s': %v", sql, err)
	}
	return postgresMaxPlanRows(plan), nil
}

// SessionTimeoutStatement 返回 PostgreSQL 会话级执行超时语句，单位为毫秒。
func (p *PostgresDialect) SessionTimeoutStatement(d time.Duration) string {
	return fmt.Sprintf("SET statement_timeout = %d", int(d/time.Millisecond))
}

// TestConnection 使用短连接执行 SELECT version() 返回数据库版本。
func (p *PostgresDialect) TestConnection(info ConnInfo) (string, error) {
	info.ConnectTimeout = 5

	db, err := p.Open(p.BuildDSN(info, info.Database))
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
	if err := sqlDB.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		return "", err
	}

	return version, nil
}

// postgresMaxPlanRows 从 EXPLAIN (FORMAT JSON) 返回的 JSON 中提取最大 "Plan Rows"。
func postgresMaxPlanRows(raw string) int64 {
	var top []interface{}
	if err := json.Unmarshal([]byte(raw), &top); err != nil {
		return 0
	}

	var max int64
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch n := v.(type) {
		case map[string]interface{}:
			if rows, ok := n["Plan Rows"]; ok {
				if f, ok := rows.(float64); ok && int64(f) > max {
					max = int64(f)
				}
			}
			for _, val := range n {
				walk(val)
			}
		case []interface{}:
			for _, val := range n {
				walk(val)
			}
		}
	}
	for _, item := range top {
		walk(item)
	}
	return max
}
