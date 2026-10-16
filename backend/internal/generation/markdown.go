package generation

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

// MarkdownExport is the published reading document that a client can save as a file.
type MarkdownExport struct {
	ProductID  string `json:"product_id"`
	RevisionID string `json:"revision_id"`
	Version    int    `json:"version"`
	Filename   string `json:"filename"`
	Markdown   string `json:"markdown"`
}

// ExportMarkdown always reads the published copy, even for a reviewer.
func (s *Service) ExportMarkdown(ctx context.Context, productID string) (MarkdownExport, error) {
	r, err := s.PublishedRevision(ctx, productID)
	if err != nil {
		return MarkdownExport{}, err
	}
	var name string
	if err := s.DB.QueryRowContext(ctx, "SELECT name FROM products WHERE id=?", productID).Scan(&name); err != nil {
		return MarkdownExport{}, fmt.Errorf("read export product name: %w", err)
	}
	return MarkdownExport{
		ProductID: r.ProductID, RevisionID: r.ID, Version: r.Version,
		Filename: markdownFilename(name, r.Version), Markdown: renderMarkdown(r.Content),
	}, nil
}

func markdownFilename(name string, version int) string {
	var stem strings.Builder
	separator := false
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			separator = true
			continue
		}
		if separator && stem.Len() > 0 {
			stem.WriteByte('-')
		}
		stem.WriteRune(unicode.ToLower(char))
		separator = false
	}
	if stem.Len() == 0 {
		stem.WriteString("product-passport")
	}
	return fmt.Sprintf("%s-v%d.md", stem.String(), version)
}

type markdownWriter struct {
	strings.Builder
	objects   map[string]passport.Object
	scenarios map[string]passport.Scenario
	rendered  map[string]bool
}

func markdownText(text string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "#", "\\#", "|", "\\|", "<", "\\<", ">", "\\>", "\r", " ", "\n", " ").Replace(text)
}

func (w *markdownWriter) heading(level int, text string) {
	fmt.Fprintf(w, "%s %s\n\n", strings.Repeat("#", level), markdownText(text))
}

func (w *markdownWriter) prose(text string) {
	if text != "" {
		w.WriteString(text)
		w.WriteString("\n\n")
	}
}

func (w *markdownWriter) items(label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(w, "**%s:**\n\n", label)
	for _, value := range values {
		fmt.Fprintf(w, "- %s\n", markdownText(value))
	}
	w.WriteString("\n")
}

func renderMarkdown(p passport.Passport) string {
	w := &markdownWriter{
		objects:   make(map[string]passport.Object, len(p.Model.Objects)),
		scenarios: make(map[string]passport.Scenario, len(p.Model.Scenarios)), rendered: make(map[string]bool),
	}
	for _, object := range p.Model.Objects {
		w.objects[object.ID] = object
	}
	for _, scenario := range p.Model.Scenarios {
		w.scenarios[scenario.ID] = scenario
	}
	w.heading(1, p.Model.Title)
	w.prose(p.Model.Tagline)
	if document := p.Views.Document; document != nil && len(document.Chapters) > 0 {
		for _, chapter := range document.Chapters {
			w.chapter(chapter)
		}
	} else {
		w.prose(p.Model.Summary)
		for _, section := range p.Views.Sections {
			w.heading(2, section.Title)
			w.prose(section.Summary)
			w.objectDescriptions(section.ObjectIDs, "")
			for _, id := range section.ScenarioIDs {
				w.scenario(id)
			}
			w.items("Checklist", section.Checklist)
		}
	}
	return w.String()
}

func (w *markdownWriter) chapter(chapter passport.Chapter) {
	w.heading(2, chapter.Title)
	for _, block := range chapter.Blocks {
		switch block.Kind {
		case "markdown":
			w.prose(block.Markdown)
		case scenarioReferenceKind:
			w.scenario(block.ReferenceID)
		case "object_table":
			w.objectDescriptions(block.ObjectIDs, block.TableStyle)
		}
	}
}

func (w *markdownWriter) objectDescriptions(ids []string, style string) {
	for _, id := range ids {
		object, ok := w.objects[id]
		if !ok {
			continue
		}
		name := object.Name
		if style == "technologies" && object.Technology != "" {
			name += " (" + object.Technology + ")"
		}
		fmt.Fprintf(w, "- **%s:** %s\n", markdownText(name), markdownText(object.Summary))
	}
	if len(ids) > 0 {
		w.WriteString("\n")
	}
}

func (w *markdownWriter) scenario(id string) {
	scenario, ok := w.scenarios[id]
	if !ok || w.rendered[id] {
		return
	}
	w.rendered[id] = true
	w.heading(3, scenario.Title)
	w.prose(scenario.Summary)
	w.items("Before you start", scenario.Inputs)
	for i, step := range scenario.Steps {
		fmt.Fprintf(w, "%d. **%s:** %s\n", i+1, markdownText(step.Title), markdownText(step.Summary))
	}
	if len(scenario.Steps) > 0 {
		w.WriteString("\n")
	}
	w.items("Result", scenario.Outputs)
	w.items("Possible problems", scenario.FailureModes)
}
