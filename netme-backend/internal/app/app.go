package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vladyslavivchenko/netme/internal/crypto"
	"github.com/vladyslavivchenko/netme/internal/db"
	"github.com/vladyslavivchenko/netme/internal/handlers"
	"github.com/vladyslavivchenko/netme/internal/jobs"
	"github.com/vladyslavivchenko/netme/internal/middleware"
	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
	"golang.org/x/time/rate"
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

	plaidKey, err := crypto.ParseKey(os.Getenv("PLAID_TOKEN_ENCRYPTION_KEY"))
	if err != nil {
		return nil, fmt.Errorf("plaid encryption key: %w", err)
	}
	if plaidKey == nil {
		log.Warn("PLAID_TOKEN_ENCRYPTION_KEY not set — Plaid access tokens stored unencrypted")
	}

	userRepo := repositories.NewUserRepository(database)
	tokenRepo := repositories.NewTokenRepository(database)
	itemRepo := repositories.NewPlaidItemRepository(database, plaidKey)
	acctRepo := repositories.NewAccountRepository(database)
	txnRepo := repositories.NewTransactionRepository(database)
	eventRepo := repositories.NewEventRepository(database)
	budgetRepo := repositories.NewBudgetRepository(database)
	rulesRepo := repositories.NewRulesRepository(database)

	jwtSvc := services.NewJWTService(jwtSecret)

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleVerifier := services.NewGoogleIDTokenVerifier(googleClientID)

	plaidSvc := services.NewPlaidService(
		os.Getenv("PLAID_CLIENT_ID"),
		os.Getenv("PLAID_SECRET"),
		os.Getenv("PLAID_ENV"),
		itemRepo,
		acctRepo,
		txnRepo,
		eventRepo,
		rulesRepo,
	)

	authSvc := services.NewAuthService(userRepo, tokenRepo, jwtSvc, googleVerifier)

	authHandler := handlers.NewAuthHandler(authSvc)
	usersHandler := handlers.NewUsersHandler(userRepo, plaidSvc)

	router := gin.Default()
	router.Use(middleware.HTTPSRedirect(os.Getenv("API_ENV")))
	router.Use(middleware.CORSMiddleware())
	router.GET("/healthz", handlers.HealthHandler())

	v1 := router.Group("/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", middleware.RateLimiter(rate.Every(12*time.Second), 5), authHandler.Register)
		auth.POST("/login", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Login)
		auth.POST("/refresh", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.Refresh)
		auth.POST("/google", middleware.RateLimiter(rate.Every(6*time.Second), 10), authHandler.GoogleAuth)
	}

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/me", usersHandler.GetMe)
		protected.DELETE("/me", usersHandler.DeleteMe)
		handlers.RegisterAccountRoutes(protected, acctRepo)
		handlers.RegisterTransactionRoutes(protected, txnRepo)
		handlers.RegisterPlaidRoutes(protected, v1, plaidSvc, itemRepo, eventRepo)
		handlers.RegisterBudgetRoutes(protected, budgetRepo)
		handlers.RegisterRulesRoutes(protected, rulesRepo)
		handlers.RegisterAnalyticsRoutes(protected, acctRepo, budgetRepo)
	}

	scheduler := jobs.NewScheduler(plaidSvc, itemRepo, acctRepo, eventRepo, log)
	go scheduler.Start(context.Background())

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
