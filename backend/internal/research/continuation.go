package research

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"charm.land/fantasy"
)

type researchQuestion struct {
	Key        string          `json:"key"`
	Kind       string          `json:"kind"`
	Repository string          `json:"repository,omitempty"`
	Path       string          `json:"path,omitempty"`
	Line       int             `json:"line,omitempty"`
	Reason     string          `json:"reason"`
	Searches   []searchedScope `json:"searches,omitempty"`
	Actionable bool            `json:"actionable"`
	priority   int
}

func (w *workspace) availablePath(scope string) bool {
	prefix := path.Clean(scope)
	for _, file := range w.paths {
		if file != prefix && !strings.HasPrefix(file, prefix+"/") {
			continue
		}
		if w.readFailures[file] == "" {
			return true
		}
	}
	return false
}

func (w *workspace) searchHasMatches(scopes []searchedScope) bool {
	for _, scope := range scopes {
		if slices.ContainsFunc(w.ledger.Searches, func(record searchRecord) bool {
			return record.Pattern == scope.Pattern && record.Glob == scope.Glob && record.Matches > 0
		}) {
			return true
		}
	}
	return false
}

func (w *workspace) pendingQuestions() []researchQuestion {
	var questions []researchQuestion
	citations := w.citations()
	roles := func(name, repo string, evidence map[string][]roleEvidence, gaps []roleGap) {
		for role, refs := range evidence {
			q := w.roleQuestion(name, role, refs, gaps, citations)
			if q.Status == "answered" || q.Status == "not_applicable" {
				continue
			}
			var searches []searchedScope
			for _, gap := range gaps {
				if gap.Role == role {
					searches = gap.Searches
				}
			}
			questions = append(questions, researchQuestion{Key: q.ID, Kind: "proof_obligation", Repository: repo, Reason: q.Question + " " + q.Result, Searches: searches, Actionable: q.Status == "pending"})
		}
	}
	for _, b := range w.plan {
		roles(b.Name, b.Repository, b.roles(), b.Gaps)
	}
	for _, h := range w.ledger.Plan.Handoffs {
		roles(h.ID, "", h.roles(), h.Gaps)
	}
	for _, repo := range w.unresearchedRepositories() {
		questions = append(questions, researchQuestion{Key: "repository/" + repo.ID, Kind: "proof_obligation", Repository: repo.ID, Path: repo.ID, Reason: "No product responsibility has been researched in this repository yet.", Actionable: w.availablePath(repo.ID)})
	}
	dispositions := map[string]candidateDisposition{}
	for _, d := range w.ledger.Plan.Candidates {
		dispositions[d.CandidateID] = d
	}
	priorities := map[string]int{"repository_reference": 1, "registry": 2, "source_area": 3, "entrypoint": 4, "manifest": 5, "configuration": 6, "repository": 7}
	for _, c := range w.inventory.Candidates {
		d, ok := dispositions[c.ID]
		if ok && d.Disposition != "deferred" {
			continue
		}
		reason := d.Reason
		if !ok {
			reason = "Not classified or researched yet"
		}
		questions = append(questions, researchQuestion{Key: c.ID, Kind: c.Kind, Repository: c.Repository, Path: c.Path, Line: c.Line, Reason: reason, Actionable: w.availablePath(c.Path), priority: priorities[c.Kind]})
	}
	slices.SortFunc(questions, func(a, b researchQuestion) int {
		if a.Actionable != b.Actionable {
			if a.Actionable {
				return -1
			}
			return 1
		}
		if a.priority != b.priority {
			return a.priority - b.priority
		}
		return strings.Compare(a.Repository+"/"+a.Key, b.Repository+"/"+b.Key)
	})
	return questions
}

func (w *workspace) partialStopReason(requested string) (bool, string) {
	// Standalone structural checks have no active runner budget.
	if w.turnBudget == 0 {
		return true, "Coverage-only check"
	}
	if w.budget != nil && w.budget.stopped {
		return true, w.budget.decisions[len(w.budget.decisions)-1].Reason
	}
	if w.turn >= w.turnBudget-1 {
		return true, fmt.Sprintf("Attempt turn budget reached (%d/%d); remaining obligations are preserved", w.turn, w.turnBudget)
	}
	if !w.deadline.IsZero() && time.Until(w.deadline) <= 30*time.Second {
		return true, "Attempt time budget nearly exhausted; remaining obligations are preserved"
	}
	for _, q := range w.pendingQuestions() {
		if q.Actionable {
			return false, "Reachable source questions remain and the attempt budget is still available"
		}
	}
	if strings.TrimSpace(requested) == "" {
		return false, "State the precise unavailable source or exhausted search scope before stopping early"
	}
	return true, "Supplied sources unavailable under the recorded searches: " + requested
}

func (w *workspace) frontierSummary() any {
	type group struct {
		Repository string `json:"repository"`
		Kind       string `json:"kind"`
		Count      int    `json:"count"`
	}
	groups := map[string]group{}
	questions := w.pendingQuestions()
	for _, q := range questions {
		key := q.Repository + "/" + q.Kind
		g := groups[key]
		g.Repository = q.Repository
		g.Kind = q.Kind
		g.Count++
		groups[key] = g
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	rows := make([]group, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, groups[key])
	}
	return map[string]any{"open_question_count": len(questions), "groups": rows, "next_questions": questions[:min(8, len(questions))], "full_frontier": "Preserved in the research ledger; page next_research_questions or list_research_inventory for every scope and candidate ID."}
}

func (w *workspace) continuationPrompt(orientation string, remaining int) string {
	state, _ := json.Marshal(map[string]any{"remaining_model_turns": remaining, "responsibilities": w.plan, "handoffs": w.ledger.Plan.Handoffs, "frontier": w.frontierSummary()})
	return orientation + "\nContinue useful source investigation across the full supplied scope. Previous model text or a deferred classification is not a stopping reason. Resolve known role/handoff gaps, then independent source areas and material variants; prioritize product behavior over inventory bookkeeping. Do not repeat completed reads or findings. Retain new findings incrementally and update only changed plan entries. Review next_research_questions for the remaining branches.\nRESUME STATE (verified original document IDs/read ranges remain available; findings below are retained verbatim):\n" + string(state) + "\nRETAINED FINDINGS:\n" + w.submittedNotes
}

func (w *workspace) questionTool() fantasy.AgentTool {
	return fantasy.NewAgentTool("next_research_questions", "Page unresolved proof obligations and independent candidate scopes, ordered to prioritize reachable role/handoff evidence and module/variant discovery. A deferred classification is open work, not a stop reason. offset=0, limit=12. Full inventory remains available separately.", func(_ context.Context, input listInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
		if err := w.tick("Selecting remaining source questions."); err != nil {
			return fantasy.ToolResponse{}, err
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		questions := w.pendingQuestions()
		start := min(max(input.Offset, 0), len(questions))
		end := min(start+min(max(input.Limit, 1), 30), len(questions))
		return toolJSON(map[string]any{"questions": questions[start:end], "total": len(questions), "next_offset": end, "has_more": end < len(questions), "turns_remaining": max(w.turnBudget-w.turn, 0)})
	})
}
