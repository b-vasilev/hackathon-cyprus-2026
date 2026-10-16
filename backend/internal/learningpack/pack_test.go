package learningpack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func question(id string) Question {
	return Question{ID: id, ChapterID: "overview", Kind: "single", Prompt: "What does it do?", Options: []string{"A", "B", "C"}, Correct: []int{1}, Explanation: "The overview says B."}
}

func validPack() generated {
	pack := generated{Mission: generatedMission{Title: "First task", Intro: "Study a real fix.", IssueNumber: 7, IssueWhy: "Small and tested.", Finale: "Nice work. You are ready to go!"}}
	for i := range 6 {
		pack.Quiz = append(pack.Quiz, question(fmt.Sprintf("q%d", i)))
	}
	for i := range 5 {
		pack.Mission.Steps = append(pack.Mission.Steps, Step{ID: fmt.Sprintf("s%d", i), Title: "Do it", Body: "Details", DoneWhen: "You see it"})
	}
	pack.Mission.Check = []Question{question("c1"), question("c2")}
	return pack
}

func TestValidate(t *testing.T) {
	chapters := []string{"overview"}
	candidates := []Issue{{Number: 7}}
	if err := validate(validPack(), chapters, candidates); err != nil {
		t.Fatalf("valid pack rejected: %v", err)
	}
	for name, tc := range map[string]struct {
		change func(*generated)
		want   string
	}{
		"invented issue":     {func(p *generated) { p.Mission.IssueNumber = 99 }, "not one of the candidate issues"},
		"bad index":          {func(p *generated) { p.Quiz[0].Correct = []int{3} }, "valid option indexes"},
		"multiple with one":  {func(p *generated) { p.Quiz[0].Kind = "multiple" }, "at least two correct"},
		"unknown chapter":    {func(p *generated) { p.Mission.Steps[0].ChapterID = "nope" }, "unknown chapter_id"},
		"duplicate id":       {func(p *generated) { p.Mission.Check[0].ID = "q0" }, "unique"},
		"missing ready line": {func(p *generated) { p.Mission.Finale = "Done." }, "You are ready to go!"},
		"too few questions":  {func(p *generated) { p.Quiz = p.Quiz[:3] }, "six to eight"},
	} {
		t.Run(name, func(t *testing.T) {
			pack := validPack()
			tc.change(&pack)
			err := validate(pack, chapters, candidates)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

// Only merged pull requests that close a real issue become candidates, smallest change first.
func TestCandidates(t *testing.T) {
	pulls := []map[string]any{
		{"number": 10, "title": "big", "body": "Fixes #1", "merged_at": "2026-10-01", "html_url": "https://github.com/o/r/pull/10"},
		{"number": 11, "title": "small", "body": "Closes #2", "merged_at": "2026-10-02", "html_url": "https://github.com/o/r/pull/11"},
		{"number": 12, "title": "unmerged", "body": "Fixes #3", "merged_at": ""},
		{"number": 13, "title": "no link", "body": "Refactor", "merged_at": "2026-10-03"},
	}
	size := map[int]int{10: 900, 11: 20}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing token on %s", r.URL.Path)
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		n := 0
		if len(parts) > 4 {
			n, _ = strconv.Atoi(parts[4])
		}
		var value any
		switch {
		case r.URL.Path == "/repos/o/r/pulls":
			value = pulls
		case len(parts) == 5 && parts[3] == "pulls":
			value = map[string]any{"number": n, "additions": size[n], "deletions": 0, "merged_at": "x", "body": ""}
		case len(parts) == 6 && parts[5] == "files":
			value = []map[string]string{{"filename": "src/" + parts[4] + ".py"}}
		case len(parts) == 5 && parts[3] == "issues":
			value = map[string]any{"number": n, "title": "Issue " + parts[4], "state": "closed", "html_url": "https://github.com/o/r/issues/" + parts[4]}
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(value)
	}))
	defer server.Close()
	client := githubClient{http: server.Client(), token: "secret", base: server.URL}
	got, err := client.candidates(context.Background(), "o/r")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 2 || got[0].PullNumber != 11 || got[0].Files[0] != "src/11.py" || got[1].Number != 1 {
		t.Fatalf("unexpected candidates: %+v", got)
	}
}
