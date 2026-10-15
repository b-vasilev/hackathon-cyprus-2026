package generation

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/analysis"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// Analyzer is the bounded, tool-driven canonical model boundary.
type Analyzer interface {
	Run(context.Context, llmgateway.Config, string, analysis.Input, analysis.Persist, analysis.ProgressFunc) (analysis.Result, error)
}

// AnalysisState exposes real activity and counts from the last durable checkpoint.
type AnalysisState struct {
	AreaID             string `json:"area_id"`
	AreaTitle          string `json:"area_title"`
	Action             string `json:"action"`
	Step               int    `json:"step"`
	CheckpointRevision int    `json:"checkpoint_revision"`
	Objects            int    `json:"objects"`
	Claims             int    `json:"claims"`
	Relations          int    `json:"relations"`
}

func checkpointAnalysisState(checkpoint analysis.Checkpoint, state AnalysisState) AnalysisState {
	state.CheckpointRevision = checkpoint.Revision
	state.Objects = len(checkpoint.Model.Objects)
	state.Claims = len(checkpoint.Model.Claims)
	state.Relations = len(checkpoint.Model.Relations)
	for _, task := range checkpoint.Tasks {
		if task.Status == analysis.TaskActive {
			state.AreaID, state.AreaTitle = task.ID, task.Title
			return state
		}
		if task.ID == state.AreaID {
			state.AreaTitle = task.Title
		}
	}
	return state
}

func (s *Service) analyzeIncrementally(ctx context.Context, run Run, researched source.Collection) (passport.Model, source.Collection, error) {
	var empty passport.Model
	full, err := s.loadSavedSources(ctx, run.ID)
	if err != nil {
		return empty, researched, err
	}
	full, err = mergeAnalysisSources(full, researched)
	if err != nil {
		return empty, researched, err
	}
	findings, tasks, err := s.analysisResearchInput(ctx, run.ID, full)
	if err != nil {
		return empty, full, err
	}
	checkpoint, err := s.loadAnalysisCheckpoint(ctx, run.ID)
	if err != nil {
		return empty, full, err
	}
	config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
	if err != nil {
		return empty, full, fmt.Errorf("load incremental analysis connection: %w", err)
	}
	if run.RequestTimeoutSeconds > 0 {
		config.TimeoutSeconds = run.RequestTimeoutSeconds
	}
	input := analysis.Input{Collection: full, Findings: findings, Tasks: tasks, Instructions: incrementalAnalysisPrompt, Checkpoint: checkpoint}
	if checkpoint != nil {
		input.Tasks = checkpoint.Tasks
	}
	areaTitles := make(map[string]string, len(input.Tasks))
	for _, task := range input.Tasks {
		areaTitles[task.ID] = task.Title
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var mu sync.Mutex
	state := AnalysisState{Action: "Preparing incremental analysis."}
	if checkpoint != nil {
		state = checkpointAnalysisState(*checkpoint, state)
	}
	var progressErr error
	recordedUsage := map[int]bool{}
	persist := func(persistCtx context.Context, next analysis.Checkpoint) error {
		mu.Lock()
		defer mu.Unlock()
		accepted := checkpointAnalysisState(next, state)
		accepted.Action = "Saved analysis checkpoint."
		if err := s.saveAnalysisCheckpoint(persistCtx, run.ID, next, accepted); err != nil {
			return err
		}
		state = accepted
		for _, task := range next.Tasks {
			areaTitles[task.ID] = task.Title
		}
		return nil
	}
	progress := func(update analysis.Progress) {
		mu.Lock()
		defer mu.Unlock()
		if progressErr != nil {
			return
		}
		if _, connectionErr := s.Connections.Load(runCtx, run.ConnectionID, run.ConnectionRevision); connectionErr != nil {
			progressErr = errors.New("the incremental analysis connection is disabled or unavailable")
			cancel()
			return
		}
		state.Action, state.Step = update.Message, update.Step
		if update.AreaID != "" {
			state.AreaID = update.AreaID
			state.AreaTitle = areaTitles[update.AreaID]
			if state.AreaTitle == "" {
				state.AreaTitle = update.AreaID
			}
		}
		// Counts come only from Persist, never from a pending tool mutation.
		usage := update.Step > 0 && !recordedUsage[update.Step] && (update.InputTokens > 0 || update.OutputTokens > 0 || update.DurationMS > 0)
		if err := s.recordAnalysisProgress(runCtx, run.ID, state, update, usage); err != nil {
			progressErr = err
			cancel()
			return
		}
		if usage {
			recordedUsage[update.Step] = true
		}
	}
	result, err := s.Analysis.Run(runCtx, config, run.Model, input, persist, progress)
	mu.Lock()
	recordErr := progressErr
	mu.Unlock()
	if recordErr != nil {
		return empty, full, recordErr
	}
	if err != nil {
		return empty, full, fmt.Errorf("incremental canonical analysis: %w", err)
	}
	if !result.Checkpoint.Finalized {
		return empty, full, errors.New("incremental analysis did not finalize the canonical model")
	}
	return result.Model, result.Collection, nil
}

func (s *Service) recordAnalysisProgress(ctx context.Context, runID string, state AnalysisState, progress analysis.Progress, usage bool) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode analysis progress: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin analysis progress: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	if checkErr := tx.QueryRowContext(ctx, "SELECT status FROM generation_runs WHERE id=?", runID).Scan(&status); checkErr != nil {
		return fmt.Errorf("read generation before analysis progress: %w", checkErr)
	}
	if status != "queued" && status != "running" {
		return context.Canceled
	}
	if usage {
		if _, err := tx.ExecContext(ctx, "INSERT INTO llm_calls(id,run_id,stage,attempt,prompt_version,input_tokens,output_tokens,duration_ms,created_at) VALUES(?,?,'analyzing',?,?,?,?,?,?)", rand.Text(), runID, progress.Step, PromptVersion, progress.InputTokens, progress.OutputTokens, progress.DurationMS, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("save incremental analysis usage: %w", err)
		}
	}
	if err := recordEvent(ctx, tx, runID, "analysis_progress", "analyzing", string(encoded)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit analysis progress: %w", err)
	}
	return nil
}
