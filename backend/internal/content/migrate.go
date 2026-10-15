package content

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// MigrateLegacy preserves existing versions before removing payloads from SQLite.
// Retrying after a failed startup verifies and reuses identical immutable files.
func MigrateLegacy(ctx context.Context, database *sql.DB, files *Store) error {
	if _, err := database.ExecContext(ctx, "PRAGMA secure_delete=ON"); err != nil {
		return fmt.Errorf("enable payload cleanup: %w", err)
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin content migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	changed := false
	for _, migration := range []struct {
		query, update string
	}{
		{"SELECT id,'products/'||product_id||'/versions/'||version||'/passport.json',content_json FROM passport_revisions WHERE content_path='' LIMIT 1", "UPDATE passport_revisions SET content_path=?,content_hash=?,content_json='{}' WHERE id=?"},
		{"SELECT d.id,'runs/'||s.run_id||'/sources/'||d.id||'.txt',d.content FROM source_documents d JOIN source_snapshots s ON s.id=d.snapshot_id WHERE d.content_path='' LIMIT 1", "UPDATE source_documents SET content_path=?,content_hash=?,content='' WHERE id=?"},
		{"SELECT id,'runs/'||run_id||'/model-responses/'||id||CASE WHEN json_valid(output_json) THEN '.json' ELSE '.txt' END,output_json FROM llm_calls WHERE output_path='' AND output_json<>'' LIMIT 1", "UPDATE llm_calls SET output_path=?,output_hash=?,output_json='' WHERE id=?"},
	} {
		for {
			var id, path, payload string
			err := tx.QueryRowContext(ctx, migration.query).Scan(&id, &path, &payload)
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			if err != nil {
				return fmt.Errorf("read legacy content: %w", err)
			}
			data := []byte(payload)
			if strings.HasSuffix(path, "/passport.json") {
				if !json.Valid(data) {
					return errors.New("legacy content contains invalid JSON; original data was retained")
				}
				var formatted bytes.Buffer
				if formatErr := json.Indent(&formatted, data, "", "  "); formatErr != nil {
					return fmt.Errorf("format legacy passport: %w", formatErr)
				}
				data = formatted.Bytes()
			}
			hash, err := files.Write(path, data)
			if err != nil {
				return fmt.Errorf("export legacy content: %w", err)
			}
			if _, err := files.Read(path, hash); err != nil {
				return fmt.Errorf("verify exported content: %w", err)
			}
			if _, err := tx.ExecContext(ctx, migration.update, path, hash, id); err != nil {
				return fmt.Errorf("save exported content reference: %w", err)
			}
			changed = true
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit content migration: %w", err)
	}
	if changed {
		if _, err := database.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			return fmt.Errorf("checkpoint content migration: %w", err)
		}
	}
	return nil
}
