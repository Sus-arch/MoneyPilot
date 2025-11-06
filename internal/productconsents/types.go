package productconsents

type CreateProductConsentRequest struct {
	Read   bool     `json:"read_product_agreements"`
	Open   bool     `json:"open_product_agreements"`
	Close  bool     `json:"close_product_agreements"`
	Types  []string `json:"allowed_product_types"`
	Amount float64  `json:"max_amount"`
}

type CreateProductConsentResponse struct {
	Status     string `json:"status"`
	ConsentID  string `json:"consent_id"`
	Bank       string `json:"bank"`
	Reused     bool   `json:"reused"`
	AutoIssued bool   `json:"auto_issued"`
	Message    string `json:"message,omitempty"`
}
