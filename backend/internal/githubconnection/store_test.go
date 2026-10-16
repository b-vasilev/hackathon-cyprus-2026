package githubconnection

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

func TestPersonalGitHubConnections(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	store.Client.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"login":"octocat"}`), nil
	})
	alice := connectionowner.WithUser(t.Context(), "alice")
	bob := connectionowner.WithUser(t.Context(), "bob")
	if _, err := store.Connect(alice, "alice-key", "token", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if status, err := store.Status(bob); err != nil || status.Connected {
		t.Fatal("another user's GitHub account is visible", err)
	}
	if token, err := store.Token(bob); err != nil || token != "" {
		t.Fatal("another user's token is available", err)
	}
	if _, err := store.Connect(bob, "bob-key", "token", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := store.Disconnect(bob); err != nil {
		t.Fatal(err)
	}
	if token, err := store.Token(alice); err != nil || token != "alice-key" {
		t.Fatal("disconnect affected another user", err)
	}
	config := OAuthConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://portal.example/callback"}
	authorization, cookie, err := store.Begin(alice, config)
	if err != nil {
		t.Fatal(err)
	}
	location, err := url.Parse(authorization)
	if err != nil {
		t.Fatal(err)
	}
	store.Client.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("a different user reached the OAuth token exchange")
		return jsonResponse(http.StatusInternalServerError, `{}`), nil
	})
	if err := store.Complete(bob, config, cookie, location.Query().Get("state"), "code"); err == nil {
		t.Fatal("OAuth state was accepted for another user")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func newTestStore(t *testing.T) *Store {
	t.Helper()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "github.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	store, err := NewStore(database, strings.Repeat("test-key", 5))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestTokenConnectionEncryptionAndReplacement(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	store.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.github.com/user" {
			t.Fatalf("unexpected endpoint %s", r.URL)
		}
		if r.Header.Get("Authorization") == "Bearer accepted-token" {
			return jsonResponse(200, `{"login":"octocat"}`), nil
		}
		return jsonResponse(401, `{"message":"do not echo provider secrets"}`), nil
	})
	status, err := store.Connect(t.Context(), "accepted-token", "token", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	public, _ := json.Marshal(status)
	if !status.Connected || status.Login != "octocat" || strings.Contains(string(public), "accepted-token") {
		t.Fatalf("unsafe account metadata: %s", public)
	}
	var sealed []byte
	if checkErr := store.db.QueryRowContext(t.Context(), "SELECT encrypted_token FROM github_connection WHERE owner_id=''").Scan(&sealed); checkErr != nil {
		t.Fatal(checkErr)
	}
	if strings.Contains(string(sealed), "accepted-token") {
		t.Fatal("credential stored in plaintext")
	}
	if _, checkErr := store.Connect(t.Context(), "invalid-token", "token", time.Time{}); checkErr == nil || strings.Contains(checkErr.Error(), "provider secrets") {
		t.Fatalf("unsafe validation error: %v", checkErr)
	}
	token, err := store.Token(t.Context())
	if err != nil || token != "accepted-token" {
		t.Fatal("failed replacement overwrote working credential", err)
	}
	if checkErr := store.Disconnect(t.Context()); checkErr != nil {
		t.Fatal(checkErr)
	}
	token, err = store.Token(t.Context())
	if err != nil || token != "" {
		t.Fatal("disconnect retained token", err)
	}
}

//nolint:gocyclo // One flow verifies state rejection, PKCE exchange, and encrypted persistence.
func TestOAuthStatePKCEAndEncryptedCredential(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	config := OAuthConfig{ClientID: "client-id", ClientSecret: "server-secret", RedirectURL: "https://portal.example/api/git-connection/oauth/callback"}
	authorization, cookie, err := store.Begin(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	location, err := url.Parse(authorization)
	if err != nil {
		t.Fatal(err)
	}
	query := location.Query()
	if query.Get("code_challenge_method") != "S256" || len(query.Get("code_challenge")) != 43 || strings.Contains(cookie, "server-secret") || strings.Contains(authorization, "server-secret") {
		t.Fatal("unsafe OAuth start")
	}
	calls := 0
	store.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch r.URL.String() {
		case "https://github.com/login/oauth/access_token":
			if checkErr := r.ParseForm(); checkErr != nil {
				t.Fatal(checkErr)
			}
			verifier := r.Form.Get("code_verifier")
			if len(verifier) != 43 || r.Form.Get("client_secret") != "server-secret" || r.Form.Get("redirect_uri") != config.RedirectURL {
				t.Fatal("incomplete token exchange")
			}
			return jsonResponse(200, `{"access_token":"oauth-token","expires_in":3600}`), nil
		case "https://api.github.com/user":
			if r.Header.Get("Authorization") != "Bearer oauth-token" {
				t.Fatal("identity not verified with issued token")
			}
			return jsonResponse(200, `{"login":"oauth-user"}`), nil
		default:
			t.Fatal("unexpected OAuth endpoint")
			return nil, context.Canceled
		}
	})
	if checkErr := store.Complete(t.Context(), config, cookie, "wrong-state", "code"); checkErr == nil || calls != 0 {
		t.Fatal("mismatched state reached GitHub")
	}
	if checkErr := store.Complete(t.Context(), config, cookie+"tampered", query.Get("state"), "code"); checkErr == nil || calls != 0 {
		t.Fatal("tampered cookie reached GitHub")
	}
	if checkErr := store.Complete(t.Context(), config, cookie, query.Get("state"), "code"); checkErr != nil {
		t.Fatal(checkErr)
	}
	token, err := store.Token(context.Background())
	if err != nil || token != "oauth-token" || calls != 2 {
		t.Fatal("OAuth credential not connected", err)
	}
	status, err := store.Status(t.Context())
	if err != nil || status.Login != "oauth-user" || status.AuthMethod != "oauth" {
		t.Fatal("wrong OAuth identity", err)
	}
}
