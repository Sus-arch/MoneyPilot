package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort   string
	PostgresDSN  string
	RedisAddr    string
	MLServiceURL string
	Login        string
	Password     string
	CORSOrigins  []string
}

func Load() *Config {
	_ = godotenv.Load(".env")

	// Парсим CORS origins из переменной окружения (разделены запятой)
	corsOriginsEnv := getEnv("CORS_ORIGINS", "http://localhost:5173")
	corsOrigins := strings.Split(corsOriginsEnv, ",")
	// Убираем пробелы вокруг каждого origin
	for i, origin := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(origin)
	}

	cfg := &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://postgres:balance@localhost:5432/finbalance?sslmode=disable"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		MLServiceURL: getEnv("ML_URL", "http://127.0.0.1:8000"),
		Login:        getEnv("LOGIN_HAC", "team081"),
		Password:     getEnv("PASSWORD_HAC", "ddslFory8voO3gxZ2CEaQnHzLfv4HVzo"),
		CORSOrigins:  corsOrigins,
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
