package storage

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("[Redis] connection failed: %v", err)
	}
	log.Println("✅ Connected to Redis")
	return rdb
}
