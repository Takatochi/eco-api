package router

import (
	"eco-api/internal/handler"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(h *handler.Handler) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/ping", h.Ping)
	engine.GET("/health", h.Health)
	engine.GET("/health/ping", h.Ping)
	engine.GET("/openapi.json", h.OpenAPI)
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/openapi.json")))

	api := engine.Group("/api")
	api.POST("/measurements", h.CreateMeasurement)
	api.GET("/measurements", h.ListMeasurements)

	return engine
}
