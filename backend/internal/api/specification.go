package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/google/jsonschema-go/jsonschema"
)

// Specification builds OpenAPI from the same Go contracts used by handlers and LLM generation.
const methodGet = "get"

func Specification() ([]byte, error) {
	var spec map[string]any
	if err := json.Unmarshal(openAPI, &spec); err != nil {
		return nil, fmt.Errorf("read base OpenAPI: %w", err)
	}
	types := map[string]func() (*jsonschema.Schema, error){
		"Connection": passport.Schema[llmgateway.Connection], "ConnectionConfig": passport.Schema[llmgateway.Config], "Model": passport.Schema[llmgateway.Model], "Parameter": passport.Schema[llmgateway.Parameter],
		"Product": passport.Schema[product.Product], "ProductInput": passport.Schema[product.Input], "SourceInput": passport.Schema[source.Input],
		"GenerationInput": passport.Schema[generation.Input], "Run": passport.Schema[generation.Run], "Revision": passport.Schema[generation.Revision], "RevisionInfo": passport.Schema[generation.RevisionInfo], "EvidenceView": passport.Schema[generation.EvidenceView],
		"Passport": passport.Schema[passport.Passport], "ProductModel": passport.Schema[passport.Model], "ProductObject": passport.Schema[passport.Object], "Claim": passport.Schema[passport.Claim], "Evidence": passport.Schema[passport.Evidence], "Relation": passport.Schema[passport.Relation], "Scenario": passport.Schema[passport.Scenario], "Section": passport.Schema[passport.Section], "Diagram": passport.Schema[passport.Diagram],
		"Problem": passport.Schema[Problem], "Probe": passport.Schema[Probe], "ProbeResult": passport.Schema[ProbeResult], "FactoryStatus": passport.Schema[FactoryStatus],
	}
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	for name, build := range types {
		schema, err := build()
		if err != nil {
			return nil, err
		}
		schemas[name] = schema
	}
	patch, err := passport.Schema[llmgateway.Config]()
	if err != nil {
		return nil, err
	}
	patch.Required = nil
	schemas["ConnectionPatch"] = patch
	paths := spec["paths"].(map[string]any)
	for _, route := range []struct {
		method, path, id, input, output string
		status                          int
		array                           bool
	}{
		{methodGet, "/api/factory/status", "getFactoryStatus", "", "FactoryStatus", 200, false},
		{methodGet, "/api/llm-connections", "listConnections", "", "Connection", 200, true},
		{"post", "/api/llm-connections", "createConnection", "ConnectionConfig", "Connection", 201, false},
		{methodGet, "/api/llm-connections/{id}", "getConnection", "", "Connection", 200, false},
		{"patch", "/api/llm-connections/{id}", "updateConnection", "ConnectionPatch", "Connection", 200, false},
		{"post", "/api/llm-connections/{id}/test", "testConnection", "Probe", "ProbeResult", 200, false},
		{methodGet, "/api/products", "listProducts", "", "Product", 200, true},
		{"post", "/api/products", "createProduct", "ProductInput", "Product", 201, false},
		{methodGet, "/api/products/{id}", "getProduct", "", "Product", 200, false},
		{"post", "/api/products/{id}/generations", "startGeneration", "GenerationInput", "Run", 202, false},
		{methodGet, "/api/generations/{id}", "getGeneration", "", "Run", 200, false},
		{methodGet, "/api/products/{id}/passport", "getLatestPassport", "", "Revision", 200, false},
		{methodGet, "/api/products/{id}/passport-revisions", "listRevisions", "", "RevisionInfo", 200, true},
		{methodGet, "/api/passport-revisions/{id}", "getRevision", "", "Revision", 200, false},
		{methodGet, "/api/passport-revisions/{id}/evidence/{evidenceId}", "getEvidence", "", "EvidenceView", 200, false},
	} {
		output := map[string]any{"$ref": "#/components/schemas/" + route.output}
		if route.array {
			output = map[string]any{"type": "array", "items": output}
		}
		operation := map[string]any{"operationId": route.id, "responses": map[string]any{strconv.Itoa(route.status): map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{"schema": output}}}, "default": map[string]any{"description": "Request failed", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Problem"}}}}}}
		if route.input != "" {
			operation["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + route.input}}}}
		}
		parameters := make([]any, 0)
		for part := range strings.SplitSeq(route.path, "/") {
			if strings.HasPrefix(part, "{") {
				parameters = append(parameters, map[string]any{"name": strings.Trim(part, "{}"), "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
			}
		}
		if len(parameters) > 0 {
			operation["parameters"] = parameters
		}
		entry, ok := paths[route.path].(map[string]any)
		if !ok {
			entry = make(map[string]any)
			paths[route.path] = entry
		}
		entry[route.method] = operation
	}
	encoded, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode OpenAPI: %w", err)
	}
	return append(encoded, '\n'), nil
}
