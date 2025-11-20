package bankapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"MoneyPilot/internal/cache"
	"MoneyPilot/internal/circuitbreaker"
	"MoneyPilot/internal/ratelimit"
	"MoneyPilot/internal/retry"
)

// ClientWrapper обертка над HTTP клиентом с retry, rate limiting, circuit breaker и кэшированием
type ClientWrapper struct {
	httpClient      *http.Client
	rateLimiter     *ratelimit.RateLimiter
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
	cacheService    *cache.CacheService
	cacheTTL        time.Duration
	maxCacheAge     time.Duration // Максимальный возраст кэша перед обновлением
}

// Config конфигурация для ClientWrapper
type ClientWrapperConfig struct {
	HTTPTimeout          time.Duration
	RateLimitPerSec      int
	CacheTTL             time.Duration // TTL для кэша
	MaxCacheAge          time.Duration // Максимальный возраст кэша перед обновлением (например, 5 минут)
	RetryConfig          retry.Config
	CircuitBreakerConfig circuitbreaker.Config
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() ClientWrapperConfig {
	return ClientWrapperConfig{
		HTTPTimeout:          15 * time.Second,
		RateLimitPerSec:      1, // 1 запрос в секунду
		CacheTTL:             1 * time.Hour,
		MaxCacheAge:          5 * time.Minute, // Обновляем, если данные старше 5 минут
		RetryConfig:          retry.DefaultConfig(),
		CircuitBreakerConfig: circuitbreaker.DefaultConfig(),
	}
}

// NewClientWrapper создает новый обернутый HTTP клиент
func NewClientWrapper(cacheService *cache.CacheService, config ClientWrapperConfig) *ClientWrapper {
	return &ClientWrapper{
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		rateLimiter:     ratelimit.NewRateLimiter(config.RateLimitPerSec, time.Second),
		circuitBreakers: make(map[string]*circuitbreaker.CircuitBreaker),
		cacheService:    cacheService,
		cacheTTL:        config.CacheTTL,
		maxCacheAge:     config.MaxCacheAge,
	}
}

// getOrCreateCircuitBreaker получает или создает circuit breaker для банка
func (cw *ClientWrapper) getOrCreateCircuitBreaker(bankCode string) *circuitbreaker.CircuitBreaker {
	if cb, exists := cw.circuitBreakers[bankCode]; exists {
		return cb
	}

	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.DefaultConfig())
	cw.circuitBreakers[bankCode] = cb
	return cb
}

// Do выполняет HTTP запрос с применением всех механизмов защиты
// bankCode - код банка для rate limiting и circuit breaker
// cacheKey - ключ для кэширования (если пустой, кэширование не используется)
// useCache - использовать ли кэш (проверка актуальности и возврат из кэша)
func (cw *ClientWrapper) Do(ctx context.Context, bankCode string, req *http.Request, cacheKey string, useCache bool) (*http.Response, error) {
	// Если используется кэш, сначала проверяем его
	if useCache && cacheKey != "" && cw.cacheService != nil {
		cached, found, err := cw.cacheService.Get(cacheKey)
		if err == nil && found {
			// Проверяем актуальность данных
			isStale, err := cw.cacheService.IsStale(cacheKey, cw.maxCacheAge)
			if err == nil && !isStale {
				// Данные актуальны, возвращаем из кэша
				return cw.createResponseFromCache(cached)
			}
			// Данные устарели, но можем вернуть их, если API недоступен
		}
	}

	// Получаем circuit breaker для банка
	cb := cw.getOrCreateCircuitBreaker(bankCode)

	var resp *http.Response
	var err error

	// Выполняем запрос через circuit breaker с retry
	err = cb.Call(func() error {
		// Применяем rate limiting
		cw.rateLimiter.Wait(bankCode)

		// Выполняем запрос с retry
		return retry.Do(ctx, retry.DefaultConfig(), func() error {
			// Создаем новый запрос с контекстом
			reqWithCtx := req.WithContext(ctx)

			var doErr error
			resp, doErr = cw.httpClient.Do(reqWithCtx)

			if doErr != nil {
				return doErr
			}

			// Проверяем статус код
			if retry.IsRetryableHTTPError(resp.StatusCode) {
				resp.Body.Close()
				return fmt.Errorf("retryable HTTP error: %d", resp.StatusCode)
			}

			return nil
		})
	})

	if err != nil {
		// Если есть закэшированные данные, возвращаем их как fallback
		if useCache && cacheKey != "" && cw.cacheService != nil {
			cached, found, _ := cw.cacheService.Get(cacheKey)
			if found {
				return cw.createResponseFromCache(cached)
			}
		}
		return nil, err
	}

	// Если запрос успешен и используется кэш, сохраняем ответ
	if useCache && cacheKey != "" && cw.cacheService != nil && resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			// Сохраняем в кэш
			var data interface{}
			if json.Unmarshal(bodyBytes, &data) == nil {
				_ = cw.cacheService.Set(cacheKey, data, cw.cacheTTL)
			}
			// Создаем новый reader для response body
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return resp, nil
}

// createResponseFromCache создает HTTP response из кэшированных данных
func (cw *ClientWrapper) CreateResponseFromCache(cached interface{}) (*http.Response, error) {
	return cw.createResponseFromCache(cached)
}

func (cw *ClientWrapper) createResponseFromCache(cached interface{}) (*http.Response, error) {
	bodyBytes, err := json.Marshal(cached)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("X-Cache", "HIT")

	return resp, nil
}
