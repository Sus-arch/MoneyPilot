package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State представляет состояние circuit breaker
type State int

const (
	StateClosed   State = iota // Закрыт - все запросы проходят
	StateOpen                  // Открыт - все запросы блокируются
	StateHalfOpen              // Полуоткрыт - пропускает ограниченное количество запросов для тестирования
)

// Config конфигурация circuit breaker
type Config struct {
	FailureThreshold    int           // Количество ошибок для открытия
	SuccessThreshold    int           // Количество успешных запросов для закрытия (в half-open)
	Timeout             time.Duration // Время ожидания перед переходом в half-open
	HalfOpenMaxRequests int           // Максимальное количество запросов в half-open состоянии
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// CircuitBreaker реализует паттерн circuit breaker
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	config          Config
}

// NewCircuitBreaker создает новый circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// Call выполняет функцию через circuit breaker
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	// Проверяем состояние перед выполнением
	if state == StateOpen {
		cb.mu.Lock()
		// Проверяем, не прошло ли достаточно времени для перехода в half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			state = StateHalfOpen
		}
		cb.mu.Unlock()

		if state == StateOpen {
			return errors.New("circuit breaker is open")
		}
	}

	// Выполняем функцию
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Ошибка - увеличиваем счетчик ошибок
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == StateHalfOpen {
			// В half-open состоянии любая ошибка переводит в open
			cb.state = StateOpen
			cb.failureCount = 0
			cb.successCount = 0
		} else if cb.state == StateClosed && cb.failureCount >= cb.config.FailureThreshold {
			// В closed состоянии превышение порога переводит в open
			cb.state = StateOpen
			cb.failureCount = 0
		}
	} else {
		// Успех - увеличиваем счетчик успехов
		cb.successCount++
		cb.failureCount = 0

		if cb.state == StateHalfOpen {
			if cb.successCount >= cb.config.SuccessThreshold {
				// Достаточно успешных запросов - переходим в closed
				cb.state = StateClosed
				cb.successCount = 0
			}
		} else if cb.state == StateClosed {
			// В closed состоянии сбрасываем счетчик ошибок при успехе
			cb.failureCount = 0
		}
	}

	return err
}

// GetState возвращает текущее состояние circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset сбрасывает состояние circuit breaker в closed
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}
}
