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
	if s.Research == nil {
		return nil, analysis.SynthesisTasks(), nil
	}
	checkpoint, err := s.latestAnalysisResearch(ctx, runID)
	if err != nil {
		return nil, nil, err
	}
	brief, err := synthesisBrief(checkpoint, collection)
	if err != nil {
		return nil, nil, err
	}
	findings := splitFinding("Architecture synthesis brief", brief)
	var prefix strings.Builder
	for index, finding := range checkpoint.Ledger.Findings {
		title := fmt.Sprintf("%s [research record %d]", finding.Name, index+1)
		findings = append(findings, splitFinding(title, finding.Notes)...)
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
	return findings, analysis.SynthesisTasks(), nil
}

// synthesisBrief selects representative accepted findings for each subsystem.
// Complete research notes stay available; source bodies are never inspected.
func synthesisBrief(checkpoint *research.Checkpoint, collection source.Collection) (string, error) {
	documents := make(map[string]source.Document, len(collection.Documents))
	for _, doc := range collection.Documents {
		documents[doc.SnapshotID+"/"+doc.Path] = doc
	}
	var brief strings.Builder
	brief.WriteString("Saved research synthesis: representative findings, not an exhaustive file audit. Preserve explicit research gaps.\n")
	for _, repo := range collection.Snapshots {
		fmt.Fprintf(&brief, "Repository %s: %s at %s\n", repo.ID, repo.Locator, repo.Revision)
	}
	writeEvidence := func(role, file, finding string, start, end int) error {
		doc, ok := documents[file]
		if !ok {
			return fmt.Errorf("saved research cites an unknown source: %s", file)
		}
		fmt.Fprintf(&brief, "%s: %s [%s:L%d-L%d]\n", role, finding, doc.ID, start, end)
		return nil
	}
	for _, boundary := range checkpoint.Ledger.Plan.Boundaries {
		if boundary.Archived() {
			continue
		}
		fmt.Fprintf(&brief, "\n## %s (%s)\n", boundary.Name, boundary.Repository)
		refs := boundary.Implementation
		for _, role := range []string{"implementation", "callers", "contracts", "validation"} {
			switch role {
			case "callers":
				refs = boundary.Callers
			case "contracts":
				refs = boundary.Contracts
			case "validation":
				refs = boundary.Validation
			}
			if len(refs) > 0 {
				ref := refs[0]
				if err := writeEvidence(role, ref.Path, ref.Finding, ref.StartLine, ref.EndLine); err != nil {
					return "", err
				}
			}
		}
		for _, gap := range boundary.Gaps {
			fmt.Fprintf(&brief, "%s %s: %s\n", gap.Role, gap.Kind, gap.Reason)
		}
	}
	for _, handoff := range checkpoint.Ledger.Plan.Handoffs {
		fmt.Fprintf(&brief, "\nHandoff %s -> %s: %s\n", handoff.FromBoundary, handoff.ToBoundary, handoff.Contract)
		refs := handoff.Producer
		for _, role := range []string{"producer", "consumer", "wiring"} {
			switch role {
			case "consumer":
				refs = handoff.Consumer
			case "wiring":
				refs = handoff.Wiring
			}
			if len(refs) > 0 {
				ref := refs[0]
				if err := writeEvidence(role, ref.Path, ref.Finding, ref.StartLine, ref.EndLine); err != nil {
					return "", err
				}
			}
		}
	}
	return brief.String(), nil
}
