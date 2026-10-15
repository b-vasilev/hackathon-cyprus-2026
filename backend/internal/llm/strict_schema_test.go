package llm

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/google/jsonschema-go/jsonschema"
)

func schemaMap[T any](t *testing.T) map[string]any {
	t.Helper()
	schema, err := passport.Schema[T]()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPassportStrictSchemaPreservesPublicOptionals(t *testing.T) {
	for name, schema := range map[string]map[string]any{
		"model": schemaMap[passport.Model](t),
		"views": schemaMap[passport.Views](t),
		"block": schemaMap[passport.DocumentBlock](t),
	} {
		t.Run(name, func(t *testing.T) {
			before, _ := json.Marshal(schema)
			strict, err := openAIStrictSchema(schema)
			if err != nil {
				t.Fatal(err)
			}
			assertStrictObjects(t, strict)
			after, _ := json.Marshal(schema)
			if !bytes.Equal(before, after) {
				t.Fatal("wire adaptation mutated the public schema")
			}
			var property string
			parent := schema
			switch name {
			case "model":
				property = "architecture_role"
				parent = schema["properties"].(map[string]any)["objects"].(map[string]any)["items"].(map[string]any)
			case "views":
				property = "document"
			case "block":
				property = "markdown"
			}
			if schemaRequired(parent)[property] {
				t.Fatalf("public field %s unexpectedly became required", property)
			}
			properties := parent["properties"].(map[string]any)
			t.Logf("public %s: %d properties, %d required; %s remains optional", name, len(properties), len(schemaRequired(parent)), property)
		})
	}
}

func assertStrictObjects(t *testing.T, schema map[string]any) {
	t.Helper()
	if properties, ok := schema["properties"].(map[string]any); ok {
		if schema["additionalProperties"] != false || len(schemaRequired(schema)) != len(properties) {
			t.Fatalf("object is not strict: required=%v properties=%v", schema["required"], properties)
		}
		for name, value := range properties {
			if !schemaRequired(schema)[name] {
				t.Errorf("property %s is not required", name)
			}
			assertStrictObjects(t, value.(map[string]any))
		}
	}
	for _, key := range []string{"$defs", "definitions"} {
		definitions, _ := schema[key].(map[string]any)
		for _, definition := range definitions {
			assertStrictObjects(t, definition.(map[string]any))
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		assertStrictObjects(t, items)
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		branches, _ := schema[key].([]any)
		for _, branch := range branches {
			assertStrictObjects(t, branch.(map[string]any))
		}
	}
}

func TestStrictSchemaRoundTripOptionalFields(t *testing.T) {
	schema := schemaMap[passport.DocumentBlock](t)
	strict, err := openAIStrictSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	wire := []byte(`{"kind":"markdown","markdown":"A fact.","reference_id":null,"object_ids":null,"claim_ids":null,"table_style":null}`)
	validateSchema(t, strict, wire)
	if _, checkErr := passport.CheckJSON[passport.DocumentBlock](wire); checkErr == nil {
		t.Fatal("public schema unexpectedly accepted optional string nulls")
	}
	normalized, err := restoreOptionalFields(wire, schema)
	if err != nil {
		t.Fatal(err)
	}
	block, err := passport.CheckJSON[passport.DocumentBlock](normalized)
	if err != nil || block.Markdown != "A fact." || block.Kind != "markdown" {
		t.Fatalf("strict response failed original contract: %s; %v", normalized, err)
	}
	for _, invalid := range []string{
		`{"kind":null,"markdown":null}`,
		`{"markdown":null}`,
		`{"kind":"markdown","markdown":null,"unknown":null}`,
	} {
		normalized, err := restoreOptionalFields([]byte(invalid), schema)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := passport.CheckJSON[passport.DocumentBlock](normalized); err == nil {
			t.Fatalf("normalization hid invalid output: %s", invalid)
		}
	}
}

func TestStrictSchemaNestedRefsAndUnion(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(`{
		"type":"object","properties":{"rows":{"type":"array","items":{"$ref":"#/$defs/Row"}}},"required":["rows"],
		"$defs":{"Row":{"anyOf":[
			{"type":"object","properties":{"kind":{"const":"required"},"value":{"type":["string","null"]}},"required":["kind","value"]},
			{"type":"object","properties":{"kind":{"const":"optional"},"value":{"enum":["known"]},"count":{"type":"integer"}},"required":["kind","count"]}
		]}}
	}`), &schema); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(schema)
	strict, err := openAIStrictSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
	assertStrictObjects(t, strict)
	wire := []byte(`{"rows":[{"kind":"required","value":null},{"kind":"optional","value":null,"count":9007199254740993}]}`)
	validateSchema(t, strict, wire)
	normalized, err := restoreOptionalFields(wire, schema)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(normalized), `"value":null`) || !strings.Contains(string(normalized), "9007199254740993") {
		t.Fatalf("lost a required null or integer precision: %s", normalized)
	}
	validateSchema(t, schema, normalized)
	after, _ := json.Marshal(schema)
	if !bytes.Equal(before, after) {
		t.Fatal("response normalization mutated the public schema")
	}
}

func validateSchema(t *testing.T, schema map[string]any, data []byte) {
	t.Helper()
	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var typed jsonschema.Schema
	if decodeErr := json.Unmarshal(encoded, &typed); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	resolved, err := typed.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if err := resolved.Validate(value); err != nil {
		t.Fatalf("schema rejected %s: %v", data, err)
	}
}
