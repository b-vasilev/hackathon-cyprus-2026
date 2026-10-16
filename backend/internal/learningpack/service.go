package learningpack

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/rs/zerolog"
)

const (
	maxAttempts    = 3
	maxOutput      = 16000
	minimumTimeout = 300
	statusNone     = "none"
	statusRunning  = "generating"
	statusReady    = "ready"
	statusFailed   = "failed"
)

// ErrNotReady means there is no finished pack to publish yet.
var ErrNotReady = errors.New("generate the learning pack before publishing it")

// View is a pack with its review state. Reviewers see the working copy; learners the published copy.
type View struct {
	ID                 string `json:"id"`
	ProductID          string `json:"product_id"`
	RevisionID         string `json:"revision_id"`
	RevisionVersion    int    `json:"revision_version"`
	Status             string `json:"status" jsonschema:"none, generating, ready or failed."`
	Error              string `json:"error"`
	UpdatedAt          string `json:"updated_at"`
	Published          bool   `json:"published"`
	PublishedAt        string `json:"published_at"`
	UnpublishedChanges bool   `json:"unpublished_changes"`
	Stale              bool   `json:"stale" jsonschema:"Practice was generated from different passport content and must be generated again."`
	Content            *Pack  `json:"content,omitempty"`
}

// Service generates and publishes learning packs. Generation runs in the background
// because one model call can outlast an HTTP request.
type Service struct {
	DB         *sql.DB
	Passports  *generation.Service
	Token      func(context.Context) (string, error)
	HTTP       *http.Client
	GitHubBase string
	Logger     zerolog.Logger
	Background context.Context
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Recover marks packs interrupted by a restart as failed so they can be generated again.
func (s *Service) Recover(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE learning_packs SET status=?, error_message=?, updated_at=? WHERE status=?", statusFailed, "Generation was interrupted by a restart. Generate the pack again.", now(), statusRunning)
	if err != nil {
		return fmt.Errorf("recover learning packs: %w", err)
	}
	return nil
}

// Get returns the working copy for a passport version, or status none.
func (s *Service) Get(ctx context.Context, revisionID string) (View, error) {
	view, err := s.read(ctx, "p.revision_id=?", revisionID, false)
	if errors.Is(err, sql.ErrNoRows) {
		return View{RevisionID: revisionID, Status: statusNone}, nil
	}
	return view, err
}

// Published returns only the practice for the product's published passport.
func (s *Service) Published(ctx context.Context, productID string) (View, error) {
	return s.read(ctx, `p.product_id=? AND p.revision_id=(SELECT published_revision_id FROM products WHERE id=p.product_id)
 AND p.published_at<>'' AND p.published_source_content_hash<>''
 AND p.published_source_content_hash=COALESCE(NULLIF(r.published_content_hash,''),r.content_hash)`, productID, true)
}

func (s *Service) read(ctx context.Context, where, arg string, published bool) (View, error) {
	var v View
	var working, learner, sourceHash, publishedHash, currentHash string
	err := s.DB.QueryRowContext(ctx, "SELECT p.id,p.product_id,p.revision_id,r.version,p.status,p.error_message,p.updated_at,p.published_at,p.content_json,p.published_content_json,p.source_content_hash,p.published_source_content_hash,r.content_hash FROM learning_packs p JOIN passport_revisions r ON r.id=p.revision_id WHERE "+where, arg).
		Scan(&v.ID, &v.ProductID, &v.RevisionID, &v.RevisionVersion, &v.Status, &v.Error, &v.UpdatedAt, &v.PublishedAt, &working, &learner, &sourceHash, &publishedHash, &currentHash)
	if err != nil {
		return View{}, fmt.Errorf("read learning pack: %w", err)
	}
	v.Published = v.PublishedAt != ""
	v.UnpublishedChanges = v.Published && (working != learner || sourceHash != publishedHash)
	v.Stale = working != "" && (sourceHash == "" || sourceHash != currentHash)
	content := working
	if published {
		content, v.Status, v.Error = learner, statusReady, ""
		v.Stale = false
	}
	if content != "" {
		var pack Pack
		if err := json.Unmarshal([]byte(content), &pack); err != nil {
			return View{}, fmt.Errorf("decode learning pack: %w", err)
		}
		v.Content = &pack
	}
	return v, nil
}

// Start generates a pack for a passport version. The previous pack stays visible until
// the new one is ready, and learners keep the published copy until a lead republishes.
func (s *Service) Start(ctx context.Context, revisionID, userID string) (View, error) {
	revision, err := s.Passports.Revision(ctx, revisionID)
	if err != nil {
		return View{}, fmt.Errorf("load passport version: %w", err)
	}
	if _, _, connectionErr := s.connection(ctx, revision.RunID); connectionErr != nil {
		return View{}, connectionErr
	}
	stamp := now()
	result, err := s.DB.ExecContext(ctx, `INSERT INTO learning_packs(id,product_id,revision_id,status,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(revision_id) DO UPDATE SET status=excluded.status, error_message='', created_by=excluded.created_by, updated_at=excluded.updated_at WHERE learning_packs.status<>?`,
		rand.Text(), revision.ProductID, revisionID, statusRunning, userID, stamp, stamp, statusRunning)
	if err != nil {
		return View{}, fmt.Errorf("start learning pack: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return View{}, fmt.Errorf("%w: the learning pack is already being generated", generation.ErrActive)
	}
	go s.run(connectionowner.UserID(ctx), revision)
	return s.Get(ctx, revisionID)
}

func (s *Service) run(ownerID string, revision generation.Revision) {
	ctx, cancel := context.WithTimeout(connectionowner.WithUser(s.Background, ownerID), 20*time.Minute)
	defer cancel()
	pack, err := s.generate(ctx, revision)
	status, message, content := statusReady, "", ""
	if err == nil {
		data, _ := json.Marshal(pack)
		content = string(data)
	} else {
		status, message = statusFailed, err.Error()
		s.Logger.Warn().Err(err).Str("revision_id", revision.ID).Msg("learning pack generation failed")
	}
	saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer saveCancel()
	query := `UPDATE learning_packs SET status=?, error_message=?, updated_at=?,
 content_json=CASE WHEN ?='' THEN content_json ELSE ? END,
 source_content_hash=CASE WHEN ?='' THEN source_content_hash ELSE ? END WHERE revision_id=?`
	if _, err := s.DB.ExecContext(saveCtx, query, status, message, now(), content, content, content, revision.ContentHash, revision.ID); err != nil {
		s.Logger.Error().Err(err).Str("revision_id", revision.ID).Msg("save learning pack")
	}
}

// Publish shows the current working copy to learners.
func (s *Service) Publish(ctx context.Context, revisionID, userID string) (View, error) {
	result, err := s.DB.ExecContext(ctx, `UPDATE learning_packs SET published_content_json=content_json, published_source_content_hash=source_content_hash, published_by=?,
 published_at=CASE WHEN published_content_json=content_json AND published_source_content_hash=source_content_hash AND published_at<>'' THEN published_at ELSE ? END
 WHERE revision_id=? AND content_json<>'' AND status<>? AND source_content_hash<>''
 AND source_content_hash=(SELECT content_hash FROM passport_revisions WHERE id=revision_id)
 AND (NOT EXISTS(SELECT 1 FROM products WHERE published_revision_id=revision_id)
 OR source_content_hash=(SELECT COALESCE(NULLIF(published_content_hash,''),content_hash) FROM passport_revisions WHERE id=revision_id))`, userID, now(), revisionID, statusRunning)
	if err != nil {
		return View{}, fmt.Errorf("publish learning pack: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return View{}, ErrNotReady
	}
	return s.Get(ctx, revisionID)
}

// Unpublish hides the pack from learners and keeps the working copy.
func (s *Service) Unpublish(ctx context.Context, revisionID string) (View, error) {
	if _, err := s.DB.ExecContext(ctx, "UPDATE learning_packs SET published_content_json='', published_source_content_hash='', published_by='', published_at='' WHERE revision_id=?", revisionID); err != nil {
		return View{}, fmt.Errorf("unpublish learning pack: %w", err)
	}
	return s.Get(ctx, revisionID)
}

func (s *Service) generate(ctx context.Context, revision generation.Revision) (Pack, error) {
	config, model, err := s.connection(ctx, revision.RunID)
	if err != nil {
		return Pack{}, err
	}
	config.TimeoutSeconds = max(config.TimeoutSeconds, minimumTimeout)
	candidates, repository, err := s.issueCandidates(ctx, revision)
	if err != nil {
		return Pack{}, err
	}
	chapters, outline := chapterOutline(revision.Content)
	paths, err := s.codePaths(ctx, revision)
	if err != nil {
		return Pack{}, err
	}
	input, err := promptInput(revision.Content, outline, paths, repository, candidates)
	if err != nil {
		return Pack{}, err
	}
	schema, err := passport.OutputSchema[generated]()
	if err != nil {
		return Pack{}, err
	}
	var format map[string]any
	encoded, _ := json.Marshal(schema)
	if err := json.Unmarshal(encoded, &format); err != nil {
		return Pack{}, fmt.Errorf("encode learning pack schema: %w", err)
	}
	issues := make([]Issue, len(candidates))
	for i, candidate := range candidates {
		issues[i] = candidate.Issue
	}
	request := llm.Request{Model: model, System: systemPrompt, Input: input, Schema: format, MaxOutputTokens: maxOutput}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := s.Passports.LLM.Generate(ctx, config, request)
		var answer generated
		switch {
		case err != nil:
			var failure *llm.ProviderError
			if !errors.As(err, &failure) || (failure.Code != "truncated" && failure.Code != "invalid_json" && failure.Code != "timeout") {
				return Pack{}, fmt.Errorf("model request failed: %w", err)
			}
			lastErr = err
		default:
			answer, lastErr = passport.CheckJSON[generated](result.JSON)
			if lastErr == nil {
				lastErr = validate(answer, chapters, issues)
			}
			if lastErr == nil {
				return answer.pack(issues), nil
			}
		}
		request.Input = input + "\n\nYOUR PREVIOUS ANSWER WAS REJECTED. Fix these problems and return the complete pack again:\n" + lastErr.Error()
	}
	return Pack{}, fmt.Errorf("the model did not return a valid learning pack after %d attempts: %w", maxAttempts, lastErr)
}

// connection prefers the current connection settings and falls back to the exact
// revision that generated the passport when the model was removed since.
func (s *Service) connection(ctx context.Context, runID string) (llmgateway.Config, string, error) {
	var connectionID, model string
	var frozen int
	if err := s.DB.QueryRowContext(ctx, "SELECT connection_id, connection_revision, model FROM generation_runs WHERE id=?", runID).Scan(&connectionID, &frozen, &model); err != nil {
		return llmgateway.Config{}, "", fmt.Errorf("find the passport's model: %w", err)
	}
	if s.Passports.Connections == nil {
		return llmgateway.Config{}, "", errors.New("LLM connections are not configured")
	}
	if current, err := s.Passports.Connections.Get(ctx, connectionID); err == nil {
		if config, err := s.Passports.Connections.Load(ctx, connectionID, current.Revision); err == nil {
			if _, err := llmgateway.ModelByID(config, model); err == nil {
				return config, model, nil
			}
		}
	}
	config, err := s.Passports.Connections.Load(ctx, connectionID, frozen)
	if err != nil {
		return llmgateway.Config{}, "", fmt.Errorf("the LLM connection that generated this passport (%s) is disabled or unavailable", model)
	}
	return config, model, nil
}

func (s *Service) issueCandidates(ctx context.Context, revision generation.Revision) ([]Candidate, string, error) {
	token := ""
	if s.Token != nil {
		var err error
		if token, err = s.Token(ctx); err != nil {
			return nil, "", fmt.Errorf("read GitHub token: %w", err)
		}
	}
	client := githubClient{http: s.HTTP, token: token, base: s.GitHubBase}
	if client.http == nil {
		client.http = &http.Client{Timeout: 30 * time.Second}
	}
	if client.base == "" {
		client.base = githubAPI
	}
	for _, snapshot := range revision.Snapshots {
		repository, ok := repositoryName(snapshot.Locator)
		if !ok {
			continue
		}
		candidates, err := client.candidates(ctx, repository)
		if err != nil {
			return nil, repository, fmt.Errorf("read closed issues of %s: %w", repository, err)
		}
		return candidates, repository, nil
	}
	return nil, "", nil
}

// codePaths lists the files the passport cites, so steps can point to real code.
func (s *Service) codePaths(ctx context.Context, revision generation.Revision) ([]string, error) {
	cited := map[string]bool{}
	for _, evidence := range revision.Content.Model.Evidence {
		cited[evidence.DocumentID] = true
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT d.id,d.path FROM source_documents d JOIN source_snapshots ss ON ss.id=d.snapshot_id WHERE ss.run_id=? ORDER BY d.path", revision.RunID)
	if err != nil {
		return nil, fmt.Errorf("read cited files: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var paths []string
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, fmt.Errorf("read cited file: %w", err)
		}
		if cited[id] {
			paths = append(paths, path)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read cited files: %w", err)
	}
	return paths, nil
}

type outlineChapter struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Question string `json:"question,omitempty"`
	Text     string `json:"text"`
}

func chapterOutline(content passport.Passport) ([]string, []outlineChapter) {
	var ids []string
	var outline []outlineChapter
	if content.Views.Document != nil {
		for _, chapter := range content.Views.Document.Chapters {
			var text []string
			for _, block := range chapter.Blocks {
				if block.Kind == "markdown" {
					text = append(text, block.Markdown)
				}
			}
			ids = append(ids, chapter.ID)
			outline = append(outline, outlineChapter{ID: chapter.ID, Title: chapter.Title, Question: chapter.Question, Text: strings.Join(text, "\n\n")})
		}
		return ids, outline
	}
	for _, section := range content.Views.Sections {
		ids = append(ids, section.ID)
		outline = append(outline, outlineChapter{ID: section.ID, Title: section.Title, Text: section.Summary + "\n" + strings.Join(section.Checklist, "\n")})
	}
	return ids, outline
}

func promptInput(content passport.Passport, outline []outlineChapter, paths []string, repository string, candidates []Candidate) (string, error) {
	type object struct {
		ID, Kind, Name, Summary, Technology string
	}
	objects := make([]object, 0, len(content.Model.Objects))
	for _, o := range content.Model.Objects {
		objects = append(objects, object{o.ID, o.Kind, o.Name, o.Summary, o.Technology})
	}
	claims := make([]string, 0, len(content.Model.Claims))
	for _, claim := range content.Model.Claims {
		claims = append(claims, claim.Text)
	}
	data, err := json.MarshalIndent(map[string]any{
		"product":           map[string]string{"title": content.Model.Title, "tagline": content.Model.Tagline, "summary": content.Model.Summary},
		"repository":        repository,
		"chapters":          outline,
		"objects":           objects,
		"facts":             claims,
		"open_questions":    content.Model.OpenQuestions,
		"cited_files":       paths,
		"candidate_issues":  candidates,
		"candidate_comment": "Closed issues resolved by merged pull requests, smallest change first. Empty means none were found.",
	}, "", " ")
	if err != nil {
		return "", fmt.Errorf("encode learning pack input: %w", err)
	}
	return "PASSPORT AND CANDIDATE ISSUES:\n" + string(data), nil
}

const systemPrompt = `You write onboarding practice for engineers who join a software product team. Many of them are moving from consulting into product engineering, so explain product-specific terms briefly and keep language plain.

You receive a reviewed product passport (chapters, objects, facts, cited files) and candidate closed GitHub issues. Return JSON matching the schema.

QUIZ: six to eight questions that check a newcomer understood the product: its purpose and users, main flows, architecture and where code lives, how to run and check a change, and known limits. Every answer must follow from the passport; never invent facts, names, commands or paths. Prefer questions that need understanding over recall of trivia. Make wrong options plausible. Spread questions across chapters and set chapter_id to the chapter that teaches the answer. Use kind multiple only when two or more options are truly correct, and say so in the prompt ("Select all that apply").

MISSION: a hands-on first task that ends with the learner ready to pick their own issue. Choose ONE candidate issue that teaches the most about how this product changes: prefer a bug fix or a small feature in product code that touches tests, over chores, dependency or model version bumps and pure documentation. Then write five to nine steps in this order:
1. Clone the repository (git clone https://github.com/<repository>.git) and open it.
2. Set up the environment and run the checks, using only commands the passport documents. If the passport says setup is undocumented, say so and tell the learner to ask the team, instead of guessing.
3. Find the code: name the folders and files from the passport (cited_files and chapters) that own the area of the chosen issue.
4. Read the issue: what was wrong or missing and how the reporter described it.
5. Study the merged pull request: which files changed (use the candidate's files list) and how the change was checked.
6. Optionally reproduce or rerun the relevant check locally.
7. Reflect: what you would do the same way for your own first issue.
Each step has concrete instructions in Markdown, commands only where documented, a chapter_id when a passport chapter helps, and done_when describing the visible result. Refer to the issue and pull request by number; the portal shows their links.
CHECK: two or three questions about the chosen issue and its pull request that a learner can answer after studying them (for example which area changed or how the fix was verified). Base them only on the candidate data.
If there are no candidate issues, set issue_number to 0, issue_why to empty, and make the mission a guided tour of making a first small change using the development chapter.
FINALE: one or two encouraging sentences that end with exactly: You are ready to go!`
