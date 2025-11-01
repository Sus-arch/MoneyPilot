package accounts

// import (
// 	"MoneyPilot/internal/bankapi"
// 	"log"
// )

// type Service struct {
// 	TokenSvc *bankapi.TokenService
// }

// func NewService(tokenSvc *bankapi.TokenService) *Service {
// 	return &Service{TokenSvc: tokenSvc}
// }

// func (s *Service) GetAllAccounts() ([]bankapi.Account, error) {
// 	var all []bankapi.Account

// 	for _, bank := range bankapi.Banks {
// 		token, err := s.TokenSvc.GetValidToken(bank)
// 		if err != nil {
// 			log.Println("❌ Token error for bank", bank.Name, err)
// 			continue
// 		}

// 		accs, err := bank.GetAccounts(token.Token)
// 		if err != nil {
// 			log.Println("❌ Account fetch error for bank", bank.Name, err)
// 			continue
// 		}

// 		all = append(all, accs...)
// 	}

// 	return all, nil
// }
