package research

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestFailedModelRequestCannotAcceptPartialResearch(t *testing.T) {
	w, _, notes := budgetFixture(t)
	agent := func(local *workspace) fantasy.Agent {
		return scriptedResearchAgent{generate: func(_ context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
			if err := local.saveFinding(researchFinding{Name: "saved before request failure", Notes: notes}); err != nil {
				return nil, err
			}
			provider := &fantasy.ProviderError{StatusCode: 503, Title: "endpoint unavailable", Message: "request could not complete"}
			for attempt := range *call.MaxRetries {
				call.OnRetry(provider, 5*time.Second*time.Duration(1<<attempt))
			}
			return nil, &fantasy.RetryError{Errors: []error{provider, provider, provider, provider}}
		}}
	}
	var result Result
	retryVisible := false
	err := w.investigateParallel(t.Context(), 2, agent, llmgateway.Config{MaxConcurrent: 2}, "fixture", "", func(p Progress) error {
		retryVisible = retryVisible || strings.Contains(p.Message, "retry 3/3 in 20s. Reason: HTTP 503")
		return nil
	}, &result)
	provider, ok := errors.AsType[*fantasy.ProviderError](err)
	if !ok || provider.StatusCode != 503 || !strings.Contains(err.Error(), "after 4 attempts") || !retryVisible {
		t.Fatalf("model failure was hidden instead of returned: %v", err)
	}
	if w.checkedNotes != "" || len(w.ledger.Findings) == 0 {
		t.Fatal("failed research was accepted as partial or lost saved findings")
	}
}

func addSchedulerFiles(w *workspace, count int) {
	for i := range count {
		file := fmt.Sprintf("repo/tools/task%d.py", i)
		w.documents[file] = source.Document{ID: file, SnapshotID: "repo", Path: file[len("repo/"):], Content: "main()", LineCount: 1}
		w.paths = append(w.paths, file)
		w.inventory.Candidates = append(w.inventory.Candidates, Candidate{ID: file, Repository: "repo", Path: file, Kind: "entrypoint", Line: 1})
	}
	slices.Sort(w.paths)
}

func TestArchitectureResearchDoesNotTurnFileHintsIntoObligations(t *testing.T) {
	baseline, _, _ := coverageFixture()
	w, plan, notes := coverageFixture()
	addSchedulerFiles(w, 400)
	w.inventory.Candidates = append(w.inventory.Candidates, Candidate{ID: "tools", Repository: "repo", Path: "repo/tools", Kind: "source_area"})
	if len(w.researchTasks()) != len(baseline.researchTasks()) || len(w.pendingQuestions()) != len(baseline.pendingQuestions()) {
		t.Fatal("400 helper entrypoints expanded the architecture research queue")
	}
	if len(w.state().Areas) != len(baseline.state().Areas) || len(w.unfinishedFrontier()) != len(baseline.unfinishedFrontier()) {
		t.Fatal("optional file hints expanded progress or the saved frontier")
	}
	if !w.availablePath("repo/tools/task0.py") {
		t.Fatal("optional source was removed from investigation tools")
	}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: researchComplete}); err != nil {
		t.Fatalf("passport required reading/classifying every file hint: %v", err)
	}
	if err := w.setPlan(planInput{Candidates: []candidateDisposition{{CandidateID: "repo/tools/task0.py", Class: "product_code", Disposition: "deferred", Reason: "This tool is a runtime stage in the main product scenario"}}}); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(w.pendingQuestions(), func(q researchQuestion) bool { return q.Key == "repo/tools/task0.py" }) {
		t.Fatal("a required product question could not promote a file hint")
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: researchComplete}); err == nil {
		t.Fatal("explicitly unresolved product question was hidden by architecture sampling")
	}
}

func TestStalledTasksGetOneFreshClosureAttempt(t *testing.T) {
	task := researchTask{Key: "responsibility/repo/tools", Boundaries: []string{"tools"}, questions: []string{"tools/implementation", "tools/callers"}}
	sibling := researchTask{Key: "scope/repo/other", questions: []string{"other"}}
	attempts := map[string]taskAttempts{task.Key: {count: 2, stalled: 2, questions: "tools/implementation\x00tools/callers"}}
	queued := queueResearchTasks([]researchTask{task, sibling}, nil, attempts)
	if len(queued) != 2 {
		t.Fatal("stalled assignment was dropped without a fresh closure attempt")
	}
	attempt := attempts[task.Key]
	attempt.repaired = true
	attempts[task.Key] = attempt
	queued = queueResearchTasks([]researchTask{task, sibling}, nil, attempts)
	if len(queued) != 1 || queued[0].Key != sibling.Key {
		t.Fatal("stalled task blocked independent work or was retried unchanged")
	}
	task.questions = []string{"tools/callers"}
	if queued = queueResearchTasks([]researchTask{task}, nil, attempts); len(queued) != 1 {
		t.Fatal("changed assignment remained permanently stalled")
	}
	if attempts[task.Key].stalled != 0 {
		t.Fatal("new questions inherited the old failure count")
	}
	active := map[string]researchTask{"discovery": {Key: "scope/repo", Boundaries: []string{"tools"}}}
	if len(queueResearchTasks([]researchTask{task}, active, attempts)) != 0 {
		t.Fatal("two workers can write the same responsibility")
	}
}

func TestParallelRefillsBeforeTheSlowWorkerFinishes(t *testing.T) {
	w, plan, notes := coverageFixture()
	addSchedulerFiles(w, 4)
	for i := range 4 {
		file := fmt.Sprintf("repo/tools/task%d.py", i)
		plan.Candidates = append(plan.Candidates, candidateDisposition{CandidateID: file, Class: "product_code", Disposition: "deferred", Reason: "Explicit architecture question for the scheduling fixture"})
	}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	w.turnBudget = 64
	w.deadline = time.Now().Add(time.Minute)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	refilled := false
	agent := func(local *workspace) fantasy.Agent {
		return scriptedResearchAgent{generate: func(ctx context.Context, _ fantasy.AgentCall) (*fantasy.AgentResult, error) {
			if !slices.Contains(local.task.Boundaries, "execution") {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			if err := local.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
				return nil, err
			}
			return &fantasy.AgentResult{Steps: make([]fantasy.StepResult, taskTurns)}, nil
		}}
	}
	var result Result
	_ = w.investigateParallel(ctx, 2, agent, llmgateway.Config{MaxConcurrent: 2}, "fixture", "", func(p Progress) error {
		if !refilled && p.State != nil && len(p.State.Parallel.Workers) == 2 && p.State.Budget.Used == taskTurns && p.State.Budget.Limit == 64 && slices.ContainsFunc(p.State.Parallel.Workers, func(worker WorkerProgress) bool { return worker.ID == "research-3" }) {
			refilled = true
			cancel()
		}
		return nil
	}, &result)
	if !refilled {
		t.Fatal("free worker waited for the slow worker despite saved evidence and queued tasks")
	}
}

func TestDiscoveryCannotCreateAResponsibilityForEveryHelper(t *testing.T) {
	w, plan, _ := coverageFixture()
	local, err := w.forkTask(researchTask{Key: "scope/repo/internal/engine", Candidates: slices.Clone(w.inventory.Candidates)}, taskTurns)
	if err != nil {
		t.Fatal(err)
	}
	answer := answerInput{Responsibility: "engine", Repository: "repo", Role: "implementation", Evidence: plan.Boundaries[0].Implementation}
	if err := local.saveAnswer(answer); err != nil {
		t.Fatal(err)
	}
	if _, err := local.taskSnapshot(); err != nil {
		t.Fatal(err)
	}
	answer.Responsibility = "engine helper"
	if err := local.saveAnswer(answer); err == nil || len(local.plan) != 1 {
		t.Fatal("one source area created another four questions for a helper")
	}
	answer.Responsibility, answer.Role = "engine", "callers"
	answer.Evidence = plan.Boundaries[0].Callers
	if err := local.saveAnswer(answer); err != nil {
		t.Fatalf("one-summary rule blocked the remaining architecture evidence: %v", err)
	}
}

func TestOneClosedQuestionCannotBuyAFullWorkerWave(t *testing.T) {
	w, plan, _ := coverageFixture()
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	w.startParallelBudget(30)
	if err := w.saveAnswer(answerInput{Responsibility: "execution", Repository: "repo", Role: "implementation", Evidence: plan.Boundaries[0].Implementation}); err != nil {
		t.Fatal(err)
	}
	previous := w.activeLimit()
	if !w.refillParallelBudget() || w.activeLimit()-previous != taskTurns {
		t.Fatal("closing one question allocated another 480-turn research wave")
	}
}

func TestParallelRefillPreservesBudgetLimits(t *testing.T) {
	for _, reason := range []string{"no-progress", "hard-limit", "deadline", "restored-evidence"} {
		t.Run(reason, func(t *testing.T) {
			w, _, notes := budgetFixture(t)
			if reason == "hard-limit" {
				w.turnBudget = 32
			}
			if reason == "restored-evidence" {
				if err := w.saveFinding(researchFinding{Name: "restored", Notes: notes}); err != nil {
					t.Fatal(err)
				}
			}
			w.startParallelBudget(2)
			if reason != "no-progress" {
				if err := w.saveFinding(researchFinding{Name: "saved", Notes: notes}); err != nil {
					t.Fatal(err)
				}
			}
			if reason == "deadline" {
				w.deadline = time.Now()
			}
			limit := w.activeLimit()
			if w.refillParallelBudget() || w.activeLimit() != limit {
				t.Fatalf("refill bypassed %s", reason)
			}
			if reason == "no-progress" && (w.budget.recovery || w.budget.stopped) {
				t.Fatal("reservation pressure spent recovery before running workers could finish")
			}
		})
	}
}
