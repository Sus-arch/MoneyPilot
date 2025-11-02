package accounts

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Service struct {
	repo     *storage.Repository
	tokenSvc *bankapi.TokenService
	banks    map[string]*bankapi.BankClient
	http     *http.Client
}

func NewService(repo *storage.Repository, ts *bankapi.TokenService, banks map[string]*bankapi.BankClient) *Service {
	return &Service{
		repo:     repo,
		tokenSvc: ts,
		banks:    banks,
		http:     &http.Client{Timeout: 15 * time.Second},
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
func (s *Service) FetchAllUserAccounts(userID int) ([]BankAccount, error) {
	consents, err := s.repo.GetValidAccountConsentsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load consents: %w", err)
	}

	var allAccounts []BankAccount
	requestingBank := os.Getenv("LOGIN_HAC")

	for _, consent := range consents {
		bankClient := s.banks[*consent.BankCode]
		if bankClient == nil {
			continue
		}

		tokenObj, err := s.tokenSvc.GetValidToken(bankClient)
		if err != nil {
			continue
		}
		user, _ := s.repo.GetUserByID(consent.UserID)
		url := strings.TrimRight(bankClient.BaseURL, "/") + "/accounts?client_id=" + user.ClientID
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
		req.Header.Set("X-Requesting-Bank", requestingBank)
		req.Header.Set("X-Consent-Id", consent.ConsentID)

		resp, err := s.http.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

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

		for _, a := range parsed.Data.Account {
			acc := BankAccount{
				BankCode:       *consent.BankCode,
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
			allAccounts = append(allAccounts, acc)
		}
	}

	return allAccounts, nil
}
