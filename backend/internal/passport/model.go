// Package passport defines the shared, source-backed product passport contract.
package passport

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

// SectionIDs is the stable reading order for every product.
func SectionIDs() []string {
	return []string{"overview", "users", "capabilities", "domain", "scenarios", "architecture", "data", "runtime", "operations", "development", "decisions", "ownership"}
}

// Evidence anchors an assertion to exact lines of a collected document.
type Evidence struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
}

// Claim distinguishes source-backed descriptions from interpretation.
type Claim struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Basis       string   `json:"basis" jsonschema:"One of source_backed, inferred, unknown. A source pointer is not expert approval."`
	EvidenceIDs []string `json:"evidence_ids"`
}

// Object is a shared actor, capability, domain entity, component or environment.
type Object struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind" jsonschema:"One of actor, capability, entity, component, store, external_system, interface, environment, decision, owner."`
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Technology string   `json:"technology"`
	Status     string   `json:"status" jsonschema:"One of implemented, in_development, experimental, planned, unknown. Do not confuse feature status with evidence confidence."`
	Inputs     []string `json:"inputs"`
	Outputs    []string `json:"outputs"`
	Contracts  []string `json:"contracts"`
	ClaimIDs   []string `json:"claim_ids"`
}

// Relation describes a directed connection between shared objects.
type Relation struct {
	ID       string   `json:"id"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind" jsonschema:"One of call, data, dependency, ownership, association, transition."`
	ClaimIDs []string `json:"claim_ids"`
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
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Kind        string   `json:"kind" jsonschema:"One of context, components, domain, data, deployment, states."`
	Summary     string   `json:"summary"`
	NodeIDs     []string `json:"node_ids"`
	RelationIDs []string `json:"relation_ids"`
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
	Sections []Section `json:"sections"`
	Diagrams []Diagram `json:"diagrams"`
}

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
	ids := make(map[string]bool)
	register := func(id string) error {
		if id == "" || ids[id] {
			return fmt.Errorf("duplicate or empty object ID: %q", id)
		}
		ids[id] = true
		return nil
	}
	evidence := make(map[string]bool)
	claims := make(map[string]bool)
	objects := make(map[string]bool)
	relations := make(map[string]Relation)
	scenarios := make(map[string]bool)
	diagrams := make(map[string]bool)
	for _, e := range p.Model.Evidence {
		if err := register(e.ID); err != nil {
			return err
		}
		evidence[e.ID] = true
		lines, ok := documents[e.DocumentID]
		if !ok || e.StartLine < 1 || e.EndLine < e.StartLine || e.EndLine > lines {
			return fmt.Errorf("invalid evidence anchor: %s", e.ID)
		}
	}
	for _, c := range p.Model.Claims {
		if err := register(c.ID); err != nil {
			return err
		}
		claims[c.ID] = true
		if c.Text == "" || !slices.Contains([]string{"source_backed", "inferred", "unknown"}, c.Basis) {
			return fmt.Errorf("invalid claim: %s", c.ID)
		}
		if c.Basis == "source_backed" && len(c.EvidenceIDs) == 0 {
			return fmt.Errorf("source-backed claim lacks evidence: %s", c.ID)
		}
		if err := checkRefs(c.EvidenceIDs, evidence); err != nil {
			return err
		}
	}
	for _, o := range p.Model.Objects {
		if err := register(o.ID); err != nil {
			return err
		}
		objects[o.ID] = true
		if o.Name == "" || !slices.Contains([]string{"actor", "capability", "entity", "component", "store", "external_system", "interface", "environment", "decision", "owner"}, o.Kind) || !slices.Contains([]string{"implemented", "in_development", "experimental", "planned", "unknown"}, o.Status) {
			return fmt.Errorf("invalid model object: %s", o.ID)
		}
		if len(o.ClaimIDs) == 0 {
			return fmt.Errorf("object lacks supporting claims: %s", o.ID)
		}
		if err := checkRefs(o.ClaimIDs, claims); err != nil {
			return err
		}
	}
	for _, r := range p.Model.Relations {
		if err := register(r.ID); err != nil {
			return err
		}
		relations[r.ID] = r
		if !objects[r.From] || !objects[r.To] || len(r.ClaimIDs) == 0 || !slices.Contains([]string{"call", "data", "dependency", "ownership", "association", "transition"}, r.Kind) {
			return fmt.Errorf("invalid relation: %s", r.ID)
		}
		if err := checkRefs(r.ClaimIDs, claims); err != nil {
			return err
		}
	}
	for _, s := range p.Model.Scenarios {
		if err := register(s.ID); err != nil {
			return err
		}
		scenarios[s.ID] = true
		for _, step := range s.Steps {
			if err := register(step.ID); err != nil {
				return err
			}
			if err := checkRefs(step.ObjectIDs, objects); err != nil {
				return err
			}
			if err := checkRefs(step.ClaimIDs, claims); err != nil {
				return err
			}
		}
	}
	for _, d := range p.Views.Diagrams {
		if err := register(d.ID); err != nil {
			return err
		}
		diagrams[d.ID] = true
		if !slices.Contains([]string{"context", "components", "domain", "data", "deployment", "states"}, d.Kind) {
			return fmt.Errorf("invalid diagram kind: %s", d.ID)
		}
		if err := checkRefs(d.NodeIDs, objects); err != nil {
			return err
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
	if len(sections) != len(SectionIDs()) || p.Model.Title == "" || p.Model.Summary == "" {
		return errors.New("passport must contain a product title, summary and all 12 sections")
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
