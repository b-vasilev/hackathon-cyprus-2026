package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/rs/zerolog"
)

func seedMCPPassport(t *testing.T, portal *testPortal, productID, id string, p passport.Passport) {
	t.Helper()
	path := "products/" + productID + "/versions/1/" + id + "/passport.json"
	hash, err := portal.content.WriteJSON(path, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{"INSERT OR IGNORE INTO llm_connections(id,name,created_at) VALUES('mcp-connection','Fixture','created')", nil},
		{"INSERT OR IGNORE INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES('mcp-connection',1,X'01','created')", nil},
		{`INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at) VALUES(?,?,'mcp-connection',1,'model','succeeded','complete','{}','created')`, []any{id, productID}},
		{`INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_path,content_hash,created_at) VALUES(?,?,?,1,'1','{}',?,?,'2026-10-16T12:00:00Z')`, []any{id, productID, id, path, hash}},
	} {
		if _, checkErr := portal.database.ExecContext(t.Context(), statement.query, statement.args...); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
}

func mcpFixture(prose string) passport.Passport {
	return passport.Passport{
		Model: passport.Model{
			Title: "Product title", Tagline: "Product tagline", Summary: "Product purpose",
			Objects:   []passport.Object{{ID: "component", Name: "API | worker", Kind: "component", Summary: "Component responsibility", Technology: "Go", Status: "implemented", Inputs: []string{"Request input"}, Outputs: []string{"Response output"}, Contracts: []string{"Public contract"}, ClaimIDs: []string{"claim"}, ArchitectureRole: "runtime_unit", ParentID: "system"}},
			Relations: []passport.Relation{{ID: "connection", From: "component", To: "store", Label: "Writes data", Kind: "data", Interaction: "sync", Contract: "Database contract", ClaimIDs: []string{"claim"}}},
			Scenarios: []passport.Scenario{{ID: "flow", Title: "User flow", Summary: "Flow purpose", Inputs: []string{"Flow input"}, Outputs: []string{"Flow output"}, FailureModes: []string{"Flow failure"}, Steps: []passport.Step{{ID: "step", Title: "Handle request", Summary: "Step details", ObjectIDs: []string{"component"}, ClaimIDs: []string{"claim"}}}}},
			Claims:    []passport.Claim{{ID: "claim", Text: "A supported claim", Basis: "source_backed", EvidenceIDs: []string{"evidence"}}},
			Evidence:  []passport.Evidence{{ID: "evidence", DocumentID: "document", StartLine: 2, EndLine: 4}}, OpenQuestions: []string{"Unresolved ownership"},
		},
		Views: passport.Views{
			Document: &passport.Document{Version: 2, Chapters: []passport.Chapter{{ID: "overview", Title: "Overview", Question: "What does it do?", Blocks: []passport.DocumentBlock{
				{Kind: "markdown", Markdown: prose, ClaimIDs: []string{"claim"}},
				{Kind: "object_table", ObjectIDs: []string{"component"}, TableStyle: "technologies"},
				{Kind: "diagram_ref", ReferenceID: "diagram"}, {Kind: "scenario_ref", ReferenceID: "flow"},
			}}}},
			Diagrams: []passport.Diagram{{ID: "diagram", Title: "System diagram", Kind: "containers", Summary: "Diagram details", Question: "What connects?", ScopeObjectID: "system", NodeIDs: []string{"component"}, RelationIDs: []string{"connection"}}},
		},
	}
}

//nolint:gocyclo // One connected session checks account isolation, publication changes and revocation.
func TestMCPAccountAccessAndExports(t *testing.T) {
	t.Parallel()
	portal := newTestPortal(t)
	service := generation.New(portal.database, nil, nil, nil, zerolog.Nop(), portal.content)
	server := httptest.NewUnstartedServer(http.NotFoundHandler())
	publicURL := "http://" + server.Listener.Addr().String()
	portal.handler = New(portal.database, zerolog.Nop(), nil, &Factory{Access: &access.Store{DB: portal.database}, Service: service, PublicURL: publicURL, FrontendDir: t.TempDir()})
	server.Config.Handler = portal.handler
	server.Start()
	t.Cleanup(server.Close)
	prose := strings.Repeat("Full passport prose.\n", 15000) + "FINAL_PARAGRAPH"
	seedMCPPassport(t, portal, portal.productID, "published", mcpFixture(prose))
	for _, statement := range []string{
		`INSERT INTO source_snapshots(id,run_id,locator,revision,coverage_json) VALUES('snapshot','published','https://github.com/org/repo.git','commit123','{}')`,
		`INSERT INTO source_documents(id,snapshot_id,path,content_hash,content,line_count) VALUES('document','snapshot','src/file name.go','hash','',5)`,
	} {
		if _, checkErr := portal.database.ExecContext(t.Context(), statement); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
	portal.request(http.MethodPost, "/api/passport-revisions/published/publish", "", portal.admin, http.StatusOK)
	var hidden struct{ ID string }
	if checkErr := json.Unmarshal([]byte(portal.request(http.MethodPost, "/api/products", `{"name":"HIDDEN_PRODUCT","sources":[{"kind":"text","text":"PRIVATE_SOURCE_CONFIG"}]}`, portal.admin, http.StatusCreated)), &hidden); checkErr != nil {
		t.Fatal(checkErr)
	}
	legacy := passport.Passport{Model: passport.Model{Title: "Hidden title"}, Views: passport.Views{Sections: []passport.Section{{ID: "overview", Title: "Legacy overview", Summary: "LEGACY_CONTENT", Checklist: []string{"Legacy checklist"}}}}}
	seedMCPPassport(t, portal, hidden.ID, "hidden", legacy)
	portal.request(http.MethodPost, "/api/passport-revisions/hidden/publish", "", portal.admin, http.StatusOK)
	outsider := portal.register("mcp-outsider@example.com")
	issuedTokens := make(map[*mcp.ClientSession]string)
	rawMCP := func(token string, cookie *http.Cookie, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		portal.handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("MCP status=%d expected=%d", w.Code, want)
		}
		return w
	}
	unauthorized := rawMCP("", nil, "", http.StatusUnauthorized)
	if !strings.Contains(unauthorized.Header().Get("WWW-Authenticate"), publicURL+"/.well-known/oauth-protected-resource/mcp") {
		t.Fatal("missing OAuth discovery challenge")
	}
	rawMCP("", portal.admin, "", http.StatusUnauthorized)
	rawMCP(portal.admin.Value, nil, "", http.StatusUnauthorized)
	rawMCP("forged", nil, "https://untrusted.example", http.StatusForbidden)

	connect := func(cookie *http.Cookie) *mcp.ClientSession {
		t.Helper()
		handler, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
			DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{Metadata: &oauthex.ClientRegistrationMetadata{ClientName: "Test AI", RedirectURIs: []string{"http://127.0.0.1:54321/callback"}, GrantTypes: []string{"authorization_code", "refresh_token"}}},
			RedirectURL:                     "http://127.0.0.1:54321/callback",
			AuthorizationCodeFetcher: func(_ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
				u, err := url.Parse(args.URL)
				if err != nil {
					return nil, fmt.Errorf("parse authorization URL: %w", err)
				}
				portal.request(http.MethodGet, "/api/ai-connections/authorize?"+u.RawQuery, "", cookie, http.StatusOK)
				body, err := json.Marshal(map[string]any{"authorization": authorizationQuery(u.Query()), "allow": true})
				if err != nil {
					return nil, fmt.Errorf("encode consent: %w", err)
				}
				var redirect struct {
					URL string `json:"redirect_url"`
				}
				if checkErr := json.Unmarshal([]byte(portal.request(http.MethodPost, "/api/ai-connections/authorize", string(body), cookie, http.StatusOK)), &redirect); checkErr != nil {
					return nil, fmt.Errorf("decode consent redirect: %w", checkErr)
				}
				callback, err := url.Parse(redirect.URL)
				if err != nil {
					return nil, fmt.Errorf("parse consent redirect: %w", err)
				}
				return &auth.AuthorizationResult{Code: callback.Query().Get("code"), State: callback.Query().Get("state"), Iss: callback.Query().Get("iss")}, nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: publicURL + "/mcp", OAuthHandler: handler}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = session.Close() })
		source, err := handler.TokenSource(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		token, err := source.Token()
		if err != nil {
			t.Fatal(err)
		}
		issuedTokens[session] = token.AccessToken
		return session
	}
	learner, admin := connect(portal.learner), connect(portal.admin)
	call := func(client *mcp.ClientSession, name string, args any, wantError bool) *mcp.CallToolResult {
		t.Helper()
		result, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
		if (err != nil || result.IsError) != wantError {
			t.Fatalf("%s error=%v result=%+v", name, err, result)
		}
		return result
	}
	tools, err := learner.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 2 {
		t.Fatalf("tools=%+v err=%v", tools, err)
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("missing read-only tool contract: %+v", tool)
		}
	}
	decode := func(result *mcp.CallToolResult, out any) {
		t.Helper()
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if checkErr := json.Unmarshal(data, out); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
	var listed mcpProducts
	decode(call(learner, "list_products", map[string]any{}, false), &listed)
	if len(listed.Products) != 1 || listed.Products[0].ID != portal.productID || !listed.Products[0].HasPublishedPassport {
		t.Fatalf("unexpected visible products: %+v", listed)
	}
	var exports mcpExports
	decode(call(learner, "export_passports", map[string]any{}, false), &exports)
	if len(exports.Passports) != 1 || exports.Passports[0].Filename != "example-v1.md" {
		t.Fatal("export filename must use the product name")
	}
	md := exports.Passports[0].Markdown
	for _, want := range []string{prose, "Product title", "Product tagline", "API \\| worker (Go)", "Component responsibility", "Flow purpose", "Flow input", "Flow output", "Flow failure", "Step details"} {
		if !strings.Contains(md, want) {
			t.Errorf("Markdown is missing %q", want[:min(len(want), 80)])
		}
	}
	for _, unwanted := range []string{"**Product ID:**", "**Chapter ID:**", "**Question:**", "**Claims:**", "## Objects", "## Connections", "## Diagrams", "## Evidence", "## Open questions", "## Source snapshots", "## Product branding", "System diagram", "A supported claim", "Unresolved ownership"} {
		if strings.Contains(md, unwanted) {
			t.Errorf("reading export includes internal detail %q", unwanted)
		}
	}
	if strings.Contains(md, "PRIVATE_SOURCE_CONFIG") || strings.Contains(md, "HIDDEN_PRODUCT") {
		t.Fatal("export leaked product configuration or another product")
	}
	call(learner, "export_passports", map[string]any{"product_ids": []string{hidden.ID}}, true)
	call(learner, "export_passports", map[string]any{"product_ids": []string{portal.productID, "missing"}}, true)
	call(learner, "export_passports", map[string]any{"product_ids": "wrong-type"}, true)
	call(learner, "search_passports", map[string]any{}, true)
	decode(call(learner, "export_passports", map[string]any{"product_ids": []string{portal.productID, portal.productID}}, false), &exports)
	if len(exports.Passports) != 1 {
		t.Fatal("duplicate product generated duplicate files")
	}
	decode(call(admin, "export_passports", map[string]any{}, false), &exports)
	if len(exports.Passports) != 2 {
		t.Fatal("admin cannot export all published products")
	}
	var legacyFound bool
	for _, export := range exports.Passports {
		legacyFound = legacyFound || strings.Contains(export.Markdown, "LEGACY_CONTENT") && strings.Contains(export.Markdown, "Legacy checklist")
	}
	if !legacyFound {
		t.Fatal("legacy sections were omitted")
	}
	if _, err := portal.database.ExecContext(t.Context(), "UPDATE products SET name='Example' WHERE id=?", hidden.ID); err != nil {
		t.Fatal(err)
	}
	decode(call(admin, "export_passports", map[string]any{}, false), &exports)
	filenames := map[string]bool{}
	for _, export := range exports.Passports {
		filenames[export.Filename] = true
	}
	if !filenames["example-v1.md"] || !filenames["example-v1-2.md"] || len(filenames) != 2 {
		t.Fatal("products with the same name must export distinct readable filenames")
	}
	portal.request(http.MethodPut, "/api/passport-revisions/published/chapters/overview", `{"title":"Unpublished title","markdown":["UNPUBLISHED_EDIT"]}`, portal.lead, http.StatusOK)
	decode(call(admin, "export_passports", map[string]any{"product_ids": []string{portal.productID}}, false), &exports)
	if !strings.Contains(exports.Passports[0].Markdown, prose) || strings.Contains(exports.Passports[0].Markdown, "UNPUBLISHED_EDIT") {
		t.Fatal("MCP exported the working copy instead of the published copy")
	}
	portal.request(http.MethodGet, "/api/products/"+portal.productID+"/passport.md", "", portal.learner, http.StatusOK)
	portal.request(http.MethodGet, "/api/products/"+hidden.ID+"/passport.md", "", portal.learner, http.StatusNotFound)
	decode(call(connect(outsider), "export_passports", map[string]any{}, false), &exports)
	if len(exports.Passports) != 0 {
		t.Fatal("account without product access received passports")
	}
	var me access.Me
	if checkErr := json.Unmarshal([]byte(portal.request(http.MethodGet, "/api/auth/me", "", portal.learner, http.StatusOK)), &me); checkErr != nil {
		t.Fatal(checkErr)
	}
	portal.request(http.MethodDelete, fmt.Sprintf("/api/products/%s/members/%s", portal.productID, me.User.ID), "", portal.admin, http.StatusNoContent)
	decode(call(learner, "list_products", map[string]any{}, false), &listed)
	if len(listed.Products) != 0 {
		t.Fatal("membership revocation did not affect the existing MCP connection")
	}
	call(learner, "export_passports", map[string]any{"product_ids": []string{portal.productID}}, true)
	portal.request(http.MethodPost, "/api/passport-revisions/published/unpublish", "", portal.admin, http.StatusOK)
	decode(call(admin, "export_passports", map[string]any{}, false), &exports)
	if len(exports.Passports) != 1 || exports.Passports[0].ProductID != hidden.ID {
		t.Fatal("unpublished passport was still exported")
	}
	var settings MCPSettings
	if checkErr := json.Unmarshal([]byte(portal.request(http.MethodGet, "/api/ai-connections", "", portal.learner, http.StatusOK)), &settings); checkErr != nil || len(settings.Connections) != 1 {
		t.Fatal("missing learner app connection")
	}
	portal.request(http.MethodDelete, "/api/ai-connections/"+settings.Connections[0].ID, "", portal.admin, http.StatusNoContent)
	rawMCP(issuedTokens[learner], nil, "", http.StatusOK)
	portal.request(http.MethodDelete, "/api/ai-connections/"+settings.Connections[0].ID, "", portal.learner, http.StatusNoContent)
	rawMCP(issuedTokens[learner], nil, "", http.StatusUnauthorized)
	r := httptest.NewRequest(http.MethodGet, "/api/products", http.NoBody)
	r.Header.Set("Authorization", "Bearer "+issuedTokens[admin])
	w := httptest.NewRecorder()
	portal.handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("MCP credentials authenticated a portal route")
	}
	if _, checkErr := portal.database.ExecContext(t.Context(), "UPDATE passport_revisions SET content_hash=?,published_content_hash=? WHERE id='hidden'", strings.Repeat("0", 64), strings.Repeat("0", 64)); checkErr != nil {
		t.Fatal(checkErr)
	}
	failed := call(admin, "export_passports", map[string]any{}, true)
	if len(failed.Content) != 1 || strings.Contains(failed.Content[0].(*mcp.TextContent).Text, "LEGACY_CONTENT") {
		t.Fatal("failed batch returned partial passport content")
	}
}
