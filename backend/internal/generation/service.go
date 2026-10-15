// Package generation orchestrates durable, bounded passport generation runs.
package generation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/productcard"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
	"maragu.dev/goqite"
)

// ErrActive indicates that a product already has a pending generation.
var ErrActive = errors.New("this product already has an active generation")

// Generator is the SDK boundary used by production clients and focused tests.
type Generator interface {
	Generate(context.Context, llmgateway.Config, llm.Request) (llm.Result, error)
}

// Collector is the source boundary, allowing tests without external repositories.
type Collector interface {
	Collect(context.Context, []source.Input) (source.Collection, error)
}

type selectingCollector interface {
	CollectSelected(context.Context, []source.Input, source.SelectFiles) (source.Collection, error)
}

// Input selects a saved connection and its configured model.
type Input struct {
	ConnectionID          string `json:"connection_id"`
	Model                 string `json:"model"`
	ResumeRunID           string `json:"resume_run_id,omitempty"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds,omitempty"` // Zero uses the frozen connection timeout; otherwise 20..900 seconds.
}

type sourceManifest struct {
	Sources      []source.Input        `json:"sources"`
	CardBrandkit *productcard.Brandkit `json:"card_brandkit,omitempty"`
}

// decodeManifest preserves queued runs created before brand kit support.
func decodeManifest(data string) (sourceManifest, error) {
	var manifest sourceManifest
	if strings.HasPrefix(strings.TrimSpace(data), "[") {
		if err := json.Unmarshal([]byte(data), &manifest.Sources); err != nil {
			return manifest, fmt.Errorf("decode legacy source manifest: %w", err)
		}
		return manifest, nil
	}
	if err := json.Unmarshal([]byte(data), &manifest); err != nil {
		return manifest, fmt.Errorf("decode source manifest: %w", err)
	}
	return manifest, nil
}

// Run exposes real processing stages and collection counts, not invented percentages.
type Run struct {
	ID                    string            `json:"id"`
	ProductID             string            `json:"product_id"`
	ConnectionID          string            `json:"connection_id"`
	ConnectionRevision    int               `json:"connection_revision"`
	Model                 string            `json:"model"`
	RequestTimeoutSeconds int               `json:"request_timeout_seconds"`
	Status                string            `json:"status"`
	Stage                 string            `json:"stage"`
	ErrorCode             string            `json:"error_code"`
	ErrorMessage          string            `json:"error_message"`
	CreatedAt             string            `json:"created_at"`
	StartedAt             string            `json:"started_at"`
	StageStartedAt        string            `json:"stage_started_at"`
	FinishedAt            string            `json:"finished_at"`
	DocumentCount         int               `json:"document_count"`
	InputTokens           int64             `json:"input_tokens"`
	OutputTokens          int64             `json:"output_tokens"`
	Events                []Event           `json:"events"`
	Research              *research.State   `json:"research,omitempty"`
	Analysis              *AnalysisState    `json:"analysis,omitempty"`
	Composition           *CompositionState `json:"composition,omitempty"`
}

// Revision includes an immutable draft and the exact source snapshot manifest.
type Revision struct {
	ID            string            `json:"id"`
	ProductID     string            `json:"product_id"`
	RunID         string            `json:"run_id"`
	Version       int               `json:"version"`
	SchemaVersion string            `json:"schema_version"`
	CreatedAt     string            `json:"created_at"`
	ApprovedBy    string            `json:"approved_by"`
	ApprovedAt    string            `json:"approved_at"`
	PublishedBy   string            `json:"published_by"`
	PublishedAt   string            `json:"published_at"`
	Published     bool              `json:"published"`
	Content       passport.Passport `json:"content"`
	Snapshots     []source.Snapshot `json:"snapshots"`
}

// RevisionInfo is a small history entry without the full passport payload.
type RevisionInfo struct {
	ID         string `json:"id"`
	Version    int    `json:"version"`
	CreatedAt  string `json:"created_at"`
	ApprovedAt string `json:"approved_at"`
	Published  bool   `json:"published"`
}

// EvidenceView contains only the source lines needed to inspect an assertion.
type EvidenceView struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Locator   string `json:"locator"`
	Revision  string `json:"revision"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Excerpt   string `json:"excerpt"`
	SourceURL string `json:"source_url"`
}

// Service owns a single-process worker pool over the existing SQLite database.
type Service struct {
	DB          *sql.DB
	Connections *llmgateway.Store
	LLM         Generator
	Collector   Collector
	Research    *research.Runner
	Analysis    Analyzer
	Products    product.Store
	Content     *content.Store
	Logger      zerolog.Logger
	queue       *goqite.Queue
	active      sync.Map // run ID -> context.CancelFunc
}

// New wires the durable queue and source/model dependencies.
func New(database *sql.DB, connections *llmgateway.Store, generator Generator, collector Collector, logger zerolog.Logger, files *content.Store) *Service {
	return &Service{DB: database, Connections: connections, LLM: generator, Collector: collector, Products: product.Store{DB: database}, Content: files, Logger: logger, queue: goqite.New(goqite.NewOpts{DB: database, Name: "passport", Timeout: 105 * time.Minute, MaxReceive: 10})}
}

// Start persists a source manifest and queue message in one transaction.
func (s *Service) Start(ctx context.Context, productID string, input Input) (Run, error) {
	if s.Connections == nil {
		return Run{}, errors.New("configure CONNECTION_MASTER_KEY before creating model connections")
	}
	if err := validateRequestTimeout(input.RequestTimeoutSeconds); err != nil {
		return Run{}, err
	}
	if input.ResumeRunID != "" {
		return s.resume(ctx, productID, input)
	}
	p, err := s.Products.Get(ctx, productID)
	if err != nil {
		return Run{}, fmt.Errorf("prepare generation: %w", err)
	}
	c, err := s.Connections.Get(ctx, input.ConnectionID)
	if err != nil {
		return Run{}, fmt.Errorf("prepare generation: %w", err)
	}
	if !c.Config.Enabled {
		return Run{}, fmt.Errorf("%w: connection is disabled", llmgateway.ErrInvalid)
	}
	if _, modelErr := llmgateway.ModelByID(c.Config, input.Model); modelErr != nil {
		return Run{}, fmt.Errorf("prepare generation: %w", modelErr)
	}
	manifest, err := json.Marshal(sourceManifest{Sources: p.Sources, CardBrandkit: p.CardBrandkit})
	if err != nil {
		return Run{}, fmt.Errorf("freeze source manifest: %w", err)
	}
	run := Run{ID: rand.Text(), ProductID: productID, ConnectionID: c.ID, ConnectionRevision: c.Revision, Model: input.Model, RequestTimeoutSeconds: input.RequestTimeoutSeconds, Status: "queued", Stage: "queued", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Events: []Event{}}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Run{}, fmt.Errorf("begin generation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var enabledConnection string
	if checkErr := tx.QueryRowContext(ctx, "SELECT id FROM llm_connections WHERE id=? AND enabled=1 AND deleted=0", c.ID).Scan(&enabledConnection); checkErr != nil {
		return Run{}, fmt.Errorf("check generation connection: %w", checkErr)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,request_timeout_seconds,status,stage,input_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)", run.ID, run.ProductID, run.ConnectionID, run.ConnectionRevision, run.Model, run.RequestTimeoutSeconds, run.Status, run.Stage, string(manifest), run.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: generation_runs.product_id") {
			return Run{}, ErrActive
		}
		return Run{}, fmt.Errorf("save generation: %w", err)
	}
	if checkErr := s.queue.SendTx(ctx, tx, goqite.Message{Body: []byte(run.ID)}); checkErr != nil {
		return Run{}, fmt.Errorf("enqueue generation: %w", checkErr)
	}
	if err := recordEvent(ctx, tx, run.ID, "run_queued", "queued", "Waiting for the worker."); err != nil {
		return Run{}, err
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return Run{}, fmt.Errorf("commit generation: %w", checkErr)
	}
	return s.GetRun(ctx, run.ID)
}

// GetRun returns processing state and recorded token usage.
func (s *Service) GetRun(ctx context.Context, id string) (Run, error) {
	var r Run
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return r, fmt.Errorf("begin generation snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	err = tx.QueryRowContext(ctx, `SELECT id,product_id,connection_id,connection_revision,model,request_timeout_seconds,status,stage,error_code,error_message,created_at,started_at,stage_started_at,COALESCE(finished_at,''),
 (SELECT COUNT(*) FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE ss.run_id=g.id),
 COALESCE((SELECT SUM(input_tokens) FROM llm_calls WHERE run_id=g.id),0),COALESCE((SELECT SUM(output_tokens) FROM llm_calls WHERE run_id=g.id),0)
 FROM generation_runs g WHERE id=?`, id).Scan(&r.ID, &r.ProductID, &r.ConnectionID, &r.ConnectionRevision, &r.Model, &r.RequestTimeoutSeconds, &r.Status, &r.Stage, &r.ErrorCode, &r.ErrorMessage, &r.CreatedAt, &r.StartedAt, &r.StageStartedAt, &r.FinishedAt, &r.DocumentCount, &r.InputTokens, &r.OutputTokens)
	if err != nil {
		return r, fmt.Errorf("read generation: %w", err)
	}
	// Older plan snapshots are represented by a short activity entry; only the
	// latest structured state is needed for polling and reconnects.
	rows, err := tx.QueryContext(ctx, `SELECT id,type,stage,
 CASE WHEN type='research_plan' AND id < (SELECT MAX(id) FROM generation_events WHERE run_id=? AND type='research_plan') THEN ''
 WHEN type='composition_progress' AND id < (SELECT MAX(id) FROM generation_events WHERE run_id=? AND type='composition_progress') THEN '' ELSE message END,
 created_at FROM generation_events WHERE run_id=? ORDER BY id`, id, id, id)
	if err != nil {
		return r, fmt.Errorf("list generation events: %w", err)
	}
	defer rows.Close()
	r.Events = make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Type, &event.Stage, &event.Message, &event.CreatedAt); err != nil {
			return r, fmt.Errorf("read generation event: %w", err)
		}
		if event.Type == "research_plan" {
			if event.Message == "" {
				event.Message = "Research plan updated."
				r.Events = append(r.Events, event)
				continue
			}
			var state research.State
			if err := json.Unmarshal([]byte(event.Message), &state); err != nil {
				return r, fmt.Errorf("read research progress: %w", err)
			}
			r.Research = &state
			event.Message = researchStateMessage(state)
		}
		if event.Type == "analysis_progress" {
			var state AnalysisState
			if decodeErr := json.Unmarshal([]byte(event.Message), &state); decodeErr != nil {
				return r, fmt.Errorf("read analysis progress: %w", decodeErr)
			}
			r.Analysis = &state
			event.Message = state.Action
		}
		if event.Type == "composition_progress" {
			if event.Message == "" {
				event.Message = "Passport writing progress updated."
				r.Events = append(r.Events, event)
				continue
			}
			var state CompositionState
			if err := json.Unmarshal([]byte(event.Message), &state); err != nil {
				return r, fmt.Errorf("read composition progress: %w", err)
			}
			r.Composition = &state
			event.Message = state.Action
		}
		r.Events = append(r.Events, event)
	}
	if err := rows.Err(); err != nil {
		return r, fmt.Errorf("read generation events: %w", err)
	}
	if err := rows.Close(); err != nil {
		return r, fmt.Errorf("close generation events: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return r, fmt.Errorf("commit generation snapshot: %w", err)
	}
	return r, nil
}

// Work recovers interrupted runs once, then processes tasks with the worker pool.
func (s *Service) Work(ctx context.Context, workers int) {
	if err := s.recoverRuns(ctx); err != nil {
		s.Logger.Error().Err(err).Msg("recover generation runs")
		return
	}
	var pool sync.WaitGroup
	for range workers {
		pool.Go(func() { s.work(ctx) })
	}
	pool.Wait()
}

func (s *Service) work(ctx context.Context) {
	for ctx.Err() == nil {
		message, err := s.queue.ReceiveAndWait(ctx, time.Second)
		if err != nil {
			if ctx.Err() == nil {
				s.Logger.Error().Err(err).Msg("receive generation task")
			}
			return
		}
		if message == nil {
			continue
		}
		run, err := s.GetRun(ctx, string(message.Body))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.Logger.Error().Err(err).Msg("read queued generation")
			continue
		}
		if err == nil && run.Status == "queued" {
			if err := s.processQueued(ctx, run); err != nil {
				s.Logger.Error().Err(err).Msg("process generation task")
				continue
			}
		}
		if ctx.Err() == nil {
			if checkErr := s.queue.Delete(ctx, message.ID); checkErr != nil {
				s.Logger.Error().Err(checkErr).Msg("acknowledge generation task")
			}
		}
	}
}

func (s *Service) processQueued(ctx context.Context, run Run) error {
	jobCtx, cancel := context.WithTimeout(ctx, 100*time.Minute)
	defer cancel()
	if _, loaded := s.active.LoadOrStore(run.ID, cancel); loaded {
		return nil
	}
	defer s.active.Delete(run.ID)
	claimed, err := s.claimRun(ctx, run.ID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	if err := s.process(jobCtx, run); err != nil {
		code := "generation_failed"
		if failure, ok := errors.AsType[*llm.ProviderError](err); ok {
			code = failure.Code
		}
		finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer finishCancel()
		if saveErr := s.failRun(finishCtx, run.ID, code, err.Error()); saveErr != nil {
			return saveErr
		}
		s.Logger.Warn().Str("run_id", run.ID).Str("code", code).Msg("passport generation failed")
	}
	return nil
}

//nolint:gocyclo // Keep the ordered generation stages and their failure boundaries visible together.
func (s *Service) process(ctx context.Context, run Run) error {
	if s.Connections == nil {
		return errors.New("connection storage is unavailable; configure CONNECTION_MASTER_KEY and start a new run")
	}
	var status string
	if err := s.DB.QueryRowContext(ctx, `SELECT status,request_timeout_seconds,
 (SELECT COUNT(*) FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE ss.run_id=g.id)
 FROM generation_runs g WHERE id=?`, run.ID).Scan(&status, &run.RequestTimeoutSeconds, &run.DocumentCount); err != nil {
		return fmt.Errorf("load generation processing state: %w", err)
	}
	if status != "queued" && status != "running" {
		return fmt.Errorf("%w: generation is not queued or running", llmgateway.ErrInvalid)
	}
	if checkErr := s.stage(ctx, run.ID, "collecting"); checkErr != nil {
		return checkErr
	}
	var manifest string
	if checkErr := s.DB.QueryRowContext(ctx, "SELECT input_json FROM generation_runs WHERE id=?", run.ID).Scan(&manifest); checkErr != nil {
		return fmt.Errorf("load source manifest: %w", checkErr)
	}
	frozen, decodeErr := decodeManifest(manifest)
	if decodeErr != nil {
		return fmt.Errorf("decode source manifest: %w", decodeErr)
	}
	inputs := frozen.Sources
	var productRepositories []string
	for _, input := range inputs {
		if input.Kind == "github" {
			productRepositories = append(productRepositories, input.Locator)
		}
	}
	collection, selected, err := s.collectOrLoadSources(ctx, run, inputs, productRepositories)
	if err != nil {
		return err
	}
	brand, err := s.collectOrLoadBrand(ctx, run, collection, frozen.CardBrandkit)
	if err != nil {
		return err
	}
	brandContext := ""
	if len(brand.References) > 0 || len(brand.Logos) > 0 {
		brandContext = brand.Context()
	}
	var contextText string
	if s.Research != nil {
		collection, contextText, err = s.researchSources(ctx, run, collection)
		if err != nil {
			return err
		}
	} else {
		contextText = source.Context(collection)
	}
	if !selected {
		if checkErr := s.event(ctx, run.ID, "stage_skipped", "selecting_sources", "Source selection was not needed for these inputs."); checkErr != nil {
			return checkErr
		}
	}
	requirements, promptErr := sectionRequirements()
	if promptErr != nil {
		return promptErr
	}
	if checkErr := s.stage(ctx, run.ID, "analyzing"); checkErr != nil {
		return checkErr
	}
	if s.Research != nil && s.Analysis == nil {
		return errors.New("incremental analysis must be configured for researched generation")
	}
	var model passport.Model
	if s.Analysis != nil {
		model, collection, err = s.analyzeIncrementally(ctx, run, collection)
		// Composition uses only the finalized canonical model, without source bodies.
	} else {
		// Compatibility for embedded consumers that have not configured the
		// incremental runner. The application always installs Service.Analysis.
		model, err = generate[passport.Model](ctx, s, run, "analyzing", analysisPrompt+"\n"+reconcilePrompt+"\n"+architectureResearchPrompt+"\nSECTION REQUIREMENTS:\n"+requirements+"\nSOURCES:\n"+contextText)
	}
	if err != nil {
		return err
	}
	reconciliationMessage := "Reconciliation is included in the analysis request."
	if s.Analysis != nil {
		reconciliationMessage = "Canonical reconciliation was completed through incremental model tools."
	}
	if checkErr := s.event(ctx, run.ID, "stage_skipped", "synthesizing", reconciliationMessage); checkErr != nil {
		return checkErr
	}
	if checkErr := s.stage(ctx, run.ID, "composing"); checkErr != nil {
		return checkErr
	}
	documents := make(map[string]int)
	for _, doc := range collection.Documents {
		documents[doc.ID] = doc.LineCount
	}
	views, err := s.compose(ctx, run, model, documents, brandContext)
	if err != nil {
		return err
	}
	p := passport.Passport{Model: model, Views: views}
	if checkErr := s.stage(ctx, run.ID, "validating"); checkErr != nil {
		return checkErr
	}
	if validationErr := validateResearchPassport(p, documents, collection, s.Research != nil || s.Analysis != nil); validationErr != nil {
		return fmt.Errorf("passport validation failed: %w", validationErr)
	}
	// Resolve assets after repair so image bytes never enter a model request.
	p.Views.CardTheme = brand.Theme(p.Views.CardTheme)
	if err := s.saveCitedCoverage(ctx, run.ID, p, collection); err != nil {
		return err
	}
	return s.saveRevision(ctx, run, p)
}

func generate[T any](ctx context.Context, s *Service, run Run, stage, input string) (T, error) {
	return generateWith[T](ctx, s, run, stage, input, generationCall{RetryProvider: true}, nil)
}

type generationCall struct {
	Label           string
	MaxOutputTokens int
	RetryProvider   bool
	OnAttempt       func(int) error
}

const (
	maxProviderAttempts   = 6
	maxValidationAttempts = 6
	// Each failure spends only its own budget; the last request exhausts one.
	maxGenerationAttempts = maxProviderAttempts + maxValidationAttempts - 1
)

func (o generationCall) maxAttempts() int {
	if o.RetryProvider {
		return maxGenerationAttempts
	}
	return maxValidationAttempts
}

//nolint:gocyclo // Keep per-attempt usage, provider retries and response checks at one request boundary.
func generateWith[T any](ctx context.Context, s *Service, run Run, stage, input string, options generationCall, check func(T) error) (T, error) {
	var empty T
	schema, err := passport.OutputSchema[T]()
	if err != nil {
		return empty, err
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		return empty, fmt.Errorf("encode output schema: %w", err)
	}
	var format map[string]any
	if checkErr := json.Unmarshal(encoded, &format); checkErr != nil {
		return empty, fmt.Errorf("decode output schema: %w", checkErr)
	}
	originalInput := input
	providerFailures, validationFailures := 0, 0
	for attempt := 1; attempt <= options.maxAttempts(); attempt++ {
		if err := ctx.Err(); err != nil {
			return empty, fmt.Errorf("model task interrupted: %w", err)
		}
		if options.OnAttempt != nil {
			if err := options.OnAttempt(attempt); err != nil {
				return empty, err
			}
		}
		config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
		if err != nil {
			return empty, errors.New("the selected connection is disabled or unavailable")
		}
		if run.RequestTimeoutSeconds > 0 {
			config.TimeoutSeconds = run.RequestTimeoutSeconds
		}
		started := time.Now()
		label := "Model request"
		if options.Label != "" {
			label = options.Label
		}
		if err := s.event(ctx, run.ID, "model_call_started", stage, fmt.Sprintf("%s started (attempt %d of %d). Waiting for the response.", label, attempt, options.maxAttempts())); err != nil {
			return empty, err
		}
		result, callErr := s.LLM.Generate(ctx, config, llm.Request{Model: run.Model, System: systemPrompt, Input: input, Schema: format, MaxOutputTokens: options.MaxOutputTokens})
		code := ""
		if callErr != nil {
			code = "provider_error"
			if failure, ok := errors.AsType[*llm.ProviderError](callErr); ok {
				code = failure.Code
			}
		}
		callID := rand.Text()
		outputPath, outputHash := "", ""
		if len(result.JSON) > 0 {
			outputPath = "runs/" + run.ID + "/model-responses/" + callID + ".json"
			if !json.Valid(result.JSON) {
				outputPath = "runs/" + run.ID + "/model-responses/" + callID + ".txt"
			}
			var writeErr error
			outputHash, writeErr = s.Content.Write(outputPath, result.JSON)
			if writeErr != nil {
				return empty, fmt.Errorf("save model response: %w", writeErr)
			}
		}
		_, saveErr := s.DB.ExecContext(ctx, "INSERT INTO llm_calls(id,run_id,stage,attempt,prompt_version,request_id,input_tokens,output_tokens,duration_ms,output_path,output_hash,error_code,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)", callID, run.ID, stage, attempt, PromptVersion, result.RequestID, result.InputTokens, result.OutputTokens, time.Since(started).Milliseconds(), outputPath, outputHash, code, time.Now().UTC().Format(time.RFC3339Nano))
		if saveErr != nil {
			return empty, fmt.Errorf("save model call: %w", saveErr)
		}
		if callErr != nil {
			if err := s.event(ctx, run.ID, "model_call_failed", stage, "The model request failed."); err != nil {
				return empty, err
			}
		}
		if callErr != nil && code != "invalid_json" {
			providerFailures++
			if options.RetryProvider && providerFailures < maxProviderAttempts && ctx.Err() == nil && (code == "timeout" || code == "provider_error" || code == "rate_limited") {
				if err := s.event(ctx, run.ID, "retry_started", stage, fmt.Sprintf("%s did not complete (API failure %d of %d). Retrying only this task.", label, providerFailures, maxProviderAttempts)); err != nil {
					return empty, err
				}
				continue
			}
			return empty, fmt.Errorf("generate %s (%s): %w", stage, label, callErr)
		}
		if callErr == nil {
			if err := s.event(ctx, run.ID, "model_call_completed", stage, fmt.Sprintf("%s response received. Recorded usage: %d input tokens and %d output tokens.", label, result.InputTokens, result.OutputTokens)); err != nil {
				return empty, err
			}
		}
		value, validationErr := empty, callErr
		if validationErr == nil {
			value, validationErr = passport.CheckJSON[T](result.JSON)
		}
		if validationErr == nil && check != nil {
			validationErr = check(value)
		}
		if validationErr == nil {
			return value, nil
		}
		validationFailures++
		if validationFailures == maxValidationAttempts {
			return empty, fmt.Errorf("%s response failed validation: %w", label, validationErr)
		}
		if err := s.event(ctx, run.ID, "retry_started", stage, fmt.Sprintf("%s did not pass validation (invalid response %d of %d). Retrying only this task.", label, validationFailures, maxValidationAttempts)); err != nil {
			return empty, err
		}
		input = originalInput + "\n" + schemaRepairPrompt + "\nERROR:\n" + validationErr.Error() + "\nPREVIOUS RESPONSE:\n" + string(result.JSON)
	}
	return empty, errors.New("model generation did not produce a result")
}

func (s *Service) saveSources(ctx context.Context, runID string, collection source.Collection) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin source snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, snapshot := range collection.Snapshots {
		coverage, _ := json.Marshal(snapshot.Coverage)
		if _, err := tx.ExecContext(ctx, "INSERT INTO source_snapshots(id,run_id,locator,revision,coverage_json) VALUES(?,?,?,?,?)", snapshot.ID, runID, snapshot.Locator, snapshot.Revision, string(coverage)); err != nil {
			return fmt.Errorf("save source snapshot: %w", err)
		}
	}
	for _, doc := range collection.Documents {
		path := "runs/" + runID + "/sources/" + doc.ID + ".txt"
		if _, err := s.Content.Write(path, []byte(doc.Content)); err != nil {
			return fmt.Errorf("save source file: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO source_documents(id,snapshot_id,path,content_hash,content,line_count,content_path) VALUES(?,?,?,?,'',?,?)", doc.ID, doc.SnapshotID, doc.Path, doc.ContentHash, doc.LineCount, path); err != nil {
			return fmt.Errorf("save source document: %w", err)
		}
	}
	if err := recordEvent(ctx, tx, runID, "sources_collected", "collecting", fmt.Sprintf("Source collection completed: %d source documents saved.", len(collection.Documents))); err != nil {
		return err
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return fmt.Errorf("commit source snapshot: %w", checkErr)
	}
	return nil
}

func (s *Service) saveRevision(ctx context.Context, run Run, p passport.Passport) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin passport revision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	if checkErr := tx.QueryRowContext(ctx, "SELECT status FROM generation_runs WHERE id=?", run.ID).Scan(&status); checkErr != nil {
		return fmt.Errorf("read generation before saving passport: %w", checkErr)
	}
	if status != "queued" && status != "running" {
		return context.Canceled
	}
	var version int
	if queryErr := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0)+1 FROM passport_revisions WHERE product_id=?", run.ProductID).Scan(&version); queryErr != nil {
		return fmt.Errorf("allocate passport version: %w", queryErr)
	}
	path := fmt.Sprintf("products/%s/versions/%d/%s/passport.json", run.ProductID, version, run.ID)
	hash, err := s.Content.WriteJSON(path, p)
	if err != nil {
		return fmt.Errorf("save passport file: %w", err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_path,content_hash,created_at) VALUES(?,?,?,?,'1','{}',?,?,?)", rand.Text(), run.ProductID, run.ID, version, path, hash, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save passport revision: %w", err)
	}
	if _, updateErr := tx.ExecContext(ctx, "UPDATE generation_runs SET status='succeeded',stage='complete',finished_at=? WHERE id=?", time.Now().UTC().Format(time.RFC3339Nano), run.ID); updateErr != nil {
		return fmt.Errorf("complete generation: %w", updateErr)
	}
	themeJSON, err := json.Marshal(p.Views.CardTheme)
	if err != nil {
		return fmt.Errorf("encode product theme: %w", err)
	}
	// Older products store a source array. Wrap it before adding local theme metadata.
	if _, err := tx.ExecContext(ctx, `UPDATE products SET sources_json=json_set(
	 CASE WHEN json_type(sources_json)='array' THEN json_object('sources',json(sources_json)) ELSE sources_json END,
	 '$.card_theme',json(?)) WHERE id=?`, string(themeJSON), run.ProductID); err != nil {
		return fmt.Errorf("save product theme: %w", err)
	}
	if err := recordEvent(ctx, tx, run.ID, "stage_completed", "validating", "Passport validation completed."); err != nil {
		return err
	}
	if err := recordEvent(ctx, tx, run.ID, "run_succeeded", "complete", "Passport draft saved. Ready for review."); err != nil {
		return err
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return fmt.Errorf("commit passport revision: %w", checkErr)
	}
	return nil
}

// RunProduct returns the product that owns a generation run.
func (s *Service) RunProduct(ctx context.Context, runID string) (string, error) {
	var productID string
	if err := s.DB.QueryRowContext(ctx, "SELECT product_id FROM generation_runs WHERE id=?", runID).Scan(&productID); err != nil {
		return "", fmt.Errorf("get run product: %w", err)
	}
	return productID, nil
}

// RevisionProduct returns the product that owns a passport revision.
func (s *Service) RevisionProduct(ctx context.Context, revisionID string) (string, error) {
	var productID string
	if err := s.DB.QueryRowContext(ctx, "SELECT product_id FROM passport_revisions WHERE id=?", revisionID).Scan(&productID); err != nil {
		return "", fmt.Errorf("get revision product: %w", err)
	}
	return productID, nil
}

// History returns compact revision metadata.
func (s *Service) History(ctx context.Context, productID string) ([]RevisionInfo, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT r.id,r.version,r.created_at,r.approved_at,COALESCE(p.published_revision_id=r.id,0) FROM passport_revisions r JOIN products p ON p.id=r.product_id WHERE r.product_id=? ORDER BY r.version DESC", productID)
	if err != nil {
		return nil, fmt.Errorf("list passport revisions: %w", err)
	}
	defer rows.Close()
	result := make([]RevisionInfo, 0)
	for rows.Next() {
		var r RevisionInfo
		if checkErr := rows.Scan(&r.ID, &r.Version, &r.CreatedAt, &r.ApprovedAt, &r.Published); checkErr != nil {
			return nil, fmt.Errorf("read revision history: %w", checkErr)
		}
		result = append(result, r)
	}
	if checkErr := rows.Err(); checkErr != nil {
		return nil, fmt.Errorf("read revision history: %w", checkErr)
	}
	return result, nil
}

// Revision returns one saved version and its exact snapshot manifest.
func (s *Service) Revision(ctx context.Context, id string) (Revision, error) {
	var r Revision
	var path, hash string
	err := s.DB.QueryRowContext(ctx, `SELECT r.id,r.product_id,r.run_id,r.version,r.schema_version,r.content_path,r.content_hash,r.created_at,
 r.approved_by,r.approved_at,r.published_by,r.published_at,COALESCE(p.published_revision_id=r.id,0)
 FROM passport_revisions r JOIN products p ON p.id=r.product_id WHERE r.id=?`, id).Scan(&r.ID, &r.ProductID, &r.RunID, &r.Version, &r.SchemaVersion, &path, &hash, &r.CreatedAt, &r.ApprovedBy, &r.ApprovedAt, &r.PublishedBy, &r.PublishedAt, &r.Published)
	if err != nil {
		return r, fmt.Errorf("get passport revision: %w", err)
	}
	if checkErr := s.Content.ReadJSON(path, hash, &r.Content); checkErr != nil {
		return r, fmt.Errorf("decode passport revision: %w", checkErr)
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id,locator,revision,coverage_json FROM source_snapshots WHERE run_id=?", r.RunID)
	if err != nil {
		return r, fmt.Errorf("get source snapshots: %w", err)
	}
	defer rows.Close()
	r.Snapshots = make([]source.Snapshot, 0)
	for rows.Next() {
		var snap source.Snapshot
		var coverage string
		if checkErr := rows.Scan(&snap.ID, &snap.Locator, &snap.Revision, &coverage); checkErr != nil {
			return r, fmt.Errorf("read snapshot: %w", checkErr)
		}
		if checkErr := json.Unmarshal([]byte(coverage), &snap.Coverage); checkErr != nil {
			return r, fmt.Errorf("decode source coverage: %w", checkErr)
		}
		r.Snapshots = append(r.Snapshots, snap)
	}
	if checkErr := rows.Err(); checkErr != nil {
		return r, fmt.Errorf("read snapshots: %w", checkErr)
	}
	return r, nil
}

// Evidence reads only an anchor belonging to this revision's run.
func (s *Service) Evidence(ctx context.Context, revisionID, evidenceID string) (EvidenceView, error) {
	r, err := s.Revision(ctx, revisionID)
	if err != nil {
		return EvidenceView{}, err
	}
	var anchor *passport.Evidence
	for _, e := range r.Content.Model.Evidence {
		if e.ID == evidenceID {
			evidence := e
			anchor = &evidence
			break
		}
	}
	if anchor == nil {
		return EvidenceView{}, sql.ErrNoRows
	}
	view := EvidenceView{ID: anchor.ID, StartLine: anchor.StartLine, EndLine: anchor.EndLine}
	var path, hash string
	err = s.DB.QueryRowContext(ctx, "SELECT d.path,d.content_path,d.content_hash,ss.locator,ss.revision FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE d.id=? AND ss.run_id=?", anchor.DocumentID, r.RunID).Scan(&view.Path, &path, &hash, &view.Locator, &view.Revision)
	if err != nil {
		return view, fmt.Errorf("get evidence document: %w", err)
	}
	data, err := s.Content.Read(path, hash)
	if err != nil {
		return view, fmt.Errorf("read evidence file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	if anchor.StartLine < 1 || anchor.EndLine > len(lines) {
		return view, errors.New("stored evidence anchor is invalid")
	}
	view.Excerpt = strings.Join(lines[anchor.StartLine-1:anchor.EndLine], "\n")
	if strings.HasPrefix(view.Locator, "https://github.com/") {
		view.SourceURL = strings.TrimSuffix(view.Locator, ".git") + "/blob/" + view.Revision + "/" + view.Path + fmt.Sprintf("#L%d-L%d", view.StartLine, view.EndLine)
	}
	return view, nil
}
