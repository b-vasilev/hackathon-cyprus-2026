package generation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/analysis"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// mergeAnalysisSources preserves every pinned body and adds verified research
// reads. The research result intentionally contains only its read documents.
func mergeAnalysisSources(full, researched source.Collection) (source.Collection, error) {
	documents := make(map[string]source.Document, len(full.Documents))
	for _, document := range full.Documents {
		documents[document.ID] = document
	}
	for _, document := range researched.Documents {
		original, ok := documents[document.ID]
		if !ok || original.SnapshotID != document.SnapshotID || original.ContentHash != document.ContentHash || original.Path != document.Path || original.LineCount != document.LineCount {
			return full, fmt.Errorf("researched source %s does not match the pinned collection", document.ID)
		}
	}
	snapshots := make(map[string]int, len(full.Snapshots))
	for index, snapshot := range full.Snapshots {
		snapshots[snapshot.ID] = index
	}
	for _, snapshot := range researched.Snapshots {
		index, ok := snapshots[snapshot.ID]
		if !ok || full.Snapshots[index].Revision != snapshot.Revision || full.Snapshots[index].Locator != snapshot.Locator {
			return full, errors.New("research snapshot differs from the pinned collection")
		}
		coverage := &full.Snapshots[index].Coverage
		files := make(map[string]int, len(coverage.Files))
		for fileIndex, file := range coverage.Files {
			files[file.ID] = fileIndex
		}
		for _, file := range snapshot.Coverage.Files {
			fileIndex, exists := files[file.ID]
			if !exists {
				files[file.ID] = len(coverage.Files)
				coverage.Files = append(coverage.Files, file)
				continue
			}
			target := &coverage.Files[fileIndex]
			if target.Path != file.Path {
				return full, fmt.Errorf("research file %s changed its source path", file.ID)
			}
			target.ReadRanges = append(target.ReadRanges, file.ReadRanges...)
			target.Cited = target.Cited || file.Cited
		}
		if snapshot.Coverage.Note != "" {
			coverage.Note = snapshot.Coverage.Note
		}
		coverage.ReadFiles, coverage.CitedFiles = 0, 0
		for _, file := range coverage.Files {
			if len(file.ReadRanges) > 0 {
				coverage.ReadFiles++
			}
			if file.Cited {
				coverage.CitedFiles++
			}
		}
	}
	return full, nil
}

// latestAnalysisResearch reads the current accepted attempt first. It never
// restarts research or uses the old giant context as a model prompt.
func (s *Service) latestAnalysisResearch(ctx context.Context, runID string) (*research.Checkpoint, error) {
	var attempt int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM generation_events WHERE run_id=? AND type='run_resumed'", runID).Scan(&attempt); err != nil {
		return nil, fmt.Errorf("read research attempt for analysis: %w", err)
	}
	root, err := os.OpenRoot(s.Content.Dir)
	if err != nil {
		return nil, fmt.Errorf("open analysis research findings: %w", err)
	}
	defer root.Close()
	for current := attempt; current >= 0; current-- {
		file, openErr := root.Open(researchArtifactPath(runID, current))
		if errors.Is(openErr, os.ErrNotExist) {
			continue
		}
		if openErr != nil {
			return nil, fmt.Errorf("open accepted analysis findings: %w", openErr)
		}
		const maxBytes = 32 << 20
		data, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
		closeErr := file.Close()
		if combined := errors.Join(readErr, closeErr); combined != nil {
			return nil, fmt.Errorf("read accepted analysis findings: %w", combined)
		}
		if len(data) > maxBytes {
			return nil, errors.New("accepted research findings exceed the size limit")
		}
		var checkpoint research.Checkpoint
		if decodeErr := json.Unmarshal(data, &checkpoint); decodeErr != nil {
			return nil, fmt.Errorf("decode accepted analysis findings: %w", decodeErr)
		}
		if strings.TrimSpace(checkpoint.Context) == "" || (checkpoint.Ledger.Status != "complete" && checkpoint.Ledger.Status != "partial") {
			return nil, errors.New("analysis requires accepted research findings, not an unfinished research checkpoint")
		}
		return &checkpoint, nil
	}
	return nil, errors.New("accepted research findings are missing; analysis did not restart research")
}

func analysisAreaID(prefix, value string) string {
	hash := sha256.Sum256([]byte(value))
	return prefix + "-" + hex.EncodeToString(hash[:8])
}

// splitFinding preserves all text, with UTF-8 boundaries and stable chunk IDs.
// Small records let finding tools page content without replaying the ledger.
func splitFinding(title, notes string) []analysis.Finding {
	const maxBytes = 6000
	var findings []analysis.Finding
	id := analysisAreaID("finding", title)
	for part := 0; len(notes) > 0; part++ {
		end := min(len(notes), maxBytes)
		for end < len(notes) && !utf8.RuneStart(notes[end]) {
			end--
		}
		findings = append(findings, analysis.Finding{ID: fmt.Sprintf("%s-%d", id, part+1), Title: title, Notes: notes[:end]})
		notes = notes[end:]
	}
	return findings
}

func (s *Service) analysisResearchInput(ctx context.Context, runID string, collection source.Collection) ([]analysis.Finding, []analysis.Task, error) {
	var findings []analysis.Finding
	var tasks []analysis.Task
	tasks = append(tasks, analysis.Task{ID: "product-context", Title: "Product purpose and boundary", Question: "Establish the evidenced product purpose, actors, outcomes, scope and supported limits. Reconcile final wording after the implementation areas.", Status: analysis.TaskPending})
	for _, snapshot := range collection.Snapshots {
		tasks = append(tasks, analysis.Task{ID: analysisAreaID("repository", snapshot.ID), Title: "Repository: " + snapshot.Locator, Question: "Account for this supplied repository's real responsibilities, source ownership and produced/consumed cross-repository contracts. A repository is not a runtime service.", Status: analysis.TaskPending})
	}
	if s.Research != nil {
		checkpoint, err := s.latestAnalysisResearch(ctx, runID)
		if err != nil {
			return nil, nil, err
		}
		byName := map[string][]string{}
		var prefix strings.Builder
		for index, finding := range checkpoint.Ledger.Findings {
			title := fmt.Sprintf("%s [research record %d]", finding.Name, index+1)
			parts := splitFinding(title, finding.Notes)
			findings = append(findings, parts...)
			for _, part := range parts {
				byName[finding.Name] = append(byName[finding.Name], part.ID)
			}
			fmt.Fprintf(&prefix, "\n## %s\n%s\n", finding.Name, finding.Notes)
		}
		summary := strings.TrimPrefix(checkpoint.Notes, prefix.String())
		findings = append(findings, splitFinding("Research summary", summary)...)
		coverage, err := json.Marshal(struct {
			Status     string   `json:"status"`
			StopReason string   `json:"stop_reason"`
			Frontier   []string `json:"frontier"`
		}{checkpoint.Ledger.Status, checkpoint.Ledger.StopReason, checkpoint.Ledger.Frontier})
		if err != nil {
			return nil, nil, fmt.Errorf("encode accepted research gaps: %w", err)
		}
		findings = append(findings, splitFinding("Accepted research coverage and unresolved questions", string(coverage))...)
		seen := map[string]bool{}
		for _, boundary := range checkpoint.Ledger.Plan.Boundaries {
			id := analysisAreaID("responsibility", boundary.Repository+"\n"+boundary.Name)
			if seen[id] {
				continue
			}
			seen[id] = true
			tasks = append(tasks, analysis.Task{ID: id, Title: boundary.Name, Question: "Explain this material responsibility in " + boundary.Repository + ": actual implementation, callers/wiring, tools, contracts, status and materially distinct variants. Use original source ranges to close gaps; missing investigation is not a product limitation.", FindingIDs: byName[boundary.Name], Status: analysis.TaskPending})
		}
	}
	for _, task := range []analysis.Task{
		{ID: analysis.SharedExecutionTaskID, Title: "Shared execution and completion contracts", Question: "Trace shared/base implementations behind specialized tasks and variants. Verify prepared inputs, invocation, candidates, accepted, rejected, cannot-resolve and repair outcomes, acceptance gates and retry/stop bounds where supported. Distinguish generation/validation/artifact persistence from remote deployment or publication."},
		{ID: "flows-variants", Title: "Main flows and material variants", Question: "Reconcile representative triggers, ordered stages, participants, handoff contracts and actual terminal outcomes; account for material variants without duplicating a whole flow per parameter combination."},
		{ID: "data-contracts", Title: "Data and artifact meaning", Question: "Reconcile core concepts, invariants, important artifact definitions/writers/readers/validation, persistence and source of truth."},
		{ID: "technology-attribution", Title: "Technology attribution", Question: "Attribute significant runtimes, frameworks, tools, agent tasks and specialized libraries to actual responsibilities and runtime/build/test/development scope. Verify version assertions."},
		{ID: "runtime-development-operations", Title: "Runtime and practical engineering", Question: "Preserve useful evidenced execution modes, startup/configuration, development checks, diagnostics and recovery conditions; record precise gaps without inventing production infrastructure or commands."},
		{ID: analysis.ReconciliationTaskID, Title: "Canonical reconciliation and overview", Question: "Reconcile identities, hierarchy, cross-area references, supported material variants and explicit research gaps. Set coherent title/tagline/summary last. Resolve validation issues and finalize the canonical model through the server tool."},
	} {
		task.Status = analysis.TaskPending
		tasks = append(tasks, task)
	}
	return findings, tasks, nil
}
