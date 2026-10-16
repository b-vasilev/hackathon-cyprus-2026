package githubconnection

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
)

// OAuthConfig describes a server-owned GitHub OAuth app.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func (o OAuthConfig) Enabled() bool {
	return o.ClientID != "" && o.ClientSecret != "" && o.RedirectURL != ""
}

type oauthState struct {
	OwnerID   string    `json:"owner_id"`
	State     string    `json:"state"`
	Verifier  string    `json:"verifier"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Begin binds authorization and its short-lived PKCE verifier to the signed-in account.
func (s *Store) Begin(ctx context.Context, o OAuthConfig) (authorizationURL, cookie string, err error) {
	if !o.Enabled() {
		return "", "", errors.New("GitHub OAuth is not configured")
	}
	verifier := make([]byte, 32)
	if _, checkErr := rand.Read(verifier); checkErr != nil {
		return "", "", fmt.Errorf("prepare GitHub OAuth: %w", checkErr)
	}
	state := oauthState{OwnerID: connectionowner.UserID(ctx), State: rand.Text(), Verifier: base64.RawURLEncoding.EncodeToString(verifier), ExpiresAt: time.Now().Add(10 * time.Minute)}
	challenge := sha256.Sum256([]byte(state.Verifier))
	query := url.Values{"client_id": {o.ClientID}, "redirect_uri": {o.RedirectURL}, "scope": {"repo"}, "state": {state.State}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	plain, err := json.Marshal(state)
	if err != nil {
		return "", "", fmt.Errorf("prepare GitHub OAuth: %w", err)
	}
	defer clear(plain)
	sealed, err := s.vault.Seal("github-oauth-state", plain)
	if err != nil {
		return "", "", fmt.Errorf("prepare GitHub OAuth: %w", err)
	}
	return "https://github.com/login/oauth/authorize?" + query.Encode(), base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Complete validates browser state before exchanging the one-time authorization code.
func (s *Store) Complete(ctx context.Context, o OAuthConfig, cookie, state, code string) error {
	if !o.Enabled() || cookie == "" || state == "" || code == "" || len(cookie) > 8192 || len(state) > 200 || len(code) > 2048 {
		return errors.New("invalid GitHub authorization")
	}
	sealed, err := base64.RawURLEncoding.DecodeString(cookie)
	if err != nil {
		return errors.New("invalid GitHub authorization")
	}
	plain, err := s.vault.Open("github-oauth-state", sealed)
	if err != nil {
		return errors.New("invalid GitHub authorization")
	}
	defer clear(plain)
	var saved oauthState
	if checkErr := json.Unmarshal(plain, &saved); checkErr != nil || !saved.matches(connectionowner.UserID(ctx), state) {
		return errors.New("GitHub authorization expired or did not match this browser")
	}
	form := url.Values{"client_id": {o.ClientID}, "client_secret": {o.ClientSecret}, "redirect_uri": {o.RedirectURL}, "code": {code}, "code_verifier": {saved.Verifier}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("prepare GitHub token exchange: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := s.Client.Do(request)
	if err != nil {
		return errors.New("GitHub authorization could not be completed")
	}
	defer response.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if response.StatusCode != http.StatusOK {
		return errors.New("GitHub rejected the authorization code")
	}
	if checkErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); checkErr != nil || result.Error != "" || result.AccessToken == "" {
		return errors.New("GitHub returned no access token")
	}
	var expiresAt time.Time
	if result.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	}
	_, err = s.Connect(ctx, result.AccessToken, "oauth", expiresAt)
	return err
}

func (s oauthState) matches(ownerID, state string) bool {
	return s.OwnerID == ownerID && time.Now().Before(s.ExpiresAt) && subtle.ConstantTimeCompare([]byte(s.State), []byte(state)) == 1
}
