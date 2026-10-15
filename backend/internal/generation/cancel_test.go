package generation

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

func TestCancelQueuedGeneration(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	ctx := t.Context()
	canceled, err := f.service.Cancel(ctx, f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != "failed" || canceled.ErrorCode != canceledErrorCode || canceled.FinishedAt == "" || canceled.StartedAt != "" {
		t.Fatalf("unexpected cancellation: %+v", canceled)
	}
	if canceled.Events[len(canceled.Events)-1].Type != "run_cancelled" { //nolint:misspell // Assert compatibility with existing event consumers.
		t.Fatal("missing cancellation event")
	}
	repeated, err := f.service.Cancel(ctx, f.run.ID)
	if err != nil || len(repeated.Events) != len(canceled.Events) {
		t.Fatalf("cancellation is not idempotent: %v", err)
	}
	if claimed, err := f.service.claimRun(ctx, f.run.ID); err != nil || claimed {
		t.Fatalf("canceled generation was claimed: %v", err)
	}
	if err := f.service.stage(ctx, f.run.ID, "collecting"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled generation was restarted: %v", err)
	}
	if err := f.service.saveRevision(ctx, f.run, passport.Passport{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled generation saved a revision: %v", err)
	}
	if _, err := f.service.Start(ctx, f.run.ProductID, Input{ConnectionID: f.run.ConnectionID, Model: f.run.Model}); err != nil {
		t.Fatalf("cancellation did not free the product: %v", err)
	}
}

type cancelledGenerator struct {
	started chan struct{}
	stopped chan struct{}
}

func (g *cancelledGenerator) Generate(ctx context.Context, _ llmgateway.Config, _ llm.Request) (llm.Result, error) {
	close(g.started)
	<-ctx.Done()
	close(g.stopped)
	return llm.Result{}, fmt.Errorf("generation canceled: %w", ctx.Err())
}

func TestCancelRunningGeneration(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	generator := &cancelledGenerator{started: make(chan struct{}), stopped: make(chan struct{})}
	f.service.LLM = generator
	ctx, stop := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); f.service.Work(ctx, 1) }()
	defer func() { stop(); <-done }()
	select {
	case <-generator.started:
	case <-time.After(10 * time.Second):
		t.Fatal("model request did not start")
	}
	canceled, err := f.service.Cancel(t.Context(), f.run.ID)
	if err != nil || canceled.ErrorCode != canceledErrorCode {
		t.Fatalf("cancel running generation: %v", err)
	}
	select {
	case <-generator.stopped:
	case <-time.After(10 * time.Second):
		t.Fatal("model request did not receive cancellation")
	}
	stop()
	<-done
	current, err := f.service.GetRun(t.Context(), f.run.ID)
	if err != nil || current.ErrorCode != canceledErrorCode || current.Status != "failed" {
		t.Fatalf("worker overwrote cancellation: %+v, %v", current, err)
	}
	history, err := f.service.History(t.Context(), f.run.ProductID)
	if err != nil || len(history) != 0 {
		t.Fatalf("canceled run published a passport: %v", err)
	}
}

func TestCancelPreservesFinishedGeneration(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	if err := f.service.process(t.Context(), f.run); err != nil {
		t.Fatal(err)
	}
	finished, err := f.service.Cancel(t.Context(), f.run.ID)
	if err != nil || finished.Status != "succeeded" || finished.ErrorCode != "" {
		t.Fatalf("cancellation changed successful generation: %+v, %v", finished, err)
	}
	history, err := f.service.History(t.Context(), f.run.ProductID)
	if err != nil || len(history) != 1 {
		t.Fatalf("previous passport was lost: %v", err)
	}
}
