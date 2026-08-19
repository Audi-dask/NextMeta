package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

/*
LoginAuditRepository 管理登录审计记录的持久化。
*/
type LoginAuditRepository interface {
	Create(audit *model.LoginAudit) error
	FindAll(offset, limit int, loginMethod, status, clientIP string) ([]model.LoginAudit, int64, error)
}

type loginAuditRepository struct {
	db *gorm.DB
}

func NewLoginAuditRepository(db *gorm.DB) LoginAuditRepository {
	return &loginAuditRepository{db: db}
}

func (r *loginAuditRepository) Create(audit *model.LoginAudit) error {
	return r.db.Omit("RealName").Create(audit).Error
}

func (r *loginAuditRepository) FindAll(offset, limit int, loginMethod, status, clientIP string) ([]model.LoginAudit, int64, error) {
	var total int64
	var audits []model.LoginAudit

	query := r.db.Table("login_audits AS la")
	if loginMethod != "" {
		query = query.Where("la.login_method = ?", loginMethod)
	}
	if status == "success" {
		query = query.Where("la.success = ?", true)
	} else if status == "failed" {
		query = query.Where("la.success = ?", false)
	}
	if clientIP != "" {
		query = query.Where("la.client_ip LIKE ?", "%"+clientIP+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Select("la.*, COALESCE(NULLIF(u.real_name, ''), la.username) AS real_name").
		Joins("LEFT JOIN users AS u ON u.id = la.user_id").
		Order("la.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&audits).Error; err != nil {
		return nil, 0, err
	}

	return audits, total, nil
}
