package passport

import (
	"testing"
)

func TestEvidenceAndReferenceValidation(t *testing.T) {
	t.Parallel()
	base := Passport{Model: Model{Title: "Product", Summary: "Purpose", Evidence: []Evidence{{ID: "evidence", DocumentID: "doc", StartLine: 1, EndLine: 1}}, Claims: []Claim{{ID: "claim", Text: "A fact.", Basis: "source_backed", EvidenceIDs: []string{"evidence"}}}}, Views: Views{Sections: make([]Section, 0)}}
	for _, id := range SectionIDs() {
		base.Views.Sections = append(base.Views.Sections, Section{ID: id, Title: id, Status: "insufficient_evidence", Summary: "Unknown from supplied sources."})
	}
	if err := Validate(base, map[string]int{"doc": 1}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Passport)
	}{
		{"out-of-range evidence", func(p *Passport) { p.Model.Evidence[0].EndLine = 2 }},
		{"unknown evidence", func(p *Passport) { p.Model.Claims[0].EvidenceIDs = []string{"missing"} }},
		{"missing core section", func(p *Passport) { p.Views.Sections = p.Views.Sections[1:] }},
		{"unsupported confidence", func(p *Passport) { p.Model.Claims[0].Basis = "verified" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := base
			p.Model.Evidence = append([]Evidence{}, base.Model.Evidence...)
			p.Model.Claims = append([]Claim{}, base.Model.Claims...)
			tc.change(&p)
			if err := Validate(p, map[string]int{"doc": 1}); err == nil {
				t.Fatal("accepted invalid passport")
			}
		})
	}
}

func TestSchemaRejectsUnknownAndMissingFields(t *testing.T) {
	t.Parallel()
	for _, input := range []string{`{}`, `{"sections":[],"diagrams":[],"jsx":"execute()"}`} {
		if _, err := CheckJSON[Views]([]byte(input)); err == nil {
			t.Fatal("invalid content accepted")
		}
	}
}

func TestCoreSectionsDoNotRequireEmptySecondaryPages(t *testing.T) {
	p := Passport{Model: Model{Title: "Product", Summary: "Purpose"}}
	for _, id := range RequiredSectionIDs() {
		p.Views.Sections = append(p.Views.Sections, Section{ID: id, Title: id, Summary: "A precise evidence gap.", Status: "insufficient_evidence"})
	}
	if err := Validate(p, nil); err != nil {
		t.Fatal(err)
	}
	p.Model.Claims = []Claim{{ID: "claim", Text: "Known repository.", Basis: "inferred"}}
	p.Model.Objects = []Object{{ID: "repo", Kind: "repository", Name: "Source", Status: "implemented", ClaimIDs: []string{"claim"}}}
	p.Views.Diagrams = []Diagram{{ID: "logical", Kind: "components", NodeIDs: []string{"repo"}}}
	if err := Validate(p, nil); err != nil {
		t.Fatalf("legacy diagram placement must remain compatible: %v", err)
	}
	p.Views.Diagrams[0].Kind = "repositories"
	if err := Validate(p, nil); err != nil {
		t.Fatal(err)
	}
	p.Model.Scenarios = []Scenario{{ID: "flow"}}
	p.Views.Sections[0].ScenarioIDs = []string{"flow"}
	if err := Validate(p, nil); err != nil {
		t.Fatalf("legacy scenario placement must remain compatible: %v", err)
	}
}

func TestOutputSchemaHasNoNullableArrays(t *testing.T) {
	t.Parallel()
	schema, err := OutputSchema[Passport]()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(schema)
	if strings.Contains(string(data), `["null","array"]`) {
		t.Fatal("output schema contains nullable arrays that inflate strict grammars")
	}
}
