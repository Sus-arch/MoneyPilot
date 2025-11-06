package storage

import "time"

type User struct {
	ID           int       `db:"id" json:"id"`
	ClientID     string    `db:"client_id" json:"client_id"`
	BankID       *int      `db:"bank_id" json:"bank_id"`
	Email        *string   `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Segment      *string   `db:"segment" json:"segment"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Bank struct {
	ID        int       `db:"id" json:"id"`
	Code      string    `db:"code" json:"code"`
	Name      string    `db:"name" json:"name"`
	APIBase   string    `db:"api_base_url" json:"api_base_url"`
	JWKSURL   *string   `db:"jwks_url" json:"jwks_url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Account struct {
	ID            int       `db:"id" json:"id"`
	UserID        int       `db:"user_id" json:"user_id"`
	BankID        int       `db:"bank_id" json:"bank_id"`
	AccountNumber string    `db:"account_number" json:"account_number"`
	AccountType   string    `db:"account_type" json:"account_type"`
	Nickname      *string   `db:"nickname" json:"nickname"`
	Currency      string    `db:"currency" json:"currency"`
	Balance       float64   `db:"balance" json:"balance"`
	Status        string    `db:"status" json:"status"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type Transaction struct {
	ID          int       `db:"id" json:"id"`
	AccountID   int       `db:"account_id" json:"account_id"`
	Amount      float64   `db:"amount" json:"amount"`
	Currency    *string   `db:"currency" json:"currency"`
	Description *string   `db:"description" json:"description"`
	Category    *string   `db:"category" json:"category"`
	BookingDate time.Time `db:"booking_date" json:"booking_date"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type AccountConsent struct {
	ID             int        `db:"id" json:"id"`
	ConsentID      string     `db:"consent_id" json:"consent_id"`
	UserID         int        `db:"user_id" json:"user_id"`
	BankID         int        `db:"bank_id" json:"bank_id"`
	BankCode       *string    `db:"-" json:"bank_code"`
	RequestingBank *string    `db:"requesting_bank" json:"requesting_bank"`
	Permissions    []string   `db:"permissions" json:"permissions"`
	Status         string     `db:"status" json:"status"`
	ExpiresAt      *time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

type PaymentConsent struct {
	ID              int        `db:"id" json:"id"`
	ConsentID       string     `db:"consent_id" json:"consent_id"`
	UserID          *int       `db:"user_id" json:"user_id"`
	BankID          *int       `db:"bank_id" json:"bank_id"`
	ConsentType     *string    `db:"consent_type" json:"consent_type"`
	Amount          *float64   `db:"amount" json:"amount"`
	DebtorAccount   *string    `db:"debtor_account" json:"debtor_account"`
	CreditorAccount *string    `db:"creditor_account" json:"creditor_account"`
	ValidUntil      *time.Time `db:"valid_until" json:"valid_until"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type Payment struct {
	ID              int        `db:"id" json:"id"`
	PaymentID       string     `db:"payment_id" json:"payment_id"`
	UserID          *int       `db:"user_id" json:"user_id"`
	DebtorAccount   *string    `db:"debtor_account" json:"debtor_account"`
	CreditorAccount *string    `db:"creditor_account" json:"creditor_account"`
	Amount          *float64   `db:"amount" json:"amount"`
	Currency        *string    `db:"currency" json:"currency"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       *time.Time `db:"updated_at" json:"updated_at"`
}

type Product struct {
	ID           int       `db:"id" json:"id"`
	ProductID    string    `db:"product_id" json:"product_id"`
	BankID       *int      `db:"bank_id" json:"bank_id"`
	ProductType  *string   `db:"product_type" json:"product_type"`
	Name         *string   `db:"name" json:"name"`
	Description  *string   `db:"description" json:"description"`
	InterestRate *float64  `db:"interest_rate" json:"interest_rate"`
	MinAmount    *float64  `db:"min_amount" json:"min_amount"`
	MaxAmount    *float64  `db:"max_amount" json:"max_amount"`
	TermMonths   *int      `db:"term_months" json:"term_months"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type ProductAgreement struct {
	ID          int       `db:"id" json:"id"`
	AgreementID string    `db:"agreement_id" json:"agreement_id"`
	UserID      int       `db:"user_id" json:"user_id"`
	ProductID   string    `db:"product_id" json:"product_id"`
	Amount      *float64  `db:"amount" json:"amount"`
	TermMonths  *int      `db:"term_months" json:"term_months"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type ProductAgreementConsent struct {
	ID                     int
	ConsentID              string
	UserID                 int
	BankID                 int
	RequestingBank         *string
	ReadProductAgreements  bool
	OpenProductAgreements  bool
	CloseProductAgreements bool
	AllowedProductTypes    []string
	MaxAmount              float64
	Status                 string
	ExpiresAt              *time.Time
	CreatedAt              time.Time
	BankCode               *string
}
