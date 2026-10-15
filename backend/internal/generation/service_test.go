package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
)

type fixtureGenerator struct{ fail atomic.Bool }

func (g *fixtureGenerator) Generate(_ context.Context, c llmgateway.Config, r llm.Request) (llm.Result, error) {
	if g.fail.Load() {
		return llm.Result{}, &llm.ProviderError{Code: "provider_error", Message: "Fixture provider is unavailable."}
	}
	if c.Secret != "original-key" {
		return llm.Result{}, errors.New("generation did not use its frozen connection revision")
	}
	var result any
	switch {
	case strings.HasPrefix(r.Input, "Analyze"):
		match := regexp.MustCompile(`<source_document id="([^"]+)"`).FindStringSubmatch(r.Input)
		if len(match) != 2 {
			return llm.Result{}, errors.New("missing source document")
		}
		name := "Payment service"
		if strings.Contains(r.Input, "Issue tracker") {
			name = "Issue tracker"
		}
		result = fixtureModel(name, match[1])
	case strings.HasPrefix(r.Input, "Reconcile"):
		text, _, _ := strings.Cut(strings.Split(r.Input, "ANALYSIS:\n")[1], "\nSOURCES:\n")
		var m passport.Model
		if checkErr := json.Unmarshal([]byte(text), &m); checkErr != nil {
			return llm.Result{}, fmt.Errorf("decode fixture model: %w", checkErr)
		}
		result = m
	case strings.HasPrefix(r.Input, "Create the standard"):
		for _, id := range passport.SectionIDs() {
			if !strings.Contains(r.Input, "# "+id+"\nGoal:") {
				return llm.Result{}, fmt.Errorf("composition omitted the %s section recipe", id)
			}
		}
		result = fixtureViews()
	default:
		return llm.Result{}, errors.New("unexpected generation stage")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return llm.Result{}, fmt.Errorf("encode fixture: %w", err)
	}
	return llm.Result{JSON: data, RequestID: "fixture-request", InputTokens: 100, OutputTokens: 50}, nil
}

func fixtureModel(name, docID string) passport.Model {
	return passport.Model{Title: name, Tagline: "Integration verification fixture", Summary: name + " stores requests.",
		Claims:    []passport.Claim{{ID: "claim-purpose", Text: name + " stores requests.", Basis: "source_backed", EvidenceIDs: []string{"evidence-purpose"}}},
		Evidence:  []passport.Evidence{{ID: "evidence-purpose", DocumentID: docID, StartLine: 1, EndLine: 1}},
		Objects:   []passport.Object{{ID: "api", Kind: "component", Name: "API", Summary: "Accepts requests.", Status: "implemented", ClaimIDs: []string{"claim-purpose"}}, {ID: "database", Kind: "store", Name: "Database", Summary: "Stores requests.", Status: "implemented", ClaimIDs: []string{"claim-purpose"}}},
		Relations: []passport.Relation{{ID: "store-request", From: "api", To: "database", Kind: "data", Label: "stores request", ClaimIDs: []string{"claim-purpose"}}},
		Scenarios: []passport.Scenario{{ID: "create-request", Title: "Create a request", Summary: "Trace request storage.", Steps: []passport.Step{{ID: "accept", Title: "Accept request", Summary: "API accepts the request.", ObjectIDs: []string{"api"}, ClaimIDs: []string{"claim-purpose"}}, {ID: "persist", Title: "Store request", Summary: "The request is persisted.", ObjectIDs: []string{"database"}, ClaimIDs: []string{"claim-purpose"}}}}}, OpenQuestions: []string{"Who owns the product?"}}
}

func fixtureViews() passport.Views {
	views := passport.Views{Sections: make([]passport.Section, 0), Diagrams: []passport.Diagram{{ID: "components", Title: "Request storage", Kind: "components", Summary: "A source-backed fixture view.", NodeIDs: []string{"api", "database"}, RelationIDs: []string{"store-request"}}}}
	for _, id := range passport.SectionIDs() {
		views.Sections = append(views.Sections, passport.Section{ID: id, Title: id, Status: "insufficient_evidence", Summary: "This integration fixture has limited source context.", ClaimIDs: []string{"claim-purpose"}, ObjectIDs: []string{"api", "database"}, DiagramIDs: []string{"components"}, ScenarioIDs: []string{"create-request"}})
	}
	return views
}

//nolint:gocyclo // End-to-end contract covers successful and failed durable runs.
func TestDurableFactoryAcrossProductsAndConnections(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	database, setupErr := db.Open(ctx, filepath.Join(t.TempDir(), "factory.db"))
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	connections, setupErr := llmgateway.NewStore(database, strings.Repeat("master-key", 5))
	if setupErr != nil {
		t.Fatal(setupErr)
	}
	generator := &fixtureGenerator{}
	service := New(database, connections, generator, &source.Collector{MaxBytes: 300000, MaxFiles: 120}, zerolog.Nop())
	runs := make([]Run, 0, 2)
	for index, protocol := range []string{"openai_responses", "anthropic_messages"} {
		name := []string{"Payment service", "Issue tracker"}[index]
		config := llmgateway.Config{Name: protocol, Enabled: true, Protocol: protocol, EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Secret: "original-key", Models: []llmgateway.Model{{ID: "same-model", OutputMode: "json_schema", MaxOutputTokens: 12000}}, TimeoutSeconds: 30, MaxConcurrent: 1}
		connection, err := connections.Save(ctx, "", config)
		if err != nil {
			t.Fatal(err)
		}
		p, err := service.Products.Create(ctx, product.Input{Name: name, Sources: []source.Input{{Kind: "text", Text: name + " stores requests."}}})
		if err != nil {
			t.Fatal(err)
		}
		run, err := service.Start(ctx, p.ID, Input{ConnectionID: connection.ID, Model: "same-model"})
		if err != nil {
			t.Fatal(err)
		}
		runs = append(runs, run)
		if _, err := service.Start(ctx, p.ID, Input{ConnectionID: connection.ID, Model: "same-model"}); !errors.Is(err, ErrActive) {
			t.Fatal("duplicate active generation was accepted", err)
		}
		config.Secret = "rotated-key"
		if _, err := connections.Save(ctx, connection.ID, config); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	go func() { defer close(done); service.Work(ctx) }()
	defer func() { cancel(); <-done }()
	for _, run := range runs {
		finished := waitForRun(t, service, run.ID)
		if finished.Status != "succeeded" {
			t.Fatalf("generation failed: %s", finished.ErrorMessage)
		}
		history, err := service.History(ctx, run.ProductID)
		if err != nil || len(history) != 1 {
			t.Fatal("missing unique revision", err)
		}
		r, err := service.Revision(ctx, history[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Content.Views.Sections) != 12 || len(r.Snapshots) != 1 {
			t.Fatal("incomplete passport")
		}
		evidence, err := service.Evidence(ctx, r.ID, "evidence-purpose")
		if err != nil || !strings.Contains(evidence.Excerpt, "stores requests") {
			t.Fatal("evidence not preserved", err)
		}
		if _, err := service.Evidence(ctx, r.ID, "unknown"); err == nil {
			t.Fatal("accepted unknown evidence")
		}
		if finished.InputTokens != 300 || finished.OutputTokens != 150 {
			t.Fatal("call usage not recorded")
		}
	}
	generator.fail.Store(true)
	failed, err := service.Start(ctx, runs[0].ProductID, Input{ConnectionID: runs[0].ConnectionID, Model: "same-model"})
	if err != nil {
		t.Fatal(err)
	}
	if run := waitForRun(t, service, failed.ID); run.Status != "failed" {
		t.Fatal("provider failure was not recorded")
	}
	history, err := service.History(ctx, runs[0].ProductID)
	if err != nil || len(history) != 1 {
		t.Fatal("failed run changed the previous passport", err)
	}
}

func waitForRun(t *testing.T, s *Service, id string) Run {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		r, err := s.GetRun(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if r.Status == "succeeded" || r.Status == "failed" {
			return r
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("generation did not finish")
	return Run{}
}
