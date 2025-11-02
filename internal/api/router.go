package api

import (
	"database/sql"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"MoneyPilot/internal/accountconsents"
	"MoneyPilot/internal/accounts"
	"MoneyPilot/internal/auth"
	bankapi "MoneyPilot/internal/bankapi"
	"MoneyPilot/internal/storage"
)

func NewRouter(db *sql.DB, jwtSecret string, rdb *redis.Client) *gin.Engine {
	r := gin.Default()

	// --- Настройка CORS ---
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Bank-Code"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- Сервисы и хендлеры ---
	repo := storage.NewRepository(db)
	ts := bankapi.NewTokenService(rdb)

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)
	r.POST("/api/auth/login", authHandler.Login)

	apiGroup := r.Group("/api")
	secured := apiGroup.Group("")
	secured.Use(auth.DecodeToken([]byte(jwtSecret)))

	consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
	accountService := accounts.NewService(repo, ts, bankapi.Banks)
	accountHandler := accounts.NewHandler(accountService)

	// --- Запускаем poller согласий ---
	stopCh := make(chan struct{})
	consentHandler.StartPoller(5*time.Second, stopCh)

	// --- Маршруты ---
	secured.POST("/account-consent", consentHandler.CreateConsent)
	secured.GET("/accounts", accountHandler.ListAccounts)
	secured.GET("/accounts/:account_id/balances", accountHandler.GetAccountBalance)
	secured.GET("/accounts/:account_id/transactions", accountHandler.GetAccountTransactions)
	secured.GET("/accounts/:account_id/details", accountHandler.GetAccountDetails)

	return r
}
