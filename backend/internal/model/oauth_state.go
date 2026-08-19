package model

import (
	"crypto/sha256"
	"fmt"
	"time"
)

/*
OAuthState 存储 OAuth 授权流程中的防 CSRF state 参数。
只保存 state 的 SHA-256 哈希，不保存明文。
*/
type OAuthState struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	StateHash   string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	Provider    string     `gorm:"type:varchar(20);not null;index:idx_provider_purpose" json:"provider"`
	Purpose     string     `gorm:"type:varchar(20);not null;index:idx_provider_purpose" json:"purpose"`
	RedirectURI string     `gorm:"type:varchar(500);not null;default:''" json:"redirect_uri"`
	ClientIP    string     `gorm:"type:varchar(45);not null;default:''" json:"client_ip"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt  *time.Time `json:"consumed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (OAuthState) TableName() string {
	return "oauth_states"
}

// HashState 对明文 state 做 SHA-256。
func HashState(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
