package generation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
	"maragu.dev/goqite"
)

type retryCollector struct {
	calls atomic.Int32
	fail  atomic.Bool
}

func (c *retryCollector) Collect(ctx context.Context, inputs []source.Input) (source.Collection, error) {
	c.calls.Add(1)
	if c.fail.Load() {
		return source.Collection{}, errors.New("fixture collection unavailable")
	}
	collector := source.Collector{MaxBytes: 300000, MaxFiles: 120}
	collection, err := collector.Collect(ctx, inputs)
	if err != nil {
		return collection, fmt.Errorf("collect retry fixture: %w", err)
	}
	return collection, nil
}

type retryFixture struct {
	service   *Service
	generator *fixtureGenerator
	collector *retryCollector
	run       Run
	config    llmgateway.Config
}

func newRetryFixture(t *testing.T) retryFixture {
	t.Helper()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "retry.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if migrateErr := db.RunMigrations(database); migrateErr != nil {
		t.Fatal(migrateErr)
	}
	connections, err := llmgateway.NewStore(database, strings.Repeat("retry-master", 4))
	if err != nil {
		t.Fatal(err)
	}
	config := llmgateway.Config{Name: "Retry fixture", Enabled: true, Protocol: "openai_responses", EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", Secret: "original-key", Models: []llmgateway.Model{{ID: "fixture", OutputMode: "json_schema", MaxOutputTokens: 12000}}, TimeoutSeconds: 30, MaxConcurrent: 1}
	connection, err := connections.Save(t.Context(), "", config)
	if err != nil {
		t.Fatal(err)
	}
	generator, collector := &fixtureGenerator{}, &retryCollector{}
	service := New(database, connections, generator, collector, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	p, err := service.Products.Create(t.Context(), product.Input{Name: "Payment service", Sources: []source.Input{{Kind: "text", Text: "Payment service stores requests."}}})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.Start(t.Context(), p.ID, Input{ConnectionID: connection.ID, Model: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	return retryFixture{service: service, generator: generator, collector: collector, run: run, config: config}
}

func (f retryFixture) input() Input {
	return Input{ConnectionID: f.run.ConnectionID, Model: f.run.Model, ResumeRunID: f.run.ID}
}

func (f *retryFixture) failRun(t *testing.T) {
	t.Helper()
	f.generator.fail.Store(true)
	err := f.service.process(t.Context(), f.run)
	if err == nil {
		t.Fatal("fixture should fail")
	}
	if saveErr := f.service.failRun(t.Context(), f.run.ID, "provider_error", err.Error()); saveErr != nil {
		t.Fatal(saveErr)
	}
	f.run, err = f.service.GetRun(t.Context(), f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func retryQueueCount(t *testing.T, service *Service, runID string) int {
	t.Helper()
	var count int
	if checkErr := service.DB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM goqite WHERE queue='passport' AND body=?", []byte(runID)).Scan(&count); checkErr != nil {
		t.Fatal(checkErr)
	}
	return count
}

//nolint:gocyclo // This lifecycle checks frozen inputs, queue replacement, successful retry and stale failure protection together.
func TestRetryPreservesFrozenSourcesConnectionAndHistory(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.failRun(t)
	ctx := t.Context()
	var manifest string
	if checkErr := f.service.DB.QueryRowContext(ctx, "SELECT input_json FROM generation_runs WHERE id=?", f.run.ID).Scan(&manifest); checkErr != nil {
		t.Fatal(checkErr)
	}
	saved, err := f.service.loadSavedSources(ctx, f.run.ID)
	if err != nil || len(saved.Documents) != 1 {
		t.Fatal("missing frozen source", err)
	}
	brandPath := filepath.Join(f.service.Content.Dir, "runs", f.run.ID, "product-brand.json")
	brandBefore, err := os.ReadFile(brandPath)
	if err != nil {
		t.Fatal(err)
	}
	// Editing the product and latest connection must not change this run's inputs.
	if _, updateErr := f.service.DB.ExecContext(ctx, "UPDATE products SET sources_json=? WHERE id=?", `{"sources":[{"kind":"text","text":"Changed product"}]}`, f.run.ProductID); updateErr != nil {
		t.Fatal(updateErr)
	}
	f.config.Secret = "rotated-key"
	f.config.Models[0].ID = "new-model"
	if _, saveErr := f.service.Connections.Save(ctx, f.run.ConnectionID, f.config); saveErr != nil {
		t.Fatal(saveErr)
	}
	if sendErr := f.service.queue.Send(ctx, goqite.Message{Body: []byte(f.run.ID)}); sendErr != nil {
		t.Fatal(sendErr)
	}
	resumed, err := f.service.Start(ctx, f.run.ProductID, f.input())
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ID != f.run.ID || resumed.ConnectionRevision != f.run.ConnectionRevision || resumed.Status != "queued" || resumed.ErrorCode != "" || resumed.ErrorMessage != "" || resumed.FinishedAt != "" || resumed.CreatedAt != f.run.CreatedAt || len(resumed.Events) != len(f.run.Events)+1 {
		t.Fatalf("retry changed identity or lost/reset incorrect state: %+v", resumed)
	}
	if resumed.Events[len(resumed.Events)-1].Type != "run_resumed" || retryQueueCount(t, f.service, f.run.ID) != 1 {
		t.Fatal("retry must append one event and replace stale deliveries with one message")
	}
	var after string
	if checkErr := f.service.DB.QueryRowContext(ctx, "SELECT input_json FROM generation_runs WHERE id=?", f.run.ID).Scan(&after); checkErr != nil || after != manifest {
		t.Fatal("frozen source manifest changed", checkErr)
	}
	f.generator.fail.Store(false)
	if checkErr := f.service.process(ctx, resumed); checkErr != nil {
		t.Fatal(checkErr)
	}
	if f.collector.calls.Load() != 1 {
		t.Fatal("retry collected sources again")
	}
	restored, err := f.service.loadSavedSources(ctx, f.run.ID)
	if err != nil || len(restored.Documents) != 1 || restored.Documents[0] != saved.Documents[0] || restored.Snapshots[0].Revision != saved.Snapshots[0].Revision {
		t.Fatal("retry changed saved sources or revision", err)
	}
	brandAfter, err := os.ReadFile(brandPath)
	if err != nil || !bytes.Equal(brandAfter, brandBefore) {
		t.Fatal("retry changed immutable branding", err)
	}
	history, err := f.service.History(ctx, f.run.ProductID)
	if err != nil || len(history) != 1 {
		t.Fatal("retry did not save exactly one revision", err)
	}
	if _, checkErr := f.service.Start(ctx, f.run.ProductID, f.input()); !errors.Is(checkErr, llmgateway.ErrInvalid) {
		t.Fatal("succeeded generation accepted a retry", checkErr)
	}
	if checkErr := f.service.process(ctx, resumed); !errors.Is(checkErr, llmgateway.ErrInvalid) {
		t.Fatal("stale processing attempt was accepted", checkErr)
	}
	if checkErr := f.service.failRun(ctx, f.run.ID, "stale", "old worker failed"); checkErr != nil {
		t.Fatal(checkErr)
	}
	finished, err := f.service.GetRun(ctx, f.run.ID)
	if err != nil || finished.Status != "succeeded" || finished.ErrorCode != "" {
		t.Fatal("stale failure overwrote success", err)
	}
}

func TestRetryValidationAndActiveConflict(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"product", "connection", "model", "active", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			f := newRetryFixture(t)
			f.failRun(t)
			input, productID := f.input(), f.run.ProductID
			switch kind {
			case "product":
				productID = "different-product"
			case "connection":
				input.ConnectionID = "different-connection"
			case "model":
				input.Model = "different-model"
			case "active":
				if _, checkErr := f.service.Start(t.Context(), productID, Input{ConnectionID: f.run.ConnectionID, Model: f.run.Model}); checkErr != nil {
					t.Fatal(checkErr)
				}
			case "disabled":
				f.config.Enabled = false
				if _, checkErr := f.service.Connections.Save(t.Context(), f.run.ConnectionID, f.config); checkErr != nil {
					t.Fatal(checkErr)
				}
			}
			_, err := f.service.Start(t.Context(), productID, input)
			if err == nil || (kind == "active" && !errors.Is(err, ErrActive)) {
				t.Fatal("invalid retry was accepted or misclassified", err)
			}
			after, err := f.service.GetRun(t.Context(), f.run.ID)
			if err != nil || after.Status != "failed" || len(after.Events) != len(f.run.Events) {
				t.Fatal("rejected retry mutated the run", err)
			}
		})
	}
}

func TestRetryConcurrentQueueAndInterruptedClaim(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.failRun(t)
	if checkErr := f.service.process(t.Context(), f.run); !errors.Is(checkErr, llmgateway.ErrInvalid) {
		t.Fatal("failed generation was silently restarted", checkErr)
	}
	var accepted atomic.Int32
	var wait sync.WaitGroup
	for range 2 {
		wait.Go(func() {
			if _, checkErr := f.service.Start(t.Context(), f.run.ProductID, f.input()); checkErr == nil {
				accepted.Add(1)
			}
		})
	}
	wait.Wait()
	if accepted.Load() != 1 || retryQueueCount(t, f.service, f.run.ID) != 1 {
		t.Fatal("concurrent retry produced multiple accepted deliveries")
	}
	claimed, err := f.service.claimRun(t.Context(), f.run.ID)
	if err != nil || !claimed {
		t.Fatal("queued retry could not be claimed", err)
	}
	claimed, err = f.service.claimRun(t.Context(), f.run.ID)
	if err != nil || claimed {
		t.Fatal("duplicate message claimed a running generation", err)
	}
	if recoverErr := f.service.recoverRuns(t.Context()); recoverErr != nil {
		t.Fatal(recoverErr)
	}
	resumed, err := f.service.Start(t.Context(), f.run.ProductID, f.input())
	if err != nil || resumed.ID != f.run.ID || retryQueueCount(t, f.service, f.run.ID) != 1 {
		t.Fatal("interrupted retry could not be safely requeued", err)
	}
	resumes := 0
	for _, event := range resumed.Events {
		if event.Type == "run_resumed" {
			resumes++
		}
	}
	if resumes != 2 {
		t.Fatal("research attempt numbering lost a resume event")
	}
}

func TestRetryRejectsCorruptSavedSourcesWithoutRecollection(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.failRun(t)
	saved, err := f.service.loadSavedSources(t.Context(), f.run.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.service.Content.Dir, "runs", f.run.ID, "sources", saved.Documents[0].ID+".txt")
	if writeErr := os.WriteFile(path, []byte("corrupted"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	resumed, err := f.service.Start(t.Context(), f.run.ProductID, f.input())
	if err != nil {
		t.Fatal(err)
	}
	f.generator.fail.Store(false)
	if checkErr := f.service.process(t.Context(), resumed); checkErr == nil || !strings.Contains(checkErr.Error(), "checksum mismatch") {
		t.Fatal("corrupt source was accepted", checkErr)
	}
	if f.collector.calls.Load() != 1 {
		t.Fatal("corrupt source silently triggered a new collection")
	}
}

func TestRetryBeforeCollectionUsesOriginalManifest(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.collector.fail.Store(true)
	f.failRun(t)
	if f.run.DocumentCount != 0 {
		t.Fatal("fixture should not have saved sources")
	}
	if _, checkErr := f.service.DB.ExecContext(t.Context(), "UPDATE products SET sources_json=? WHERE id=?", `{"sources":[{"kind":"text","text":"Issue tracker stores requests."}]}`, f.run.ProductID); checkErr != nil {
		t.Fatal(checkErr)
	}
	f.collector.fail.Store(false)
	f.generator.fail.Store(false)
	resumed, err := f.service.Start(t.Context(), f.run.ProductID, f.input())
	if err != nil {
		t.Fatal(err)
	}
	if checkErr := f.service.process(t.Context(), resumed); checkErr != nil {
		t.Fatal(checkErr)
	}
	if f.collector.calls.Load() != 2 {
		t.Fatal("collection was not retried after a collection failure")
	}
	saved, loadErr := f.service.loadSavedSources(t.Context(), f.run.ID)
	if loadErr != nil || len(saved.Documents) != 1 || saved.Documents[0].Content != "Payment service stores requests." {
		t.Fatal("collection retry used edited product inputs instead of its original manifest", loadErr)
	}
}

type timeoutCheckingGenerator struct {
	expected llmgateway.Config
	timeout  int
	fixture  fixtureGenerator
	calls    atomic.Int32
}

func (g *timeoutCheckingGenerator) Generate(ctx context.Context, config llmgateway.Config, request llm.Request) (llm.Result, error) {
	expected := g.expected
	expected.TimeoutSeconds = g.timeout
	if !reflect.DeepEqual(config, expected) || request.Model != expected.Models[0].ID {
		return llm.Result{}, errors.New("request timeout override changed frozen connection or model settings")
	}
	g.calls.Add(1)
	return g.fixture.Generate(ctx, config, request)
}

func TestRetryRequestTimeoutPreservesFrozenConfiguration(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.failRun(t)
	ctx := t.Context()
	original, err := f.service.Connections.Load(ctx, f.run.ConnectionID, f.run.ConnectionRevision)
	if err != nil {
		t.Fatal(err)
	}
	// Latest connection edits must not affect this frozen run.
	f.config.Secret = "different-secret"
	f.config.EndpointURL = "https://different.example/v2"
	f.config.TimeoutSeconds = 55
	f.config.Models[0].ID = "different-model"
	f.config.Models[0].MaxOutputTokens = 14000
	if _, checkErr := f.service.Connections.Save(ctx, f.run.ConnectionID, f.config); checkErr != nil {
		t.Fatal(checkErr)
	}
	input := f.input()
	input.RequestTimeoutSeconds = 600
	resumed, err := f.service.Start(ctx, f.run.ProductID, input)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.RequestTimeoutSeconds != 600 || resumed.ConnectionRevision != f.run.ConnectionRevision || resumed.Model != f.run.Model {
		t.Fatal("timeout retry changed frozen model identity or lost the override")
	}
	if !strings.Contains(resumed.Events[len(resumed.Events)-1].Message, "30 -> 600 seconds") {
		t.Fatal("retry event omitted the old and new effective timeouts")
	}
	generator := &timeoutCheckingGenerator{expected: original, timeout: 600}
	reloaded := New(f.service.DB, f.service.Connections, generator, f.collector, zerolog.Nop(), f.service.Content)
	persisted, err := reloaded.GetRun(ctx, f.run.ID)
	if err != nil || persisted.RequestTimeoutSeconds != 600 {
		t.Fatal("timeout override did not survive service reload", err)
	}
	// process restores execution settings from storage, not a stale caller copy.
	persisted.RequestTimeoutSeconds = 20
	if checkErr := reloaded.process(ctx, persisted); checkErr != nil {
		t.Fatal(checkErr)
	}
	if generator.calls.Load() <= 2 || f.collector.calls.Load() != 1 {
		t.Fatal("retry repeated collection or did not apply the timeout to the composition tasks")
	}
	unchanged, err := reloaded.Connections.Load(ctx, f.run.ConnectionID, f.run.ConnectionRevision)
	if err != nil || !reflect.DeepEqual(unchanged, original) {
		t.Fatal("generation changed the stored frozen connection", err)
	}
}

func TestInitialRequestTimeoutDefaultsAndBounds(t *testing.T) {
	t.Parallel()
	for _, seconds := range []int{0, 20, 900} {
		t.Run(strconv.Itoa(seconds), func(t *testing.T) {
			t.Parallel()
			f := newRetryFixture(t)
			if checkErr := f.service.failRun(t.Context(), f.run.ID, "fixture", "retire the queued fixture"); checkErr != nil {
				t.Fatal(checkErr)
			}
			original, err := f.service.Connections.Load(t.Context(), f.run.ConnectionID, f.run.ConnectionRevision)
			if err != nil {
				t.Fatal(err)
			}
			timeout := seconds
			if timeout == 0 {
				timeout = original.TimeoutSeconds
			}
			generator := &timeoutCheckingGenerator{expected: original, timeout: timeout}
			f.service.LLM = generator
			run, err := f.service.Start(t.Context(), f.run.ProductID, Input{ConnectionID: f.run.ConnectionID, Model: f.run.Model, RequestTimeoutSeconds: seconds})
			if err != nil || run.RequestTimeoutSeconds != seconds {
				t.Fatal("initial execution timeout was not saved", err)
			}
			if checkErr := f.service.process(t.Context(), run); checkErr != nil {
				t.Fatal(checkErr)
			}
			if generator.calls.Load() <= 2 {
				t.Fatal("timeout was not applied to the generation pipeline")
			}
		})
	}
}

func TestRequestTimeoutValidationLeavesFailedRunUnchanged(t *testing.T) {
	t.Parallel()
	f := newRetryFixture(t)
	f.failRun(t)
	for _, seconds := range []int{-1, 1, 19, 901} {
		for _, resume := range []bool{false, true} {
			input := f.input()
			input.RequestTimeoutSeconds = seconds
			if !resume {
				input.ResumeRunID = ""
			}
			if _, checkErr := f.service.Start(t.Context(), f.run.ProductID, input); !errors.Is(checkErr, llmgateway.ErrInvalid) {
				t.Fatalf("invalid timeout %d was accepted (resume=%t): %v", seconds, resume, checkErr)
			}
		}
	}
	run, err := f.service.GetRun(t.Context(), f.run.ID)
	if err != nil || run.Status != "failed" || run.RequestTimeoutSeconds != 0 || len(run.Events) != len(f.run.Events) || retryQueueCount(t, f.service, f.run.ID) != 1 {
		t.Fatal("invalid execution timeout mutated the failed run or queue", err)
	}
}
