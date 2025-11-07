package productagreements

type Product struct {
	AgreementID   string  `json:"agreement_id"`
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	ProductType   string  `json:"product_type"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date,omitempty"`
	AccountNumber *string `json:"account_number,omitempty"`
}

type ProductDetails struct {
	AgreementID    string   `json:"agreement_id"`
	ProductID      string   `json:"product_id"`
	ProductName    string   `json:"product_name"`
	ProductType    string   `json:"product_type"`
	InterestRate   *float64 `json:"interest_rate,omitempty"`
	Amount         float64  `json:"amount"`
	Status         string   `json:"status"`
	StartDate      string   `json:"start_date"`
	EndDate        *string  `json:"end_date,omitempty"`
	AccountNumber  *string  `json:"account_number,omitempty"`
	AccountBalance *float64 `json:"account_balance,omitempty"`
}
