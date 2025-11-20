package paymentconsents

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

type Service struct {
	Repo        *storage.Repository
	TokenSvc    *bankapi.TokenService
	BankClients map[string]*bankapi.BankClient
	HTTPWrapper *bankapi.ClientWrapper
}

func NewService(repo *storage.Repository, ts *bankapi.TokenService, clients map[string]*bankapi.BankClient) *Service {
	config := bankapi.DefaultConfig()
	httpWrapper := bankapi.NewClientWrapper(nil, config) // nil cacheService, так как кэш не нужен

	return &Service{
		Repo:        repo,
		TokenSvc:    ts,
		BankClients: clients,
		HTTPWrapper: httpWrapper,
	}
}

type CreatePaymentConsentRequest struct {
	ConsentType             string     `json:"consent_type" binding:"required"`
	Amount                  *float64   `json:"amount"`
	Currency                *string    `json:"currency"`
	DebtorAccount           string     `json:"debtor_account" binding:"required"`
	CreditorAccount         *string    `json:"creditor_account"`
	CreditorName            *string    `json:"creditor_name"`
	Reference               *string    `json:"reference"`
	MaxUses                 *int       `json:"max_uses"`
	MaxAmountPerPayment     *float64   `json:"max_amount_per_payment"`
	MaxTotalAmount          *float64   `json:"max_total_amount"`
	AllowedCreditorAccounts []string   `json:"allowed_creditor_accounts"`
	VRPMaxIndividualAmount  *float64   `json:"vrp_max_individual_amount"`
	VRPDailyLimit           *float64   `json:"vrp_daily_limit"`
	VRPMonthlyLimit         *float64   `json:"vrp_monthly_limit"`
	ValidFrom               *time.Time `json:"valid_from"`
	ValidUntil              *time.Time `json:"valid_until"`
	Reason                  *string    `json:"reason"`
}

// POST /api/payment-consents/request
// Requires Authorization: Bearer <user-jwt> (middleware must set user_id)
// Requires header X-Bank-Code: <vbank|sbank|abank>
func (s *Service) CreatePaymentConsent(c *gin.Context) {
	// 1) авторизация: берем user_id из контекста
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 2) парсим тело запроса
	var req CreatePaymentConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// 3) выбираем банк по заголовку
	bankCode := c.GetHeader("X-Bank-Code")
	if bankCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Bank-Code header required"})
		return
	}
	bankClient, ok := s.BankClients[bankCode]
	if !ok || bankClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown bank code"})
		return
	}

	// 4) загружаем пользователя
	user, err := s.Repo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user", "details": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user record missing"})
		return
	}

	// 5) проверяем — нет ли уже действующего consent для этого user+bank+consent_type
	existing, err := s.Repo.GetValidPaymentConsentsByUserIDAndBank(userID, bankCode, req.ConsentType)
	if err == nil && len(existing) > 0 {
		// Проверяем совместимость существующего согласия
		// Для single_use проверяем debtor_account, creditor_account, amount
		// Для multi_use и vrp проверяем debtor_account
		compatible := false
		for _, consent := range existing {
			if consent.DebtorAccount == req.DebtorAccount {
				if req.ConsentType == "single_use" {
					// Для single_use проверяем creditor_account и amount
					if req.CreditorAccount != nil && consent.CreditorAccount != nil &&
						*req.CreditorAccount == *consent.CreditorAccount {
						if req.Amount != nil && consent.Amount != nil &&
							*req.Amount == *consent.Amount {
							compatible = true
							break
						}
					}
				} else {
					// Для multi_use и vrp достаточно debtor_account
					compatible = true
					break
				}
			}
		}

		if compatible {
			// Активное согласие уже существует - возвращаем его
			consent := existing[0]
			var consentIDValue interface{}
			if consent.ConsentID != nil && *consent.ConsentID != "" {
				consentIDValue = *consent.ConsentID
			} else {
				consentIDValue = nil
			}
			c.JSON(http.StatusOK, gin.H{
				"message":     "active consent exists",
				"request_id":  consent.RequestID,
				"consent_id":  consentIDValue,
				"status":      consent.Status,
				"valid_until": consent.ValidUntil,
			})
			return
		}
	}

	// 6) Проверка TokenSvc
	if s.TokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token service not configured"})
		return
	}

	// 7) Получаем валидный токен банка через TokenService
	tokenObj, err := s.TokenSvc.GetValidToken(bankClient)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to obtain bank token", "details": err.Error()})
		return
	}
	if tokenObj == nil || tokenObj.Token == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "bank token empty"})
		return
	}
	bankToken := tokenObj.Token

	// 8) Формируем payload для запроса в банк
	payload := map[string]interface{}{
		"requesting_bank": "team081",
		"client_id":       user.ClientID,
		"consent_type":    req.ConsentType,
		"debtor_account":  req.DebtorAccount,
	}

	if req.Amount != nil {
		payload["amount"] = *req.Amount
	}
	if req.Currency != nil {
		payload["currency"] = *req.Currency
	} else {
		payload["currency"] = "RUB" // значение по умолчанию
	}
	if req.CreditorAccount != nil {
		payload["creditor_account"] = *req.CreditorAccount
	}
	if req.CreditorName != nil {
		payload["creditor_name"] = *req.CreditorName
	}
	if req.Reference != nil {
		payload["reference"] = *req.Reference
	}
	if req.MaxUses != nil {
		payload["max_uses"] = *req.MaxUses
	}
	if req.MaxAmountPerPayment != nil {
		payload["max_amount_per_payment"] = *req.MaxAmountPerPayment
	}
	if req.MaxTotalAmount != nil {
		payload["max_total_amount"] = *req.MaxTotalAmount
	}
	if len(req.AllowedCreditorAccounts) > 0 {
		payload["allowed_creditor_accounts"] = req.AllowedCreditorAccounts
	}
	if req.VRPMaxIndividualAmount != nil {
		payload["vrp_max_individual_amount"] = *req.VRPMaxIndividualAmount
	}
	if req.VRPDailyLimit != nil {
		payload["vrp_daily_limit"] = *req.VRPDailyLimit
	}
	if req.VRPMonthlyLimit != nil {
		payload["vrp_monthly_limit"] = *req.VRPMonthlyLimit
	}
	if req.ValidFrom != nil {
		payload["valid_from"] = req.ValidFrom.Format(time.RFC3339)
	}
	if req.ValidUntil != nil {
		payload["valid_until"] = req.ValidUntil.Format(time.RFC3339)
	}
	if req.Reason != nil {
		payload["reason"] = *req.Reason
	}

	bodyBytes, _ := json.Marshal(payload)

	// 9) Отправляем запрос в банк
	target := strings.TrimRight(bankClient.BaseURL, "/") + "/payment-consents/request"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, target, strings.NewReader(string(bodyBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request", "details": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+bankToken)
	httpReq.Header.Set("X-Requesting-Bank", "team081")

	// Используем обертку с retry, rate limiting и circuit breaker (без кэширования)
	resp, err := s.HTTPWrapper.Do(c.Request.Context(), bankCode, httpReq, "", false)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "bank request failed", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 10) Если банк вернул ошибку — проксируем её клиенту
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return
	}

	// 11) Парсим ответ банка
	bankResp := make(map[string]json.RawMessage)
	var requestID string
	var status string
	var autoApproved bool
	var validUntil *time.Time
	var message string

	if err := json.Unmarshal(respBody, &bankResp); err != nil {
		log.Printf("warning: can't parse bank response: %v body=%s", err, string(respBody))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unmarshal failed", "details": err.Error()})
		return
	}

	// Parse top-level fields
	json.Unmarshal(bankResp["request_id"], &requestID)
	json.Unmarshal(bankResp["requestId"], &requestID)
	var consentID string
	json.Unmarshal(bankResp["consent_id"], &consentID)
	json.Unmarshal(bankResp["consentId"], &consentID)
	json.Unmarshal(bankResp["status"], &status)
	json.Unmarshal(bankResp["auto_approved"], &autoApproved)
	json.Unmarshal(bankResp["autoApproved"], &autoApproved)
	json.Unmarshal(bankResp["message"], &message)

	var validUntilStr string
	if json.Unmarshal(bankResp["valid_until"], &validUntilStr) == nil && validUntilStr != "" {
		if parsed, err := time.Parse(time.RFC3339, validUntilStr); err == nil {
			validUntil = &parsed
		}
	}

	// Decide which id to persist: if status indicates approved -> use consentID (final), else use requestID
	normalized := strings.ToLower(status)
	idToSave := ""
	if consentID != "" && (strings.Contains(normalized, "approved") || strings.Contains(normalized, "authorised") || strings.Contains(normalized, "authorized")) {
		idToSave = consentID
	} else if requestID != "" {
		idToSave = requestID
	} else if consentID != "" {
		idToSave = consentID
	}

	// 12) Сохраняем согласие в БД
	saveErr := s.Repo.SavePaymentConsentByClientIdAndBank(
		user.ClientID, bankCode, requestID, idToSave, "team081", req.ConsentType,
		getStringValue(req.Currency, "RUB"), req.DebtorAccount,
		req.CreditorAccount, req.CreditorName, req.Reference,
		req.Amount, req.MaxAmountPerPayment, req.MaxTotalAmount,
		req.VRPMaxIndividualAmount, req.VRPDailyLimit, req.VRPMonthlyLimit,
		req.MaxUses, req.AllowedCreditorAccounts,
		req.ValidFrom, validUntil, req.Reason, status,
	)
	if saveErr != nil {
		log.Printf("error saving payment consent: %v", saveErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save consent", "details": saveErr.Error()})
		return
	}

	// 13) Отдаём ответ клиенту
	response := gin.H{
		"request_id":    requestID,
		"status":        status,
		"consent_type":  req.ConsentType,
		"auto_approved": autoApproved,
		"bank":          bankCode,
	}
	if consentID != "" {
		response["consent_id"] = consentID
	}
	if validUntil != nil {
		response["valid_until"] = validUntil.Format(time.RFC3339)
	}
	if message != "" {
		response["message"] = message
	}

	c.JSON(http.StatusOK, response)
}

func getStringValue(ptr *string, defaultValue string) string {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}
