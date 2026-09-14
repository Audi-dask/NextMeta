package datasource

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ConnInfo 承载关系型方言共用的连接字段。
// ConnectTimeout 以秒为单位，与持久化的数据源模型保持一致。
type ConnInfo struct {
	Host           string
	Port           int
	User           string
	Password       string
	Database       string
	ConnectTimeout int64
}

// TableMeta 是 information_schema.TABLES 中描述表或视图的一行。
type TableMeta struct {
	Name string `gorm:"column:TABLE_NAME"`
	Type string `gorm:"column:TABLE_TYPE"`
}

// TableSize 是 information_schema.TABLES 中携带存储大小的一行。
type TableSize struct {
	Database    string        `gorm:"column:TABLE_SCHEMA"`
	Table       string        `gorm:"column:TABLE_NAME"`
	DataLength  sql.NullInt64 `gorm:"column:DATA_LENGTH"`
	IndexLength sql.NullInt64 `gorm:"column:INDEX_LENGTH"`
}

// ColumnMeta 是 information_schema.COLUMNS 中的一行。
type ColumnMeta struct {
	Name string `gorm:"column:COLUMN_NAME"`
	Type string `gorm:"column:COLUMN_TYPE"`
	Key  string `gorm:"column:COLUMN_KEY"`
}

// Dialect 隔离关系型数据源各自的 SQL 方言行为。
// MySQL 专属的 SQL、DSN 拼接与会话语句都收敛在具体方言实现中。
type Dialect interface {
	Type() string
	BuildDSN(info ConnInfo, database string) string
	Open(dsn string) (*gorm.DB, error)
	SystemSchemas() []string
	ListDatabases(ctx context.Context, db *gorm.DB) ([]string, error)
	ListTables(ctx context.Context, db *gorm.DB, database string) ([]TableMeta, error)
	ListTableSizes(ctx context.Context, db *gorm.DB) ([]TableSize, error)
	ListColumns(ctx context.Context, db *gorm.DB, database, table string) ([]ColumnMeta, error)
	CheckSyntax(ctx context.Context, db *sql.DB, sql string) error
	Explain(ctx context.Context, db *sql.DB, sql string) (int64, error)
	SessionTimeoutStatement(d time.Duration) string
	TestConnection(info ConnInfo) (string, error)
}

// Registry 按数据源类型分派到对应的方言实现。
// 边界：仅兼容关系型数据库，范围收敛到 MySQL 与 PG，不再扩展其他类型。
type Registry struct {
	dialects map[string]Dialect
}

// NewRegistry 使用给定的方言构建注册表。
func NewRegistry(dialects ...Dialect) *Registry {
	r := &Registry{dialects: make(map[string]Dialect)}
	for _, d := range dialects {
		r.dialects[strings.ToLower(strings.TrimSpace(d.Type()))] = d
	}
	return r
}

// Resolve 返回指定数据源类型对应的方言，空类型回退为 mysql。
func (r *Registry) Resolve(dsType string) (Dialect, error) {
	key := strings.ToLower(strings.TrimSpace(dsType))
	if key == "" {
		key = "mysql"
	}
	if d, ok := r.dialects[key]; ok {
		return d, nil
	}
	return nil, errors.New("only MySQL and PostgreSQL are supported")
}

// DefaultRegistry 是服务层使用的单例注册表。
var DefaultRegistry = NewRegistry(NewMySQLDialect(), NewPostgresDialect())
