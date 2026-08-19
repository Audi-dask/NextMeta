package dto

/*
QueryAuditRecordResponse 是查询审计日志中的单条 SQL 明细。
audit_log_handler 会把 model.AuditLog 映射为该结构返回给前端审计查询页。
*/
type QueryAuditRecordResponse struct {
	ID          uint   `json:"id"`
	DataSource  string `json:"data_source"`
	Database    string `json:"database"`
	SQL         string `json:"sql"`
	SubmittedAt string `json:"submitted_at"`
	Duration    int64  `json:"duration"`
	Status      string `json:"status"`
	Rows        int64  `json:"rows"`
}

/*
QueryAuditLogResponse 是查询审计列表返回给前端的聚合结构。
每条日志当前会包装为一个 Records 明细列表，便于前端展示详情。
*/
type QueryAuditLogResponse struct {
	ID          uint                       `json:"id"`
	TicketNo    string                     `json:"ticket_no"`
	Querier     string                     `json:"querier"`
	QuerierName string                     `json:"querier_name"`
	SubmittedAt string                     `json:"submitted_at"`
	Description string                     `json:"description"`
	Exported    bool                       `json:"exported"`
	Records     []QueryAuditRecordResponse `json:"records"`
}
