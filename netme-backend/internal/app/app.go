package app

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/db"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/middleware"
)

type App struct {
	db     *sql.DB
	router *gin.Engine
}

func New() (*App, error) {
	// Connect to database
	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	// Create Gin router
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Health check endpoints (public)
	router.GET("/healthz", handlers.HealthHandler())

	// Register API routes
	api := router.Group("/api/v1")
	{
		// Auth endpoints
		handlers.RegisterAuthRoutes(api, database)
		handlers.RegisterAccountRoutes(api, database)
		handlers.RegisterTransactionRoutes(api, database)
	}

	return &App{
		db:     database,
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
	return a.db.Close()
}
