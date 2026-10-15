package content

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

func TestImmutableContentAndChecksum(t *testing.T) {
	t.Parallel()
	files := &Store{Dir: filepath.Join(t.TempDir(), "content")}
	path := "products/product/versions/1/passport.json"
	hash, err := files.WriteJSON(path, map[string]string{"title": "Product"})
	if err != nil {
		t.Fatal(err)
	}
	var restored map[string]string
	if err := files.ReadJSON(path, hash, &restored); err != nil || restored["title"] != "Product" {
		t.Fatalf("could not read persisted content: %v", err)
	}
	if _, err := files.WriteJSON(path, map[string]string{"title": "Changed"}); err == nil {
		t.Fatal("overwrote an immutable version")
	}
	if _, err := files.Write("../outside.txt", []byte("outside")); err == nil {
		t.Fatal("accepted a path outside the content root")
	}
	if err := os.WriteFile(filepath.Join(files.Dir, path), []byte("corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Read(path, hash); err == nil {
		t.Fatal("accepted corrupted content")
	}
}

func TestLegacyExportIsRecoverableAndIdempotent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	_, err = database.ExecContext(ctx, `
CREATE TABLE passport_revisions(id TEXT,product_id TEXT,version INTEGER,content_json TEXT,content_path TEXT DEFAULT '',content_hash TEXT DEFAULT '');
CREATE TABLE source_snapshots(id TEXT,run_id TEXT);
CREATE TABLE source_documents(id TEXT,snapshot_id TEXT,content TEXT,content_hash TEXT DEFAULT '',content_path TEXT DEFAULT '');
CREATE TABLE llm_calls(id TEXT,run_id TEXT,output_json TEXT,output_path TEXT DEFAULT '',output_hash TEXT DEFAULT '');
INSERT INTO passport_revisions(id,product_id,version,content_json) VALUES('revision','product',1,'{"title":"Original"}');
INSERT INTO source_snapshots VALUES('snapshot','run');
INSERT INTO source_documents(id,snapshot_id,content) VALUES('document','snapshot','original source');
INSERT INTO llm_calls(id,run_id,output_json) VALUES('call','run','{"model":"response"}');`)
	if err != nil {
		t.Fatal(err)
	}
	files := &Store{Dir: t.TempDir()}
	// Fail after the passport file is published, before any metadata is committed.
	blocked := filepath.Join(files.Dir, "runs")
	if err := os.WriteFile(blocked, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(ctx, database, files); err == nil {
		t.Fatal("migration should fail on an unwritable destination")
	}
	var original string
	if err := database.QueryRowContext(ctx, "SELECT content_json FROM passport_revisions").Scan(&original); err != nil || original != `{"title":"Original"}` {
		t.Fatalf("failed export changed the original data: %q, %v", original, err)
	}
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := MigrateLegacy(ctx, database, files); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct{ query, empty, want string }{
		{"SELECT content_json,content_path,content_hash FROM passport_revisions", "{}", `{"title":"Original"}`},
		{"SELECT content,content_path,content_hash FROM source_documents", "", "original source"},
		{"SELECT output_json,output_path,output_hash FROM llm_calls", "", `{"model":"response"}`},
	} {
		var payload, path, hash string
		if err := database.QueryRowContext(ctx, test.query).Scan(&payload, &path, &hash); err != nil {
			t.Fatal(err)
		}
		data, err := files.Read(path, hash)
		if json.Valid(data) {
			var compact bytes.Buffer
			if compactErr := json.Compact(&compact, data); compactErr != nil {
				t.Fatal(compactErr)
			}
			data = compact.Bytes()
		}
		if err != nil || payload != test.empty || string(data) != test.want {
			t.Fatalf("export lost content or left a database payload: %q, %q, %v", payload, data, err)
		}
	}
}
