package accounts

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// type Handler struct {
// 	Service *Service
// }

// func NewHandler(s *Service) *Handler {
// 	return &Handler{Service: s}
// }

// func (h *Handler) GetAccounts(c *gin.Context) {
// 	accs, err := h.Service.GetAllAccounts()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load accounts"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"accounts": accs})
// }
