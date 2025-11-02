package accounts

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler оборачивает сервис для работы с аккаунтами
type Handler struct {
	service *Service
}

// NewHandler конструктор
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListAccounts — эндпоинт GET /api/accounts
// Требует JWT (middleware добавляет user_id в контекст)
func (h *Handler) ListAccounts(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accounts, err := h.service.FetchAllUserAccounts(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch accounts", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    len(accounts),
		"accounts": accounts,
	})
}

func (h *Handler) GetAccountBalance(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID := c.Param("account_id")
	bankCode := c.GetHeader("X-Bank-Code")
	if bankCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Bank-Code header required"})
		return
	}

	data, err := h.service.FetchAccountBalance(userID, bankCode, accountID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch balance", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetAccountTransactions(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID := c.Param("account_id")
	bankCode := c.GetHeader("X-Bank-Code")
	from := c.Query("from_booking_date_time")
	to := c.Query("to_booking_date_time")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")

	data, err := h.service.FetchAccountTransactions(userID, bankCode, accountID, from, to, page, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch transactions", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetAccountDetails(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID := c.Param("account_id")
	bankCode := c.GetHeader("X-Bank-Code")

	data, err := h.service.FetchAccountDetails(userID, bankCode, accountID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch account details", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
