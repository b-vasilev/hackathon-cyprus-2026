package analysis

import "testing"

// Real failure from production: the model added one character to a document ID.
func TestResolveDocumentIDRepairsCopyMistakes(t *testing.T) {
	lines := map[string]int{"YDCKATYFGYXZZRRHDMFXKQWLP7": 10, "UVDOOSYSMLQ2YWK3XQMGKPMBIW": 5, "TWXTSE34PHHHSZC5AYZIEDGH4P": 6}
	for typed, want := range map[string]string{
		"YDCKATYFGYXZZRRHDMFXKQWLP77": "YDCKATYFGYXZZRRHDMFXKQWLP7",
		"UVDOOSYSMLQ2YWK3XQMGKPMBlW":  "UVDOOSYSMLQ2YWK3XQMGKPMBIW",
		"TWXTSE34PHHHSZC5AYZIEDGH4P":  "TWXTSE34PHHHSZC5AYZIEDGH4P",
		"AAAAAAAAAAAAAAAAAAAAAAAAAA":  "AAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		if got := resolveDocumentID(typed, lines); got != want {
			t.Errorf("resolveDocumentID(%q) = %q, want %q", typed, got, want)
		}
	}
	// Two equally close documents are ambiguous, so the ID is left for validation to reject.
	ambiguous := map[string]int{"ABCDEFGHIJ": 1, "ABCDEFGHIK": 1}
	if got := resolveDocumentID("ABCDEFGHIX", ambiguous); got != "ABCDEFGHIX" {
		t.Errorf("ambiguous ID resolved to %q", got)
	}
}
