package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"nextmeta-backend/internal/api/dto"
	"nextmeta-backend/internal/model"
	"nextmeta-backend/internal/repository"
	"nextmeta-backend/pkg/logger"

	"nextmeta-backend/pkg/masking"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"vitess.io/vitess/go/vt/sqlparser"
)

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
	schemaCache  map[uint]cachedSchema
	cacheMutex   sync.RWMutex
	connCache    map[uint]*gorm.DB // 按数据源 ID 缓存的 GORM 连接池。
	connMutex    sync.RWMutex      // 保护连接池缓存的读写锁。
}

/*
NewDataSourceService 创建数据源业务服务。
settingsRepo 用于读取全局查询限制等运行配置。
*/
func NewDataSourceService(repo repository.DataSourceRepository, settingsRepo repository.SystemSettingRepository) DataSourceService {
	return &dataSourceService{
		repo:         repo,
		settingsRepo: settingsRepo,
		schemaCache:  make(map[uint]cachedSchema),
		connCache:    make(map[uint]*gorm.DB),
	}
}

/*
ensureMySQLDataSourceType 校验数据源类型。
当前后端只支持 MySQL，空类型会在后续规范化为 mysql。
*/
func ensureMySQLDataSourceType(dsType string) error {
	if dsType == "" {
		return nil
	}
	if !strings.EqualFold(dsType, "mysql") {
		return errors.New("currently only MySQL is supported")
	}
	return nil
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
	if err := ensureMySQLDataSourceType(req.Type); err != nil {
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
	if err := ensureMySQLDataSourceType(req.Type); err != nil {
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

	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
		return nil, err
	}

	// 这里连接 information_schema 读取全实例库表元数据，不复用默认库连接池，避免切库影响业务连接上下文。

	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/information_schema?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		ds.Username, ds.Password, ds.Host, ds.Port, normalizeConnectTimeout(ds.ConnectTimeout))

	db, err := gorm.Open(mysql.Open(rootDSN), logger.GormConfig())
	if err != nil {
		return nil, err
	}

	// 预加载库 / 表容量信息（如果没有权限，会静默降级）
	type SizeRow struct {
		TableSchema string
		TableName   string
		DataLength  sql.NullInt64
		IndexLength sql.NullInt64
	}
	tableSizeBytes := make(map[string]map[string]int64) // db -> table -> bytes
	dbTotalBytes := make(map[string]int64)              // db -> total bytes

	var sizeRows []SizeRow
	if err := db.Raw(`
		SELECT TABLE_SCHEMA, TABLE_NAME, DATA_LENGTH, INDEX_LENGTH
		FROM TABLES
	`).Scan(&sizeRows).Error; err == nil {
		for _, row := range sizeRows {
			// 跳过系统库
			if row.TableSchema == "information_schema" ||
				row.TableSchema == "mysql" ||
				row.TableSchema == "performance_schema" ||
				row.TableSchema == "sys" {
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

			if _, ok := tableSizeBytes[row.TableSchema]; !ok {
				tableSizeBytes[row.TableSchema] = make(map[string]int64)
			}
			tableSizeBytes[row.TableSchema][row.TableName] = bytes
			dbTotalBytes[row.TableSchema] += bytes
		}
	}

	// 读取实例下所有数据库，系统库会在后续过滤掉。
	var databases []string
	if err := db.Raw("SHOW DATABASES").Scan(&databases).Error; err != nil {
		return nil, err
	}

	var nodes []*dto.SchemaNode

	for _, dbName := range databases {
		if dbName == "information_schema" || dbName == "mysql" || dbName == "performance_schema" || dbName == "sys" {
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

		// 从 information_schema.TABLES 读取当前库下的表和视图。
		type TableInfo struct {
			TableName string
			TableType string
		}
		var tables []TableInfo
		if err := db.Raw("SELECT TABLE_NAME, TABLE_TYPE FROM TABLES WHERE TABLE_SCHEMA = ?", dbName).Scan(&tables).Error; err != nil {
			continue
		}

		for _, tbl := range tables {
			nodeType := "table"
			if tbl.TableType == "VIEW" {
				nodeType = "view"
			}

			tableNode := &dto.SchemaNode{
				Key:      fmt.Sprintf("tbl:%d:%s:%d:%s", len(dbName), dbName, len(tbl.TableName), tbl.TableName),
				Title:    tbl.TableName,
				Type:     nodeType,
				Database: dbName,
				Table:    tbl.TableName,
				IsLeaf:   true,
			}
			// 表级容量
			if perDB, ok := tableSizeBytes[dbName]; ok {
				if sizeBytes, ok2 := perDB[tbl.TableName]; ok2 && sizeBytes > 0 {
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

	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		ds.Username, ds.Password, ds.Host, ds.Port, dbName, normalizeConnectTimeout(ds.ConnectTimeout))

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
	if err != nil {
		return nil, err
	}

	// 从 information_schema.COLUMNS 按字段顺序读取列名、类型和索引标记。
	type ColumnInfo struct {
		ColumnName string
		ColumnType string
		ColumnKey  string // MySQL 字段索引标记，例如 PRI、MUL、UNI。
	}
	var columns []ColumnInfo
	if err := db.Raw("SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_KEY FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION", dbName, tableName).Scan(&columns).Error; err != nil {
		return nil, err
	}

	var nodes []*dto.SchemaNode
	for _, col := range columns {
		iconType := "column"
		if col.ColumnKey == "PRI" {
			iconType = "key"
		}

		title := fmt.Sprintf("%s %s", col.ColumnName, col.ColumnType)

		nodes = append(nodes, &dto.SchemaNode{
			Key:      fmt.Sprintf("col:%d:%s:%d:%s:%d:%s", len(dbName), dbName, len(tableName), tableName, len(col.ColumnName), col.ColumnName),
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
testDataSourceConnection 使用临时连接测试数据源可用性。
成功时返回 SELECT VERSION() 的结果，调用结束后立即关闭底层连接。
*/
func testDataSourceConnection(dsType string, host string, port int, database string, username string, password string) (string, error) {
	if err := ensureMySQLDataSourceType(dsType); err != nil {
		return "", err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s&readTimeout=5s",
		username, password, host, port, database)

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

/*
GetConnection 返回数据源的缓存连接池。
缓存未命中时会创建新的 GORM 连接，并设置连接池大小，减少重复建连和连接数失控风险。
*/
func (s *dataSourceService) GetConnection(ds *model.DataSource) (*gorm.DB, error) {
	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
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

	timeoutParams := fmt.Sprintf("timeout=%ds", normalizeConnectTimeout(ds.ConnectTimeout))
	// 连接参数只设置建连超时；查询和执行超时通过每次请求的 context 控制。

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&%s",
		ds.Username, ds.Password, ds.Host, ds.Port, ds.Database, timeoutParams)

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
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
	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
		return nil, err
	}

	// 默认使用缓存连接池；当请求指定非默认库时，临时打开独立连接以保持库上下文隔离。
	var db *gorm.DB
	if dbName != "" && dbName != ds.Database {
		tmpDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
			ds.Username, ds.Password, ds.Host, ds.Port, dbName, normalizeConnectTimeout(ds.ConnectTimeout))
		db, err = gorm.Open(mysql.Open(tmpDSN), logger.GormConfig())
		if err != nil {
			return nil, err
		}
	} else {
		db, err = s.GetConnection(ds)
		if err != nil {
			return nil, err
		}
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

		// MySQL 会话级 max_execution_time 与 context 超时共同限制执行耗时。
		if timeoutVal > 0 {
			timeoutMs := timeoutVal * 1000
			_, _ = conn.ExecContext(ctx, fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutMs))
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

	stmt, err := parser.Parse(sql)
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
		timeoutMs := timeoutVal * 1000
		_, err = conn.ExecContext(ctx, fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutMs))
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

	return testDataSourceConnection(ds.Type, ds.Host, ds.Port, ds.Database, ds.Username, ds.Password)
}

/*
TestConnectionConfig 测试尚未保存的数据源连接配置。
该方法固定按 MySQL 测试临时连接，不会写入数据库。
*/
func (s *dataSourceService) TestConnectionConfig(req *dto.TestDataSourceConnectionRequest) (string, error) {
	return testDataSourceConnection("mysql", req.Host, req.Port, req.Database, req.Username, req.Password)
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

	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
		return err
	}

	// 使用独立短连接进行语法检查，避免影响连接池中的业务连接。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		ds.Username, ds.Password, ds.Host, ds.Port, ds.Database, normalizeConnectTimeout(ds.ConnectTimeout))

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
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

	stmt, err := sqlDB.PrepareContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("syntax error: %v", err)
	}
	defer stmt.Close()

	return nil
}

/*
analyzeLineage 分析 SELECT 结果列与脱敏规则的关系。
它会识别直接敏感字段和子查询传递出的敏感别名，返回结果列下标到脱敏规则的映射。
*/
func (s *dataSourceService) analyzeLineage(stmt *sqlparser.Select, rules []model.DataSourceMaskingRule) map[int]model.DataSourceMaskingRule {
	// 先分析 FROM 子查询，找出从子查询传递出来的敏感别名。
	taintedAliases := make(map[string]model.DataSourceMaskingRule)

	for _, tableExpr := range stmt.From {
		if aliasedTable, ok := tableExpr.(*sqlparser.AliasedTableExpr); ok {
			if derivedTable, ok := aliasedTable.Expr.(*sqlparser.DerivedTable); ok {
				if subSelect, ok := derivedTable.Select.(*sqlparser.Select); ok {
					subTainted := s.analyzeSubquery(subSelect, rules)
					for alias, rule := range subTainted {
						taintedAliases[alias] = rule
					}
				}
			}
		}
	}

	// 再分析当前 SELECT 投影表达式，判断每个结果列是否需要脱敏。
	maskingMap := make(map[int]model.DataSourceMaskingRule)

	if stmt.SelectExprs == nil {
		return maskingMap
	}
	for i, expr := range stmt.SelectExprs.Exprs {
		if aliased, ok := expr.(*sqlparser.AliasedExpr); ok {
			foundRule := false
			var matchedRule model.DataSourceMaskingRule

			_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
				if foundRule {
					return false, nil
				}
				if colName, ok := node.(*sqlparser.ColName); ok {
					colStr := colName.Name.String()

					// 直接引用敏感字段时命中对应脱敏规则。
					for _, rule := range rules {
						if strings.EqualFold(colStr, rule.Pattern) {
							foundRule = true
							matchedRule = rule
							return false, nil
						}
					}

					// 引用子查询传递出的敏感别名时也需要脱敏。
					if rule, ok := taintedAliases[colStr]; ok {
						foundRule = true
						matchedRule = rule
						return false, nil
					}
				}
				return true, nil
			}, aliased.Expr)

			if foundRule {
				maskingMap[i] = matchedRule
			}
		}
	}

	return maskingMap
}

/*
analyzeSubquery 分析子查询输出列的敏感来源。
返回值以子查询输出列名或别名为 key，记录该输出列应使用的脱敏规则。
*/
func (s *dataSourceService) analyzeSubquery(stmt *sqlparser.Select, rules []model.DataSourceMaskingRule) map[string]model.DataSourceMaskingRule {
	// 递归分析更深层子查询传递出的敏感别名。
	taintedAliases := make(map[string]model.DataSourceMaskingRule)
	for _, tableExpr := range stmt.From {
		if aliasedTable, ok := tableExpr.(*sqlparser.AliasedTableExpr); ok {
			if derivedTable, ok := aliasedTable.Expr.(*sqlparser.DerivedTable); ok {
				if subSelect, ok := derivedTable.Select.(*sqlparser.Select); ok {
					subTainted := s.analyzeSubquery(subSelect, rules)
					for alias, rule := range subTainted {
						taintedAliases[alias] = rule
					}
				}
			}
		}
	}

	// 分析当前子查询的投影列，确定哪些输出列需要继续向外层传播脱敏规则。
	projectedTaints := make(map[string]model.DataSourceMaskingRule)

	if stmt.SelectExprs == nil {
		return projectedTaints
	}
	for _, expr := range stmt.SelectExprs.Exprs {
		if aliased, ok := expr.(*sqlparser.AliasedExpr); ok {
			// 优先使用显式别名，没有别名时仅对简单字段引用使用字段名。
			outputName := aliased.As.String()
			if outputName == "" {
				if colName, ok := aliased.Expr.(*sqlparser.ColName); ok {
					outputName = colName.Name.String()
				}
			}

			if outputName != "" {
				foundRule := false
				var matchedRule model.DataSourceMaskingRule

				_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
					if foundRule {
						return false, nil
					}
					if colName, ok := node.(*sqlparser.ColName); ok {
						colStr := colName.Name.String()
						// 直接命中敏感字段规则时，当前输出列继承该脱敏规则。
						for _, rule := range rules {
							if strings.EqualFold(colStr, rule.Pattern) {
								foundRule = true
								matchedRule = rule
								return false, nil
							}
						}
						// 引用内层敏感别名时，当前输出列继续继承该脱敏规则。
						if rule, ok := taintedAliases[colStr]; ok {
							foundRule = true
							matchedRule = rule
							return false, nil
						}
					}
					return true, nil
				}, aliased.Expr)

				if foundRule {
					projectedTaints[outputName] = matchedRule
				}
			}
		}
	}

	return projectedTaints
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
	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
		return err
	}

	// 与普通执行一致，默认复用连接池，跨库查询使用临时连接隔离上下文。
	var db *gorm.DB
	if dbName != "" && dbName != ds.Database {
		tmpDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
			ds.Username, ds.Password, ds.Host, ds.Port, dbName, normalizeConnectTimeout(ds.ConnectTimeout))
		db, err = gorm.Open(mysql.Open(tmpDSN), logger.GormConfig())
		if err != nil {
			return err
		}
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
	maskingMap := make(map[int]model.DataSourceMaskingRule)
	parser := sqlparser.NewTestParser()
	stmt, err := parser.Parse(sql)
	if err == nil {
		if selectStmt, ok := stmt.(*sqlparser.Select); ok {
			maskingMap = s.analyzeLineage(selectStmt, ds.MaskingRules)
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

	if strings.EqualFold(ds.Type, "mysql") && timeoutVal > 0 {
		timeoutMs := timeoutVal * 1000
		_, _ = conn.ExecContext(ctx, fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutMs))
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

	if err := ensureMySQLDataSourceType(ds.Type); err != nil {
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		ds.Username, ds.Password, ds.Host, ds.Port, targetDB, normalizeConnectTimeout(ds.ConnectTimeout))

	db, err := gorm.Open(mysql.Open(dsn), logger.GormConfig())
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
		explainSQL := "EXPLAIN " + cmd
		rows, err := sqlDB.QueryContext(ctx, explainSQL)
		if err != nil {
			return nil, fmt.Errorf("EXPLAIN failed for '%s': %v", cmd, err)
		}

		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}

		rowsIdx := -1
		for i, col := range columns {
			if strings.ToLower(col) == "rows" {
				rowsIdx = i
				break
			}
		}

		if rowsIdx != -1 {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			var maxRowsInPlan int64 = 0
			for rows.Next() {
				if err := rows.Scan(valuePtrs...); err != nil {
					rows.Close()
					return nil, err
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
			// 每条语句取执行计划中的最大 rows，再累加为整体预估扫描量。
			totalEstimated += maxRowsInPlan
		}
		rows.Close()
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
