package storage

import (
	"database/sql"
	"time"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) SaveConsent(userID, bankID int, token string, expires time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO consents (user_id, bank_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, userID, bankID, token, expires)
	return err
}

func (r *Repository) GetValidConsents(userID int) ([]Consent, error) {
	rows, err := r.DB.Query(`
		SELECT id, user_id, bank_id, token, expires_at, created_at
		FROM consents WHERE user_id=$1 AND expires_at > NOW()
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []Consent
	for rows.Next() {
		var c Consent
		if err := rows.Scan(&c.ID, &c.UserID, &c.BankID, &c.Token, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}
