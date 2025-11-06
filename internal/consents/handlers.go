package consents

// import (
// 	"context"
// 	"encoding/json"
// 	"io"
// 	"net/http"
// 	"strings"
// 	"time"

// 	bankapi "MoneyPilot/internal/bankApi"
// 	"MoneyPilot/internal/storage"

// 	"github.com/gin-gonic/gin"
// )

// type Handler struct {
// 	Repo       *storage.Repository
// 	HTTPClient *http.Client
// }

// func NewHandler(repo *storage.Repository) *Handler {
// 	return &Handler{Repo: repo, HTTPClient: &http.Client{Timeout: 15 * time.Second}}
// }

// type ConsentRequest struct {
// 	ClientID           string   `json:"client_id" binding:"required"`
// 	Permissions        []string `json:"permissions" binding:"required"`
// 	Reason             string   `json:"reason"`
// 	RequestingBank     string   `json:"requesting_bank"`
// 	RequestingBankName string   `json:"requesting_bank_name"`
// }

// // RequestConsent proxies a consent request to the target bank API.
// // Expects header X-Bank-Code with the bank code (e.g. vbank)
// func (h *Handler) RequestConsent(c *gin.Context) {
// 	var body ConsentRequest
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
// 		return
// 	}

// 	bankCode := c.GetHeader("X-Bank-Code")
// 	if bankCode == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Bank-Code header required"})
// 		return
// 	}

// 	bankClient, ok := bankapi.Banks[bankCode]
// 	if !ok {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown bank code"})
// 		return
// 	}

// 	// Use existing GetToken implementation
// 	tokenObj, err := bankClient.GetToken()
// 	if err != nil {
// 		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to obtain bank token", "details": err.Error()})
// 		return
// 	}

// 	token := tokenObj.Token

// 	target := strings.TrimRight(bankClient.BaseURL, "/") + "/account-consents/request"
// 	payload := map[string]interface{}{
// 		"client_id":            body.ClientID,
// 		"permissions":          body.Permissions,
// 		"reason":               body.Reason,
// 		"requesting_bank":      body.RequestingBank,
// 		"requesting_bank_name": body.RequestingBankName,
// 	}
// 	b, err := json.Marshal(payload)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal payload", "details": err.Error()})
// 		return
// 	}

// 	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, target, strings.NewReader(string(b)))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request", "details": err.Error()})
// 		return
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	if body.RequestingBank != "" {
// 		req.Header.Set("X-Requesting-Bank", body.RequestingBank)
// 	}

// 	resp, err := h.HTTPClient.Do(req)
// 	if err != nil {
// 		c.JSON(http.StatusBadGateway, gin.H{"error": "bank request failed", "details": err.Error()})
// 		return
// 	}
// 	defer resp.Body.Close()

// 	respBody, _ := io.ReadAll(resp.Body)
// 	contentType := resp.Header.Get("Content-Type")
// 	if contentType == "" {
// 		contentType = "application/json"
// 	}
// 	c.Data(resp.StatusCode, contentType, respBody)
// }

// // CreateConsent ensures the caller is authenticated (user JWT) and then proxies the consent request to the bank.
// // It expects Authorization header (user JWT) to be decoded by middleware which sets "user_id" in context.
// func (h *Handler) CreateConsent(c *gin.Context) {
// 	// check user_id set by auth middleware
// 	if _, ok := c.Get("user_id"); !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
// 		return
// 	}

// 	// reuse RequestConsent which performs validation and forwarding
// 	h.RequestConsent(c)
// }
