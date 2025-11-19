package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter ограничивает количество запросов в единицу времени для каждого банка
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*tokenBucket
	rate     int           // Количество запросов
	per      time.Duration // За период времени
}

// tokenBucket реализует алгоритм token bucket
type tokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter создает новый rate limiter
// rate - количество запросов
// per - период времени (например, 1 секунда)
func NewRateLimiter(rate int, per time.Duration) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*tokenBucket),
		rate:     rate,
		per:      per,
	}
}

// getOrCreateBucket получает или создает bucket для банка
func (rl *RateLimiter) getOrCreateBucket(bankCode string) *tokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.limiters[bankCode]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Двойная проверка
	if bucket, exists := rl.limiters[bankCode]; exists {
		return bucket
	}

	// Создаем новый bucket
	refillRate := float64(rl.rate) / rl.per.Seconds()
	bucket = &tokenBucket{
		tokens:     float64(rl.rate),
		capacity:   float64(rl.rate),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}

	rl.limiters[bankCode] = bucket
	return bucket
}

// Allow проверяет, разрешен ли запрос для указанного банка
// Возвращает true, если запрос разрешен, и время ожидания до следующего разрешенного запроса
func (rl *RateLimiter) Allow(bankCode string) (bool, time.Duration) {
	bucket := rl.getOrCreateBucket(bankCode)
	return bucket.allow()
}

// Wait блокирует выполнение до тех пор, пока запрос не будет разрешен
func (rl *RateLimiter) Wait(bankCode string) {
	bucket := rl.getOrCreateBucket(bankCode)
	bucket.wait()
}

// allow проверяет, есть ли токены в bucket
func (tb *tokenBucket) allow() (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Пополняем токены
	tb.tokens = tb.tokens + elapsed*tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	// Проверяем, есть ли токены
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true, 0
	}

	// Вычисляем время до следующего доступного токена
	needed := 1.0 - tb.tokens
	waitTime := time.Duration(needed/tb.refillRate) * time.Second
	return false, waitTime
}

// wait блокирует выполнение до получения токена
func (tb *tokenBucket) wait() {
	for {
		allowed, waitTime := tb.allow()
		if allowed {
			return
		}
		time.Sleep(waitTime)
	}
}
