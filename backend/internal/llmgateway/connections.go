// Package llmgateway manages versioned LLM connections.
// Connection validation and credential handling are adapted from antares-app.
package llmgateway

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/credentialvault"
)

// ErrInvalid marks invalid connection configuration.
var ErrInvalid = errors.New("invalid connection")

// ErrInUse marks a connection used by a pending generation.
var ErrInUse = errors.New("stop active generations before deleting this connection")

const MaxConcurrentRequests = 30

// Parameter is an additional HTTP header or query parameter.
type Parameter struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

// Model describes an endpoint's model and structured-output capabilities.
type Model struct {
	ID              string `json:"id"`
	OutputMode      string `json:"output_mode"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	ReasoningEffort string `json:"reasoning_effort"`
}

// Config is encrypted as a whole. Secret values are removed from public responses.
type Config struct {
	Name           string      `json:"name"`
	Enabled        bool        `json:"enabled"`
	Protocol       string      `json:"protocol"`
	EndpointURL    string      `json:"endpoint_url"`
	EndpointMode   string      `json:"endpoint_mode"`
	AuthMode       string      `json:"auth_mode"`
	AuthHeader     string      `json:"auth_header"`
	AuthPrefix     string      `json:"auth_prefix"`
	Secret         string      `json:"secret"`
	Headers        []Parameter `json:"headers"`
	QueryParams    []Parameter `json:"query_params"`
	Models         []Model     `json:"models"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	MaxConcurrent  int         `json:"max_concurrent"`
	QuotaGroup     string      `json:"quota_group"`
}

// Connection is the safe view returned by connection APIs.
type Connection struct {
	ID        string `json:"id"`
	Revision  int    `json:"revision"`
	HasSecret bool   `json:"has_secret"`
	Config    Config `json:"config"`
	CreatedAt string `json:"created_at"`
}

// Store persists immutable connection configurations and secrets.
type Store struct {
	db    *sql.DB
	vault *credentialvault.Vault
}

// NewStore creates a store using the existing database and a server master key.
func NewStore(database *sql.DB, masterKey string) (*Store, error) {
	vault, err := credentialvault.New(masterKey)
	if err != nil {
		return nil, fmt.Errorf("initialize credential vault: %w", err)
	}
	return &Store{db: database, vault: vault}, nil
}

// Validate normalizes configuration without restricting a connection to a vendor.
//
//nolint:gocyclo // One boundary validates the complete connection contract before persistence.
func Validate(c *Config) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || len(c.Name) > 100 {
		return fmt.Errorf("%w: name must contain 1–100 characters", ErrInvalid)
	}
	if !slices.Contains([]string{"openai_responses", "openai_chat_completions", "anthropic_messages"}, c.Protocol) {
		return fmt.Errorf("%w: unsupported API protocol", ErrInvalid)
	}
	u, err := url.Parse(c.EndpointURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || len(c.EndpointURL) > 2048 {
		return fmt.Errorf("%w: endpoint must be an HTTP(S) URL without user information or a fragment", ErrInvalid)
	}
	if u.RawQuery != "" {
		for name, values := range u.Query() {
			for _, value := range values {
				c.QueryParams = append(c.QueryParams, Parameter{Name: name, Value: value, Secret: true})
			}
		}
		u.RawQuery = ""
		u.ForceQuery = false
		c.EndpointURL = u.String()
	}
	if !slices.Contains([]string{"base", "full"}, c.EndpointMode) {
		return fmt.Errorf("%w: endpoint mode must be base or full", ErrInvalid)
	}
	if c.AuthMode != "none" && c.AuthMode != "header" {
		return fmt.Errorf("%w: auth mode must be none or header", ErrInvalid)
	}
	if c.AuthMode == "header" && (!safeHeader(c.AuthHeader) || c.Secret == "" || !safeValue(c.Secret) || !safeValue(c.AuthPrefix)) {
		return fmt.Errorf("%w: a valid auth header and key are required", ErrInvalid)
	}
	if c.AuthMode == "none" {
		c.Secret = ""
		c.AuthHeader = ""
		c.AuthPrefix = ""
	}
	if len(c.Headers) > 20 || len(c.QueryParams) > 20 {
		return fmt.Errorf("%w: too many HTTP parameters", ErrInvalid)
	}
	for _, h := range c.Headers {
		if !safeHeader(h.Name) || !safeValue(h.Value) {
			return fmt.Errorf("%w: invalid additional header", ErrInvalid)
		}
	}
	for _, q := range c.QueryParams {
		if q.Name == "" || len(q.Name) > 100 || !safeValue(q.Name) || !safeValue(q.Value) {
			return fmt.Errorf("%w: invalid query parameter", ErrInvalid)
		}
	}
	if len(c.Models) == 0 || len(c.Models) > 20 {
		return fmt.Errorf("%w: configure 1–20 models", ErrInvalid)
	}
	seen := make(map[string]bool)
	for _, m := range c.Models {
		if !slices.Contains([]string{"", "none", "minimal", "low", "medium", "high", "xhigh"}, m.ReasoningEffort) {
			return fmt.Errorf("%w: unsupported reasoning effort", ErrInvalid)
		}
		if strings.TrimSpace(m.ID) == "" || len(m.ID) > 200 || seen[m.ID] || !safeValue(m.ID) {
			return fmt.Errorf("%w: model IDs must be non-empty and unique", ErrInvalid)
		}
		seen[m.ID] = true
		if !slices.Contains([]string{"json_schema", "tool_call", "prompt_json"}, m.OutputMode) || m.MaxOutputTokens < 256 || m.MaxOutputTokens > 64000 {
			return fmt.Errorf("%w: invalid model output mode or token limit", ErrInvalid)
		}
	}
	if c.TimeoutSeconds < 10 || c.TimeoutSeconds > 600 || c.MaxConcurrent < 1 || c.MaxConcurrent > MaxConcurrentRequests || len(c.QuotaGroup) > 64 || !safeValue(c.QuotaGroup) {
		return fmt.Errorf("%w: invalid timeout, concurrency or quota group", ErrInvalid)
	}
	return nil
}

func safeValue(value string) bool {
	return len(value) <= 8192 && strings.IndexFunc(value, func(r rune) bool { return r < ' ' || r == 127 }) < 0
}

func safeHeader(name string) bool {
	if name == "" || len(name) > 100 {
		return false
	}
	for _, r := range name {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	switch strings.ToLower(name) {
	case "host", "cookie", "set-cookie", "content-length", "transfer-encoding", "connection", "forwarded", "upgrade", "te", "trailer":
		return false
	}
	return !strings.HasPrefix(strings.ToLower(name), "x-forwarded-") && !strings.HasPrefix(strings.ToLower(name), "proxy-")
}

// Save creates a connection or appends a revision, retaining unchanged masked secrets.
func (s *Store) Save(ctx context.Context, id string, config Config) (Connection, error) {
	revision := 1
	if id != "" {
		current, err := s.Get(ctx, id)
		if err != nil {
			return Connection{}, err
		}
		old, err := s.load(ctx, id, current.Revision, false)
		if err != nil {
			return Connection{}, err
		}
		preserveSecrets(&config, old)
		revision = current.Revision + 1
	} else {
		id = rand.Text()
	}
	if err := Validate(&config); err != nil {
		return Connection{}, err
	}
	//nolint:gosec // Only encrypted configuration is persisted; this buffer is cleared below.
	plaintext, err := json.Marshal(config)
	if err != nil {
		return Connection{}, fmt.Errorf("encode connection: %w", err)
	}
	defer clear(plaintext)
	sealed, err := s.vault.Seal(fmt.Sprintf("llm-connection:%s:%d", id, revision), plaintext)
	if err != nil {
		return Connection{}, fmt.Errorf("encrypt connection: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Connection{}, fmt.Errorf("begin connection update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if revision == 1 {
		_, err = tx.ExecContext(ctx, "INSERT INTO llm_connections(id,name,enabled,current_revision,created_at) VALUES(?,?,?,?,?)", id, config.Name, config.Enabled, revision, now)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, "UPDATE llm_connections SET name=?,enabled=?,current_revision=? WHERE id=? AND current_revision=? AND deleted=0", config.Name, config.Enabled, revision, id, revision-1)
		if err == nil {
			var count int64
			count, err = result.RowsAffected()
			if err == nil && count != 1 {
				return Connection{}, fmt.Errorf("%w: connection changed; reload before saving", ErrInvalid)
			}
		}
	}
	if err != nil {
		return Connection{}, fmt.Errorf("save connection: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO llm_connection_revisions(connection_id,revision,encrypted_config,created_at) VALUES(?,?,?,?)", id, revision, sealed, now); err != nil {
		return Connection{}, fmt.Errorf("save connection revision: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Connection{}, fmt.Errorf("commit connection: %w", err)
	}
	return s.Get(ctx, id)
}

// Prepare restores masked credentials and validates an unsaved connection draft.
func (s *Store) Prepare(ctx context.Context, id string, config Config) (Config, error) {
	if id != "" {
		current, err := s.Get(ctx, id)
		if err != nil {
			return Config{}, err
		}
		old, err := s.Load(ctx, id, current.Revision)
		if err != nil {
			return Config{}, err
		}
		preserveSecrets(&config, old)
	}
	if err := Validate(&config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func preserveSecrets(next *Config, previous Config) {
	if next.AuthMode == "header" && next.Secret == "" {
		next.Secret = previous.Secret
	}
	preserveParameters(next.Headers, previous.Headers)
	preserveParameters(next.QueryParams, previous.QueryParams)
}

func preserveParameters(next, previous []Parameter) {
	for i := range next {
		if next[i].Secret && next[i].Value == "" {
			for _, old := range previous {
				if old.Name == next[i].Name && old.Secret {
					next[i].Value = old.Value
					break
				}
			}
		}
	}
}

// Load returns private configuration to server-side callers only.
func (s *Store) Load(ctx context.Context, id string, revision int) (Config, error) {
	return s.load(ctx, id, revision, true)
}

func (s *Store) load(ctx context.Context, id string, revision int, enabledOnly bool) (Config, error) {
	var encrypted []byte
	err := s.db.QueryRowContext(ctx, "SELECT r.encrypted_config FROM llm_connection_revisions r JOIN llm_connections c ON c.id=r.connection_id WHERE r.connection_id=? AND r.revision=? AND c.deleted=0 AND (?=0 OR c.enabled=1)", id, revision, enabledOnly).Scan(&encrypted)
	if err != nil {
		return Config{}, fmt.Errorf("load enabled connection: %w", err)
	}
	data, err := s.vault.Open(fmt.Sprintf("llm-connection:%s:%d", id, revision), encrypted)
	if err != nil {
		return Config{}, fmt.Errorf("decrypt connection: %w", err)
	}
	defer clear(data)
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode connection: %w", err)
	}
	return config, nil
}

// Get returns a connection with write-only secrets redacted, including disabled connections.
func (s *Store) Get(ctx context.Context, id string) (Connection, error) {
	var c Connection
	var encrypted []byte
	var enabled bool
	err := s.db.QueryRowContext(ctx, "SELECT c.id,c.current_revision,c.enabled,c.created_at,r.encrypted_config FROM llm_connections c JOIN llm_connection_revisions r ON r.connection_id=c.id AND r.revision=c.current_revision WHERE c.id=? AND c.deleted=0", id).Scan(&c.ID, &c.Revision, &enabled, &c.CreatedAt, &encrypted)
	if err != nil {
		return Connection{}, fmt.Errorf("get connection: %w", err)
	}
	data, err := s.vault.Open(fmt.Sprintf("llm-connection:%s:%d", id, c.Revision), encrypted)
	if err != nil {
		return Connection{}, fmt.Errorf("decrypt public connection: %w", err)
	}
	defer clear(data)
	if err := json.Unmarshal(data, &c.Config); err != nil {
		return Connection{}, fmt.Errorf("decode public connection: %w", err)
	}
	c.HasSecret = c.Config.Secret != ""
	c.Config.Secret = ""
	c.Config.Enabled = enabled
	for i := range c.Config.Headers {
		if c.Config.Headers[i].Secret {
			c.Config.Headers[i].Value = ""
		}
	}
	for i := range c.Config.QueryParams {
		if c.Config.QueryParams[i].Secret {
			c.Config.QueryParams[i].Value = ""
		}
	}
	return c, nil
}

// List returns all connections without holding a database cursor while decrypting.
func (s *Store) List(ctx context.Context) ([]Connection, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM llm_connections WHERE deleted=0 ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read connection ID: %w", err)
		}
		ids = append(ids, id)
	}
	readErr := rows.Err()
	_ = rows.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read connections: %w", readErr)
	}
	result := make([]Connection, 0, len(ids))
	for _, id := range ids {
		c, err := s.Get(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			continue // A connection may have been deleted after collecting its ID.
		}
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// ModelByID resolves only models explicitly configured for this connection.
func ModelByID(c Config, id string) (Model, error) {
	for _, m := range c.Models {
		if m.ID == id {
			return m, nil
		}
	}
	return Model{}, fmt.Errorf("%w: model is not configured for this connection", ErrInvalid)
}

// CanonicalHeader normalizes a configured HTTP header name.
func CanonicalHeader(name string) string { return http.CanonicalHeaderKey(name) }
