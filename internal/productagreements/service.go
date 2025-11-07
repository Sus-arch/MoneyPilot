package productagreements

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Service struct {
	Repo        *storage.Repository
	TokenSvc    *bankapi.TokenService
	BankClients map[string]*bankapi.BankClient
	HTTPClient  *http.Client
}

func NewService(repo *storage.Repository, ts *bankapi.TokenService, clients map[string]*bankapi.BankClient) *Service {
	return &Service{
		Repo:        repo,
		TokenSvc:    ts,
		BankClients: clients,
		HTTPClient:  &http.Client{},
	}
}

// Получение списка продуктов
func (s *Service) GetProducts(userID int, bankCode string) ([]Product, error) {
	bankClient := s.BankClients[bankCode]
	if bankClient == nil {
		return nil, fmt.Errorf("unknown bank code %s", bankCode)
	}

	user, err := s.Repo.GetUserByUserIDAndBank(userID, bankCode)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found for bank " + bankCode)
	}

	consent, err := s.Repo.GetActiveProductConsentByUserAndBank(userID, bankCode)
	log.Println(consent)
	if err != nil || consent == nil {
		return nil, errors.New("no active consent for bank " + bankCode)
	}

	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/product-agreements?client_id=%s", bankClient.BaseURL, user.ClientID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-product-agreement-consent-id", consent.ConsentID)
	req.Header.Set("x-requesting-bank", "team081")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bank returned %d", resp.StatusCode)
	}

	var result struct {
		Data []Product `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// Получение деталей продукта
func (s *Service) GetProductDetails(userID int, bankCode, agreementID string) (*ProductDetails, error) {
	bankClient := s.BankClients[bankCode]
	if bankClient == nil {
		return nil, fmt.Errorf("unknown bank code %s", bankCode)
	}

	user, err := s.Repo.GetUserByUserIDAndBank(userID, bankCode)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found for bank " + bankCode)
	}

	consent, err := s.Repo.GetActiveProductConsentByUserAndBank(userID, bankCode)
	if err != nil || consent == nil {
		return nil, errors.New("no active consent for bank " + bankCode)
	}

	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/product-agreements/%s?client_id=%s", bankClient.BaseURL, agreementID, user.ClientID)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-product-agreement-consent-id", consent.ConsentID)
	req.Header.Set("x-requesting-bank", "team081")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bank returned %d", resp.StatusCode)
	}

	var result struct {
		Data ProductDetails `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// Удаление продукта (если разрешено согласием)
func (s *Service) DeleteProduct(userID int, bankCode, agreementID string, payload map[string]interface{}) error {
	bankClient := s.BankClients[bankCode]
	if bankClient == nil {
		return fmt.Errorf("unknown bank code %s", bankCode)
	}

	user, err := s.Repo.GetUserByUserIDAndBank(userID, bankCode)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found for bank " + bankCode)
	}

	consent, err := s.Repo.GetActiveProductConsentByUserAndBank(userID, bankCode)
	if err != nil || consent == nil {
		return errors.New("no active consent for bank " + bankCode)
	}

	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/product-agreements/%s?client_id=%s", bankClient.BaseURL, agreementID, user.ClientID)
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodDelete, url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-product-agreement-consent-id", consent.ConsentID)
	req.Header.Set("x-requesting-bank", "team081")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("bank returned %d: %v", resp.StatusCode, errResp)
	}

	return nil
}
