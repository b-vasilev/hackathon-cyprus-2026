package generation

import (
	"embed"
	"fmt"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

// PromptVersion is persisted with each model call for reproducibility.
const PromptVersion = "passport-v4"

//go:embed prompts/system.md
var systemPrompt string

//go:embed prompts/analyze.md
var analysisPrompt string

//go:embed prompts/research-architecture.md
var architectureResearchPrompt string

//go:embed prompts/reconcile.md
var reconcilePrompt string

//go:embed prompts/compose.md
var compositionPrompt string

//go:embed prompts/repair.md
var repairPrompt string

//go:embed prompts/select-files.md
var selectionPrompt string

//go:embed prompts/repair-schema.md
var schemaRepairPrompt string

//go:embed prompts/sections/*.md
var sectionPrompts embed.FS

// sectionRequirements loads recipes in the same order as the public passport contract.
func sectionRequirements() (string, error) {
	var result strings.Builder
	for _, id := range passport.SectionIDs() {
		content, err := sectionPrompts.ReadFile("prompts/sections/" + id + ".md")
		if err != nil {
			return "", fmt.Errorf("read %s section prompt: %w", id, err)
		}
		result.Write(content)
		result.WriteString("\n")
	}
	return result.String(), nil
}
