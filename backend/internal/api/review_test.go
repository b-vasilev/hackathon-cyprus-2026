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
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/rs/zerolog"
)

// TestPassportReview covers the reviewer workflow and what learners can read at each step.
func TestPassportReview(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "review.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	store := &content.Store{Dir: t.TempDir()}
	service := generation.New(database, nil, nil, nil, zerolog.Nop(), store)
	handler := New(database, zerolog.Nop(), nil, &Factory{Access: &access.Store{DB: database}, Service: service})
	request := func(method, path, body string, cookie *http.Cookie, status int) string {
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
	register := func(email string) *http.Cookie {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"`+email+`","password":"correct-horse"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Result().Cookies()[0]
	}
	admin := register("admin@example.com")
	lead := register("lead@example.com")
	learner := register("learner@example.com")
	var product struct {
		ID string `json:"id"`
	}
	if decodeErr := json.Unmarshal([]byte(request(http.MethodPost, "/api/products", `{"name":"Example","sources":[{"kind":"text","text":"Purpose"}]}`, admin, http.StatusCreated)), &product); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	members := "/api/products/" + product.ID + "/members"
	request(http.MethodPut, members, `{"email":"lead@example.com","role":"lead"}`, admin, http.StatusOK)
	request(http.MethodPut, members, `{"email":"learner@example.com","role":"learner"}`, admin, http.StatusOK)

	draft := passport.Passport{Views: passport.Views{Document: &passport.Document{Version: 2, Chapters: []passport.Chapter{{
		ID: "overview", Title: "Overview", Question: "What is it?",
		Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "Generated text."}, {Kind: "diagram_ref", ReferenceID: "context"}},
	}}}}}
	path := "products/" + product.ID + "/versions/1/run/passport.json"
	hash, err := store.WriteJSON(path, draft)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"INSERT INTO llm_connections(id,name,created_at) VALUES('connection','Fixture','created')",
		"INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES('connection',1,X'01','created')",
		`INSERT INTO generation_runs(id,product_id,connection_id,connection_revision,model,status,stage,input_json,created_at) VALUES('run','` + product.ID + `','connection',1,'model','succeeded','complete','{}','created')`,
		`INSERT INTO passport_revisions(id,product_id,run_id,version,schema_version,content_json,content_path,content_hash,created_at) VALUES('revision','` + product.ID + `','run',1,'1','{}','` + path + `','` + hash + `','created')`,
	} {
		if _, execErr := database.ExecContext(t.Context(), statement); execErr != nil {
			t.Fatal(execErr)
		}
	}
	revision := "/api/passport-revisions/revision"
	published := "/api/products/" + product.ID + "/passport"

	// New versions are drafts: learners see nothing and cannot reach review tools.
	request(http.MethodGet, published, "", learner, http.StatusNotFound)
	request(http.MethodGet, revision+"/review", "", learner, http.StatusForbidden)
	request(http.MethodPost, revision+"/comments", `{"chapter_id":"overview","body":"Looks good"}`, learner, http.StatusForbidden)
	request(http.MethodPost, revision+"/publish", "", learner, http.StatusForbidden)

	request(http.MethodPut, revision+"/chapters/overview", `{"title":"Overview","markdown":["<script>x</script>"]}`, lead, http.StatusBadRequest)
	request(http.MethodPut, revision+"/chapters/overview", `{"title":"Overview","markdown":["one","two"]}`, lead, http.StatusBadRequest)
	if body := request(http.MethodPut, revision+"/chapters/overview", `{"title":"Product overview","markdown":["Edited by the lead."]}`, lead, http.StatusOK); !strings.Contains(body, "Edited by the lead.") || !strings.Contains(body, `"reference_id":"context"`) {
		t.Fatal("edit was not saved or changed generated references", body)
	}
	request(http.MethodPost, revision+"/comments", `{"chapter_id":"overview","body":"Check the purpose wording."}`, lead, http.StatusOK)
	request(http.MethodPut, revision+"/flag", `{"reason":"Needs a security review."}`, lead, http.StatusOK)
	request(http.MethodPost, revision+"/publish", "", lead, http.StatusConflict)
	review := request(http.MethodDelete, revision+"/flag", "", admin, http.StatusOK)
	for _, want := range []string{`"kind":"edit"`, "Check the purpose wording.", "Needs a security review.", `"kind":"resolve"`, `"author":"lead@example.com"`} {
		if !strings.Contains(review, want) {
			t.Fatal("review activity is missing", want, review)
		}
	}

	request(http.MethodPost, revision+"/publish", "", lead, http.StatusOK)
	if body := request(http.MethodGet, published, "", learner, http.StatusOK); !strings.Contains(body, "Edited by the lead.") || strings.Contains(body, "security review") {
		t.Fatal("learner does not see only the published content", body)
	}

	// Edits to a published version wait for the next publication.
	if body := request(http.MethodPut, revision+"/chapters/overview", `{"title":"Product overview","markdown":["Second edit."]}`, lead, http.StatusOK); !strings.Contains(body, `"unpublished_changes":true`) {
		t.Fatal("edited published version does not report unpublished changes", body)
	}
	if body := request(http.MethodGet, published, "", learner, http.StatusOK); strings.Contains(body, "Second edit.") {
		t.Fatal("learner sees an unpublished edit", body)
	}
	request(http.MethodPost, revision+"/publish", "", admin, http.StatusOK)
	if body := request(http.MethodGet, published, "", learner, http.StatusOK); !strings.Contains(body, "Second edit.") {
		t.Fatal("republished edit is not visible", body)
	}

	request(http.MethodPost, revision+"/unpublish", "", lead, http.StatusOK)
	request(http.MethodGet, published, "", learner, http.StatusNotFound)
	request(http.MethodGet, revision+"/evidence/missing", "", learner, http.StatusNotFound)
}
