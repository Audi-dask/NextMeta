package model

import "time"

/*
FeishuConfig 存储飞书企业自建应用的 OAuth 网页授权配置。
独立于 system_settings 表，由前端「三方登录配置」页面管理。
*/
type FeishuConfig struct {
	ID                  uint   `gorm:"primaryKey" json:"id"`
	Enabled             bool   `gorm:"not null;default:false" json:"enabled"`
	AppID               string `gorm:"type:varchar(100);not null;default:''" json:"app_id"`
	AppSecret           string `gorm:"type:varchar(255);not null;default:''" json:"-"`
	AppSecretConfigured bool   `gorm:"-" json:"app_secret_configured"` // 前端判断是否已配置
	RedirectURI         string `gorm:"type:varchar(500);not null;default:''" json:"redirect_uri"`
	DefaultRole         string `gorm:"type:varchar(20);not null;default:'developer'" json:"default_role"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func (FeishuConfig) TableName() string {
	return "feishu_config"
}
