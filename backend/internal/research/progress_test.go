package research

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func budgetFixture(t *testing.T) (*workspace, planInput, string) {
	t.Helper()
	w, plan, notes := coverageFixture()
	plan.Candidates[0].Disposition = "deferred"
	plan.Candidates[0].Reason = "A reachable independent branch remains"
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	w.turnBudget = 120
	w.deadline = time.Now().Add(30 * time.Minute)
	w.event = func(string) error { return nil }
	w.startBudget()
	return w, plan, notes
}

func TestVisiblePlanRequiresReadAndCitedEvidence(t *testing.T) {
	w, _, notes := budgetFixture(t)
	state := w.state()
	if state.Areas[0].Status == "checked" || reachableQuestions(state) == 0 {
		t.Fatal("planned/read evidence without saved citations was marked checked")
	}
	if err := w.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	state = w.state()
	checked := 0
	for _, area := range state.Areas {
		if area.Status == "checked" {
			checked++
			if area.Questions[0].Result == "" || len(area.Questions[0].Evidence) == 0 {
				t.Fatal("checked area has no visible answer or evidence")
			}
		}
	}
	if checked != 1 || reachableQuestions(state) != 1 {
		t.Fatalf("incorrect current plan: checked=%d reachable=%d", checked, reachableQuestions(state))
	}
	w.reads["AAAAAAAAAAAAAAAAAAAAAAAAAA"] = []source.LineRange{{StartLine: 1, EndLine: 2}}
	if areaStatus(w.state().Areas[0].Questions) == "checked" {
		t.Fatal("unread implementation became checked")
	}
}

func TestExclusionsCannotHideAnUnresearchedRepository(t *testing.T) {
	w, _, _ := coverageFixture()
	for _, c := range w.inventory.Candidates {
		w.ledger.Plan.Candidates = append(w.ledger.Plan.Candidates, candidateDisposition{CandidateID: c.ID, Disposition: "excluded", Reason: "No product role established yet"})
	}
	if reachableQuestions(w.state()) != 1 || len(w.pendingQuestions()) != 1 {
		t.Fatal("excluded discovery scopes hid the missing repository investigation")
	}
}

func TestBudgetExtendsForEvidenceAndStopsAfterOneRecovery(t *testing.T) {
	w, _, notes := budgetFixture(t)
	if err := w.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	w.turn = 30
	if !w.reviewBudget() || w.activeLimit() != 40 || w.budget.decisions[1].Kind != "extended" {
		t.Fatal("new verified evidence did not extend the allowance")
	}
	// Renaming and repeating the same source-backed finding must not earn turns.
	w.plan[0].Name = "renamed execution"
	if err := w.saveFinding(researchFinding{Name: "same facts again", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	w.turn = 40
	if !w.reviewBudget() || w.activeLimit() != 50 || w.budget.decisions[2].Kind != "recovery" {
		t.Fatal("bookkeeping was credited as research progress")
	}
	w.turn = 50
	if w.reviewBudget() || !w.budget.stopped || w.activeLimit() != 50 {
		t.Fatal("repeated no-progress windows were extended indefinitely")
	}
	if allowed, reason := w.partialStopReason(""); !allowed || !strings.Contains(reason, "recovery") {
		t.Fatalf("stopping reason is not explicit: %v %q", allowed, reason)
	}
}

func TestBudgetDoesNotCreditRestoredEvidenceAndHonorsHardLimits(t *testing.T) {
	for _, limit := range []string{"restored", "turns", "time", "unavailable"} {
		t.Run(limit, func(t *testing.T) {
			w, plan, notes := budgetFixture(t)
			if err := w.saveFinding(researchFinding{Name: "existing", Notes: notes}); err != nil {
				t.Fatal(err)
			}
			w.startBudget()
			w.turn = 30
			switch limit {
			case "restored":
				if !w.reviewBudget() || w.budget.decisions[1].Kind != "recovery" {
					t.Fatal("restored evidence earned a fresh progress extension")
				}
				return
			case "turns":
				w.turnBudget = 30
			case "time":
				w.deadline = time.Now().Add(20 * time.Second)
			case "unavailable":
				plan.Candidates[0].Disposition = "included"
				if err := w.setPlan(plan); err != nil {
					t.Fatal(err)
				}
			}
			if w.reviewBudget() || w.budget.decisions[1].Kind != "stopped" {
				t.Fatal("hard limit or exhausted question set was ignored")
			}
		})
	}
}

func TestNewEvidenceWithoutFewerQuestionsStopsWithPartialPlan(t *testing.T) {
	w, plan, notes := budgetFixture(t)
	turns := 0
	agent := scriptedResearchAgent{generate: func(ctx context.Context, call fantasy.AgentCall) (response *fantasy.AgentResult, resultErr error) {
		defer func() {
			if resultErr != nil {
				t.Errorf("scripted research turn %d: %v", turns, resultErr)
			}
		}()
		var steps []fantasy.StepResult
		for {
			_, _, err := call.PrepareStep(ctx, fantasy.PrepareStepFunctionOptions{StepNumber: len(steps), Steps: steps})
			if err != nil {
				return nil, fmt.Errorf("prepare budget test step: %w", err)
			}
			turns++
			if turns == 1 {
				if err := w.saveFinding(researchFinding{Name: "initial", Notes: notes}); err != nil {
					return nil, err
				}
			}
			start := map[int]int{31: 3, 41: 2, 51: 1}[turns]
			if start != 0 {
				plan.Boundaries[0].Implementation[0].StartLine = start
				plan.Boundaries[0].Implementation[0].EndLine = 6
				if err := w.setPlan(plan); err != nil {
					return nil, err
				}
				extra := map[int]string{3: "3", 2: "2", 1: "1"}[start]
				if err := w.saveFinding(researchFinding{Name: "expanded", Notes: "Additional implementation evidence [AAAAAAAAAAAAAAAAAAAAAAAAAA:L" + extra + "-L6]"}); err != nil {
					return nil, err
				}
			}
			steps = append(steps, scriptedStep(2, 1).Steps[0])
			if call.StopWhen[0](steps) {
				return &fantasy.AgentResult{Steps: steps}, nil
			}
			if turns > 120 {
				t.Fatal("research exceeded the hard limit")
			}
		}
	}}
	var result Result
	var last *State
	err := w.investigate(t.Context(), agent, llmgateway.Config{}, "fixture", "", func(p Progress) error { last = p.State; return nil }, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps != 50 || result.InputTokens != 100 || last == nil || last.Status != researchPartial || last.Budget.Limit != 50 {
		t.Fatalf("new lines kept flat-frontier research running: %+v state=%+v", result, last)
	}
	if reachableQuestions(*last) != 1 || len(w.ledger.Frontier) == 0 || !strings.Contains(w.ledger.StopReason, "converge") {
		t.Fatal("stopping lost the open question or convergence reason")
	}
}

func TestGrowingFrontierCannotEarnExtensions(t *testing.T) {
	w, plan, notes := budgetFixture(t)
	if err := w.saveFinding(researchFinding{Name: "initial", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	w.startBudget() // One existing question; source evidence already credited.
	helper := plan.Boundaries[0]
	helper.Name = "helper"
	helper.Implementation, helper.Callers, helper.Contracts, helper.Validation = nil, nil, nil, nil
	if err := w.setPlan(planInput{Boundaries: []researchBoundary{helper}}); err != nil {
		t.Fatal(err)
	}
	w.turn = 30
	if !w.reviewBudget() || w.budget.decisions[1].Kind != "recovery" || w.budget.pending != 1 {
		t.Fatal("growing the frontier earned an extension or raised its baseline")
	}
	// Answer a new question with fresh source lines. The frontier is smaller
	// than at recovery, but still larger than before expansion.
	if err := w.saveAnswer(answerInput{Responsibility: "helper", Repository: "repo", Role: "implementation", Evidence: []roleEvidence{{Path: "repo/internal/engine/run.go", StartLine: 1, EndLine: 6, Finding: "Helper implementation"}}}); err != nil {
		t.Fatal(err)
	}
	w.turn = 40
	if w.reviewBudget() || !w.budget.stopped || w.activeLimit() != 40 {
		t.Fatal("new evidence on newly created questions extended a growing frontier")
	}
}

func TestParallelRecoveryIsShort(t *testing.T) {
	w, _, _ := budgetFixture(t)
	w.turnBudget = 0
	w.startParallelBudget(defaultParallelism)
	w.turn = w.activeLimit()
	if !w.reviewBudget() || w.activeLimit()-w.turn != 30 || w.budget.decisions[1].Kind != "recovery" {
		t.Fatal("recovery allocated another full worker-pool allowance")
	}
}

func TestNoEvidenceProgressStopsWithAnHonestPartialPlan(t *testing.T) {
	w, _, notes := budgetFixture(t)
	if err := w.saveFinding(researchFinding{Name: "previous evidence", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	w.startBudget()
	agent := scriptedResearchAgent{generate: func(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
		var steps []fantasy.StepResult
		for {
			if _, _, err := call.PrepareStep(ctx, fantasy.PrepareStepFunctionOptions{StepNumber: len(steps), Steps: steps}); err != nil {
				return nil, fmt.Errorf("prepare research test step: %w", err)
			}
			steps = append(steps, scriptedStep(1, 1).Steps[0])
			if call.StopWhen[0](steps) {
				return &fantasy.AgentResult{Steps: steps}, nil
			}
		}
	}}
	var result Result
	var state *State
	if err := w.investigate(t.Context(), agent, llmgateway.Config{}, "fixture", "", func(p Progress) error { state = p.State; return nil }, &result); err != nil {
		t.Fatal(err)
	}
	if result.Steps != 40 || state == nil || state.Status != researchPartial || reachableQuestions(*state) != 1 {
		t.Fatalf("no-progress research did not stop with its open question: steps=%d state=%+v", result.Steps, state)
	}
	if !strings.Contains(w.ledger.StopReason, "recovery") || len(w.ledger.Frontier) == 0 {
		t.Fatal("partial research lost its stopping reason or unfinished scope")
	}
}
