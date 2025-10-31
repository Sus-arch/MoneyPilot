package bankapi

import "time"

type ActiveToken struct {
	Token     string
	ExpiresAt time.Time
}

type Account struct {
	ID      string  `json:"account_id"`
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID        string    `json:"transaction_id"`
	Amount    float64   `json:"amount"`
	Date      time.Time `json:"date"`
	Category  string    `json:"category"`
	AccountID string    `json:"account_id"`
}
