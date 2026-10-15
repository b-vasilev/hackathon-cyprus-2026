package research

import (
	"slices"
	"strings"
)

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
