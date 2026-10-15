package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

type repairingGenerator struct {
	fixture     fixtureGenerator
	retried     atomic.Bool
	partRetried atomic.Bool
}

func (g *repairingGenerator) Generate(ctx context.Context, config llmgateway.Config, request llm.Request) (llm.Result, error) {
	if strings.HasPrefix(request.Input, "Analyze") && !g.retried.Swap(true) {
		return llm.Result{JSON: []byte(`{}`)}, nil
	}
	result, err := g.fixture.Generate(ctx, config, request)
	if err != nil {
		return result, fmt.Errorf("generate event fixture: %w", err)
	}
	if strings.HasPrefix(request.Input, "Generate ONLY") && !g.partRetried.Swap(true) {
		var fragment compositionFragment
		if err := json.Unmarshal(result.JSON, &fragment); err != nil {
			return result, fmt.Errorf("decode repair fixture fragment: %w", err)
		}
		fragment.Block.Markdown = "<script>invalid</script>"
		data, err := json.Marshal(fragment)
		if err != nil {
			return result, fmt.Errorf("encode repair fixture fragment: %w", err)
		}
		result.JSON = data
	}
	return result, nil
}

//nolint:gocyclo // Integration test checks retries, repair, reload and interruption as one durable lifecycle.
func TestRunEventsSurviveReloadAndRecordPartRepair(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	connections, err := llmgateway.NewStore(database, strings.Repeat("event-master", 4))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := connections.Save(ctx, "", llmgateway.Config{Name: "Fixture", Enabled: true, Protocol: "openai_responses", EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", Secret: "original-key", Models: []llmgateway.Model{{ID: "fixture", OutputMode: "json_schema", MaxOutputTokens: 12000}}, TimeoutSeconds: 30, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	service := New(database, connections, &repairingGenerator{}, &source.Collector{MaxBytes: 300000, MaxFiles: 120}, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	p, err := service.Products.Create(ctx, product.Input{Name: "Fixture", Sources: []source.Input{{Kind: "text", Text: "Payment service stores requests."}}})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{ConnectionID: connection.ID, Model: "fixture"}
	run, err := service.Start(ctx, p.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Events) != 1 || run.Events[0].Type != "run_queued" || run.StartedAt != "" {
		t.Fatalf("incorrect queued snapshot: %+v", run)
	}
	if checkErr := service.process(ctx, run); checkErr != nil {
		t.Fatal(checkErr)
	}
	// A fresh service reads events from SQLite without any in-memory state.
	reloaded := New(database, connections, nil, nil, zerolog.Nop(), service.Content)
	finished, err := reloaded.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != "succeeded" || finished.StartedAt == "" || finished.StageStartedAt == "" || finished.FinishedAt == "" {
		t.Fatalf("missing final state or timings: %+v", finished)
	}
	seen := make(map[string]bool)
	previous := int64(0)
	for _, event := range finished.Events {
		if event.ID <= previous || event.CreatedAt == "" {
			t.Fatal("events are not ordered and timestamped")
		}
		previous = event.ID
		seen[event.Type+":"+event.Stage] = true
	}
	for _, expected := range []string{"sources_collected:collecting", "stage_skipped:selecting_sources", "model_call_started:analyzing", "model_call_completed:analyzing", "retry_started:analyzing", "retry_started:composing", "composition_progress:composing", "stage_completed:composing", "stage_completed:validating", "run_succeeded:complete"} {
		if !seen[expected] {
			t.Errorf("missing event %s", expected)
		}
	}
	if finished.Composition == nil || len(finished.Composition.Tasks) == 0 {
		t.Fatal("composition tasks were not restored after reload")
	}
	interrupted, err := service.Start(ctx, p.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if checkErr := service.stage(ctx, interrupted.ID, "analyzing"); checkErr != nil {
		t.Fatal(checkErr)
	}
	if checkErr := reloaded.recoverRuns(ctx); checkErr != nil {
		t.Fatal(checkErr)
	}
	failed, err := reloaded.GetRun(ctx, interrupted.ID)
	if err != nil || failed.Status != "failed" || failed.ErrorCode != "interrupted" || failed.Events[len(failed.Events)-1].Type != "run_failed" {
		t.Fatalf("interrupted run lost its failure event: %+v, %v", failed, err)
	}
}
