package storage

import "time"

type User struct {
	ID           int       `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	IsPremium    bool      `db:"is_premium" json:"is_premium"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Bank struct {
	ID      int    `db:"id" json:"id"`
	Code    string `db:"code" json:"code"`
	Name    string `db:"name" json:"name"`
	APIBase string `db:"api_base_url" json:"api_base_url"`
}

type Account struct {
	ID            int       `db:"id" json:"id"`
	BankID        int       `db:"bank_id" json:"bank_id"`
	UserID        int       `db:"user_id" json:"user_id"`
	AccountNumber string    `db:"account_number" json:"account_number"`
	Currency      string    `db:"currency" json:"currency"`
	Balance       float64   `db:"balance" json:"balance"`
	Rate          float64   `db:"rate" json:"rate"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type Transaction struct {
	ID          int       `db:"id" json:"id"`
	AccountID   int       `db:"account_id" json:"account_id"`
	Amount      float64   `db:"amount" json:"amount"`
	Currency    string    `db:"currency" json:"currency"`
	Description string    `db:"description" json:"description"`
	Category    string    `db:"category" json:"category"`
	Date        time.Time `db:"date" json:"date"`
}

type Consent struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	BankID    int       `db:"bank_id" json:"bank_id"`
	Token     string    `db:"token" json:"token"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Recommendation struct {
	ID          int       `db:"id" json:"id"`
	UserID      int       `db:"user_id" json:"user_id"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	Benefit     float64   `db:"benefit" json:"benefit"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Lead struct {
	ID        int       `db:"id" json:"id"`
	UserID    int       `db:"user_id" json:"user_id"`
	ProductID string    `db:"product_id" json:"product_id"`
	Amount    float64   `db:"amount" json:"amount"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
