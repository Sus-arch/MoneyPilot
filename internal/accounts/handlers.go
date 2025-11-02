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
