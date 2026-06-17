package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/db"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/middleware"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

type App struct {
	db     *sql.DB
	router *gin.Engine
	log    *slog.Logger
}

func New() (*App, error) {
	log := newLogger()

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Error("JWT_SECRET_KEY environment variable is required")
		os.Exit(1)
	}

	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	log.Info("database connected")

	userRepo := repositories.NewUserRepository(database)
	tokenRepo := repositories.NewTokenRepository(database)
	jwtSvc := services.NewJWTService(jwtSecret)

	authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, jwtSvc)
	usersHandler := handlers.NewUsersHandler(userRepo)

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	router.GET("/healthz", handlers.HealthHandler())

	v1 := router.Group("/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/me", usersHandler.GetMe)
		protected.DELETE("/me", usersHandler.DeleteMe)
		handlers.RegisterAccountRoutes(protected, database)
		handlers.RegisterTransactionRoutes(protected, database)
	}

	return &App{
		db:     database,
		router: router,
		log:    log,
	}, nil
}

func (a *App) Start() error {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	a.log.Info("starting server", "port", port)
	return a.router.Run(":" + port)
}

func (a *App) Close() error {
	return a.db.Close()
}

func newLogger() *slog.Logger {
	if os.Getenv("API_ENV") == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
