package model

import "time"

/*
LoginAudit 登录审计记录。
每次登录（成功或失败）写入一条记录。
*/
type LoginAudit struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"column:user_id" json:"user_id"`
	Username     string    `gorm:"column:username" json:"username"`
	RealName     string    `gorm:"column:real_name;->" json:"real_name"`
	LoginMethod  string    `gorm:"column:login_method" json:"login_method"`
	ClientIP     string    `gorm:"column:client_ip" json:"client_ip"`
	UserAgent    string    `gorm:"column:user_agent" json:"-"`
	Success      bool      `gorm:"column:success" json:"success"`
	ErrorMessage string    `gorm:"column:error_message" json:"error_message"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

/*
TableName 指定登录审计表名。
*/
func (LoginAudit) TableName() string {
	return "login_audits"
}
