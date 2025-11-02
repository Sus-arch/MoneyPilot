package bankapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type BankClient struct {
	Name         string
	BaseURL      string
	ClientID     string
	ClientSecret string
}

func (b *BankClient) GetToken() (*ActiveToken, error) {
	authURL := fmt.Sprintf("%s/auth/bank-token", b.BaseURL)

	params := url.Values{}
	params.Set("client_id", b.ClientID)
	params.Set("client_secret", b.ClientSecret)

	req, err := http.NewRequest("POST", authURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("[%s] token request failed: %s", b.Name, string(body))
	}

	var body map[string]json.RawMessage
	data, _ := io.ReadAll(resp.Body)
	// log.Println(string(data))
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}

	dateStr := resp.Header.Get("date")
	expiry, _ := time.Parse(time.RFC1123, dateStr)
	var token string
	json.Unmarshal(body["access_token"], &token)
	return &ActiveToken{
		Token:     token,
		ExpiresAt: expiry,
	}, nil
}
