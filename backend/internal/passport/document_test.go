package passport

import (
	"encoding/json"
	"strings"
	"testing"
)

func documentFixture() Passport {
	p := Passport{Model: Model{Title: "Product", Summary: "Purpose", Claims: []Claim{{ID: "claim", Text: "A known fact", Basis: "inferred"}}}, Views: Views{Sections: []Section{}, Document: &Document{Version: 2}}}
	for _, o := range []Object{
		{ID: "repo", Kind: "repository"},
		{ID: "system", Kind: "component", ArchitectureRole: "software_system"},
		{ID: "app", Kind: "component", ArchitectureRole: "runtime_unit", ParentID: "system"},
		{ID: "engine", Kind: "component", ArchitectureRole: "component", ParentID: "app"},
		{ID: "parser", Kind: "component", ArchitectureRole: "component", ParentID: "engine"},
		{ID: "external", Kind: "external_system", ArchitectureRole: "software_system"},
	} {
		o.Name, o.Status, o.ClaimIDs = o.ID, "implemented", []string{"claim"}
		p.Model.Objects = append(p.Model.Objects, o)
	}
	p.Model.Relations = []Relation{{ID: "call", From: "app", To: "external", Kind: "call", Interaction: "sync", Contract: "HTTPS JSON", ClaimIDs: []string{"claim"}}}
	p.Model.Scenarios = []Scenario{{ID: "flow", Steps: []Step{{ID: "step", ObjectIDs: []string{"app"}, ClaimIDs: []string{"claim"}}}}}
	p.Views.Diagrams = []Diagram{
		{ID: "repos", Kind: "repositories", NodeIDs: []string{"repo"}},
		{ID: "containers", Kind: "containers", ScopeObjectID: "system", NodeIDs: []string{"app", "external"}, RelationIDs: []string{"call"}},
		{ID: "components", Kind: "components", ScopeObjectID: "app", NodeIDs: []string{"engine", "external"}},
	}
	for _, id := range ChapterIDs()[:5] {
		p.Views.Document.Chapters = append(p.Views.Document.Chapters, Chapter{ID: id, Title: id, Question: "What is known?", Blocks: []DocumentBlock{{Kind: "markdown", Markdown: "A supported paragraph.", ClaimIDs: []string{"claim"}}}})
	}
	p.Views.Document.Chapters[1].Blocks = append(p.Views.Document.Chapters[1].Blocks, DocumentBlock{Kind: "diagram_ref", ReferenceID: "repos"}, DocumentBlock{Kind: "object_table", ObjectIDs: []string{"repo"}, TableStyle: "repositories", ClaimIDs: []string{"claim"}})
	p.Views.Document.Chapters[2].Blocks = append(p.Views.Document.Chapters[2].Blocks, DocumentBlock{Kind: "diagram_ref", ReferenceID: "containers"}, DocumentBlock{Kind: "markdown", Markdown: "The engine has a parser.", ClaimIDs: []string{"claim"}}, DocumentBlock{Kind: "diagram_ref", ReferenceID: "components"})
	p.Views.Document.Chapters[3].Blocks = append(p.Views.Document.Chapters[3].Blocks, DocumentBlock{Kind: "scenario_ref", ReferenceID: "flow"})
	p.Views.Document.Chapters[4].Blocks = append(p.Views.Document.Chapters[4].Blocks, DocumentBlock{Kind: "object_table", ObjectIDs: []string{"app"}, TableStyle: "technologies"})
	return p
}

func TestDocumentSchemaAndLegacyCompatibility(t *testing.T) {
	for _, v2 := range []bool{false, true} {
		p := documentFixture()
		if !v2 {
			p.Views.Document = nil
			for _, id := range RequiredSectionIDs() {
				p.Views.Sections = append(p.Views.Sections, Section{ID: id, Title: id, Summary: "Unknown", Status: "insufficient_evidence"})
			}
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := CheckJSON[Passport](data)
		if err != nil {
			t.Fatalf("v2=%v schema: %v", v2, err)
		}
		if err := Validate(decoded, nil); err != nil {
			t.Fatalf("v2=%v validation: %v", v2, err)
		}
	}
	if _, err := CheckJSON[DocumentBlock]([]byte(`{"kind":"markdown","markdown":"Text","jsx":"execute()"}`)); err == nil {
		t.Fatal("document block schema accepted an unknown executable field")
	}
	if _, err := CheckJSON[Document]([]byte(`{"chapters":[]}`)); err == nil {
		t.Fatal("document schema accepted a missing version")
	}
}

func TestDocumentRejectsInvalidPayloadsAndReferences(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Passport)
		want   string
	}{
		{"version", func(p *Passport) { p.Views.Document.Version = 1 }, "unsupported document version"},
		{"missing core chapter", func(p *Passport) { p.Views.Document.Chapters = p.Views.Document.Chapters[1:] }, "lacks required chapter"},
		{"duplicate chapter", func(p *Passport) { p.Views.Document.Chapters[1].ID = "overview" }, "invalid document chapter"},
		{"unknown category", func(p *Passport) { p.Views.Document.Chapters[0].ID = "invented" }, "invalid document chapter"},
		{"empty chapter", func(p *Passport) { p.Views.Document.Chapters[0].Blocks = nil }, "invalid document chapter"},
		{"unknown kind", func(p *Passport) { p.Views.Document.Chapters[0].Blocks[0].Kind = "jsx" }, "unknown document block"},
		{"markdown mixed payload", func(p *Passport) { p.Views.Document.Chapters[0].Blocks[0].ReferenceID = "containers" }, "invalid markdown payload"},
		{"wrong typed diagram ref", func(p *Passport) { p.Views.Document.Chapters[2].Blocks[1].ReferenceID = "flow" }, "unknown diagram reference"},
		{"wrong typed scenario ref", func(p *Passport) { p.Views.Document.Chapters[3].Blocks[1].ReferenceID = "containers" }, "invalid scenario placement or reference"},
		{"ref mixed payload", func(p *Passport) { p.Views.Document.Chapters[2].Blocks[1].Markdown = "extra" }, "mixed reference payload"},
		{"unknown claim", func(p *Passport) { p.Views.Document.Chapters[0].Blocks[0].ClaimIDs = []string{"missing"} }, "unknown reference"},
		{"unknown table object", func(p *Passport) { p.Views.Document.Chapters[4].Blocks[1].ObjectIDs = []string{"missing"} }, "unknown reference"},
		{"table mixed payload", func(p *Passport) { p.Views.Document.Chapters[4].Blocks[1].ReferenceID = "flow" }, "invalid object table payload"},
		{"table style", func(p *Passport) { p.Views.Document.Chapters[4].Blocks[1].TableStyle = "arbitrary" }, "invalid object table payload"},
		{"scenario in overview", func(p *Passport) {
			p.Views.Document.Chapters[0].Blocks = append(p.Views.Document.Chapters[0].Blocks, DocumentBlock{Kind: "scenario_ref", ReferenceID: "flow"})
		}, "invalid scenario placement or reference"},
		{"repeated scenario", func(p *Passport) {
			p.Views.Document.Chapters[3].Blocks = append(p.Views.Document.Chapters[3].Blocks, DocumentBlock{Kind: "scenario_ref", ReferenceID: "flow"})
		}, "invalid scenario placement or reference"},
		{"repository in architecture", func(p *Passport) {
			p.Views.Document.Chapters[2].Blocks = append(p.Views.Document.Chapters[2].Blocks, DocumentBlock{Kind: "object_table", ObjectIDs: []string{"repo"}, TableStyle: "objects"})
		}, "mixes repository provenance"},
		{"repository diagram in overview", func(p *Passport) {
			p.Views.Document.Chapters[0].Blocks = append(p.Views.Document.Chapters[0].Blocks, DocumentBlock{Kind: "diagram_ref", ReferenceID: "repos"})
		}, "repository diagrams belong"},
		{"repository table with runtime unit", func(p *Passport) { p.Views.Document.Chapters[1].Blocks[2].ObjectIDs = []string{"app"} }, "mixes repository provenance"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := documentFixture()
			tc.change(&p)
			if err := Validate(p, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestArchitectureHierarchyAndScope(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Passport)
		want   string
	}{
		{"role", func(p *Passport) { p.Model.Objects[1].ArchitectureRole = "process" }, "invalid architecture role"},
		{"repository role", func(p *Passport) { p.Model.Objects[0].ArchitectureRole = "runtime_unit" }, "invalid architecture role"},
		{"unknown parent", func(p *Passport) { p.Model.Objects[2].ParentID = "missing" }, "invalid architectural parent"},
		{"repository parent", func(p *Passport) { p.Model.Objects[2].ParentID = "repo" }, "invalid architectural parent"},
		{"runtime parent", func(p *Passport) { p.Model.Objects[2].ParentID = "engine" }, "runtime unit parent"},
		{"cycle", func(p *Passport) { p.Model.Objects[3].ParentID = "parser" }, "architectural parent cycle"},
		{"unknown scope", func(p *Passport) { p.Views.Diagrams[1].ScopeObjectID = "missing" }, "invalid diagram scope"},
		{"repository scope", func(p *Passport) { p.Views.Diagrams[1].ScopeObjectID = "repo" }, "invalid diagram scope"},
		{"scope role", func(p *Passport) { p.Views.Diagrams[1].ScopeObjectID = "app" }, "incompatible architecture role"},
		{"skipped level", func(p *Passport) { p.Views.Diagrams[2].NodeIDs = []string{"parser"} }, "skips an architectural level"},
		{"mixed level", func(p *Passport) {
			p.Views.Diagrams[2].ScopeObjectID = "system"
			p.Views.Diagrams[2].NodeIDs = []string{"app"}
		}, "mixes architectural levels"},
		{"repository runtime node", func(p *Passport) {
			p.Views.Diagrams[1].NodeIDs = []string{"repo"}
			p.Views.Diagrams[1].RelationIDs = nil
		}, "mixes repository provenance"},
		{"unknown interaction", func(p *Passport) { p.Model.Relations[0].Interaction = "http" }, "invalid relation interaction"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := documentFixture()
			tc.change(&p)
			if err := Validate(p, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDocumentMarkdownSafety(t *testing.T) {
	for _, markdown := range []string{
		`<script>alert(1)</script>`, `<img src=x onerror=alert(1)>`, `<img/src=x onerror=alert(1)>`, `<Widget />`, `<>JSX fragment</>`, `<!-- hidden -->`, "import x from 'x'", "export default () => null",
		`[link](javascript:alert%281%29)`, `[link](java&#x73;cript:alert(1))`, "[link](java\nscript:alert(1))", `[link](%6aavascript%3Aalert(1))`, `[link](data:text/html,payload)`, "[x][ref]\n\n[ref]: vbscript:execute",
	} {
		p := documentFixture()
		p.Views.Document.Chapters[0].Blocks[0].Markdown = markdown
		if err := Validate(p, nil); err == nil {
			t.Errorf("accepted unsafe markdown %q", markdown)
		}
	}
	for _, markdown := range []string{"## Purpose\n\nA **supported** paragraph with [source](https://example.com/docs).", "Use `x < y` for comparison.", "<https://example.com>", "<team@example.com>", "[Email](mailto:team@example.com)", "Data: input and output. Metadata: supported fields."} {
		if err := validateMarkdown(markdown); err != nil {
			t.Errorf("rejected plain markdown %q: %v", markdown, err)
		}
	}
}

func TestDocumentMarkdownCodeExamples(t *testing.T) {
	for _, markdown := range []string{
		"```sh\nexport DATABASE_PATH=/var/lib/product.db\n```",
		"~~~html\n<script>alert('literal example')</script>\n~~~",
		"````markdown\n```js\nexport default example\n```\n````",
		"Use `<host>` and `export DATABASE_PATH=...` as literal examples.",
		"A safe example: `[link](javascript:example)`.",
		"Double delimiters allow embedded ticks: ``a ` b <script>``.",
		"Set the server to &lt;host&gt;.",
		`Use \<host> as a placeholder.`,
		"    export DATABASE_PATH=/var/lib/product.db\n    <host>\n",
		"```html\n<script>unclosed fence remains a code block",
	} {
		if err := validateMarkdown(markdown); err != nil {
			t.Errorf("rejected literal code %q: %v", markdown, err)
		}
	}
	for _, markdown := range []string{
		"```sh\nexport DATABASE_PATH=/var/lib/product.db\n```\nexport default () => null",
		"~~~html\n<script>literal</script>\n~~~\n<script>actual</script>",
		"`literal` [link](javascript:alert(1))",
		"`unclosed <script>",
		"\\`<script>\\`",
		"``literal` <script>",
		"```sh`invalid-info\nexport default example",
		"Paragraph continues\n    <script>",
		"```text\nliteral\n````\n<script>",
		"~~~text\nliteral\n```\n~~~\n[link](javascript:alert(1))",
		"    literal indented example\n\n<script>",
	} {
		if err := validateMarkdown(markdown); err == nil {
			t.Errorf("code example hid rendered unsafe content %q", markdown)
		}
	}
}
