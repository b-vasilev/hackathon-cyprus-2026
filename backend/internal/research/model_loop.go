package research

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func (w *workspace) workFingerprint() [32]byte {
	reads := map[string][]source.LineRange{}
	for id, spans := range w.reads {
		reads[id] = mergeRanges(spans)
	}
	data, _ := json.Marshal(map[string]any{"reads": reads, "plan": w.ledger.Plan, "findings": w.ledger.Findings, "searches": w.ledger.Searches})
	return sha256.Sum256(data)
}

//nolint:gocyclo // Keep adaptive budget decisions and continuation in one bounded lifecycle.
func (w *workspace) investigate(ctx context.Context, agent fantasy.Agent, config llmgateway.Config, modelID, orientation string, progress func(Progress) error, result *Result) error {
	stalled := 0
	for w.turnBudget == 0 || result.Steps < w.turnBudget {
		if w.budget != nil && (result.Steps >= w.activeLimit() || (!w.deadline.IsZero() && time.Until(w.deadline) <= 30*time.Second)) {
			continued := w.reviewBudget()
			if err := w.publish(progress, w.budget.decisions[len(w.budget.decisions)-1].Reason); err != nil {
				return err
			}
			if !continued {
				break
			}
		}
		before := w.workFingerprint()
		callPrompt := orientation + "Investigate the complete supplied product scope and prepare source-backed findings."
		if result.Steps > 0 || w.submittedNotes != "" {
			callPrompt = w.continuationPrompt(orientation, max(0, w.turnBudget-result.Steps))
		}
		if err := w.modelPass(ctx, agent, config, modelID, callPrompt, progress, result); err != nil {
			return err
		}
		if w.checkedNotes != "" {
			return w.finishProgress(progress)
		}
		if before == w.workFingerprint() {
			stalled++
		} else {
			stalled = 0
		}
		if stalled >= 3 {
			if w.budget != nil {
				w.budget.stopped = true
				w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: "stopped", Reason: "The research model repeatedly stopped without progress while source questions remained. Saved findings and open scopes can be resumed."})
				break
			}
			return errors.New("research model repeatedly stopped without progress while source questions remained; saved findings and open scopes can be resumed")
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("research attempt time budget ended: %w", err)
		}
		if w.turnBudget == 0 || result.Steps < w.turnBudget {
			if err := w.event("Continuing source research: unresolved branches are still available within the time budget."); err != nil {
				return err
			}
		}
	}
	// At the real attempt boundary retain a useful partial result only when its
	// source ranges and citations pass the same coverage checks as a normal stop.
	if w.budget != nil {
		if !w.budget.stopped {
			w.reviewBudget()
		}
	} else if w.turnBudget > 0 {
		w.turn = w.turnBudget
	}
	if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchComplete}); err == nil {
		return w.finishProgress(progress)
	}
	if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchPartial, StopReason: "Attempt turn budget exhausted"}); err != nil {
		if publishErr := w.publish(progress, "Research stopped with unchecked evidence. Findings and open questions are saved for continuation."); publishErr != nil {
			return publishErr
		}
		reason := "Research budget ended"
		if w.budget != nil && len(w.budget.decisions) > 0 {
			reason = w.budget.decisions[len(w.budget.decisions)-1].Reason
		}
		return fmt.Errorf("%s Coverage checks were not accepted: %w", reason, err)
	}
	return w.finishProgress(progress)
}

func (w *workspace) finishProgress(progress func(Progress) error) error {
	reason := w.ledger.StopReason
	if w.ledger.Status == researchComplete {
		reason = "All questions in the current plan passed the source coverage checks. This does not guarantee exhaustive discovery."
	}
	if w.budget != nil {
		w.budget.stopped = true
		w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: w.ledger.Status, Reason: reason})
	}
	return w.publish(progress, reason)
}

func (w *workspace) modelPass(ctx context.Context, agent fantasy.Agent, config llmgateway.Config, modelID, callPrompt string, progress func(Progress) error, result *Result) error {
	base, reported := result.Steps, 0
	gateHeld := false
	release := func() {
		if gateHeld {
			<-w.stepGate
			gateHeld = false
		}
	}
	defer release()
	var stepStarted time.Time
	report := func(steps []fantasy.StepResult) error {
		for reported < len(steps) {
			release()
			usage := steps[reported].Usage
			step := base + reported + 1
			state := w.state()
			state.Budget.Used = step
			if err := progress(Progress{Message: fmt.Sprintf("Research model turn %d completed. Recorded usage: %d input tokens and %d output tokens.", step, usage.InputTokens, usage.OutputTokens), Step: step, InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, DurationMS: time.Since(stepStarted).Milliseconds(), State: &state}); err != nil {
				return fmt.Errorf("report research usage: %w", err)
			}
			result.InputTokens += usage.InputTokens
			result.OutputTokens += usage.OutputTokens
			result.Steps = step
			reported++
		}
		return nil
	}
	maxOutput := int64(14000)
	stop := func(steps []fantasy.StepResult) bool {
		return base+len(steps) >= w.activeLimit() || w.checkedNotes != "" || (!w.deadline.IsZero() && time.Until(w.deadline) <= 30*time.Second)
	}
	response, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: callPrompt, MaxOutputTokens: &maxOutput, ProviderOptions: providerOptions(config, modelID), StopWhen: []fantasy.StopCondition{stop}, PrepareStep: func(stepCtx context.Context, options fantasy.PrepareStepFunctionOptions) (context.Context, fantasy.PrepareStepResult, error) {
		if usageErr := report(options.Steps); usageErr != nil {
			return stepCtx, fantasy.PrepareStepResult{}, usageErr
		}
		if w.stepGate != nil {
			select {
			case w.stepGate <- struct{}{}:
				gateHeld = true
			case <-stepCtx.Done():
				return stepCtx, fantasy.PrepareStepResult{}, stepCtx.Err()
			}
		}
		w.turn = base + options.StepNumber + 1
		stepStarted = time.Now()
		hint := fmt.Sprintf("Attempt budget: turn %d; current allowance %d; hard limit %d. The runner alone extends the allowance in blocks of 10 for new read-and-cited planned evidence and reachable questions. The current allowance is a review point, not permission to finalize partial research. Save findings and plan evidence incrementally. Update set_research_focus before changing areas or questions. Continue independent useful work; do not repeat classifications or findings to earn turns.", w.turn, w.activeLimit(), w.turnBudget)
		if w.turnBudget == 0 {
			hint = fmt.Sprintf("Attempt turn %d; current allowance %d. No fixed turn cap. The runner extends the allowance for new verified evidence while reachable questions and time remain. Save findings and plan evidence incrementally.", w.turn, w.activeLimit())
		}
		if w.task != nil {
			hint = fmt.Sprintf("Assigned task turn %d/%d. Stay within this task. Save read-and-cited findings AND matching plan evidence incrementally. With three turns left, prioritize saving and validating the evidence already read. The coordinator merges your results and requeues unfinished work; you do not decide global completion.", w.turn, w.turnBudget)
		}
		if w.budget != nil && len(w.budget.decisions) > 0 {
			hint += "\nBudget decision: " + w.budget.decisions[len(w.budget.decisions)-1].Reason
		}
		budgetHint := fantasy.NewUserMessage(hint)
		messages := slices.Clone(options.Messages)
		messages = append(messages, budgetHint)
		return stepCtx, fantasy.PrepareStepResult{Messages: messages}, w.event(fmt.Sprintf("Research model turn %d started. Current allowance: %d.", w.turn, w.activeLimit()))
	}})
	if err != nil {
		return errors.New("repository research failed through the selected connection; check protocol support, endpoint availability and timeout")
	}
	return report(response.Steps)
}
