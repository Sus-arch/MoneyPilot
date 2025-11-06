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
	"MoneyPilot/internal/poller"
	"MoneyPilot/internal/productconsents"
	"MoneyPilot/internal/storage"
	"MoneyPilot/internal/websockets"
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

	// --- Инициализация зависимостей ---
	repo := storage.NewRepository(db)
	ts := bankapi.NewTokenService(rdb)

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)
	r.POST("/api/auth/login", authHandler.Login)

	apiGroup := r.Group("/api")
	secured := apiGroup.Group("")
	secured.Use(auth.DecodeToken([]byte(jwtSecret)))

	// --- Handlers ---
	consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
	accountService := accounts.NewService(repo, ts, bankapi.Banks)
	accountHandler := accounts.NewHandler(accountService)

	productService := productconsents.NewService(repo, ts, bankapi.Banks)
	productHandler := productconsents.NewHandler(productService)

	// --- WebSocket Hub ---
	wsHub := websockets.NewHub()
	r.GET("/ws", wsHub.HandleConnection)

	// --- Репозитории для Poller ---
	AccountRepo := poller.AccountConsentRepoAdapter{
		Repo: repo,
	}
	ProductRepo := poller.ProductConsentRepoAdapter{
		Repo: repo,
	}

	// --- Poller ---
	pl := poller.NewPoller(
		[]poller.ConsentRepo{&AccountRepo, &ProductRepo},
		ts,
		bankapi.Banks,
		wsHub, // уведомления через WebSocket
	)
	stopCh := make(chan struct{})
	pl.Start(5*time.Second, stopCh)

	// --- Маршруты ---
	secured.POST("/account-consent", consentHandler.CreateConsent)

	// 👇 Добавляем маршруты для product consents
	secured.POST("/product-consents/request", productHandler.CreateConsent)
	// secured.GET("/product-consents", productHandler.ListConsents) // опционально
	// secured.GET("/product-consents/:consent_id", productHandler.GetConsentStatus) // опционально

	// --- Account-related endpoints ---
	secured.GET("/accounts", accountHandler.ListAccounts)
	secured.GET("/accounts/:account_id/balances", accountHandler.GetAccountBalance)
	secured.GET("/accounts/:account_id/transactions", accountHandler.GetAccountTransactions)
	secured.GET("/accounts/:account_id/details", accountHandler.GetAccountDetails)

	return r
}
