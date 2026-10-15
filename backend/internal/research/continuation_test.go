package research

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func fixtureCheckpoint(t *testing.T, status string) (*workspace, source.Collection, Checkpoint) {
	t.Helper()
	w, plan, notes := coverageFixture()
	if status == "partial" {
		plan.Candidates[0].Disposition = "deferred"
		plan.Candidates[0].Reason = "Independent source area still needs investigation"
	}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: "Preserved summary.", Status: status}); err != nil {
		t.Fatal(err)
	}
	var collection source.Collection
	for _, r := range w.inventory.Repositories {
		collection.Snapshots = append(collection.Snapshots, source.Snapshot{ID: r.ID, Locator: r.Locator, Revision: r.Revision})
	}
	for _, p := range w.paths {
		collection.Documents = append(collection.Documents, w.documents[p])
	}
	read := w.readCollection(collection)
	return w, collection, Checkpoint{Context: "Untrusted old packed context", Ledger: w.ledger, Notes: w.checkedNotes, Snapshots: read.Snapshots}
}

func TestCompletedCheckpointReusesSourcesWithoutModelOrReader(t *testing.T) {
	_, collection, checkpoint := fixtureCheckpoint(t, "complete")
	originalNotes := checkpoint.Notes
	runner := Runner{WorkDir: t.TempDir(), RepomixBin: "/does/not/exist"}
	result, err := runner.RunFromCheckpoint(t.Context(), llmgateway.Config{}, "", collection, checkpoint, func(Progress) error { t.Fatal("complete checkpoint attempted external work"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps != 0 || result.MCPReads != 0 || result.InputTokens != 0 || result.Ledger.Status != "complete" {
		t.Fatalf("complete checkpoint was not reused: %+v", result)
	}
	if result.Progress == nil || result.Progress.Status != "complete" || len(result.Progress.Areas) == 0 || result.Progress.Budget.Used != 0 {
		t.Fatal("reused completed research lost its visible plan")
	}
	if !strings.Contains(result.Context, "3: func run() {") || strings.Contains(result.Context, checkpoint.Context) {
		t.Fatal("source context was not rebuilt from original pinned lines")
	}
	if checkpoint.Notes != originalNotes {
		t.Fatal("original checkpoint mutated")
	}
}

func TestCheckpointRejectsChangedRevisionDocumentOrRange(t *testing.T) {
	for _, kind := range []string{"revision", "document", "range"} {
		t.Run(kind, func(t *testing.T) {
			_, collection, checkpoint := fixtureCheckpoint(t, "complete")
			switch kind {
			case "revision":
				checkpoint.Snapshots[0].Revision = "different"
			case "document":
				checkpoint.Snapshots[0].Coverage.Files[0].ID = "different"
			case "range":
				checkpoint.Snapshots[0].Coverage.Files[0].ReadRanges = []source.LineRange{{StartLine: 1, EndLine: 1000}}
			}
			runner := Runner{WorkDir: t.TempDir(), RepomixBin: "/does/not/exist"}
			if _, err := runner.RunFromCheckpoint(t.Context(), llmgateway.Config{}, "", collection, checkpoint, func(Progress) error { return nil }); err == nil {
				t.Fatal("mismatched checkpoint accepted")
			}
		})
	}
}

func TestPrematurePartialRequiresContinuationOfIndependentWork(t *testing.T) {
	w, _, checkpoint := fixtureCheckpoint(t, "partial")
	w.turnBudget, w.turn = 60, 24
	w.deadline = time.Now().Add(20 * time.Minute)
	if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: "partial", StopReason: "large scope"}); err == nil || !strings.Contains(err.Error(), "continue research") {
		t.Fatalf("premature partial accepted: %v", err)
	}
	if w.checkedNotes != "" || !strings.Contains(w.submittedNotes, "Preserved summary.") {
		t.Fatal("early stop lost findings or became accepted")
	}
	if len(w.pendingQuestions()) == 0 {
		t.Fatal("independent questions disappeared")
	}
	w.turn = 59
	if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: "partial"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.ledger.StopReason, "59/60") {
		t.Fatal("actual budget stopping reason missing")
	}
	if checkpoint.Ledger.Status != "partial" {
		t.Fatal("prior checkpoint mutated")
	}
}

func TestUnavailableEvidenceCanStopEarlyButNotIndependentBranches(t *testing.T) {
	w, plan, notes := coverageFixture()
	scope := searchedScope{Pattern: "external_receiver_not_in_supplied_sources", Glob: "repo/**"}
	w.search(searchQuery{Pattern: scope.Pattern, Glob: scope.Glob, Limit: 1})
	plan.Boundaries[0].Contracts = nil
	plan.Boundaries[0].Gaps = []roleGap{{Role: "contracts", Kind: "research_gap", Reason: "Receiving implementation is outside supplied sources", Searches: []searchedScope{scope}}}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	w.turnBudget, w.turn = 60, 5
	stop := coverageInput{Notes: notes, Status: "partial", StopReason: "Receiver repository is unavailable; no matching contract in supplied sources"}
	if err := w.checkCoverage(stop); err != nil {
		t.Fatal(err)
	}
	plan.Candidates[0].Disposition = "deferred"
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(stop); err == nil {
		t.Fatal("external blocker waived independent source work")
	}
}

type scriptedResearchAgent struct {
	fantasy.Agent
	generate func(context.Context, fantasy.AgentCall) (*fantasy.AgentResult, error)
}

func (a scriptedResearchAgent) Generate(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
	return a.generate(ctx, call)
}

func TestLoopContinuesAfterEarlyTextAndPartialCheckpoint(t *testing.T) {
	original, _, checkpoint := fixtureCheckpoint(t, "partial")
	w, _, _ := coverageFixture()
	w.turnBudget = 60
	w.event = func(string) error { return nil }
	if err := w.restore(checkpoint); err != nil {
		t.Fatal(err)
	}
	beforeReads := w.readCollection(source.Collection{Snapshots: checkpoint.Snapshots})
	calls := 0
	var steps []int
	agent := scriptedResearchAgent{generate: func(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
		calls++
		if _, _, err := call.PrepareStep(ctx, fantasy.PrepareStepFunctionOptions{Messages: []fantasy.Message{fantasy.NewUserMessage(call.Prompt)}}); err != nil {
			return nil, fmt.Errorf("prepare scripted turn: %w", err)
		}
		switch calls {
		case 1:
			if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: "partial"}); err == nil {
				t.Fatal("partial accepted before remaining useful work")
			}
		default:
			if !strings.Contains(call.Prompt, "remaining_model_turns") || !strings.Contains(call.Prompt, "Preserved summary.") {
				t.Fatal("continuation omitted budget or prior findings")
			}
			d := w.ledger.Plan.Candidates[0]
			d.Disposition = "included"
			d.Reason = "Independent branch resolved using existing source findings"
			if err := w.setPlan(planInput{Candidates: []candidateDisposition{d}}); err != nil {
				return nil, err
			}
			if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: "complete"}); err != nil {
				return nil, err
			}
		}
		return scriptedStep(10, 5), nil
	}}
	var result Result
	err := w.investigate(t.Context(), agent, llmgateway.Config{}, "fixture", "", func(p Progress) error {
		if p.Step > 0 {
			steps = append(steps, p.Step)
		}
		return nil
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || result.Steps != 2 || result.InputTokens != 20 || !reflect.DeepEqual(steps, []int{1, 2}) {
		t.Fatalf("continuation usage lost: calls=%d result=%+v steps=%v", calls, result, steps)
	}
	if w.mcpReads != 0 || !reflect.DeepEqual(beforeReads, w.readCollection(source.Collection{Snapshots: checkpoint.Snapshots})) {
		t.Fatal("checkpoint reads were repeated or changed")
	}
	if original.ledger.Plan.Candidates[0].Disposition != "deferred" {
		t.Fatal("continuation mutated previous checkpoint plan")
	}
}

func TestLoopFailsRepeatedNoProgressWithoutClaimingPartial(t *testing.T) {
	w, _, _ := fixtureCheckpoint(t, "partial")
	w.checkedNotes = ""
	w.turnBudget = 60
	w.event = func(string) error { return nil }
	calls := 0
	agent := scriptedResearchAgent{generate: func(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
		calls++
		if _, _, err := call.PrepareStep(ctx, fantasy.PrepareStepFunctionOptions{}); err != nil {
			return nil, fmt.Errorf("prepare stalled turn: %w", err)
		}
		return scriptedStep(1, 0), nil
	}}
	var result Result
	err := w.investigate(t.Context(), agent, llmgateway.Config{}, "fixture", "", func(Progress) error { return nil }, &result)
	if err == nil || !strings.Contains(err.Error(), "without progress") || calls != 3 {
		t.Fatalf("no-progress stop was not honest and bounded: calls=%d err=%v", calls, err)
	}
	if w.checkedNotes != "" || w.submittedNotes == "" {
		t.Fatal("failed no-progress run claimed acceptance or lost findings")
	}
}

func scriptedStep(input, output int64) *fantasy.AgentResult {
	var step fantasy.StepResult
	step.Usage.InputTokens = input
	step.Usage.OutputTokens = output
	return &fantasy.AgentResult{Steps: []fantasy.StepResult{step}}
}

func TestEmptyPartialCheckpointCanContinue(t *testing.T) {
	w, _, checkpoint := fixtureCheckpoint(t, "partial")
	checkpoint.Notes = ""
	checkpoint.Context = ""
	checkpoint.Ledger.Findings = nil
	if err := w.restore(checkpoint); err != nil {
		t.Fatal(err)
	}
	if w.checkedNotes != "" || w.ledger.Status != "in_progress" || len(w.reads) == 0 || len(w.pendingQuestions()) == 0 {
		t.Fatal("empty checkpoint was accepted complete or lost resumable state")
	}
	checkpoint.Ledger.Status = "complete"
	if err := w.restore(checkpoint); err == nil {
		t.Fatal("empty completed checkpoint was trusted")
	}
}

func TestSourceReaderErrorsHideInfrastructureButKeepCause(t *testing.T) {
	cause := errors.New("repomix at /private/operator/path failed")
	err := &sourceReaderError{message: "source reader is unavailable", cause: cause}
	if strings.Contains(strings.ToLower(err.Error()), "repomix") || !errors.Is(err, cause) {
		t.Fatal("infrastructure leaked or diagnostic cause was lost")
	}
}

func TestSourceContextMergesOverlappingRangesWithoutDroppingAnchors(t *testing.T) {
	w, _, _ := coverageFixture()
	notes := "[AAAAAAAAAAAAAAAAAAAAAAAAAA:L1-L4] [AAAAAAAAAAAAAAAAAAAAAAAAAA:L3-L6]"
	contextText, err := w.evidenceContext(notes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contextText, notes) || strings.Count(contextText, "3: func run() {") != 1 || strings.Count(contextText, "<source_document ") != 1 {
		t.Fatal("overlap duplicated lines or original anchors were removed")
	}
}
