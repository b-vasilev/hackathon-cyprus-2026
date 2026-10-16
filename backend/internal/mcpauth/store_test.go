package mcpauth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

//nolint:gocyclo // One grant lifecycle checks PKCE, replay protection, rotation, expiry and revocation.
func TestAuthorizationLifecycle(t *testing.T) {
	t.Parallel()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "oauth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	for _, id := range []string{"owner", "other"} {
		if _, checkErr := database.ExecContext(t.Context(), "INSERT INTO users(id,email,password_hash,role,created_at) VALUES(?,?,?,'user','created')", id, id+"@example.com", "unused"); checkErr != nil {
			t.Fatal(checkErr)
		}
	}
	store := Store{DB: database}
	for _, callback := range []string{"http://remote.example/callback", "javascript:alert(1)", "https://example.com/callback#fragment", "https://user:secret@example.com/callback"} {
		if _, checkErr := store.Register(t.Context(), "AI", []string{callback}); !errors.Is(checkErr, ErrInvalid) {
			t.Fatal("unsafe callback was accepted")
		}
	}
	client, err := store.Register(t.Context(), "AI", []string{"http://127.0.0.1:45678/callback"})
	if err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("v", 43)
	digest := sha256.Sum256([]byte(verifier))
	resource := "https://portal.example/mcp"
	a := Authorization{ClientID: client.ID, RedirectURI: client.RedirectURIs[0], ResponseType: "code", CodeChallenge: base64.RawURLEncoding.EncodeToString(digest[:]), CodeChallengeMethod: "S256", Resource: resource, Scope: Scope, State: "client-state"}
	for _, change := range []func(*Authorization){
		func(a *Authorization) { a.RedirectURI += "/wrong" },
		func(a *Authorization) { a.CodeChallengeMethod = "plain" },
		func(a *Authorization) { a.Scope = "passports:write" },
		func(a *Authorization) { a.Resource += "/wrong" },
		func(a *Authorization) { a.ClientID = "unknown" },
	} {
		invalid := a
		change(&invalid)
		if _, checkErr := store.Validate(t.Context(), invalid, resource); !errors.Is(checkErr, ErrInvalid) {
			t.Fatal("invalid authorization was accepted")
		}
	}
	code, err := store.Authorize(t.Context(), "owner", a, resource)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"code": {code}, "client_id": {client.ID}, "redirect_uri": {a.RedirectURI}, "code_verifier": {strings.Repeat("x", 43)}, "resource": {resource}}
	if _, checkErr := store.Exchange(t.Context(), form, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("wrong PKCE verifier was accepted")
	}
	form.Set("code_verifier", verifier)
	form.Set("resource", resource+"/wrong")
	if _, checkErr := store.Exchange(t.Context(), form, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("wrong resource was accepted")
	}
	form.Set("resource", resource)
	tokens, err := store.Exchange(t.Context(), form, resource)
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := store.Exchange(t.Context(), form, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("authorization code replay was accepted")
	}
	user, err := store.Authenticate(t.Context(), tokens.AccessToken, resource)
	if err != nil || user.ID != "owner" || tokens.Scope != Scope || tokens.ExpiresIn != 3600 {
		t.Fatal("issued token did not authenticate its owner")
	}
	if _, checkErr := store.Authenticate(t.Context(), tokens.AccessToken, resource+"/wrong"); !errors.Is(checkErr, access.ErrSession) {
		t.Fatal("token was accepted for another resource")
	}
	var accessHash, refreshHash string
	if checkErr := database.QueryRowContext(t.Context(), "SELECT access_hash,refresh_hash FROM mcp_tokens").Scan(&accessHash, &refreshHash); checkErr != nil || len(accessHash) != 64 || accessHash == tokens.AccessToken || refreshHash == tokens.RefreshToken {
		t.Fatal("plaintext token storage")
	}
	refresh := url.Values{"client_id": {client.ID}, "refresh_token": {tokens.RefreshToken}, "resource": {resource}}
	refresh.Set("client_id", "wrong")
	if _, checkErr := store.Refresh(t.Context(), refresh, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("refresh token accepted for another client")
	}
	refresh.Set("client_id", client.ID)
	rotated, err := store.Refresh(t.Context(), refresh, resource)
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := store.Refresh(t.Context(), refresh, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("refresh token replay was accepted")
	}
	if _, checkErr := store.Authenticate(t.Context(), tokens.AccessToken, resource); !errors.Is(checkErr, access.ErrSession) {
		t.Fatal("old access token survived rotation")
	}
	connections, err := store.Connections(t.Context(), "owner")
	if err != nil || len(connections) != 1 || connections[0].Name != "AI" {
		t.Fatal("missing account connection")
	}
	if others, checkErr := store.Connections(t.Context(), "other"); checkErr != nil || len(others) != 0 {
		t.Fatal("another account saw the connection")
	}
	if checkErr := store.Disconnect(t.Context(), "other", connections[0].ID); checkErr != nil {
		t.Fatal(checkErr)
	}
	if _, checkErr := store.Authenticate(t.Context(), rotated.AccessToken, resource); checkErr != nil {
		t.Fatal("another account disconnected the app")
	}
	if _, checkErr := database.ExecContext(t.Context(), "UPDATE mcp_tokens SET access_expires_at='2000-01-01T00:00:00Z'"); checkErr != nil {
		t.Fatal(checkErr)
	}
	if _, checkErr := store.Authenticate(t.Context(), rotated.AccessToken, resource); !errors.Is(checkErr, access.ErrSession) {
		t.Fatal("expired access token was accepted")
	}
	if checkErr := store.Disconnect(t.Context(), "owner", connections[0].ID); checkErr != nil {
		t.Fatal(checkErr)
	}
	refresh.Set("refresh_token", rotated.RefreshToken)
	if _, checkErr := store.Refresh(t.Context(), refresh, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("disconnected app refreshed its authorization")
	}
	code, err = store.Authorize(t.Context(), "owner", a, resource)
	if err != nil {
		t.Fatal(err)
	}
	if _, checkErr := database.ExecContext(t.Context(), "UPDATE mcp_codes SET expires_at='2000-01-01T00:00:00Z'"); checkErr != nil {
		t.Fatal(checkErr)
	}
	form.Set("code", code)
	if _, checkErr := store.Exchange(t.Context(), form, resource); !errors.Is(checkErr, ErrInvalid) {
		t.Fatal("expired authorization code was accepted")
	}
}
