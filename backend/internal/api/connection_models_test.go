package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/rs/zerolog"
)

func TestDiscoverModelsWithDraftAndSavedSecrets(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer private-key" || r.URL.Query().Get("token") != "private-query" || r.Header.Get("X-Tenant") != "private-header" {
			t.Error("discovery lost URL or credentials")
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"custom-model"}]}`))
	}))
	defer upstream.Close()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "models.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if migrateErr := db.RunMigrations(database); migrateErr != nil {
		t.Fatal(migrateErr)
	}
	store, err := llmgateway.NewStore(database, strings.Repeat("master", 8))
	if err != nil {
		t.Fatal(err)
	}
	client := &llm.Client{Limiter: llmgateway.NewLimiter(2, 2, 2, 2)}
	service := generation.New(database, store, client, nil, zerolog.Nop(), &content.Store{Dir: t.TempDir()})
	users := &access.Store{DB: database}
	admin := signInAdmin(t, users)
	handler := New(database, zerolog.Nop(), nil, &Factory{Access: users, Service: service})
	config := llmgateway.Config{Name: "Saved", Enabled: true, Protocol: "openai_chat_completions", EndpointURL: upstream.URL + "/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Secret: "private-key", Headers: []llmgateway.Parameter{{Name: "X-Tenant", Value: "private-header", Secret: true}}, QueryParams: []llmgateway.Parameter{{Name: "token", Value: "private-query", Secret: true}}, Models: []llmgateway.Model{{ID: "custom-model", OutputMode: "json_schema", MaxOutputTokens: 1024}}, TimeoutSeconds: 10, MaxConcurrent: 1}
	saved, err := store.Save(t.Context(), "", config)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []ModelDiscoveryInput{{Config: config}, {ConnectionID: saved.ID, Config: saved.Config}} {
		input.Config.Name = ""
		input.Config.Models = nil
		body, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/llm-connections/models", strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(admin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-") || !strings.Contains(response.Body.String(), "custom-model") {
			t.Fatalf("discovery: %d %s", response.Code, response.Body.String())
		}
	}
	current, err := store.Get(t.Context(), saved.ID)
	if err != nil || current.Revision != saved.Revision {
		t.Fatal("discovery changed the connection", err)
	}
	disabled := saved.Config
	disabled.Enabled = false
	if _, saveErr := store.Save(t.Context(), saved.ID, disabled); saveErr != nil {
		t.Fatal(saveErr)
	}
	body, err := json.Marshal(ModelDiscoveryInput{ConnectionID: saved.ID, Config: saved.Config})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/llm-connections/models", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(admin)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("disabled connection discovery: %d", response.Code)
	}
}
