package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort   string
	PostgresDSN  string
	RedisAddr    string
	MLServiceURL string
}

func Load() *Config {
	_ = godotenv.Load(".env")

	cfg := &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://postgres:balance@localhost:5432/finbalance?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		MLServiceURL: getEnv("ML_URL", "http://127.0.0.1:8000"),
	}

	log.Println("✅ Config loaded")
	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
