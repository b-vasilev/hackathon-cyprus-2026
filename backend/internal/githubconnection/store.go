// Package githubconnection stores personal GitHub source accounts securely.
package githubconnection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/connectionowner"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/credentialvault"
)

// ErrInvalid is safe to show to the user; provider responses never enter errors.
var ErrInvalid = errors.New("invalid GitHub connection")

// Status exposes account metadata, never credentials.
type Status struct {
	Enabled      bool   `json:"enabled"`
	Connected    bool   `json:"connected"`
	Login        string `json:"login"`
	AuthMethod   string `json:"auth_method"`
	ConnectedAt  string `json:"connected_at"`
	OAuthEnabled bool   `json:"oauth_enabled"`
}

// TokenInput accepts a personal access token only in the request body.
type TokenInput struct {
	Token string `json:"token"`
}

type credential struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Store holds one source account per user.
type Store struct {
	db     *sql.DB
	vault  *credentialvault.Vault
	Client *http.Client
}

func NewStore(database *sql.DB, masterKey string) (*Store, error) {
	vault, err := credentialvault.New(masterKey)
	if err != nil {
		return nil, fmt.Errorf("initialize GitHub credential vault: %w", err)
	}
	return &Store{db: database, vault: vault, Client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	value := Status{Enabled: true}
	err := s.db.QueryRowContext(ctx, "SELECT login,auth_method,connected_at FROM github_connection WHERE owner_id=?", connectionowner.UserID(ctx)).Scan(&value.Login, &value.AuthMethod, &value.ConnectedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return value, nil
	}
	if err != nil {
		return Status{}, fmt.Errorf("read GitHub connection: %w", err)
	}
	value.Connected = true
	return value, nil
}

// Connect validates identity before replacing an existing working connection.
func (s *Store) Connect(ctx context.Context, token, method string, expiresAt time.Time) (Status, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 8192 || strings.IndexFunc(token, func(r rune) bool { return r <= ' ' || r == 127 }) >= 0 || (method != "token" && method != "oauth") {
		return Status{}, fmt.Errorf("%w: provide a valid GitHub token", ErrInvalid)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", http.NoBody)
	if err != nil {
		return Status{}, fmt.Errorf("prepare GitHub credential: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := s.Client.Do(request)
	if err != nil {
		return Status{}, fmt.Errorf("%w: GitHub could not be reached; try again", ErrInvalid)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("%w: GitHub did not accept this credential; check its access and expiry", ErrInvalid)
	}
	var user struct {
		Login string `json:"login"`
	}
	if checkErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&user); checkErr != nil || user.Login == "" {
		return Status{}, fmt.Errorf("%w: GitHub returned no account identity", ErrInvalid)
	}
	//nolint:gosec // The token is encrypted immediately and plaintext is cleared before return.
	plain, err := json.Marshal(credential{AccessToken: token, ExpiresAt: expiresAt})
	if err != nil {
		return Status{}, fmt.Errorf("prepare GitHub credential: %w", err)
	}
	defer clear(plain)
	sealed, err := s.vault.Seal("github-connection", plain)
	if err != nil {
		return Status{}, fmt.Errorf("prepare GitHub credential: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, "INSERT INTO github_connection(owner_id,login,auth_method,encrypted_token,connected_at) VALUES(?,?,?,?,?) ON CONFLICT(owner_id) DO UPDATE SET login=excluded.login,auth_method=excluded.auth_method,encrypted_token=excluded.encrypted_token,connected_at=excluded.connected_at", connectionowner.UserID(ctx), user.Login, method, sealed, now)
	if err != nil {
		return Status{}, fmt.Errorf("save GitHub connection: %w", err)
	}
	return s.Status(ctx)
}

// Token loads credentials at the start of each collection; disconnected accounts return none.
func (s *Store) Token(ctx context.Context) (string, error) {
	var sealed []byte
	err := s.db.QueryRowContext(ctx, "SELECT encrypted_token FROM github_connection WHERE owner_id=?", connectionowner.UserID(ctx)).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read GitHub credential: %w", err)
	}
	plain, err := s.vault.Open("github-connection", sealed)
	if err != nil {
		return "", fmt.Errorf("decrypt GitHub credential: %w", err)
	}
	defer clear(plain)
	var value credential
	if checkErr := json.Unmarshal(plain, &value); checkErr != nil {
		return "", errors.New("invalid saved GitHub credential; reconnect in Connections")
	}
	if !value.ExpiresAt.IsZero() && !time.Now().Before(value.ExpiresAt) {
		return "", errors.New("GitHub login expired; reconnect in Connections")
	}
	return value.AccessToken, nil
}

func (s *Store) Disconnect(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM github_connection WHERE owner_id=?", connectionowner.UserID(ctx))
	if err != nil {
		return fmt.Errorf("disconnect GitHub: %w", err)
	}
	return nil
}
