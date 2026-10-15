// Package passport defines the shared, source-backed product passport contract.
package passport

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

const repositoriesCategory = "repositories"

// SectionIDs is the allowed reading order; secondary sections require useful evidence.
func SectionIDs() []string {
	return []string{"overview", repositoriesCategory, "architecture", "technologies", "scenarios", "data", "runtime", "development", "operations", "decisions", "ownership", "capabilities", "users", "domain"}
}

// RequiredSectionIDs contains the core questions every passport must answer.
func RequiredSectionIDs() []string {
	return []string{"overview", repositoriesCategory, "architecture", "technologies", "scenarios"}
}

// Evidence anchors an assertion to exact lines of a collected document.
type Evidence struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id" description:"Exact document_id returned by list_sources or read_source; never a file path or snapshot ID."`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
}

// Claim distinguishes source-backed descriptions from interpretation.
type Claim struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Basis       string   `json:"basis" enum:"source_backed,inferred,unknown" jsonschema:"One of source_backed, inferred, unknown. A source pointer is not expert approval."`
	EvidenceIDs []string `json:"evidence_ids" description:"Exact evidence IDs supporting this claim. Required for source_backed claims before finalization; referenced evidence may be saved in a later part."`
}

// Object is a shared actor, capability, domain entity, component or environment.
type Object struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind" enum:"repository,actor,capability,entity,component,store,external_system,interface,environment,decision,owner" description:"Use component for a software system or runtime unit, with architecture_role specifying its level. A repository describes source ownership." jsonschema:"One of repository, actor, capability, entity, component, store, external_system, interface, environment, decision, owner. Repository objects describe source ownership, never runtime units."`
	Name             string   `json:"name"`
	Summary          string   `json:"summary"`
	Technology       string   `json:"technology"`
	Status           string   `json:"status" enum:"implemented,in_development,experimental,planned,unknown" jsonschema:"One of implemented, in_development, experimental, planned, unknown. Do not confuse feature status with evidence confidence."`
	Inputs           []string `json:"inputs"`
	Outputs          []string `json:"outputs"`
	Contracts        []string `json:"contracts"`
	ClaimIDs         []string `json:"claim_ids" description:"Supporting canonical claim IDs. Can be filled in a later draft part; must be nonempty and resolved before finalization."`
	ArchitectureRole string   `json:"architecture_role,omitempty" enum:"software_system,runtime_unit,component" description:"Architectural level of a component; distinct from kind. Omit for source repository provenance." jsonschema:"Optional software_system, runtime_unit or component. Describes logical structure, not source ownership."`
	ParentID         string   `json:"parent_id,omitempty" jsonschema:"Optional architectural parent object ID. Never a repository provenance link."`
}

// Relation describes a directed connection between shared objects.
type Relation struct {
	ID          string   `json:"id"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Label       string   `json:"label"`
	Kind        string   `json:"kind" enum:"call,data,dependency,ownership,association,transition" jsonschema:"One of call, data, dependency, ownership, association, transition."`
	ClaimIDs    []string `json:"claim_ids"`
	Interaction string   `json:"interaction,omitempty" enum:"in_process,sync,async,artifact,build,unknown" jsonschema:"Optional in_process, sync, async, artifact, build or unknown."`
	Contract    string   `json:"contract,omitempty" jsonschema:"Optional specific protocol, payload or artifact contract."`
}

// Step is one part of a product scenario.
type Step struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	ObjectIDs []string `json:"object_ids"`
	ClaimIDs  []string `json:"claim_ids"`
}

// Scenario explains a product outcome, its steps and failure conditions.
type Scenario struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Inputs       []string `json:"inputs"`
	Outputs      []string `json:"outputs"`
	FailureModes []string `json:"failure_modes"`
	Steps        []Step   `json:"steps"`
}

// Model is the canonical LLM-created description reused across all views.
type Model struct {
	Title         string     `json:"title"`
	Tagline       string     `json:"tagline"`
	Summary       string     `json:"summary"`
	Objects       []Object   `json:"objects"`
	Relations     []Relation `json:"relations"`
	Scenarios     []Scenario `json:"scenarios"`
	Claims        []Claim    `json:"claims"`
	Evidence      []Evidence `json:"evidence"`
	OpenQuestions []string   `json:"open_questions"`
}

// Diagram selects objects and relations from the canonical model, without generated code.
type Diagram struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Kind          string   `json:"kind" jsonschema:"One of repositories, context, containers, components, domain, data, deployment, states. In v2, repositories views contain only repository objects; other views exclude repository objects."`
	Summary       string   `json:"summary"`
	NodeIDs       []string `json:"node_ids"`
	RelationIDs   []string `json:"relation_ids"`
	ScopeObjectID string   `json:"scope_object_id,omitempty" jsonschema:"Optional object ID whose immediate architectural children this view explains."`
	Question      string   `json:"question,omitempty"`
}

// Section is one standardized part of the passport.
type Section struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status" jsonschema:"One of available, insufficient_evidence, not_applicable. Explain missing evidence in summary."`
	Summary     string   `json:"summary"`
	ClaimIDs    []string `json:"claim_ids"`
	ObjectIDs   []string `json:"object_ids"`
	DiagramIDs  []string `json:"diagram_ids"`
	ScenarioIDs []string `json:"scenario_ids"`
	Checklist   []string `json:"checklist"`
}

// Views contains LLM-written sections and diagram definitions sharing model IDs.
type Views struct {
	Sections  []Section         `json:"sections"`
	Diagrams  []Diagram         `json:"diagrams"`
	CardTheme *ProductCardTheme `json:"card_theme"`
	Document  *Document         `json:"document,omitempty"`
}

// Document is an authored reading path through the canonical facts and views.
type Document struct {
	Version  int       `json:"version" jsonschema:"Must be 2."`
	Chapters []Chapter `json:"chapters"`
}

// Chapter answers one standard category question using ordered typed blocks.
type Chapter struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Question string          `json:"question"`
	Blocks   []DocumentBlock `json:"blocks"`
}

// DocumentBlock has exactly one payload, selected by Kind. Claims support prose
// and tables; diagrams and scenarios carry their own canonical references.
type DocumentBlock struct {
	Kind        string   `json:"kind" jsonschema:"One of markdown, diagram_ref, scenario_ref, object_table."`
	Markdown    string   `json:"markdown,omitempty" jsonschema:"Plain Markdown only, without raw HTML or executable JS/MDX."`
	ReferenceID string   `json:"reference_id,omitempty"`
	ObjectIDs   []string `json:"object_ids,omitempty"`
	ClaimIDs    []string `json:"claim_ids,omitempty"`
	TableStyle  string   `json:"table_style,omitempty" jsonschema:"For object_table only: objects, repositories or technologies."`
}

// ChapterIDs is the default reading order; legacy section ordering is unchanged.
func ChapterIDs() []string {
	return []string{"overview", repositoriesCategory, "architecture", "scenarios", "technologies", "data", "runtime", "development", "operations", "decisions", "ownership", "capabilities", "users", "domain"}
}

// ProductCardTheme scopes product colors and original logos to product content.
type ProductCardTheme struct {
	AccentColor     string        `json:"accent_color" jsonschema:"Brand accent as a six-digit hex color, for example #6550B9."`
	BackgroundColor string        `json:"background_color" jsonschema:"Product card background as a six-digit hex color."`
	LogoURL         string        `json:"logo_url" jsonschema:"Exact original logo URL from the supplied verified logo candidates, or empty. Never invent a logo URL."`
	Sources         []BrandSource `json:"sources" jsonschema:"Brand provenance populated by the server; leave empty."`
}

// BrandSource records the actual material used for a product's visual identity.
type BrandSource struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

var themeColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Passport is an immutable draft, pending expert review.
type Passport struct {
	Model Model `json:"model"`
	Views Views `json:"views"`
}

// Schema derives a strict schema from the same Go types used by API responses.
func Schema[T any]() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		return nil, fmt.Errorf("build JSON schema: %w", err)
	}
	return schema, nil
}

// OutputSchema is Schema with non-nullable arrays. Nullable unions inflate the
// grammar compiled by strict structured-output modes beyond provider limits.
func OutputSchema[T any]() (*jsonschema.Schema, error) {
	schema, err := Schema[T]()
	if err != nil {
		return nil, err
	}
	var visit func(*jsonschema.Schema)
	visit = func(s *jsonschema.Schema) {
		if s == nil {
			return
		}
		if slices.Equal(s.Types, []string{"null", "array"}) {
			s.Type, s.Types = "array", nil
		}
		visit(s.Items)
		for _, child := range s.Properties {
			visit(child)
		}
		for _, child := range s.Defs {
			visit(child)
		}
	}
	visit(schema)
	return schema, nil
}

// CheckJSON validates the original response, including unknown and missing fields.
func CheckJSON[T any](data []byte) (T, error) {
	var result T
	schema, err := Schema[T]()
	if err != nil {
		return result, err
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return result, fmt.Errorf("resolve JSON schema: %w", err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return result, errors.New("model returned invalid JSON")
	}
	if err := resolved.Validate(value); err != nil {
		return result, fmt.Errorf("response schema: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// Validate checks cross-object references and source bounds independently of JSON syntax.
//
//nolint:gocyclo // Validate all cross references together against one immutable model.
func Validate(p Passport, documents map[string]int) error {
	v2 := p.Views.Document != nil
	if p.Views.CardTheme != nil && (!themeColor.MatchString(p.Views.CardTheme.AccentColor) || !themeColor.MatchString(p.Views.CardTheme.BackgroundColor)) {
		return errors.New("product theme colors must be six-digit hex colors")
	}
	index, err := validateModel(p.Model, documents, false)
	if err != nil {
		return err
	}
	register := index.register
	claims, objects, relations, scenarios := index.claims, index.objects, index.relations, index.scenarios
	diagrams := make(map[string]bool)
	if err := validateArchitecture(p.Model.Objects, p.Views.Diagrams, v2); err != nil {
		return err
	}
	for _, d := range p.Views.Diagrams {
		if err := register(d.ID); err != nil {
			return err
		}
		diagrams[d.ID] = true
		if !slices.Contains([]string{repositoriesCategory, "context", "containers", "components", "domain", "data", "deployment", "states"}, d.Kind) {
			return fmt.Errorf("invalid diagram kind: %s", d.ID)
		}
		if err := checkRefs(d.NodeIDs, objects); err != nil {
			return err
		}
		for _, object := range p.Model.Objects {
			if v2 && slices.Contains(d.NodeIDs, object.ID) && (object.Kind == "repository") != (d.Kind == repositoriesCategory) {
				return fmt.Errorf("diagram mixes repository provenance and logical structure: %s", d.ID)
			}
		}
		for _, id := range d.RelationIDs {
			r, ok := relations[id]
			if !ok || !slices.Contains(d.NodeIDs, r.From) || !slices.Contains(d.NodeIDs, r.To) {
				return fmt.Errorf("diagram has a dangling relation: %s", id)
			}
		}
	}
	sections := make(map[string]bool)
	for _, s := range p.Views.Sections {
		if !slices.Contains(SectionIDs(), s.ID) || sections[s.ID] {
			return fmt.Errorf("invalid section: %s", s.ID)
		}
		sections[s.ID] = true
		if v2 && s.ID != "scenarios" && len(s.ScenarioIDs) > 0 {
			return fmt.Errorf("execution scenarios belong only in the scenarios section: %s", s.ID)
		}
		if s.Title == "" || s.Summary == "" || !slices.Contains([]string{"available", "insufficient_evidence", "not_applicable"}, s.Status) {
			return fmt.Errorf("invalid section content: %s", s.ID)
		}
		for _, pair := range []struct {
			refs  []string
			known map[string]bool
		}{{s.ClaimIDs, claims}, {s.ObjectIDs, objects}, {s.DiagramIDs, diagrams}, {s.ScenarioIDs, scenarios}} {
			if err := checkRefs(pair.refs, pair.known); err != nil {
				return err
			}
		}
	}
	if v2 {
		if err := validateDocument(*p.Views.Document, claims, objects, diagrams, scenarios, p.Model.Objects, p.Views.Diagrams); err != nil {
			return err
		}
	} else {
		for _, id := range RequiredSectionIDs() {
			if !sections[id] {
				return fmt.Errorf("passport lacks required section: %s", id)
			}
		}
	}
	return nil
}

func checkRefs(refs []string, known map[string]bool) error {
	for _, id := range refs {
		if !known[id] {
			return fmt.Errorf("unknown reference: %s", id)
		}
	}
	return nil
}
