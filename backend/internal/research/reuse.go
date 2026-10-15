package research

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// ReuseAcceptedCheckpoint rechecks an already accepted result for a downstream
// retry. It uses only the supplied pinned collection: no model, reader process,
// network connection or filesystem workspace is opened. Partial stays partial.
func (r *Runner) ReuseAcceptedCheckpoint(ctx context.Context, collection source.Collection, checkpoint Checkpoint) (Result, error) {
	started := time.Now()
	if err := validateAcceptedCheckpoint(checkpoint); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("reuse research checkpoint: %w", err)
	}
	w, err := checkpointWorkspace(collection)
	if err != nil {
		return Result{}, err
	}
	if restoreErr := w.restore(checkpoint); restoreErr != nil {
		return Result{}, restoreErr
	}
	// This is a mechanical recheck of a previous attempt's accepted stop. A new
	// continuation budget must not reinterpret the previous stop decision.
	if checkpoint.Ledger.Status == researchPartial {
		if checkErr := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchPartial, StopReason: checkpoint.Ledger.StopReason}); checkErr != nil {
			return Result{}, fmt.Errorf("recheck accepted partial research: %w", checkErr)
		}
	}
	w.ledger.StopReason = checkpoint.Ledger.StopReason
	frontier := slices.Clone(checkpoint.Ledger.Frontier)
	for _, gap := range w.ledger.Frontier {
		if !slices.Contains(frontier, gap) {
			frontier = append(frontier, gap)
		}
	}
	w.ledger.Frontier = frontier
	if contextErr := ctx.Err(); contextErr != nil {
		return Result{}, fmt.Errorf("reuse research checkpoint: %w", contextErr)
	}
	contextText, err := w.resultContext(w.orientation(collection))
	if err != nil {
		return Result{}, err
	}
	w.turnBudget = r.MaxTurns
	if w.turnBudget <= 0 {
		w.turnBudget = 120
	}
	w.budget = &attemptBudget{stopped: true, decisions: []BudgetDecision{{Kind: w.ledger.Status, Reason: "Reused previously accepted source evidence without new model turns. " + w.ledger.StopReason}}}
	state := w.state()
	return Result{Context: contextText, Plan: w.plan, Ledger: w.ledger, Notes: w.submittedNotes, Collection: w.readCollection(collection), DurationMS: time.Since(started).Milliseconds(), Progress: &state}, nil
}

func validateAcceptedCheckpoint(checkpoint Checkpoint) error {
	if strings.TrimSpace(checkpoint.Context) == "" {
		return errors.New("research checkpoint has no accepted context; continue research instead")
	}
	if !slices.Contains([]string{researchComplete, researchPartial}, checkpoint.Ledger.Status) {
		return errors.New("only accepted complete or partial research can be reused")
	}
	if checkpoint.Ledger.Status == researchPartial && strings.TrimSpace(checkpoint.Ledger.StopReason) == "" {
		return errors.New("accepted partial research needs its original stopping reason")
	}
	if checkpoint.Ledger.Status == researchComplete && len(checkpoint.Ledger.Frontier) > 0 {
		return errors.New("completed research checkpoint contains unresolved research obligations")
	}
	return nil
}

func checkpointWorkspace(collection source.Collection) (*workspace, error) {
	w := &workspace{documents: map[string]source.Document{}, reads: map[string][]source.LineRange{}}
	repositories := map[string]bool{}
	for _, snapshot := range collection.Snapshots {
		if repositories[snapshot.ID] {
			return nil, errors.New("pinned source collection repeats a repository identity")
		}
		repositories[snapshot.ID] = true
	}
	ids := map[string]bool{}
	for _, doc := range collection.Documents {
		file := doc.SnapshotID + "/" + doc.Path
		if !repositories[doc.SnapshotID] || !filepath.IsLocal(file) || !filepath.IsLocal(doc.Path) || doc.ID == "" || ids[doc.ID] {
			return nil, errors.New("pinned source collection has an invalid document identity or path")
		}
		if _, exists := w.documents[file]; exists {
			return nil, errors.New("pinned source collection repeats a document path")
		}
		if err := validatePinnedDocument(doc); err != nil {
			return nil, err
		}
		ids[doc.ID] = true
		w.documents[file] = doc
		w.paths = append(w.paths, file)
	}
	slices.Sort(w.paths)
	w.inventory = buildInventory(collection)
	return w, nil
}

func validatePinnedDocument(doc source.Document) error {
	if doc.LineCount != len(strings.Split(doc.Content, "\n")) {
		return fmt.Errorf("pinned source line count changed: %s", doc.Path)
	}
	if doc.ContentHash != "" {
		hash := sha256.Sum256([]byte(doc.Content))
		if hex.EncodeToString(hash[:]) != doc.ContentHash {
			return fmt.Errorf("pinned source content changed: %s", doc.Path)
		}
	}
	return nil
}
