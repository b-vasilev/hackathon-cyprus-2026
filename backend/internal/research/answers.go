package research

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/fantasy"
)

type answerInput struct {
	Responsibility string         `json:"responsibility"`
	Repository     string         `json:"repository"`
	Role           string         `json:"role"`
	Evidence       []roleEvidence `json:"evidence"`
}

// Save one verified role and its citations together. Other roles stay pending;
// missing research cannot be mistaken for a completed responsibility.
func (w *workspace) saveAnswer(input answerInput) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(input.Responsibility) == "" || len(input.Evidence) == 0 {
		return errors.New("provide a responsibility and at least one exact source range with its finding")
	}
	boundary := researchBoundary{Name: input.Responsibility, Repository: input.Repository}
	for _, b := range w.plan {
		if b.Name == input.Responsibility {
			if b.Repository != input.Repository {
				return errors.New("responsibility already belongs to a different repository")
			}
			boundary = b
			break
		}
	}
	var refs *[]roleEvidence
	switch input.Role {
	case "implementation":
		refs = &boundary.Implementation
	case "callers":
		refs = &boundary.Callers
	case "contracts":
		refs = &boundary.Contracts
	case "validation":
		refs = &boundary.Validation
	default:
		return errors.New("role must be implementation, callers, contracts or validation")
	}
	var notes strings.Builder
	for _, ref := range input.Evidence {
		doc, ok := w.documents[ref.Path]
		if !ok || doc.SnapshotID != input.Repository || ref.StartLine < 1 || ref.EndLine < ref.StartLine || ref.EndLine > doc.LineCount || strings.TrimSpace(ref.Finding) == "" {
			return fmt.Errorf("invalid source range or finding: %s", ref.Path)
		}
		fmt.Fprintf(&notes, "%s [%s:L%d-L%d]\n", ref.Finding, doc.ID, ref.StartLine, ref.EndLine)
		if !slices.Contains(*refs, ref) {
			*refs = append(slices.Clone(*refs), ref)
		}
	}
	if _, err := w.evidenceRanges(notes.String()); err != nil {
		return err
	}
	boundary.Gaps = slices.DeleteFunc(slices.Clone(boundary.Gaps), func(g roleGap) bool { return g.Role == input.Role })
	patch := planInput{Boundaries: []researchBoundary{boundary}}
	for _, c := range w.inventory.Candidates {
		if c.Repository != input.Repository || c.Kind == "repository_reference" || !boundaryTouches(boundary, c) || (w.task != nil && !w.task.ownsCandidate(c.ID)) {
			continue
		}
		// Preserve other responsibilities already attached to this candidate.
		d := candidateDisposition{CandidateID: c.ID, Class: "product_code", Disposition: "included", Reason: "Mapped to source-cited responsibility evidence"}
		for _, old := range w.ledger.Plan.Candidates {
			if old.CandidateID == c.ID && old.Disposition == "included" {
				d = old
				d.Boundaries = slices.Clone(old.Boundaries)
			}
		}
		if !slices.Contains(d.Boundaries, boundary.Name) {
			d.Boundaries = append(d.Boundaries, boundary.Name)
		}
		patch.Candidates = append(patch.Candidates, d)
	}
	if err := w.setPlanLocked(patch); err != nil {
		return err
	}
	finding := researchFinding{Name: input.Responsibility + "/" + input.Role, Notes: notes.String()}
	if !slices.Contains(w.ledger.Findings, finding) {
		w.ledger.Findings = append(w.ledger.Findings, finding)
	}
	w.submittedNotes = w.findingsText(w.summaryNotes)
	return nil
}

func (w *workspace) answerTool() fantasy.AgentTool {
	return fantasy.NewAgentTool("save_research_answer", "Save a verified answer immediately after reading its source. Provide responsibility, repository ID, one role (implementation/callers/contracts/validation), and evidence with exact path/start_line/end_line/finding. This atomically saves citations and plan evidence and maps matching assigned candidates. Other roles remain pending. No separate retain_research_findings or set_research_plan call is needed for this answer. Add evidence to the same responsibility as you investigate its remaining roles.", func(_ context.Context, input answerInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
		if err := w.saveAnswer(input); err != nil {
			if eventErr := w.tick("Answer could not be saved: " + err.Error()); eventErr != nil {
				return fantasy.ToolResponse{}, eventErr
			}
			return toolJSON(map[string]string{toolErrorKey: err.Error()})
		}
		if err := w.tick("Saved answer: " + input.Responsibility + "/" + input.Role); err != nil {
			return fantasy.ToolResponse{}, err
		}
		return toolJSON(map[string]any{"saved": true, "responsibility": input.Responsibility, "role": input.Role})
	})
}
