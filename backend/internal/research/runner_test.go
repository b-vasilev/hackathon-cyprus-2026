package research

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// This exercises the actual Fantasy loop and Repomix MCP process for all gateway protocols.
//
//nolint:gocyclo // The three wire formats exercise one identical research lifecycle.
func TestResearchProtocolsKeepGatewayAndSourceAnchors(t *testing.T) {
	binary, err := filepath.Abs("../../../frontend/node_modules/.bin/repomix")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skip("Repomix is not installed")
	}
	for _, protocol := range []string{"openai_responses", "openai_chat_completions", "anthropic_messages"} {
		t.Run(protocol, func(t *testing.T) {
			var calls atomic.Int64
			args := map[string]any{"reads": []map[string]any{{"path": "repo1/main.go", "start_line": 1, "end_line": 2}, {"path": "repo2/runner.cs", "start_line": 1, "end_line": 2}}}
			argsJSON, _ := json.Marshal(args)
			notes := "Go invokes the converter. [AAAAAAAAAAAAAAAAAAAAAAAAAA:L1-L2]\nC# runs the stage. [BBBBBBBBBBBBBBBBBBBBBBBBBB:L1-L2]\nRESEARCH_COMPLETE"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer fixture-secret" {
					t.Errorf("gateway credential was not applied: %q", got)
				}
				if !strings.HasPrefix(r.URL.Path, "/tenant/v1/") {
					t.Errorf("gateway base path lost: %s", r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["model"] != "test-model" {
					t.Errorf("model changed: %v", body["model"])
				}
				first := calls.Add(1) == 1
				w.Header().Set("Content-Type", "application/json")
				var value any
				switch protocol {
				case "openai_responses":
					output := []any{map[string]any{"type": "message", "id": "message1", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": notes, "annotations": []any{}}}}}
					if first {
						output = []any{map[string]any{"type": "function_call", "id": "fc1", "call_id": "call1", "name": "read_files", "arguments": string(argsJSON), "status": "completed"}}
					}
					value = map[string]any{"id": "resp1", "object": "response", "status": "completed", "output": output, "usage": map[string]int{"input_tokens": 100, "output_tokens": 50}}
				case "openai_chat_completions":
					message := map[string]any{"role": "assistant", "content": notes}
					reason := "stop"
					if first {
						message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": "read_files", "arguments": string(argsJSON)}}}}
						reason = "tool_calls"
					}
					value = map[string]any{"id": "chat1", "object": "chat.completion", "choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": reason}}, "usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 50}}
				case "anthropic_messages":
					blocks := []any{map[string]any{"type": "text", "text": notes}}
					reason := "end_turn"
					if first {
						blocks = []any{map[string]any{"type": "tool_use", "id": "call1", "name": "read_files", "input": args}}
						reason = "tool_use"
					}
					value = map[string]any{"id": "msg1", "type": "message", "role": "assistant", "model": "test-model", "content": blocks, "stop_reason": reason, "usage": map[string]int{"input_tokens": 100, "output_tokens": 50}}
				}
				if err := json.NewEncoder(w).Encode(value); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			policy, err := llmgateway.NewNetworkPolicy(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			config := llmgateway.Config{Protocol: protocol, EndpointURL: server.URL + "/tenant/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Secret: "fixture-secret", TimeoutSeconds: 20, MaxConcurrent: 1, Models: []llmgateway.Model{{ID: "test-model", OutputMode: "json_schema", MaxOutputTokens: 6000}}}
			collection := source.Collection{Snapshots: []source.Snapshot{{ID: "repo1", Locator: "fixture-one", Coverage: source.Coverage{FileCount: 2}}, {ID: "repo2", Locator: "fixture-two", Coverage: source.Coverage{FileCount: 1}}}, Documents: []source.Document{{ID: "AAAAAAAAAAAAAAAAAAAAAAAAAA", SnapshotID: "repo1", Path: "main.go", Content: "package main\n// invokes converter", LineCount: 2}, {ID: "BBBBBBBBBBBBBBBBBBBBBBBBBB", SnapshotID: "repo2", Path: "runner.cs", Content: "// stage runner\nclass Runner {}", LineCount: 2}, {ID: "CCCCCCCCCCCCCCCCCCCCCCCCCC", SnapshotID: "repo1", Path: "unused.go", Content: "package unused", LineCount: 1}}}
			runner := Runner{Policy: policy, Limiter: llmgateway.NewLimiter(8, 16, 2, 8), WorkDir: t.TempDir(), RepomixBin: binary}
			result, err := runner.Run(t.Context(), config, "test-model", collection, func(string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 || result.MCPReads != 2 || len(result.Collection.Documents) != 2 || result.Collection.Snapshots[0].Coverage.ReadFiles != 1 {
				t.Fatalf("unexpected research coverage: calls=%d reads=%d documents=%d", calls.Load(), result.MCPReads, len(result.Collection.Documents))
			}
			if !strings.Contains(result.Context, "1: package main") || !strings.Contains(result.Context, "1: // stage runner") {
				t.Fatal("original source evidence is missing")
			}
		})
	}
}

func TestEvidenceCannotCiteUnreadLines(t *testing.T) {
	doc := source.Document{ID: "AAAAAAAAAAAAAAAAAAAAAAAAAA", Content: "one\ntwo\nthree", LineCount: 3}
	w := workspace{documents: map[string]source.Document{"a": doc}, reads: map[string][]source.LineRange{doc.ID: {{StartLine: 1, EndLine: 2}}}}
	if _, err := w.evidenceContext("[AAAAAAAAAAAAAAAAAAAAAAAAAA:L1-L3]"); err == nil {
		t.Fatal("accepted unread source lines")
	}
}
