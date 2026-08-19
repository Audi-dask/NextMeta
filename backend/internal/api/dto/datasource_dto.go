package dto

/*
CreateDataSourceRequest 是前端创建数据源时提交的请求体。
除基础连接信息外，还包含执行超时、查询超时、连接超时和结果脱敏规则配置。
*/
type CreateDataSourceRequest struct {
	Name                    string `json:"name" binding:"required"`
	Type                    string `json:"type"`
	Host                    string `json:"host" binding:"required"`
	Port                    int    `json:"port" binding:"required"`
	Database                string `json:"database"`
	Username                string `json:"username"`
	Password                string `json:"password"`
	Environment             string `json:"environment"`
	ExecutionTimeoutSeconds int    `json:"executionTimeoutSeconds" binding:"omitempty,min=1,max=3600"`
	QueryTimeoutSeconds     int    `json:"queryTimeoutSeconds" binding:"omitempty,min=1,max=3600"`
	ConnectTimeout          int64  `json:"connectTimeout" binding:"omitempty,min=1,max=3600"`
	Description             string `json:"description"`
	Status                  string `json:"status" binding:"omitempty,oneof=enabled disabled"`
	AccessMode              string `json:"accessMode" binding:"omitempty,oneof=read_only read_write"`
	MaskingRules            []struct {
		Pattern     string `json:"pattern" binding:"required"`
		RuleType    string `json:"ruleType" binding:"required"`
		Description string `json:"description"`
	} `json:"maskingRules"`
}

/*
UpdateDataSourceRequest 是前端更新数据源时提交的请求体。
Password 为空时由 service 保留原密码，其余字段用于覆盖数据源基础配置和脱敏规则。
*/
type UpdateDataSourceRequest struct {
	ID                      uint   `json:"id" binding:"required"`
	Name                    string `json:"name" binding:"required"`
	Type                    string `json:"type"`
	Host                    string `json:"host" binding:"required"`
	Port                    int    `json:"port" binding:"required"`
	Database                string `json:"database"`
	Username                string `json:"username"`
	Password                string `json:"password"`
	Environment             string `json:"environment"`
	ExecutionTimeoutSeconds int    `json:"executionTimeoutSeconds" binding:"omitempty,min=1,max=3600"`
	QueryTimeoutSeconds     int    `json:"queryTimeoutSeconds" binding:"omitempty,min=1,max=3600"`
	ConnectTimeout          int64  `json:"connectTimeout" binding:"omitempty,min=1,max=3600"`
	Description             string `json:"description"`
	Status                  string `json:"status" binding:"omitempty,oneof=enabled disabled"`
	AccessMode              string `json:"accessMode" binding:"omitempty,oneof=read_only read_write"`
	MaskingRules            []struct {
		Pattern     string `json:"pattern" binding:"required"`
		RuleType    string `json:"ruleType" binding:"required"`
		Description string `json:"description"`
	} `json:"maskingRules"`
}

/*
TestDataSourceConnectionRequest 是前端在保存前测试连接时提交的请求体。
该结构只用于临时连接测试，不会直接写入数据库。
*/
type TestDataSourceConnectionRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

/*
DataSourceResponse 是数据源列表接口返回给前端的结构。
它由 model.DataSource 转换而来，只返回 PasswordConfigured，不返回真实数据库密码。
*/
type DataSourceResponse struct {
	ID                      uint   `json:"id"`
	Name                    string `json:"name"`
	Type                    string `json:"type"`
	Host                    string `json:"host"`
	Port                    int    `json:"port"`
	Database                string `json:"database"`
	Username                string `json:"username"`
	PasswordConfigured      bool   `json:"passwordConfigured"`
	Environment             string `json:"environment"`
	ExecutionTimeoutSeconds int    `json:"executionTimeoutSeconds"`
	QueryTimeoutSeconds     int    `json:"queryTimeoutSeconds"`
	ConnectTimeout          int64  `json:"connectTimeout"`
	Description             string `json:"description"`
	Status                  string `json:"status"`
	AccessMode              string `json:"accessMode"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
	MaskingRules            []struct {
		Pattern     string `json:"pattern"`
		RuleType    string `json:"ruleType"`
		Description string `json:"description"`
	} `json:"maskingRules"`
}

/*
SchemaNode 是数据库结构树节点。
FetchSchemas 和 FetchColumns 使用该结构向前端返回库、表、视图和列的层级信息。
*/
type SchemaNode struct {
	Key      string        `json:"key"`
	Title    string        `json:"title"`
	Type     string        `json:"type"`
	Database string        `json:"database,omitempty"`
	Table    string        `json:"table,omitempty"`
	Children []*SchemaNode `json:"children,omitempty"`
	IsLeaf   bool          `json:"isLeaf,omitempty"`
	Size     string        `json:"size,omitempty"`
}

/*
ExecuteSQLRequest 是 SQL 查询窗口提交执行 SELECT 时的请求体。
SQL 为用户输入的查询语句，Database 指定本次查询使用的库名。
*/
type ExecuteSQLRequest struct {
	SQL            string `json:"sql" binding:"required"`
	Database       string `json:"database"`
	Description    string `json:"description" binding:"required"`
	QuerySessionID string `json:"query_session_id" binding:"required,uuid"`
}

type StatementExecutionResult struct {
	Index               int    `json:"index"`
	SQL                 string `json:"sql"`
	Status              string `json:"status"`
	AffectedRows        int64  `json:"affectedRows"`
	ExecutionDurationMS int64  `json:"executionDurationMS"`
	Message             string `json:"message,omitempty"`
}

/*
ExecuteSQLResponse 是 SQL 查询窗口返回给前端的执行结果。
Rows 保留按列名访问的兼容结构，RowValues 按列位置保存完整结果，可正确表达重复列名。
StatementResults 保存多语句变更中每条语句的执行事实。
*/
type ExecuteSQLResponse struct {
	Columns          []string                   `json:"columns"`
	Rows             []map[string]interface{}   `json:"rows"`
	RowValues        [][]interface{}            `json:"rowValues"`
	AffectedRows     int64                      `json:"affectedRows"`
	ExecutionTime    int64                      `json:"executionTime"`
	StatementResults []StatementExecutionResult `json:"statementResults,omitempty"`
	Error            string                     `json:"error,omitempty"`
}

/*
ExplainResult 是 SQL 语法或执行计划检查结果摘要。
工单语法检查链路会通过该结构返回预估扫描行数和是否支持检查等信息。
*/
type ExplainResult struct {
	Type          string `json:"type"`
	EstimatedRows int64  `json:"estimatedRows"`
	IsSupported   bool   `json:"isSupported"`
	Message       string `json:"message"`
}
