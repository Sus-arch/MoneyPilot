package bankapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func GetToken() (*ActiveToken, error) {
	client := &http.Client{}
	baseUrl := buildURLWithParams("https://vbank.open.bankingapi.ru/auth/bank-token", map[string]string{"client_id": "team081", "client_secret": "ddslFory8voO3gxZ2CEaQnHzLfv4HVzo"})
	req, err := http.NewRequest("POST", baseUrl, nil)
	if err != nil {
		log.Println("Error to get token:", err.Error())
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error to get token:", err.Error())
		return nil, err
	}
	dateStr := resp.Header.Get("date")
	body := make(map[string]string)
	bodyByte, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyByte, &body)
	dateTime, _ := time.Parse(time.RFC1123, dateStr)

	token := ActiveToken{
		Token:     body["access_token"],
		ExpiresAt: dateTime,
	}
	return &token, nil
}

func buildURLWithParams(baseURL string, params map[string]string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()

	for key, value := range params {
		q.Add(key, value)
	}

	u.RawQuery = q.Encode()
	return u.String()
}
