package model

import "time"

/*
UserOAuthBinding 记录用户与第三方 OAuth 身份（飞书）的绑定关系。
一个 NextMeta 用户可绑定一个飞书 union_id，便于后续扩展 open_id。
*/
type UserOAuthBinding struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;uniqueIndex:idx_provider_user" json:"user_id"`
	Provider       string     `gorm:"type:varchar(20);not null;uniqueIndex:idx_provider_user;index:idx_provider_provider_user" json:"provider"`
	ProviderUserID string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_provider_provider_user" json:"provider_user_id"`
	UnionID        string     `gorm:"type:varchar(100);not null;default:'';index" json:"union_id"`
	OpenID         string     `gorm:"type:varchar(100);not null;default:''" json:"open_id"`
	Nickname       string     `gorm:"type:varchar(100);not null;default:''" json:"nickname"`
	AvatarURL      string     `gorm:"type:varchar(500);not null;default:''" json:"avatar_url"`
	Email          string     `gorm:"type:varchar(100);not null;default:'';index" json:"email"`
	RawProfile     string     `gorm:"type:text" json:"raw_profile"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (UserOAuthBinding) TableName() string {
	return "user_oauth_bindings"
}
