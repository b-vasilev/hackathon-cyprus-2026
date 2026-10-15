package generation

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestDeleteProductRemovesPassportData(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	ctx := t.Context()
	if err := f.service.process(ctx, f.run); err != nil {
		t.Fatal(err)
	}
	other, err := f.service.Products.Create(ctx, product.Input{Name: "Keep this product", Sources: []source.Input{{Kind: "text", Text: "Keep this source."}}})
	if err != nil {
		t.Fatal(err)
	}
	otherPath := "products/" + other.ID + "/keep.json"
	hash, err := f.service.Content.WriteJSON(otherPath, map[string]string{"keep": "yes"})
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := f.service.DB.ExecContext(ctx, "CREATE TRIGGER prevent_product_delete BEFORE DELETE ON products BEGIN SELECT RAISE(ABORT, 'blocked'); END"); checkErr != nil {
		t.Fatal(checkErr)
	}
	if _, checkErr := f.service.DeleteProduct(ctx, f.run.ProductID); checkErr == nil {
		t.Fatal("deletion should roll back when a database write fails")
	}
	history, err := f.service.History(ctx, f.run.ProductID)
	if err != nil || len(history) != 1 {
		t.Fatalf("rollback lost passport metadata: %v", err)
	}
	if _, err := f.service.Revision(ctx, history[0].ID); err != nil {
		t.Fatalf("rollback lost passport content: %v", err)
	}
	if _, err := f.service.DB.ExecContext(ctx, "DROP TRIGGER prevent_product_delete"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.DeleteProduct(ctx, f.run.ProductID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"passport_revisions", "generation_runs", "source_snapshots", "source_documents", "llm_calls", "generation_events", "analysis_checkpoints", "goqite"} {
		var count int
		if err := f.service.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s: remaining rows=%d, err=%v", table, count, err)
		}
	}
	for _, path := range []string{"products/" + f.run.ProductID, "runs/" + f.run.ID} {
		if _, err := os.Stat(filepath.Join(f.service.Content.Dir, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("deleted content remains at %s: %v", path, err)
		}
	}
	if _, err := f.service.Products.Get(ctx, other.ID); err != nil {
		t.Fatalf("unrelated product lost: %v", err)
	}
	if _, err := f.service.Content.Read(otherPath, hash); err != nil {
		t.Fatalf("unrelated content lost: %v", err)
	}
	if _, err := f.service.Connections.Load(ctx, f.run.ConnectionID, 1); err != nil {
		t.Fatalf("shared connection lost: %v", err)
	}
	if _, err := f.service.DeleteProduct(ctx, f.run.ProductID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing product deletion: %v", err)
	}
}

func TestDeleteProductRejectsActiveGeneration(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"queued", "running", "draining"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			f := newRetryFixture(t)
			if state == "running" {
				if _, err := f.service.claimRun(t.Context(), f.run.ID); err != nil {
					t.Fatal(err)
				}
			}
			if state == "draining" {
				if _, err := f.service.Cancel(t.Context(), f.run.ID); err != nil {
					t.Fatal(err)
				}
				f.service.active.Store(f.run.ID, context.CancelFunc(func() {}))
			}
			if _, err := f.service.DeleteProduct(t.Context(), f.run.ProductID); !errors.Is(err, ErrActive) {
				t.Fatalf("active product deletion: %v", err)
			}
			if _, err := f.service.Products.Get(t.Context(), f.run.ProductID); err != nil {
				t.Fatalf("active product lost: %v", err)
			}
		})
	}
}

func TestDeleteConnectionPreservesPassport(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	ctx := t.Context()
	if err := f.service.process(ctx, f.run); err != nil {
		t.Fatal(err)
	}
	updated, err := f.service.Connections.Save(ctx, f.run.ConnectionID, f.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := f.service.Connections.Delete(ctx, updated.ID); checkErr != nil {
		t.Fatal(checkErr)
	}
	connections, err := f.service.Connections.List(ctx)
	if err != nil || len(connections) != 0 {
		t.Fatalf("deleted connection still listed: %v", err)
	}
	for _, revision := range []int{1, updated.Revision} {
		if _, checkErr := f.service.Connections.Load(ctx, updated.ID, revision); !errors.Is(checkErr, sql.ErrNoRows) {
			t.Fatalf("deleted connection can be loaded: %v", checkErr)
		}
	}
	var secrets int
	if checkErr := f.service.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM llm_connection_revisions WHERE connection_id=? AND length(encrypted_config)>0", updated.ID).Scan(&secrets); checkErr != nil || secrets != 0 {
		t.Fatalf("saved credentials remain: %d, %v", secrets, checkErr)
	}
	if _, checkErr := f.service.Connections.Save(ctx, updated.ID, f.config); !errors.Is(checkErr, sql.ErrNoRows) {
		t.Fatalf("deleted connection can be restored by editing: %v", checkErr)
	}
	if _, checkErr := f.service.Start(ctx, f.run.ProductID, Input{ConnectionID: updated.ID, Model: f.run.Model}); !errors.Is(checkErr, sql.ErrNoRows) {
		t.Fatalf("deleted connection can start a run: %v", checkErr)
	}
	history, err := f.service.History(ctx, f.run.ProductID)
	if err != nil || len(history) != 1 {
		t.Fatalf("passport history lost: %v", err)
	}
	if _, err := f.service.Revision(ctx, history[0].ID); err != nil {
		t.Fatalf("passport lost: %v", err)
	}
	if _, err := f.service.GetRun(ctx, f.run.ID); err != nil {
		t.Fatalf("generation history lost: %v", err)
	}
}
