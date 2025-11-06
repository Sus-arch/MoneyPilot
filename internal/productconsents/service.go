package productconsents

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
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
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) CreateConsent(ctx context.Context, userID int, bankCode string, req CreateProductConsentRequest) (*CreateProductConsentResponse, error) {
	bankClient := s.BankClients[bankCode]
	if bankClient == nil {
		return nil, errors.New("unknown bank code")
	}

	user, err := s.Repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// 1️⃣ Проверка существующего согласия
	existing, _ := s.Repo.GetActiveProductConsentByUserAndBank(userID, bankCode)
	if existing != nil {
		// если старое согласие не покрывает новые разрешения → отзываем
		if (req.Open && !existing.OpenProductAgreements) || (req.Close && !existing.CloseProductAgreements) || (req.Read && !existing.ReadProductAgreements) {
			_ = s.revokeConsent(bankClient, existing.ConsentID)
			_ = s.Repo.DeleteProductConsent(existing.ConsentID)
		} else {
			return &CreateProductConsentResponse{
				Status:     existing.Status,
				ConsentID:  existing.ConsentID,
				Bank:       bankCode,
				Reused:     true,
				Message:    "Active consent already exists",
				AutoIssued: false,
			}, nil
		}
	}

	// 2️⃣ Получаем токен
	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		return nil, err
	}

	// 3️⃣ Создаём запрос к банку
	target := fmt.Sprintf(strings.TrimRight(bankClient.BaseURL, "/")+"/product-agreement-consents/request?client_id=%s", user.ClientID)
	payload := map[string]interface{}{
		"requesting_bank":          "team081",
		"client_id":                user.ClientID,
		"read_product_agreements":  req.Read,
		"open_product_agreements":  req.Open,
		"close_product_agreements": req.Close,
		"allowed_product_types":    req.Types,
		"max_amount":               req.Amount,
		"valid_until":              time.Now().Add(90 * 24 * time.Hour).Format("2006-01-02T15:04:05"),
		"reason":                   "Финансовый агрегатор для управления продуктами",
	}
	log.Println(payload)
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+tokenObj.Token)

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read bank response: %w", err)
	}

	log.Printf("[product-consents] bank raw response: %s", string(bodyBytes))

	var bankResp struct {
		RequestID    string    `json:"request_id"`
		ConsentID    string    `json:"consent_id"`
		Status       string    `json:"status"`
		AutoApproved bool      `json:"auto_approved"`
		Message      string    `json:"message"`
		ValidUntil   time.Time `json:"valid_until"`
	}

	if err := json.Unmarshal(bodyBytes, &bankResp); err != nil {
		return nil, fmt.Errorf("failed to decode bank response: %w", err)
	}

	log.Printf("[product-consents] parsed response: %+v", bankResp)

	// 4️⃣ Сохраняем в БД
	// expires := time.Now().Add(90 * 24 * time.Hour)
	if err := s.Repo.SaveProductAgreementConsent(
		user.ClientID,
		bankCode,
		bankResp.RequestID,
		bankResp.ConsentID,
		"team081",
		req.Read,
		req.Open,
		req.Close,
		req.Types,
		req.Amount,
		bankResp.Status,
		bankResp.ValidUntil,
	); err != nil {
		return nil, err
	}

	return &CreateProductConsentResponse{
		Status:     bankResp.Status,
		ConsentID:  bankResp.ConsentID,
		Bank:       bankCode,
		Reused:     false,
		AutoIssued: false,
	}, nil
}

func (s *Service) revokeConsent(bankClient *bankapi.BankClient, consentID string) error {
	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		return err
	}
	url := strings.TrimRight(bankClient.BaseURL, "/") + "/product-agreement-consents/" + consentID
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
