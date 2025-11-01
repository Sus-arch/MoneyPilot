package storage

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// Deprecated: SaveConsent kept for compatibility but project uses account_consents table now.
func (r *Repository) SaveConsent(userID, bankID int, consent_id string, expires time.Time, requesting_bank string) error {
	_, err := r.DB.Exec(
		`INSERT INTO account_consents (consent_id, user_id, bank_id, requesting_bank, permissions, status, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		consent_id, userID, bankID, requesting_bank, nil, "approved", expires,
	)
	return err
}

func (r *Repository) GetBankByCode(code string) (*Bank, error) {
	var b Bank
	err := r.DB.QueryRow(`SELECT id, code, name, api_base_url, jwks_url, created_at FROM banks WHERE code=$1`, code).
		Scan(&b.ID, &b.Code, &b.Name, &b.APIBase, &b.JWKSURL, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetUserByClientIDAndBank returns a user by client_id and optional bank code.
// If bankCode is empty, it returns the user matching client_id regardless of bank.
// If bankCode is provided, it returns a user whose client_id matches and whose bank_id is either NULL (global) or matches the bank's id.
func (r *Repository) GetUserByClientIDAndBank(clientID, bankCode string) (*User, error) {
	var u User
	if bankCode == "" {
		err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE client_id=$1`, clientID).
			Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		return &u, nil
	}

	// resolve bank id
	var bankID int
	if err := r.DB.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		return nil, err
	}

	// allow user records that are global (bank_id IS NULL) or tied to the specific bank
	err := r.DB.QueryRow(`SELECT id, client_id, bank_id, email, password_hash, segment, created_at FROM users WHERE client_id=$1 AND (bank_id IS NULL OR bank_id=$2)`, clientID, bankID).
		Scan(&u.ID, &u.ClientID, &u.BankID, &u.Email, &u.PasswordHash, &u.Segment, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SaveAccountConsentByEmailAndBank inserts an account_consents row by resolving user and bank
// from human-friendly values (email and bank code). This avoids requiring caller to know DB ids.
func (r *Repository) SaveAccountConsentByEmailAndBank(email, bankCode, consentID, requestingBank string, permissions []string, status string, expires time.Time) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int
	if err := tx.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	var bankID int
	if err := tx.QueryRow(`SELECT id FROM banks WHERE code=$1`, bankCode).Scan(&bankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("bank not found")
		}
		return err
	}

	_, err = tx.Exec(`INSERT INTO account_consents (consent_id, user_id, bank_id, requesting_bank, permissions, status, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, consentID, userID, bankID, requestingBank, pq.Array(permissions), status, expires)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetValidAccountConsentsByEmail(email string) ([]AccountConsent, error) {
	var userID int
	if err := r.DB.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	rows, err := r.DB.Query(`
		SELECT ac.id, ac.consent_id, ac.user_id, ac.bank_id, b.code AS bank_code, ac.requesting_bank, ac.permissions, ac.status, ac.expires_at, ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id=$1 AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(&c.ID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode, &c.RequestingBank, pq.Array(&c.Permissions), &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}

func (r *Repository) GetValidAccountConsentsByEmailAndBank(email, bank string) ([]AccountConsent, error) {
	var userID int
	if err := r.DB.QueryRow(`SELECT id FROM users WHERE email=$1`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	var BankID int
	if err := r.DB.QueryRow(`SELECT id FROM banks WHERE code=$1`, bank).Scan(&BankID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("bank not found")
		}
		return nil, err
	}
	rows, err := r.DB.Query(`
		SELECT ac.id, ac.consent_id, ac.user_id, ac.bank_id, b.code AS bank_code, ac.requesting_bank, ac.permissions, ac.status, ac.expires_at, ac.created_at
		FROM account_consents ac
		LEFT JOIN banks b ON b.id = ac.bank_id
		WHERE ac.user_id=$1 AND b.code=$2 AND (ac.expires_at IS NULL OR ac.expires_at > NOW())
	`, userID, BankID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []AccountConsent
	for rows.Next() {
		var c AccountConsent
		if err := rows.Scan(&c.ID, &c.ConsentID, &c.UserID, &c.BankID, &c.BankCode, &c.RequestingBank, pq.Array(&c.Permissions), &c.Status, &c.ExpiresAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}
