package bankapi

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenService struct {
	rdb *redis.Client
}

func NewTokenService(rdb *redis.Client) *TokenService {
	return &TokenService{rdb: rdb}
}

func (s *TokenService) GetValidToken(bank *BankClient) (*ActiveToken, error) {
	ctx := context.Background()

	// Сначала пробуем взять токен из Redis
	data, err := s.rdb.Get(ctx, "token:"+bank.Name).Bytes()
	if err == nil {
		var token ActiveToken
		json.Unmarshal(data, &token)

		// Проверяем срок действия
		// можно хранить expiresAt вручную или TTL Redis
		if token.ExpiresAt.After(time.Now()) {
			return &token, nil
		}
	}

	// Иначе — запрашиваем новый
	token, err := bank.GetToken()
	if err != nil {
		return nil, err
	}

	// Кэшируем с TTL = 24 часа
	raw, _ := json.Marshal(token)
	err = s.rdb.Set(ctx, "token:"+bank.Name, raw, 24*time.Hour).Err()
	if err != nil {
		log.Println("[Redis] cannot save token:", err)
	}

	return token, nil
}
