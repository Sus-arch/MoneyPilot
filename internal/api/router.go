package api

import (
	"database/sql"
	"log"
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
	"MoneyPilot/internal/subscriptions"
	"MoneyPilot/internal/websockets"
)

func NewRouter(db *sql.DB, jwtSecret string, rdb *redis.Client, corsOrigins []string) *gin.Engine {
	r := gin.Default()

	// --- Настройка CORS ---
	// Если corsOrigins пустой, используем значение по умолчанию
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:5173", "http://147.45.219.12"}
	}

	// Добавляем поддержку для localhost с разными портами и IP-адреса
	defaultOrigins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
		"http://147.45.219.12",
		"http://147.45.219.12:80",
		"http://147.45.219.12:5173",
		"http://147.45.219.12:3000",
	}

	// Объединяем переданные origins с дефолтными, убираем дубликаты
	allOrigins := make(map[string]bool)
	for _, origin := range corsOrigins {
		if origin != "" {
			allOrigins[origin] = true
		}
	}
	for _, origin := range defaultOrigins {
		allOrigins[origin] = true
	}

	uniqueOrigins := make([]string, 0, len(allOrigins))
	for origin := range allOrigins {
		uniqueOrigins = append(uniqueOrigins, origin)
	}

	// Логируем разрешенные origins для отладки
	log.Printf("🌐 CORS configured with origins: %v", uniqueOrigins)

	// Настройка CORS с поддержкой динамических origins для разработки
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Bank-Code", "X-Payment-Consent-Id", "X-Fapi-Interaction-Id", "X-Fapi-Customer-Ip-Address", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Если origins пустой или содержит "*", разрешаем все origins (только для разработки!)
	allowAll := false
	for _, origin := range uniqueOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}

	if allowAll || len(uniqueOrigins) == 0 {
		// В режиме разработки разрешаем все origins
		corsConfig.AllowOriginFunc = func(origin string) bool {
			log.Printf("🔍 CORS request from origin: %s", origin)
			return true
		}
		log.Printf("⚠️  CORS: Allowing all origins (development mode)")
	} else {
		corsConfig.AllowOrigins = uniqueOrigins
	}

	r.Use(cors.New(corsConfig))

	// Middleware для логирования CORS запросов (только для отладки)
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			log.Printf("📡 Request from origin: %s, method: %s, path: %s", origin, c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	})

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

	subscriptionsHandler := subscriptions.NewHandler(repo)

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

	// --- Subscription endpoints ---
	secured.POST("/subscriptions", subscriptionsHandler.CreateSubscription)
	secured.GET("/subscriptions/status", subscriptionsHandler.GetSubscriptionStatus)

	return r
}
