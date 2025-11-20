package payments

import (
	"MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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
	httpWrapper := bankapi.NewClientWrapper(nil, config) // nil cacheService, так как кэш не нужен для платежей

	return &Service{
		Repo:        repo,
		TokenSvc:    ts,
		BankClients: clients,
		HTTPWrapper: httpWrapper,
	}
}

type CreatePaymentRequest struct {
	Data struct {
		Initiation struct {
			InstructedAmount struct {
				Amount   string `json:"amount" binding:"required"`
				Currency string `json:"currency" binding:"required"`
			} `json:"instructedAmount" binding:"required"`
			DebtorAccount struct {
				SchemeName     string `json:"schemeName" binding:"required"`
				Identification string `json:"identification" binding:"required"`
			} `json:"debtorAccount" binding:"required"`
			CreditorAccount struct {
				SchemeName     string  `json:"schemeName"`
				Identification string  `json:"identification" binding:"required"`
				BankCode       *string `json:"bank_code"` // для межбанковских переводов
			} `json:"creditorAccount" binding:"required"`
			Comment *string `json:"comment"`
		} `json:"initiation" binding:"required"`
	} `json:"data" binding:"required"`
	Risk map[string]interface{} `json:"risk"` // опциональное поле для рисковых данных
}

// POST /api/payments
// Requires Authorization: Bearer <user-jwt> (middleware must set user_id)
// Requires header X-Bank-Code: <vbank|sbank|abank>
// Optional headers: X-Payment-Consent-Id, X-Fapi-Interaction-Id, X-Fapi-Customer-Ip-Address
func (s *Service) CreatePayment(c *gin.Context) {
	// 1) авторизация: берем user_id из контекста
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 2) парсим тело запроса
	var req CreatePaymentRequest
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

	// 5) Получаем правильного пользователя для данного банка
	bank, err := s.Repo.GetBankByCode(bankCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load bank", "details": err.Error()})
		return
	}
	bankUser, err := s.Repo.GetUserByClientIDAndBankID(user.ClientID, bank.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user for bank", "details": err.Error()})
		return
	}

	// 6) Проверяем согласие на платеж, если указано в заголовке
	paymentConsentID := c.GetHeader("X-Payment-Consent-Id")
	var paymentConsent *storage.PaymentConsent
	if paymentConsentID != "" {
		consent, err := s.Repo.GetPaymentConsentByConsentID(paymentConsentID)
		if err == nil && consent != nil {
			paymentConsent = consent
			// Проверяем, что согласие действительно для этого пользователя и банка
			if consent.UserID != bankUser.ID || consent.BankID != bank.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "payment consent does not belong to this user"})
				return
			}
			// Проверяем, что согласие активно
			if consent.Status != "approved" {
				c.JSON(http.StatusForbidden, gin.H{"error": "payment consent is not approved"})
				return
			}
			// Проверяем валидность согласия
			if consent.ValidUntil != nil && consent.ValidUntil.Before(time.Now()) {
				c.JSON(http.StatusForbidden, gin.H{"error": "payment consent has expired"})
				return
			}
		}
	}

	// 7) Парсим сумму
	amount, err := strconv.ParseFloat(req.Data.Initiation.InstructedAmount.Amount, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount format", "details": err.Error()})
		return
	}

	currency := req.Data.Initiation.InstructedAmount.Currency
	if currency == "" {
		currency = "RUB" // значение по умолчанию
	}

	debtorAccount := req.Data.Initiation.DebtorAccount.Identification
	creditorAccount := req.Data.Initiation.CreditorAccount.Identification
	creditorBankCode := req.Data.Initiation.CreditorAccount.BankCode

	// 8) Проверяем согласие на платеж (если указано)
	if paymentConsent != nil {
		// Проверяем, что счет списания совпадает
		if paymentConsent.DebtorAccount != debtorAccount {
			c.JSON(http.StatusForbidden, gin.H{"error": "debtor account does not match payment consent"})
			return
		}
		// Для single_use проверяем creditor_account и amount
		if paymentConsent.ConsentType == "single_use" {
			if paymentConsent.CreditorAccount != nil && *paymentConsent.CreditorAccount != creditorAccount {
				c.JSON(http.StatusForbidden, gin.H{"error": "creditor account does not match payment consent"})
				return
			}
			if paymentConsent.Amount != nil && *paymentConsent.Amount != amount {
				c.JSON(http.StatusForbidden, gin.H{"error": "amount does not match payment consent"})
				return
			}
		}
		// Для multi_use проверяем лимиты
		if paymentConsent.ConsentType == "multi_use" {
			if paymentConsent.MaxAmountPerPayment != nil && *paymentConsent.MaxAmountPerPayment < amount {
				c.JSON(http.StatusForbidden, gin.H{"error": "amount exceeds max_amount_per_payment limit"})
				return
			}
			if len(paymentConsent.AllowedCreditorAccounts) > 0 {
				allowed := false
				for _, acc := range paymentConsent.AllowedCreditorAccounts {
					if acc == creditorAccount {
						allowed = true
						break
					}
				}
				if !allowed {
					c.JSON(http.StatusForbidden, gin.H{"error": "creditor account is not in allowed list"})
					return
				}
			}
		}
		// Для vrp проверяем лимиты
		if paymentConsent.ConsentType == "vrp" {
			if paymentConsent.VRPMaxIndividualAmount != nil && *paymentConsent.VRPMaxIndividualAmount < amount {
				c.JSON(http.StatusForbidden, gin.H{"error": "amount exceeds vrp_max_individual_amount limit"})
				return
			}
		}
	}

	// 9) Проверка TokenSvc
	if s.TokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token service not configured"})
		return
	}

	// 10) Получаем валидный токен банка через TokenService
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

	// 11) Формируем payload для запроса в банк
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"initiation": map[string]interface{}{
				"instructedAmount": map[string]interface{}{
					"amount":   req.Data.Initiation.InstructedAmount.Amount,
					"currency": currency,
				},
				"debtorAccount": map[string]interface{}{
					"schemeName":     req.Data.Initiation.DebtorAccount.SchemeName,
					"identification": debtorAccount,
				},
				"creditorAccount": map[string]interface{}{
					"schemeName":     req.Data.Initiation.CreditorAccount.SchemeName,
					"identification": creditorAccount,
				},
			},
		},
	}

	// Добавляем bank_code для межбанковских переводов
	if creditorBankCode != nil && *creditorBankCode != "" {
		creditorAcc := payload["data"].(map[string]interface{})["initiation"].(map[string]interface{})["creditorAccount"].(map[string]interface{})
		creditorAcc["bank_code"] = *creditorBankCode
	}

	// Добавляем comment, если указан
	if req.Data.Initiation.Comment != nil && *req.Data.Initiation.Comment != "" {
		initiation := payload["data"].(map[string]interface{})["initiation"].(map[string]interface{})
		initiation["comment"] = *req.Data.Initiation.Comment
	}

	// Добавляем risk, если указан
	if req.Risk != nil && len(req.Risk) > 0 {
		payload["risk"] = req.Risk
	}

	bodyBytes, _ := json.Marshal(payload)

	// 12) Отправляем запрос в банк
	requestingBank := os.Getenv("LOGIN_HAC")
	if requestingBank == "" {
		requestingBank = "team081"
	}

	target := strings.TrimRight(bankClient.BaseURL, "/") + "/payments?client_id=" + bankUser.ClientID
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, target, strings.NewReader(string(bodyBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request", "details": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+bankToken)
	httpReq.Header.Set("X-Requesting-Bank", requestingBank)

	// Опциональные заголовки
	if interactionID := c.GetHeader("X-Fapi-Interaction-Id"); interactionID != "" {
		httpReq.Header.Set("X-Fapi-Interaction-Id", interactionID)
	}
	if customerIP := c.GetHeader("X-Fapi-Customer-Ip-Address"); customerIP != "" {
		httpReq.Header.Set("X-Fapi-Customer-Ip-Address", customerIP)
	}
	if paymentConsentID != "" {
		httpReq.Header.Set("X-Payment-Consent-Id", paymentConsentID)
	}

	// Используем обертку с retry, rate limiting и circuit breaker (без кэширования)
	resp, err := s.HTTPWrapper.Do(c.Request.Context(), bankCode, httpReq, "", false)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "bank request failed", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// 13) Если банк вернул ошибку — проксируем её клиенту
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return
	}

	// 14) Парсим ответ банка
	var bankResp map[string]interface{}
	if err := json.Unmarshal(respBody, &bankResp); err != nil {
		log.Printf("warning: can't parse bank response: %v body=%s", err, string(respBody))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unmarshal failed", "details": err.Error()})
		return
	}

	// Извлекаем payment_id и статус из ответа
	var paymentID string
	var status string
	var description string
	var creationDateTime, statusUpdateDateTime string

	if data, ok := bankResp["data"].(map[string]interface{}); ok {
		if id, ok := data["paymentId"].(string); ok {
			paymentID = id
		}
		if s, ok := data["status"].(string); ok {
			status = s
		}
		if d, ok := data["description"].(string); ok {
			description = d
		}
		if cdt, ok := data["creationDateTime"].(string); ok {
			creationDateTime = cdt
		}
		if sudt, ok := data["statusUpdateDateTime"].(string); ok {
			statusUpdateDateTime = sudt
		}
	}

	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id not found in response"})
		return
	}

	// 15) Сохраняем платеж в БД
	var paymentConsentIDPtr *string
	if paymentConsentID != "" {
		paymentConsentIDPtr = &paymentConsentID
	}
	var commentPtr *string
	if req.Data.Initiation.Comment != nil {
		commentPtr = req.Data.Initiation.Comment
	}
	var descriptionPtr *string
	if description != "" {
		descriptionPtr = &description
	}

	saveErr := s.Repo.SavePaymentByClientIdAndBank(
		bankUser.ClientID, bankCode, paymentID, debtorAccount, creditorAccount,
		creditorBankCode, amount, currency, commentPtr, descriptionPtr,
		paymentConsentIDPtr, status,
	)
	if saveErr != nil {
		log.Printf("error saving payment: %v", saveErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save payment", "details": saveErr.Error()})
		return
	}

	// 16) Отдаём ответ клиенту
	response := gin.H{
		"data": gin.H{
			"paymentId": paymentID,
			"status":    status,
			"amount":    req.Data.Initiation.InstructedAmount.Amount,
			"currency":  currency,
		},
	}
	if description != "" {
		response["data"].(gin.H)["description"] = description
	}
	if creationDateTime != "" {
		response["data"].(gin.H)["creationDateTime"] = creationDateTime
	}
	if statusUpdateDateTime != "" {
		response["data"].(gin.H)["statusUpdateDateTime"] = statusUpdateDateTime
	}

	c.JSON(http.StatusCreated, response)
}

// GetPaymentStatus получает статус платежа
func (s *Service) GetPaymentStatus(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id required"})
		return
	}

	// Получаем платеж из БД
	payment, err := s.Repo.GetPaymentByPaymentID(paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	// Проверяем, что платеж принадлежит пользователю
	user, err := s.Repo.GetUserByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	bankUser, err := s.Repo.GetUserByClientIDAndBankID(user.ClientID, payment.BankID)
	if err != nil || bankUser.ID != payment.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "payment does not belong to this user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"paymentId":            payment.PaymentID,
			"status":               payment.Status,
			"amount":               fmt.Sprintf("%.2f", payment.Amount),
			"currency":             payment.Currency,
			"creationDateTime":     payment.CreatedAt.Format(time.RFC3339),
			"statusUpdateDateTime": payment.UpdatedAt.Format(time.RFC3339),
		},
	})
}
