package generation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
)

//nolint:gocyclo // One lifecycle checks live counts, partial failure and durable reload.
func TestResearchCountsAreVisibleWhileRunningAndSurviveFailure(t *testing.T) {
	binary, err := filepath.Abs("../../../frontend/node_modules/.bin/repomix")
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := os.Stat(binary); checkErr != nil {
		t.Skip("Repomix is not installed")
	}
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "progress.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	connections, err := llmgateway.NewStore(database, strings.Repeat("progress-key", 4))
	if err != nil {
		t.Fatal(err)
	}
	service := New(database, connections, nil, &source.Collector{MaxBytes: 10000, MaxFiles: 10}, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	var runID string
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		run, runErr := service.GetRun(ctx, runID)
		if runErr != nil {
			t.Error(runErr)
			http.Error(w, "missing run", http.StatusInternalServerError)
			return
		}
		if run.Status != "running" || run.Stage != "researching" || run.DocumentCount != 1 || run.InputTokens != (call-1)*100 || run.OutputTokens != (call-1)*50 {
			t.Errorf("incorrect in-progress counts: status=%s stage=%s documents=%d input=%d output=%d", run.Status, run.Stage, run.DocumentCount, run.InputTokens, run.OutputTokens)
		}
		if run.Research == nil || run.Research.Budget.HardLimit != 120 || len(run.Research.Areas) == 0 {
			t.Error("live research plan or hard budget is missing")
		}
		message := map[string]any{"role": "assistant", "content": "RESEARCH_INCOMPLETE"}
		reason := "stop"
		if call == 1 {
			message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": "list_files", "arguments": `{"pattern":"**","offset":0,"limit":10}`}}}}
			reason = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		if checkErr := json.NewEncoder(w).Encode(map[string]any{"id": "chat1", "object": "chat.completion", "choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": reason}}, "usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 50}}); checkErr != nil {
			t.Error(checkErr)
		}
	}))
	defer server.Close()
	policy, err := llmgateway.NewNetworkPolicy(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := connections.Save(ctx, "", llmgateway.Config{Name: "Fixture", Enabled: true, Protocol: "openai_chat_completions", EndpointURL: server.URL + "/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", Secret: "fixture-secret", TimeoutSeconds: 20, MaxConcurrent: 1, Models: []llmgateway.Model{{ID: "fixture", OutputMode: "json_schema", MaxOutputTokens: 6000}}})
	if err != nil {
		t.Fatal(err)
	}
	service.Research = &research.Runner{Policy: policy, Limiter: llmgateway.NewLimiter(8, 16, 2, 8), WorkDir: t.TempDir(), RepomixBin: binary}
	p, err := service.Products.Create(ctx, product.Input{Name: "Fixture", Sources: []source.Input{{Kind: "text", Text: "Payment service stores requests."}}})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.Start(ctx, p.ID, Input{ConnectionID: connection.ID, Model: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	runID = run.ID
	processErr := service.process(ctx, run)
	if processErr == nil || !strings.Contains(processErr.Error(), "repeatedly stopped without progress") {
		t.Fatalf("expected an explicit no-progress research failure: %v", processErr)
	}
	if checkErr := service.failRun(ctx, run.ID, "generation_failed", processErr.Error()); checkErr != nil {
		t.Fatal(checkErr)
	}
	reloaded := New(database, connections, nil, nil, zerolog.Nop(), service.Content)
	finished, err := reloaded.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 || finished.Status != "failed" || finished.DocumentCount != 1 || finished.InputTokens != 400 || finished.OutputTokens != 200 {
		t.Fatalf("counts were lost or duplicated after failure: calls=%d documents=%d input=%d output=%d", calls.Load(), finished.DocumentCount, finished.InputTokens, finished.OutputTokens)
	}
	if finished.Research == nil || finished.Research.Status != "partial" || finished.Research.Budget.Used != 4 || len(finished.Research.Budget.Decisions) < 2 {
		t.Fatal("final research plan and stopping reason did not survive reload")
	}
	for _, event := range finished.Events {
		if event.Type == "research_plan" && strings.HasPrefix(event.Message, "{") {
			t.Fatal("raw plan JSON leaked into the activity log")
		}
	}
}
