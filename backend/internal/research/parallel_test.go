package research

import (
	"slices"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestResearchQueuePrioritizesProductOverAgentGuidanceAndCI(t *testing.T) {
	w, _, _ := coverageFixture()
	for _, file := range []string{".agents/skills/review.md", ".claude/commands/test.md", ".codex/config.toml", "AGENTS.md", "CLAUDE.md", ".github/workflows/build.yml"} {
		p := "repo/" + file
		w.documents[p] = source.Document{ID: p, SnapshotID: "repo", Path: file}
		w.paths = append(w.paths, p)
		w.inventory.Candidates = append(w.inventory.Candidates, Candidate{ID: p, Repository: "repo", Path: p, Kind: "source_area"})
	}
	if err := w.excludeDevelopmentGuidance(); err != nil {
		t.Fatal(err)
	}
	for _, q := range w.pendingQuestions() {
		if isDevelopmentGuidance(q.Path) {
			t.Fatalf("agent guidance consumes an independent task: %s", q.Path)
		}
	}
	tasks := w.researchTasks()
	if len(tasks) < 2 || tasks[len(tasks)-1].Key != "scope/repo/.github/workflows/build.yml" {
		t.Fatalf("CI was scheduled before product work: %+v", tasks)
	}
	// Explicit evidence can bring an instruction area into product scope.
	if err := w.setPlan(planInput{Candidates: []candidateDisposition{{CandidateID: "repo/AGENTS.md", Class: "docs", Disposition: "deferred", Reason: "Runtime code reads this file; inspect its product role"}}}); err != nil {
		t.Fatal(err)
	}
	if err := w.excludeDevelopmentGuidance(); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(w.pendingQuestions(), func(q researchQuestion) bool { return q.Key == "repo/AGENTS.md" }) {
		t.Fatal("explicit product research was overwritten by the default exclusion")
	}
}

func TestWorkerPublishesAnswersBeforeFinishingAndCanCorrectThem(t *testing.T) {
	w, plan, notes := coverageFixture()
	local, err := w.forkTask(researchTask{Key: "scope/repo", Candidates: slices.Clone(w.inventory.Candidates)}, taskTurns)
	if err != nil {
		t.Fatal(err)
	}
	local.toolCalls, local.mcpReads = 7, 2
	if err := local.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := local.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := local.taskSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(w.plan) != 0 {
		t.Fatal("worker mutated the shared plan directly")
	}
	if err := w.mergeTask(snapshot); err != nil {
		t.Fatal(err)
	}
	if w.state().Areas[0].Status != "checked" {
		t.Fatal("answers were not visible while the worker was still active")
	}
	if w.toolCalls != 0 || w.mcpReads != 0 {
		t.Fatal("intermediate snapshot double-counts tool usage")
	}
	corrected, err := clonePlan(local.ledger.Plan)
	if err != nil {
		t.Fatal(err)
	}
	corrected.Boundaries[0].Implementation[0].Finding = "Corrected implementation finding"
	corrected.Reclassifications = []reclassification{{Boundary: "execution", Reason: "Refined interpretation of the same source"}}
	if err := local.setPlan(corrected); err != nil {
		t.Fatal(err)
	}
	if snapshot.plan[0].Implementation[0].Finding == corrected.Boundaries[0].Implementation[0].Finding {
		t.Fatal("published snapshot changed with its worker")
	}
	next, err := local.taskSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.mergeTask(next); err != nil {
		t.Fatalf("worker correction conflicted with its earlier snapshot: %v", err)
	}
	if err := w.mergeTask(local); err != nil {
		t.Fatal(err)
	}
	if w.toolCalls != 7 || w.mcpReads != 2 || w.plan[0].Implementation[0].Finding != "Corrected implementation finding" {
		t.Fatal("final merge lost corrections or counted tool usage incorrectly")
	}
}
