package productagreements

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{Service: s}
}

func (h *Handler) ListProducts(c *gin.Context) {
	userID := c.GetInt("user_id")
	bankCode := c.GetHeader("X-Bank-Code")
	if bankCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Bank-Code header required"})
		return
	}

	products, err := h.Service.GetProducts(userID, bankCode)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": products})
}

func (h *Handler) GetProductDetails(c *gin.Context) {
	userID := c.GetInt("user_id")
	bankCode := c.GetHeader("X-Bank-Code")
	agreementID := c.Param("agreement_id")

	product, err := h.Service.GetProductDetails(userID, bankCode, agreementID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	userID := c.GetInt("user_id")
	bankCode := c.GetHeader("X-Bank-Code")
	agreementID := c.Param("agreement_id")

	var payload map[string]interface{}
	c.ShouldBindJSON(&payload)

	err := h.Service.DeleteProduct(userID, bankCode, agreementID, payload)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
