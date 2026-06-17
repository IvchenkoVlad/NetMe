package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/vladyslavivchenko/netme/internal/db"
)

func main() {
	// Load environment variables
	if err := godotenv.Load("../../.env.local"); err != nil {
		log.Println("No .env.local file found, using system environment")
	}

	// Parse command-line arguments
	upCmd := flag.NewFlagSet("up", flag.ExitOnError)
	downCmd := flag.NewFlagSet("down", flag.ExitOnError)
	downSteps := downCmd.Int("steps", 1, "Number of migrations to rollback")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Connect to database
	database, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	switch command {
	case "up":
		upCmd.Parse(os.Args[2:])
		if err := db.Migrate(database); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Migrations completed successfully")

	case "down":
		downCmd.Parse(os.Args[2:])
		steps := *downSteps
		if len(downCmd.Args()) > 0 {
			if s, err := strconv.Atoi(downCmd.Args()[0]); err == nil {
				steps = s
			}
		}
		if err := db.MigrateDown(database, steps); err != nil {
			fmt.Fprintf(os.Stderr, "Migration rollback failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Migrations rolled back successfully")

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: migrate <command> [options]

Commands:
  up              Apply all pending migrations
  down [steps]    Rollback migrations (default: 1 step)

Examples:
  migrate up           # Apply all pending migrations
  migrate down         # Rollback last migration
  migrate down 3       # Rollback last 3 migrations
  migrate down --steps 2  # Rollback 2 migrations (same as above)
`)
}
