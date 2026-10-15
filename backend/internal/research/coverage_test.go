package research

import (
	"slices"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func coverageFixture() (*workspace, planInput, string) {
	docs := []source.Document{
		{ID: "AAAAAAAAAAAAAAAAAAAAAAAAAA", SnapshotID: "repo", Path: "internal/engine/run.go", Content: "package engine\n// header\nfunc run() {\n execute()\n}\n", LineCount: 6},
		{ID: "BBBBBBBBBBBBBBBBBBBBBBBBBB", SnapshotID: "repo", Path: "cmd/api/main.go", Content: "package main\nfunc main() { run() }", LineCount: 2},
		{ID: "CCCCCCCCCCCCCCCCCCCCCCCCCC", SnapshotID: "repo", Path: "internal/engine/contract.go", Content: "package engine\ntype Request struct {}", LineCount: 2},
		{ID: "DDDDDDDDDDDDDDDDDDDDDDDDDD", SnapshotID: "repo", Path: "internal/engine/run_test.go", Content: "package engine\nfunc TestRun() {}", LineCount: 2},
	}
	collection := source.Collection{Snapshots: []source.Snapshot{{ID: "repo", Locator: "fixture", Revision: "pinned"}}, Documents: docs}
	w := &workspace{documents: map[string]source.Document{}, reads: map[string][]source.LineRange{}}
	for _, doc := range docs {
		w.documents[doc.SnapshotID+"/"+doc.Path] = doc
		w.paths = append(w.paths, doc.SnapshotID+"/"+doc.Path)
		w.reads[doc.ID] = []source.LineRange{{StartLine: 1, EndLine: doc.LineCount}}
	}
	slices.Sort(w.paths)
	w.inventory = buildInventory(collection)
	w.ledger = Ledger{Inventory: w.inventory, Status: "in_progress"}
	ref := func(file string, start, end int) []roleEvidence {
		return []roleEvidence{{Path: "repo/" + file, StartLine: start, EndLine: end, Finding: "Fixture role for " + file}}
	}
	b := researchBoundary{Name: "execution", Repository: "repo", Implementation: ref("internal/engine/run.go", 3, 5), Callers: ref("cmd/api/main.go", 1, 2), Contracts: ref("internal/engine/contract.go", 1, 2), Validation: ref("internal/engine/run_test.go", 1, 2)}
	plan := planInput{Boundaries: []researchBoundary{b}}
	for _, c := range w.inventory.Candidates {
		plan.Candidates = append(plan.Candidates, candidateDisposition{CandidateID: c.ID, Class: "product_code", Disposition: "included", Reason: "Used by the fixture execution boundary", Boundaries: []string{"execution"}})
	}
	notes := "Execution is wired to its request contract; test source inspected, not executed. [AAAAAAAAAAAAAAAAAAAAAAAAAA:L3-L5] [BBBBBBBBBBBBBBBBBBBBBBBBBB:L1-L2] [CCCCCCCCCCCCCCCCCCCCCCCCCC:L1-L2] [DDDDDDDDDDDDDDDDDDDDDDDDDD:L1-L2]"
	return w, plan, notes
}

func TestCoverageRequiresExactRoleReadAndCitedRanges(t *testing.T) {
	w, plan, notes := coverageFixture()
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	w.reads["AAAAAAAAAAAAAAAAAAAAAAAAAA"] = []source.LineRange{{StartLine: 1, EndLine: 2}}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err == nil || !strings.Contains(err.Error(), "unread") {
		t.Fatalf("header-only read accepted: %v", err)
	}
	w.reads["AAAAAAAAAAAAAAAAAAAAAAAAAA"] = []source.LineRange{{StartLine: 1, EndLine: 6}}
	wrongRange := strings.ReplaceAll(notes, "AAAAAAAAAAAAAAAAAAAAAAAAAA:L3-L5", "AAAAAAAAAAAAAAAAAAAAAAAAAA:L1-L2")
	if err := w.checkCoverage(coverageInput{Notes: wrongRange, Status: "complete"}); err == nil {
		t.Fatal("citation to another range in the same file accepted")
	}
	withoutCaller := strings.ReplaceAll(notes, "[BBBBBBBBBBBBBBBBBBBBBBBBBB:L1-L2]", "")
	if err := w.checkCoverage(coverageInput{Notes: withoutCaller, Status: "complete"}); err == nil {
		t.Fatal("uncited caller accepted")
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err != nil {
		t.Fatal(err)
	}
	if w.checkedNotes != notes {
		t.Fatal("notes were lost")
	}
}

func TestRoleGapsCannotWaiveOtherRolesOrClaimComplete(t *testing.T) {
	w, plan, notes := coverageFixture()
	search := searchedScope{Pattern: "Request", Glob: "repo/**"}
	w.search(searchQuery{Pattern: search.Pattern, Glob: search.Glob, Limit: 2})
	plan.Boundaries[0].Contracts = nil
	plan.Boundaries[0].Validation = nil
	plan.Boundaries[0].Gaps = []roleGap{{Role: "contracts", Kind: "research_gap", Reason: "Contract behavior was not resolved", Searches: []searchedScope{search}}}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err == nil {
		t.Fatal("pending validation was accepted as complete")
	}
	plan.Boundaries[0].Validation = []roleEvidence{{Path: "repo/internal/engine/run_test.go", StartLine: 1, EndLine: 2, Finding: "Test source"}}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err == nil {
		t.Fatal("research gap accepted as complete")
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "partial"}); err != nil {
		t.Fatal(err)
	}
	if w.ledger.Status != "partial" || !slices.Contains(w.ledger.Frontier, "execution/contracts: Contract behavior was not resolved") {
		t.Fatal("precise frontier was lost")
	}
	plan.Boundaries[0].Gaps[0].Searches[0].Pattern = "never searched"
	if err := w.setPlan(plan); err == nil {
		t.Fatal("invented search accepted")
	}
}

func TestIndependentCandidatesCannotDisappear(t *testing.T) {
	w, plan, notes := coverageFixture()
	plan.Candidates = plan.Candidates[:len(plan.Candidates)-1]
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err == nil || !strings.Contains(err.Error(), "unclassified") {
		t.Fatalf("inventory omission accepted: %v", err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "partial"}); err != nil {
		t.Fatal(err)
	}
}

func TestReclassificationNeedsReasonAndRetainsHistory(t *testing.T) {
	w, plan, _ := coverageFixture()
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	plan.Boundaries[0].Implementation[0].StartLine = 4
	if err := w.setPlan(plan); err == nil {
		t.Fatal("silently dropped discovered range")
	}
	plan.Reclassifications = []reclassification{{Boundary: "execution", Reason: "Declaration was a false lead; implementation starts at line 4"}}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if len(w.ledger.History) != 2 || w.ledger.History[0].Boundaries[0].Implementation[0].StartLine != 3 {
		t.Fatal("history did not preserve original range")
	}
}

func TestCrossRepositoryHandoffRequiresBothEndpointRoles(t *testing.T) {
	w, plan, notes := coverageFixture()
	doc := source.Document{ID: "EEEEEEEEEEEEEEEEEEEEEEEEEE", SnapshotID: "consumer", Path: "client.go", Content: "package consumer\nfunc consume() {}", LineCount: 2}
	w.documents["consumer/client.go"] = doc
	w.paths = append(w.paths, "consumer/client.go")
	w.reads[doc.ID] = []source.LineRange{{StartLine: 1, EndLine: 2}}
	w.inventory.Repositories = append(w.inventory.Repositories, InventoryRepository{ID: "consumer", Revision: "pinned-two"})
	refs := []roleEvidence{{Path: "consumer/client.go", StartLine: 1, EndLine: 2, Finding: "Consumer calls the contract"}}
	plan.Boundaries = append(plan.Boundaries, researchBoundary{Name: "consumer", Repository: "consumer", Implementation: refs, Callers: refs, Contracts: refs, Validation: refs})
	h := repositoryHandoff{ID: "request", FromBoundary: "execution", ToBoundary: "consumer", Contract: "Request package export and call binding", Producer: plan.Boundaries[0].Contracts, Consumer: refs, Wiring: plan.Boundaries[0].Callers}
	plan.Handoffs = []repositoryHandoff{h}
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err == nil {
		t.Fatal("handoff accepted without consumer citation")
	}
	notes += " [EEEEEEEEEEEEEEEEEEEEEEEEEE:L1-L2]"
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err != nil {
		t.Fatal(err)
	}
	plan.Handoffs[0].Consumer = plan.Boundaries[0].Contracts
	if err := w.setPlan(plan); err == nil {
		t.Fatal("producer evidence used as consumer endpoint")
	}
	plan.Handoffs[0] = h
	plan.Handoffs[0].Consumer = nil
	if err := w.setPlan(plan); err == nil {
		t.Fatal("missing endpoint accepted without a role gap")
	}
	search := searchedScope{Pattern: "consume", Glob: "consumer/**"}
	w.search(searchQuery{Pattern: search.Pattern, Glob: search.Glob, Limit: 2})
	plan.Handoffs[0].Gaps = []roleGap{{Role: "consumer", Kind: "research_gap", Reason: "Consumer endpoint evidence is not yet verified", Searches: []searchedScope{search}}}
	if err := w.setPlan(plan); err != nil {
		t.Fatalf("explicitly recorded endpoint gap rejected: %v", err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: researchPartial}); err != nil {
		t.Fatalf("honest partial handoff coverage rejected: %v", err)
	}
}

func TestRetainedSubsystemFindingsSurviveFinalSummary(t *testing.T) {
	w, plan, notes := coverageFixture()
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.saveFinding(researchFinding{Name: "Execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	if err := w.checkCoverage(coverageInput{Notes: "Final summary.", Status: "complete"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.checkedNotes, notes) {
		t.Fatal("detailed findings compressed out of final notes")
	}
	if err := w.saveFinding(researchFinding{Name: "Unread", Notes: "[AAAAAAAAAAAAAAAAAAAAAAAAAA:L1-L99]"}); err == nil {
		t.Fatal("saved unread evidence")
	}
}

func TestRangeCoveredDoesNotSkipHoles(t *testing.T) {
	if !RangeCovered([]source.LineRange{{StartLine: 4, EndLine: 7}, {StartLine: 1, EndLine: 3}}, 1, 7) {
		t.Fatal("adjacent reads rejected")
	}
	if RangeCovered([]source.LineRange{{StartLine: 1, EndLine: 3}, {StartLine: 5, EndLine: 7}}, 1, 7) {
		t.Fatal("unread hole accepted")
	}
}

func TestPlanPatchesAndAreaClassificationPreservePriorWork(t *testing.T) {
	w, plan, notes := coverageFixture()
	first := plan.Candidates[:1]
	remaining := slices.Clone(plan.Candidates[1:])
	plan.Candidates = first
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.setPlan(planInput{Candidates: remaining}); err != nil {
		t.Fatal(err)
	}
	if len(w.plan) != 1 || len(w.ledger.Plan.Candidates) != len(w.inventory.Candidates) {
		t.Fatal("patch discarded earlier responsibilities or dispositions")
	}
	if len(w.ledger.History[1].Boundaries) != 0 {
		t.Fatal("history repeats the whole plan instead of the patch")
	}
	count, err := w.classifyArea(classifyAreaInput{PathPrefix: "repo", Kinds: []string{"source_area"}, Class: "product_code", Disposition: "included", Reason: "Execution implementation and caller evidence", Boundaries: []string{"execution"}})
	if err != nil || count != 2 {
		t.Fatalf("area classification failed: count=%d err=%v", count, err)
	}
	if err := w.checkCoverage(coverageInput{Notes: notes, Status: "complete"}); err != nil {
		t.Fatal(err)
	}
	if err := w.setPlan(planInput{Boundaries: []researchBoundary{w.plan[0], w.plan[0]}}); err == nil {
		t.Fatal("duplicate patch keys accepted")
	}
}

func TestUnfinishedResearchRetainsPreciseFrontier(t *testing.T) {
	w, plan, notes := coverageFixture()
	plan.Candidates = nil
	if err := w.setPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := w.saveFinding(researchFinding{Name: "execution", Notes: notes}); err != nil {
		t.Fatal(err)
	}
	w.reads["BBBBBBBBBBBBBBBBBBBBBBBBBB"] = nil
	frontier := w.unfinishedFrontier()
	if !slices.Contains(frontier, "execution/callers: unread repo/cmd/api/main.go:L1-L2") {
		t.Fatal("unfinished caller range lost")
	}
	if !slices.ContainsFunc(frontier, func(item string) bool { return strings.HasPrefix(item, "unclassified source_area:") }) {
		t.Fatal("unclassified source frontier lost")
	}
	if !strings.Contains(w.submittedNotes, notes) {
		t.Fatal("saved findings lost when investigation remained unfinished")
	}
}
