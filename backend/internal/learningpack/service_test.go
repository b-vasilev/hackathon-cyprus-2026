package learningpack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/rs/zerolog"
)

type generatorFunc func(context.Context, llmgateway.Config, llm.Request) (llm.Result, error)

func (f generatorFunc) Generate(ctx context.Context, config llmgateway.Config, request llm.Request) (llm.Result, error) {
	return f(ctx, config, request)
}

func generationFixture(t *testing.T) (*Service, generation.Revision) {
	t.Helper()
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "practice.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	connections, err := llmgateway.NewStore(database, strings.Repeat("master", 8))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := connections.Save(ctx, "", llmgateway.Config{
		Name: "Fixture", Enabled: true, Protocol: "openai_responses", EndpointURL: "https://example.com/v1",
		EndpointMode: "base", AuthMode: "none", Models: []llmgateway.Model{{ID: "model", OutputMode: "json_schema", MaxOutputTokens: 16000}},
		TimeoutSeconds: 30, MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	files := &content.Store{Dir: t.TempDir()}
	draft := passport.Passport{Views: passport.Views{Document: &passport.Document{
		Version: 2, Chapters: []passport.Chapter{{ID: "overview", Title: "Overview", Question: "How to run?",
			Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "The service listens on port 8080."}}}},
	}}}
	hash, err := files.WriteJSON("passport.json", draft)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO users VALUES('reviewer','reviewer@example.com','hash','admin','created')`,
		`INSERT INTO products(id,name,sources_json,created_at) VALUES('product','Product','[]','created')`,
	} {
		if _, execErr := database.ExecContext(ctx, statement); execErr != nil {
			t.Fatal(execErr)
		}
	}
	if _, execErr := database.ExecContext(ctx, `INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at)
 VALUES('run','product',?,1,'model','succeeded','complete','{}','created')`, connection.ID); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := database.ExecContext(ctx, `INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_path,content_hash,created_at)
 VALUES('revision','product','run',1,'2','{}','passport.json',?,'created')`, hash); execErr != nil {
		t.Fatal(execErr)
	}
	if _, execErr := database.ExecContext(ctx, `INSERT INTO learning_packs(id,product_id,revision_id,status,content_json,source_content_hash,created_at,updated_at)
 VALUES('pack','product','revision','generating','{"mission":{"title":"Previous pack"}}',?,'created','created')`, hash); execErr != nil {
		t.Fatal(execErr)
	}
	factory := generation.New(database, connections, nil, nil, zerolog.Nop(), files)
	revision, err := factory.Revision(ctx, "revision")
	if err != nil {
		t.Fatal(err)
	}
	return &Service{DB: database, Passports: factory, Background: ctx, Logger: zerolog.Nop()}, revision
}

func TestGenerationKeepsItsSourceWhenPassportChanges(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failed=%t", fail), func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			service, revision := generationFixture(t)
			factory := service.Passports
			factory.LLM = generatorFunc(func(callCtx context.Context, _ llmgateway.Config, request llm.Request) (llm.Result, error) {
				if !strings.Contains(request.Input, "8080") {
					t.Fatal("generation did not receive its original passport snapshot")
				}
				if _, err := factory.EditChapter(callCtx, "revision", "overview", "reviewer", passport.ChapterEdit{
					Title: "Overview", Markdown: []string{"The service listens on port 9090."},
				}); err != nil {
					t.Fatal(err)
				}
				if fail {
					return llm.Result{}, errors.New("provider unavailable")
				}
				answer := validPack()
				answer.Mission.IssueNumber = 0
				encoded, marshalErr := json.Marshal(answer)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				return llm.Result{JSON: encoded}, nil
			})
			service.run("", revision)
			view, err := service.Get(ctx, "revision")
			if err != nil {
				t.Fatal(err)
			}
			wantStatus, wantTitle := statusReady, "First task"
			if fail {
				wantStatus, wantTitle = statusFailed, "Previous pack"
			}
			if !view.Stale || view.Status != wantStatus || view.Content == nil || view.Content.Mission.Title != wantTitle {
				t.Fatalf("generation lost its original source or working copy: %+v", view)
			}
			var savedHash string
			if scanErr := service.DB.QueryRowContext(ctx, "SELECT source_content_hash FROM learning_packs WHERE id='pack'").Scan(&savedHash); scanErr != nil {
				t.Fatal(scanErr)
			}
			if savedHash != revision.ContentHash || savedHash == "" {
				t.Fatalf("generation was rebound to edited content: saved=%q original=%q", savedHash, revision.ContentHash)
			}
			if _, err := service.Publish(ctx, "revision", "reviewer"); !errors.Is(err, ErrNotReady) {
				t.Fatalf("outdated practice could be published: %v", err)
			}
		})
	}
}
