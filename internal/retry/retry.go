package retry

import (
	"context"
	"math"
	"net/http"
	"time"
)

// Config конфигурация для retry механизма
type Config struct {
	MaxAttempts       int           // Максимальное количество попыток
	InitialDelay      time.Duration // Начальная задержка
	MaxDelay          time.Duration // Максимальная задержка
	BackoffMultiplier float64       // Множитель для экспоненциальной задержки
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// IsRetryableError проверяет, можно ли повторить запрос при данной ошибке
func IsRetryableError(err error, statusCode int) bool {
	if err != nil {
		// Сетевые ошибки, таймауты - можно повторить
		return true
	}

	// Временные ошибки HTTP - можно повторить
	if statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		(statusCode >= 500 && statusCode < 600) {
		return true
	}

	return false
}

// Do выполняет функцию с retry механизмом
func Do(ctx context.Context, config Config, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Проверяем контекст перед каждой попыткой
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Выполняем функцию
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Если это последняя попытка, не ждем
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Вычисляем задержку с экспоненциальным backoff
		delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffMultiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// DoWithResponse выполняет функцию, которая возвращает статус код, с retry механизмом
func DoWithResponse(ctx context.Context, config Config, fn func() (int, error)) (int, error) {
	var lastErr error
	var lastStatusCode int

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		statusCode, err := fn()
		if err == nil && !IsRetryableError(nil, statusCode) {
			return statusCode, nil
		}

		lastErr = err
		lastStatusCode = statusCode

		if attempt == config.MaxAttempts-1 {
			break
		}

		delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffMultiplier, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastStatusCode, lastErr
}

// IsRetryableHTTPError проверяет, является ли HTTP ошибка временной
func IsRetryableHTTPError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout ||
		(statusCode >= 500 && statusCode < 600)
}
