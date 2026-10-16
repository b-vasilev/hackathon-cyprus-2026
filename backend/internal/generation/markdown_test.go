package generation

import (
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

func TestMarkdownReadingDocument(t *testing.T) {
	t.Parallel()
	p := passport.Passport{
		Model: passport.Model{
			Title: "Product's guide", Tagline: "A useful product.", Summary: "INTERNAL_MODEL_SUMMARY",
			Objects: []passport.Object{
				{ID: "internal-object", Name: "Application", Summary: "Helps people finish their work.", Technology: "Go", ClaimIDs: []string{"internal-claim"}},
				{ID: "internal-unused", Name: "Unreferenced technical component"},
			},
			Scenarios: []passport.Scenario{{
				ID: "internal-flow", Title: "Complete a task", Summary: "This flow describes the result.",
				Inputs: []string{"An account"}, Outputs: []string{"A finished task"}, FailureModes: []string{"Access denied"},
				Steps: []passport.Step{{ID: "internal-step", Title: "Start work", Summary: "Choose a task.", ObjectIDs: []string{"internal-object"}, ClaimIDs: []string{"internal-claim"}}},
			}},
			Claims:   []passport.Claim{{ID: "internal-claim", Text: "INTERNAL_CLAIM", Basis: "inferred"}},
			Evidence: []passport.Evidence{{ID: "internal-evidence", DocumentID: "internal-document"}}, OpenQuestions: []string{"INTERNAL_QUESTION"},
		},
		Views: passport.Views{
			Document: &passport.Document{Version: 2, Chapters: []passport.Chapter{{
				ID: "internal-chapter", Title: "Product overview", Question: "INTERNAL_PROMPT",
				Blocks: []passport.DocumentBlock{
					{Kind: "markdown", Markdown: "People can use this product.\n\nIt keeps their work together.", ClaimIDs: []string{"internal-claim"}},
					{Kind: "object_table", ObjectIDs: []string{"internal-object"}, TableStyle: "technologies"},
					{Kind: "diagram_ref", ReferenceID: "internal-diagram"},
					{Kind: "scenario_ref", ReferenceID: "internal-flow"},
					{Kind: "scenario_ref", ReferenceID: "internal-flow"},
				},
			}}},
			Diagrams:  []passport.Diagram{{ID: "internal-diagram", Title: "Converter components and local contracts"}},
			CardTheme: &passport.ProductCardTheme{AccentColor: "INTERNAL_COLOR"},
		},
	}
	md := renderMarkdown(p)
	for _, want := range []string{"# Product's guide", "A useful product.", "## Product overview", "People can use this product.\n\nIt keeps their work together.", "**Application (Go):** Helps people finish their work.", "1. **Start work:** Choose a task.", "An account", "A finished task", "Access denied"} {
		if !strings.Contains(md, want) {
			t.Errorf("reading export lost %q", want)
		}
	}
	for _, unwanted := range []string{"INTERNAL_", "internal-", "Converter components and local contracts", "Unreferenced technical component", "## Evidence", "## Open questions", "## Product branding", "## Claims", "## Diagrams", "&\\#39;"} {
		if strings.Contains(md, unwanted) {
			t.Errorf("reading export contains %q", unwanted)
		}
	}
	if strings.Count(md, "### Complete a task") != 1 {
		t.Fatal("a referenced flow should appear once")
	}
}

func TestMarkdownLegacySections(t *testing.T) {
	t.Parallel()
	p := passport.Passport{
		Model: passport.Model{Title: "Older product", Summary: "A short introduction.", Objects: []passport.Object{{ID: "internal-object", Name: "App", Summary: "Does useful work."}}},
		Views: passport.Views{Sections: []passport.Section{{
			ID: "internal-section", Title: "Overview", Summary: "The product's purpose.", Status: "available", ObjectIDs: []string{"internal-object"}, ClaimIDs: []string{"internal-claim"}, Checklist: []string{"Try the product"},
		}}},
	}
	md := renderMarkdown(p)
	for _, want := range []string{"A short introduction.", "## Overview", "The product's purpose.", "**App:** Does useful work.", "Try the product"} {
		if !strings.Contains(md, want) {
			t.Errorf("legacy reading export lost %q", want)
		}
	}
	if strings.Contains(md, "internal-") || strings.Contains(md, "**Status:**") {
		t.Fatal("legacy export contains internal metadata")
	}
}

func TestMarkdownFilenames(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		want string
	}{
		{"antares", "antares-v2.md"},
		{"CLI-Anything", "cli-anything-v2.md"},
		{"Product name / Windows: tools", "product-name-windows-tools-v2.md"},
		{"../../", "product-passport-v2.md"},
		{"  ", "product-passport-v2.md"},
		{"Café", "café-v2.md"},
	} {
		if got := markdownFilename(test.name, 2); got != test.want {
			t.Errorf("filename for %q = %q, want %q", test.name, got, test.want)
		}
	}
}
