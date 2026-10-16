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

func TestPersonalConnectionsHTTP(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "personal.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	store, err := llmgateway.NewStore(database, strings.Repeat("key", 16))
	if err != nil {
		t.Fatal(err)
	}
	users := &access.Store{DB: database}
	admin := signInAdmin(t, users)
	user, err := users.Register(t.Context(), access.Credentials{Email: "user@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := users.CreateSession(t.Context(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	member := &http.Cookie{Name: sessionCookie, Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
	service := generation.New(database, store, &llm.Client{}, nil, zerolog.Nop(), &content.Store{Dir: t.TempDir()})
	handler := New(database, zerolog.Nop(), nil, &Factory{Access: users, Service: service})
	request := func(cookie *http.Cookie, method, path, body string, status int) string {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	config := `{"name":"Personal","enabled":true,"protocol":"openai_responses","endpoint_url":"https://example.com/v1","endpoint_mode":"base","auth_mode":"header","auth_header":"Authorization","secret":"private-key","models":[{"id":"model","output_mode":"json_schema","max_output_tokens":1024}],"timeout_seconds":30,"max_concurrent":1}`
	owned := make([]llmgateway.Connection, 0, 2)
	for _, cookie := range []*http.Cookie{admin, member} {
		var c llmgateway.Connection
		if err := json.Unmarshal([]byte(request(cookie, http.MethodPost, "/api/llm-connections", config, http.StatusCreated)), &c); err != nil {
			t.Fatal(err)
		}
		owned = append(owned, c)
	}
	for i, cookie := range []*http.Cookie{admin, member} {
		own, foreign := owned[i], owned[1-i]
		body := request(cookie, http.MethodGet, "/api/llm-connections", "", http.StatusOK)
		if !strings.Contains(body, own.ID) || strings.Contains(body, foreign.ID) {
			t.Fatal("list exposes another user's connection")
		}
		request(cookie, http.MethodGet, "/api/llm-connections/"+own.ID, "", http.StatusOK)
		request(cookie, http.MethodPatch, "/api/llm-connections/"+own.ID, `{"name":"Renamed"}`, http.StatusOK)
		path := "/api/llm-connections/" + foreign.ID
		request(cookie, http.MethodGet, path, "", http.StatusNotFound)
		request(cookie, http.MethodPatch, path, `{"name":"Stolen"}`, http.StatusNotFound)
		request(cookie, http.MethodDelete, path, "", http.StatusNotFound)
		request(cookie, http.MethodPost, path+"/test", `{"model":"model"}`, http.StatusNotFound)
		request(cookie, http.MethodPost, "/api/llm-connections/models", `{"connection_id":"`+foreign.ID+`","config":`+config+`}`, http.StatusNotFound)
	}
	var product struct{ ID string }
	if err := json.Unmarshal([]byte(request(admin, http.MethodPost, "/api/products", `{"name":"Example","sources":[{"kind":"text","text":"Purpose"}]}`, http.StatusCreated)), &product); err != nil {
		t.Fatal(err)
	}
	request(admin, http.MethodPost, "/api/products/"+product.ID+"/generations", `{"connection_id":"`+owned[1].ID+`","model":"model"}`, http.StatusNotFound)
	request(nil, http.MethodGet, "/api/llm-connections", "", http.StatusUnauthorized)
	for i, cookie := range []*http.Cookie{admin, member} {
		request(cookie, http.MethodDelete, "/api/llm-connections/"+owned[i].ID, "", http.StatusOK)
	}
}
