package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // Registers the pure-Go SQLite driver.
)

// Open opens a local SQLite file with WAL and connection-local foreign key checks.
// A single connection serializes writes for this single-process MVP.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(absolute), 0o750); mkdirErr != nil {
		return nil, fmt.Errorf("create database directory: %w", mkdirErr)
	}
	query := url.Values{}
	for _, pragma := range []string{"journal_mode(WAL)", "busy_timeout(5000)", "foreign_keys(1)", "synchronous(NORMAL)"} {
		query.Add("_pragma", pragma)
	}
	dsn := url.URL{Scheme: "file", Path: absolute, RawQuery: query.Encode()}
	database, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return database, nil
}
