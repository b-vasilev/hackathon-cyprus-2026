package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

const (
	researchComplete = "complete"
	researchPartial  = "partial"
)

type roleEvidence struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Finding   string `json:"finding"`
}

type searchedScope struct {
	Pattern string `json:"pattern"`
	Glob    string `json:"glob"`
}

// A gap applies to one role. research_gap describes investigation, never a
// demonstrated limitation in the product.
type roleGap struct {
	Role     string          `json:"role"`
	Kind     string          `json:"kind"`
	Reason   string          `json:"reason"`
	Searches []searchedScope `json:"searches"`
}

type researchBoundary struct {
	Name           string         `json:"name"`
	Repository     string         `json:"repository"`
	Implementation []roleEvidence `json:"implementation"`
	Callers        []roleEvidence `json:"callers"`
	Contracts      []roleEvidence `json:"contracts"`
	Validation     []roleEvidence `json:"validation"`
	Gaps           []roleGap      `json:"gaps"`
}

type candidateDisposition struct {
	CandidateID string   `json:"candidate_id"`
	Class       string   `json:"class"`
	Disposition string   `json:"disposition"`
	Reason      string   `json:"reason"`
	Boundaries  []string `json:"boundaries"`
	HandoffID   string   `json:"handoff_id"`
}

type repositoryHandoff struct {
	ID           string         `json:"id"`
	FromBoundary string         `json:"from_boundary"`
	ToBoundary   string         `json:"to_boundary"`
	Contract     string         `json:"contract"`
	Producer     []roleEvidence `json:"producer"`
	Consumer     []roleEvidence `json:"consumer"`
	Wiring       []roleEvidence `json:"wiring"`
	Gaps         []roleGap      `json:"gaps"`
}

type planInput struct {
	Replace           bool                   `json:"replace"`
	Boundaries        []researchBoundary     `json:"boundaries"`
	Candidates        []candidateDisposition `json:"candidates"`
	Handoffs          []repositoryHandoff    `json:"handoffs"`
	Reclassifications []reclassification     `json:"reclassifications"`
}

type reclassification struct {
	Boundary string `json:"boundary"`
	Reason   string `json:"reason"`
}

type researchFinding struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

type coverageInput struct {
	Notes      string `json:"notes"`
	Status     string `json:"status"`
	StopReason string `json:"stop_reason"`
}

type Ledger struct {
	Inventory  Inventory         `json:"inventory"`
	Plan       planInput         `json:"plan"`
	Status     string            `json:"status"`
	Frontier   []string          `json:"frontier"`
	History    []planInput       `json:"plan_history,omitempty"`
	Findings   []researchFinding `json:"findings"`
	Searches   []searchRecord    `json:"searches"`
	StopReason string            `json:"stop_reason,omitempty"`
	Focus      Focus             `json:"focus"`
}

type searchRecord struct {
	Pattern string `json:"pattern"`
	Glob    string `json:"glob"`
	Matches int    `json:"matches"`
}

func (b researchBoundary) roles() map[string][]roleEvidence {
	return map[string][]roleEvidence{"implementation": b.Implementation, "callers": b.Callers, "contracts": b.Contracts, "validation": b.Validation}
}

func (h repositoryHandoff) roles() map[string][]roleEvidence {
	return map[string][]roleEvidence{"producer": h.Producer, "consumer": h.Consumer, "wiring": h.Wiring}
}

func (w *workspace) validateGap(name string, gap roleGap, roles map[string][]roleEvidence, seen map[string]bool) error {
	if _, ok := roles[gap.Role]; !ok || seen[gap.Role] || strings.TrimSpace(gap.Reason) == "" || !slices.Contains([]string{"research_gap", "not_applicable"}, gap.Kind) {
		return fmt.Errorf("%s: each gap needs one valid unique role, kind and precise reason", name)
	}
	seen[gap.Role] = true
	if gap.Kind == "not_applicable" {
		if slices.Contains([]string{"implementation", "producer", "consumer", "wiring"}, gap.Role) {
			return fmt.Errorf("%s: %s needs evidence or a research_gap", name, gap.Role)
		}
		if len(roles[gap.Role]) > 0 {
			return fmt.Errorf("%s: %s cannot be both evidenced and not_applicable", name, gap.Role)
		}
	}
	if len(gap.Searches) == 0 {
		return fmt.Errorf("%s: %s gap needs actual search scopes", name, gap.Role)
	}
	for _, search := range gap.Searches {
		if !slices.Contains(w.searches, search) {
			return fmt.Errorf("%s: gap search was not executed: %s", name, search.Pattern)
		}
	}
	return nil
}

func (w *workspace) validateRoles(name, repository string, roles map[string][]roleEvidence, gaps []roleGap, allowOtherRepo bool) error {
	seen := map[string]bool{}
	for _, gap := range gaps {
		if err := w.validateGap(name, gap, roles, seen); err != nil {
			return err
		}
	}
	for role, evidence := range roles {
		for _, ref := range evidence {
			doc, ok := w.documents[ref.Path]
			if !ok || (!allowOtherRepo && doc.SnapshotID != repository) || ref.StartLine < 1 || ref.EndLine < ref.StartLine || ref.EndLine > doc.LineCount || strings.TrimSpace(ref.Finding) == "" {
				return fmt.Errorf("%s: invalid %s source range/finding: %s", name, role, ref.Path)
			}
		}
	}
	return nil
}

func (w *workspace) validateBoundaries(items []researchBoundary) (map[string]researchBoundary, error) {
	result := map[string]researchBoundary{}
	for _, b := range items {
		if strings.TrimSpace(b.Name) == "" || result[b.Name].Name != "" {
			return nil, errors.New("each responsibility needs a unique name")
		}
		if !slices.ContainsFunc(w.inventory.Repositories, func(r InventoryRepository) bool { return r.ID == b.Repository }) {
			return nil, fmt.Errorf("unknown repository %s", b.Repository)
		}
		if err := w.validateRoles(b.Name, b.Repository, b.roles(), b.Gaps, false); err != nil {
			return nil, err
		}
		result[b.Name] = b
	}
	return result, nil
}

func (w *workspace) validateHandoffs(items []repositoryHandoff, boundaries map[string]researchBoundary) (map[string]repositoryHandoff, error) {
	result := map[string]repositoryHandoff{}
	for _, h := range items {
		from, fromOK := boundaries[h.FromBoundary]
		to, toOK := boundaries[h.ToBoundary]
		if h.ID == "" || result[h.ID].ID != "" || !fromOK || !toOK || from.Repository == to.Repository || strings.TrimSpace(h.Contract) == "" {
			return nil, errors.New("handoff needs unique ID, specific contract and boundaries in two different repositories")
		}
		if err := w.validateRoles(h.ID, "", h.roles(), h.Gaps, true); err != nil {
			return nil, err
		}
		for role, refs := range h.roles() {
			if len(refs) == 0 && !slices.ContainsFunc(h.Gaps, func(g roleGap) bool { return g.Role == role }) {
				return nil, fmt.Errorf("handoff %s: missing %s evidence needs an explicit research_gap", h.ID, role)
			}
		}
		for role, repo := range map[string]string{"producer": from.Repository, "consumer": to.Repository} {
			for _, ref := range h.roles()[role] {
				if w.documents[ref.Path].SnapshotID != repo {
					return nil, fmt.Errorf("handoff %s: %s evidence is from the wrong repository", h.ID, role)
				}
			}
		}
		for _, ref := range h.Wiring {
			repo := w.documents[ref.Path].SnapshotID
			if repo != from.Repository && repo != to.Repository {
				return nil, fmt.Errorf("handoff %s: wiring belongs to neither endpoint", h.ID)
			}
		}
		result[h.ID] = h
	}
	return result, nil
}

func validateReference(c Candidate, d candidateDisposition, boundaries map[string]researchBoundary, handoffs map[string]repositoryHandoff) error {
	h, ok := handoffs[d.HandoffID]
	if !ok {
		return fmt.Errorf("candidate %s: cross-repository reference needs a paired handoff or exclusion/defer reason", c.Path)
	}
	from, to := boundaries[h.FromBoundary].Repository, boundaries[h.ToBoundary].Repository
	forward := from == c.Repository && to == c.TargetRepository
	reverse := to == c.Repository && from == c.TargetRepository
	if !forward && !reverse {
		return fmt.Errorf("candidate %s: handoff endpoints do not match reference", c.Path)
	}
	return nil
}

func validateDisposition(c Candidate, d candidateDisposition, boundaries map[string]researchBoundary, handoffs map[string]repositoryHandoff) error {
	if strings.TrimSpace(d.Reason) == "" || !slices.Contains([]string{"included", "excluded", "deferred"}, d.Disposition) || !slices.Contains([]string{"product_code", "contract", "infrastructure", "tests", "docs", "generated", "vendor", "fixtures", "unknown"}, d.Class) {
		return errors.New("each disposition needs a class, disposition and precise reason")
	}
	if d.Disposition != "included" {
		return nil
	}
	if len(d.Boundaries) == 0 {
		return fmt.Errorf("candidate %s: included candidates need mapped responsibilities", c.Path)
	}
	for _, name := range d.Boundaries {
		b, ok := boundaries[name]
		if !ok || b.Repository != c.Repository || !boundaryTouches(b, c) {
			return fmt.Errorf("candidate %s: responsibility %s has no evidence in this candidate", c.Path, name)
		}
	}
	if c.Kind == "repository_reference" {
		return validateReference(c, d, boundaries, handoffs)
	}
	return nil
}

func (w *workspace) validateCandidates(items []candidateDisposition, boundaries map[string]researchBoundary, handoffs map[string]repositoryHandoff) error {
	candidates := map[string]Candidate{}
	for _, c := range w.inventory.Candidates {
		candidates[c.ID] = c
	}
	seen := map[string]bool{}
	for _, d := range items {
		c, ok := candidates[d.CandidateID]
		if !ok || seen[d.CandidateID] {
			return errors.New("each disposition needs a known unique candidate")
		}
		seen[d.CandidateID] = true
		if err := validateDisposition(c, d, boundaries, handoffs); err != nil {
			return err
		}
	}
	return nil
}

func (w *workspace) validateReclassifications(input planInput, boundaries map[string]researchBoundary) error {
	for _, previous := range w.plan {
		current, exists := boundaries[previous.Name]
		dropped := !exists || current.Repository != previous.Repository
		for role, refs := range previous.roles() {
			for _, ref := range refs {
				if !slices.ContainsFunc(current.roles()[role], func(candidate roleEvidence) bool {
					return candidate.Path == ref.Path && candidate.StartLine <= ref.StartLine && candidate.EndLine >= ref.EndLine && candidate.Finding == ref.Finding
				}) {
					dropped = true
				}
			}
		}
		explained := slices.ContainsFunc(input.Reclassifications, func(r reclassification) bool { return r.Boundary == previous.Name && strings.TrimSpace(r.Reason) != "" })
		if dropped && !explained {
			return fmt.Errorf("responsibility %s lost evidence; supply an explicit reclassification reason", previous.Name)
		}
	}
	return nil
}

func clonePlan(input planInput) (planInput, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return planInput{}, fmt.Errorf("encode research plan: %w", err)
	}
	var result planInput
	if decodeErr := json.Unmarshal(encoded, &result); decodeErr != nil {
		return planInput{}, fmt.Errorf("decode research plan copy: %w", decodeErr)
	}
	return result, nil
}

func (w *workspace) setPlan(input planInput) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.setPlanLocked(input)
}

func (w *workspace) setPlanLocked(input planInput) error {
	if err := w.validateTaskPatch(input); err != nil {
		return err
	}
	patch := input
	if duplicateKey(input.Boundaries, func(b researchBoundary) string { return b.Name }) || duplicateKey(input.Candidates, func(d candidateDisposition) string { return d.CandidateID }) || duplicateKey(input.Handoffs, func(h repositoryHandoff) string { return h.ID }) {
		return errors.New("a plan patch cannot repeat a responsibility, candidate or handoff ID")
	}
	if !input.Replace {
		input = mergePlan(w.ledger.Plan, input)
	}
	boundaries, err := w.validateBoundaries(input.Boundaries)
	if err != nil {
		return err
	}
	handoffs, err := w.validateHandoffs(input.Handoffs, boundaries)
	if err != nil {
		return err
	}
	if validationErr := w.validateCandidates(input.Candidates, boundaries, handoffs); validationErr != nil {
		return validationErr
	}
	if validationErr := w.validateReclassifications(input, boundaries); validationErr != nil {
		return validationErr
	}
	copied, err := clonePlan(input)
	if err != nil {
		return err
	}
	copiedPatch, err := clonePlan(patch)
	if err != nil {
		return err
	}
	w.plan = copied.Boundaries
	w.ledger.Plan = copied
	w.ledger.History = append(w.ledger.History, copiedPatch)
	w.ledger.Status = "in_progress"
	w.ledger.StopReason = ""
	w.checkedNotes = ""
	return nil
}

func duplicateKey[T any](items []T, key func(T) string) bool {
	seen := map[string]bool{}
	for _, item := range items {
		k := key(item)
		if seen[k] {
			return true
		}
		seen[k] = true
	}
	return false
}

func (w *workspace) unfinishedFrontier() []string {
	frontier := slices.Clone(w.ledger.Frontier)
	dispositions := map[string]candidateDisposition{}
	for _, d := range w.ledger.Plan.Candidates {
		dispositions[d.CandidateID] = d
	}
	for _, c := range w.inventory.Candidates {
		d, ok := dispositions[c.ID]
		if !isResearchCandidate(c, d) {
			continue
		}
		if !ok {
			frontier = append(frontier, "unclassified "+c.Kind+": "+c.Path)
		} else if d.Disposition == "deferred" {
			frontier = append(frontier, "deferred "+c.Kind+": "+c.Path+": "+d.Reason)
		}
	}
	check := func(name string, roles map[string][]roleEvidence, gaps []roleGap) {
		for role, refs := range roles {
			if len(refs) == 0 && !slices.ContainsFunc(gaps, func(g roleGap) bool { return g.Role == role }) {
				frontier = append(frontier, name+"/"+role+": not researched yet")
			}
			for _, ref := range refs {
				if !RangeCovered(w.reads[w.documents[ref.Path].ID], ref.StartLine, ref.EndLine) {
					frontier = append(frontier, fmt.Sprintf("%s/%s: unread %s:L%d-L%d", name, role, ref.Path, ref.StartLine, ref.EndLine))
				}
			}
		}
		for _, gap := range gaps {
			if gap.Kind == "research_gap" {
				frontier = append(frontier, name+"/"+gap.Role+": "+gap.Reason)
			}
		}
	}
	for _, b := range w.plan {
		check(b.Name, b.roles(), b.Gaps)
	}
	for _, h := range w.ledger.Plan.Handoffs {
		check(h.ID, h.roles(), h.Gaps)
	}
	slices.Sort(frontier)
	return slices.Compact(frontier)
}

// Incremental patches keep model requests and audit history proportional to the
// discoveries in this turn, rather than retransmitting a growing full plan.
func mergePlan(current, patch planInput) planInput {
	result := planInput{Boundaries: slices.Clone(current.Boundaries), Candidates: slices.Clone(current.Candidates), Handoffs: slices.Clone(current.Handoffs), Reclassifications: patch.Reclassifications}
	for _, b := range patch.Boundaries {
		i := slices.IndexFunc(result.Boundaries, func(old researchBoundary) bool { return old.Name == b.Name })
		if i < 0 {
			result.Boundaries = append(result.Boundaries, b)
		} else {
			result.Boundaries[i] = b
		}
	}
	for _, d := range patch.Candidates {
		i := slices.IndexFunc(result.Candidates, func(old candidateDisposition) bool { return old.CandidateID == d.CandidateID })
		if i < 0 {
			result.Candidates = append(result.Candidates, d)
		} else {
			result.Candidates[i] = d
		}
	}
	for _, h := range patch.Handoffs {
		i := slices.IndexFunc(result.Handoffs, func(old repositoryHandoff) bool { return old.ID == h.ID })
		if i < 0 {
			result.Handoffs = append(result.Handoffs, h)
		} else {
			result.Handoffs[i] = h
		}
	}
	return result
}

type classifyAreaInput struct {
	PathPrefix  string   `json:"path_prefix"`
	Kinds       []string `json:"kinds"`
	Class       string   `json:"class"`
	Disposition string   `json:"disposition"`
	Reason      string   `json:"reason"`
	Boundaries  []string `json:"boundaries"`
}

func (w *workspace) classifyArea(input classifyAreaInput) (int, error) {
	w.mu.Lock()
	var patch planInput
	if input.PathPrefix == "" {
		w.mu.Unlock()
		return 0, errors.New("use an exact repository/path prefix")
	}
	prefix := path.Clean(input.PathPrefix)
	for _, c := range w.inventory.Candidates {
		if w.task != nil && !w.task.ownsCandidate(c.ID) {
			continue
		}
		if (path.Clean(c.Path) != prefix && !strings.HasPrefix(c.Path, prefix+"/")) || (len(input.Kinds) > 0 && !slices.Contains(input.Kinds, c.Kind)) {
			continue
		}
		d := candidateDisposition{CandidateID: c.ID, Class: input.Class, Disposition: input.Disposition, Reason: input.Reason}
		for _, name := range input.Boundaries {
			for _, b := range w.plan {
				if b.Name == name && b.Repository == c.Repository && boundaryTouches(b, c) {
					d.Boundaries = append(d.Boundaries, name)
				}
			}
		}
		patch.Candidates = append(patch.Candidates, d)
	}
	w.mu.Unlock()
	if len(patch.Candidates) == 0 {
		return 0, errors.New("no candidates matched this path prefix and kinds")
	}
	return len(patch.Candidates), w.setPlan(patch)
}

func boundaryTouches(b researchBoundary, c Candidate) bool {
	for _, refs := range b.roles() {
		for _, ref := range refs {
			if (c.Kind == "source_area" || c.Kind == "repository") && (path.Clean(ref.Path) == path.Clean(c.Path) || strings.HasPrefix(ref.Path, path.Clean(c.Path)+"/")) {
				return true
			}
			if ref.Path == c.Path && (c.Line == 0 || (ref.StartLine <= c.Line && ref.EndLine >= c.Line)) {
				return true
			}
		}
	}
	return false
}

// RangeCovered accepts adjacent reads but never skips unread lines.
func RangeCovered(ranges []source.LineRange, start, end int) bool {
	if start < 1 || end < start {
		return false
	}
	ordered := slices.Clone(ranges)
	slices.SortFunc(ordered, func(a, b source.LineRange) int {
		if a.StartLine < b.StartLine {
			return -1
		}
		if a.StartLine > b.StartLine {
			return 1
		}
		return 0
	})
	next := start
	for _, span := range ordered {
		if span.EndLine < next {
			continue
		}
		if span.StartLine > next {
			return false
		}
		next = span.EndLine + 1
		if next > end {
			return true
		}
	}
	return false
}

func (w *workspace) checkCoverage(input coverageInput) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.summaryNotes = input.Notes
	input.Notes = w.findingsText(w.summaryNotes)
	w.submittedNotes = input.Notes
	w.checkedNotes = ""
	if !slices.Contains([]string{researchComplete, researchPartial}, input.Status) {
		return errors.New("coverage status must be complete or partial")
	}
	if len(w.ledger.History) == 0 {
		return errors.New("review inventory and call set_research_plan first")
	}
	citations := map[string][]source.LineRange{}
	for _, m := range citationPattern.FindAllStringSubmatch(input.Notes, -1) {
		start, _ := strconv.Atoi(m[2])
		end, _ := strconv.Atoi(m[3])
		citations[m[1]] = append(citations[m[1]], source.LineRange{StartLine: start, EndLine: end})
	}
	var missing, frontier []string

	for _, b := range w.plan {
		m, f := w.roleCoverage(b.Name, b.roles(), b.Gaps, citations)
		missing = append(missing, m...)
		frontier = append(frontier, f...)
	}
	for _, h := range w.ledger.Plan.Handoffs {
		m, f := w.roleCoverage(h.ID, h.roles(), h.Gaps, citations)
		missing = append(missing, m...)
		frontier = append(frontier, f...)
	}
	dispositions := map[string]candidateDisposition{}
	for _, d := range w.ledger.Plan.Candidates {
		dispositions[d.CandidateID] = d
	}
	for _, c := range w.inventory.Candidates {
		d, ok := dispositions[c.ID]
		if !isResearchCandidate(c, d) {
			continue
		}
		if !ok {
			frontier = append(frontier, "unclassified "+c.Kind+": "+c.Path)
		} else if d.Disposition == "deferred" {
			frontier = append(frontier, "deferred "+c.Kind+": "+c.Path+": "+d.Reason)
		}
	}
	for _, snap := range w.unresearchedRepositories() {
		frontier = append(frontier, "repository has no researched responsibility: "+snap.ID)
	}
	slices.Sort(missing)
	slices.Sort(frontier)
	w.ledger.Frontier = append(slices.Clone(frontier), missing...)
	if len(missing) > 0 {
		return fmt.Errorf("research evidence is incomplete:\n%s", strings.Join(missing, "\n"))
	}
	if input.Status == researchComplete && len(frontier) > 0 {
		return fmt.Errorf("research is partial; close obligations or submit status partial:\n%s", strings.Join(frontier, "\n"))
	}
	if input.Status == researchPartial && len(frontier) == 0 {
		return errors.New("partial research needs a precise open role gap or deferred/unclassified scope; otherwise use complete")
	}
	if input.Status == researchPartial {
		allowed, reason := w.partialStopReason(input.StopReason)
		if !allowed {
			return fmt.Errorf("continue research: %s (%d/%d turns used); use next_research_questions and retain findings before continuing", reason, w.turn, w.turnBudget)
		}
		w.ledger.StopReason = reason
	}
	if _, err := w.evidenceRanges(input.Notes); err != nil {
		return err
	}
	w.checkedNotes = input.Notes
	w.ledger.Status = input.Status
	return nil
}

func (w *workspace) findingsText(summary string) string {
	var b strings.Builder
	for _, finding := range w.ledger.Findings {
		fmt.Fprintf(&b, "\n## %s\n%s\n", finding.Name, finding.Notes)
	}
	b.WriteString(summary)
	return b.String()
}

func (w *workspace) saveFinding(input researchFinding) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("a finding needs a responsibility or route name")
	}
	if _, err := w.evidenceRanges(input.Notes); err != nil {
		return err
	}
	if slices.Contains(w.ledger.Findings, input) {
		return nil
	}
	w.ledger.Findings = append(w.ledger.Findings, input)
	w.submittedNotes = w.findingsText(w.summaryNotes)
	w.checkedNotes = ""
	return nil
}

func (w *workspace) ledgerContext() (string, error) {
	// Keep exact role/finding links and explicit uncertainty for analysis. The
	// complete inventory/history is persisted separately, avoiding duplicate trees.
	data, err := json.Marshal(struct {
		Status     string              `json:"status"`
		Frontier   any                 `json:"research_gaps"`
		Boundaries []researchBoundary  `json:"responsibilities"`
		Handoffs   []repositoryHandoff `json:"cross_repository_handoffs"`
		StopReason string              `json:"stop_reason,omitempty"`
	}{w.ledger.Status, w.frontierSummary(), w.plan, w.ledger.Plan.Handoffs, w.ledger.StopReason})
	if err != nil {
		return "", fmt.Errorf("encode research ledger context: %w", err)
	}
	return "\nRESEARCH EVIDENCE LEDGER (range checks are mechanical, not semantic proof; research gaps are not product limitations):\n" + string(data), nil
}

func (w *workspace) coverageTools() []fantasy.AgentTool {
	return []fantasy.AgentTool{
		w.answerTool(),
		w.inventoryTool(),
		w.questionTool(),
		w.focusTool(),
		fantasy.NewAgentTool("classify_research_area", "Classify candidates in one repository/path prefix with a shared class, disposition and observable reason, optionally filtered by kinds. Use to account for generated/vendor/docs/test areas without listing every ID. For included candidates supply known boundary names; only boundaries with evidence inside each candidate are mapped. Cross-repository references require individual paired handoffs via set_research_plan. Atomic incremental update; earlier classifications remain unless matched.", func(_ context.Context, input classifyAreaInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if err := w.tick("Classifying source area: " + input.PathPrefix); err != nil {
				return fantasy.ToolResponse{}, err
			}
			count, err := w.classifyArea(input)
			if err != nil {
				return toolJSON(map[string]string{toolErrorKey: err.Error()})
			}
			return toolJSON(map[string]any{"classified_candidates": count})
		}),
		fantasy.NewAgentTool("retain_research_findings", "Save source-cited findings incrementally for each responsibility, subsystem or variant. Include detailed relevant behavior and exact original citations. Saved notes are retained verbatim and combined with final coverage notes; no need to compress the whole system into one response. Append corrections explicitly; do not hide contradictions.", func(_ context.Context, input researchFinding, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if err := w.tick("Retaining researched findings: " + input.Name); err != nil {
				return fantasy.ToolResponse{}, err
			}
			if err := w.saveFinding(input); err != nil {
				return toolJSON(map[string]string{toolErrorKey: err.Error()})
			}
			return toolJSON(map[string]any{"retained": true, "finding_groups": len(w.ledger.Findings)})
		}),
		fantasy.NewAgentTool("set_research_plan", "Incrementally upsert responsibilities by name, candidates by candidate_id, and handoffs by ID. Send only additions/changes; no massive plan retransmission. replace=true explicitly replaces the full plan. History retains patches. Classes: product_code, contract, infrastructure, tests, docs, generated, vendor, fixtures, unknown. Responsibility roles may be saved incrementally; omitted roles remain pending. Prefer save_research_answer for a verified role. Explicit gaps need a role-specific reason and actual prior search_code scopes. Missing research is research_gap; absent role can be not_applicable after search. Pair cross-repository references with producer, consumer and wiring evidence. Unknown candidates stay visible.", func(_ context.Context, input planInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if err := w.tick(fmt.Sprintf("Planning research coverage (%d responsibilities).", len(input.Boundaries))); err != nil {
				return fantasy.ToolResponse{}, err
			}
			if err := w.setPlan(input); err != nil {
				return toolJSON(map[string]string{toolErrorKey: err.Error()})
			}
			return toolJSON(map[string]any{"planned_responsibilities": len(w.plan), "classified_candidates": len(w.ledger.Plan.Candidates), "handoffs": len(w.ledger.Plan.Handoffs)})
		}),
		fantasy.NewAgentTool("check_research_coverage", "Submit all retained source-cited findings with status complete or partial. Checks exact read AND cited ranges for every role and both cross-repository endpoints, plus inventory dispositions and role gaps. A partial result retains its unexplored frontier; research_gap is not a product limitation. After acceptance return RESEARCH_COMPLETE or RESEARCH_PARTIAL matching accepted status. Mechanical checks do not establish semantic entailment or completeness of heuristic discovery.", func(_ context.Context, input coverageInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if err := w.tick("Checking researched responsibilities and exact source evidence."); err != nil {
				return fantasy.ToolResponse{}, err
			}
			if err := w.checkCoverage(input); err != nil {
				return toolJSON(map[string]string{toolErrorKey: err.Error()})
			}
			return toolJSON(map[string]any{"accepted": true, "status": w.ledger.Status, "frontier": w.ledger.Frontier})
		}),
	}
}

func (w *workspace) roleCoverage(name string, roles map[string][]roleEvidence, gaps []roleGap, citations map[string][]source.LineRange) (missing, frontier []string) {
	for role, refs := range roles {
		if len(refs) == 0 && !slices.ContainsFunc(gaps, func(g roleGap) bool { return g.Role == role }) {
			frontier = append(frontier, name+"/"+role+": not researched yet")
		}
		for _, ref := range refs {
			doc := w.documents[ref.Path]
			if !RangeCovered(w.reads[doc.ID], ref.StartLine, ref.EndLine) {
				missing = append(missing, fmt.Sprintf("%s/%s: unread %s:L%d-L%d", name, role, ref.Path, ref.StartLine, ref.EndLine))
			}
			if !RangeCovered(citations[doc.ID], ref.StartLine, ref.EndLine) {
				missing = append(missing, fmt.Sprintf("%s/%s: notes lack exact evidence for %s:L%d-L%d", name, role, ref.Path, ref.StartLine, ref.EndLine))
			}
		}
	}
	for _, gap := range gaps {
		if gap.Kind == "research_gap" {
			frontier = append(frontier, name+"/"+gap.Role+": "+gap.Reason)
		}
	}
	return missing, frontier
}
