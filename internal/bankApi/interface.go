package bankapi

type BankAPI interface {
	GetToken() (*ActiveToken, error)
	GetAccounts(token string) ([]Account, error)
	GetTransactions(token string, accountID string) ([]Transaction, error)
}
