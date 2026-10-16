// Package mcpauth stores account-wide, read-only MCP authorizations.
package mcpauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
)

// Scope is the only permission issued by this authorization server.
const Scope = "passports:read"

// ErrInvalid reports an invalid authorization request or grant without exposing credentials.
var ErrInvalid = errors.New("invalid MCP authorization")

// Store shares the portal database but never accepts portal sessions as MCP tokens.
type Store struct{ DB *sql.DB }

// Client describes a registered public OAuth client.
type Client struct {
	ID           string   `json:"client_id"`
	Name         string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

// Authorization is the authorization-code request approved in the signed-in portal.
type Authorization struct {
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	ResponseType        string `json:"response_type"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	Resource            string `json:"resource"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
}

// Tokens is an OAuth token response. Plaintext secrets are returned only on issuance.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Connection is a revocable app connection shown to its account owner.
type Connection struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// Register accepts HTTPS callbacks and local loopback HTTP callbacks only.
func (s Store) Register(ctx context.Context, name string, redirects []string) (Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "AI app"
	}
	if len(name) > 100 || len(redirects) == 0 || len(redirects) > 10 {
		return Client{}, ErrInvalid
	}
	for _, redirect := range redirects {
		u, err := url.Parse(redirect)
		if err != nil || len(redirect) > 2048 || u.Host == "" || u.User != nil || u.Fragment != "" ||
			(u.Scheme != "https" && (u.Scheme != "http" || !slices.Contains([]string{"127.0.0.1", "localhost", "::1"}, u.Hostname()))) {
			return Client{}, ErrInvalid
		}
	}
	client := Client{ID: rand.Text(), Name: name, RedirectURIs: redirects}
	data, err := json.Marshal(redirects)
	if err != nil {
		return Client{}, fmt.Errorf("encode MCP callbacks: %w", err)
	}
	_, err = s.DB.ExecContext(ctx, "INSERT INTO mcp_clients(id,name,redirect_uris_json,created_at) VALUES(?,?,?,?)", client.ID, name, string(data), timestamp(time.Now()))
	if err != nil {
		return Client{}, fmt.Errorf("register MCP client: %w", err)
	}
	return client, nil
}

// Validate checks the client, exact redirect, read-only scope, resource and S256 PKCE.
func (s Store) Validate(ctx context.Context, input Authorization, resource string) (Client, error) {
	challenge, err := base64.RawURLEncoding.DecodeString(input.CodeChallenge)
	if err != nil || len(challenge) != sha256.Size || input.CodeChallengeMethod != "S256" || input.ResponseType != "code" ||
		input.Resource != resource || (input.Scope != "" && input.Scope != Scope) || len(input.State) > 2048 {
		return Client{}, ErrInvalid
	}
	var client Client
	var redirects string
	if err := s.DB.QueryRowContext(ctx, "SELECT id,name,redirect_uris_json FROM mcp_clients WHERE id=?", input.ClientID).
		Scan(&client.ID, &client.Name, &redirects); err != nil {
		return Client{}, ErrInvalid
	}
	if err := json.Unmarshal([]byte(redirects), &client.RedirectURIs); err != nil {
		return Client{}, fmt.Errorf("decode MCP callbacks: %w", err)
	}
	if !slices.Contains(client.RedirectURIs, input.RedirectURI) {
		return Client{}, ErrInvalid
	}
	return client, nil
}

// Authorize issues a short-lived code after explicit browser consent.
func (s Store) Authorize(ctx context.Context, userID string, input Authorization, resource string) (string, error) {
	if _, err := s.Validate(ctx, input, resource); err != nil {
		return "", err
	}
	code := rand.Text() + rand.Text()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO mcp_codes(code_hash,client_id,user_id,redirect_uri,challenge,resource,expires_at)
 VALUES(?,?,?,?,?,?,?)`, hash(code), input.ClientID, userID, input.RedirectURI, input.CodeChallenge, resource, timestamp(time.Now().Add(5*time.Minute)))
	if err != nil {
		return "", fmt.Errorf("save MCP authorization code: %w", err)
	}
	return code, nil
}

func newTokens() Tokens {
	return Tokens{AccessToken: rand.Text() + rand.Text(), RefreshToken: rand.Text() + rand.Text(), TokenType: "Bearer", ExpiresIn: 3600, Scope: Scope}
}

// Exchange consumes a code once and binds tokens to the approved resource and account.
func (s Store) Exchange(ctx context.Context, form url.Values, resource string) (Tokens, error) {
	verifier := form.Get("code_verifier")
	if len(verifier) < 43 || len(verifier) > 128 || strings.ContainsFunc(verifier, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && !strings.ContainsRune("-._~", r)
	}) || form.Get("resource") != resource {
		return Tokens{}, ErrInvalid
	}
	digest := sha256.Sum256([]byte(verifier))
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Tokens{}, fmt.Errorf("begin MCP token exchange: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var userID string
	err = tx.QueryRowContext(ctx, `DELETE FROM mcp_codes WHERE code_hash=? AND client_id=? AND redirect_uri=? AND challenge=? AND resource=? AND expires_at>? RETURNING user_id`,
		hash(form.Get("code")), form.Get("client_id"), form.Get("redirect_uri"), base64.RawURLEncoding.EncodeToString(digest[:]), resource, timestamp(time.Now())).Scan(&userID)
	if err != nil {
		return Tokens{}, ErrInvalid
	}
	grantID, tokens := rand.Text(), newTokens()
	if _, err := tx.ExecContext(ctx, "INSERT INTO mcp_grants(id,client_id,user_id,created_at) VALUES(?,?,?,?)", grantID, form.Get("client_id"), userID, timestamp(time.Now())); err != nil {
		return Tokens{}, fmt.Errorf("save MCP grant: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO mcp_tokens(grant_id,access_hash,refresh_hash,resource,access_expires_at,refresh_expires_at) VALUES(?,?,?,?,?,?)",
		grantID, hash(tokens.AccessToken), hash(tokens.RefreshToken), resource, timestamp(time.Now().Add(time.Hour)), timestamp(time.Now().Add(30*24*time.Hour))); err != nil {
		return Tokens{}, fmt.Errorf("save MCP tokens: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Tokens{}, fmt.Errorf("commit MCP token exchange: %w", err)
	}
	return tokens, nil
}

// Refresh rotates both tokens; disconnecting an app also prevents refresh.
func (s Store) Refresh(ctx context.Context, form url.Values, resource string) (Tokens, error) {
	if form.Get("resource") != resource || (form.Get("scope") != "" && form.Get("scope") != Scope) {
		return Tokens{}, ErrInvalid
	}
	tokens := newTokens()
	result, err := s.DB.ExecContext(ctx, `UPDATE mcp_tokens SET access_hash=?,refresh_hash=?,access_expires_at=?
 WHERE refresh_hash=? AND resource=? AND refresh_expires_at>? AND grant_id IN
 (SELECT id FROM mcp_grants WHERE client_id=? AND revoked_at='')`,
		hash(tokens.AccessToken), hash(tokens.RefreshToken), timestamp(time.Now().Add(time.Hour)), hash(form.Get("refresh_token")), resource, timestamp(time.Now()), form.Get("client_id"))
	if err != nil {
		return Tokens{}, fmt.Errorf("refresh MCP tokens: %w", err)
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return Tokens{}, ErrInvalid
	}
	return tokens, nil
}

// Authenticate reloads the current account role on every MCP request.
func (s Store) Authenticate(ctx context.Context, token, resource string) (access.User, error) {
	var user access.User
	err := s.DB.QueryRowContext(ctx, `SELECT u.id,u.email,u.role,u.created_at FROM mcp_tokens t
 JOIN mcp_grants g ON g.id=t.grant_id JOIN users u ON u.id=g.user_id
 WHERE t.access_hash=? AND t.resource=? AND t.access_expires_at>? AND t.refresh_expires_at>? AND g.revoked_at=''`, hash(token), resource, timestamp(time.Now()), timestamp(time.Now())).
		Scan(&user.ID, &user.Email, &user.Role, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return access.User{}, access.ErrSession
	}
	if err != nil {
		return access.User{}, fmt.Errorf("read MCP authorization: %w", err)
	}
	return user, nil
}

// Connections returns active apps without returning any credentials.
func (s Store) Connections(ctx context.Context, userID string) ([]Connection, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT g.id,c.name,g.created_at,t.refresh_expires_at FROM mcp_grants g
 JOIN mcp_clients c ON c.id=g.client_id JOIN mcp_tokens t ON t.grant_id=g.id
 WHERE g.user_id=? AND g.revoked_at='' AND t.refresh_expires_at>? ORDER BY g.created_at DESC`, userID, timestamp(time.Now()))
	if err != nil {
		return nil, fmt.Errorf("list MCP connections: %w", err)
	}
	defer rows.Close()
	result := make([]Connection, 0)
	for rows.Next() {
		var connection Connection
		if err := rows.Scan(&connection.ID, &connection.Name, &connection.CreatedAt, &connection.ExpiresAt); err != nil {
			return nil, fmt.Errorf("read MCP connection: %w", err)
		}
		result = append(result, connection)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read MCP connections: %w", err)
	}
	return result, nil
}

// Disconnect revokes only a connection owned by this account.
func (s Store) Disconnect(ctx context.Context, userID, id string) error {
	if _, err := s.DB.ExecContext(ctx, "UPDATE mcp_grants SET revoked_at=? WHERE id=? AND user_id=?", timestamp(time.Now()), id, userID); err != nil {
		return fmt.Errorf("disconnect MCP app: %w", err)
	}
	return nil
}
