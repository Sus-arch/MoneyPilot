package api

import (
    "database/sql"
    "time"

    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"

    "MoneyPilot/internal/accountconsents"
    "MoneyPilot/internal/auth"
    bankapi "MoneyPilot/internal/bankapi"
    "MoneyPilot/internal/storage"
)

func NewRouter(db *sql.DB, jwtSecret string, rdb *redis.Client) *gin.Engine {
    r := gin.Default()

    // ✅ Правильный CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:5173"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Bank-Code"},
        ExposeHeaders:    []string{"Content-Length", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    authService := auth.NewAuthService(db, jwtSecret)
    authHandler := auth.NewHandler(authService)

    // 🔐 Эндпоинты авторизации
    r.POST("/api/auth/login", authHandler.Login)

    apiGroup := r.Group("/api")

    // 🔒 Защищённые маршруты
    secured := apiGroup.Group("")
    secured.Use(auth.DecodeToken([]byte(jwtSecret)))

    repo := storage.NewRepository(db)
    ts := bankapi.NewTokenService(rdb)
    consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
    secured.POST("/account-consent", consentHandler.CreateConsent)

    return r
}
