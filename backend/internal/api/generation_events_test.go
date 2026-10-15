package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/rs/zerolog"
)

func readStreamRun(t *testing.T, scanner *bufio.Scanner) generation.Run {
	t.Helper()
	for scanner.Scan() {
		if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
			var run generation.Run
			if err := json.Unmarshal([]byte(data), &run); err != nil {
				t.Fatal(err)
			}
			return run
		}
	}
	t.Fatalf("stream ended without a snapshot: %v", scanner.Err())
	return generation.Run{}
}

//nolint:gocyclo // Integration test covers one connection, terminal delivery and reconnect replay.
func TestGenerationStreamReplayAndCompletion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "stream.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	connections, err := llmgateway.NewStore(database, strings.Repeat("stream-master", 4))
	if err != nil {
		t.Fatal(err)
	}
	connection, err := connections.Save(ctx, "", llmgateway.Config{Name: "Fixture", Enabled: true, Protocol: "openai_responses", EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", Secret: "fixture", Models: []llmgateway.Model{{ID: "fixture", OutputMode: "json_schema", MaxOutputTokens: 12000}}, TimeoutSeconds: 30, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	service := generation.New(database, connections, nil, nil, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	p, err := service.Products.Create(ctx, product.Input{Name: "Fixture", Sources: []source.Input{{Kind: "text", Text: "Fixture source"}}})
	if err != nil {
		t.Fatal(err)
	}
	run, err := service.Start(ctx, p.ID, generation.Input{ConnectionID: connection.ID, Model: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	users := &access.Store{DB: database}
	admin := signInAdmin(t, users)
	server := httptest.NewServer(New(database, zerolog.Nop(), nil, &Factory{Access: users, Service: service}))
	defer server.Close()
	url := server.URL + "/api/generations/" + run.ID + "/events"
	requestStream := func(cursor string) *http.Response {
		t.Helper()
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		request.AddCookie(admin)
		if cursor != "" {
			request.Header.Set("Last-Event-ID", cursor)
		}
		response, requestErr := server.Client().Do(request)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		return response
	}
	response := requestStream("")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal("stream was not flushed through middleware", response.Status)
	}
	scanner := bufio.NewScanner(response.Body)
	snapshot := readStreamRun(t, scanner)
	if snapshot.Status != "queued" || len(snapshot.Events) != 1 {
		t.Fatalf("incorrect initial snapshot: %+v", snapshot)
	}
	// Finish while the connection is open. State and terminal event commit together.
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "UPDATE generation_runs SET status='failed',finished_at=? WHERE id=?", now, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO generation_events(run_id,type,stage,message,created_at) VALUES(?,'run_failed','queued','Fixture stopped.',?)", run.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	update := readStreamRun(t, scanner)
	if update.Status != "failed" || len(update.Events) != 1 || update.Events[0].Type != "run_failed" {
		t.Fatalf("incorrect terminal update: %+v", update)
	}
	for scanner.Scan() {
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	replay := requestStream(strconv.FormatInt(snapshot.Events[0].ID, 10))
	defer replay.Body.Close()
	replayed := readStreamRun(t, bufio.NewScanner(replay.Body))
	if len(replayed.Events) != 1 || replayed.Events[0].ID != update.Events[0].ID {
		t.Fatal("reconnect did not replay only missed events")
	}
	url += "?after=" + strconv.FormatInt(snapshot.Events[0].ID, 10)
	queryReplay := requestStream("")
	defer queryReplay.Body.Close()
	if len(readStreamRun(t, bufio.NewScanner(queryReplay.Body)).Events) != 1 {
		t.Fatal("a new EventSource did not resume from its query cursor")
	}
	caughtUp := requestStream(strconv.FormatInt(update.Events[0].ID, 10))
	defer caughtUp.Body.Close()
	if len(readStreamRun(t, bufio.NewScanner(caughtUp.Body)).Events) != 0 {
		t.Fatal("reconnect replayed events already acknowledged")
	}
	invalid := requestStream("-1")
	defer invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatal("invalid cursor accepted")
	}
	_, _ = io.Copy(io.Discard, invalid.Body)
}
