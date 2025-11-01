package api

import (
	"MoneyPilot/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
		api.GET("/example", handlers.Example)
	}

	return r
}
