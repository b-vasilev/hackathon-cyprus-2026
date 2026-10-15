package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const schemaTypeKey = "type"

// openAIStrictSchema adapts the wire contract without changing the public Go
// schema: OpenAI requires every property, so optional values become nullable.
func openAIStrictSchema(schema map[string]any) (map[string]any, error) {
	encoded, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("encode output schema: %w", err)
	}
	var strict map[string]any
	if err := json.Unmarshal(encoded, &strict); err != nil {
		return nil, fmt.Errorf("decode output schema: %w", err)
	}
	makeSchemaStrict(strict)
	return strict, nil
}

func makeSchemaStrict(schema map[string]any) {
	for _, keyword := range []string{"$defs", "definitions"} {
		definitions, _ := schema[keyword].(map[string]any)
		for _, value := range definitions {
			definition, _ := value.(map[string]any)
			makeSchemaStrict(definition)
		}
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		required := schemaRequired(schema)
		names := make([]string, 0, len(properties))
		for name, value := range properties {
			property, _ := value.(map[string]any)
			makeSchemaStrict(property)
			if !required[name] && property[schemaTypeKey] != "array" {
				properties[name] = map[string]any{"anyOf": []any{property, map[string]any{schemaTypeKey: "null"}}}
			}
			names = append(names, name)
		}
		slices.Sort(names)
		schema["required"] = names
		schema["additionalProperties"] = false
	}
	objectType := schema[schemaTypeKey] == "object"
	if types, ok := schema[schemaTypeKey].([]any); ok {
		objectType = slices.Contains(types, "object")
	}
	if objectType {
		schema["additionalProperties"] = false
	}
	if items, ok := schema["items"].(map[string]any); ok {
		makeSchemaStrict(items)
	}
	for _, keyword := range []string{"anyOf", "oneOf", "allOf"} {
		branches, _ := schema[keyword].([]any)
		for _, value := range branches {
			branch, _ := value.(map[string]any)
			makeSchemaStrict(branch)
		}
	}
}

func schemaRequired(schema map[string]any) map[string]bool {
	required := make(map[string]bool)
	switch names := schema["required"].(type) {
	case []any:
		for _, name := range names {
			if name, ok := name.(string); ok {
				required[name] = true
			}
		}
	case []string:
		for _, name := range names {
			required[name] = true
		}
	}
	return required
}

// restoreOptionalFields converts the wire's optional nulls to omitted fields
// before the caller validates the original schema. Required/unknown fields stay
// intact, so this never repairs missing required values or hides unknown fields.
func restoreOptionalFields(data json.RawMessage, schema map[string]any) (json.RawMessage, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode structured output: %w", err)
	}
	if !omitOptionalNulls(value, schema, schema) {
		return data, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode structured output: %w", err)
	}
	return encoded, nil
}

func omitOptionalNulls(value any, schema, root map[string]any) bool {
	if ref, ok := schema["$ref"].(string); ok {
		if target := localSchemaRef(root, ref); target != nil {
			return omitOptionalNulls(value, target, root)
		}
	}
	changed := false
	switch value := value.(type) {
	case map[string]any:
		required := schemaRequired(schema)
		properties, _ := schema["properties"].(map[string]any)
		for name, child := range properties {
			field, exists := value[name]
			if !exists {
				continue
			}
			if field == nil && !required[name] {
				delete(value, name)
				changed = true
				continue
			}
			property, _ := child.(map[string]any)
			changed = omitOptionalNulls(field, property, root) || changed
		}
	case []any:
		items, _ := schema["items"].(map[string]any)
		for _, item := range value {
			changed = omitOptionalNulls(item, items, root) || changed
		}
	}
	// In a union, normalize only a matching wire branch. Applying every branch
	// could erase a field that another branch requires.
	for _, keyword := range []string{"anyOf", "oneOf"} {
		branches, _ := schema[keyword].([]any)
		for _, child := range branches {
			branch, _ := child.(map[string]any)
			if matchesStrictBranch(value, branch, root) {
				return omitOptionalNulls(value, branch, root) || changed
			}
		}
	}
	return changed
}

func localSchemaRef(root map[string]any, ref string) map[string]any {
	if ref == "#" {
		return root
	}
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var current any = root
	for part := range strings.SplitSeq(ref[2:], "/") {
		object, _ := current.(map[string]any)
		current = object[strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")]
	}
	result, _ := current.(map[string]any)
	return result
}

func matchesStrictBranch(value any, branch, root map[string]any) bool {
	combined := make(map[string]any, len(branch)+2)
	maps.Copy(combined, branch)
	for _, keyword := range []string{"$defs", "definitions"} {
		if definitions, ok := root[keyword]; ok {
			combined[keyword] = definitions
		}
	}
	strict, err := openAIStrictSchema(combined)
	if err != nil {
		return false
	}
	encoded, err := json.Marshal(strict)
	if err != nil {
		return false
	}
	var schema jsonschema.Schema
	if json.Unmarshal(encoded, &schema) != nil {
		return false
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return false
	}
	// The validator expects ordinary decoded JSON numbers. Convert only the
	// matching copy; the returned document retains exact json.Number values.
	encoded, err = json.Marshal(value)
	if err != nil {
		return false
	}
	var candidate any
	if json.Unmarshal(encoded, &candidate) != nil {
		return false
	}
	return resolved.Validate(candidate) == nil
}
