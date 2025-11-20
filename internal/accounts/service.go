package accounts

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/cache"
	"MoneyPilot/internal/storage"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Service struct {
	repo         *storage.Repository
	tokenSvc     *bankapi.TokenService
	banks        map[string]*bankapi.BankClient
	httpWrapper  *bankapi.ClientWrapper
	cacheService *cache.CacheService
}

func NewService(repo *storage.Repository, ts *bankapi.TokenService, banks map[string]*bankapi.BankClient, cacheService *cache.CacheService) *Service {
	config := bankapi.DefaultConfig()
	httpWrapper := bankapi.NewClientWrapper(cacheService, config)

	return &Service{
		repo:         repo,
		tokenSvc:     ts,
		banks:        banks,
		httpWrapper:  httpWrapper,
		cacheService: cacheService,
	}
}

type BankAccount struct {
	BankCode       string `json:"bank"`
	AccountID      string `json:"account_id"`
	Nickname       string `json:"nickname"`
	AccountType    string `json:"account_type"`
	AccountSubType string `json:"account_subtype"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	Owner          string `json:"owner,omitempty"`
}

// FetchAllUserAccounts получает счета со всех банков, на которые есть согласие
// Если указан список bankCodes, возвращаются счета только для этих банков
func (s *Service) FetchAllUserAccounts(userID int, bankCodes []string) ([]BankAccount, error) {
	consents, err := s.repo.GetValidAccountConsentsByUserID(userID)

	if err != nil {
		return nil, fmt.Errorf("failed to load consents: %w", err)
	}

	// Создаем map для быстрой проверки, какие банки нужно включить
	allowedBanks := make(map[string]bool)
	if len(bankCodes) > 0 {
		for _, code := range bankCodes {
			allowedBanks[strings.ToLower(strings.TrimSpace(code))] = true
		}
	}

	// Группируем согласия по банку, используем только самое свежее согласие для каждого банка
	consentsByBank := make(map[string]storage.AccountConsent)
	for _, consent := range consents {
		if consent.BankCode == nil {
			continue
		}
		bankCode := *consent.BankCode
		// Если указан список банков, фильтруем по нему
		if len(allowedBanks) > 0 && !allowedBanks[strings.ToLower(bankCode)] {
			continue
		}
		// Если согласия для этого банка еще нет, или текущее согласие новее - используем его
		if existing, exists := consentsByBank[bankCode]; !exists || consent.CreatedAt.After(existing.CreatedAt) {
			consentsByBank[bankCode] = consent
		}
	}

	var allAccounts []BankAccount
	// карта для удаления дубликатов по паре bank+account_id
	seen := make(map[string]struct{})
	requestingBank := os.Getenv("LOGIN_HAC")
	ctx := context.Background()

	// Используем map для дедупликации счетов по bank_code + account_id
	accountsMap := make(map[string]BankAccount)

	// Получаем банк по коду для получения bank_id
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	for bankCode, consent := range consentsByBank {
		bankClient := s.banks[bankCode]
		if bankClient == nil {
			continue
		}

		tokenObj, err := s.tokenSvc.GetValidToken(bankClient)
		if err != nil {
			continue
		}

		url := strings.TrimRight(bankClient.BaseURL, "/") + "/accounts?client_id=" + user.ClientID
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
		req.Header.Set("X-Requesting-Bank", requestingBank)
		req.Header.Set("X-Consent-Id", consent.ConsentID)

		// Генерируем ключ кэша
		cacheKey := fmt.Sprintf("accounts:%s:%s:%d", bankCode, user.ClientID, userID)

		// Используем обертку с кэшированием, retry, rate limiting и circuit breaker
		resp, err := s.httpWrapper.Do(ctx, bankCode, req, cacheKey, true)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		body, _ := io.ReadAll(resp.Body)

		// Структура OpenBanking
		var parsed struct {
			Data struct {
				Account []struct {
					AccountID      string `json:"accountId"`
					Status         string `json:"status"`
					Currency       string `json:"currency"`
					AccountType    string `json:"accountType"`
					AccountSubType string `json:"accountSubType"`
					Nickname       string `json:"nickname"`
					Account        []struct {
						Name string `json:"name"`
					} `json:"account"`
				} `json:"account"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &parsed); err != nil {
			continue
		}

		// Получаем bank_id для сохранения в БД
		bank, err := s.repo.GetBankByCode(bankCode)
		if err != nil {
			continue
		}

		// Парсим и сохраняем счета в БД
		for _, a := range parsed.Data.Account {
			key := fmt.Sprintf("%s|%s", *consent.BankCode, a.AccountID)
			if _, ok := seen[key]; ok {
				continue
			}
			acc := BankAccount{
				BankCode:       bankCode,
				AccountID:      a.AccountID,
				AccountType:    a.AccountType,
				AccountSubType: a.AccountSubType,
				Currency:       a.Currency,
				Status:         a.Status,
				Nickname:       a.Nickname,
			}
			if len(a.Account) > 0 {
				acc.Owner = a.Account[0].Name
			}

			// Дедупликация: используем bank_code + account_id как ключ
			key := fmt.Sprintf("%s:%s", bankCode, a.AccountID)
			if _, exists := accountsMap[key]; !exists {
				accountsMap[key] = acc
			}

			// Сохраняем счет в БД
			accountType := a.AccountType
			if accountType == "" {
				accountType = "checking"
			}
			if err := s.repo.UpsertAccount(userID, bank.ID, a.AccountID, accountType, a.Nickname, a.Currency, a.Status); err != nil {
				// Логируем ошибку, но не прерываем выполнение
				fmt.Printf("Failed to save account %s to DB: %v\n", a.AccountID, err)
			}
		}
	}

	// Преобразуем map в slice
	for _, acc := range accountsMap {
		allAccounts = append(allAccounts, acc)
	}

	return allAccounts, nil
}

func (s *Service) FetchAccountBalance(userID int, bankCode, accountID string) (map[string]interface{}, error) {
	result, err := s.proxyBankRequest(userID, bankCode, "/accounts/"+accountID+"/balances")
	if err != nil {
		return nil, err
	}

	// Парсим и сохраняем балансы в БД
	if data, ok := result["data"].(map[string]interface{}); ok {
		if balanceArray, ok := data["balance"].([]interface{}); ok {
			for _, balanceItem := range balanceArray {
				if balanceMap, ok := balanceItem.(map[string]interface{}); ok {
					// Ищем баланс типа InterimAvailable
					if balanceType, ok := balanceMap["type"].(string); ok && balanceType == "InterimAvailable" {
						if amountMap, ok := balanceMap["amount"].(map[string]interface{}); ok {
							if amountStr, ok := amountMap["amount"].(string); ok {
								var balance float64
								if _, err := fmt.Sscanf(amountStr, "%f", &balance); err == nil {
									// Обновляем баланс в БД
									if err := s.repo.UpdateAccountBalance(accountID, balance); err != nil {
										fmt.Printf("Failed to update balance for account %s: %v\n", accountID, err)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}

func (s *Service) FetchAccountTransactions(userID int, bankCode, accountID, from, to, page, limit string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/accounts/%s/transactions?from_booking_date_time=%s&to_booking_date_time=%s&page=%s&limit=%s",
		accountID, from, to, page, limit)
	return s.proxyBankRequest(userID, bankCode, path)
}

func (s *Service) FetchAccountDetails(userID int, bankCode, accountID string) (map[string]interface{}, error) {
	return s.proxyBankRequest(userID, bankCode, "/accounts/"+accountID)
}

func (s *Service) proxyBankRequest(userID int, bankCode, path string) (map[string]interface{}, error) {
	consents, err := s.repo.GetValidAccountConsentsByUserIDAndBank(userID, bankCode)
	if err != nil || len(consents) == 0 {
		return nil, fmt.Errorf("no valid consent found for bank %s", bankCode)
	}
	consent := consents[len(consents)-1]

	bank := s.banks[bankCode]
	if bank == nil {
		return nil, fmt.Errorf("unknown bank %s", bankCode)
	}

	token, err := s.tokenSvc.GetValidToken(bank)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(bank.BaseURL, "/") + path
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("X-Requesting-Bank", os.Getenv("LOGIN_HAC"))
	req.Header.Set("X-Consent-Id", consent.ConsentID)

	// Генерируем ключ кэша на основе пути и параметров
	cacheKey := fmt.Sprintf("api:%s:%s:%x", bankCode, path, md5.Sum([]byte(path+consent.ConsentID)))

	ctx := context.Background()
	// Используем обертку с кэшированием, retry, rate limiting и circuit breaker
	resp, err := s.httpWrapper.Do(ctx, bankCode, req, cacheKey, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	// Простая дедубликация для массива балансов: если в ответе есть data.balance — удалим точные дубликаты
	if dataRaw, ok := result["data"]; ok {
		if dataMap, ok := dataRaw.(map[string]interface{}); ok {
			if balancesRaw, ok := dataMap["balance"]; ok {
				if balancesSlice, ok := balancesRaw.([]interface{}); ok {
					seenBal := make(map[string]struct{})
					var deduped []interface{}
					for _, item := range balancesSlice {
						if itmMap, ok := item.(map[string]interface{}); ok {
							// формируем ключ по accountId + amount.amount + amount.currency
							key := ""
							if accId, ok := itmMap["accountId"].(string); ok {
								key = accId
							}
							if amountObj, ok := itmMap["amount"].(map[string]interface{}); ok {
								if amt, ok := amountObj["amount"].(string); ok {
									key += "|" + amt
								}
								if cur, ok := amountObj["currency"].(string); ok {
									key += "|" + cur
								}
							}
							if key == "" {
								// fallback: marshal item
								kb, _ := json.Marshal(itmMap)
								key = string(kb)
							}
							if _, exists := seenBal[key]; !exists {
								seenBal[key] = struct{}{}
								deduped = append(deduped, item)
							}
						} else {
							// Если не map, просто добавляем по-быстрому, уникальность не гарантируем
							deduped = append(deduped, item)
						}
					}
					dataMap["balance"] = deduped
					result["data"] = dataMap
				}
			}
		}
	}
	return result, nil
}
