package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/rs/zerolog"
)

type githubRoundTrip func(*http.Request) (*http.Response, error)

func (fn githubRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

//nolint:gocyclo // One HTTP flow verifies token login, OAuth cookies/callback, disconnect, and disabled storage.
func TestGitHubConnectionHTTPFlow(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "github.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	store, err := githubconnection.NewStore(database, strings.Repeat("test-master", 4))
	if err != nil {
		t.Fatal(err)
	}
	store.Client.Transport = githubRoundTrip(func(r *http.Request) (*http.Response, error) {
		body := `{"login":"octocat"}`
		if r.URL.Path == "/login/oauth/access_token" {
			body = `{"access_token":"secret-oauth-token"}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	factory := &Factory{GitHub: store, GitHubOAuth: githubconnection.OAuthConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://portal.example/api/git-connection/oauth/callback"}}
	handler := New(database, zerolog.Nop(), nil, factory)
	request := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if strings.Contains(w.Body.String(), "secret-pat-token") || strings.Contains(w.Body.String(), "secret-oauth-token") {
			t.Fatal("API returned a credential")
		}
		return w
	}
	if w := request(http.MethodGet, "/api/git-connection", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"connected":false`) {
		t.Fatal("wrong initial status", w.Body.String())
	}
	if w := request(http.MethodPut, "/api/git-connection", `{"token":"secret-pat-token"}`, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"login":"octocat"`) {
		t.Fatal("token connection failed", w.Body.String())
	}
	start := request(http.MethodPost, "/api/git-connection/oauth/start", "", nil)
	if start.Code != 200 {
		t.Fatal(start.Body.String())
	}
	var authorization OAuthURL
	if checkErr := json.Unmarshal(start.Body.Bytes(), &authorization); checkErr != nil {
		t.Fatal(checkErr)
	}
	location, err := url.Parse(authorization.URL)
	if err != nil {
		t.Fatal(err)
	}
	cookies := start.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("unsafe browser-state cookie")
	}
	if w := request(http.MethodGet, "/api/git-connection/oauth/callback?code=test-code&state=wrong", "", cookies[0]); w.Code != 303 || w.Header().Get("Location") != "/connections?github=error" {
		t.Fatal("bad state accepted")
	}
	callback := "/api/git-connection/oauth/callback?code=test-code&state=" + url.QueryEscape(location.Query().Get("state"))
	if w := request(http.MethodGet, callback, "", cookies[0]); w.Code != 303 || w.Header().Get("Location") != "/connections?github=connected" {
		t.Fatal("OAuth callback failed", w.Body.String())
	}
	if w := request(http.MethodGet, "/api/git-connection", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"auth_method":"oauth"`) {
		t.Fatal("OAuth status missing")
	}
	if w := request(http.MethodDelete, "/api/git-connection", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"connected":false`) {
		t.Fatal("disconnect failed")
	}
	factory.GitHub = nil
	if w := request(http.MethodGet, "/api/git-connection", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatal("disabled status missing")
	}
	if w := request(http.MethodPut, "/api/git-connection", `{"token":"secret-pat-token"}`, nil); w.Code != 503 {
		t.Fatal("storage allowed without master key")
	}
}
