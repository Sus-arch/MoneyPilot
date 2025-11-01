package api

import (
	"MoneyPilot/internal/auth"
	"database/sql"

	"github.com/gin-gonic/gin"
)

func NewRouter(db *sql.DB, jwtSecret string) *gin.Engine {
	r := gin.Default()

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)
	r.POST("/api/auth/login", authHandler.Login)

	return r
}
