package repository

import (
	"time"

	"nextmeta-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OAuthLoginTicketRepository interface {
	Create(ticket *model.OAuthLoginTicket) error
	Consume(ticketHash string) (*model.OAuthLoginTicket, error)
	DeleteExpired() error
}

type oauthLoginTicketRepository struct {
	db *gorm.DB
}

func NewOAuthLoginTicketRepository(db *gorm.DB) OAuthLoginTicketRepository {
	return &oauthLoginTicketRepository{db: db}
}

func (r *oauthLoginTicketRepository) Create(ticket *model.OAuthLoginTicket) error {
	return r.db.Create(ticket).Error
}

// Consume atomically marks a ticket as consumed and returns it.
func (r *oauthLoginTicketRepository) Consume(ticketHash string) (*model.OAuthLoginTicket, error) {
	var ticket model.OAuthLoginTicket
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("ticket_hash = ? AND consumed_at IS NULL AND expires_at > ?",
				ticketHash, time.Now()).
			First(&ticket).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&ticket).Update("consumed_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *oauthLoginTicketRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.OAuthLoginTicket{}).Error
}
