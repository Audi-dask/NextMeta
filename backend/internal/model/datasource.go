package model

import (
	"gorm.io/gorm"
)

/*
DataSource 是 MySQL 数据源模型。
它保存连接信息、环境、超时配置、启用状态，以及与字段脱敏规则的关联。
*/
type DataSource struct {
	gorm.Model
	Name                string                  `gorm:"column:name;type:varchar(100);not null;comment:数据源名称" json:"name"`
	Type                string                  `gorm:"column:type;type:varchar(20);not null;comment:数据库类型(MySQL)" json:"type"`
	Host                string                  `gorm:"column:host;type:varchar(255);not null;comment:主机地址" json:"host"`
	Port                int                     `gorm:"column:port;type:int;not null;comment:端口" json:"port"`
	Database            string                  `gorm:"column:database;type:varchar(100);comment:数据库名" json:"database"`
	Username            string                  `gorm:"column:username;type:varchar(100);comment:用户名" json:"username"`
	Password            string                  `gorm:"column:password;type:varchar(255);comment:密码(加密)" json:"-"`
	Environment         string                  `gorm:"column:environment;type:varchar(20);default:'生产';comment:环境" json:"environment"`
	ExecutionTimeout    int                     `gorm:"column:execution_timeout_seconds;default:30;comment:执行超时时间(秒)" json:"executionTimeoutSeconds"`
	QueryTimeoutSeconds int                     `gorm:"column:query_timeout_seconds;default:30;comment:查询超时时间(秒)" json:"queryTimeoutSeconds"`
	ConnectTimeout      int64                   `gorm:"column:connect_timeout;default:10;comment:连接超时时间(秒)" json:"connectTimeout"`
	Description         string                  `gorm:"column:description;type:varchar(255);comment:描述" json:"description"`
	MaskingRules        []DataSourceMaskingRule `gorm:"foreignKey:DataSourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"maskingRules"`
	Status              string                  `gorm:"column:status;type:varchar(20);default:'enabled';comment:状态(enabled/disabled)" json:"status"`
	AccessMode          string                  `gorm:"column:access_mode;type:varchar(20);default:'read_write';comment:访问模式(read_only/read_write)" json:"accessMode"`
}

/*
TableName 指定数据源模型对应 data_sources 表。
*/
func (DataSource) TableName() string {
	return "data_sources"
}
