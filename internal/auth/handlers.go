package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *AuthService
}

func NewHandler(s *AuthService) *Handler {
	return &Handler{Service: s}
}

func (h *Handler) Login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Bank     string `json:"bank"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, err := h.Service.Authenticate(body.Email, body.Password, body.Bank)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
