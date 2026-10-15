package generation

import (
	"context"
	"fmt"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
)

// DeleteProduct removes a product and all its passport and generation data.
// Active workers must finish before their files can be safely removed.
func (s *Service) DeleteProduct(ctx context.Context, id string) (product.Product, error) {
	deleted, err := s.Products.Get(ctx, id)
	if err != nil {
		return deleted, fmt.Errorf("get product to delete: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return product.Product{}, fmt.Errorf("begin product deletion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if checkErr := tx.QueryRowContext(ctx, "SELECT id FROM products WHERE id=?", id).Scan(&deleted.ID); checkErr != nil {
		return deleted, fmt.Errorf("read product to delete: %w", checkErr)
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,status IN ('queued','running') FROM generation_runs WHERE product_id=?", id)
	if err != nil {
		return deleted, fmt.Errorf("list product generations: %w", err)
	}
	defer rows.Close()
	var runIDs []string
	for rows.Next() {
		var runID string
		var pending bool
		if err := rows.Scan(&runID, &pending); err != nil {
			_ = rows.Close()
			return deleted, fmt.Errorf("read product generation: %w", err)
		}
		_, working := s.active.Load(runID)
		if pending || working {
			_ = rows.Close()
			return deleted, fmt.Errorf("%w: stop generation and wait for it to finish before deleting the passport", ErrActive)
		}
		runIDs = append(runIDs, runID)
	}
	readErr := rows.Err()
	_ = rows.Close()
	if readErr != nil {
		return deleted, fmt.Errorf("read product generations: %w", readErr)
	}
	// Events and analysis checkpoints are removed by their foreign-key cascades.
	for _, query := range []string{
		"DELETE FROM goqite WHERE queue='passport' AND body IN (SELECT CAST(id AS BLOB) FROM generation_runs WHERE product_id=?)",
		"DELETE FROM passport_revisions WHERE product_id=?",
		"DELETE FROM source_documents WHERE snapshot_id IN (SELECT id FROM source_snapshots WHERE run_id IN (SELECT id FROM generation_runs WHERE product_id=?))",
		"DELETE FROM source_snapshots WHERE run_id IN (SELECT id FROM generation_runs WHERE product_id=?)",
		"DELETE FROM llm_calls WHERE run_id IN (SELECT id FROM generation_runs WHERE product_id=?)",
		"DELETE FROM generation_runs WHERE product_id=?",
		"DELETE FROM products WHERE id=?",
	} {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return deleted, fmt.Errorf("delete product data: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return deleted, fmt.Errorf("commit product deletion: %w", err)
	}
	// Only remove files after the metadata transaction succeeds.
	paths := []string{"products/" + id}
	for _, runID := range runIDs {
		paths = append(paths, "runs/"+runID)
	}
	for _, path := range paths {
		if err := s.Content.RemoveAll(path); err != nil {
			s.Logger.Error().Err(err).Str("path", path).Msg("clean up deleted product content")
		}
	}
	return deleted, nil
}
