package payments

import "github.com/gin-gonic/gin"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreatePayment(c *gin.Context) {
	h.service.CreatePayment(c)
}

func (h *Handler) GetPaymentStatus(c *gin.Context) {
	h.service.GetPaymentStatus(c)
}

func (h *Handler) CheckPaymentConsents(c *gin.Context) {
	h.service.CheckPaymentConsents(c)
}
