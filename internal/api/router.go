package api

import (
	"database/sql"

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

	// Разрешаем CORS для фронта
	r.Use(cors.Default())

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)

	// Эндпоинты авторизации
	r.POST("/api/auth/login", authHandler.Login)

	apiGroup := r.Group("/api")

	// secured routes require user JWT
	secured := apiGroup.Group("")
	secured.Use(auth.DecodeToken([]byte(jwtSecret)))

	// consents handler uses storage repository and proxies requests to banks
	repo := storage.NewRepository(db)
	ts := bankapi.NewTokenService(rdb)
	consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
	secured.POST("/account-consent", consentHandler.CreateConsent)
	return r
}
