package repository

import (
	"time"

	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OAuthStateRepository interface {
	Create(state *model.OAuthState) error
	Consume(stateHash, provider string) (*model.OAuthState, error)
	DeleteExpired() error
}

type oauthStateRepository struct {
	db *gorm.DB
}

func NewOAuthStateRepository(db *gorm.DB) OAuthStateRepository {
	return &oauthStateRepository{db: db}
}

func (r *oauthStateRepository) Create(state *model.OAuthState) error {
	return r.db.Create(state).Error
}

// Consume atomically marks a state as consumed and returns it.
// Only succeeds if the state exists, is not consumed, and has not expired.
func (r *oauthStateRepository) Consume(stateHash, provider string) (*model.OAuthState, error) {
	var state model.OAuthState
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("state_hash = ? AND provider = ? AND consumed_at IS NULL AND expires_at > ?",
				stateHash, provider, time.Now()).
			First(&state).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&state).Update("consumed_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *oauthStateRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.OAuthState{}).Error
}
