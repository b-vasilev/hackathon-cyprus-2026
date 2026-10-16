package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
	"github.com/rs/zerolog"
)

// The committed contract must be what the generator produces now, so clients never
// see stale or removed routes.
func TestCommittedSpecificationIsCurrent(t *testing.T) {
	t.Parallel()
	generated, err := Specification()
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal(generated, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(openAPI, &got); err != nil {
		t.Fatal(err)
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if !bytes.Equal(wantJSON, gotJSON) {
		t.Fatal("openapi.json is stale; run `just generate-api` and commit the result")
	}
}

// Every documented operation must reach a registered handler instead of the mux's
// plain-text 404 or 405 response.
func TestSpecificationRoutesAreRegistered(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "routes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	service := generation.New(database, nil, nil, nil, zerolog.Nop(), &content.Store{Dir: t.TempDir()})
	handler := New(database, zerolog.Nop(), nil, &Factory{
		Access: &access.Store{DB: database}, Service: service,
		Packs: &learningpack.Service{DB: database, Passports: service},
	})
	generated, err := Specification()
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(generated, &spec); err != nil {
		t.Fatal(err)
	}
	parameter := regexp.MustCompile(`\{[^}]+\}`)
	for path, operations := range spec.Paths {
		for method := range operations {
			if method == "parameters" {
				continue
			}
			target := parameter.ReplaceAllString(path, "missing")
			request := httptest.NewRequest(strings.ToUpper(method), target, strings.NewReader("{}"))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code == http.StatusMethodNotAllowed || response.Body.String() == "404 page not found\n" {
				t.Errorf("%s %s is documented but not registered (status %d)", strings.ToUpper(method), path, response.Code)
			}
		}
	}
}
