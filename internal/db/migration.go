package db

import (
	"context"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all pending migrations against the database.
// It reads SQL files from the embedded filesystem and executes the "create" section.
// Uses a simple version tracking table to avoid re-running migrations.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationFS embed.FS) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for migrations: %w", err)
	}
	defer conn.Release()

	// Create version tracking table if it doesn't exist
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = conn.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	// Read migration files
	entries, err := migrationFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migration dir: %w", err)
	}

	applied := 0
	for i, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := i + 1
		if version <= currentVersion {
			continue
		}

		content, err := migrationFS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", entry.Name(), err)
		}

		// Extract the "create" section (between ---- create and ---- drop)
		sql := extractCreateSQL(string(content))

		// Execute migration
		_, err = conn.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}

		// Record version
		_, err = conn.Exec(ctx, `INSERT INTO schema_version (version) VALUES ($1)`, version)
		if err != nil {
			return fmt.Errorf("record migration version %d: %w", version, err)
		}

		applied++
		log.Printf("Applied migration: %s (version %d)", entry.Name(), version)
	}

	if applied == 0 {
		log.Printf("Migrations up to date (version %d)", currentVersion)
	} else {
		log.Printf("Applied %d migration(s)", applied)
	}

	return nil
}

// extractCreateSQL extracts the SQL between "---- create" and "---- drop" markers.
// If no markers are found, returns the entire content.
func extractCreateSQL(content string) string {
	createIdx := strings.Index(content, "---- create")
	dropIdx := strings.Index(content, "---- drop")

	if createIdx == -1 {
		return content
	}

	// Start after the "---- create" line
	start := strings.Index(content[createIdx:], "\n")
	if start == -1 {
		return content
	}
	start += createIdx + 1

	if dropIdx == -1 {
		return content[start:]
	}

	return content[start:dropIdx]
}
