package passport

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

//nolint:gocyclo // Check the hierarchy and its view scopes against one object index.
func validateArchitecture(objects []Object, diagrams []Diagram, v2 bool) error {
	byID := make(map[string]Object, len(objects))
	for _, o := range objects {
		byID[o.ID] = o
	}
	for _, d := range diagrams {
		if d.ScopeObjectID != "" {
			scope, ok := byID[d.ScopeObjectID]
			if !ok || scope.Kind == "repository" || d.Kind == repositoriesCategory {
				return fmt.Errorf("invalid diagram scope: %s", d.ID)
			}
			if v2 && d.Kind == "containers" && scope.ArchitectureRole != "" && scope.ArchitectureRole != "software_system" {
				return fmt.Errorf("diagram scope has an incompatible architecture role: %s", d.ID)
			}
		}
		if !v2 || !slices.Contains([]string{"context", "containers", "components"}, d.Kind) {
			continue
		}
		expectedRole := map[string]string{"context": "software_system", "containers": "runtime_unit", "components": "component"}[d.Kind]
		for _, id := range d.NodeIDs {
			o := byID[id]
			// Supporting nodes outside scope may show a coarser boundary. Inside
			// the scope, only immediate children belong on this view.
			if d.ScopeObjectID != "" {
				for parent := o.ParentID; parent != "" && parent != d.ScopeObjectID; parent = byID[parent].ParentID {
					if byID[parent].ParentID == d.ScopeObjectID {
						return fmt.Errorf("diagram skips an architectural level: %s", d.ID)
					}
				}
				if o.ParentID != d.ScopeObjectID {
					continue
				}
			}
			if o.ArchitectureRole != "" && o.ArchitectureRole != expectedRole {
				return fmt.Errorf("diagram mixes architectural levels: %s", d.ID)
			}
		}
	}
	return nil
}

// validateArchitectureHierarchy permits unresolved parents only while building.
func validateArchitectureHierarchy(objects []Object, draft bool) error {
	byID := make(map[string]Object, len(objects))
	for _, object := range objects {
		byID[object.ID] = object
	}
	for _, o := range objects {
		if o.ArchitectureRole != "" {
			if !slices.Contains([]string{"software_system", "runtime_unit", "component"}, o.ArchitectureRole) || !slices.Contains([]string{"component", "store", "external_system", "interface"}, o.Kind) {
				return fmt.Errorf("invalid architecture role: %s", o.ID)
			}
		}
		if o.ParentID == "" {
			continue
		}
		parent, ok := byID[o.ParentID]
		if o.ParentID == o.ID || o.ArchitectureRole == "" || o.ArchitectureRole == "software_system" {
			return fmt.Errorf("invalid architectural parent: %s", o.ID)
		}
		if !ok && draft {
			continue
		}
		if !ok || parent.ArchitectureRole == "" || parent.Kind == "repository" {
			return fmt.Errorf("invalid architectural parent: %s", o.ID)
		}
		if o.ArchitectureRole == "runtime_unit" && parent.ArchitectureRole != "software_system" {
			return fmt.Errorf("runtime unit parent must be a software system: %s", o.ID)
		}
		seen := map[string]bool{o.ID: true}
		for id := o.ParentID; id != ""; id = byID[id].ParentID {
			if seen[id] {
				return fmt.Errorf("architectural parent cycle: %s", o.ID)
			}
			seen[id] = true
		}
	}
	return nil
}

//nolint:gocyclo // Keep each typed block's payload, references and placement checks together.
func validateDocument(document Document, claims, objects, diagrams, scenarios map[string]bool, modelObjects []Object, views []Diagram) error {
	if document.Version != 2 {
		return fmt.Errorf("unsupported document version: %d", document.Version)
	}
	repositories := make(map[string]bool)
	for _, o := range modelObjects {
		repositories[o.ID] = o.Kind == "repository"
	}
	repositoryViews := make(map[string]bool)
	for _, d := range views {
		repositoryViews[d.ID] = d.Kind == repositoriesCategory
	}
	chapters := make(map[string]bool)
	placedScenarios := make(map[string]bool)
	for _, chapter := range document.Chapters {
		if !slices.Contains(ChapterIDs(), chapter.ID) || chapters[chapter.ID] || strings.TrimSpace(chapter.Title) == "" || strings.TrimSpace(chapter.Question) == "" || len(chapter.Blocks) == 0 {
			return fmt.Errorf("invalid document chapter: %s", chapter.ID)
		}
		chapters[chapter.ID] = true
		for _, block := range chapter.Blocks {
			if err := checkRefs(block.ClaimIDs, claims); err != nil {
				return err
			}
			switch block.Kind {
			case "markdown":
				if strings.TrimSpace(block.Markdown) == "" || block.ReferenceID != "" || len(block.ObjectIDs) != 0 || block.TableStyle != "" {
					return fmt.Errorf("invalid markdown payload in chapter: %s", chapter.ID)
				}
				if err := validateMarkdown(block.Markdown); err != nil {
					return fmt.Errorf("chapter %s: %w", chapter.ID, err)
				}
			case "diagram_ref", "scenario_ref":
				if block.Markdown != "" || len(block.ObjectIDs) != 0 || len(block.ClaimIDs) != 0 || block.TableStyle != "" {
					return fmt.Errorf("mixed reference payload in chapter: %s", chapter.ID)
				}
				if block.Kind == "diagram_ref" {
					if !diagrams[block.ReferenceID] {
						return fmt.Errorf("unknown diagram reference: %s", block.ReferenceID)
					}
					if repositoryViews[block.ReferenceID] && chapter.ID != repositoriesCategory {
						return fmt.Errorf("repository diagrams belong in the repositories chapter: %s", chapter.ID)
					}
				} else {
					if !scenarios[block.ReferenceID] || chapter.ID != "scenarios" || placedScenarios[block.ReferenceID] {
						return fmt.Errorf("invalid scenario placement or reference: %s", block.ReferenceID)
					}
					placedScenarios[block.ReferenceID] = true
				}
			case "object_table":
				if block.Markdown != "" || block.ReferenceID != "" || len(block.ObjectIDs) == 0 || !slices.Contains([]string{"objects", repositoriesCategory, "technologies"}, block.TableStyle) {
					return fmt.Errorf("invalid object table payload in chapter: %s", chapter.ID)
				}
				if err := checkRefs(block.ObjectIDs, objects); err != nil {
					return err
				}
				for _, id := range block.ObjectIDs {
					if (block.TableStyle == repositoriesCategory && !repositories[id]) || (chapter.ID == "architecture" && repositories[id]) {
						return fmt.Errorf("object table mixes repository provenance and logical structure: %s", chapter.ID)
					}
				}
			default:
				return fmt.Errorf("unknown document block kind: %s", block.Kind)
			}
		}
	}
	for _, id := range RequiredSectionIDs() {
		if !chapters[id] {
			return fmt.Errorf("passport lacks required chapter: %s", id)
		}
	}
	return nil
}

var (
	rawHTML      = regexp.MustCompile(`(?i)<(?:/?[a-z]|!|\?|>)`)
	safeAutolink = regexp.MustCompile(`(?i)<(?:https?://[^<>\s]+|mailto:[^<>\s]+|[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,})>`)
	moduleJS     = regexp.MustCompile(`(?m)^\s*(?:import|export)\s`)
	unsafeScheme = regexp.MustCompile(`(?i)(?:\]\(|\]:|<)<?(?:javascript|vbscript|data|file|filesystem|blob):`)
	codeFence    = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
)

func validateMarkdown(markdown string) error {
	markdown = markdownOutsideCode(markdown)
	if rawHTML.MatchString(safeAutolink.ReplaceAllString(markdown, "")) || moduleJS.MatchString(markdown) {
		return errors.New("document markdown must not contain raw HTML or JS modules")
	}
	// Decode common obfuscation before checking URI schemes. The renderer must
	// also keep raw HTML disabled and apply its own safe URL handling.
	decoded := html.UnescapeString(markdown)
	for range 3 {
		next, err := url.PathUnescape(decoded)
		if err != nil || next == decoded {
			break
		}
		decoded = next
	}
	decoded = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return -1
		}
		return r
	}, decoded)
	if unsafeScheme.MatchString(decoded) {
		return errors.New("document markdown contains an unsafe URL scheme")
	}
	return nil
}

// markdownOutsideCode masks literal code before inspecting rendered content.
// It preserves line breaks so separate blocks cannot combine into module syntax.
// Inline spans are limited to a single line; uncertain syntax is checked rather
// than exempted. Rendering still must disable HTML and filter unsafe URLs.
func markdownOutsideCode(markdown string) string {
	lines := strings.SplitAfter(markdown, "\n")
	var fence string
	indented, previousBlank := false, true
	for i, line := range lines {
		blank := strings.TrimSpace(line) == ""
		marker := codeFence.FindStringSubmatch(strings.TrimSuffix(line, "\n"))
		if fence != "" {
			if len(marker) != 0 && marker[1][0] == fence[0] && len(marker[1]) >= len(fence) && strings.TrimSpace(marker[2]) == "" {
				fence = ""
			}
			lines[i] = maskCode(line)
			previousBlank = true
			continue
		}
		isIndented := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
		if (indented && (blank || isIndented)) || (previousBlank && isIndented) {
			indented = true
			lines[i] = maskCode(line)
			previousBlank = blank
			continue
		}
		indented = false
		if len(marker) != 0 && (marker[1][0] != '`' || !strings.Contains(marker[2], "`")) {
			fence = marker[1]
			lines[i] = maskCode(line)
		} else {
			lines[i] = maskInlineCode(line)
		}
		previousBlank = blank
	}
	return strings.Join(lines, "")
}

func maskCode(code string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return r
		}
		return ' '
	}, code)
}

func maskInlineCode(line string) string {
	var result strings.Builder
	for start := 0; start < len(line); {
		if line[start] != '`' {
			// An escaped angle bracket is literal text, not raw HTML.
			if line[start] == '\\' && start+1 < len(line) && (line[start+1] == '<' || line[start+1] == '`' || line[start+1] == '\\') {
				result.WriteByte(' ')
				result.WriteByte(' ')
				start += 2
			} else {
				result.WriteByte(line[start])
				start++
			}
			continue
		}
		end := start
		for end < len(line) && line[end] == '`' {
			end++
		}
		width := end - start
		matched := false
		for next := end; next < len(line); {
			if line[next] != '`' {
				next++
				continue
			}
			endTick := next
			for endTick < len(line) && line[endTick] == '`' {
				endTick++
			}
			if endTick-next == width {
				result.WriteString(maskCode(line[start:endTick]))
				start, matched = endTick, true
				break
			}
			next = endTick
		}
		if !matched {
			result.WriteString(line[start:end])
			start = end
		}
	}
	return result.String()
}
