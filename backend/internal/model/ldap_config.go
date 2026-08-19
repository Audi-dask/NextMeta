package model

import "time"

/*
LdapConfig 存储 LDAP 连接信息和过滤/映射策略。
代替 config.yaml 中的 ldap 段和 system_settings 中分散的 ldap_* key，
统一由前端「三方登录配置」页面管理，支持热更新。
*/
type LdapConfig struct {
	ID                 uint   `gorm:"primaryKey" json:"id"`
	Enabled            bool   `gorm:"not null;default:false" json:"enabled"`
	URL                string `gorm:"type:varchar(255);not null;default:''" json:"url"`
	BaseDN             string `gorm:"type:varchar(255);not null;default:''" json:"base_dn"`
	GroupBaseDN        string `gorm:"type:varchar(255);not null;default:''" json:"group_base_dn"`
	BindDN             string `gorm:"type:varchar(255);not null;default:''" json:"bind_dn"`
	BindPass           string `gorm:"type:varchar(255);not null;default:''" json:"-"` // 不序列化到 JSON
	BindPassConfigured bool   `gorm:"-" json:"bind_pass_configured"`                  // 前端判断是否已配置

	// 过滤规则与字段映射（原 system_settings 中的 key 迁移到这里）
	UserFilter      string `gorm:"type:varchar(500);not null;default:''" json:"user_filter"`
	GroupFilter     string `gorm:"type:varchar(500);not null;default:''" json:"group_filter"`
	MappingUsername string `gorm:"type:varchar(100);not null;default:''" json:"mapping_username"`
	MappingRealName string `gorm:"type:varchar(100);not null;default:''" json:"mapping_real_name"`
	MappingEmail    string `gorm:"type:varchar(100);not null;default:''" json:"mapping_email"`
	SyncInterval    int    `gorm:"not null;default:30" json:"sync_interval"`
	ExcludeKeywords string `gorm:"type:varchar(255);not null;default:''" json:"exclude_keywords"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func (LdapConfig) TableName() string {
	return "ldap_config"
}
