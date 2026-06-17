package app

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/vladyslavivchenko/netme/internal/db"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/middleware"
)

type App struct {
	db    *sql.DB
	redis *redis.Client
	router *gin.Engine
}

func New() (*App, error) {
	// Connect to database
	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	// Create Gin router
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Health check endpoints (public)
	router.GET("/healthz", handlers.HealthHandler())

	// Register API routes
	api := router.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/hello", handlers.HelloHandler())

		// Auth endpoints
		handlers.RegisterAuthRoutes(api, database)
		handlers.RegisterAccountRoutes(api, database)
		handlers.RegisterTransactionRoutes(api, database)
		handlers.RegisterAnalyticsRoutes(api, database)
	}

	return &App{
		db:    database,
		redis: redisClient,
		router: router,
	}, nil
}

func (a *App) Start() error {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting server on port %s\n", port)
	return a.router.Run(":" + port)
}

func (a *App) Close() error {
	if err := a.db.Close(); err != nil {
		return err
	}
	return a.redis.Close()
}
