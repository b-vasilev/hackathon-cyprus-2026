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
		if version != 17 || dirty != 0 {
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

func TestPracticeSourceMigrationRequiresKnownContent(t *testing.T) {
	t.Parallel()
	database, err := Open(t.Context(), filepath.Join(t.TempDir(), "practice.db"))
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
	if err := migrator.Migrate(16); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO llm_connections(id,name,created_at) VALUES('connection','Fixture','created')`,
		`INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES('connection',1,X'01','created')`,
		`INSERT INTO products(id,name,sources_json,created_at) VALUES('product','Product','[]','created')`,
		`INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at)
 VALUES('run','product','connection',1,'model','succeeded','complete','{}','created'),
 ('edited-run','product','connection',1,'model','succeeded','complete','{}','created')`,
		`INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_hash,published_content_hash,edited_at,created_at)
 VALUES('original','product','run',1,'2','{}','original-hash','original-hash','','created'),
 ('edited','product','edited-run',2,'2','{}','edited-hash','original-hash','edited','created')`,
		`INSERT INTO learning_packs(id,product_id,revision_id,status,content_json,published_content_json,created_at,updated_at,published_at)
 VALUES('original-pack','product','original','ready','{}','{}','created','created','published'),
 ('edited-pack','product','edited','ready','{}','{}','created','created','published')`,
	} {
		if _, err := database.ExecContext(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := migrator.Migrate(17); err != nil {
			t.Fatal(err)
		}
		for _, revision := range []string{"original", "edited"} {
			var working, published string
			if err := database.QueryRowContext(t.Context(), "SELECT source_content_hash,published_source_content_hash FROM learning_packs WHERE revision_id=?", revision).Scan(&working, &published); err != nil {
				t.Fatal(err)
			}
			want := ""
			if revision == "original" {
				want = "original-hash"
			}
			if working != want || published != want {
				t.Fatalf("%s practice source = %q/%q, want %q", revision, working, published, want)
			}
		}
		if err := migrator.Steps(-1); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPersonalConnectionsMigration(t *testing.T) {
	t.Parallel()
	database, err := Open(t.Context(), filepath.Join(t.TempDir(), "personal.db"))
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
	if migrationErr := migrator.Migrate(14); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	for _, statement := range []string{
		`INSERT INTO users VALUES('admin','admin@example.com','hash','admin','2026-01-02'),('user','user@example.com','hash','user','2026-01-01'),('later-admin','later@example.com','hash','admin','2026-01-03')`,
		`INSERT INTO llm_connections(id,name,created_at) VALUES('llm','LLM','now')`,
		`INSERT INTO llm_connection_revisions VALUES('llm',1,X'1234','now')`,
		`INSERT INTO products(id,name,sources_json,created_at) VALUES('product','Product','[]','now')`,
		`INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at) VALUES('run','product','llm',1,'model','queued','queued','[]','now')`,
		`INSERT INTO github_connection VALUES(1,'octocat','token',X'5678','now')`,
	} {
		if _, insertErr := database.ExecContext(t.Context(), statement); insertErr != nil {
			t.Fatal(insertErr)
		}
	}
	for range 2 {
		if migrationErr := migrator.Migrate(15); migrationErr != nil {
			t.Fatal(migrationErr)
		}
		for _, table := range []string{"llm_connections", "generation_runs", "github_connection"} {
			var owner string
			if queryErr := database.QueryRowContext(t.Context(), "SELECT owner_id FROM "+table).Scan(&owner); queryErr != nil || owner != "admin" {
				t.Fatalf("%s owner=%q: %v", table, owner, queryErr)
			}
		}
		var sealed string
		if queryErr := database.QueryRowContext(t.Context(), "SELECT hex(encrypted_token) FROM github_connection").Scan(&sealed); queryErr != nil || sealed != "5678" {
			t.Fatal("migration changed the saved GitHub credential", queryErr)
		}
		if migrationErr := migrator.Steps(-1); migrationErr != nil {
			t.Fatal(migrationErr)
		}
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
			if version != 17 || dirty != 0 {
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
