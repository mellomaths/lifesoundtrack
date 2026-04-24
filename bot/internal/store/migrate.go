package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations runs SQL migrations in migrationsDir (Goose) against the database.
func RunMigrations(migrationsDir, databaseURL string) error {
	if databaseURL == "" {
		return fmt.Errorf("empty database url")
	}
	dir, err := filepath.Abs(filepath.Clean(migrationsDir))
	if err != nil {
		return fmt.Errorf("migrations dir: %w", err)
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.UpContext(context.Background(), db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
