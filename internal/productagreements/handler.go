package productagreements

import (
	"net/http"
	"strings"

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

// GetProductsCatalog — эндпоинт GET /api/products/catalog
// Получает каталог продуктов из всех указанных банков
// Опционально принимает заголовок X-Bank-Code со списком банков через запятую
// Опционально принимает query параметр product_type для фильтрации (deposit, loan, card, account)
func (h *Handler) GetProductsCatalog(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Получаем список банков из заголовка X-Bank-Code
	bankCodesHeader := c.GetHeader("X-Bank-Code")
	var bankCodes []string
	if bankCodesHeader != "" {
		// Разделяем по запятой и очищаем от пробелов
		codes := strings.Split(bankCodesHeader, ",")
		for _, code := range codes {
			trimmed := strings.TrimSpace(code)
			if trimmed != "" {
				bankCodes = append(bankCodes, trimmed)
			}
		}
	}

	// Получаем тип продукта из query параметра (опционально)
	productType := c.Query("product_type")

	result, err := h.Service.GetProductsCatalog(userID, bankCodes, productType)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch products catalog", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
