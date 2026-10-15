package generation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/analysis"
)

func (s *Service) loadAnalysisCheckpoint(ctx context.Context, runID string) (*analysis.Checkpoint, error) {
	var path, hash string
	var revision int
	err := s.DB.QueryRowContext(ctx, "SELECT revision,content_path,content_hash FROM analysis_checkpoints WHERE run_id=? ORDER BY revision DESC LIMIT 1", runID).Scan(&revision, &path, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load canonical analysis checkpoint reference: %w", err)
	}
	var checkpoint analysis.Checkpoint
	if err := s.Content.ReadJSON(path, hash, &checkpoint); err != nil {
		return nil, fmt.Errorf("verify canonical analysis checkpoint: %w", err)
	}
	if checkpoint.Version != analysis.CheckpointVersion || checkpoint.Revision != revision {
		return nil, errors.New("canonical analysis checkpoint version or revision does not match its durable reference")
	}
	// Runner/NewModelStore additionally verifies the source fingerprint,
	// every read span and every retained source ID against the full collection.
	return &checkpoint, nil
}

func (s *Service) saveAnalysisCheckpoint(ctx context.Context, runID string, checkpoint analysis.Checkpoint, state AnalysisState) error {
	if checkpoint.Version != analysis.CheckpointVersion || checkpoint.Revision < 1 {
		return errors.New("invalid canonical analysis checkpoint version or revision")
	}
	// A random suffix permits retry after a file was published but SQLite
	// rolled back. No later attempt can collide with an orphaned immutable file.
	path := fmt.Sprintf("runs/%s/analysis/checkpoints/%06d-%s.json", runID, checkpoint.Revision, rand.Text())
	hash, err := s.Content.WriteJSON(path, checkpoint)
	if err != nil {
		return fmt.Errorf("write canonical analysis checkpoint: %w", err)
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode durable analysis state: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin canonical analysis checkpoint: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var latest int
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(revision),0) FROM analysis_checkpoints WHERE run_id=?", runID).Scan(&latest); err != nil {
		return fmt.Errorf("read previous analysis checkpoint: %w", err)
	}
	if checkpoint.Revision != latest+1 {
		return fmt.Errorf("canonical analysis checkpoint revision %d must follow durable revision %d", checkpoint.Revision, latest)
	}
	var status string
	if err := tx.QueryRowContext(ctx, "SELECT status FROM generation_runs WHERE id=?", runID).Scan(&status); err != nil {
		return fmt.Errorf("read generation before saving analysis: %w", err)
	}
	if status != "running" && status != "queued" {
		return context.Canceled
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO analysis_checkpoints(run_id,revision,content_path,content_hash,created_at) VALUES(?,?,?,?,?)", runID, checkpoint.Revision, path, hash, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("save canonical analysis checkpoint reference: %w", err)
	}
	if err := recordEvent(ctx, tx, runID, "analysis_progress", "analyzing", string(encoded)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit canonical analysis checkpoint: %w", err)
	}
	return nil
}
