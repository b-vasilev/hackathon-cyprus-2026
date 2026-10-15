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
	block     int
	pending   int
}

func (w *workspace) capTurns(turns int) int {
	if w.turnBudget > 0 {
		return min(turns, w.turnBudget)
	}
	return turns
}

func (w *workspace) activeLimit() int {
	if w.budget != nil {
		return w.budget.limit
	}
	return w.turnBudget
}

func (w *workspace) startBudget() {
	w.budget = &attemptBudget{limit: w.capTurns(30), block: 10, seen: map[string][]source.LineRange{}}
	w.evidenceProgress() // Restored evidence must not earn another extension.
	w.budget.pending = reachableQuestions(w.state())
	reason := fmt.Sprintf("Start with %d turns. Extend only when new read-and-cited evidence reduces the open question count. Time limit: 60 minutes.", w.budget.limit)
	if w.turnBudget > 0 {
		reason = fmt.Sprintf("Start with %d turns. Hard limit: %d turns or 60 minutes, whichever comes first.", w.budget.limit, w.turnBudget)
	}
	w.budget.decisions = []BudgetDecision{{Limit: w.budget.limit, Kind: "started", Reason: reason}}
}

func (w *workspace) startParallelBudget(workers int) {
	w.startBudget()
	w.budget.block = max(10, workers*taskTurns)
	w.budget.limit = w.capTurns(max(30, workers*taskTurns))
	w.budget.decisions = []BudgetDecision{{Limit: w.budget.limit, Kind: "started", Reason: fmt.Sprintf("Shared allowance: %d turns for up to %d workers. Extend only when new verified evidence reduces the open question count. Time limit: 60 minutes.", w.budget.limit, workers)}}
}

// Only new source ranges attached to planned proof roles count. Searches,
// classifications, renamed areas and repeated findings do not earn more turns.
// New evidence alone is insufficient: the open question count must also fall.
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

// Refill reserved worker windows as soon as saved evidence earns an extension.
// Waiting for every worker to finish turns each budget block into a slow wave.
// Without verified progress, keep the normal end-of-window recovery decision.
func (w *workspace) refillParallelBudget() bool {
	if w.budget == nil || w.budget.stopped {
		return false
	}
	state := w.state()
	if reachableQuestions(state) >= w.budget.pending {
		return false
	}
	for _, area := range state.Areas {
		for _, q := range area.Questions {
			for _, ref := range q.Evidence {
				if !RangeCovered(w.budget.seen[ref.DocumentID], ref.StartLine, ref.EndLine) {
					previous := w.activeLimit()
					return w.reviewBudget() && w.activeLimit() > previous
				}
			}
		}
	}
	return false
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
	case w.turnBudget > 0 && w.turn >= w.turnBudget:
		decision.Reason = fmt.Sprintf("Hard limit reached: %d/%d turns. Remaining questions are saved for continuation.", w.turn, w.turnBudget)
	case !w.deadline.IsZero() && time.Until(w.deadline) <= 30*time.Second:
		decision.Reason = "Time limit reached; reserve the final 30 seconds for saving and checking the partial result."
	case reachable == 0:
		decision.Reason = "No reachable unanswered questions remain in the current plan. Source gaps stay visible."
	case newLines > 0 && reachable < b.pending:
		// A single closed question must not buy a whole 480-turn worker wave.
		extension := min(max(b.block, 10), max(10, (b.pending-reachable)*taskTurns))
		b.limit = w.capTurns(b.limit + extension)
		decision.Kind, decision.Limit = "extended", b.limit
		decision.Reason = fmt.Sprintf("Added %d turns: %d new source lines were verified and open questions decreased from %d to %d.", b.limit-state.Budget.Limit, newLines, b.pending, reachable)
	case !b.recovery:
		b.recovery = true
		b.limit = w.capTurns(b.limit + min(max(b.block, 10), 30))
		decision.Kind, decision.Limit = "recovery", b.limit
		decision.Reason = fmt.Sprintf("Research is not converging: open questions %d -> %d; %d new verified source lines. Allow one %d-turn recovery to close existing questions and save findings without expanding scope.", b.pending, reachable, newLines, b.limit-state.Budget.Limit)
	default:
		decision.Reason = "Research did not converge after the recovery attempt. Stop and save the unresolved questions as partial coverage."
	}
	// Growing the plan must not make the next extension easier to earn.
	b.pending = min(b.pending, reachable)
	b.decisions = append(b.decisions, decision)
	b.stopped = decision.Kind == "stopped"
	return !b.stopped
}
