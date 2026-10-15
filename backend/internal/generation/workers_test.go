package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
	"maragu.dev/goqite"
)

type blockingCollector struct {
	started chan string
	release chan struct{}
}

func (c *blockingCollector) Collect(ctx context.Context, inputs []source.Input) (source.Collection, error) {
	select {
	case c.started <- inputs[0].Text:
	case <-ctx.Done():
		return source.Collection{}, fmt.Errorf("start worker fixture: %w", ctx.Err())
	}
	select {
	case <-c.release:
	case <-ctx.Done():
		return source.Collection{}, fmt.Errorf("wait for worker fixture: %w", ctx.Err())
	}
	// Finish at collection so this queue test does not need model credentials.
	return source.Collection{}, errors.New("worker fixture finished")
}

//nolint:gocyclo // Pool contract covers concurrency, queued work, startup recovery and duplicate delivery.
func TestWorkerPool(t *testing.T) {
	t.Parallel()
	for _, workers := range []int{1, 3} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			database, setupErr := db.Open(ctx, filepath.Join(t.TempDir(), "workers.db"))
			if setupErr != nil {
				t.Fatal(setupErr)
			}
			defer database.Close()
			if err := db.RunMigrations(database); err != nil {
				t.Fatal(err)
			}
			// Only metadata is needed: collection stops before credential lookup.
			if _, err := database.ExecContext(ctx, `
INSERT INTO llm_connections(id,name,created_at) VALUES('fixture','Fixture','2026-10-15T00:00:00Z');
INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at)
VALUES('fixture',1,X'00','2026-10-15T00:00:00Z');`); err != nil {
				t.Fatal(err)
			}
			collector := &blockingCollector{started: make(chan string, workers+1), release: make(chan struct{})}
			service := New(database, &llmgateway.Store{}, nil, collector, zerolog.Nop(), nil)
			var runs []Run
			var interruptedID string
			for index := range workers + 2 {
				name := fmt.Sprintf("Product %d", index)
				inputs := []source.Input{{Kind: "text", Text: name + " stores requests."}}
				p, err := service.Products.Create(ctx, product.Input{Name: name, Sources: inputs})
				if err != nil {
					t.Fatal(err)
				}
				manifest, err := json.Marshal(inputs)
				if err != nil {
					t.Fatal(err)
				}
				status := "queued"
				if index == 0 {
					status, interruptedID = "running", name
				} else {
					runs = append(runs, Run{ID: name})
				}
				if _, err := database.ExecContext(ctx, `INSERT INTO generation_runs
(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at)
VALUES(?,?,'fixture',1,'fixture',?,?,?,'2026-10-15T00:00:00Z')`, name, p.ID, status, status, string(manifest)); err != nil {
					t.Fatal(err)
				}
				if err := service.queue.Send(ctx, goqite.Message{Body: []byte(name)}); err != nil {
					t.Fatal(err)
				}
			}
			// A duplicate delivery must not cause a second execution of a run.
			if err := service.queue.Send(ctx, goqite.Message{Body: []byte(runs[0].ID)}); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() { defer close(done); service.Work(ctx, workers) }()
			defer func() { cancel(); <-done }()
			seen := make(map[string]bool)
			awaitStart := func() {
				t.Helper()
				select {
				case name := <-collector.started:
					if seen[name] {
						t.Fatalf("run started twice: %s", name)
					}
					seen[name] = true
				case <-time.After(10 * time.Second):
					t.Fatal("worker did not start a queued run")
				}
			}
			for range workers {
				awaitStart()
			}
			counts := make(map[string]int)
			for _, run := range runs {
				current, err := service.GetRun(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				counts[current.Status]++
			}
			if counts["running"] != workers || counts["queued"] != 1 {
				t.Fatalf("unexpected pool occupancy: %v", counts)
			}
			interrupted, err := service.GetRun(ctx, interruptedID)
			if err != nil || interrupted.Status != "failed" || interrupted.ErrorCode != "interrupted" {
				t.Fatalf("startup recovery: status=%s code=%s err=%v", interrupted.Status, interrupted.ErrorCode, err)
			}
			// Free one worker and verify that it picks up the waiting run.
			collector.release <- struct{}{}
			awaitStart()
			close(collector.release)
			for _, run := range runs {
				finished := waitForRun(t, service, run.ID)
				if finished.Status != "failed" || !strings.Contains(finished.ErrorMessage, "worker fixture finished") {
					t.Fatalf("unexpected completion: status=%s error=%s", finished.Status, finished.ErrorMessage)
				}
			}
		})
	}
}
