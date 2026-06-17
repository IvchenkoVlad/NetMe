package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ensureMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) UNIQUE NOT NULL,
			direction VARCHAR(10) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func getMigrationDirs() []string {
	return []string{"tables", "indices", "functions"}
}

func getMigrationVersions(db *sql.DB) (map[string]bool, error) {
	versions := make(map[string]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations WHERE direction = 'up'")
	if err != nil {
		return versions, err
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return versions, err
		}
		versions[version] = true
	}
	return versions, rows.Err()
}

func recordMigration(db *sql.DB, version, direction string) error {
	_, err := db.Exec(
		"INSERT INTO schema_migrations (version, direction) VALUES ($1, $2)",
		version, direction,
	)
	return err
}

func removeMigrationRecord(db *sql.DB, version string) error {
	_, err := db.Exec(
		"DELETE FROM schema_migrations WHERE version = $1 AND direction = 'up'",
		version,
	)
	return err
}


type Migration struct {
	Version   string
	Type      string
	Name      string
	UpSQL     string
	DownSQL   string
}

func loadMigrations() ([]Migration, error) {
	var migrations []Migration
	dirs := getMigrationDirs()

	for _, dir := range dirs {
		pattern := filepath.Join("internal/db/migrations", dir, "*.up.sql")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			base := filepath.Base(match)
			name := strings.TrimSuffix(base, ".up.sql")

			downFile := strings.TrimSuffix(match, ".up.sql") + ".down.sql"
			upContent, err := os.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("failed to read migration %s: %w", match, err)
			}

			downContent, err := os.ReadFile(downFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read migration %s: %w", downFile, err)
			}

			migrations = append(migrations, Migration{
				Version: name,
				Type:    dir,
				Name:    name,
				UpSQL:   string(upContent),
				DownSQL: string(downContent),
			})
		}
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}