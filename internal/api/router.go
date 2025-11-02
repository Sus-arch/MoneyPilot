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
	r.Use(cors.Default())

	repo := storage.NewRepository(db)
	ts := bankapi.NewTokenService(rdb)

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)
	r.POST("/api/auth/login", authHandler.Login)

	apiGroup := r.Group("/api")
	secured := apiGroup.Group("")
	secured.Use(auth.DecodeToken([]byte(jwtSecret)))

	// --- handlers ---
	consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
	accountService := accounts.NewService(repo, ts, bankapi.Banks)
	accountHandler := accounts.NewHandler(accountService)
	stopCh := make(chan struct{})

	// запускаем poller с интервалом 5 секунд
	consentHandler.StartPoller(5*time.Second, stopCh)
	// --- routes ---
	secured.POST("/account-consent", consentHandler.CreateConsent)
	secured.GET("/accounts", accountHandler.ListAccounts)

	return r
}
