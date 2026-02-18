package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"subscription-service/internal/config"
	"subscription-service/internal/handlers"
	"subscription-service/internal/logger"
	"subscription-service/internal/repository"

	_ "subscription-service/docs"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger.InitLogger(cfg.LogLevel)
	log := logger.GetLogger()

	// Connect to database
	db, err := sql.Open("postgres", cfg.GetDBConnString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Info("Connected to database successfully")

	// Initialize repository and handlers
	repo := repository.NewPostgresRepository(db)
	handler := handlers.NewSubscriptionHandler(repo)

	// Setup router
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Subscription routes
		v1.POST("/subscriptions", handler.CreateSubscription)
		v1.GET("/subscriptions", handler.GetAllSubscriptions)
		v1.GET("/subscriptions/:id", handler.GetSubscription)
		v1.PATCH("/subscriptions/:id", handler.UpdateSubscription)
		v1.DELETE("/subscriptions/:id", handler.DeleteSubscription)
		v1.GET("/subscriptions/total-cost", handler.GetTotalCost)

		// Health check
		v1.GET("/health", handler.HealthCheck)
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	log.Infof("Server starting on %s", serverAddr)

	if err := router.Run(serverAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
