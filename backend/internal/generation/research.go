package generation

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
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
	result, err := s.Research.Run(ctx, config, run.Model, collection, func(progress research.Progress) error {
		if _, checkErr := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision); checkErr != nil {
			return errors.New("the research connection was disabled or removed")
		}
		return s.recordResearchProgress(ctx, run.ID, progress)
	})
	if err != nil {
		return collection, "", fmt.Errorf("research product sources: %w", err)
	}
	if _, err := s.Content.WriteJSON("runs/"+run.ID+"/research/findings.json", map[string]any{"context": result.Context, "duration_ms": result.DurationMS, "model_turns": result.Steps, "tool_batches": result.ToolCalls, "mcp_reads": result.MCPReads, "input_tokens": result.InputTokens, "output_tokens": result.OutputTokens, "snapshots": result.Collection.Snapshots}); err != nil {
		return collection, "", fmt.Errorf("save research findings: %w", err)
	}
	if err := s.event(ctx, run.ID, "research_completed", "researching", fmt.Sprintf("Research completed: %d model turns, %d tool batches, %d Repomix reads; %d source files read.", result.Steps, result.ToolCalls, result.MCPReads, len(result.Collection.Documents))); err != nil {
		return collection, "", err
	}
	return result.Collection, result.Context, nil
}

func (s *Service) recordResearchProgress(ctx context.Context, runID string, progress research.Progress) error {
	if progress.Step == 0 {
		return s.event(ctx, runID, "research_activity", "researching", progress.Message)
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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit research usage: %w", err)
	}
	return nil
}

func validateResearchPassport(p passport.Passport, documents map[string]int, collection source.Collection, researched bool) error {
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
		found := false
		for _, span := range read[anchor.DocumentID] {
			if anchor.StartLine >= span.StartLine && anchor.EndLine <= span.EndLine {
				found = true
			}
		}
		if !found {
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
