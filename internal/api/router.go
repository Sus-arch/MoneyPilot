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
	"MoneyPilot/internal/cache"
	"MoneyPilot/internal/paymentconsents"
	"MoneyPilot/internal/payments"
	"MoneyPilot/internal/poller"
	"MoneyPilot/internal/productagreements"
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
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Bank-Code", "X-Payment-Consent-Id", "X-Fapi-Interaction-Id", "X-Fapi-Customer-Ip-Address"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- Инициализация зависимостей ---
	repo := storage.NewRepository(db)
	ts := bankapi.NewTokenService(rdb)
	cacheService := cache.NewCacheService(db)

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := auth.NewHandler(authService)
	r.POST("/api/auth/login", authHandler.Login)

	apiGroup := r.Group("/api")
	secured := apiGroup.Group("")
	secured.Use(auth.DecodeToken([]byte(jwtSecret)))

	// --- Handlers ---
	consentHandler := accountconsents.NewConsentHandler(repo, ts, bankapi.Banks)
	accountService := accounts.NewService(repo, ts, bankapi.Banks, cacheService)
	accountHandler := accounts.NewHandler(accountService)

	productConsentsService := productconsents.NewService(repo, ts, bankapi.Banks)
	productConsentsHandler := productconsents.NewHandler(productConsentsService)

	productAgreementService := productagreements.NewService(repo, ts, bankapi.Banks, cacheService)
	productAgreementHandler := productagreements.NewHandler(productAgreementService)

	paymentConsentsService := paymentconsents.NewService(repo, ts, bankapi.Banks)
	paymentConsentsHandler := paymentconsents.NewHandler(paymentConsentsService)

	paymentsService := payments.NewService(repo, ts, bankapi.Banks)
	paymentsHandler := payments.NewHandler(paymentsService)

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
	PaymentRepo := poller.PaymentConsentRepoAdapter{
		Repo: repo,
	}

	// --- Poller ---
	pl := poller.NewPoller(
		[]poller.ConsentRepo{&AccountRepo, &ProductRepo, &PaymentRepo},
		ts,
		bankapi.Banks,
		wsHub, // уведомления через WebSocket
	)
	stopCh := make(chan struct{})
	pl.Start(5*time.Second, stopCh)

	// --- Маршруты ---
	secured.POST("/account-consent", consentHandler.CreateConsent)

	// 👇 Добавляем маршруты для product consents
	secured.POST("/product-consents/request", productConsentsHandler.CreateConsent)
	// secured.GET("/product-consents", productHandler.ListConsents) // опционально
	// secured.GET("/product-consents/:consent_id", productHandler.GetConsentStatus) // опционально

	// 👇 Добавляем маршруты для payment consents
	secured.POST("/payment-consents/request", paymentConsentsHandler.CreatePaymentConsent)

	// 👇 Добавляем маршруты для payments
	secured.POST("/payments/check-consents", paymentsHandler.CheckPaymentConsents)
	secured.POST("/payments", paymentsHandler.CreatePayment)
	secured.GET("/payments/:payment_id", paymentsHandler.GetPaymentStatus)

	// --- Account-related endpoints ---
	secured.GET("/accounts", accountHandler.ListAccounts)
	secured.GET("/accounts/:account_id/balances", accountHandler.GetAccountBalance)
	secured.GET("/accounts/:account_id/transactions", accountHandler.GetAccountTransactions)
	secured.GET("/accounts/:account_id/details", accountHandler.GetAccountDetails)

	secured.GET("/products", productAgreementHandler.ListProducts)
	secured.GET("/products/catalog", productAgreementHandler.GetProductsCatalog) // Каталог продуктов банков
	secured.GET("/products/:agreement_id", productAgreementHandler.GetProductDetails)
	secured.DELETE("/products/:agreement_id", productAgreementHandler.DeleteProduct)
	return r
}
