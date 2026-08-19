package repository

import (
	"nextmeta-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type FeishuConfigRepository interface {
	Get() (*model.FeishuConfig, error)
	Save(config *model.FeishuConfig) error
}

type feishuConfigRepository struct {
	db *gorm.DB
}

func NewFeishuConfigRepository(db *gorm.DB) FeishuConfigRepository {
	return &feishuConfigRepository{db: db}
}

func (r *feishuConfigRepository) Get() (*model.FeishuConfig, error) {
	var config model.FeishuConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &model.FeishuConfig{}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *feishuConfigRepository) Save(config *model.FeishuConfig) error {
	now := time.Now()
	if config.CreatedAt == nil {
		config.CreatedAt = &now
	}
	config.UpdatedAt = &now
	config.ID = 1
	return r.db.Save(config).Error
}
