package model

import (
	"crypto/sha256"
	"fmt"
	"time"
)

/*
OAuthLoginTicket 是一次性登录票据。
回调完成后生成 ticket，重定向到前端，前端凭 ticket 换取 JWT。
每个 ticket 只能兑换一次，过期作废。
*/
type OAuthLoginTicket struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	TicketHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	Provider   string     `gorm:"type:varchar(20);not null" json:"provider"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (OAuthLoginTicket) TableName() string {
	return "oauth_login_tickets"
}

// HashTicket 对明文 ticket 做 SHA-256。
func HashTicket(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
