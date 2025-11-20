package bankapi

var Banks = map[string]*BankClient{
	"vbank": {
		Name:         "VBank",
		BaseURL:      "https://vbank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "Nx1FIkTkSG2Sxk9R",
	},
	"abank": {
		Name:         "ABank",
		BaseURL:      "https://abank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "Nx1FIkTkSG2Sxk9R",
	},
	"sbank": {
		Name:         "SBank",
		BaseURL:      "https://sbank.open.bankingapi.ru",
		ClientID:     "team081",
		ClientSecret: "Nx1FIkTkSG2Sxk9R",
	},
}
