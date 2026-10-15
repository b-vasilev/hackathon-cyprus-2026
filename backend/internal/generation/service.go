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
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
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
	ConnectionID string `json:"connection_id"`
	Model        string `json:"model"`
}

// Run exposes real processing stages and collection counts, not invented percentages.
type Run struct {
	ID                 string `json:"id"`
	ProductID          string `json:"product_id"`
	ConnectionID       string `json:"connection_id"`
	ConnectionRevision int    `json:"connection_revision"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Stage              string `json:"stage"`
	ErrorCode          string `json:"error_code"`
	ErrorMessage       string `json:"error_message"`
	CreatedAt          string `json:"created_at"`
	FinishedAt         string `json:"finished_at"`
	DocumentCount      int    `json:"document_count"`
	InputTokens        int64  `json:"input_tokens"`
	OutputTokens       int64  `json:"output_tokens"`
}

// Revision includes an immutable draft and the exact source snapshot manifest.
type Revision struct {
	ID            string            `json:"id"`
	ProductID     string            `json:"product_id"`
	RunID         string            `json:"run_id"`
	Version       int               `json:"version"`
	SchemaVersion string            `json:"schema_version"`
	CreatedAt     string            `json:"created_at"`
	Content       passport.Passport `json:"content"`
	Snapshots     []source.Snapshot `json:"snapshots"`
}

// RevisionInfo is a small history entry without the full passport payload.
type RevisionInfo struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
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

// Service owns a single-process worker over the existing SQLite database.
type Service struct {
	DB          *sql.DB
	Connections *llmgateway.Store
	LLM         Generator
	Collector   Collector
	Products    product.Store
	Logger      zerolog.Logger
	queue       *goqite.Queue
}

// New wires the durable queue and source/model dependencies.
func New(database *sql.DB, connections *llmgateway.Store, generator Generator, collector Collector, logger zerolog.Logger) *Service {
	return &Service{DB: database, Connections: connections, LLM: generator, Collector: collector, Products: product.Store{DB: database}, Logger: logger, queue: goqite.New(goqite.NewOpts{DB: database, Name: "passport", Timeout: 25 * time.Minute, MaxReceive: 10})}
}

// Start persists a source manifest and queue message in one transaction.
func (s *Service) Start(ctx context.Context, productID string, input Input) (Run, error) {
	if s.Connections == nil {
		return Run{}, errors.New("configure LLM_MASTER_KEY before creating model connections")
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
	manifest, err := json.Marshal(p.Sources)
	if err != nil {
		return Run{}, fmt.Errorf("freeze source manifest: %w", err)
	}
	run := Run{ID: rand.Text(), ProductID: productID, ConnectionID: c.ID, ConnectionRevision: c.Revision, Model: input.Model, Status: "queued", Stage: "queued", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Run{}, fmt.Errorf("begin generation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, "INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at) VALUES(?,?,?,?,?,?,?,?,?)", run.ID, run.ProductID, run.ConnectionID, run.ConnectionRevision, run.Model, run.Status, run.Stage, string(manifest), run.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: generation_runs.product_id") {
			return Run{}, ErrActive
		}
		return Run{}, fmt.Errorf("save generation: %w", err)
	}
	if checkErr := s.queue.SendTx(ctx, tx, goqite.Message{Body: []byte(run.ID)}); checkErr != nil {
		return Run{}, fmt.Errorf("enqueue generation: %w", checkErr)
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return Run{}, fmt.Errorf("commit generation: %w", checkErr)
	}
	return run, nil
}

// GetRun returns processing state and recorded token usage.
func (s *Service) GetRun(ctx context.Context, id string) (Run, error) {
	var r Run
	err := s.DB.QueryRowContext(ctx, `SELECT id,product_id,connection_id,connection_revision,model,status,stage,error_code,error_message,created_at,COALESCE(finished_at,''),
 (SELECT COUNT(*) FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE ss.run_id=g.id),
 COALESCE((SELECT SUM(input_tokens) FROM llm_calls WHERE run_id=g.id),0),COALESCE((SELECT SUM(output_tokens) FROM llm_calls WHERE run_id=g.id),0)
 FROM generation_runs g WHERE id=?`, id).Scan(&r.ID, &r.ProductID, &r.ConnectionID, &r.ConnectionRevision, &r.Model, &r.Status, &r.Stage, &r.ErrorCode, &r.ErrorMessage, &r.CreatedAt, &r.FinishedAt, &r.DocumentCount, &r.InputTokens, &r.OutputTokens)
	if err != nil {
		return r, fmt.Errorf("read generation: %w", err)
	}
	return r, nil
}

// Work recovers interrupted runs and processes one durable task at a time.
//
//nolint:nestif // A single worker owns task completion, failure recording and acknowledgement.
func (s *Service) Work(ctx context.Context) {
	if _, err := s.DB.ExecContext(ctx, "UPDATE generation_runs SET status='failed',error_code='interrupted',error_message='The server stopped during generation. Start a new run.',finished_at=? WHERE status='running'", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		s.Logger.Error().Err(err).Msg("recover generation runs")
		return
	}
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
		if err == nil && run.Status == "queued" {
			jobCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
			err = s.process(jobCtx, run)
			cancel()
			if err != nil {
				code := "generation_failed"
				if failure, ok := errors.AsType[*llm.ProviderError](err); ok {
					code = failure.Code
				}
				finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
				_, saveErr := s.DB.ExecContext(finishCtx, "UPDATE generation_runs SET status='failed',error_code=?,error_message=?,finished_at=? WHERE id=? AND status!='succeeded'", code, err.Error(), time.Now().UTC().Format(time.RFC3339Nano), run.ID)
				finishCancel()
				if saveErr != nil {
					s.Logger.Error().Err(saveErr).Msg("save generation failure")
				}
				s.Logger.Warn().Str("run_id", run.ID).Str("code", code).Msg("passport generation failed")
			}
		}
		if ctx.Err() == nil {
			if checkErr := s.queue.Delete(ctx, message.ID); checkErr != nil {
				s.Logger.Error().Err(checkErr).Msg("acknowledge generation task")
			}
		}
	}
}

func (s *Service) stage(ctx context.Context, id, stage string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE generation_runs SET status='running',stage=? WHERE id=?", stage, id)
	if err != nil {
		return fmt.Errorf("update generation stage: %w", err)
	}
	return nil
}

func (s *Service) collectSources(ctx context.Context, run Run, inputs []source.Input) (source.Collection, error) {
	var collection source.Collection
	var err error
	if collector, ok := s.Collector.(selectingCollector); ok {
		collection, err = collector.CollectSelected(ctx, inputs, func(selectCtx context.Context, catalog string) ([]string, error) {
			if checkErr := s.stage(selectCtx, run.ID, "selecting_sources"); checkErr != nil {
				return nil, checkErr
			}
			selection, selectErr := generate[source.FileSelection](selectCtx, s, run, "selecting_sources", catalog+"\n"+selectionPrompt)
			return selection.Paths, selectErr
		})
	} else {
		collection, err = s.Collector.Collect(ctx, inputs)
	}
	if err != nil {
		return source.Collection{}, fmt.Errorf("collect generation sources: %w", err)
	}
	return collection, nil
}

func (s *Service) process(ctx context.Context, run Run) error {
	if s.Connections == nil {
		return errors.New("connection storage is unavailable; configure LLM_MASTER_KEY and start a new run")
	}
	if checkErr := s.stage(ctx, run.ID, "collecting"); checkErr != nil {
		return checkErr
	}
	var manifest string
	if checkErr := s.DB.QueryRowContext(ctx, "SELECT input_json FROM generation_runs WHERE id=?", run.ID).Scan(&manifest); checkErr != nil {
		return fmt.Errorf("load source manifest: %w", checkErr)
	}
	var inputs []source.Input
	if checkErr := json.Unmarshal([]byte(manifest), &inputs); checkErr != nil {
		return fmt.Errorf("decode source manifest: %w", checkErr)
	}
	collection, err := s.collectSources(ctx, run, inputs)
	if err != nil {
		return err
	}
	if checkErr := s.saveSources(ctx, run.ID, collection); checkErr != nil {
		return checkErr
	}
	contextText := source.Context(collection)
	requirements, promptErr := sectionRequirements()
	if promptErr != nil {
		return promptErr
	}
	if checkErr := s.stage(ctx, run.ID, "analyzing"); checkErr != nil {
		return checkErr
	}
	analysis, err := generate[passport.Model](ctx, s, run, "analyzing", analysisPrompt+"\n"+architectureResearchPrompt+"\nSECTION REQUIREMENTS:\n"+requirements+"\nSOURCES:\n"+contextText)
	if err != nil {
		return err
	}
	if checkErr := s.stage(ctx, run.ID, "synthesizing"); checkErr != nil {
		return checkErr
	}
	analysisJSON, _ := json.Marshal(analysis)
	model, err := generate[passport.Model](ctx, s, run, "synthesizing", reconcilePrompt+"\nANALYSIS:\n"+string(analysisJSON)+"\nSOURCES:\n"+contextText)
	if err != nil {
		return err
	}
	if checkErr := s.stage(ctx, run.ID, "composing"); checkErr != nil {
		return checkErr
	}
	modelJSON, _ := json.Marshal(model)
	views, err := generate[passport.Views](ctx, s, run, "composing", compositionPrompt+"\nSECTION REQUIREMENTS:\n"+requirements+"\nMODEL:\n"+string(modelJSON))
	if err != nil {
		return err
	}
	p := passport.Passport{Model: model, Views: views}
	if checkErr := s.stage(ctx, run.ID, "validating"); checkErr != nil {
		return checkErr
	}
	documents := make(map[string]int)
	for _, doc := range collection.Documents {
		documents[doc.ID] = doc.LineCount
	}
	if validationErr := passport.Validate(p, documents); validationErr != nil {
		encoded, _ := json.Marshal(p)
		p, err = generate[passport.Passport](ctx, s, run, "repairing", repairPrompt+"\nSECTION REQUIREMENTS:\n"+requirements+"\nERROR:\n"+validationErr.Error()+"\nPASSPORT:\n"+string(encoded)+"\nSOURCES:\n"+contextText)
		if err != nil {
			return err
		}
		if checkErr := passport.Validate(p, documents); checkErr != nil {
			return fmt.Errorf("passport validation failed: %w", checkErr)
		}
	}
	return s.saveRevision(ctx, run, p)
}

func generate[T any](ctx context.Context, s *Service, run Run, stage, input string) (T, error) {
	var empty T
	schema, err := passport.Schema[T]()
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
	for attempt := 1; attempt <= 2; attempt++ {
		config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
		if err != nil {
			return empty, errors.New("the selected connection is disabled or unavailable")
		}
		started := time.Now()
		result, callErr := s.LLM.Generate(ctx, config, llm.Request{Model: run.Model, System: systemPrompt, Input: input, Schema: format})
		code := ""
		if callErr != nil {
			code = "provider_error"
			if failure, ok := errors.AsType[*llm.ProviderError](callErr); ok {
				code = failure.Code
			}
		}
		_, saveErr := s.DB.ExecContext(ctx, "INSERT INTO llm_calls(id,run_id,stage,attempt,prompt_version,request_id,input_tokens,output_tokens,duration_ms,output_json,error_code,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", rand.Text(), run.ID, stage, attempt, PromptVersion, result.RequestID, result.InputTokens, result.OutputTokens, time.Since(started).Milliseconds(), string(result.JSON), code, time.Now().UTC().Format(time.RFC3339Nano))
		if saveErr != nil {
			return empty, fmt.Errorf("save model call: %w", saveErr)
		}
		if callErr != nil {
			return empty, fmt.Errorf("generate %s: %w", stage, callErr)
		}
		value, validationErr := passport.CheckJSON[T](result.JSON)
		if validationErr == nil {
			return value, nil
		}
		if attempt == 2 {
			return empty, fmt.Errorf("model response failed schema validation: %w", validationErr)
		}
		input += "\n" + schemaRepairPrompt + "\nERROR:\n" + validationErr.Error() + "\nPREVIOUS RESPONSE:\n" + string(result.JSON)
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
		if _, err := tx.ExecContext(ctx, "INSERT INTO source_documents(id,snapshot_id,path,content_hash,content,line_count) VALUES(?,?,?,?,?,?)", doc.ID, doc.SnapshotID, doc.Path, doc.ContentHash, doc.Content, doc.LineCount); err != nil {
			return fmt.Errorf("save source document: %w", err)
		}
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return fmt.Errorf("commit source snapshot: %w", checkErr)
	}
	return nil
}

func (s *Service) saveRevision(ctx context.Context, run Run, p passport.Passport) error {
	encoded, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("encode passport: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin passport revision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, "INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,created_at) SELECT ?,?,?,COALESCE(MAX(version),0)+1,'1',?,? FROM passport_revisions WHERE product_id=?", rand.Text(), run.ProductID, run.ID, string(encoded), time.Now().UTC().Format(time.RFC3339Nano), run.ProductID)
	if err != nil {
		return fmt.Errorf("save passport revision: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE generation_runs SET status='succeeded',stage='complete',finished_at=? WHERE id=?", time.Now().UTC().Format(time.RFC3339Nano), run.ID); err != nil {
		return fmt.Errorf("complete generation: %w", err)
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return fmt.Errorf("commit passport revision: %w", checkErr)
	}
	return nil
}

// History returns compact revision metadata.
func (s *Service) History(ctx context.Context, productID string) ([]RevisionInfo, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,version,created_at FROM passport_revisions WHERE product_id=? ORDER BY version DESC", productID)
	if err != nil {
		return nil, fmt.Errorf("list passport revisions: %w", err)
	}
	defer rows.Close()
	result := make([]RevisionInfo, 0)
	for rows.Next() {
		var r RevisionInfo
		if checkErr := rows.Scan(&r.ID, &r.Version, &r.CreatedAt); checkErr != nil {
			return nil, fmt.Errorf("read revision history: %w", checkErr)
		}
		result = append(result, r)
	}
	if checkErr := rows.Err(); checkErr != nil {
		return nil, fmt.Errorf("read revision history: %w", checkErr)
	}
	return result, nil
}

// Revision returns one draft and its exact snapshot manifest.
func (s *Service) Revision(ctx context.Context, id string) (Revision, error) {
	var r Revision
	var content string
	err := s.DB.QueryRowContext(ctx, "SELECT id,product_id,run_id,version,schema_version,content_json,created_at FROM passport_revisions WHERE id=?", id).Scan(&r.ID, &r.ProductID, &r.RunID, &r.Version, &r.SchemaVersion, &content, &r.CreatedAt)
	if err != nil {
		return r, fmt.Errorf("get passport revision: %w", err)
	}
	if checkErr := json.Unmarshal([]byte(content), &r.Content); checkErr != nil {
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
	var content string
	err = s.DB.QueryRowContext(ctx, "SELECT d.path,d.content,ss.locator,ss.revision FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE d.id=? AND ss.run_id=?", anchor.DocumentID, r.RunID).Scan(&view.Path, &content, &view.Locator, &view.Revision)
	if err != nil {
		return view, fmt.Errorf("get evidence document: %w", err)
	}
	lines := strings.Split(content, "\n")
	if anchor.StartLine < 1 || anchor.EndLine > len(lines) {
		return view, errors.New("stored evidence anchor is invalid")
	}
	view.Excerpt = strings.Join(lines[anchor.StartLine-1:anchor.EndLine], "\n")
	if strings.HasPrefix(view.Locator, "https://github.com/") {
		view.SourceURL = strings.TrimSuffix(view.Locator, ".git") + "/blob/" + view.Revision + "/" + view.Path + fmt.Sprintf("#L%d-L%d", view.StartLine, view.EndLine)
	}
	return view, nil
}
