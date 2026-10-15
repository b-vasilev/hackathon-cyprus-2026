package llmgateway

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

//nolint:gocyclo // Covers the connection secret lifecycle across immutable revisions.
func TestConnectionRevisionsKeepSecretsPrivate(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	store, err := NewStore(database, strings.Repeat("master", 8))
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Name: "Corporate", Enabled: true, Protocol: "openai_responses", EndpointURL: "https://example.com/v1", EndpointMode: "base", AuthMode: "header", AuthHeader: "api-key", Secret: "first-secret", Models: []Model{{ID: "same-model", OutputMode: "json_schema", MaxOutputTokens: 1024}}, Headers: []Parameter{{Name: "X-Tenant-Key", Value: "tenant-secret", Secret: true}}, QueryParams: []Parameter{}, TimeoutSeconds: 30, MaxConcurrent: 1}
	c, err := store.Save(ctx, "", config)
	if err != nil {
		t.Fatal(err)
	}
	if c.Config.Secret != "" || !c.HasSecret || c.Config.Headers[0].Value != "" {
		t.Fatal("public response contains secrets")
	}
	var ciphertext []byte
	if checkErr := database.QueryRowContext(ctx, "SELECT encrypted_config FROM llm_connection_revisions WHERE connection_id=?", c.ID).Scan(&ciphertext); checkErr != nil {
		t.Fatal(checkErr)
	}
	if strings.Contains(string(ciphertext), "first-secret") || strings.Contains(string(ciphertext), "tenant-secret") {
		t.Fatal("secrets stored in plaintext")
	}
	updated := c.Config
	updated.Name = "Updated"
	c, err = store.Save(ctx, c.ID, updated)
	if err != nil {
		t.Fatal(err)
	}
	private, err := store.Load(ctx, c.ID, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if private.Secret != "first-secret" || private.Headers[0].Value != "tenant-secret" || c.Revision != 2 {
		t.Fatal("masked update lost credentials or revision")
	}
	private.Secret = "second-secret"
	c, err = store.Save(ctx, c.ID, private)
	if err != nil {
		t.Fatal(err)
	}
	old, err := store.Load(ctx, c.ID, 1)
	if err != nil || old.Secret != "first-secret" {
		t.Fatal("previous revision changed")
	}
	disabled := c.Config
	disabled.Enabled = false
	c, err = store.Save(ctx, c.ID, disabled)
	if err != nil {
		t.Fatal(err)
	}
	if _, loadErr := store.Load(ctx, c.ID, c.Revision); loadErr == nil {
		t.Fatal("disabled connection can be used")
	}
	enabled := c.Config
	enabled.Enabled = true
	c, err = store.Save(ctx, c.ID, enabled)
	if err != nil {
		t.Fatal(err)
	}
	if _, loadErr := store.Load(ctx, c.ID, c.Revision); loadErr != nil {
		t.Fatal("could not re-enable connection", loadErr)
	}
	wrong, err := NewStore(database, strings.Repeat("different", 8))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Get(ctx, c.ID); err == nil {
		t.Fatal("wrong master key decrypted connection")
	}
}

func TestNetworkPolicyAndPaths(t *testing.T) {
	t.Parallel()
	policy, err := NewNetworkPolicy("http://127.0.0.1:11434")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		endpoint string
		valid    bool
	}{{"https://api.example.com/v1", true}, {"http://api.example.com/v1", false}, {"http://127.0.0.1:11434/v1", true}, {"http://127.0.0.1:9000/v1", false}, {"https://127.0.0.1/v1", false}} {
		if got := policy.ValidateEndpoint(tc.endpoint) == nil; got != tc.valid {
			t.Errorf("endpoint %s valid=%v", tc.endpoint, got)
		}
	}
	for _, tc := range []struct{ base, path, want string }{{"/openai/v1", "/v1/responses", "/openai/v1/responses"}, {"/v1/responses", "/v1/responses", "/v1/responses"}, {"/anthropic", "/v1/messages", "/anthropic/v1/messages"}} {
		if got := joinPath(tc.base, tc.path); got != tc.want {
			t.Errorf("joined %q, want %q", got, tc.want)
		}
	}
	if _, err := NewNetworkPolicy("http://localhost/path"); err == nil {
		t.Fatal("accepted a non-origin allowlist entry")
	}
}
