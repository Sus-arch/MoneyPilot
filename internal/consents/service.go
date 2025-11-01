package consents

import (
	"MoneyPilot/internal/bankapi"
)

type Service struct {
	TokenSvc *bankapi.TokenService
}

func NewService(tokenSvc *bankapi.TokenService) *Service {
	return &Service{TokenSvc: tokenSvc}
}
