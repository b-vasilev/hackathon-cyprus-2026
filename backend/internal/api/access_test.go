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
	"github.com/rs/zerolog"
)

// signInAdmin registers the first account, which becomes an admin, and returns its session cookie.
func signInAdmin(t *testing.T, store *access.Store) *http.Cookie {
	t.Helper()
	user, err := store.Register(t.Context(), access.Credentials{Email: "admin@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.CreateSession(t.Context(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: sessionCookie, Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}
}

func TestAccessRoles(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "access.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if migrationErr := db.RunMigrations(database); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	service := generation.New(database, nil, nil, nil, zerolog.Nop(), &content.Store{Dir: t.TempDir()})
	handler := New(database, zerolog.Nop(), nil, &Factory{Access: &access.Store{DB: database}, Service: service})
	request := func(method, path, body string, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
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
		return w
	}
	register := func(email string) (*http.Cookie, access.Me) {
		t.Helper()
		w := request(http.MethodPost, "/api/auth/register", `{"email":"`+email+`","password":"correct-horse"}`, nil, http.StatusCreated)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Fatal("unsafe session cookie")
		}
		var me access.Me
		if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
			t.Fatal(err)
		}
		return cookies[0], me
	}
	admin, adminMe := register("Admin@Example.com")
	lead, _ := register("lead@example.com")
	learner, learnerMe := register("learner@example.com")
	outsider, _ := register("outsider@example.com")
	if adminMe.User.Role != access.RoleAdmin || adminMe.User.Email != "admin@example.com" || learnerMe.User.Role != access.RoleUser {
		t.Fatal("first account must be the only admin", adminMe, learnerMe)
	}
	request(http.MethodPost, "/api/auth/register", `{"email":"lead@example.com","password":"correct-horse"}`, nil, http.StatusConflict)
	request(http.MethodPost, "/api/auth/register", `{"email":"short@example.com","password":"short"}`, nil, http.StatusBadRequest)
	request(http.MethodGet, "/api/products", "", nil, http.StatusUnauthorized)
	request(http.MethodGet, "/api/products", "", &http.Cookie{Name: sessionCookie, Value: "forged", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode}, http.StatusUnauthorized)

	productBody := `{"name":"Example","sources":[{"kind":"text","text":"Product purpose"}]}`
	request(http.MethodPost, "/api/products", productBody, lead, http.StatusForbidden)
	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(http.MethodPost, "/api/products", productBody, admin, http.StatusCreated).Body.Bytes(), &product); err != nil {
		t.Fatal(err)
	}
	request(http.MethodPost, "/api/products", `{"name":"Hidden","sources":[{"kind":"text","text":"Other"}]}`, admin, http.StatusCreated)
	members := "/api/products/" + product.ID + "/members"
	request(http.MethodPut, members, `{"email":"lead@example.com","role":"lead"}`, lead, http.StatusNotFound)
	request(http.MethodPut, members, `{"email":"lead@example.com","role":"lead"}`, admin, http.StatusOK)
	request(http.MethodPut, members, `{"email":"nobody@example.com","role":"learner"}`, lead, http.StatusNotFound)
	request(http.MethodPut, members, `{"email":"learner@example.com","role":"owner"}`, lead, http.StatusBadRequest)
	request(http.MethodPut, members, `{"email":"learner@example.com","role":"learner"}`, lead, http.StatusOK)
	request(http.MethodPut, members, `{"email":"outsider@example.com","role":"learner"}`, learner, http.StatusForbidden)
	request(http.MethodGet, members, "", learner, http.StatusForbidden)
	if body := request(http.MethodGet, members, "", lead, http.StatusOK).Body.String(); !strings.Contains(body, "learner@example.com") {
		t.Fatal("lead cannot see members", body)
	}

	var visible []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request(http.MethodGet, "/api/products", "", learner, http.StatusOK).Body.Bytes(), &visible); err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != product.ID {
		t.Fatal("learner sees products outside their access", visible)
	}
	if body := request(http.MethodGet, "/api/products", "", outsider, http.StatusOK).Body.String(); strings.TrimSpace(body) != "[]" {
		t.Fatal("outsider sees products", body)
	}
	request(http.MethodGet, "/api/products/"+product.ID, "", learner, http.StatusOK)
	request(http.MethodGet, "/api/products/"+product.ID+"/passport-revisions", "", learner, http.StatusForbidden)
	request(http.MethodGet, "/api/products/"+product.ID, "", outsider, http.StatusNotFound)
	request(http.MethodGet, "/api/generations/missing", "", learner, http.StatusNotFound)
	request(http.MethodPost, "/api/products/"+product.ID+"/generations", `{"connection_id":"x","model":"y"}`, learner, http.StatusForbidden)
	request(http.MethodGet, "/api/llm-connections", "", learner, http.StatusServiceUnavailable)
	// Signed-in users pass authorization; this server has no master key configured.
	request(http.MethodGet, "/api/llm-connections", "", lead, http.StatusServiceUnavailable)
	request(http.MethodPost, "/api/llm-connections", `{}`, lead, http.StatusServiceUnavailable)
	request(http.MethodDelete, "/api/llm-connections/any", "", lead, http.StatusServiceUnavailable)
	request(http.MethodDelete, "/api/products/"+product.ID, "", lead, http.StatusForbidden)
	request(http.MethodPost, "/api/generations/missing/cancel", "", lead, http.StatusNotFound)
	request(http.MethodGet, "/api/git-connection", "", lead, http.StatusOK)
	request(http.MethodGet, "/api/users", "", learner, http.StatusForbidden)
	request(http.MethodGet, "/api/users", "", lead, http.StatusOK)
	request(http.MethodPatch, "/api/users/"+learnerMe.User.ID, `{"role":"admin"}`, lead, http.StatusForbidden)

	request(http.MethodPatch, "/api/users/"+adminMe.User.ID, `{"role":"user"}`, admin, http.StatusConflict)
	request(http.MethodDelete, "/api/users/"+adminMe.User.ID, "", admin, http.StatusConflict)
	request(http.MethodPatch, "/api/users/"+learnerMe.User.ID, `{"role":"admin"}`, admin, http.StatusOK)
	request(http.MethodGet, "/api/users", "", learner, http.StatusOK)
	request(http.MethodDelete, "/api/products/"+product.ID+"/members/"+learnerMe.User.ID, "", lead, http.StatusNoContent)
	request(http.MethodDelete, "/api/users/"+learnerMe.User.ID, "", admin, http.StatusNoContent)
	request(http.MethodGet, "/api/auth/me", "", learner, http.StatusUnauthorized)

	request(http.MethodPost, "/api/auth/login", `{"email":"lead@example.com","password":"wrong-horse"}`, nil, http.StatusUnauthorized)
	request(http.MethodPost, "/api/auth/login", `{"email":"missing@example.com","password":"correct-horse"}`, nil, http.StatusUnauthorized)
	login := request(http.MethodPost, "/api/auth/login", `{"email":"LEAD@example.com","password":"correct-horse"}`, nil, http.StatusOK)
	if !strings.Contains(login.Body.String(), `"role":"lead"`) {
		t.Fatal("login did not return memberships", login.Body.String())
	}
	request(http.MethodPost, "/api/auth/logout", "", lead, http.StatusNoContent)
	request(http.MethodGet, "/api/auth/me", "", lead, http.StatusUnauthorized)
}
