package api

import (
    "database/sql"

    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"

    "MoneyPilot/internal/auth"
)

func NewRouter(db *sql.DB, jwtSecret string) *gin.Engine {
    r := gin.Default()

    // Разрешаем CORS для фронта
    r.Use(cors.Default())

    authService := auth.NewAuthService(db, jwtSecret)
    authHandler := auth.NewHandler(authService)

    // Эндпоинты авторизации
    r.POST("/api/auth/login", authHandler.Login)

    return r
}
