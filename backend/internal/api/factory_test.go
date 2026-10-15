package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/rs/zerolog"
)

func TestFactoryHTTPBoundaries(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	store, err := llmgateway.NewStore(database, strings.Repeat("test-master", 5))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := llmgateway.NewNetworkPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	service := generation.New(database, store, nil, nil, zerolog.Nop(), &content.Store{Dir: filepath.Join(t.TempDir(), "content")})
	handler := New(database, zerolog.Nop(), nil, &Factory{Service: service, Network: policy})
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	config := `{"name":"Gateway","enabled":true,"protocol":"openai_responses","endpoint_url":"https://example.com/v1?access_token=private-query","endpoint_mode":"base","auth_mode":"header","auth_header":"Authorization","auth_prefix":"Bearer ","secret":"private-key","models":[{"id":"custom-model","output_mode":"json_schema","max_output_tokens":12000}],"timeout_seconds":60,"max_concurrent":2}`
	response := request(http.MethodPost, "/api/llm-connections", config, http.StatusCreated)
	if strings.Contains(response.Body.String(), "private-") {
		t.Fatal("connection response leaked credentials")
	}
	var connection llmgateway.Connection
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &connection); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	request(http.MethodPatch, "/api/llm-connections/"+connection.ID, `{"name":"Renamed"}`, http.StatusOK)
	loaded, err := store.Load(t.Context(), connection.ID, 2)
	if err != nil || loaded.Secret != "private-key" {
		t.Fatal("patch lost the saved credential", err)
	}
	request(http.MethodPatch, "/api/llm-connections/"+connection.ID, `{"unexpected":true}`, http.StatusBadRequest)
	request(http.MethodPatch, "/api/llm-connections/"+connection.ID, `{"endpoint_url":"http://127.0.0.1:80"}`, http.StatusBadRequest)
	request(http.MethodPost, "/api/products", `{"name":"Example","sources":[{"kind":"text","text":"Product purpose"}],"unknown":true}`, http.StatusBadRequest)
	created := request(http.MethodPost, "/api/products", `{"name":"Example","sources":[{"kind":"text","text":"Product purpose"}]}`, http.StatusCreated)
	var product struct {
		ID string `json:"id"`
	}
	if decodeErr := json.Unmarshal(created.Body.Bytes(), &product); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	runInput := `{"connection_id":"` + connection.ID + `","model":"custom-model"}`
	request(http.MethodPost, "/api/products/"+product.ID+"/generations", runInput, http.StatusAccepted)
	request(http.MethodPost, "/api/products/"+product.ID+"/generations", runInput, http.StatusConflict)
	request(http.MethodGet, "/api/products/"+product.ID+"/passport", "", http.StatusNotFound)
}

func TestFrontendRoutesDoNotMaskAPIOrMissingAssets(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>Portal</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := New(testDatabase{}, zerolog.Nop(), nil, &Factory{Service: &generation.Service{}, FrontendDir: dir})
	for _, path := range []string{"/", "/products/example", "/connections"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Portal") {
			t.Fatal("client route failed", path)
		}
		if !strings.Contains(response.Header().Get("Content-Security-Policy"), "script-src 'self'") {
			t.Fatal("missing portal security policy")
		}
	}
	for _, path := range []string{"/api/missing", "/assets/missing.js", "/unknown"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		if response.Code != http.StatusNotFound {
			t.Fatal("unknown route served portal HTML", path)
		}
	}
}
