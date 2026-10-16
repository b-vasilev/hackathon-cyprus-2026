package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func openCounter(t *testing.T) *sql.DB {
	t.Helper()
	database, err := Open(t.Context(), filepath.Join(t.TempDir(), "counter.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.ExecContext(t.Context(), "CREATE TABLE counter (value INTEGER NOT NULL); INSERT INTO counter VALUES (0)"); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestReadDuringWrite(t *testing.T) {
	t.Parallel()
	database := openCounter(t)
	writer, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback() }()
	if _, execErr := writer.ExecContext(t.Context(), "UPDATE counter SET value=1"); execErr != nil {
		t.Fatal(execErr)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	var value int
	if scanErr := database.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); scanErr != nil {
		t.Fatalf("reader blocked by an open write transaction: %v", scanErr)
	}
	if value != 0 {
		t.Fatalf("reader saw uncommitted value %d", value)
	}
	reader, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatalf("read-only transaction blocked by writer: %v", err)
	}
	defer func() { _ = reader.Rollback() }()
	if err := reader.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); err != nil || value != 0 {
		t.Fatalf("read-only snapshot = %d: %v", value, err)
	}
	if err := writer.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); err != nil || value != 1 {
		t.Fatalf("committed value = %d: %v", value, err)
	}
	if err := reader.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); err != nil || value != 0 {
		t.Fatalf("read-only snapshot changed during write: %d: %v", value, err)
	}
}

func incrementCounter(ctx context.Context, database *sql.DB) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin counter update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var value int
	if err := tx.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); err != nil {
		return fmt.Errorf("read counter: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE counter SET value=?", value+1); err != nil {
		return fmt.Errorf("update counter: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit counter: %w", err)
	}
	return nil
}

func TestConcurrentWriteTransactions(t *testing.T) {
	t.Parallel()
	database := openCounter(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	const writers = 8
	start := make(chan struct{})
	results := make(chan error, writers)
	for range writers {
		go func() {
			<-start
			results <- incrementCounter(ctx, database)
		}()
	}
	close(start)
	for range writers {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	var value int
	if err := database.QueryRowContext(ctx, "SELECT value FROM counter").Scan(&value); err != nil || value != writers {
		t.Fatalf("counter = %d, want %d: %v", value, writers, err)
	}
}
