package bankapi

var Banks = map[string]*BankClient{
	"vbank": {
		Name:         "VBank",
		BaseURL:      "https://vbank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "ddslFory8voO3gxZ2CEaQnHzLfv4HVzo",
	},
	"abank": {
		Name:         "ABank",
		BaseURL:      "https://abank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "ddslFory8voO3gxZ2CEaQnHzLfv4HVzo",
	},
	"sbank": {
		Name:         "SBank",
		BaseURL:      "https://sbank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "ddslFory8voO3gxZ2CEaQnHzLfv4HVzo",
	},
}
