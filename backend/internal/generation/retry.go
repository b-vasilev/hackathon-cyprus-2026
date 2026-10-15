package generation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/productcard"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"maragu.dev/goqite"
)

func validateRequestTimeout(seconds int) error {
	if seconds != 0 && (seconds < 20 || seconds > 900) {
		return fmt.Errorf("%w: request_timeout_seconds must be 0 or between 20 and 900", llmgateway.ErrInvalid)
	}
	return nil
}

// resume requeues the same failed run without rewriting its frozen inputs.
func (s *Service) resume(ctx context.Context, productID string, input Input) (Run, error) {
	run, err := s.GetRun(ctx, input.ResumeRunID)
	if err != nil {
		return Run{}, fmt.Errorf("load generation to retry: %w", err)
	}
	if run.ProductID != productID {
		return Run{}, fmt.Errorf("%w: generation does not belong to this product", llmgateway.ErrInvalid)
	}
	if run.Status != "failed" {
		return Run{}, fmt.Errorf("%w: only failed generations can be retried", llmgateway.ErrInvalid)
	}
	if _, active := s.active.Load(run.ID); active {
		return Run{}, ErrActive
	}
	if input.ConnectionID != run.ConnectionID || input.Model != run.Model {
		return Run{}, fmt.Errorf("%w: retry must use the original connection and model", llmgateway.ErrInvalid)
	}
	// Load the recorded revision, including its original model configuration.
	// Current connection edits must never silently replace the frozen revision.
	config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
	if err != nil {
		return Run{}, fmt.Errorf("load original generation connection: %w", err)
	}
	if _, modelErr := llmgateway.ModelByID(config, run.Model); modelErr != nil {
		return Run{}, fmt.Errorf("load original generation model: %w", modelErr)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Run{}, fmt.Errorf("begin generation retry: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var enabledConnection string
	if checkErr := tx.QueryRowContext(ctx, "SELECT id FROM llm_connections WHERE id=? AND enabled=1 AND deleted=0", run.ConnectionID).Scan(&enabledConnection); checkErr != nil {
		return Run{}, fmt.Errorf("check retry connection: %w", checkErr)
	}
	result, err := tx.ExecContext(ctx, "UPDATE generation_runs SET status='queued',stage='queued',stage_started_at='',error_code='',error_message='',finished_at=NULL,request_timeout_seconds=? WHERE id=? AND product_id=? AND status='failed'", input.RequestTimeoutSeconds, run.ID, productID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: generation_runs.product_id") {
			return Run{}, ErrActive
		}
		return Run{}, fmt.Errorf("requeue generation: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Run{}, fmt.Errorf("check generation retry: %w", err)
	}
	if changed != 1 {
		return Run{}, fmt.Errorf("%w: generation is no longer failed", llmgateway.ErrInvalid)
	}
	// Replace stale deliveries, including messages left by interrupted workers.
	if _, checkErr := tx.ExecContext(ctx, "DELETE FROM goqite WHERE queue='passport' AND body=?", []byte(run.ID)); checkErr != nil {
		return Run{}, fmt.Errorf("remove stale generation deliveries: %w", checkErr)
	}
	if checkErr := s.queue.SendTx(ctx, tx, goqite.Message{Body: []byte(run.ID)}); checkErr != nil {
		return Run{}, fmt.Errorf("enqueue generation retry: %w", checkErr)
	}
	previousTimeout, requestTimeout := run.RequestTimeoutSeconds, input.RequestTimeoutSeconds
	if previousTimeout == 0 {
		previousTimeout = config.TimeoutSeconds
	}
	if requestTimeout == 0 {
		requestTimeout = config.TimeoutSeconds
	}
	message := fmt.Sprintf("Retry queued with the original source manifest, connection revision and model. Request timeout: %d -> %d seconds.", previousTimeout, requestTimeout)
	if checkErr := recordEvent(ctx, tx, run.ID, "run_resumed", "queued", message); checkErr != nil {
		return Run{}, checkErr
	}
	if checkErr := tx.Commit(); checkErr != nil {
		return Run{}, fmt.Errorf("commit generation retry: %w", checkErr)
	}
	return s.GetRun(ctx, run.ID)
}

// claimRun makes duplicate queue deliveries harmless, even with two receivers.
func (s *Service) claimRun(ctx context.Context, runID string) (bool, error) {
	result, err := s.DB.ExecContext(ctx, "UPDATE generation_runs SET status='running' WHERE id=? AND status='queued'", runID)
	if err != nil {
		return false, fmt.Errorf("claim queued generation: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check generation claim: %w", err)
	}
	return changed == 1, nil
}

func (s *Service) collectOrLoadSources(ctx context.Context, run Run, inputs []source.Input, repositories []string) (source.Collection, bool, error) {
	if run.DocumentCount > 0 {
		collection, err := s.loadSavedSources(ctx, run.ID)
		if err != nil {
			return source.Collection{}, false, err
		}
		if len(collection.Documents) != run.DocumentCount {
			return source.Collection{}, false, errors.New("saved source document count changed; collection was not repeated")
		}
		if checkErr := s.event(ctx, run.ID, "sources_reused", "collecting", fmt.Sprintf("Reused %d saved source documents after checksum verification.", len(collection.Documents))); checkErr != nil {
			return source.Collection{}, false, checkErr
		}
		return collection, true, nil
	}
	var collection source.Collection
	var err error
	selected := false
	if s.Research != nil {
		collection, err = s.Collector.Collect(ctx, inputs)
	} else if collector, ok := s.Collector.(selectingCollector); ok {
		collection, err = collector.CollectSelected(ctx, inputs, func(selectCtx context.Context, catalog string) ([]string, error) {
			selected = true
			if stageErr := s.stage(selectCtx, run.ID, "selecting_sources"); stageErr != nil {
				return nil, stageErr
			}
			selection, selectErr := generate[source.FileSelection](selectCtx, s, run, "selecting_sources", catalog+"\nPRODUCT REPOSITORIES:\n"+strings.Join(repositories, "\n")+"\n"+selectionPrompt)
			return selection.Paths, selectErr
		})
	} else {
		collection, err = s.Collector.Collect(ctx, inputs)
	}
	if err != nil {
		return collection, selected, fmt.Errorf("collect generation sources: %w", err)
	}
	if checkErr := s.saveSources(ctx, run.ID, collection); checkErr != nil {
		return collection, selected, checkErr
	}
	return collection, selected, nil
}

// loadSavedSources restores pinned metadata and original bodies without a collector.
// Research continuation may reuse this collection; it must persist new attempts
// separately rather than replace immutable findings from an earlier attempt.
func (s *Service) loadSavedSources(ctx context.Context, runID string) (source.Collection, error) {
	collection := source.Collection{}
	rows, err := s.DB.QueryContext(ctx, "SELECT id,locator,revision,coverage_json FROM source_snapshots WHERE run_id=? ORDER BY rowid", runID)
	if err != nil {
		return collection, fmt.Errorf("load saved source snapshots: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var snapshot source.Snapshot
		var coverage string
		if scanErr := rows.Scan(&snapshot.ID, &snapshot.Locator, &snapshot.Revision, &coverage); scanErr != nil {
			_ = rows.Close()
			return collection, fmt.Errorf("read saved source snapshot: %w", scanErr)
		}
		if decodeErr := json.Unmarshal([]byte(coverage), &snapshot.Coverage); decodeErr != nil {
			_ = rows.Close()
			return collection, fmt.Errorf("decode saved source coverage: %w", decodeErr)
		}
		collection.Snapshots = append(collection.Snapshots, snapshot)
	}
	if readErr := errors.Join(rows.Err(), rows.Close()); readErr != nil {
		return collection, fmt.Errorf("read saved source snapshots: %w", readErr)
	}
	rows, err = s.DB.QueryContext(ctx, "SELECT d.id,d.snapshot_id,d.path,d.content_hash,d.content,d.line_count,d.content_path FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE ss.run_id=? ORDER BY d.rowid", runID)
	if err != nil {
		return collection, fmt.Errorf("load saved source documents: %w", err)
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var document source.Document
		var path string
		if checkErr := rows.Scan(&document.ID, &document.SnapshotID, &document.Path, &document.ContentHash, &document.Content, &document.LineCount, &path); checkErr != nil {
			_ = rows.Close()
			return collection, fmt.Errorf("read saved source document: %w", checkErr)
		}
		collection.Documents = append(collection.Documents, document)
		paths = append(paths, path)
	}
	if checkErr := errors.Join(rows.Err(), rows.Close()); checkErr != nil {
		return collection, fmt.Errorf("read saved source documents: %w", checkErr)
	}
	for index := range collection.Documents {
		document := &collection.Documents[index]
		if paths[index] != "" {
			data, readErr := s.Content.Read(paths[index], document.ContentHash)
			if readErr != nil {
				return collection, fmt.Errorf("verify saved source %s: %w", document.ID, readErr)
			}
			document.Content = string(data)
		} else {
			hash := sha256.Sum256([]byte(document.Content))
			if hex.EncodeToString(hash[:]) != document.ContentHash {
				return collection, fmt.Errorf("saved source %s checksum mismatch", document.ID)
			}
		}
		if len(strings.Split(document.Content, "\n")) != document.LineCount {
			return collection, fmt.Errorf("saved source %s line count mismatch", document.ID)
		}
	}
	return collection, nil
}

func (s *Service) collectOrLoadBrand(ctx context.Context, run Run, collection source.Collection, manual *productcard.Brandkit) (productcard.Discovery, error) {
	path := "runs/" + run.ID + "/product-brand.json"
	var brand productcard.Discovery
	root, err := os.OpenRoot(s.Content.Dir)
	if err == nil {
		defer root.Close()
		var data []byte
		data, err = root.ReadFile(path)
		if err == nil {
			if decodeErr := json.Unmarshal(data, &brand); decodeErr != nil {
				return brand, fmt.Errorf("decode saved product branding: %w", decodeErr)
			}
			return brand, s.event(ctx, run.ID, "brand_sources_reused", "collecting", "Reused saved product branding.")
		}
	}
	if !errors.Is(err, os.ErrNotExist) {
		return brand, fmt.Errorf("load saved product branding: %w", err)
	}
	if run.DocumentCount > 0 {
		return brand, errors.New("saved product branding is missing; start a new generation rather than change this run's frozen branding")
	}
	var readAsset productcard.AssetReader
	if collector, ok := s.Collector.(interface {
		ReadBrandAsset(context.Context, source.Snapshot, string) ([]byte, error)
	}); ok {
		readAsset = collector.ReadBrandAsset
	}
	brand = productcard.Discover(ctx, collection, manual, readAsset)
	if _, checkErr := s.Content.WriteJSON(path, brand); checkErr != nil {
		return brand, fmt.Errorf("save product brand sources: %w", checkErr)
	}
	return brand, s.event(ctx, run.ID, "brand_sources_collected", "collecting", fmt.Sprintf("Product branding: %d references, %d verified logos, %d unavailable resources.", len(brand.References), len(brand.Logos), len(brand.Notes)))
}
