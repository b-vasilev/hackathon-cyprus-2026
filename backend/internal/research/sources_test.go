package research

import (
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestAvailablePathKeepsScopeBoundariesAndLiveFailures(t *testing.T) {
	w := &workspace{
		documents: map[string]source.Document{
			"repo/src/a.go": {}, "repo/src/b.go": {}, "repo/src-other/c.go": {},
		},
		readFailures: map[string]string{"repo/src/a.go": "unavailable"},
	}
	if !w.availablePath("repo/src/.") || !w.availablePath("repo/src/b.go") || w.availablePath("repo/s") {
		t.Fatal("scope lookup lost a readable child or matched a partial path")
	}
	w.readFailures["repo/src/b.go"] = "unavailable"
	if w.availablePath("repo/src") || w.availablePath("repo/src/a.go") {
		t.Fatal("failed sources or a neighboring directory made the scope readable")
	}
	delete(w.readFailures, "repo/src/a.go")
	if !w.availablePath("repo/src") {
		t.Fatal("scope lookup cached a stale read failure")
	}
}

func TestWorkersShareSourcesButValidateTheirOwnReads(t *testing.T) {
	w, _, notes := coverageFixture()
	local, err := w.forkTask(researchTask{Key: "scope/repo"}, taskTurns)
	if err != nil {
		t.Fatal(err)
	}
	if local.sourceIndex() != w.sourceIndex() {
		t.Fatal("worker rebuilt the pinned source index")
	}
	if _, err := local.evidenceRanges(notes); err != nil {
		t.Fatal(err)
	}
	delete(local.reads, "AAAAAAAAAAAAAAAAAAAAAAAAAA")
	if _, err := local.evidenceRanges(notes); err == nil {
		t.Fatal("worker accepted an unread citation through the shared index")
	}
	if _, err := w.evidenceRanges(notes); err != nil {
		t.Fatal("worker changed the parent's read evidence")
	}
}
