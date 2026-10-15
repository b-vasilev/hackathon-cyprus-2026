package research

import (
	"fmt"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

type attemptBudget struct {
	limit     int
	stopped   bool
	recovery  bool
	seen      map[string][]source.LineRange
	decisions []BudgetDecision
}

func (w *workspace) activeLimit() int {
	if w.budget != nil {
		return w.budget.limit
	}
	return w.turnBudget
}

func (w *workspace) startBudget() {
	w.budget = &attemptBudget{limit: min(30, w.turnBudget), seen: map[string][]source.LineRange{}}
	w.evidenceProgress() // Restored evidence must not earn another extension.
	w.budget.decisions = []BudgetDecision{{Limit: w.budget.limit, Kind: "started", Reason: fmt.Sprintf("Start with %d turns. Extend by up to 10 only for new read-and-cited source evidence and reachable questions. Hard limit: %d turns or 30 minutes, whichever comes first.", w.budget.limit, w.turnBudget)}}
}

// Only new source ranges attached to planned proof roles count. Searches,
// classifications, renamed areas and repeated findings do not earn more turns.
func (w *workspace) evidenceProgress() int {
	state := w.state()
	newLines := 0
	for _, area := range state.Areas {
		for _, q := range area.Questions {
			for _, ref := range q.Evidence {
				seen := w.budget.seen[ref.DocumentID]
				previous := rangeLines(seen)
				merged := mergeRanges(append(seen, source.LineRange{StartLine: ref.StartLine, EndLine: ref.EndLine}))
				newLines += rangeLines(merged) - previous
				w.budget.seen[ref.DocumentID] = merged
			}
		}
	}
	return newLines
}

func rangeLines(ranges []source.LineRange) int {
	count := 0
	for _, span := range ranges {
		count += span.EndLine - span.StartLine + 1
	}
	return count
}

func reachableQuestions(state State) int {
	count := 0
	for _, area := range state.Areas {
		for _, q := range area.Questions {
			if q.Status == "pending" {
				count++
			}
		}
	}
	return count
}

// reviewBudget runs between model passes, never in a tool goroutine.
func (w *workspace) reviewBudget() bool {
	if w.budget == nil || w.budget.stopped {
		return false
	}
	b := w.budget
	decision := BudgetDecision{Turn: w.turn, Limit: b.limit, Kind: "stopped"}
	state := w.state()
	reachable := reachableQuestions(state)
	newLines := w.evidenceProgress()
	switch {
	case w.turn >= w.turnBudget:
		decision.Reason = fmt.Sprintf("Hard limit reached: %d/%d turns. Remaining questions are saved for continuation.", w.turn, w.turnBudget)
	case !w.deadline.IsZero() && time.Until(w.deadline) <= 30*time.Second:
		decision.Reason = "Time limit reached; reserve the final 30 seconds for saving and checking the partial result."
	case reachable == 0:
		decision.Reason = "No reachable unanswered questions remain in the current plan. Source gaps stay visible."
	case newLines > 0:
		b.limit = min(b.limit+10, w.turnBudget)
		decision.Kind, decision.Limit = "extended", b.limit
		decision.Reason = fmt.Sprintf("Added %d turns: %d new source lines were read, cited and attached to planned evidence; %d reachable questions remain.", b.limit-state.Budget.Limit, newLines, reachable)
	case !b.recovery:
		b.recovery = true
		b.limit = min(b.limit+10, w.turnBudget)
		decision.Kind, decision.Limit = "recovery", b.limit
		decision.Reason = fmt.Sprintf("No new verified evidence. Allow one final %d-turn attempt with a different search or independent area; %d reachable questions remain. Repeated bookkeeping will not extend the budget.", b.limit-state.Budget.Limit, reachable)
	default:
		decision.Reason = "No new read-and-cited evidence after the recovery attempt. Stop and save the unresolved questions."
	}
	b.decisions = append(b.decisions, decision)
	b.stopped = decision.Kind == "stopped"
	return !b.stopped
}
