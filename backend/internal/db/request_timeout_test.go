package db

import (
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//nolint:gocyclo // One lifecycle verifies upgrade defaults, constraints, rollback and reapplication on an existing run.
func TestGenerationRequestTimeoutMigration(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "timeout-migration.db"))
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
	if checkErr := migrator.Steps(4); checkErr != nil {
		t.Fatal(checkErr)
	}
	for _, statement := range []string{
		"INSERT INTO llm_connections(id,name,created_at) VALUES('connection','Fixture','created')",
		"INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES('connection',1,X'01','created')",
		"INSERT INTO products(id,name,sources_json,created_at) VALUES('product','Fixture','[]','created')",
		`INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at) VALUES('legacy','product','connection',1,'original-model','failed','analyzing','{"sources":[]}','created')`,
	} {
		if _, checkErr := database.ExecContext(ctx, statement); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
	if checkErr := migrator.Steps(1); checkErr != nil {
		t.Fatal(checkErr)
	}
	var timeout, revision int
	var model, manifest string
	if checkErr := database.QueryRowContext(ctx, "SELECT request_timeout_seconds,connection_revision,model,input_json FROM generation_runs WHERE id='legacy'").Scan(&timeout, &revision, &model, &manifest); checkErr != nil {
		t.Fatal(checkErr)
	}
	if timeout != 0 || revision != 1 || model != "original-model" || manifest != `{"sources":[]}` {
		t.Fatal("migration changed a legacy run's frozen inputs")
	}
	for _, invalid := range []int{-1, 1, 19, 901} {
		if _, checkErr := database.ExecContext(ctx, "UPDATE generation_runs SET request_timeout_seconds=? WHERE id='legacy'", invalid); checkErr == nil {
			t.Fatalf("database accepted invalid timeout %d", invalid)
		}
	}
	for _, valid := range []int{0, 20, 600, 900} {
		if _, checkErr := database.ExecContext(ctx, "UPDATE generation_runs SET request_timeout_seconds=? WHERE id='legacy'", valid); checkErr != nil {
			t.Fatalf("database rejected timeout %d: %v", valid, checkErr)
		}
	}
	if checkErr := migrator.Steps(-1); checkErr != nil {
		t.Fatal(checkErr)
	}
	if checkErr := migrator.Steps(1); checkErr != nil {
		t.Fatal(checkErr)
	}
	if checkErr := database.QueryRowContext(ctx, "SELECT request_timeout_seconds FROM generation_runs WHERE id='legacy'").Scan(&timeout); checkErr != nil || timeout != 0 {
		t.Fatal("down/up did not preserve the legacy run with its default timeout", checkErr)
	}
	if checkErr := RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
}
