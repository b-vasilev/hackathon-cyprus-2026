package generation

import (
	"context"
	"errors"
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
	cancelled, err := f.service.Cancel(ctx, f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "failed" || cancelled.ErrorCode != "cancelled" || cancelled.FinishedAt == "" || cancelled.StartedAt != "" {
		t.Fatalf("unexpected cancellation: %+v", cancelled)
	}
	if cancelled.Events[len(cancelled.Events)-1].Type != "run_cancelled" {
		t.Fatal("missing cancellation event")
	}
	repeated, err := f.service.Cancel(ctx, f.run.ID)
	if err != nil || len(repeated.Events) != len(cancelled.Events) {
		t.Fatalf("cancellation is not idempotent: %v", err)
	}
	if claimed, err := f.service.claimRun(ctx, f.run.ID); err != nil || claimed {
		t.Fatalf("cancelled generation was claimed: %v", err)
	}
	if err := f.service.stage(ctx, f.run.ID, "collecting"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled generation was restarted: %v", err)
	}
	if err := f.service.saveRevision(ctx, f.run, passport.Passport{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled generation saved a revision: %v", err)
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
	return llm.Result{}, ctx.Err()
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
	cancelled, err := f.service.Cancel(t.Context(), f.run.ID)
	if err != nil || cancelled.ErrorCode != "cancelled" {
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
	if err != nil || current.ErrorCode != "cancelled" || current.Status != "failed" {
		t.Fatalf("worker overwrote cancellation: %+v, %v", current, err)
	}
	history, err := f.service.History(t.Context(), f.run.ProductID)
	if err != nil || len(history) != 0 {
		t.Fatalf("cancelled run published a passport: %v", err)
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
