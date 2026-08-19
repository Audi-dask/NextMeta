package repository

import (
	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
)

type UserOAuthBindingRepository interface {
	Create(binding *model.UserOAuthBinding) error
	FindByProvider(provider, providerUserID string) (*model.UserOAuthBinding, error)
	FindByUserID(userID uint, provider string) (*model.UserOAuthBinding, error)
	DeleteByUserID(userID uint, provider string) error
}

type userOAuthBindingRepository struct {
	db *gorm.DB
}

func NewUserOAuthBindingRepository(db *gorm.DB) UserOAuthBindingRepository {
	return &userOAuthBindingRepository{db: db}
}

func (r *userOAuthBindingRepository) Create(binding *model.UserOAuthBinding) error {
	return r.db.Create(binding).Error
}

func (r *userOAuthBindingRepository) FindByProvider(provider, providerUserID string) (*model.UserOAuthBinding, error) {
	var binding model.UserOAuthBinding
	err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *userOAuthBindingRepository) FindByUserID(userID uint, provider string) (*model.UserOAuthBinding, error) {
	var binding model.UserOAuthBinding
	err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *userOAuthBindingRepository) DeleteByUserID(userID uint, provider string) error {
	return r.db.Where("user_id = ? AND provider = ?", userID, provider).Delete(&model.UserOAuthBinding{}).Error
}
