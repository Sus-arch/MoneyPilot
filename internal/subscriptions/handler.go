package subscriptions

import (
	"net/http"

	"MoneyPilot/internal/storage"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo *storage.Repository
}

func NewHandler(repo *storage.Repository) *Handler {
	return &Handler{Repo: repo}
}

// CreateSubscription создает подписку для текущего пользователя
func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id type"})
		return
	}

	// Получаем user для получения client_id
	user, err := h.Repo.GetUserByID(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	err = h.Repo.CreateSubscription(user.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription created successfully"})
}

// GetSubscriptionStatus получает статус подписки текущего пользователя
func (h *Handler) GetSubscriptionStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	userIDInt, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user_id type"})
		return
	}

	// Получаем user для получения client_id
	user, err := h.Repo.GetUserByID(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	hasActive, err := h.Repo.HasActiveSubscription(user.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_subscribed": hasActive})
}
