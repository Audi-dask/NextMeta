package service

import (
	"context"
	"errors"
	"fmt"
	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/datasource"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"

	"nextmeta-backend/pkg/masking"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"vitess.io/vitess/go/vt/sqlparser"
)

// setOpKeywordRe 匹配 MySQL 集合运算关键字 INTERSECT / EXCEPT。
// Vitess v0.24.1 尚不支持这两种语法，但它们的结果列与 UNION 一样按位置对应，
// 脱敏下标合并逻辑完全相同，因此解析失败时临时替换为 UNION 复用既有血缘分析。
var setOpKeywordRe = regexp.MustCompile(`(?i)\b(INTERSECT|EXCEPT)\b`)

/*
DataSourceService 定义数据源模块的核心业务能力。
它覆盖数据源 CRUD、连接测试、库表元数据读取、SQL 执行、语法检查、执行计划预估和流式查询。
*/
type DataSourceService interface {
	Create(req *dto.CreateDataSourceRequest) error
	Update(req *dto.UpdateDataSourceRequest) error
	Delete(id uint) error
	List() ([]dto.DataSourceResponse, error)
	TestConnection(id uint) (string, error)
	TestConnectionConfig(req *dto.TestDataSourceConnectionRequest) (string, error)
	FetchSchemas(id uint, refresh bool) ([]*dto.SchemaNode, error)
	FetchColumns(id uint, dbName string, tableName string) ([]*dto.SchemaNode, error)
	ExecuteSQL(id uint, sql string, dbName string) (*dto.ExecuteSQLResponse, error)
	CheckSyntax(id uint, sql string) error
	ExplainSQL(id uint, sql string, dbName string) (*dto.ExplainResult, error)
	QueryStream(id uint, sql string, dbName string, onRow func([]string, []interface{}) error) error
	Copy(id uint) error
	Get(id uint) (*model.DataSource, error)
}

const schemaCacheTTL = 24 * time.Hour

/*
cachedSchema 保存数据源结构树缓存。
FetchSchemas 会使用该缓存减少频繁读取 information_schema 的开销。
*/
type cachedSchema struct {
	data   []*dto.SchemaNode
	expiry time.Time
}

/*
dataSourceService 是 DataSourceService 的默认实现。
它维护数据源结构缓存和 GORM 连接池缓存，避免重复建立数据库连接。
*/
type dataSourceService struct {
	repo         repository.DataSourceRepository
	settingsRepo repository.SystemSettingRepository
	registry     *datasource.Registry
	schemaCache  map[uint]cachedSchema
	cacheMutex   sync.RWMutex
	connCache    map[uint]*gorm.DB // 按数据源 ID 缓存的 GORM 连接池。
	connMutex    sync.RWMutex      // 保护连接池缓存的读写锁。
}

/*
NewDataSourceService 创建数据源业务服务。
settingsRepo 用于读取全局查询限制等运行配置。
*/
func NewDataSourceService(repo repository.DataSourceRepository, settingsRepo repository.SystemSettingRepository, registry *datasource.Registry) DataSourceService {
	return &dataSourceService{
		repo:         repo,
		settingsRepo: settingsRepo,
		registry:     registry,
		schemaCache:  make(map[uint]cachedSchema),
		connCache:    make(map[uint]*gorm.DB),
	}
}

/*
normalizeDataSourceType 规范化数据源类型。
未填写时默认使用 mysql，其余值统一转为小写并去除首尾空格。
*/
func normalizeDataSourceType(dsType string) string {
	if strings.TrimSpace(dsType) == "" {
		return "mysql"
	}
	return strings.ToLower(strings.TrimSpace(dsType))
}

/*
normalizeDataSourceEnvironment 规范化数据源环境。
未填写时默认展示为生产环境。
*/
func normalizeDataSourceEnvironment(environment string) string {
	if strings.TrimSpace(environment) == "" {
		return "生产"
	}
	return strings.TrimSpace(environment)
}

/*
normalizeDataSourceStatus 规范化数据源状态。
disabled 和 inactive 都视为停用，其余值默认启用。
*/
func normalizeDataSourceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "disabled", "inactive":
		return "disabled"
	default:
		return "enabled"
	}
}

/*
normalizeDataSourceAccessMode 规范化数据源访问模式。
只读库不允许提交 DDL/DML 变更工单；未填写或非法值默认按读写处理。
*/
func normalizeDataSourceAccessMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "read_only", "readonly":
		return "read_only"
	default:
		return "read_write"
	}
}

/*
normalizeQueryTimeoutSeconds 返回查询窗口使用的超时时间。
未配置或配置非法时默认使用 30 秒。
*/
func normalizeQueryTimeoutSeconds(timeout int) int {
	if timeout <= 0 {
		return 30
	}
	return timeout
}

/*
normalizeExecutionTimeoutSeconds 返回工单执行类 SQL 使用的超时时间。
未配置或配置非法时默认使用 30 秒。
*/
func normalizeExecutionTimeoutSeconds(timeout int) int {
	if timeout <= 0 {
		return 30
	}
	return timeout
}

/*
normalizeConnectTimeout 返回建立数据库连接时的超时时间。
未配置或配置非法时默认使用 10 秒。
*/
func normalizeConnectTimeout(timeout int64) int64 {
	if timeout <= 0 {
		return 10
	}
	return timeout
}

/*
connInfoFor 将数据源模型转换为方言连接信息。
连接超时沿用服务层的规范化逻辑，方言层只负责根据秒数拼接 DSN。
*/
func connInfoFor(ds *model.DataSource) datasource.ConnInfo {
	return datasource.ConnInfo{
		Host:           ds.Host,
		Port:           ds.Port,
		User:           ds.Username,
		Password:       ds.Password,
		Database:       ds.Database,
		ConnectTimeout: normalizeConnectTimeout(ds.ConnectTimeout),
	}
}

/*
isSystemSchema 判断库名是否属于当前方言的系统库。
结构树构建和容量统计都会跳过系统库。
*/
func isSystemSchema(dialect datasource.Dialect, name string) bool {
	for _, sys := range dialect.SystemSchemas() {
		if name == sys {
			return true
		}
	}
	return false
}

/*
hasConfiguredPassword 判断数据源是否已配置密码。
列表响应只返回该布尔值，不暴露真实密码内容。
*/
func hasConfiguredPassword(password string) bool {
	return strings.TrimSpace(password) != ""
}

/*
normalizeSQLStatement 规范化待执行 SQL。
目前只去除首尾空白和末尾分号，避免解析和执行阶段出现不必要差异。
*/
func normalizeSQLStatement(sql string) string {
	return strings.TrimRight(strings.TrimSpace(sql), ";")
}

/*
classifyTimeoutError 将数据库操作错误归类为连接超时、查询超时或执行超时。
kind 取 "查询" 或 "执行"，用于生成对应前缀的中文提示，便于前端 toast 或工单结果准确展示。
*/
func classifyTimeoutError(err error, kind string, timeoutVal int) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())

	// 建立数据库连接阶段超时，driver 通常返回 dial / i/o timeout。
	if strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "connection timed out") || strings.Contains(msg, "dial tcp") {
		return fmt.Errorf("连接超时：%v", err)
	}

	// 查询或执行阶段超过限制（context 或 MySQL max_execution_time）。
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "execution time exceeded") {
		return fmt.Errorf("%s超时：SQL执行超过限制，已取消任务 (限制 %d 秒)", kind, timeoutVal)
	}

	return err
}

/*
ensureDataSourceEnabled 校验数据源是否可用。
停用的数据源禁止连接测试、元数据读取和 SQL 执行。
*/
func ensureDataSourceEnabled(ds *model.DataSource) error {
	if normalizeDataSourceStatus(ds.Status) != "enabled" {
		return errors.New("data source is disabled")
	}
	return nil
}

/*
Create 创建数据源记录。
创建时会校验数据源类型，规范化环境、状态、超时配置，并把请求中的脱敏规则转换为模型关联。
*/
func (s *dataSourceService) Create(req *dto.CreateDataSourceRequest) error {
	if _, err := s.registry.Resolve(req.Type); err != nil {
		return err
	}

	var rules []model.DataSourceMaskingRule
	for _, r := range req.MaskingRules {
		rules = append(rules, model.DataSourceMaskingRule{
			Pattern:     r.Pattern,
			RuleType:    r.RuleType,
			Description: r.Description,
		})
	}

	queryTimeoutSeconds := normalizeQueryTimeoutSeconds(req.QueryTimeoutSeconds)
	ds := &model.DataSource{
		Name:                req.Name,
		Type:                normalizeDataSourceType(req.Type),
		Host:                req.Host,
		Port:                req.Port,
		Database:            req.Database,
		Username:            req.Username,
		Password:            req.Password,
		Environment:         normalizeDataSourceEnvironment(req.Environment),
		ExecutionTimeout:    normalizeExecutionTimeoutSeconds(req.ExecutionTimeoutSeconds),
		QueryTimeoutSeconds: queryTimeoutSeconds,
		ConnectTimeout:      normalizeConnectTimeout(req.ConnectTimeout),
		Description:         req.Description,
		Status:              normalizeDataSourceStatus(req.Status),
		AccessMode:          normalizeDataSourceAccessMode(req.AccessMode),
		MaskingRules:        rules,
	}
	return s.repo.Create(ds)
}

/*
Update 更新数据源记录。
Password 为空时保留原密码，脱敏规则按请求体整体替换。
*/
func (s *dataSourceService) Update(req *dto.UpdateDataSourceRequest) error {
	if _, err := s.registry.Resolve(req.Type); err != nil {
		return err
	}

	ds, err := s.repo.FindByID(req.ID)
	if err != nil {
		return err
	}

	ds.Name = req.Name
	ds.Type = normalizeDataSourceType(req.Type)
	ds.Host = req.Host
	ds.Port = req.Port
	ds.Database = req.Database
	ds.Username = req.Username
	if req.Password != "" {
		ds.Password = req.Password
	}
	ds.Environment = normalizeDataSourceEnvironment(req.Environment)
	ds.ExecutionTimeout = normalizeExecutionTimeoutSeconds(req.ExecutionTimeoutSeconds)
	ds.QueryTimeoutSeconds = normalizeQueryTimeoutSeconds(req.QueryTimeoutSeconds)
	ds.ConnectTimeout = normalizeConnectTimeout(req.ConnectTimeout)
	ds.Description = req.Description
	ds.Status = normalizeDataSourceStatus(req.Status)
	ds.AccessMode = normalizeDataSourceAccessMode(req.AccessMode)

	var rules []model.DataSourceMaskingRule
	for _, r := range req.MaskingRules {
		rules = append(rules, model.DataSourceMaskingRule{
			Pattern:     r.Pattern,
			RuleType:    r.RuleType,
			Description: r.Description,
		})
	}
	ds.MaskingRules = rules

	if err := s.repo.Update(ds); err != nil {
		return err
	}
	s.invalidateCaches(req.ID)
	return nil
}

/*
invalidateCaches 移除指定数据源的结构缓存和连接池缓存。
数据源配置更新或删除后调用，确保后续请求根据最新配置重新连接并加载元数据。
*/
func (s *dataSourceService) invalidateCaches(id uint) {
	s.cacheMutex.Lock()
	delete(s.schemaCache, id)
	s.cacheMutex.Unlock()

	s.connMutex.Lock()
	db := s.connCache[id]
	delete(s.connCache, id)
	s.connMutex.Unlock()

	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

/*
Delete 删除数据源记录。
删除前会关闭并移除该数据源的连接池缓存，避免继续复用旧连接。
*/
func (s *dataSourceService) Delete(id uint) error {
	s.invalidateCaches(id)
	return s.repo.Delete(id)
}

/*
Copy 复制一个已有数据源。
新数据源会复制连接配置和脱敏规则，并在名称后追加 _Copy。
*/
func (s *dataSourceService) Copy(id uint) error {
	// 先读取原始数据源配置。
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// 通过创建新模型对象完成深拷贝，避免复用原记录的主键和关联状态。
	newDS := &model.DataSource{
		Name:                ds.Name + "_Copy",
		Type:                normalizeDataSourceType(ds.Type),
		Host:                ds.Host,
		Port:                ds.Port,
		Database:            ds.Database,
		Username:            ds.Username,
		Password:            ds.Password,
		Environment:         normalizeDataSourceEnvironment(ds.Environment),
		ExecutionTimeout:    normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout),
		QueryTimeoutSeconds: normalizeQueryTimeoutSeconds(ds.QueryTimeoutSeconds),
		ConnectTimeout:      normalizeConnectTimeout(ds.ConnectTimeout),
		Description:         ds.Description,
		Status:              normalizeDataSourceStatus(ds.Status),
		AccessMode:          normalizeDataSourceAccessMode(ds.AccessMode),
	}
	// 脱敏规则也复制为新的关联记录。
	var newRules []model.DataSourceMaskingRule
	for _, r := range ds.MaskingRules {
		newRules = append(newRules, model.DataSourceMaskingRule{
			Pattern:     r.Pattern,
			RuleType:    r.RuleType,
			Description: r.Description,
		})
	}
	newDS.MaskingRules = newRules

	return s.repo.Create(newDS)
}

/*
Get 按 ID 查询数据源模型。
调用方需要数据源详情或审计展示名称时使用该方法。
*/
func (s *dataSourceService) Get(id uint) (*model.DataSource, error) {
	return s.repo.FindByID(id)
}

/*
List 返回数据源列表响应。
响应会隐藏真实密码，只返回 PasswordConfigured，并附带脱敏规则配置。
*/
func (s *dataSourceService) List() ([]dto.DataSourceResponse, error) {
	dss, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	var responses []dto.DataSourceResponse
	for _, ds := range dss {
		resp := dto.DataSourceResponse{
			ID:                      ds.ID,
			Name:                    ds.Name,
			Type:                    normalizeDataSourceType(ds.Type),
			Host:                    ds.Host,
			Port:                    ds.Port,
			Database:                ds.Database,
			Username:                ds.Username,
			PasswordConfigured:      hasConfiguredPassword(ds.Password),
			Environment:             normalizeDataSourceEnvironment(ds.Environment),
			ExecutionTimeoutSeconds: normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout),
			QueryTimeoutSeconds:     normalizeQueryTimeoutSeconds(ds.QueryTimeoutSeconds),
			ConnectTimeout:          normalizeConnectTimeout(ds.ConnectTimeout),
			Description:             ds.Description,
			Status:                  normalizeDataSourceStatus(ds.Status),
			AccessMode:              normalizeDataSourceAccessMode(ds.AccessMode),
			CreatedAt:               ds.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:               ds.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		var rules []struct {
			Pattern     string `json:"pattern"`
			RuleType    string `json:"ruleType"`
			Description string `json:"description"`
		}
		for _, r := range ds.MaskingRules {
			rules = append(rules, struct {
				Pattern     string `json:"pattern"`
				RuleType    string `json:"ruleType"`
				Description string `json:"description"`
			}{
				Pattern:     r.Pattern,
				RuleType:    r.RuleType,
				Description: r.Description,
			})
		}
		resp.MaskingRules = rules
		responses = append(responses, resp)
	}
	return responses, nil
}

/*
FetchSchemas 获取数据源的库表结构树。
默认优先读取缓存，refresh=true 时重新读取 information_schema 并刷新缓存。
*/
func (s *dataSourceService) FetchSchemas(id uint, refresh bool) ([]*dto.SchemaNode, error) {
	// 非强制刷新时优先返回未过期的结构树缓存。
	if !refresh {
		s.cacheMutex.RLock()
		if cache, ok := s.schemaCache[id]; ok && time.Now().Before(cache.expiry) {
			data := cache.data
			s.cacheMutex.RUnlock()
			return data, nil
		}
		s.cacheMutex.RUnlock()
	}

	ds, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return nil, err
	}

	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return nil, err
	}

	// 根连接数据库：MySQL 连接 information_schema 读取全实例库表元数据；
	// PostgreSQL 没有跨库信息模式，改连数据源默认库，再逐库建连读取各库表结构。

	rootDatabase := "information_schema"
	if dialect.Type() == "postgres" {
		rootDatabase = ds.Database
	}
	rootDSN := dialect.BuildDSN(connInfoFor(ds), rootDatabase)

	db, err := dialect.Open(rootDSN)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	ctx := context.Background()

	// 预加载库 / 表容量信息（如果没有权限，会静默降级）
	tableSizeBytes := make(map[string]map[string]int64) // db -> table -> bytes
	dbTotalBytes := make(map[string]int64)              // db -> total bytes

	sizeRows, err := dialect.ListTableSizes(ctx, db)
	if err == nil {
		for _, row := range sizeRows {
			// 跳过系统库
			if isSystemSchema(dialect, row.Database) {
				continue
			}

			var dataLen, indexLen int64
			if row.DataLength.Valid {
				dataLen = row.DataLength.Int64
			}
			if row.IndexLength.Valid {
				indexLen = row.IndexLength.Int64
			}
			bytes := dataLen + indexLen
			if bytes < 0 {
				bytes = 0
			}

			if _, ok := tableSizeBytes[row.Database]; !ok {
				tableSizeBytes[row.Database] = make(map[string]int64)
			}
			tableSizeBytes[row.Database][row.Table] = bytes
			dbTotalBytes[row.Database] += bytes
		}
	}

	// 读取实例下所有数据库，系统库会在后续过滤掉。
	databases, err := dialect.ListDatabases(ctx, db)
	if err != nil {
		return nil, err
	}

	var nodes []*dto.SchemaNode

	for _, dbName := range databases {
		if isSystemSchema(dialect, dbName) {
			continue
		}

		dbNode := &dto.SchemaNode{
			Key:      fmt.Sprintf("db:%d:%s", len(dbName), dbName),
			Title:    dbName,
			Type:     "database",
			Database: dbName,
			Children: []*dto.SchemaNode{},
		}
		// 数据库总容量
		if totalBytes, ok := dbTotalBytes[dbName]; ok && totalBytes > 0 {
			dbNode.Size = formatBytes(totalBytes)
		}

		// MySQL 通过 information_schema 根连接按库过滤；PostgreSQL 需逐库建连读取 public schema 下的表。
		tables, err := s.listSchemaTables(ctx, dialect, connInfoFor(ds), db, dbName)
		if err != nil {
			continue
		}

		for _, tbl := range tables {
			nodeType := "table"
			if tbl.Type == "VIEW" {
				nodeType = "view"
			}

			tableNode := &dto.SchemaNode{
				Key:      fmt.Sprintf("tbl:%d:%s:%d:%s", len(dbName), dbName, len(tbl.Name), tbl.Name),
				Title:    tbl.Name,
				Type:     nodeType,
				Database: dbName,
				Table:    tbl.Name,
				IsLeaf:   true,
			}
			// 表级容量
			if perDB, ok := tableSizeBytes[dbName]; ok {
				if sizeBytes, ok2 := perDB[tbl.Name]; ok2 && sizeBytes > 0 {
					tableNode.Size = formatBytes(sizeBytes)
				}
			}

			dbNode.Children = append(dbNode.Children, tableNode)
		}

		nodes = append(nodes, dbNode)
	}

	// 写入结构树缓存，后续未强制刷新时可直接复用。
	s.cacheMutex.Lock()
	s.schemaCache[id] = cachedSchema{
		data:   nodes,
		expiry: time.Now().Add(schemaCacheTTL),
	}
	s.cacheMutex.Unlock()

	return nodes, nil
}

/*
listSchemaTables 读取指定库下的表与视图。
MySQL 复用 information_schema 根连接并按库过滤；PostgreSQL 需要单独连接目标库读取 public schema。
*/
func (s *dataSourceService) listSchemaTables(ctx context.Context, dialect datasource.Dialect, info datasource.ConnInfo, rootDB *gorm.DB, dbName string) ([]datasource.TableMeta, error) {
	if dialect.Type() == "postgres" {
		db, err := dialect.Open(dialect.BuildDSN(info, dbName))
		if err != nil {
			return nil, err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		defer sqlDB.Close()
		return dialect.ListTables(ctx, db, dbName)
	}
	return dialect.ListTables(ctx, rootDB, dbName)
}

/*
FetchColumns 获取指定库表的字段列表。
结果会标记主键字段为 key，其余字段为普通 column，供前端结构树展示。
*/
func (s *dataSourceService) FetchColumns(id uint, dbName string, tableName string) ([]*dto.SchemaNode, error) {
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return nil, err
	}

	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return nil, err
	}

	dsn := dialect.BuildDSN(connInfoFor(ds), dbName)

	db, err := dialect.Open(dsn)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	ctx := context.Background()

	// 从 information_schema.COLUMNS 按字段顺序读取列名、类型和索引标记。
	columns, err := dialect.ListColumns(ctx, db, dbName, tableName)
	if err != nil {
		return nil, err
	}

	var nodes []*dto.SchemaNode
	for _, col := range columns {
		iconType := "column"
		if col.Key == "PRI" {
			iconType = "key"
		}

		title := fmt.Sprintf("%s %s", col.Name, col.Type)

		nodes = append(nodes, &dto.SchemaNode{
			Key:      fmt.Sprintf("col:%d:%s:%d:%s:%d:%s", len(dbName), dbName, len(tableName), tableName, len(col.Name), col.Name),
			Title:    title,
			Type:     iconType,
			Database: dbName,
			Table:    tableName,
			IsLeaf:   true,
		})
	}

	return nodes, nil
}

/*
GetConnection 返回数据源的缓存连接池。
缓存未命中时会创建新的 GORM 连接，并设置连接池大小，减少重复建连和连接数失控风险。
*/
func (s *dataSourceService) GetConnection(ds *model.DataSource) (*gorm.DB, error) {
	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return nil, err
	}

	s.connMutex.RLock()
	if db, ok := s.connCache[ds.ID]; ok {
		s.connMutex.RUnlock()
		return db, nil
	}
	s.connMutex.RUnlock()

	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// 获取写锁后再次检查，避免并发请求重复创建同一数据源连接池。
	if db, ok := s.connCache[ds.ID]; ok {
		return db, nil
	}

	// 连接参数只设置建连超时；查询和执行超时通过每次请求的 context 控制。
	dsn := dialect.BuildDSN(connInfoFor(ds), ds.Database)

	db, err := dialect.Open(dsn)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 限制每个数据源的连接池规模，避免多数据源场景下打满 MySQL 连接数。
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	s.connCache[ds.ID] = db
	return db, nil
}

/*
ExecuteSQL 执行查询窗口或工单传入的 SQL。
它会根据 SQL 类型选择查询超时或执行超时，SELECT 会应用全局 LIMIT 和脱敏规则，DML/DDL 返回影响行数。
*/
func (s *dataSourceService) ExecuteSQL(id uint, sql string, dbName string) (*dto.ExecuteSQLResponse, error) {
	sql = normalizeSQLStatement(sql)
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return nil, err
	}
	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return nil, err
	}

	// 默认使用缓存连接池；当请求指定非默认库时，临时打开独立连接以保持库上下文隔离。
	var db *gorm.DB
	if dbName != "" && dbName != ds.Database {
		tmpDSN := dialect.BuildDSN(connInfoFor(ds), dbName)
		db, err = dialect.Open(tmpDSN)
		if err != nil {
			return nil, err
		}
		// 跨库临时连接用完即关，避免每个请求泄漏一个 MySQL 连接。
		tmpSQLDB, sqlErr := db.DB()
		if sqlErr != nil {
			return nil, sqlErr
		}
		defer tmpSQLDB.Close()
	} else {
		db, err = s.GetConnection(ds)
		if err != nil {
			return nil, err
		}
	}

	// PostgreSQL 走直连执行，跳过下方 MySQL 专属的 Vitess 解析与 LIMIT 改写逻辑。
	if dialect.Type() == "postgres" {
		return s.executePostgresDirect(ds, dialect, db, sql)
	}

	// 读取全局查询行数限制，配置缺失或非法时使用 1000。
	globalLimitStr, err := s.settingsRepo.Get("global_sql_limit")
	if err != nil {
		globalLimitStr = "1000"
	}
	globalLimit := 1000
	if val, err := strconv.Atoi(globalLimitStr); err == nil && val > 0 {
		globalLimit = val
	}

	// 多语句只支持变更类 SQL，避免一个请求中混入查询结果集导致响应结构不可控。
	parser := sqlparser.NewTestParser()
	pieces, splitErr := parser.SplitStatementToPieces(sql)
	if splitErr == nil && len(pieces) > 1 {
		for _, p := range pieces {
			if strings.TrimSpace(p) == "" {
				continue
			}
			s, err := parser.Parse(p)
			if err != nil {
				return nil, fmt.Errorf("Parse error in multi-statement: %v", err)
			}
			if _, ok := s.(*sqlparser.Select); ok {
				return nil, errors.New("Multi-statement QUERY is not supported")
			}
		}

		// 多语句执行使用单独 connection，保证同一 context 和会话超时配置生效。
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		// 多语句变更类 SQL 使用执行超时时间。
		timeoutVal := normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutVal)*time.Second)
		defer cancel()

		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			return nil, classifyTimeoutError(err, "执行", timeoutVal)
		}
		defer conn.Close()

		// 会话级执行耗时限制与 context 超时共同限制执行耗时。
		if timeoutVal > 0 {
			_, _ = conn.ExecContext(ctx, dialect.SessionTimeoutStatement(time.Duration(timeoutVal)*time.Second))
		}

		var totalAffected int64
		start := time.Now()
		statementResults := make([]dto.StatementExecutionResult, 0, len(pieces))
		statementIndex := 0
		for pieceIndex, cmd := range pieces {
			trimmedCmd := strings.TrimSpace(cmd)
			if trimmedCmd == "" {
				continue
			}
			statementIndex++
			statementStart := time.Now()
			res, err := conn.ExecContext(ctx, cmd)
			if err != nil {
				classifiedErr := classifyTimeoutError(err, "执行", timeoutVal)
				statementResults = append(statementResults, dto.StatementExecutionResult{
					Index:               statementIndex,
					SQL:                 trimmedCmd,
					Status:              "failed",
					ExecutionDurationMS: time.Since(statementStart).Milliseconds(),
					Message:             classifiedErr.Error(),
				})
				for _, pendingCmd := range pieces[pieceIndex+1:] {
					trimmedPendingCmd := strings.TrimSpace(pendingCmd)
					if trimmedPendingCmd == "" {
						continue
					}
					statementIndex++
					statementResults = append(statementResults, dto.StatementExecutionResult{
						Index:   statementIndex,
						SQL:     trimmedPendingCmd,
						Status:  "skipped",
						Message: "前序语句执行失败，未执行",
					})
				}
				return &dto.ExecuteSQLResponse{
					AffectedRows:     totalAffected,
					ExecutionTime:    time.Since(start).Milliseconds(),
					StatementResults: statementResults,
				}, fmt.Errorf("Execution failed at statement: %s. Error: %v", cmd, classifiedErr)
			}
			aff, _ := res.RowsAffected()
			totalAffected += aff
			statementResults = append(statementResults, dto.StatementExecutionResult{
				Index:               statementIndex,
				SQL:                 trimmedCmd,
				Status:              "success",
				AffectedRows:        aff,
				ExecutionDurationMS: time.Since(statementStart).Milliseconds(),
				Message:             "执行成功",
			})
		}
		duration := time.Since(start).Milliseconds()

		return &dto.ExecuteSQLResponse{
			Columns:          []string{"AffectedRows"},
			Rows:             []map[string]interface{}{{"AffectedRows": totalAffected}},
			RowValues:        [][]interface{}{{totalAffected}},
			AffectedRows:     totalAffected,
			ExecutionTime:    duration,
			StatementResults: statementResults,
		}, nil
	}

	// 解析 SQL 用于判断执行类型，并尽量通过 AST 分析结果列是否需要脱敏。
	maskingMap := make(map[int]model.DataSourceMaskingRule)
	isQuery := true

	stmt, err := parseQueryForMasking(parser, sql)
	if err == nil {
		switch selectStmt := stmt.(type) {
		case *sqlparser.Select:
			// SELECT 未指定 LIMIT 时自动补全，超过全局限制时改写为全局限制。
			var newLimit *sqlparser.Limit
			shouldRewrite := false

			if selectStmt.Limit == nil {
				newLimit = &sqlparser.Limit{Rowcount: sqlparser.NewIntLiteral(fmt.Sprintf("%d", globalLimit))}
				shouldRewrite = true
			} else {
				if limitVal, ok := selectStmt.Limit.Rowcount.(*sqlparser.Literal); ok && limitVal.Type == sqlparser.IntVal {
					userLimit, _ := strconv.Atoi(limitVal.Val)
					if userLimit > globalLimit {
						newLimit = &sqlparser.Limit{
							Rowcount: sqlparser.NewIntLiteral(fmt.Sprintf("%d", globalLimit)),
							Offset:   selectStmt.Limit.Offset,
						}
						shouldRewrite = true
					}
				}
			}

			if shouldRewrite {
				selectStmt.Limit = newLimit
				sql = sqlparser.String(selectStmt)
			}

			// 基于字段血缘分析结果列脱敏规则，兼容别名和子查询传递。
			maskingMap = s.analyzeLineage(selectStmt, ds.MaskingRules)
		case *sqlparser.Union:
			// UNION 结果列由最左侧分支决定，递归分析得到脱敏下标。
			maskingMap = s.analyzeUnion(selectStmt, ds.MaskingRules)
		case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete, sqlparser.DDLStatement:
			isQuery = false
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 查询类 SQL 使用 QueryTimeout，变更类 SQL 使用 ExecutionTimeout。
	var timeoutVal int
	if isQuery {
		timeoutVal = normalizeQueryTimeoutSeconds(ds.QueryTimeoutSeconds)
	} else {
		timeoutVal = normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout)
	}

	// context 超时用于限制本次请求，不影响连接池生命周期。
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutVal)*time.Second)
	defer cancel()

	start := time.Now()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		kind := "执行"
		if isQuery {
			kind = "查询"
		}
		return nil, classifyTimeoutError(err, kind, timeoutVal)
	}
	defer conn.Close()

	if timeoutVal > 0 {
		_, err = conn.ExecContext(ctx, dialect.SessionTimeoutStatement(time.Duration(timeoutVal)*time.Second))
		if err != nil {
			return nil, fmt.Errorf("failed to set max_execution_time: %v", err)
		}
	}

	if isQuery {
		// 查询类 SQL 返回列和行数据。
		rows, err := conn.QueryContext(ctx, sql)
		if err != nil {
			return nil, classifyTimeoutError(err, "查询", timeoutVal)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		resultRows := make([]map[string]interface{}, 0)
		resultRowValues := make([][]interface{}, 0)

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 脱敏引擎按字段血缘优先、列名匹配兜底的顺序处理结果值。
		engine := masking.NewEngine()

		for rows.Next() {
			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, err
			}

			rowMap := make(map[string]interface{})
			rowValues := make([]interface{}, len(columns))
			for i, col := range columns {
				val := values[i]
				var finalVal interface{}

				// 字节数组转字符串；大整数转字符串避免前端 JavaScript 精度丢失。
				switch v := val.(type) {
				case []byte:
					finalVal = string(v)
				case int64:
					finalVal = strconv.FormatInt(v, 10)
				case uint64:
					finalVal = strconv.FormatUint(v, 10)
				default:
					finalVal = val
				}

				// 优先使用 AST 血缘定位的脱敏规则。
				if rule, ok := maskingMap[i]; ok {
					strVal := fmt.Sprintf("%v", finalVal)
					finalVal = engine.Mask(strVal, rule.RuleType)
				} else {
					// 血缘分析未命中时，按返回列名匹配脱敏规则兜底。
					for _, rule := range ds.MaskingRules {
						if engine.ShouldMask(col, rule.Pattern) {
							strVal := fmt.Sprintf("%v", finalVal)
							finalVal = engine.Mask(strVal, rule.RuleType)
							break
						}
					}
				}

				rowMap[col] = finalVal
				rowValues[i] = finalVal
			}
			resultRows = append(resultRows, rowMap)
			resultRowValues = append(resultRowValues, rowValues)
		}

		executionTime := time.Since(start).Milliseconds()

		return &dto.ExecuteSQLResponse{
			Columns:       columns,
			Rows:          resultRows,
			RowValues:     resultRowValues,
			AffectedRows:  0,
			ExecutionTime: executionTime,
		}, nil

	} else {
		// 变更类 SQL 只返回影响行数和执行耗时，不返回结果集。
		res, err := conn.ExecContext(ctx, sql)
		if err != nil {
			return nil, classifyTimeoutError(err, "执行", timeoutVal)
		}

		affected, _ := res.RowsAffected()
		executionTime := time.Since(start).Milliseconds()

		return &dto.ExecuteSQLResponse{
			Columns:       []string{},
			Rows:          []map[string]interface{}{},
			RowValues:     [][]interface{}{},
			AffectedRows:  affected,
			ExecutionTime: executionTime,
		}, nil
	}
}

/*
executePostgresDirect 以直连方式执行 PostgreSQL 单条 SQL。
PG 暂不接入 MySQL 专属的 Vitess 解析、LIMIT 自动补全和 AST 血缘脱敏，
语句类型按前缀轻量判断，脱敏仅按返回列名兜底匹配。
*/
func (s *dataSourceService) executePostgresDirect(ds *model.DataSource, dialect datasource.Dialect, db *gorm.DB, sql string) (*dto.ExecuteSQLResponse, error) {
	isQuery := isPostgresQuery(sql)

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	var timeoutVal int
	if isQuery {
		timeoutVal = normalizeQueryTimeoutSeconds(ds.QueryTimeoutSeconds)
	} else {
		timeoutVal = normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutVal)*time.Second)
	defer cancel()

	start := time.Now()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		kind := "执行"
		if isQuery {
			kind = "查询"
		}
		return nil, classifyTimeoutError(err, kind, timeoutVal)
	}
	defer conn.Close()

	if timeoutVal > 0 {
		if _, err := conn.ExecContext(ctx, dialect.SessionTimeoutStatement(time.Duration(timeoutVal)*time.Second)); err != nil {
			return nil, fmt.Errorf("failed to set session timeout: %v", err)
		}
	}

	if !isQuery {
		res, err := conn.ExecContext(ctx, sql)
		if err != nil {
			return nil, classifyTimeoutError(err, "执行", timeoutVal)
		}
		affected, _ := res.RowsAffected()
		return &dto.ExecuteSQLResponse{
			Columns:       []string{},
			Rows:          []map[string]interface{}{},
			RowValues:     [][]interface{}{},
			AffectedRows:  affected,
			ExecutionTime: time.Since(start).Milliseconds(),
		}, nil
	}

	rows, err := conn.QueryContext(ctx, sql)
	if err != nil {
		return nil, classifyTimeoutError(err, "查询", timeoutVal)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	resultRows := make([]map[string]interface{}, 0)
	resultRowValues := make([][]interface{}, 0)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	engine := masking.NewEngine()

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		rowValues := make([]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			var finalVal interface{}

			switch v := val.(type) {
			case []byte:
				finalVal = string(v)
			case int64:
				finalVal = strconv.FormatInt(v, 10)
			case uint64:
				finalVal = strconv.FormatUint(v, 10)
			default:
				finalVal = val
			}

			// PG 直连执行没有 AST 血缘映射，脱敏规则按返回列名兜底匹配。
			for _, rule := range ds.MaskingRules {
				if engine.ShouldMask(col, rule.Pattern) {
					finalVal = engine.Mask(fmt.Sprintf("%v", finalVal), rule.RuleType)
					break
				}
			}

			rowMap[col] = finalVal
			rowValues[i] = finalVal
		}
		resultRows = append(resultRows, rowMap)
		resultRowValues = append(resultRowValues, rowValues)
	}

	return &dto.ExecuteSQLResponse{
		Columns:       columns,
		Rows:          resultRows,
		RowValues:     resultRowValues,
		AffectedRows:  0,
		ExecutionTime: time.Since(start).Milliseconds(),
	}, nil
}

/*
stripLeadingSQLComments 去除语句开头的空白与注释。
查询窗口会为 SQL 注入执行人注释，PG 前缀判断前需先跳过，否则首关键字被注释遮挡导致误判。
*/
func stripLeadingSQLComments(sql string) string {
	s := strings.TrimSpace(sql)
	for {
		switch {
		case strings.HasPrefix(s, "/*"):
			if idx := strings.Index(s, "*/"); idx >= 0 {
				s = strings.TrimSpace(s[idx+2:])
				continue
			}
		case strings.HasPrefix(s, "--"), strings.HasPrefix(s, "#"):
			if idx := strings.IndexByte(s, '\n'); idx >= 0 {
				s = strings.TrimSpace(s[idx+1:])
				continue
			}
		}
		return s
	}
}

/*
isPostgresQuery 按语句首关键字轻量判断 PostgreSQL 语句是否返回结果集。
PG 直连执行不做 AST 解析，仅用于区分查询类与变更类语句。
*/
func isPostgresQuery(sql string) bool {
	upper := strings.ToUpper(stripLeadingSQLComments(sql))
	for _, prefix := range []string{"SELECT", "WITH", "SHOW", "EXPLAIN", "VALUES", "TABLE"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

/*
postgresStatementType 按语句首关键字轻量判断 PostgreSQL 语句类型。
PG 直连 EXPLAIN 不做 AST 解析，仅用于区分 DML 与 DDL。
*/
func postgresStatementType(sql string) string {
	upper := strings.ToUpper(stripLeadingSQLComments(sql))
	for _, prefix := range []string{"SELECT", "WITH", "SHOW", "EXPLAIN", "VALUES", "TABLE"} {
		if strings.HasPrefix(upper, prefix) {
			return "SELECT"
		}
	}
	for _, prefix := range []string{"INSERT", "UPDATE", "DELETE"} {
		if strings.HasPrefix(upper, prefix) {
			return prefix
		}
	}
	for _, prefix := range []string{"CREATE", "ALTER", "DROP", "TRUNCATE", "COMMENT"} {
		if strings.HasPrefix(upper, prefix) {
			return "DDL"
		}
	}
	return "Other"
}

/*
TestConnection 测试已保存数据源连接。
测试前会确认数据源存在且未停用，成功时返回数据库版本。
*/
func (s *dataSourceService) TestConnection(id uint) (string, error) {
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return "", err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return "", err
	}

	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return "", err
	}
	return dialect.TestConnection(connInfoFor(ds))
}

/*
TestConnectionConfig 测试尚未保存的数据源连接配置。
该方法按请求类型分派方言测试临时连接，不会写入数据库。
*/
func (s *dataSourceService) TestConnectionConfig(req *dto.TestDataSourceConnectionRequest) (string, error) {
	dialect, err := s.registry.Resolve(req.Type)
	if err != nil {
		return "", err
	}
	return dialect.TestConnection(datasource.ConnInfo{
		Host:           req.Host,
		Port:           req.Port,
		User:           req.Username,
		Password:       req.Password,
		Database:       req.Database,
		ConnectTimeout: normalizeConnectTimeout(0),
	})
}

/*
CheckSyntax 使用数据库 prepare 能力检查 SQL 语法。
该方法不会实际执行 SQL，只验证当前数据源连接上下文下语句是否能被准备。
*/
func (s *dataSourceService) CheckSyntax(id uint, sql string) error {
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return err
	}

	// 使用独立短连接进行语法检查，避免影响连接池中的业务连接。
	dsn := dialect.BuildDSN(connInfoFor(ds), ds.Database)

	db, err := dialect.Open(dsn)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout))*time.Second)
	defer cancel()

	return dialect.CheckSyntax(ctx, sqlDB, sql)
}

/*
parseQueryForMasking 解析查询语句用于脱敏血缘分析。
Vitess v0.24.1 尚不支持 MySQL 8.0.31+ 的 INTERSECT / EXCEPT 语法，
这两种集合运算的结果列与 UNION 一样按位置对应，脱敏下标合并逻辑完全相同，
因此解析失败时临时替换为 UNION 重新解析，原始 SQL 仍由 MySQL 原样执行。
*/
func parseQueryForMasking(parser *sqlparser.Parser, sql string) (sqlparser.Statement, error) {
	stmt, err := parser.Parse(sql)
	if err != nil && setOpKeywordRe.MatchString(sql) {
		stmt, err = parser.Parse(setOpKeywordRe.ReplaceAllString(sql, "UNION"))
	}
	return stmt, err
}

/*
analyzeLineage 分析 SELECT 结果列与脱敏规则的关系。
它会识别直接敏感字段、子查询传递出的敏感别名，以及 SELECT * 展开子查询列的场景，
返回结果列下标到脱敏规则的映射。
*/
func (s *dataSourceService) analyzeLineage(stmt *sqlparser.Select, rules []model.DataSourceMaskingRule) map[int]model.DataSourceMaskingRule {
	taintedAliases := s.collectTaintedAliases(stmt, rules)

	maskingMap := make(map[int]model.DataSourceMaskingRule)
	if stmt.SelectExprs == nil {
		return maskingMap
	}

	for i, expr := range stmt.SelectExprs.Exprs {
		switch e := expr.(type) {
		case *sqlparser.AliasedExpr:
			if rule, ok := s.matchExpr(e.Expr, rules, taintedAliases); ok {
				maskingMap[i] = rule
			}
		case *sqlparser.StarExpr:
			// SELECT * / t.* 从子查询展开时，结果列顺序与子查询输出列一致，
			// 需要把子查询敏感列按顺序传播到外层下标。
			s.applyStarLineage(e, stmt, rules, maskingMap)
		}
	}

	return maskingMap
}

/*
collectTaintedAliases 收集 FROM 子查询输出列中携带的敏感别名，
供外层投影引用子查询别名时传播脱敏规则。
*/
func (s *dataSourceService) collectTaintedAliases(stmt *sqlparser.Select, rules []model.DataSourceMaskingRule) map[string]model.DataSourceMaskingRule {
	taintedAliases := make(map[string]model.DataSourceMaskingRule)
	for _, tableExpr := range stmt.From {
		s.collectTaintedAliasesFromTableExpr(tableExpr, rules, taintedAliases)
	}
	return taintedAliases
}

/*
collectTaintedAliasesFromTableExpr 递归遍历 FROM 中的表表达式，收集子查询输出列里的敏感别名。
JOIN（含 LATERAL 派生表）和括号包裹的表表达式都会向下递归，避免敏感字段藏在这些子查询中被漏判。
*/
func (s *dataSourceService) collectTaintedAliasesFromTableExpr(tableExpr sqlparser.TableExpr, rules []model.DataSourceMaskingRule, taintedAliases map[string]model.DataSourceMaskingRule) {
	switch t := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		derivedTable, ok := t.Expr.(*sqlparser.DerivedTable)
		if !ok {
			return
		}
		subTainted := s.analyzeSubquery(derivedTable.Select, rules)
		for alias, rule := range subTainted {
			taintedAliases[alias] = rule
		}
	case *sqlparser.JoinTableExpr:
		s.collectTaintedAliasesFromTableExpr(t.LeftExpr, rules, taintedAliases)
		s.collectTaintedAliasesFromTableExpr(t.RightExpr, rules, taintedAliases)
	case *sqlparser.ParenTableExpr:
		for _, inner := range t.Exprs {
			s.collectTaintedAliasesFromTableExpr(inner, rules, taintedAliases)
		}
	}
}

/*
analyzeSubquery 分析子查询输出列的敏感来源。
返回值以子查询输出列名或别名为 key，记录该输出列应使用的脱敏规则。
*/
func (s *dataSourceService) analyzeSubquery(stmt sqlparser.TableStatement, rules []model.DataSourceMaskingRule) map[string]model.DataSourceMaskingRule {
	sel, ok := stmt.(*sqlparser.Select)
	if !ok {
		// UNION 等复合子查询暂不参与输出别名传播。
		return map[string]model.DataSourceMaskingRule{}
	}

	taintedAliases := s.collectTaintedAliases(sel, rules)
	projectedTaints := make(map[string]model.DataSourceMaskingRule)

	if sel.SelectExprs == nil {
		return projectedTaints
	}
	for _, expr := range sel.SelectExprs.Exprs {
		aliased, ok := expr.(*sqlparser.AliasedExpr)
		if !ok {
			continue
		}
		// 优先使用显式别名，没有别名时仅对简单字段引用使用字段名。
		outputName := aliased.As.String()
		if outputName == "" {
			if colName, ok := aliased.Expr.(*sqlparser.ColName); ok {
				outputName = colName.Name.String()
			}
		}
		if outputName == "" {
			continue
		}

		if rule, found := s.matchExpr(aliased.Expr, rules, taintedAliases); found {
			projectedTaints[outputName] = rule
		}
	}

	return projectedTaints
}

/*
matchExpr 判断单个表达式是否引用敏感字段，返回对应的脱敏规则。
当敏感字段被 GROUP_CONCAT 等多值拼接聚合函数包裹时，强制使用 mask_all，
避免 mask_middle 只遮挡中间 1/3 导致头尾明文泄漏。
*/
func (s *dataSourceService) matchExpr(expr sqlparser.Expr, rules []model.DataSourceMaskingRule, taintedAliases map[string]model.DataSourceMaskingRule) (model.DataSourceMaskingRule, bool) {
	var matchedRule model.DataSourceMaskingRule
	found := false
	engine := masking.NewEngine()

	_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		if found {
			return false, nil
		}

		switch n := node.(type) {
		case sqlparser.AggrFunc:
			// 多值拼接型聚合(GROUP_CONCAT/JSON_ARRAYAGG/JSON_OBJECTAGG)会把多行敏感值拼成一条记录，
			// mask_middle 只遮中间 1/3，头尾仍会暴露明文，这里强制整列脱敏。
			if isMultiValueAggregate(n.AggrName()) {
				for _, arg := range n.GetArgs() {
					if rule, ok := s.matchExpr(arg, rules, taintedAliases); ok {
						rule.RuleType = "mask_all"
						matchedRule = rule
						found = true
						return false, nil
					}
				}
			}

		case *sqlparser.ColName:
			colStr := n.Name.String()
			for _, rule := range rules {
				if engine.ShouldMask(colStr, rule.Pattern) {
					matchedRule = rule
					found = true
					return false, nil
				}
			}
			if rule, ok := taintedAliases[colStr]; ok {
				matchedRule = rule
				found = true
				return false, nil
			}
		}

		return true, nil
	}, expr)

	return matchedRule, found
}

/*
isMultiValueAggregate 判断聚合函数是否属于“多值拼接成单串”的类型。
这类函数会把多行敏感值拼进同一条记录，mask_middle 只遮中间 1/3 会暴露头尾明文，
需要强制 mask_all。单值聚合(MAX/MIN/SUM 等)不在此列，仍沿用原规则。
*/
func isMultiValueAggregate(name string) bool {
	switch name {
	case "group_concat", "json_arrayagg", "json_objectagg":
		return true
	default:
		return false
	}
}

/*
applyStarLineage 处理 SELECT * / t.* 从子查询展开的场景。
仅处理“唯一投影为 SELECT *”的简单情况，保证结果列顺序与子查询输出列一致，
将子查询敏感列按顺序映射到外层结果列下标。
*/
func (s *dataSourceService) applyStarLineage(star *sqlparser.StarExpr, stmt *sqlparser.Select, rules []model.DataSourceMaskingRule, maskingMap map[int]model.DataSourceMaskingRule) {
	// 存在显式列混合时列顺序复杂，交给列名兜底匹配，避免下标错位。
	if len(stmt.SelectExprs.Exprs) != 1 {
		return
	}

	var derived *sqlparser.DerivedTable
	for _, tableExpr := range stmt.From {
		aliasedTable, ok := tableExpr.(*sqlparser.AliasedTableExpr)
		if !ok {
			continue
		}
		d, ok := aliasedTable.Expr.(*sqlparser.DerivedTable)
		if !ok {
			continue
		}
		// 无表名限定（SELECT *）接受任一子查询；有表名限定（SELECT t.*）要求别名一致。
		if star.TableName.Name.String() == "" || strings.EqualFold(aliasedTable.As.String(), star.TableName.Name.String()) {
			derived = d
			break
		}
	}
	if derived == nil {
		return
	}

	subMap := s.analyzeTableStatementOutputs(derived.Select, rules)
	for idx, rule := range subMap {
		maskingMap[idx] = rule
	}
}

/*
analyzeTableStatementOutputs 分析 SELECT/UNION 输出列，返回下标到脱敏规则的映射。
UNION 的结果列由最左分支决定。
*/
func (s *dataSourceService) analyzeTableStatementOutputs(stmt sqlparser.TableStatement, rules []model.DataSourceMaskingRule) map[int]model.DataSourceMaskingRule {
	switch t := stmt.(type) {
	case *sqlparser.Select:
		return s.analyzeSelectOutputs(t, rules)
	case *sqlparser.Union:
		// UNION 要求各分支列数一致且按位置对应，任一分支在某个下标命中敏感字段，
		// 该下标结果列就应脱敏。左分支决定结果列名，但敏感来源可能只在右分支，
		// 因此必须合并左右两个分支的下标脱敏规则。
		result := s.analyzeTableStatementOutputs(t.Left, rules)
		right := s.analyzeTableStatementOutputs(t.Right, rules)
		for idx, rule := range right {
			result[idx] = rule
		}
		return result
	default:
		return map[int]model.DataSourceMaskingRule{}
	}
}

/*
analyzeSelectOutputs 按投影顺序分析 SELECT 输出列，返回下标到脱敏规则的映射。
用于 SELECT * 展开子查询列时对齐结果列下标。
*/
func (s *dataSourceService) analyzeSelectOutputs(stmt *sqlparser.Select, rules []model.DataSourceMaskingRule) map[int]model.DataSourceMaskingRule {
	taintedAliases := s.collectTaintedAliases(stmt, rules)
	result := make(map[int]model.DataSourceMaskingRule)

	if stmt.SelectExprs == nil {
		return result
	}
	idx := 0
	for _, expr := range stmt.SelectExprs.Exprs {
		aliased, ok := expr.(*sqlparser.AliasedExpr)
		if !ok {
			// StarExpr 等无法确定具体列数，跳过，交给列名兜底。
			continue
		}
		if rule, found := s.matchExpr(aliased.Expr, rules, taintedAliases); found {
			result[idx] = rule
		}
		idx++
	}
	return result
}

/*
analyzeUnion 分析 UNION 结果列脱敏规则。
UNION 结果列由最左侧分支决定，递归分析得到脱敏下标。
*/
func (s *dataSourceService) analyzeUnion(stmt *sqlparser.Union, rules []model.DataSourceMaskingRule) map[int]model.DataSourceMaskingRule {
	return s.analyzeTableStatementOutputs(stmt, rules)
}

/*
QueryStream 执行 SQL 并逐行回调结果。
它复用查询超时和脱敏逻辑，适合导出等不希望一次性加载全部结果的场景。
*/
func (s *dataSourceService) QueryStream(id uint, sql string, dbName string, onRow func([]string, []interface{}) error) error {
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return err
	}
	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return err
	}

	// 与普通执行一致，默认复用连接池，跨库查询使用临时连接隔离上下文。
	var db *gorm.DB
	if dbName != "" && dbName != ds.Database {
		tmpDSN := dialect.BuildDSN(connInfoFor(ds), dbName)
		db, err = dialect.Open(tmpDSN)
		if err != nil {
			return err
		}
		// 跨库临时连接用完即关，避免每个请求泄漏一个 MySQL 连接。
		tmpSQLDB, sqlErr := db.DB()
		if sqlErr != nil {
			return sqlErr
		}
		defer tmpSQLDB.Close()
	} else {
		db, err = s.GetConnection(ds)
		if err != nil {
			return err
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 流式查询同样先分析脱敏规则，保证导出结果也会脱敏。
	// PostgreSQL 直连执行不接 Vitess AST 血缘，脱敏仅保留返回列名兜底匹配。
	maskingMap := make(map[int]model.DataSourceMaskingRule)
	if !strings.EqualFold(ds.Type, "postgres") {
		parser := sqlparser.NewTestParser()
		stmt, err := parseQueryForMasking(parser, sql)
		if err == nil {
			switch selectStmt := stmt.(type) {
			case *sqlparser.Select:
				maskingMap = s.analyzeLineage(selectStmt, ds.MaskingRules)
			case *sqlparser.Union:
				maskingMap = s.analyzeUnion(selectStmt, ds.MaskingRules)
			}
		}
	}

	// 流式查询使用查询超时时间，避免大结果集长时间占用连接。
	timeoutVal := normalizeQueryTimeoutSeconds(ds.QueryTimeoutSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutVal)*time.Second)
	defer cancel()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if timeoutVal > 0 {
		_, _ = conn.ExecContext(ctx, dialect.SessionTimeoutStatement(time.Duration(timeoutVal)*time.Second))
	}

	rows, err := conn.QueryContext(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 逐行扫描结果并交给调用方回调处理。
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	engine := masking.NewEngine()

	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		rowData := make([]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			var finalVal interface{}

			if b, ok := val.([]byte); ok {
				finalVal = string(b)
			} else {
				finalVal = val
			}

			// 导出流式结果时也按血缘优先、列名兜底的顺序脱敏。
			if rule, ok := maskingMap[i]; ok {
				strVal := fmt.Sprintf("%v", finalVal)
				finalVal = engine.Mask(strVal, rule.RuleType)
			} else {
				for _, rule := range ds.MaskingRules {
					if engine.ShouldMask(col, rule.Pattern) {
						strVal := fmt.Sprintf("%v", finalVal)
						finalVal = engine.Mask(strVal, rule.RuleType)
						break
					}
				}
			}
			rowData[i] = finalVal
		}

		if err := onRow(columns, rowData); err != nil {
			return err
		}
	}

	return nil
}

/*
ExplainSQL 对 SQL 执行 EXPLAIN 并返回预估扫描行数。
多语句会逐条分析并累加每条执行计划中的最大 rows，DDL 当前返回不支持执行计划。
*/
func (s *dataSourceService) ExplainSQL(id uint, sql string, dbName string) (*dto.ExplainResult, error) {
	ds, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if err := ensureDataSourceEnabled(ds); err != nil {
		return nil, err
	}

	dialect, err := s.registry.Resolve(ds.Type)
	if err != nil {
		return &dto.ExplainResult{
			Type:          "Unsupported",
			EstimatedRows: 0,
			IsSupported:   false,
			Message:       err.Error(),
		}, nil
	}

	// Explain 使用独立短连接，dbName 为空时回退到数据源默认库。
	targetDB := ds.Database
	if dbName != "" {
		targetDB = dbName
	}
	dsn := dialect.BuildDSN(connInfoFor(ds), targetDB)

	db, err := dialect.Open(dsn)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(normalizeExecutionTimeoutSeconds(ds.ExecutionTimeout))*time.Second)
	defer cancel()

	// PostgreSQL 直连 EXPLAIN，跳过 MySQL 专属的 Vitess 拆分与语句类型判断。
	if dialect.Type() == "postgres" {
		stmtType := postgresStatementType(sql)
		if stmtType == "DDL" {
			return &dto.ExplainResult{
				Type:          "DDL",
				EstimatedRows: 0,
				IsSupported:   false,
				Message:       "Schema Change (DDL)",
			}, nil
		}
		rows, err := dialect.Explain(ctx, sqlDB, sql)
		if err != nil {
			return nil, err
		}
		return &dto.ExplainResult{
			Type:          stmtType,
			EstimatedRows: rows,
			IsSupported:   true,
		}, nil
	}

	// 多语句先拆分，再逐条判断语句类型并执行 EXPLAIN。
	parser := sqlparser.NewTestParser()
	pieces, err := parser.SplitStatementToPieces(sql)
	if err != nil {
		return nil, fmt.Errorf("SQL split error: %v", err)
	}

	var totalEstimated int64 = 0
	var overallType string = ""

	for _, cmd := range pieces {
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		stmt, err := parser.Parse(cmd)
		if err != nil {
			return nil, fmt.Errorf("SQL parse error in statement '%s': %v", cmd, err)
		}

		var currentType string
		switch stmt.(type) {
		case *sqlparser.Select:
			currentType = "SELECT"
		case *sqlparser.Insert:
			currentType = "INSERT"
		case *sqlparser.Update:
			currentType = "UPDATE"
		case *sqlparser.Delete:
			currentType = "DELETE"
		case sqlparser.DDLStatement:
			return &dto.ExplainResult{
				Type:          "DDL",
				EstimatedRows: 0,
				IsSupported:   false,
				Message:       "Schema Change (DDL)",
			}, nil
		default:
			if overallType == "" {
				overallType = "Other"
			}
			continue
		}

		if overallType == "" {
			overallType = currentType
		} else if overallType != currentType {
			overallType = "Mixed"
		}

		// MySQL 对 SELECT/INSERT/UPDATE/DELETE 支持 EXPLAIN，用于估算扫描行数。
		maxRowsInPlan, err := dialect.Explain(ctx, sqlDB, cmd)
		if err != nil {
			return nil, err
		}
		// 每条语句取执行计划中的最大 rows，再累加为整体预估扫描量。
		totalEstimated += maxRowsInPlan
	}

	return &dto.ExplainResult{
		Type:          overallType,
		EstimatedRows: totalEstimated,
		IsSupported:   true,
	}, nil
}

/*
formatBytes 将字节数格式化为带单位的展示字符串。
库表结构树会用它展示数据库和表的容量。
*/
func formatBytes(size int64) string {
	if size <= 0 {
		return ""
	}

	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	var value float64
	var unit string

	switch {
	case size >= TB:
		value = float64(size) / float64(TB)
		unit = "TB"
	case size >= GB:
		value = float64(size) / float64(GB)
		unit = "GB"
	case size >= MB:
		value = float64(size) / float64(MB)
		unit = "MB"
	case size >= KB:
		value = float64(size) / float64(KB)
		unit = "KB"
	default:
		value = float64(size)
		unit = "B"
	}

	return fmt.Sprintf("%.2f %s", value, unit)
}
