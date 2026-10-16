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

const maxConnections = 8

// Open opens a local SQLite file with WAL and connection-local foreign key checks.
// A small pool permits concurrent readers while SQLite serializes writers.
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
	// Lock before reading in write transactions to avoid lock-upgrade races.
	// The driver keeps read-only transactions deferred.
	query.Set("_txlock", "immediate")
	dsn := url.URL{Scheme: "file", Path: absolute, RawQuery: query.Encode()}
	database, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	database.SetMaxOpenConns(maxConnections)
	database.SetMaxIdleConns(maxConnections)
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return database, nil
}
