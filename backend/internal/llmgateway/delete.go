package llmgateway

import (
	"context"
	"database/sql"
	"fmt"
)

// Delete erases saved credentials while keeping reference IDs for run history.
func (s *Store) Delete(ctx context.Context, id string) (Connection, error) {
	deleted, err := s.Get(ctx, id)
	if err != nil {
		return Connection{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Connection{}, fmt.Errorf("begin connection deletion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var active bool
	if checkErr := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM generation_runs WHERE connection_id=? AND status IN ('queued','running'))", id).Scan(&active); checkErr != nil {
		return Connection{}, fmt.Errorf("check connection generations: %w", checkErr)
	}
	if active {
		return Connection{}, ErrInUse
	}
	result, err := tx.ExecContext(ctx, "UPDATE llm_connections SET deleted=1,enabled=0 WHERE id=? AND deleted=0", id)
	if err != nil {
		return Connection{}, fmt.Errorf("delete connection: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Connection{}, fmt.Errorf("check connection deletion: %w", err)
	}
	if count == 0 {
		return Connection{}, sql.ErrNoRows
	}
	// Old runs retain their foreign keys, but no usable configuration or secrets.
	if _, err := tx.ExecContext(ctx, "UPDATE llm_connection_revisions SET encrypted_config=X'' WHERE connection_id=?", id); err != nil {
		return Connection{}, fmt.Errorf("erase connection credentials: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Connection{}, fmt.Errorf("commit connection deletion: %w", err)
	}
	deleted.Config.Enabled = false
	deleted.HasSecret = false
	return deleted, nil
}
