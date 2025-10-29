package main

import (
	"MoneyPilot/internal/api"
	"MoneyPilot/internal/config"
	"MoneyPilot/internal/storage"
	"log"
)

func main() {
	cfg := config.Load()

	db := storage.NewPostgres(cfg.PostgresDSN)
	redis := storage.NewRedis(cfg.RedisAddr)
	defer db.Close()
	defer redis.Close()

	router := api.NewRouter()
	log.Printf("🚀 Server running on port %s\n", cfg.ServerPort)
	router.Run(":" + cfg.ServerPort)
}
