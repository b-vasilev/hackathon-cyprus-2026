package db

import (
	"path/filepath"
	"testing"
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
		if version != 1 || dirty != 0 {
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
