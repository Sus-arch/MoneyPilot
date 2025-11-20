package productagreements

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/cache"
	"MoneyPilot/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Service struct {
	Repo         *storage.Repository
	TokenSvc     *bankapi.TokenService
	BankClients  map[string]*bankapi.BankClient
	HTTPWrapper  *bankapi.ClientWrapper
	CacheService *cache.CacheService
}

func NewService(repo *storage.Repository, ts *bankapi.TokenService, clients map[string]*bankapi.BankClient, cacheService *cache.CacheService) *Service {
	config := bankapi.DefaultConfig()
	httpWrapper := bankapi.NewClientWrapper(cacheService, config)

	return &Service{
		Repo:         repo,
		TokenSvc:     ts,
		BankClients:  clients,
		HTTPWrapper:  httpWrapper,
		CacheService: cacheService,
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

	// Генерируем ключ кэша
	cacheKey := fmt.Sprintf("products:%s:%s:%d", bankCode, user.ClientID, userID)
	ctx := context.Background()

	// Используем обертку с кэшированием, retry, rate limiting и circuit breaker
	resp, err := s.HTTPWrapper.Do(ctx, bankCode, req, cacheKey, true)
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

	// Сохраняем product agreements в БД
	for _, p := range result.Data {
		var amount *float64
		if p.Amount > 0 {
			amount = &p.Amount
		}
		if err := s.Repo.UpsertProductAgreement(user.ID, p.AgreementID, p.ProductID, amount, nil, p.Status); err != nil {
			log.Printf("Failed to save product agreement %s to DB: %v\n", p.AgreementID, err)
		}
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

	// Генерируем ключ кэша
	cacheKey := fmt.Sprintf("product:%s:%s:%s:%d", bankCode, user.ClientID, agreementID, userID)
	ctx := context.Background()

	// Используем обертку с кэшированием, retry, rate limiting и circuit breaker
	resp, err := s.HTTPWrapper.Do(ctx, bankCode, req, cacheKey, true)
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

	// Сохраняем product agreement в БД
	var amount *float64
	if result.Data.Amount > 0 {
		amount = &result.Data.Amount
	}
	var termMonths *int
	if err := s.Repo.UpsertProductAgreement(user.ID, result.Data.AgreementID, result.Data.ProductID, amount, termMonths, result.Data.Status); err != nil {
		log.Printf("Failed to save product agreement %s to DB: %v\n", result.Data.AgreementID, err)
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

	ctx := context.Background()
	// Для DELETE запросов не используем кэш, но применяем retry, rate limiting и circuit breaker
	resp, err := s.HTTPWrapper.Do(ctx, bankCode, req, "", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp map[string]interface{}
		json.Unmarshal(bodyBytes, &errResp)
		return fmt.Errorf("bank returned %d: %v", resp.StatusCode, errResp)
	}

	// Удаляем из кэша при успешном удалении
	cacheKey := fmt.Sprintf("product:%s:%s:%s:%d", bankCode, user.ClientID, agreementID, userID)
	_ = s.CacheService.Delete(cacheKey)

	// Также удаляем из БД, если есть метод для этого
	// Пока оставляем в БД для истории

	return nil
}

// ProductCatalogItem представляет продукт из каталога банка
type ProductCatalogItem struct {
	ProductID    string  `json:"productId"`
	ProductType  string  `json:"productType"`
	ProductName  string  `json:"productName"`
	Description  *string `json:"description,omitempty"`
	InterestRate *string `json:"interestRate,omitempty"`
	MinAmount    *string `json:"minAmount,omitempty"`
	MaxAmount    *string `json:"maxAmount,omitempty"`
	TermMonths   *int    `json:"termMonths,omitempty"`
}

// GetProductsCatalog получает каталог продуктов из всех указанных банков
// Не требует согласия, использует bank_token
func (s *Service) GetProductsCatalog(userID int, bankCodes []string, productType string) (map[string]interface{}, error) {
	// Получаем текущего пользователя для получения client_id
	currentUser, err := s.Repo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load current user: %w", err)
	}

	// Если bankCodes не указаны, получаем все банки
	if len(bankCodes) == 0 {
		for code := range s.BankClients {
			bankCodes = append(bankCodes, code)
		}
	}

	var allProducts []ProductCatalogItem
	ctx := context.Background()

	for _, bankCode := range bankCodes {
		bankCode = strings.ToLower(strings.TrimSpace(bankCode))
		bankClient, ok := s.BankClients[bankCode]
		if !ok || bankClient == nil {
			continue
		}

		// Получаем bank_id для этого банка
		bank, err := s.Repo.GetBankByCode(bankCode)
		if err != nil {
			continue
		}

		// Получаем правильного пользователя для этого банка по client_id и bank_id
		bankUser, err := s.Repo.GetUserByClientIDAndBankID(currentUser.ClientID, bank.ID)
		if err != nil {
			continue
		}

		// Получаем token для банка
		tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
		if err != nil {
			continue
		}

		// Формируем URL с опциональным фильтром product_type
		url := strings.TrimRight(bankClient.BaseURL, "/") + "/products?client_id=" + bankUser.ClientID
		if productType != "" {
			url += "&product_type=" + productType
		}

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("x-requesting-bank", "team081")

		// Генерируем ключ кэша
		cacheKey := fmt.Sprintf("products_catalog:%s:%s:%s", bankCode, productType, bankUser.ClientID)

		// Используем обертку с кэшированием, retry, rate limiting и circuit breaker
		resp, err := s.HTTPWrapper.Do(ctx, bankCode, req, cacheKey, true)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		body, _ := io.ReadAll(resp.Body)

		var parsed struct {
			Data struct {
				Product []ProductCatalogItem `json:"product"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &parsed); err != nil {
			continue
		}

		// Сохраняем продукты в БД
		for _, p := range parsed.Data.Product {
			// Парсим числовые значения
			var interestRate *float64
			if p.InterestRate != nil && *p.InterestRate != "" {
				if rate, err := strconv.ParseFloat(*p.InterestRate, 64); err == nil {
					interestRate = &rate
				}
			}

			var minAmount, maxAmount *float64
			if p.MinAmount != nil && *p.MinAmount != "" {
				if amt, err := strconv.ParseFloat(*p.MinAmount, 64); err == nil {
					minAmount = &amt
				}
			}
			if p.MaxAmount != nil && *p.MaxAmount != "" {
				if amt, err := strconv.ParseFloat(*p.MaxAmount, 64); err == nil {
					maxAmount = &amt
				}
			}

			// Сохраняем продукт в БД
			if err := s.Repo.UpsertProduct(bank.ID, p.ProductID, p.ProductType, p.ProductName, p.Description, interestRate, minAmount, maxAmount, p.TermMonths); err != nil {
				log.Printf("Failed to save product %s to DB: %v\n", p.ProductID, err)
			}
		}

		// Добавляем продукты в общий список
		allProducts = append(allProducts, parsed.Data.Product...)
	}

	result := map[string]interface{}{
		"data": map[string]interface{}{
			"product": allProducts,
		},
	}

	return result, nil
}
