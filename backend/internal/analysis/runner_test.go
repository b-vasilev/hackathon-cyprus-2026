package analysis

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
)

func TestSynthesisRunsIndependentStepsTogetherBeforeReconciliation(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	store := &ModelStore{state: Checkpoint{Tasks: SynthesisTasks()}, persist: func(context.Context, Checkpoint) error { return nil }}
	started := make(chan string, 2)
	release := make(chan struct{})
	finished := make(chan error, 1)
	var mu sync.Mutex
	var completed []string
	go func() {
		finished <- runSynthesisPlan(ctx, SynthesisTasks(), func(ctx context.Context, id string) error {
			if err := store.SetTaskStatus(ctx, id, TaskActive); err != nil {
				return err
			}
			if id != ReconciliationTaskID {
				started <- id
				select {
				case <-release:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if id == ReconciliationTaskID && len(completed) != 2 {
				return errors.New("reconciliation ran before both synthesis steps finished")
			}
			completed = append(completed, id)
			return store.SetTaskStatus(ctx, id, TaskDone)
		})
	}()
	awaitStart := func() string {
		t.Helper()
		select {
		case id := <-started:
			return id
		case err := <-finished:
			t.Fatalf("synthesis stopped before both workers started: %v", err)
		case <-ctx.Done():
			t.Fatal("independent synthesis steps did not start together")
		}
		return ""
	}
	first, second := awaitStart(), awaitStart()
	if err := store.SetTaskStatus(ctx, ReconciliationTaskID, TaskActive); err == nil {
		t.Fatal("reconciliation gained access while synthesis workers were active")
	}
	before := store.AreaRevision(ArchitectureTaskID)
	if err := store.SetTaskStatus(ctx, PassportTaskID, TaskPending); err != nil {
		t.Fatal(err)
	}
	if err := store.SetTaskStatus(ctx, PassportTaskID, TaskActive); err != nil {
		t.Fatal(err)
	}
	if store.AreaRevision(ArchitectureTaskID) != before {
		t.Fatal("peer writes were counted as progress by a stalled worker")
	}
	close(release)
	if first == second {
		t.Fatal("the same synthesis step ran twice")
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if len(completed) != 3 || completed[2] != ReconciliationTaskID {
		t.Fatalf("unexpected completion order: %v", completed)
	}
}

func TestSynthesisRetrySkipsCompletedStepsAndDoesNotReconcileFailure(t *testing.T) {
	tasks := SynthesisTasks()
	tasks[0].Status = TaskDone
	failure := errors.New("save rejected")
	var calls []string
	err := runSynthesisPlan(t.Context(), tasks, func(_ context.Context, id string) error {
		calls = append(calls, id)
		return failure
	})
	if !errors.Is(err, failure) || !slices.Equal(calls, []string{tasks[1].ID}) {
		t.Fatalf("retry repeated accepted work or reconciled a failed step: %v, %v", calls, err)
	}
}

func TestAnalysisWindowsRetainResearchAcrossTurns(t *testing.T) {
	previous := []fantasy.Message{fantasy.NewUserMessage("Previously read research batch")}
	steps := []fantasy.StepResult{
		{Messages: []fantasy.Message{fantasy.NewUserMessage("Research citations")}},
		{Messages: []fantasy.Message{fantasy.NewUserMessage("Saved model IDs")}},
	}
	retained := recentAnalysisMessages(previous, steps)
	if len(retained) != 3 {
		t.Fatalf("window discarded research or model state: %d messages", len(retained))
	}
}
