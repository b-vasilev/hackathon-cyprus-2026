package research

import (
	"path"
	"slices"
	"strings"
)

// File patterns are navigation hints, not a checklist for an architecture
// passport. Read support areas only when a concrete product flow needs them.
func isArchitectureScope(c Candidate) bool {
	if c.Kind == "repository" {
		return true
	}
	if c.Kind != "source_area" {
		return false
	}
	file := strings.TrimPrefix(path.Clean(c.Path), c.Repository+"/")
	for _, part := range strings.Split(strings.ToLower(file), "/") {
		if slices.Contains([]string{".github", ".agents", ".claude", ".codex", "docs", "doc", "tests", "test", "tools", "scripts", "examples", "fixtures", "benchmarks", "vendor", "generated"}, part) {
			return false
		}
	}
	return true
}

func isResearchCandidate(c Candidate, d candidateDisposition) bool {
	// An explicit classification can promote a hint when source investigation
	// establishes that it matters to a product scenario or external contract.
	return isArchitectureScope(c) || d.Disposition != ""
}

// These conventional instruction locations are available as evidence, but do
// not need independent research unless product code actually uses them.
func isDevelopmentGuidance(file string) bool {
	parts := strings.Split(strings.ToLower(file), "/")
	for _, part := range parts {
		if slices.Contains([]string{".agents", ".claude", ".codex", "agents.md", "claude.md"}, part) {
			return true
		}
	}
	return false
}

func (w *workspace) excludeDevelopmentGuidance() error {
	classified := map[string]bool{}
	for _, d := range w.ledger.Plan.Candidates {
		classified[d.CandidateID] = true
	}
	var patch planInput
	for _, c := range w.inventory.Candidates {
		if !classified[c.ID] && isDevelopmentGuidance(strings.TrimPrefix(c.Path, c.Repository+"/")) {
			patch.Candidates = append(patch.Candidates, candidateDisposition{
				CandidateID: c.ID, Class: "docs", Disposition: "excluded",
				Reason: "Conventional developer-agent instruction location; not an independent product responsibility. Sources remain available if runtime code references them.",
			})
		}
	}
	if len(patch.Candidates) == 0 {
		return nil
	}
	return w.setPlan(patch)
}

func candidatePriority(c Candidate) int {
	file := strings.TrimPrefix(c.Path, c.Repository+"/")
	if isDevelopmentGuidance(file) {
		return 100
	}
	if file == ".github" || strings.HasPrefix(file, ".github/") {
		return 50
	}
	switch c.Kind {
	case "repository":
		return 1
	case "entrypoint", "manifest":
		return 2
	case "registry", "repository_reference":
		return 3
	case "source_area":
		return 4
	default:
		return 5
	}
}
