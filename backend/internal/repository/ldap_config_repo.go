package repository

import (
	"nextmeta-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type LdapConfigRepository interface {
	Get() (*model.LdapConfig, error)
	Save(config *model.LdapConfig) error
}

type ldapConfigRepository struct {
	db *gorm.DB
}

func NewLdapConfigRepository(db *gorm.DB) LdapConfigRepository {
	return &ldapConfigRepository{db: db}
}

func (r *ldapConfigRepository) Get() (*model.LdapConfig, error) {
	var config model.LdapConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回空配置，不报错
			return &model.LdapConfig{}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *ldapConfigRepository) Save(config *model.LdapConfig) error {
	now := time.Now()
	if config.CreatedAt == nil {
		config.CreatedAt = &now
	}
	config.UpdatedAt = &now
	// 固定 id=1，upsert
	config.ID = 1
	return r.db.Save(config).Error
}
