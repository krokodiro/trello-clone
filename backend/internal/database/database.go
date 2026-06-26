package database

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func RunMigrations(databaseURL string) error {
	return RunMigrationsFrom(databaseURL, "migrations")
}

func RunMigrationsFrom(databaseURL, migrationsDir string) error {
	// The pgx/v5 migrate driver registers under the "pgx5" scheme, so rewrite
	// the standard postgres:// URL (used by pgxpool) to match.
	migrateURL := databaseURL
	if rest, ok := strings.CutPrefix(migrateURL, "postgres://"); ok {
		migrateURL = "pgx5://" + rest
	} else if rest, ok := strings.CutPrefix(migrateURL, "postgresql://"); ok {
		migrateURL = "pgx5://" + rest
	}

	m, err := migrate.New("file://"+filepath.ToSlash(migrationsDir), migrateURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Println("migrations applied")
	return nil
}
