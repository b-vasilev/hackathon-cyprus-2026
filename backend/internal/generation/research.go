package generation

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func (s *Service) researchSources(ctx context.Context, run Run, collection source.Collection) (source.Collection, string, error) {
	if err := s.stage(ctx, run.ID, "researching"); err != nil {
		return collection, "", err
	}
	config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
	if err != nil {
		return collection, "", fmt.Errorf("load research connection: %w", err)
	}
	checkpoint, artifactPath, err := s.researchCheckpoint(ctx, run.ID)
	if err != nil {
		return collection, "", err
	}
	progress := func(progress research.Progress) error {
		if _, checkErr := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision); checkErr != nil {
			return errors.New("the research connection was disabled or removed")
		}
		if progress.Checkpoint != nil {
			if saveErr := s.Content.SaveCheckpoint(researchLivePath(artifactPath), progress.Checkpoint); saveErr != nil {
				return fmt.Errorf("save live research checkpoint: %w", saveErr)
			}
		}
		return s.recordResearchProgress(ctx, run.ID, progress)
	}
	var result research.Result
	reused := checkpoint != nil && checkpoint.Context != ""
	if checkpoint != nil {
		if eventErr := s.announceResearchResume(ctx, run.ID, reused); eventErr != nil {
			return collection, "", eventErr
		}
		if reused {
			result, err = s.Research.ReuseAcceptedCheckpoint(ctx, collection, checkpoint.Checkpoint)
		} else {
			result, err = s.Research.RunFromCheckpoint(ctx, config, run.Model, collection, checkpoint.Checkpoint, progress)
		}
	} else {
		result, err = s.Research.Run(ctx, config, run.Model, collection, progress)
	}
	if err == nil && result.Steps == 0 && result.Progress != nil {
		if saveErr := s.recordResearchProgress(ctx, run.ID, research.Progress{State: result.Progress}); saveErr != nil {
			return collection, "", saveErr
		}
	}
	// Persist the independent inventory, exact role evidence and unfinished
	// frontier even when a model/time budget ends before acceptance.
	payload := map[string]any{"context": result.Context, "notes": result.Notes, "plan": result.Plan, "ledger": result.Ledger, "duration_ms": result.DurationMS, "model_turns": result.Steps, "tool_batches": result.ToolCalls, "mcp_reads": result.MCPReads, "input_tokens": result.InputTokens, "output_tokens": result.OutputTokens, "snapshots": result.Collection.Snapshots, "accepted_research_reused": reused}
	payload["sessions"] = result.Sessions
	payload["progress"] = result.Progress
	if checkpoint != nil {
		payload["checkpoint_source"] = map[string]string{"path": checkpoint.Path, "hash": checkpoint.Hash}
	}
	if _, saveErr := s.Content.WriteJSON(artifactPath, payload); saveErr != nil {
		return collection, "", fmt.Errorf("save research findings: %w", saveErr)
	}
	if err != nil {
		return collection, "", fmt.Errorf("research product sources: %w", err)
	}
	eventKind, message := researchCompletionMessage(result, reused)
	if err := s.event(ctx, run.ID, eventKind, "researching", message); err != nil {
		return collection, "", err
	}
	return result.Collection, result.Context, nil
}

func (s *Service) recordResearchProgress(ctx context.Context, runID string, progress research.Progress) error {
	if progress.Step == 0 {
		return s.recordResearchActivity(ctx, runID, progress)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin research usage: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "INSERT INTO llm_calls(id,run_id,stage,attempt,prompt_version,input_tokens,output_tokens,duration_ms,created_at) VALUES(?,?,'researching',?,?,?,?,?,?)", rand.Text(), runID, progress.Step, PromptVersion, progress.InputTokens, progress.OutputTokens, progress.DurationMS, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("save research usage: %w", err)
	}
	if err := recordEvent(ctx, tx, runID, "model_call_completed", "researching", progress.Message); err != nil {
		return err
	}
	if progress.State != nil {
		if err := recordResearchState(ctx, tx, runID, progress.State); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research usage: %w", err)
	}
	return nil
}

// Store structured progress in the existing durable event stream. GetRun exposes
// only the latest state plus short activity messages, including on SSE reconnect.
func recordResearchState(ctx context.Context, writer eventWriter, runID string, state *research.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode research progress: %w", err)
	}
	return recordEvent(ctx, writer, runID, "research_plan", "researching", string(data))
}

func researchStateMessage(state research.State) string {
	checked := 0
	for _, area := range state.Areas {
		if area.Status == "checked" {
			checked++
		}
	}
	message := fmt.Sprintf("Research plan: %d of %d current areas checked; %d turns used.", checked, len(state.Areas), state.Budget.Used)
	if n := len(state.Budget.Decisions); n > 0 {
		decision := state.Budget.Decisions[n-1]
		if decision.Turn == state.Budget.Used {
			message += " " + decision.Reason
		}
	}
	return message
}

func validateResearchPassport(p passport.Passport, documents map[string]int, collection source.Collection, researched bool) error {
	if p.Views.Document == nil {
		return errors.New("newly generated passports require a version 2 document; create views.document with the required chapters and blocks")
	}
	if err := passport.Validate(p, documents); err != nil {
		return fmt.Errorf("validate passport: %w", err)
	}
	if !researched {
		return nil
	}
	read := map[string][]source.LineRange{}
	for _, snapshot := range collection.Snapshots {
		for _, file := range snapshot.Coverage.Files {
			read[file.ID] = file.ReadRanges
		}
	}
	for _, anchor := range p.Model.Evidence {
		if !research.RangeCovered(read[anchor.DocumentID], anchor.StartLine, anchor.EndLine) {
			return errors.New("passport cites source lines that the researcher did not read")
		}
	}
	return nil
}

func (s *Service) saveCitedCoverage(ctx context.Context, runID string, p passport.Passport, collection source.Collection) error {
	cited := map[string]bool{}
	for _, anchor := range p.Model.Evidence {
		cited[anchor.DocumentID] = true
	}
	for _, snapshot := range collection.Snapshots {
		for i := range snapshot.Coverage.Files {
			file := &snapshot.Coverage.Files[i]
			file.Cited = cited[file.ID]
			if file.Cited {
				snapshot.Coverage.CitedFiles++
			}
		}
		encoded, err := json.Marshal(snapshot.Coverage)
		if err != nil {
			return fmt.Errorf("encode citation coverage: %w", err)
		}
		if _, err := s.DB.ExecContext(ctx, "UPDATE source_snapshots SET coverage_json=? WHERE id=? AND run_id=?", string(encoded), snapshot.ID, runID); err != nil {
			return fmt.Errorf("save citation coverage: %w", err)
		}
	}
	return nil
}

func (s *Service) announceResearchResume(ctx context.Context, runID string, reused bool) error {
	message := "Continuing from saved source evidence and research findings."
	if reused {
		message = "Reusing verified research evidence for analysis."
	}
	if eventErr := s.event(ctx, runID, "research_resumed", "researching", message); eventErr != nil {
		return eventErr
	}
	return nil
}

func researchCompletionMessage(result research.Result, reused bool) (string, string) {
	eventKind, label := "research_completed", "Research completed"
	if result.Ledger.Status == "partial" {
		eventKind, label = "research_partial", "Partial research accepted with explicit research gaps"
		if len(result.Ledger.Frontier) > 0 {
			label += ": " + strings.Join(result.Ledger.Frontier[:min(2, len(result.Ledger.Frontier))], "; ")
		}
	}
	message := fmt.Sprintf("%s: %d model turns, %d tool batches, %d source reads; %d source files read; %d unresolved research obligations.", label, result.Steps, result.ToolCalls, result.MCPReads, len(result.Collection.Documents), len(result.Ledger.Frontier))
	if reused {
		message = fmt.Sprintf("Saved research evidence reused: %d source files read; %d unresolved research obligations. No additional research model calls.", len(result.Collection.Documents), len(result.Ledger.Frontier))
		if result.Ledger.Status == "partial" {
			message = "Saved partial research evidence reused: " + strings.TrimPrefix(message, "Saved research evidence reused: ")
		}
	}
	return eventKind, message
}

func (s *Service) recordResearchActivity(ctx context.Context, runID string, progress research.Progress) error {
	if progress.State != nil {
		if err := recordResearchState(ctx, s.DB, runID, progress.State); err != nil {
			return err
		}
		if !progress.Retry {
			return nil
		}
	}
	return s.event(ctx, runID, "research_activity", "researching", progress.Message)
}
