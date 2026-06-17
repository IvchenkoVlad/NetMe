package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladyslavivchenko/netme/internal/app"
)

func main() {
	// Load environment variables
	envPaths := []string{
		".env.local",           // Current directory
		"../../.env.local",     // Running from cmd/server
		"/Users/vladyslavivchenko/Desktop/netme/.env.local", // Absolute path
	}

	for _, envPath := range envPaths {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded environment from: %s\n", envPath)
			break
		}
	}

	// Verify DATABASE_URL is set
	if os.Getenv("DATABASE_URL") == "" {
		log.Println("WARNING: DATABASE_URL not set. Using default: postgres://netme:devpassword@localhost:5432/netme_dev")
		os.Setenv("DATABASE_URL", "postgres://netme:devpassword@localhost:5432/netme_dev")
	}
	if os.Getenv("REDIS_URL") == "" {
		log.Println("WARNING: REDIS_URL not set. Using default: redis://localhost:6379")
		os.Setenv("REDIS_URL", "redis://localhost:6379")
	}

	// Create and start application
	application, err := app.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	if err := application.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start app: %v\n", err)
		os.Exit(1)
	}
}
