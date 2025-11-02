package accountconsents

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler хранит зависимости: репозиторий, сервис токенов и реестр банков
type ConsentHandler struct {
	Repo        *storage.Repository
	TokenSvc    *bankapi.TokenService
	BankClients map[string]*bankapi.BankClient
	HTTPClient  *http.Client
}

func NewConsentHandler(repo *storage.Repository, ts *bankapi.TokenService, clients map[string]*bankapi.BankClient) *ConsentHandler {
	return &ConsentHandler{
		Repo:        repo,
		TokenSvc:    ts,
		BankClients: clients,
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

type CreateConsentRequest struct {
	ClientID           string   `json:"client_id" binding:"required"`
	Permissions        []string `json:"permissions" binding:"required"`
	Reason             string   `json:"reason"`
	RequestingBank     string   `json:"requesting_bank"`
	RequestingBankName string   `json:"requesting_bank_name"`
}

// POST /api/account-consent
// Requires Authorization: Bearer <user-jwt> (middleware must set user_id)
// Requires header X-Bank-Code: <vbank|sbank|abank>
func (h *ConsentHandler) CreateConsent(c *gin.Context) {
	// 1) авторизация: берем user_id из контекста
	userID := c.GetInt("user_id")
	log.Println(userID)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 2) парсим тело
	// var req CreateConsentRequest
	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
	// 	return
	// }

	// 3) выбираем банк по заголовку
	bankCode := c.GetHeader("X-Bank-Code")
	if bankCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Bank-Code header required"})
		return
	}
	bankClient, ok := h.BankClients[bankCode]
	if !ok || bankClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown bank code"})
		return
	}

	// 4) загружаем пользователя
	user, err := h.Repo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user", "details": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user record missing"})
		return
	}
	// 5) проверяем — нет ли уже действующего consent для этого user+bank
	// предпочитаем метод по userID, если он есть; иначе используем email
	var existing []storage.AccountConsent
	if methodExists := true; methodExists {
		// если у репозитория есть GetValidAccountConsentsByUserIDAndBank — вызываем его
		if cons, err2 := h.Repo.GetValidAccountConsentsByEmailAndBank(user.ClientID, bankCode); err2 == nil && len(cons) > 0 {
			existing = cons
		}
	}
	// fallback: по email (если email непустой)
	if len(existing) == 0 && user.Email != nil {
		if cons, err2 := h.Repo.GetValidAccountConsentsByEmailAndBank(*user.Email, bankCode); err2 == nil && len(cons) > 0 {
			existing = cons
		}
	}

	if len(existing) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":    "active consent exists",
			"consent_id": existing[0].ConsentID,
			"status":     existing[0].Status,
			"expires_at": existing[0].ExpiresAt,
		})
		return
	}

	// 6) Проверка TokenSvc
	if h.TokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token service not configured"})
		return
	}

	// 7) Получаем валидный токен банка через TokenService
	tokenObj, err := h.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to obtain bank token", "details": err.Error()})
		return
	}
	if tokenObj == nil || tokenObj.Token == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "bank token empty"})
		return
	}
	bankToken := tokenObj.Token

	// 8) Формируем и отправляем запрос в банк
	target := strings.TrimRight(bankClient.BaseURL, "/") + "/account-consents/request"
	payload := map[string]interface{}{
		"client_id":            user.ClientID,
		"permissions":          []string{"ReadAccountsDetail", "ReadBalances"},
		"reason":               "Агрегация счетов для HackAPI",
		"requesting_bank":      "team081",
		"requesting_bank_name": "MoneyPilot",
	}
	bodyBytes, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, target, strings.NewReader(string(bodyBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request", "details": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+bankToken)

	resp, err := h.HTTPClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "bank request failed", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 9) Если банк вернул ошибку — проксируем её клиенту
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return
	}

	// 10) Парсим ответ
	bankResp := make(map[string]json.RawMessage)
	var consent_id string
	var status string
	var auto_approved bool

	if err := json.Unmarshal(respBody, &bankResp); err != nil {
		log.Printf("warning: can't parse bank response: %v body=%s", err, string(respBody))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unmarshal failed", "details": err.Error()})
	}
	json.Unmarshal(bankResp["consent_id"], &consent_id)
	json.Unmarshal(bankResp["status"], &status)
	json.Unmarshal(bankResp["auto_approved"], &auto_approved)

	// 11) Сохраняем согласие в БД (если банк вернул consent_id)
	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	saveErr := h.Repo.SaveAccountConsentByClientIdAndBank(user.ClientID, bankCode, consent_id, "team081", []string{"ReadAccountsDetail", "ReadBalances"}, status, expiresAt)
	if saveErr != nil {
		log.Printf("error saving consent: %v", saveErr)
		// не скрываем ошибку от клиента: говорим 500, т.к. данные не сохранились
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save consent", "details": saveErr.Error()})
		return
	}

	// 12) Отдаём ответ клиенту
	c.JSON(http.StatusOK, gin.H{
		"status":        status,
		"consent_id":    consent_id,
		"auto_approved": auto_approved,
		"bank":          bankCode,
	})
}
