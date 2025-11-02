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
		"permissions":          []string{"ReadAccountsDetail", "ReadBalances", "ReadTransactionsDetail"},
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

	// 10) Парсим ответ — банки возвращают разные формы (top-level consent_id/status OR data.consentId/data.status)
	bankResp := make(map[string]json.RawMessage)
	var consentID string
	var requestID string
	var status string
	var autoApproved bool

	if err := json.Unmarshal(respBody, &bankResp); err != nil {
		log.Printf("warning: can't parse bank response: %v body=%s", err, string(respBody))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unmarshal failed", "details": err.Error()})
		return
	}

	// Try top-level fields
	json.Unmarshal(bankResp["consent_id"], &consentID)
	json.Unmarshal(bankResp["consentId"], &consentID)
	json.Unmarshal(bankResp["request_id"], &requestID)
	json.Unmarshal(bankResp["requestId"], &requestID)
	json.Unmarshal(bankResp["status"], &status)
	json.Unmarshal(bankResp["auto_approved"], &autoApproved)

	// If response wrapped in {"data": {...}}
	if dataRaw, ok := bankResp["data"]; ok {
		var dataMap map[string]json.RawMessage
		if err := json.Unmarshal(dataRaw, &dataMap); err == nil {
			json.Unmarshal(dataMap["consentId"], &consentID)
			json.Unmarshal(dataMap["consent_id"], &consentID)
			json.Unmarshal(dataMap["requestId"], &requestID)
			json.Unmarshal(dataMap["request_id"], &requestID)
			json.Unmarshal(dataMap["status"], &status)
			// some banks use different status strings like AwaitingAuthorization
		}
	}

	// Decide which id to persist: if status indicates approved -> use consentID (final), else use requestID
	// Normalize status checks: treat approved/authorised/authorized as approved
	normalized := strings.ToLower(status)
	idToSave := ""
	if consentID != "" && (strings.Contains(normalized, "approved") || strings.Contains(normalized, "authorised") || strings.Contains(normalized, "authorized")) {
		idToSave = consentID
	} else if requestID != "" {
		idToSave = requestID
	} else if consentID != "" {
		// fallback to consentID even if status not explicitly approved
		idToSave = consentID
	}

	// 11) Сохраняем согласие в БД
	expiresAt := time.Now().Add(90 * 24 * time.Hour)
	saveErr := h.Repo.SaveAccountConsentByClientIdAndBank(user.ClientID, bankCode, idToSave, "team081", []string{"ReadAccountsDetail", "ReadBalances"}, status, expiresAt)
	if saveErr != nil {
		log.Printf("error saving consent: %v", saveErr)
		// не скрываем ошибку от клиента: говорим 500, т.к. данные не сохранились
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save consent", "details": saveErr.Error()})
		return
	}

	// 12) Отдаём ответ клиенту
	c.JSON(http.StatusOK, gin.H{
		"status":        status,
		"consent_id":    idToSave,
		"auto_approved": autoApproved,
		"bank":          bankCode,
	})
}

// StartPoller runs PollPendingConsents periodically in a background goroutine.
// Callers should provide a stop channel which will stop the goroutine when closed.
func (h *ConsentHandler) StartPoller(interval time.Duration, stopCh <-chan struct{}) {
	log.Println("НАчало работы StartPoller")
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		log.Printf("consent poller started, interval=%s", interval)
		for {
			select {
			case <-ticker.C:
				h.PollPendingConsents()
			case <-stopCh:
				log.Println("consent poller stopping")
				return
			}
		}
	}()
}

// PollPendingConsents iterates over consents in DB with status='pending', queries the corresponding
// bank's GET /account-consents/{consent_id} endpoint and updates the DB status to 'approved' when
// the bank reports the consent as Authorised/Authorized.
// This function is safe to call periodically (e.g. from a scheduler or a background goroutine).
func (h *ConsentHandler) PollPendingConsents() {
	consents, err := h.Repo.GetPendingAccountConsents()
	if err != nil {
		log.Printf("poll: failed to load pending consents: %v", err)
		return
	}

	for _, c := range consents {
		if c.BankCode == nil || *c.BankCode == "" {
			log.Printf("poll: skipping consent %s: bank code missing", c.ConsentID)
			continue
		}

		bankClient, ok := h.BankClients[*c.BankCode]
		if !ok || bankClient == nil {
			log.Printf("poll: unknown bank code for consent %s: %s", c.ConsentID, *c.BankCode)
			continue
		}

		// Obtain bank token
		tokenObj, tErr := h.TokenSvc.GetValidToken(bankClient)
		if tErr != nil {
			log.Printf("poll: failed to get token for bank %s: %v", *c.BankCode, tErr)
			continue
		}
		if tokenObj == nil || tokenObj.Token == "" {
			log.Printf("poll: empty token for bank %s", *c.BankCode)
			continue
		}

		target := strings.TrimRight(bankClient.BaseURL, "/") + "/account-consents/" + c.ConsentID
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			log.Printf("poll: failed to build request for %s: %v", c.ConsentID, err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+tokenObj.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Fapi-Interaction-Id", "team081")
		// include interaction id header if we have requesting bank info
		// if c.RequestingBank != nil && *c.RequestingBank != "" {
		// 	req.Header.Set("X-Fapi-Interaction-Id", *c.RequestingBank)
		// } else {
		// 	req.Header.Set("X-Fapi-Interaction-Id", "team081")
		// }

		resp, err := h.HTTPClient.Do(req)
		if err != nil {
			log.Printf("poll: bank request failed for %s: %v", c.ConsentID, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("poll: bank returned non-2xx for %s: %d body=%s", c.ConsentID, resp.StatusCode, string(body))
			continue
		}

		var bankResp map[string]json.RawMessage
		if err := json.Unmarshal(body, &bankResp); err != nil {
			log.Printf("poll: unmarshal failed for %s: %v body=%s", c.ConsentID, err, string(body))
			continue
		}

		// parse possible shapes: top-level status OR data.status
		var status string
		var got bool
		if dataRaw, ok := bankResp["data"]; ok {
			var dataMap map[string]json.RawMessage
			if err := json.Unmarshal(dataRaw, &dataMap); err == nil {

				if v, ok := dataMap["status"]; ok {
					json.Unmarshal(v, &status)
					got = true
				}
				log.Println(status)
				// if bank returned a final consentId here, consider updating stored id
				var finalConsentId string
				json.Unmarshal(dataMap["consentId"], &finalConsentId)
				if finalConsentId != "" && finalConsentId != c.ConsentID {
					// update DB to use final consent id (best-effort)
					if uErr := h.Repo.UpdateAccountConsentIDAndStatus(c.ConsentID, finalConsentId, "approved"); uErr != nil {
						log.Printf("poll: failed to update DB for %s -> final %s: %v", c.ConsentID, finalConsentId, uErr)
					} else {
						log.Printf("poll: consent %s updated to approved (final id %s)", c.ConsentID, finalConsentId)
					}
					continue
				}
			}
		}

		if !got {
			log.Printf("poll: consent %s no status found in response (body=%s)", c.ConsentID, string(body))
			continue
		}

		if strings.EqualFold(status, "authorised") || strings.EqualFold(status, "authorized") {
			if uErr := h.Repo.UpdateAccountConsentStatusByConsentID(c.ConsentID, "approved"); uErr != nil {
				log.Printf("poll: failed to update DB for %s: %v", c.ConsentID, uErr)
			} else {
				log.Printf("poll: consent %s updated to approved", c.ConsentID)
			}
		} else {
			log.Printf("poll: consent %s status=%s (no update)", c.ConsentID, status)
		}
	}
}
