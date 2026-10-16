package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//nolint:gocyclo // One proxy session verifies the tool allowlist, full exports and access errors.
func TestProxyOnlyForwardsReadOnlyTools(t *testing.T) {
	t.Parallel()
	upstream := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	fullText := strings.Repeat("Complete Markdown.\n", 15000) + "END_OF_PASSPORT"
	for _, name := range []string{"list_products", "export_passports", "write_product"} {
		upstream.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: name != "write_product"}}, func(_ context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if name == "list_products" {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "products"}}}, nil
			}
			var args struct {
				ProductIDs []string `json:"product_ids"`
			}
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return nil, fmt.Errorf("decode tool arguments: %w", err)
			}
			if len(args.ProductIDs) != 1 || args.ProductIDs[0] != "visible" {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "not found"}}}, nil
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fullText}}, StructuredContent: map[string]any{"filename": "passport.md", "markdown": fullText}}, nil
		})
	}
	connect := func(server *mcp.Server) *mcp.ClientSession {
		t.Helper()
		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		serverSession, err := server.Connect(t.Context(), serverTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = serverSession.Close() })
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(t.Context(), clientTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = session.Close() })
		return session
	}
	proxy, err := proxyServer(t.Context(), connect(upstream))
	if err != nil {
		t.Fatal(err)
	}
	client := connect(proxy)
	tools, err := client.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 2 {
		t.Fatalf("proxy tool count: %v", err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "write_product" || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatal("proxy exposed a write tool")
		}
	}
	products, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_products", Arguments: map[string]any{}})
	if err != nil || products.Content[0].(*mcp.TextContent).Text != "products" {
		t.Fatal("list tool was not forwarded")
	}
	export, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "export_passports", Arguments: map[string]any{"product_ids": []string{"visible"}}})
	if err != nil || export.IsError || export.Content[0].(*mcp.TextContent).Text != fullText {
		t.Fatal("full export was not forwarded")
	}
	structured, ok := export.StructuredContent.(map[string]any)
	if !ok || structured["markdown"] != fullText {
		t.Fatal("structured Markdown was lost")
	}
	denied, err := client.CallTool(t.Context(), &mcp.CallToolParams{Name: "export_passports", Arguments: map[string]any{"product_ids": []string{"hidden"}}})
	if err != nil || !denied.IsError {
		t.Fatal("upstream access error was lost")
	}
}
