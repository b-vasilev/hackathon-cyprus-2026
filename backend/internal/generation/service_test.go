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

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/productcard"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
)

type fixtureGenerator struct{ fail atomic.Bool }

//nolint:gocyclo // Shared fixture validates each distinct generation request contract.
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
		if strings.Contains(r.Input, "brand-accent-fixture") {
			return llm.Result{}, errors.New("brand kit leaked into product evidence")
		}
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
	case strings.HasPrefix(r.Input, "Plan one product-passport"):
		id, _, _ := strings.Cut(strings.Split(r.Input, "CHAPTER ID: ")[1], "\n")
		if !strings.Contains(r.Input, "RECIPE:\n# "+id+"\nGoal:") {
			return llm.Result{}, errors.New("chapter planning omitted its detailed recipe")
		}
		plan := chapterPlan{Title: id, Question: "What does " + id + " explain?", Parts: []compositionPart{}}
		for _, chapter := range fixtureViews().Document.Chapters {
			if chapter.ID != id {
				continue
			}
			plan.Title, plan.Question = chapter.Title, chapter.Question
			for i, block := range chapter.Blocks {
				part := compositionPart{Title: fmt.Sprintf("Point %d", i+1), Instructions: fmt.Sprintf("Fixture block %d", i), Kind: block.Kind, ObjectIDs: []string{"api", "database"}, ClaimIDs: []string{"claim-purpose"}, ScenarioIDs: []string{}, TableStyle: block.TableStyle}
				if block.Kind == "scenario_ref" {
					part.ScenarioIDs = []string{block.ReferenceID}
				}
				if block.Kind == "diagram_ref" {
					diagram := fixtureViews().Diagrams[0]
					part.Diagram = &compositionDiagramPlan{Kind: diagram.Kind, ScopeObjectID: diagram.ScopeObjectID}
				}
				plan.Parts = append(plan.Parts, part)
			}
		}
		result = plan
	case strings.HasPrefix(r.Input, "Generate ONLY the ASSIGNED PART"):
		id, _, _ := strings.Cut(strings.Split(r.Input, "CHAPTER ID: ")[1], "\n")
		if !strings.Contains(r.Input, "DETAILED CHAPTER REQUIREMENTS:\n# "+id+"\nGoal:") {
			return llm.Result{}, errors.New("part generation omitted its detailed chapter recipe")
		}
		partJSON, _, _ := strings.Cut(strings.Split(r.Input, "ASSIGNED PART:\n")[1], "\nMODEL:\n")
		var part compositionPart
		if err := json.Unmarshal([]byte(partJSON), &part); err != nil {
			return llm.Result{}, fmt.Errorf("decode fixture part: %w", err)
		}
		var index int
		if _, err := fmt.Sscanf(part.Instructions, "Fixture block %d", &index); err != nil {
			return llm.Result{}, fmt.Errorf("decode fixture block index: %w", err)
		}
		for _, chapter := range fixtureViews().Document.Chapters {
			if chapter.ID != id {
				continue
			}
			fragment := compositionFragment{Block: chapter.Blocks[index]}
			if fragment.Block.Kind == "diagram_ref" {
				diagram := fixtureViews().Diagrams[0]
				fragment.Diagram = &diagram
			}
			result = fragment
		}
	case strings.HasPrefix(r.Input, "Choose only the product card theme"):
		theme := compositionTheme{}
		if strings.Contains(r.Input, "brand-accent-fixture") {
			theme.CardTheme = &passport.ProductCardTheme{AccentColor: "#123456", BackgroundColor: "#FDFCFD"}
		}
		if strings.Contains(r.Input, "auto-brand-fixture") {
			theme.CardTheme = &passport.ProductCardTheme{AccentColor: "#357ABC", BackgroundColor: "#FDFCFD"}
		}
		result = theme
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
	return passport.Views{
		Sections: []passport.Section{},
		Diagrams: []passport.Diagram{{ID: "components", Title: "Request storage", Kind: "components", Summary: "A source-backed fixture view.", NodeIDs: []string{"api", "database"}, RelationIDs: []string{"store-request"}}},
		Document: &passport.Document{Version: 2, Chapters: []passport.Chapter{
			{ID: "overview", Title: "Product overview", Question: "What does it do?", Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "The fixture stores requests.", ClaimIDs: []string{"claim-purpose"}}}},
			{ID: "repositories", Title: "Repositories and connections", Question: "Where is its code?", Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "The supplied fixture does not establish repository wiring."}}},
			{ID: "architecture", Title: "Logical architecture", Question: "How is responsibility divided?", Blocks: []passport.DocumentBlock{
				{Kind: "markdown", Markdown: "The API persists requests in a store.", ClaimIDs: []string{"claim-purpose"}},
				{Kind: "diagram_ref", ReferenceID: "components"},
				{Kind: "markdown", Markdown: "The next chapter follows the same components in execution order.", ClaimIDs: []string{"claim-purpose"}},
			}},
			{ID: "scenarios", Title: "End-to-end flows", Question: "How is a result produced?", Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "The API accepts requests and persists them.", ClaimIDs: []string{"claim-purpose"}}, {Kind: "scenario_ref", ReferenceID: "create-request"}}},
			{ID: "technologies", Title: "Technologies and tools", Question: "What implements these responsibilities?", Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "An API and database implement request storage.", ClaimIDs: []string{"claim-purpose"}}, {Kind: "object_table", TableStyle: "objects", ObjectIDs: []string{"api", "database"}, ClaimIDs: []string{"claim-purpose"}}}},
		}},
	}
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
	service := New(database, connections, generator, &source.Collector{MaxBytes: 300000, MaxFiles: 120}, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	runs := make([]Run, 0, 2)
	for index, protocol := range []string{"openai_responses", "anthropic_messages", "openai_chat_completions"} {
		name := []string{"Payment service", "Issue tracker", "Automatic brand"}[index]
		ownerCtx := connectionowner.WithUser(ctx, protocol)
		config := llmgateway.Config{Name: protocol, Enabled: true, Protocol: protocol, EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Secret: "original-key", Models: []llmgateway.Model{{ID: "same-model", OutputMode: "json_schema", MaxOutputTokens: 12000}}, TimeoutSeconds: 30, MaxConcurrent: 1}
		connection, err := connections.Save(ownerCtx, "", config)
		if err != nil {
			t.Fatal(err)
		}
		input := product.Input{Name: name, Sources: []source.Input{{Kind: "text", Text: name + " stores requests."}}}
		if index == 0 {
			input.CardBrandkit = &productcard.Brandkit{FileName: "brand.md", Text: "brand-accent-fixture: #123456; card background #FDFCFD"}
		}
		if index == 2 {
			input.Sources[0].Text += "\nauto-brand-fixture: primary color #357ABC"
		}
		p, err := service.Products.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		if index == 1 {
			// Existing products and queued runs used a bare source array.
			legacy, _ := json.Marshal(input.Sources)
			if _, updateErr := database.ExecContext(ctx, "UPDATE products SET sources_json=? WHERE id=?", string(legacy), p.ID); updateErr != nil {
				t.Fatal(updateErr)
			}
		}
		run, err := service.Start(ownerCtx, p.ID, Input{ConnectionID: connection.ID, Model: "same-model"})
		if err != nil {
			t.Fatal(err)
		}
		if index == 1 {
			legacy, _ := json.Marshal(input.Sources)
			if _, err := database.ExecContext(ctx, "UPDATE generation_runs SET input_json=? WHERE id=?", string(legacy), run.ID); err != nil {
				t.Fatal(err)
			}
		}
		runs = append(runs, run)
		if _, err := service.Start(ownerCtx, p.ID, Input{ConnectionID: connection.ID, Model: "same-model"}); !errors.Is(err, ErrActive) {
			t.Fatal("duplicate active generation was accepted", err)
		}
		config.Secret = "rotated-key"
		if _, err := connections.Save(ownerCtx, connection.ID, config); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	go func() { defer close(done); service.Work(ctx, 1) }()
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
		if r.Content.Views.Document == nil || r.Content.Views.Document.Version != 2 || len(r.Content.Views.Document.Chapters) != 5 || len(r.Snapshots) != 1 {
			t.Fatal("incomplete passport")
		}
		architecture := r.Content.Views.Document.Chapters[2].Blocks
		if len(architecture) != 3 || architecture[0].Kind != "markdown" || architecture[1].Kind != "diagram_ref" || architecture[2].Kind != "markdown" {
			t.Fatal("authored narrative order was not preserved in the saved revision")
		}
		p, err := service.Products.Get(ctx, run.ProductID)
		if err != nil {
			t.Fatal(err)
		}
		switch p.Name {
		case "Payment service":
			if p.CardBrandkit == nil || p.CardTheme == nil || r.Content.Views.CardTheme == nil || p.CardTheme.AccentColor != "#123456" {
				t.Fatal("brand kit or generated product theme was not preserved")
			}
		case "Automatic brand":
			if p.CardBrandkit != nil || p.CardTheme == nil || r.Content.Views.CardTheme == nil || p.CardTheme.AccentColor != "#357ABC" || len(p.CardTheme.Sources) == 0 {
				t.Fatal("branding discovered in product sources was not applied to both product and passport")
			}
		default:
			if p.CardBrandkit != nil || p.CardTheme != nil || r.Content.Views.CardTheme != nil {
				t.Fatal("brand styling leaked into a product without a kit")
			}
		}
		evidence, err := service.Evidence(ctx, r.ID, "evidence-purpose")
		if err != nil || !strings.Contains(evidence.Excerpt, "stores requests") {
			t.Fatal("evidence not preserved", err)
		}
		if _, err := service.Evidence(ctx, r.ID, "unknown"); err == nil {
			t.Fatal("accepted unknown evidence")
		}
		var calls int
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM llm_calls WHERE run_id=?", run.ID).Scan(&calls); err != nil {
			t.Fatal(err)
		}
		if calls <= 2 || finished.InputTokens != int64(calls*100) || finished.OutputTokens != int64(calls*50) {
			t.Fatal("call usage not recorded")
		}
	}
	generator.fail.Store(true)
	failed, err := service.Start(connectionowner.WithUser(ctx, "openai_responses"), runs[0].ProductID, Input{ConnectionID: runs[0].ConnectionID, Model: "same-model"})
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
