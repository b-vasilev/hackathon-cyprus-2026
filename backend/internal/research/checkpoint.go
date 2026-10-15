package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/fantasy"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// Checkpoint reads the existing immutable findings JSON. Context is retained for
// compatibility; source excerpts are rebuilt from verified original documents.
type Checkpoint struct {
	Context   string                 `json:"context"`
	Ledger    Ledger                 `json:"ledger"`
	Notes     string                 `json:"notes"`
	Snapshots []source.Snapshot      `json:"snapshots"`
	Sessions  map[string]taskSession `json:"sessions,omitempty"`
}

type taskSession struct {
	Turns [][]fantasy.Message `json:"turns"`
}

// Run researches the collected pinned sources with a bounded model budget.
func (r *Runner) Run(ctx context.Context, config llmgateway.Config, modelID string, collection source.Collection, progress func(Progress) error) (Result, error) {
	return r.run(ctx, config, modelID, collection, nil, progress)
}

// RunFromCheckpoint resumes partial research. A complete checkpoint is checked
// against the original source collection and reused without a model call.
func (r *Runner) RunFromCheckpoint(ctx context.Context, config llmgateway.Config, modelID string, collection source.Collection, checkpoint Checkpoint, progress func(Progress) error) (Result, error) {
	return r.run(ctx, config, modelID, collection, &checkpoint, progress)
}

func (w *workspace) restore(checkpoint Checkpoint) error {
	w.sessions = maps.Clone(checkpoint.Sessions)
	if !slices.Contains([]string{researchComplete, researchPartial, "in_progress"}, checkpoint.Ledger.Status) {
		return errors.New("research checkpoint has an unsupported status")
	}
	if err := w.restoreReads(checkpoint); err != nil {
		return err
	}
	// Deep-copy so continuation cannot mutate the previous immutable attempt.
	data, err := json.Marshal(checkpoint.Ledger)
	if err != nil {
		return fmt.Errorf("encode research checkpoint: %w", err)
	}
	var ledger Ledger
	if decodeErr := json.Unmarshal(data, &ledger); decodeErr != nil {
		return fmt.Errorf("decode research checkpoint: %w", decodeErr)
	}
	w.ledger = ledger
	w.ledger.Inventory = w.inventory
	w.ledger.Status = "in_progress"
	w.ledger.Frontier = nil
	for _, search := range ledger.Searches {
		w.searches = append(w.searches, searchedScope{Pattern: search.Pattern, Glob: search.Glob})
	}
	if err := w.setPlan(ledger.Plan); err != nil {
		return fmt.Errorf("restore research plan: %w", err)
	}
	prefix := w.findingsText("")
	w.summaryNotes = strings.TrimPrefix(checkpoint.Notes, prefix)
	w.submittedNotes = w.findingsText(w.summaryNotes)
	if strings.TrimSpace(w.submittedNotes) != "" {
		if _, err := w.evidenceContext(w.submittedNotes); err != nil {
			return fmt.Errorf("restore research findings: %w", err)
		}
	} else if checkpoint.Ledger.Status == researchComplete {
		return errors.New("completed research checkpoint has no source-backed findings")
	}
	if checkpoint.Ledger.Status == researchComplete {
		if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchComplete}); err != nil {
			return fmt.Errorf("recheck completed research: %w", err)
		}
	}
	return nil
}

func (w *workspace) restoreReads(checkpoint Checkpoint) error {
	if len(checkpoint.Snapshots) != len(w.inventory.Repositories) {
		return errors.New("research checkpoint repository scope differs from pinned sources")
	}
	seen := map[string]bool{}
	for _, snap := range checkpoint.Snapshots {
		index := slices.IndexFunc(w.inventory.Repositories, func(repo InventoryRepository) bool {
			return repo.ID == snap.ID && repo.Revision == snap.Revision && repo.Locator == snap.Locator
		})
		if index < 0 || seen[snap.ID] {
			return errors.New("research checkpoint repository revision or identity differs from pinned sources")
		}
		seen[snap.ID] = true
		for _, file := range snap.Coverage.Files {
			doc, ok := w.documents[snap.ID+"/"+file.Path]
			if !ok || doc.ID != file.ID {
				return errors.New("research checkpoint document identity differs from pinned sources")
			}
			for _, span := range file.ReadRanges {
				if span.StartLine < 1 || span.EndLine < span.StartLine || span.EndLine > doc.LineCount {
					return fmt.Errorf("research checkpoint has invalid read range: %s", file.Path)
				}
			}
			w.reads[doc.ID] = slices.Clone(file.ReadRanges)
		}
	}
	return nil
}
