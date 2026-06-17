package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is empty")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

func Migrate(db *sql.DB) error {
	if err := ensureMigrationTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := getMigrationVersions(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, m := range migrations {
		if applied[m.Version] {
			log.Printf("Migration %s already applied, skipping\n", m.Version)
			continue
		}

		log.Printf("Running migration %s...\n", m.Version)
		if _, err := db.Exec(m.UpSQL); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}

		if err := recordMigration(db, m.Version, "up"); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
		}

		log.Printf("✓ Migration %s completed\n", m.Version)
	}

	return nil
}

func MigrateDown(db *sql.DB, steps int) error {
	if err := ensureMigrationTable(db); err != nil {
		return fmt.Errorf("failed to access schema_migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	applied, err := getMigrationVersions(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Reverse order to rollback
	for i := len(migrations) - 1; i >= 0 && steps > 0; i-- {
		m := migrations[i]
		if !applied[m.Version] {
			continue
		}

		log.Printf("Rolling back migration %s...\n", m.Version)
		if _, err := db.Exec(m.DownSQL); err != nil {
			return fmt.Errorf("migration %s rollback failed: %w", m.Version, err)
		}

		if err := removeMigrationRecord(db, m.Version); err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", m.Version, err)
		}

		log.Printf("✓ Migration %s rolled back\n", m.Version)
		steps--
	}

	return nil
}
