package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/google/jsonschema-go/jsonschema"
)

// Specification builds OpenAPI from the same Go contracts used by handlers and LLM generation.
const (
	methodGet     = "get"
	schemaTypeKey = "type"
	schemaKey     = "schema"
)

func Specification() ([]byte, error) {
	var spec map[string]any
	if err := json.Unmarshal(openAPI, &spec); err != nil {
		return nil, fmt.Errorf("read base OpenAPI: %w", err)
	}
	types := map[string]func() (*jsonschema.Schema, error){
		"GitHubConnection": passport.Schema[githubconnection.Status], "GitHubTokenInput": passport.Schema[githubconnection.TokenInput], "OAuthURL": passport.Schema[OAuthURL],
		"AvailableModel": passport.Schema[llm.AvailableModel], "ModelDiscoveryInput": passport.Schema[ModelDiscoveryInput],
		"Connection": passport.Schema[llmgateway.Connection], "ConnectionConfig": passport.Schema[llmgateway.Config], "Model": passport.Schema[llmgateway.Model], "Parameter": passport.Schema[llmgateway.Parameter],
		"Product": passport.Schema[product.Product], "ProductInput": passport.Schema[product.Input], "SourceInput": passport.Schema[source.Input],
		"GenerationInput": passport.Schema[generation.Input], "Run": passport.Schema[generation.Run], "Revision": passport.Schema[generation.Revision], "RevisionInfo": passport.Schema[generation.RevisionInfo], "EvidenceView": passport.Schema[generation.EvidenceView],
		"GenerationEvent": passport.Schema[generation.Event],
		"Passport":        passport.Schema[passport.Passport], "ProductModel": passport.Schema[passport.Model], "ProductObject": passport.Schema[passport.Object], "Claim": passport.Schema[passport.Claim], "Evidence": passport.Schema[passport.Evidence], "Relation": passport.Schema[passport.Relation], "Scenario": passport.Schema[passport.Scenario], "Section": passport.Schema[passport.Section], "Diagram": passport.Schema[passport.Diagram],
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
		if name == "Run" {
			// GetRun always returns an empty array when no events exist.
			schema.Properties["events"].Type = "array"
			schema.Properties["events"].Types = nil
			researchProgressSchema(schema.Properties["research"])
		}
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
		{methodGet, "/api/git-connection", "getGitHubConnection", "", "GitHubConnection", 200, false},
		{"put", "/api/git-connection", "connectGitHubToken", "GitHubTokenInput", "GitHubConnection", 200, false},
		{"delete", "/api/git-connection", "disconnectGitHub", "", "GitHubConnection", 200, false},
		{"post", "/api/git-connection/oauth/start", "startGitHubOAuth", "", "OAuthURL", 200, false},
		{methodGet, "/api/factory/status", "getFactoryStatus", "", "FactoryStatus", 200, false},
		{"post", "/api/llm-connections/models", "listConnectionModels", "ModelDiscoveryInput", "AvailableModel", 200, true},
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
		{"post", "/api/generations/{id}/cancel", "cancelGeneration", "", "Run", 200, false},
		{methodGet, "/api/generations/{id}/events", "streamGeneration", "", "Run", 200, false},
		{methodGet, "/api/products/{id}/passport", "getLatestPassport", "", "Revision", 200, false},
		{methodGet, "/api/products/{id}/passport-revisions", "listRevisions", "", "RevisionInfo", 200, true},
		{methodGet, "/api/passport-revisions/{id}", "getRevision", "", "Revision", 200, false},
		{methodGet, "/api/passport-revisions/{id}/evidence/{evidenceId}", "getEvidence", "", "EvidenceView", 200, false},
	} {
		output := map[string]any{"$ref": "#/components/schemas/" + route.output}
		if route.array {
			output = map[string]any{schemaTypeKey: "array", "items": output}
		}
		operation := map[string]any{"operationId": route.id, "responses": map[string]any{strconv.Itoa(route.status): map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{schemaKey: output}}}, "default": map[string]any{"description": "Request failed", "content": map[string]any{"application/json": map[string]any{schemaKey: map[string]any{"$ref": "#/components/schemas/Problem"}}}}}}
		if route.id == "streamGeneration" {
			operation["description"] = "SSE snapshot and update events contain a Run with events after Last-Event-ID. IDs are ordered and durable. Reconnects replay missed events. Terminal snapshots close the stream."
			operation["responses"].(map[string]any)["200"] = map[string]any{"description": "Generation event stream", "content": map[string]any{"text/event-stream": map[string]any{schemaKey: map[string]any{schemaTypeKey: "string"}}}}
		}
		if route.input != "" {
			operation["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{schemaKey: map[string]any{"$ref": "#/components/schemas/" + route.input}}}}
		}
		parameters := make([]any, 0)
		for part := range strings.SplitSeq(route.path, "/") {
			if strings.HasPrefix(part, "{") {
				parameters = append(parameters, map[string]any{"name": strings.Trim(part, "{}"), "in": "path", "required": true, schemaKey: map[string]any{schemaTypeKey: "string"}})
			}
		}
		if len(parameters) > 0 {
			operation["parameters"] = parameters
		}
		if route.id == "streamGeneration" {
			operation["parameters"] = append(parameters,
				map[string]any{"name": "Last-Event-ID", "in": "header", "required": false, schemaKey: map[string]any{schemaTypeKey: "integer", "minimum": 0}},
				map[string]any{"name": "after", "in": "query", "required": false, "description": "Replay cursor when creating a new EventSource. Last-Event-ID takes precedence on automatic reconnect.", schemaKey: map[string]any{schemaTypeKey: "integer", "minimum": 0}},
			)
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

// The progress builder always initializes these collections and omits a missing
// state. Keep the generated client contract as precise as the returned JSON.
func researchProgressSchema(state *jsonschema.Schema) {
	state.Type, state.Types = "object", nil
	areas := state.Properties["areas"]
	questions := areas.Items.Properties["questions"]
	for _, field := range []*jsonschema.Schema{areas, questions, questions.Items.Properties["evidence"], state.Properties["budget"].Properties["decisions"]} {
		field.Type, field.Types = "array", nil
	}
	areas.Items.Properties["status"].Enum = []any{"not_started", "investigating", "checked", "blocked", "excluded"}
	questions.Items.Properties["status"].Enum = []any{"pending", "answered", "not_applicable", "blocked", "excluded"}
}
