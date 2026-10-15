package generation

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrApprovalRequired prevents publishing an unreviewed version.
var ErrApprovalRequired = errors.New("approve this passport version before publishing")

// Approve records the first approval of this immutable version.
func (s *Service) Approve(ctx context.Context, id, userID string) (Revision, error) {
	_, err := s.DB.ExecContext(ctx, "UPDATE passport_revisions SET approved_by=?,approved_at=? WHERE id=? AND approved_at=''", userID, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return Revision{}, fmt.Errorf("approve passport: %w", err)
	}
	return s.Revision(ctx, id)
}

// Publish changes the reader version without changing any saved content.
func (s *Service) Publish(ctx context.Context, id, userID string) (Revision, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Revision{}, fmt.Errorf("begin publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var productID, approvedAt string
	if err := tx.QueryRowContext(ctx, "SELECT product_id,approved_at FROM passport_revisions WHERE id=?", id).Scan(&productID, &approvedAt); err != nil {
		return Revision{}, fmt.Errorf("read publication version: %w", err)
	}
	if approvedAt == "" {
		return Revision{}, ErrApprovalRequired
	}
	result, err := tx.ExecContext(ctx, "UPDATE products SET published_revision_id=? WHERE id=? AND COALESCE(published_revision_id,'')<>?", id, productID, id)
	if err != nil {
		return Revision{}, fmt.Errorf("publish passport: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Revision{}, fmt.Errorf("read publication result: %w", err)
	}
	if changed > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE passport_revisions SET published_by=?,published_at=? WHERE id=?", userID, time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
			return Revision{}, fmt.Errorf("record publication: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Revision{}, fmt.Errorf("commit publication: %w", err)
	}
	return s.Revision(ctx, id)
}
