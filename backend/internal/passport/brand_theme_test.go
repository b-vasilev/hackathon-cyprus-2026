package passport

import (
	"slices"
	"testing"
)

func TestBrandThemeSupportsStrictStructuredOutput(t *testing.T) {
	schema, err := Schema[ProductCardTheme]()
	if err != nil {
		t.Fatal(err)
	}
	// OpenAI strict structured output requires every property to be required,
	// including empty logo and provenance fields when no asset was selected.
	for name := range schema.Properties {
		if !slices.Contains(schema.Required, name) {
			t.Fatal("brand schema is incompatible with strict structured output", name)
		}
	}
}
