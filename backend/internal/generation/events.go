package generation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Event is a durable, user-facing fact about a generation run.
// Structured research records are decoded into Run.Research by GetRun; clients
// receive short activity messages alongside that separate progress state.
type Event struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type eventWriter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func recordEvent(ctx context.Context, writer eventWriter, runID, kind, stage, message string) error {
	_, err := writer.ExecContext(ctx, "INSERT INTO generation_events(run_id,type,stage,message,created_at) VALUES(?,?,?,?,?)", runID, kind, stage, message, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record generation event: %w", err)
	}
	return nil
}

func (s *Service) event(ctx context.Context, runID, kind, stage, message string) error {
	return recordEvent(ctx, s.DB, runID, kind, stage, message)
}

func (s *Service) stage(ctx context.Context, id, stage string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin generation stage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var previous, status string
	if err := tx.QueryRowContext(ctx, "SELECT stage,status FROM generation_runs WHERE id=?", id).Scan(&previous, &status); err != nil {
		return fmt.Errorf("read generation stage: %w", err)
	}
	if status != "queued" && status != "running" {
		return context.Canceled
	}
	if previous == stage {
		return nil
	}
	if previous != "queued" {
		kind, message := "stage_completed", "Stage completed."
		if previous == "validating" && stage == "repairing" {
			kind, message = "validation_failed", "Validation found issues. Repairing the passport."
		}
		if err := recordEvent(ctx, tx, id, kind, previous, message); err != nil {
			return err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "UPDATE generation_runs SET status='running',stage=?,stage_started_at=?,started_at=CASE WHEN started_at='' THEN ? ELSE started_at END WHERE id=?", stage, now, now, id); err != nil {
		return fmt.Errorf("update generation stage: %w", err)
	}
	if err := recordEvent(ctx, tx, id, "stage_started", stage, "Stage started."); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generation stage: %w", err)
	}
	return nil
}

func (s *Service) failRun(ctx context.Context, id, code, message string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin generation failure: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status, stage string
	if err := tx.QueryRowContext(ctx, "SELECT status,stage FROM generation_runs WHERE id=?", id).Scan(&status, &stage); err != nil {
		return fmt.Errorf("read failed generation: %w", err)
	}
	if status == "succeeded" || status == "failed" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, "UPDATE generation_runs SET status='failed',error_code=?,error_message=?,finished_at=? WHERE id=?", code, message, time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
		return fmt.Errorf("save generation failure: %w", err)
	}
	kind, eventMessage := "run_failed", "Generation stopped. See the error above for details."
	if code == "cancelled" {
		kind, eventMessage = "run_cancelled", "Generation cancelled."
	}
	if err := recordEvent(ctx, tx, id, kind, stage, eventMessage); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generation failure: %w", err)
	}
	return nil
}

func (s *Service) recoverRuns(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, "SELECT id FROM generation_runs WHERE status='running'")
	if err != nil {
		return fmt.Errorf("list interrupted generations: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("read interrupted generation: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read interrupted generations: %w", err)
	}
	// Release the single SQLite connection before recording failures.
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close interrupted generations: %w", err)
	}
	for _, id := range ids {
		if err := s.failRun(ctx, id, "interrupted", "The server stopped during generation. Start a new run."); err != nil {
			return err
		}
	}
	return nil
}
