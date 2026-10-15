package generation

import (
	"embed"
	"fmt"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

// PromptVersion is persisted with each model call for reproducibility.
const PromptVersion = "passport-v13-balanced-architecture"

//go:embed prompts/system.md
var systemPrompt string

//go:embed prompts/analyze.md
var analysisPrompt string

//go:embed prompts/incremental-analysis.md
var incrementalAnalysisPrompt string

//go:embed prompts/research-architecture.md
var architectureResearchPrompt string

//go:embed prompts/reconcile.md
var reconcilePrompt string

//go:embed prompts/compose.md
var compositionPrompt string

//go:embed prompts/compose-plan.md
var chapterPlanPrompt string

//go:embed prompts/compose-part.md
var fragmentPrompt string

//go:embed prompts/select-files.md
var selectionPrompt string

//go:embed prompts/repair-schema.md
var schemaRepairPrompt string

//go:embed prompts/information-architecture.md
var informationArchitecturePrompt string

//go:embed prompts/sections/*.md
var sectionPrompts embed.FS

// sectionRequirements loads v2 reading order, followed by legacy optional recipes.
// Legacy section order remains a separate compatibility contract.
func sectionRequirements() (string, error) {
	var result strings.Builder
	result.WriteString(informationArchitecturePrompt + "\n")
	for _, id := range passport.ChapterIDs() {
		content, err := sectionPrompts.ReadFile("prompts/sections/" + id + ".md")
		if err != nil {
			return "", fmt.Errorf("read %s section prompt: %w", id, err)
		}
		result.Write(content)
		result.WriteString("\n")
	}
	return result.String(), nil
}
