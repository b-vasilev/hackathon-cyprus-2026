package analysis

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

const (
	SharedExecutionTaskID = "shared-execution-completion"
	ReconciliationTaskID  = "final-reconciliation"
	defaultWindowBytes    = 64 << 10
	recentHistoryBytes    = 20 << 10
)

//go:embed prompts/analysis.md
var analysisPrompt string

// Runner builds the canonical model through durable tool mutations. It never
// executes repository code or sends the full source/research/model as a prompt.
type Runner struct {
	Policy         llmgateway.NetworkPolicy
	Limiter        *llmgateway.Limiter
	MaxTurns       int
	WindowSteps    int
	MaxWindowBytes int
}

var errAnalysisWindowFull = errors.New("analysis history window is full; accepted work is checkpointed")

func (r *Runner) Run(ctx context.Context, config llmgateway.Config, modelID string, input Input, persist Persist, progress ProgressFunc) (result Result, err error) {
	if persist == nil {
		return result, errors.New("incremental analysis requires durable checkpoint storage")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	input = prepareAnalysisInput(input)
	store, err := NewModelStore(input, persist)
	if err != nil {
		return result, err
	}
	defer func() {
		result.Checkpoint = store.Checkpoint()
		result.Model = result.Checkpoint.Model
		result.Collection = store.Collection()
	}()
	if store.Checkpoint().Finalized {
		return result, nil
	}
	session := &analysisSession{store: store, input: input, progress: progress}
	if err := session.ensureTasks(ctx); err != nil {
		return result, err
	}
	client, err := r.Policy.Client(config, r.Limiter)
	if err != nil {
		return result, fmt.Errorf("analysis connection: %w", err)
	}
	client.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
	defer client.CloseIdleConnections()
	model, err := languageModel(ctx, config, modelID, client)
	if err != nil {
		return result, err
	}
	tools, err := session.tools()
	if err != nil {
		return result, err
	}
	agent := fantasy.NewAgent(model, fantasy.WithSystemPrompt(analysisPrompt), fantasy.WithTools(tools...))
	if err := session.runWindows(ctx, agent, config, modelID, r); err != nil {
		return result, err
	}
	return result, nil
}

func (s *analysisSession) tools() ([]fantasy.AgentTool, error) {
	sources, err := SourceTools(s.input.Collection, s.store.RecordRead)
	if err != nil {
		return nil, err
	}
	findings, err := FindingTools(s.input.Findings)
	if err != nil {
		return nil, err
	}
	all := append(s.modelTools(), sources...)
	all = append(all, findings...)
	for i, tool := range all {
		all[i] = observedAnalysisTool{AgentTool: tool, session: s}
	}
	return all, nil
}

func prepareAnalysisInput(input Input) Input {
	input.Tasks = slices.Clone(input.Tasks)
	for i := range input.Tasks {
		if input.Tasks[i].Status == "" {
			input.Tasks[i].Status = TaskPending
		}
	}
	if len(input.Tasks) == 0 && input.Checkpoint == nil {
		for _, snapshot := range input.Collection.Snapshots {
			input.Tasks = append(input.Tasks, Task{ID: "repository-" + snapshot.ID, Title: "Repository " + snapshot.Locator, Question: "Establish this repository's verified responsibilities, manifests, source ownership, entry points and concrete contracts with other supplied repositories. A repository is not automatically a service. Reuse shared canonical entities.", Status: TaskPending})
		}
		for _, finding := range input.Findings {
			sum := sha256.Sum256([]byte(finding.ID))
			title := finding.Title
			if strings.TrimSpace(title) == "" {
				title = "Research area " + finding.ID
			}
			input.Tasks = append(input.Tasks, Task{ID: fmt.Sprintf("finding-%x", sum[:8]), Title: title, Question: "Model the material responsibilities, supported variants and source-backed behavior in this finding. Inspect exact original sources and discover omitted shared implementations and cross-area dependencies. Save distinct meaningful components and scenarios; retain explicit evidence questions.", FindingIDs: []string{finding.ID}, Status: TaskPending})
		}
	}
	return input
}

func requiredAnalysisTasks() []Task {
	return []Task{
		{ID: "product-context", Title: "Product scope and core concerns", Question: "Establish source-backed product purpose, intended users, capabilities, boundaries and major outcomes. Reconcile technology uses, domain/data, runtime and operations where materially supported. Reuse shared IDs; distinguish unknown research from product limitations. Preserve important modules and variants rather than compressing an entire engine into one object.", Status: TaskPending},
		{ID: SharedExecutionTaskID, Title: "Shared execution, completion and contracts", Question: "For every important agent/generator/task family, follow target-specific registrations through shared base implementations and actual callers. Verify prepared inputs, invocation, generated candidates, accepted outputs, rejected or unresolved/cannot-resolve results, validation/acceptance gates, repair/retry bounds and stop rules. Follow materially different variants and both cross-repository endpoints. Separate generation, acceptance and deployment completion. Inspect code behind generic wrappers, not only registries or existing notes. Save relevant objects, contracts, claims and ordered scenarios; create material follow-up tasks for newly found areas.", Status: TaskPending},
		{ID: ReconciliationTaskID, Title: "Canonical reconciliation", Question: "Reconcile all saved areas into one coherent canonical model. Resolve shared IDs, hierarchy, cross-area references, technology attribution, important variants, evidence and open questions without erasing researched detail. Verify shared execution/completion contracts and generation versus deployment. Set title, tagline and summary from the verified shared facts, inspect validation issues, then finalize the server-assembled model. Do not print the whole model.", Status: TaskPending},
	}
}

func (s *analysisSession) ensureTasks(ctx context.Context) error {
	cp := s.store.Checkpoint()
	var missing []Task
	for _, required := range requiredAnalysisTasks() {
		if !slices.ContainsFunc(cp.Tasks, func(t Task) bool { return t.ID == required.ID }) {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return s.store.AddTasks(ctx, missing)
	}
	return nil
}

func nextAnalysisTask(tasks []Task) (Task, bool) {
	for _, status := range []string{TaskActive, TaskPending} {
		for _, task := range tasks {
			if task.Status == status && task.ID != SharedExecutionTaskID && task.ID != ReconciliationTaskID {
				return task, true
			}
		}
	}
	for _, id := range []string{SharedExecutionTaskID, ReconciliationTaskID} {
		for _, task := range tasks {
			if task.ID == id && task.Status != TaskDone {
				return task, true
			}
		}
	}
	for _, task := range tasks {
		if task.ID == ReconciliationTaskID {
			return task, true
		}
	}
	return Task{}, false
}

func (s *analysisSession) runWindows(ctx context.Context, agent fantasy.Agent, config llmgateway.Config, modelID string, runner *Runner) error {
	maxTurns := runner.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 240
	}
	windowSteps := runner.WindowSteps
	if windowSteps <= 0 {
		windowSteps = 6
	}
	windowBytes := runner.MaxWindowBytes
	if windowBytes <= 0 {
		windowBytes = defaultWindowBytes
	}
	var recent []fantasy.Message
	previousArea := ""
	stalled := 0
	for s.turns < maxTurns {
		checkpoint := s.store.Checkpoint()
		if checkpoint.Finalized {
			return nil
		}
		task, ok := nextAnalysisTask(checkpoint.Tasks)
		if !ok {
			return errors.New("analysis has no area to reconcile; saved model remains resumable")
		}
		if task.Status != TaskActive {
			for _, old := range checkpoint.Tasks {
				if old.Status == TaskActive && old.ID != task.ID {
					if err := s.store.SetTaskStatus(ctx, old.ID, TaskPending); err != nil {
						return err
					}
				}
			}
			if err := s.store.SetTaskStatus(ctx, task.ID, TaskActive); err != nil {
				return err
			}
			task.Status = TaskActive
		}
		s.current = task
		if previousArea != task.ID {
			recent = nil
			stalled = 0
		}
		previousArea = task.ID
		checkpoint = s.store.Checkpoint()
		windowContext, err := BuildWindowContext(s.input, checkpoint, task)
		if err != nil {
			return err
		}
		windowContext += "\nThis is a fresh bounded model window. Earlier accepted parts/read evidence remain in the durable store; inspect paged tools for details. Additional instructions are available through read_analysis_instructions. Save useful model parts before ending this window; do not print a final JSON document."
		s.emit("Analyzing area: " + task.Title)
		before := analysisProgressKey(checkpoint)
		recent, err = s.runWindow(ctx, agent, config, modelID, windowContext, recent, min(windowSteps, maxTurns-s.turns), windowBytes)
		if err != nil {
			return err
		}
		current := s.store.Checkpoint()
		if current.Finalized {
			return nil
		}
		if analysisProgressKey(current) == before {
			stalled++
		} else {
			stalled = 0
		}
		if stalled >= 3 {
			return errors.New("analysis repeatedly returned without saving progress; accepted parts and source reads are preserved for retry")
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("analysis attempt ended: %w", err)
		}
	}
	return errors.New("analysis attempt budget ended with unfinished areas; accepted parts and source reads are preserved for retry")
}

func (s *analysisSession) runWindow(ctx context.Context, agent fantasy.Agent, config llmgateway.Config, modelID, prompt string, recent []fantasy.Message, maxSteps, maxBytes int) ([]fantasy.Message, error) {
	base, reported := s.turns, 0
	var stepStarted time.Time
	var completed []fantasy.StepResult
	var prepareFailure error
	report := func(steps []fantasy.StepResult) {
		for reported < len(steps) {
			step := steps[reported]
			cp := s.store.Checkpoint()
			s.turns = base + reported + 1
			if s.progress != nil {
				s.progress(Progress{Message: "Analysis model response received for " + s.current.Title, AreaID: s.current.ID, Step: s.turns, InputTokens: step.Usage.InputTokens, OutputTokens: step.Usage.OutputTokens, DurationMS: time.Since(stepStarted).Milliseconds(), ObjectCount: len(cp.Model.Objects), ClaimCount: len(cp.Model.Claims), RelationCount: len(cp.Model.Relations)})
			}
			reported++
		}
	}
	maxOutput := int64(6000)
	if selected, err := llmgateway.ModelByID(config, modelID); err == nil && selected.MaxOutputTokens > 0 {
		maxOutput = min(maxOutput, int64(selected.MaxOutputTokens))
	}
	response, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: prompt, Messages: recent, MaxOutputTokens: &maxOutput, ProviderOptions: providerOptions(config, modelID), StopWhen: []fantasy.StopCondition{func(steps []fantasy.StepResult) bool {
		cp := s.store.Checkpoint()
		return len(steps) >= maxSteps || cp.Finalized || !slices.ContainsFunc(cp.Tasks, func(t Task) bool { return t.ID == s.current.ID && t.Status == TaskActive })
	}}, PrepareStep: func(stepCtx context.Context, options fantasy.PrepareStepFunctionOptions) (context.Context, fantasy.PrepareStepResult, error) {
		report(options.Steps)
		completed = slices.Clone(options.Steps)
		data, encodeErr := json.Marshal(options.Messages)
		if encodeErr != nil {
			prepareFailure = fmt.Errorf("measure analysis history: %w", encodeErr)
			return stepCtx, fantasy.PrepareStepResult{}, prepareFailure
		}
		if len(data) > maxBytes {
			if len(options.Steps) == 0 && len(recent) == 0 {
				prepareFailure = fmt.Errorf("analysis window seed exceeds %d bytes; split the current question or increase the bounded window", maxBytes)
				return stepCtx, fantasy.PrepareStepResult{}, prepareFailure
			}
			return stepCtx, fantasy.PrepareStepResult{}, errAnalysisWindowFull
		}
		stepStarted = time.Now()
		s.emit("Model request started for " + s.current.Title)
		return stepCtx, fantasy.PrepareStepResult{}, nil
	}})
	if errors.Is(err, errAnalysisWindowFull) {
		s.emit("Starting a fresh bounded analysis window; saved work is retained")
		return recentAnalysisMessages(completed), nil
	}
	if err != nil {
		if prepareFailure != nil {
			return nil, prepareFailure
		}
		return nil, errors.New("analysis model request failed through the selected connection; accepted parts and source reads are saved for retry")
	}
	report(response.Steps)
	return recentAnalysisMessages(response.Steps), nil
}

func recentAnalysisMessages(steps []fantasy.StepResult) []fantasy.Message {
	if len(steps) == 0 {
		return nil
	}
	last := steps[len(steps)-1].Messages
	data, err := json.Marshal(last)
	if err == nil && len(data) <= recentHistoryBytes {
		return slices.Clone(last)
	}
	return []fantasy.Message{fantasy.NewUserMessage("The previous complete tool turn exceeded the carry-history budget. Accepted mutations and exact source read ranges are durably saved. Inspect saved entities or page only the relevant original ranges; no source text was clipped.")}
}

func analysisProgressKey(checkpoint Checkpoint) [32]byte {
	checkpoint.Revision = 0
	checkpoint.LastChange = nil
	data, _ := json.Marshal(checkpoint)
	return sha256.Sum256(data)
}
