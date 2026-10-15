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
		{"missing section", func(p *Passport) { p.Views.Sections = p.Views.Sections[:11] }},
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
