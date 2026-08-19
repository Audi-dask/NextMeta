package model

import (
	"gorm.io/gorm"
)

/*
AuditLog 是审计日志模型。
它同时承载通用操作日志和查询窗口 SQL 审计字段，用于审计列表和首页统计。
*/
type AuditLog struct {
	gorm.Model
	UserID         uint   `gorm:"not null;comment:操作用户ID"`
	User           User   `gorm:"foreignKey:UserID"`
	Username       string `gorm:"type:varchar(50);not null;comment:操作用户名"`
	Action         string `gorm:"type:varchar(50);not null;comment:操作类型"`
	IP             string `gorm:"type:varchar(50);comment:客户端IP"`
	Details        string `gorm:"type:text;comment:详情"`
	Status         int    `gorm:"type:tinyint;default:1;comment:状态(1:成功, 0:失败)"`
	DataSourceID   uint   `gorm:"comment:数据源ID"`
	DataSource     string `gorm:"type:varchar(100);comment:数据源名称"`
	Database       string `gorm:"type:varchar(100);comment:数据库名"`
	QuerySessionID string `gorm:"type:varchar(36);index;comment:查询说明会话ID，普通操作日志可为空"`
	SQLContent     string `gorm:"type:text;comment:SQL内容"`
	DurationMS     int64  `gorm:"comment:执行耗时毫秒"`
	RowCount       int64  `gorm:"comment:查询返回行数"`
	Exported       bool   `gorm:"type:tinyint(1);default:0;comment:是否导出"`
}

/*
TableName 指定审计日志模型对应 audit_logs 表。
*/
func (AuditLog) TableName() string {
	return "audit_logs"
}
