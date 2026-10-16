package db

import (
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestOpenAndMigrate(t *testing.T) {
	t.Parallel()
	// Include URI-significant characters to verify that file paths are escaped.
	path := filepath.Join(t.TempDir(), "nested", "onboarding ?#.db")
	for range 2 {
		database, err := Open(t.Context(), path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = database.Close() })
		for _, pragma := range []struct{ query, want string }{
			{"PRAGMA journal_mode", "wal"},
			{"PRAGMA foreign_keys", "1"},
			{"PRAGMA busy_timeout", "5000"},
		} {
			var got string
			if err := database.QueryRowContext(t.Context(), pragma.query).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != pragma.want {
				t.Errorf("%s = %q, want %q", pragma.query, got, pragma.want)
			}
		}
		for range 2 {
			if err := RunMigrations(database); err != nil {
				t.Fatal(err)
			}
		}
		var version, dirty int
		if err := database.QueryRowContext(t.Context(), "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
			t.Fatal(err)
		}
		if version != 12 || dirty != 0 {
			t.Fatalf("migration state = %d/%d", version, dirty)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenRejectsDirectory(t *testing.T) {
	t.Parallel()
	database, err := Open(t.Context(), t.TempDir())
	if err == nil {
		_ = database.Close()
		t.Fatal("expected opening a directory to fail")
	}
}

func TestPollingIndexesMigrationPreservesExistingIndexes(t *testing.T) {
	t.Parallel()
	for _, existing := range []string{"source_snapshots_run_id", "llm_calls_run_tokens", "both"} {
		t.Run(existing, func(t *testing.T) {
			t.Parallel()
			database, err := Open(t.Context(), filepath.Join(t.TempDir(), "polling.db"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = database.Close() })
			sourceDriver, err := iofs.New(migrationsFS, "migrations")
			if err != nil {
				t.Fatal(err)
			}
			databaseDriver, err := sqlite.WithInstance(database, &sqlite.Config{})
			if err != nil {
				t.Fatal(err)
			}
			migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", databaseDriver)
			if err != nil {
				t.Fatal(err)
			}
			if err := migrator.Migrate(8); err != nil {
				t.Fatal(err)
			}
			for name, statement := range map[string]string{
				"source_snapshots_run_id": "CREATE INDEX source_snapshots_run_id ON source_snapshots(run_id, id)",
				"llm_calls_run_tokens":    "CREATE INDEX llm_calls_run_tokens ON llm_calls(run_id, input_tokens, output_tokens)",
			} {
				if existing == name || existing == "both" {
					if _, err := database.ExecContext(t.Context(), statement); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := RunMigrations(database); err != nil {
				t.Fatal(err)
			}
			var version, dirty, indexes int
			if err := database.QueryRowContext(t.Context(), "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
				t.Fatal(err)
			}
			if version != 12 || dirty != 0 {
				t.Fatalf("migration state = %d/%d", version, dirty)
			}
			if err := database.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name IN ('source_snapshots_run_id', 'llm_calls_run_tokens')").Scan(&indexes); err != nil {
				t.Fatal(err)
			}
			if indexes != 2 {
				t.Fatalf("polling index count = %d, want 2", indexes)
			}
		})
	}
}
