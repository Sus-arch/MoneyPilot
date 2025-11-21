package cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// CacheService предоставляет методы для работы с кэшем API ответов
type CacheService struct {
	db *sql.DB
}

// CacheEntry представляет запись в кэше
type CacheEntry struct {
	CacheKey  string
	Data      interface{}
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// NewCacheService создает новый сервис кэширования
func NewCacheService(db *sql.DB) *CacheService {
	return &CacheService{db: db}
}

// Get получает данные из кэша по ключу
// Возвращает данные и true, если данные актуальны (не истекли)
// Возвращает nil и false, если данных нет или они устарели
func (c *CacheService) Get(cacheKey string) (interface{}, bool, error) {
	var dataJSON []byte
	var expiresAt time.Time
	var updatedAt time.Time

	err := c.db.QueryRow(
		`SELECT data, expires_at, updated_at FROM api_cache WHERE cache_key = $1`,
		cacheKey,
	).Scan(&dataJSON, &expiresAt, &updatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	// Проверяем, не истек ли срок действия
	if time.Now().After(expiresAt) {
		// Удаляем устаревшую запись
		_ = c.Delete(cacheKey)
		return nil, false, nil
	}

	// Десериализуем данные
	var data interface{}
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, false, err
	}

	return data, true, nil
}

// Set сохраняет данные в кэш с указанным TTL
func (c *CacheService) Set(cacheKey string, data interface{}, ttl time.Duration) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(ttl)

	_, err = c.db.Exec(
		`INSERT INTO api_cache (cache_key, data, expires_at, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (cache_key) 
		 DO UPDATE SET data = $2, expires_at = $3, updated_at = NOW()`,
		cacheKey, dataJSON, expiresAt,
	)

	return err
}

// Delete удаляет запись из кэша
func (c *CacheService) Delete(cacheKey string) error {
	_, err := c.db.Exec(`DELETE FROM api_cache WHERE cache_key = $1`, cacheKey)
	return err
}

// IsStale проверяет, устарели ли данные (старше указанного времени)
// Возвращает true, если данные нужно обновить
func (c *CacheService) IsStale(cacheKey string, maxAge time.Duration) (bool, error) {
	var updatedAt time.Time
	err := c.db.QueryRow(
		`SELECT updated_at FROM api_cache WHERE cache_key = $1`,
		cacheKey,
	).Scan(&updatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil // Данных нет, нужно обновить
		}
		return false, err
	}

	// Проверяем, прошло ли достаточно времени с последнего обновления
	return time.Since(updatedAt) > maxAge, nil
}

// CleanupExpired удаляет все истекшие записи из кэша
func (c *CacheService) CleanupExpired() error {
	_, err := c.db.Exec(`DELETE FROM api_cache WHERE expires_at < NOW()`)
	return err
}
