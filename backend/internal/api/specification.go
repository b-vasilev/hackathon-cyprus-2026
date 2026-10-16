package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learning"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/mcpauth"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/google/jsonschema-go/jsonschema"
)

// Specification builds OpenAPI from the same Go contracts used by handlers and LLM generation.
const (
	methodGet     = "get"
	methodPost    = "post"
	methodPut     = "put"
	methodDelete  = "delete"
	productSchema = "Product"
	schemaTypeKey = "type"
	schemaKey     = "schema"
)

func Specification() ([]byte, error) {
	var spec map[string]any
	if err := json.Unmarshal(openAPI, &spec); err != nil {
		return nil, fmt.Errorf("read base OpenAPI: %w", err)
	}
	types := map[string]func() (*jsonschema.Schema, error){
		"MCPSettings": passport.Schema[MCPSettings], "MCPConsentInput": passport.Schema[MCPConsentInput],
		"MCPClient": passport.Schema[mcpauth.Client], "MCPRedirect": passport.Schema[MCPRedirect],
		"GitHubConnection": passport.Schema[githubconnection.Status], "GitHubTokenInput": passport.Schema[githubconnection.TokenInput], "OAuthURL": passport.Schema[OAuthURL], "LearningState": passport.Schema[learning.State], "LearningPack": passport.Schema[learningpack.View],
		"AvailableModel": passport.Schema[llm.AvailableModel], "ModelDiscoveryInput": passport.Schema[ModelDiscoveryInput],
		"Connection": passport.Schema[llmgateway.Connection], "ConnectionConfig": passport.Schema[llmgateway.Config], "Model": passport.Schema[llmgateway.Model], "Parameter": passport.Schema[llmgateway.Parameter],
		"Product": passport.Schema[product.Product], "ProductInput": passport.Schema[product.Input], "IconInput": passport.Schema[product.IconInput], "SourceInput": passport.Schema[source.Input],
		"GenerationInput": passport.Schema[generation.Input], "Run": passport.Schema[generation.Run], "Revision": passport.Schema[generation.Revision], "RevisionInfo": passport.Schema[generation.RevisionInfo], "EvidenceView": passport.Schema[generation.EvidenceView],
		"Review": passport.Schema[generation.Review], "ReviewActivity": passport.Schema[generation.ReviewActivity], "CommentInput": passport.Schema[generation.CommentInput], "FlagInput": passport.Schema[generation.FlagInput], "ChapterEdit": passport.Schema[passport.ChapterEdit],
		"GenerationEvent": passport.Schema[generation.Event],
		"User":            passport.Schema[access.User], "Me": passport.Schema[access.Me], "Membership": passport.Schema[access.Membership], "Member": passport.Schema[access.Member], "Credentials": passport.Schema[access.Credentials], "MemberInput": passport.Schema[access.MemberInput], "RoleInput": passport.Schema[access.RoleInput],
		"ProductSharing": passport.Schema[access.Sharing],
		"Passport":       passport.Schema[passport.Passport], "ProductModel": passport.Schema[passport.Model], "ProductObject": passport.Schema[passport.Object], "Claim": passport.Schema[passport.Claim], "Evidence": passport.Schema[passport.Evidence], "Relation": passport.Schema[passport.Relation], "Scenario": passport.Schema[passport.Scenario], "Section": passport.Schema[passport.Section], "Diagram": passport.Schema[passport.Diagram],
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
		if name == "MCPSettings" {
			schema.Properties["connections"].Type = "array"
			schema.Properties["connections"].Types = nil
		}
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
		{methodPost, "/api/auth/register", "register", "Credentials", "Me", 201, false},
		{methodPost, "/api/auth/login", "login", "Credentials", "Me", 200, false},
		{methodPost, "/api/auth/logout", "logout", "", "", 204, false},
		{methodGet, "/api/auth/me", "getMe", "", "Me", 200, false},
		{methodGet, "/api/ai-connections", "getAIConnections", "", "MCPSettings", 200, false},
		{methodDelete, "/api/ai-connections/{id}", "disconnectAIApp", "", "", 204, false},
		{methodGet, "/api/ai-connections/authorize", "previewAIConsent", "", "MCPClient", 200, false},
		{methodPost, "/api/ai-connections/authorize", "authorizeAIApp", "MCPConsentInput", "MCPRedirect", 200, false},
		{methodGet, "/api/products/{id}/passport.md", "downloadPublishedPassport", "", "", 200, false},
		{methodGet, "/api/me/learning", "getLearningProgress", "", "LearningState", 200, false},
		{methodPut, "/api/me/learning", "saveLearningProgress", "LearningState", "LearningState", 200, false},
		{methodGet, "/api/users", "listUsers", "", "User", 200, true},
		{"patch", "/api/users/{id}", "setUserRole", "RoleInput", "User", 200, false},
		{methodDelete, "/api/users/{id}", "deleteUser", "", "", 204, false},
		{methodGet, "/api/products/{id}/members", "listMembers", "", "Member", 200, true},
		{methodGet, "/api/products/{id}/sharing", "getProductSharing", "", "ProductSharing", 200, false},
		{methodPut, "/api/products/{id}/sharing", "setProductSharing", "ProductSharing", "ProductSharing", 200, false},
		{methodPut, "/api/products/{id}/members", "setMember", "MemberInput", "Member", 200, false},
		{methodDelete, "/api/products/{id}/members/{userId}", "removeMember", "", "", 204, false},
		{methodGet, "/api/git-connection", "getGitHubConnection", "", "GitHubConnection", 200, false},
		{methodPut, "/api/git-connection", "connectGitHubToken", "GitHubTokenInput", "GitHubConnection", 200, false},
		{methodDelete, "/api/git-connection", "disconnectGitHub", "", "GitHubConnection", 200, false},
		{methodPost, "/api/git-connection/oauth/start", "startGitHubOAuth", "", "OAuthURL", 200, false},
		{methodGet, "/api/factory/status", "getFactoryStatus", "", "FactoryStatus", 200, false},
		{methodPost, "/api/llm-connections/models", "listConnectionModels", "ModelDiscoveryInput", "AvailableModel", 200, true},
		{methodGet, "/api/llm-connections", "listConnections", "", "Connection", 200, true},
		{methodPost, "/api/llm-connections", "createConnection", "ConnectionConfig", "Connection", 201, false},
		{methodGet, "/api/llm-connections/{id}", "getConnection", "", "Connection", 200, false},
		{"patch", "/api/llm-connections/{id}", "updateConnection", "ConnectionPatch", "Connection", 200, false},
		{methodDelete, "/api/llm-connections/{id}", "deleteConnection", "", "Connection", 200, false},
		{methodPost, "/api/llm-connections/{id}/test", "testConnection", "Probe", "ProbeResult", 200, false},
		{methodGet, "/api/products", "listProducts", "", "Product", 200, true},
		{methodPost, "/api/products", "createProduct", "ProductInput", productSchema, 201, false},
		{methodGet, "/api/products/{id}", "getProduct", "", productSchema, 200, false},
		{methodDelete, "/api/products/{id}", "deleteProduct", "", productSchema, 200, false},
		{methodPut, "/api/products/{id}/icon", "setProductIcon", "IconInput", productSchema, 200, false},
		{methodDelete, "/api/products/{id}/icon", "removeProductIcon", "", productSchema, 200, false},
		{methodPost, "/api/products/{id}/generations", "startGeneration", "GenerationInput", "Run", 202, false},
		{methodGet, "/api/generations/{id}", "getGeneration", "", "Run", 200, false},
		{methodPost, "/api/generations/{id}/cancel", "cancelGeneration", "", "Run", 200, false},
		{methodGet, "/api/generations/{id}/events", "streamGeneration", "", "Run", 200, false},
		{methodGet, "/api/products/{id}/passport", "getPublishedPassport", "", "Revision", 200, false},
		{methodGet, "/api/products/{id}/passport-revisions", "listRevisions", "", "RevisionInfo", 200, true},
		{methodGet, "/api/passport-revisions/{id}", "getRevision", "", "Revision", 200, false},
		{methodPost, "/api/passport-revisions/{id}/publish", "publishRevision", "", "Revision", 200, false},
		{methodPost, "/api/passport-revisions/{id}/unpublish", "unpublishRevision", "", "Revision", 200, false},
		{methodGet, "/api/passport-revisions/{id}/review", "getRevisionReview", "", "Review", 200, false},
		{methodPut, "/api/passport-revisions/{id}/flag", "flagRevision", "FlagInput", "Review", 200, false},
		{methodDelete, "/api/passport-revisions/{id}/flag", "resolveRevisionFlag", "", "Review", 200, false},
		{methodPost, "/api/passport-revisions/{id}/comments", "commentOnRevision", "CommentInput", "Review", 200, false},
		{methodPut, "/api/passport-revisions/{id}/chapters/{chapterId}", "editRevisionChapter", "ChapterEdit", "Revision", 200, false},
		{methodGet, "/api/passport-revisions/{id}/evidence/{evidenceId}", "getEvidence", "", "EvidenceView", 200, false},
		{methodGet, "/api/passport-revisions/{id}/learning-pack", "getLearningPack", "", "LearningPack", 200, false},
		{methodPost, "/api/passport-revisions/{id}/learning-pack", "generateLearningPack", "", "LearningPack", 200, false},
		{methodPost, "/api/passport-revisions/{id}/learning-pack/publish", "publishLearningPack", "", "LearningPack", 200, false},
		{methodPost, "/api/passport-revisions/{id}/learning-pack/unpublish", "unpublishLearningPack", "", "LearningPack", 200, false},
		{methodGet, "/api/products/{id}/learning-pack", "getPublishedLearningPack", "", "LearningPack", 200, false},
	} {
		output := map[string]any{"$ref": "#/components/schemas/" + route.output}
		if route.array {
			output = map[string]any{schemaTypeKey: "array", "items": output}
		}
		success := map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{schemaKey: output}}}
		if route.output == "" {
			success = map[string]any{"description": "No content"}
		}
		operation := map[string]any{"operationId": route.id, "responses": map[string]any{strconv.Itoa(route.status): success, "default": map[string]any{"description": "Request failed", "content": map[string]any{"application/json": map[string]any{schemaKey: map[string]any{"$ref": "#/components/schemas/Problem"}}}}}}
		if route.id == "downloadPublishedPassport" {
			operation["responses"].(map[string]any)["200"] = map[string]any{"description": "Complete published passport download", "content": map[string]any{"text/markdown": map[string]any{schemaKey: map[string]any{schemaTypeKey: "string"}}}}
		}
		if route.id == "streamGeneration" {
			operation["description"] = "SSE snapshot and update events contain a Run with events after Last-Event-ID. IDs are ordered and durable. Reconnects replay missed events. Terminal snapshots close the stream."
			operation["responses"].(map[string]any)["200"] = map[string]any{"description": "Generation event stream", "content": map[string]any{"text/event-stream": map[string]any{schemaKey: map[string]any{schemaTypeKey: "string"}}}}
		}
		if route.input != "" {
			operation["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{schemaKey: map[string]any{"$ref": "#/components/schemas/" + route.input}}}}
		}
		if parameters := operationParameters(route.id, route.path); len(parameters) > 0 {
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
	workers := state.Properties["parallel"].Properties["workers"]
	workers.Type, workers.Types = "array", nil
}

func operationParameters(id, path string) []any {
	parameters := make([]any, 0)
	if id == "previewAIConsent" {
		for _, name := range []string{"client_id", "redirect_uri", "response_type", "code_challenge", "code_challenge_method", "resource", "scope", "state"} {
			parameters = append(parameters, map[string]any{"name": name, "in": "query", "required": name != "scope" && name != "state", schemaKey: map[string]any{schemaTypeKey: "string"}})
		}
	}
	for part := range strings.SplitSeq(path, "/") {
		if strings.HasPrefix(part, "{") {
			parameters = append(parameters, map[string]any{"name": strings.Trim(part, "{}"), "in": "path", "required": true, schemaKey: map[string]any{schemaTypeKey: "string"}})
		}
	}
	if id == "streamGeneration" {
		parameters = append(parameters,
			map[string]any{"name": "Last-Event-ID", "in": "header", "required": false, schemaKey: map[string]any{schemaTypeKey: "integer", "minimum": 0}},
			map[string]any{"name": "after", "in": "query", "required": false, "description": "Replay cursor when creating a new EventSource. Last-Event-ID takes precedence on automatic reconnect.", schemaKey: map[string]any{schemaTypeKey: "integer", "minimum": 0}},
		)
	}
	return parameters
}
