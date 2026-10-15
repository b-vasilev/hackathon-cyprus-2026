package research

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// State is a view of the current evidence ledger, not an estimate of all code.
type State struct {
	Execution  string           `json:"execution,omitempty"`
	Status     string           `json:"status"`
	Areas      []AreaProgress   `json:"areas"`
	Focus      Focus            `json:"focus"`
	Activity   string           `json:"activity"`
	LastResult string           `json:"last_result"`
	Budget     BudgetProgress   `json:"budget"`
	Parallel   ParallelProgress `json:"parallel,omitempty"`
}

type ParallelProgress struct {
	Waiting  int              `json:"waiting,omitempty"`
	Requests int              `json:"requests,omitempty"`
	Limit    int              `json:"limit"`
	Active   int              `json:"active"`
	Queued   int              `json:"queued"`
	Workers  []WorkerProgress `json:"workers"`
}

type WorkerProgress struct {
	Question string `json:"question,omitempty"`
	Status   string `json:"status,omitempty"`
	Turns    int    `json:"turns,omitempty"`
	ID       string `json:"id"`
	Task     string `json:"task"`
	Focus    Focus  `json:"focus"`
	Activity string `json:"activity"`
}

type Focus struct {
	Area     string `json:"area"`
	Question string `json:"question"`
	Reason   string `json:"reason"`
}

type AreaProgress struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Repository string             `json:"repository"`
	Status     string             `json:"status"`
	Questions  []QuestionProgress `json:"questions"`
}

type QuestionProgress struct {
	ID       string             `json:"id"`
	Question string             `json:"question"`
	Status   string             `json:"status"`
	Result   string             `json:"result"`
	Evidence []ProgressEvidence `json:"evidence"`
}

type ProgressEvidence struct {
	DocumentID string `json:"document_id"`
	Path       string `json:"path"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
}

type BudgetProgress struct {
	Used      int              `json:"used"`
	Limit     int              `json:"limit"`
	HardLimit int              `json:"hard_limit"`
	Deadline  string           `json:"deadline"`
	Decisions []BudgetDecision `json:"decisions"`
}

type BudgetDecision struct {
	Turn   int    `json:"turn"`
	Limit  int    `json:"limit"`
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

func (w *workspace) citations() map[string][]source.LineRange {
	result := map[string][]source.LineRange{}
	for _, match := range citationPattern.FindAllStringSubmatch(w.findingsText(w.summaryNotes), -1) {
		start, _ := strconv.Atoi(match[2])
		end, _ := strconv.Atoi(match[3])
		result[match[1]] = append(result[match[1]], source.LineRange{StartLine: start, EndLine: end})
	}
	return result
}

var roleQuestions = map[string]string{
	"implementation": "How does the implementation work?",
	"callers":        "What calls this behavior and with which inputs?",
	"contracts":      "Which interfaces and data contracts apply?",
	"validation":     "What checks and test cases cover this behavior?",
	"producer":       "What does the sending component produce?",
	"consumer":       "How does the receiving component handle it?",
	"wiring":         "How are the two components connected?",
}

func (w *workspace) roleQuestion(id, role string, refs []roleEvidence, gaps []roleGap, citations map[string][]source.LineRange) QuestionProgress {
	q := QuestionProgress{ID: id + "/" + role, Question: roleQuestions[role], Status: "pending", Evidence: []ProgressEvidence{}}
	verified := len(refs) > 0
	var findings []string
	for _, ref := range refs {
		doc := w.documents[ref.Path]
		if !RangeCovered(w.reads[doc.ID], ref.StartLine, ref.EndLine) || !RangeCovered(citations[doc.ID], ref.StartLine, ref.EndLine) {
			verified = false
			continue
		}
		q.Evidence = append(q.Evidence, ProgressEvidence{doc.ID, ref.Path, ref.StartLine, ref.EndLine})
		findings = append(findings, ref.Finding)
	}
	if verified {
		q.Status, q.Result = "answered", strings.Join(findings, " ")
	} else if len(refs) > 0 {
		q.Result = "Planned evidence still needs matching source reads and citations in saved findings."
	}
	for _, gap := range gaps {
		if gap.Role != role {
			continue
		}
		q.Result = gap.Reason
		switch {
		case gap.Kind == "not_applicable":
			q.Status = "not_applicable"
		case w.searchHasMatches(gap.Searches):
			q.Status = "pending"
		default:
			q.Status = "blocked"
		}
	}
	return q
}

func areaStatus(questions []QuestionProgress) string {
	answered, blocked, excluded := 0, 0, 0
	for _, q := range questions {
		switch q.Status {
		case "answered", "not_applicable":
			answered++
		case "blocked":
			blocked++
		case "excluded":
			excluded++
		}
	}
	switch {
	case excluded == len(questions):
		return "excluded"
	case answered+excluded == len(questions):
		return "checked"
	case blocked+answered+excluded == len(questions):
		return "blocked"
	case answered > 0:
		return "investigating"
	default:
		return "not_started"
	}
}

func (w *workspace) unresearchedRepositories() []InventoryRepository {
	var result []InventoryRepository
	for _, repo := range w.inventory.Repositories {
		if !slices.ContainsFunc(w.plan, func(b researchBoundary) bool { return b.Repository == repo.ID }) {
			result = append(result, repo)
		}
	}
	return result
}

func (w *workspace) state() State {
	w.mu.Lock()
	defer w.mu.Unlock()
	state := State{Status: w.ledger.Status, Focus: w.ledger.Focus, Activity: w.activity, Areas: []AreaProgress{}, Parallel: w.parallel, Execution: w.execution}
	state.Parallel.Workers = slices.Clone(w.parallel.Workers)
	if state.Parallel.Workers == nil {
		state.Parallel.Workers = []WorkerProgress{}
	}
	repositories := map[string]string{}
	for _, repo := range w.inventory.Repositories {
		repositories[repo.ID] = repo.Locator
	}
	citations := w.citations()
	for _, b := range w.plan {
		area := AreaProgress{ID: "responsibility/" + b.Repository + "/" + b.Name, Name: b.Name, Repository: b.Repository}
		roles := b.roles()
		for _, role := range []string{"implementation", "callers", "contracts", "validation"} {
			area.Questions = append(area.Questions, w.roleQuestion(area.ID, role, roles[role], b.Gaps, citations))
		}
		area.Status = areaStatus(area.Questions)
		state.Areas = append(state.Areas, area)
	}
	for _, h := range w.ledger.Plan.Handoffs {
		area := AreaProgress{ID: "handoff/" + h.ID, Name: h.FromBoundary + " → " + h.ToBoundary, Repository: "Cross-repository"}
		roles := h.roles()
		for _, role := range []string{"producer", "consumer", "wiring"} {
			area.Questions = append(area.Questions, w.roleQuestion(area.ID, role, roles[role], h.Gaps, citations))
		}
		area.Status = areaStatus(area.Questions)
		state.Areas = append(state.Areas, area)
	}
	// Discovery scopes stay visible until they are mapped to responsibilities.
	// Exclusions have their own state and never count as checked product behavior.
	dispositions := map[string]candidateDisposition{}
	for _, d := range w.ledger.Plan.Candidates {
		dispositions[d.CandidateID] = d
	}
	groups := map[string]*AreaProgress{}
	for _, c := range w.inventory.Candidates {
		d := dispositions[c.ID]
		if d.Disposition == "included" {
			continue
		}
		area := groups[c.Path]
		if area == nil {
			area = &AreaProgress{ID: "scope/" + c.Path, Name: strings.TrimPrefix(c.Path, c.Repository+"/"), Repository: c.Repository}
			groups[c.Path] = area
		}
		q := QuestionProgress{ID: c.ID, Question: "Investigate " + strings.ReplaceAll(c.Kind, "_", " ") + ": " + c.Reason, Status: "pending", Result: d.Reason, Evidence: []ProgressEvidence{}}
		if d.Disposition == "excluded" {
			q.Status = "excluded"
		} else if !w.availablePath(c.Path) {
			q.Status = "blocked"
			q.Result = "No readable indexed sources are available in this scope. " + q.Result
		}
		area.Questions = append(area.Questions, q)
	}
	for _, repo := range w.unresearchedRepositories() {
		key := repo.ID + "/."
		area := groups[key]
		if area == nil {
			area = &AreaProgress{ID: "scope/" + key, Name: "Repository overview", Repository: repo.ID}
			groups[key] = area
		}
		status := "pending"
		if !w.availablePath(repo.ID) {
			status = "blocked"
		}
		area.Questions = append(area.Questions, QuestionProgress{ID: "repository/" + repo.ID, Question: "Which product responsibilities does this repository implement?", Status: status, Result: "No responsibility has been researched in this repository yet.", Evidence: []ProgressEvidence{}})
	}
	for _, area := range groups {
		area.Status = areaStatus(area.Questions)
		state.Areas = append(state.Areas, *area)
	}
	slices.SortFunc(state.Areas, func(a, b AreaProgress) int { return strings.Compare(a.ID, b.ID) })
	for i := range state.Areas {
		area := &state.Areas[i]
		if label := repositories[area.Repository]; label != "" {
			area.Repository = label
		}
		if area.Name == "." {
			area.Name = "Repository overview"
		}
		if (area.Name == state.Focus.Area || area.ID == "scope/"+state.Focus.Area) && area.Status == "not_started" {
			area.Status = "investigating"
		}
	}
	if len(w.ledger.Findings) > 0 {
		state.LastResult = w.ledger.Findings[len(w.ledger.Findings)-1].Notes
	}
	state.Budget = BudgetProgress{Used: w.turn, Limit: w.activeLimit(), HardLimit: w.turnBudget, Decisions: []BudgetDecision{}}
	if !w.deadline.IsZero() {
		state.Budget.Deadline = w.deadline.UTC().Format(time.RFC3339)
	}
	if w.budget != nil {
		state.Budget.Decisions = slices.Clone(w.budget.decisions)
	}
	return state
}

func (w *workspace) publish(progress func(Progress) error, message string) error {
	state := w.state()
	return progress(Progress{Message: message, State: &state})
}

func (w *workspace) focusTool() fantasy.AgentTool {
	return fantasy.NewAgentTool("set_research_focus", "Before investigating a new area or question, explain the current area, specific question and why it matters. Use a responsibility name or discovery path visible in the research plan. This updates the user's compact progress panel; it does not close any evidence obligation.", func(_ context.Context, input Focus, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
		if strings.TrimSpace(input.Area) == "" || strings.TrimSpace(input.Question) == "" || strings.TrimSpace(input.Reason) == "" {
			return toolJSON(map[string]string{toolErrorKey: "area, question and reason are required"})
		}
		w.mu.Lock()
		w.ledger.Focus = input
		w.mu.Unlock()
		if err := w.tick(fmt.Sprintf("Investigating %s: %s", input.Area, input.Question)); err != nil {
			return fantasy.ToolResponse{}, err
		}
		return toolJSON(map[string]bool{"updated": true})
	})
}
