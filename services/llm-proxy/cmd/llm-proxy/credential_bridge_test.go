package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialBridgeReconcile(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantCreated int
		wantTokens  map[string]string
		wantError   string
	}{
		{
			name: "imports two accounts and chooses the newest copy of a duplicate account",
			files: map[string]string{
				"auth.json":       cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"),
				"auth-copy.json":  cliTestJSON("account-a", "new", "2026-09-24T18:28:40Z"),
				"auth-other.json": cliTestJSON("account-b", "other", "2026-09-23T18:28:40Z"),
				"cache.json":      `[]`,
			},
			wantCreated: 2,
			wantTokens:  map[string]string{"account-a": "refresh-new", "account-b": "refresh-other"},
		},
		{
			name: "does not duplicate or overwrite an existing proxy account",
			files: map[string]string{
				"auth.json":           cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"),
				"existing-codex.json": `{"type":"codex","account_id":"account-a","id_token":"id-current","access_token":"access-current","refresh_token":"refresh-current"}`,
				"auth-other.json":     cliTestJSON("account-b", "other", "2026-09-23T18:28:40Z"),
			},
			wantCreated: 1,
			wantTokens:  map[string]string{"account-a": "refresh-current", "account-b": "refresh-other"},
		},
		{
			name:      "rejects a source without a refresh token",
			files:     map[string]string{"auth.json": `{"auth_mode":"chatgpt","tokens":{"id_token":"id","access_token":"access","account_id":"account-a"}}`},
			wantError: "incomplete Codex tokens",
		},
		{
			name:      "rejects an unsafe account id",
			files:     map[string]string{"auth.json": cliTestJSON("../account-a", "old", "2026-09-23T18:28:40Z")},
			wantError: "unsafe characters",
		},
		{
			name: "rejects conflicting copies of one account at the same refresh time",
			files: map[string]string{
				"auth.json":      cliTestJSON("account-a", "one", "2026-09-23T18:28:40Z"),
				"auth-copy.json": cliTestJSON("account-a", "two", "2026-09-23T18:28:40Z"),
			},
			wantError: "conflicting Codex CLI files",
		},
		{
			name: "rejects conflicting CLI and proxy tokens at the same refresh time",
			files: map[string]string{
				"auth.json":    cliTestJSON("account-a", "one", "2026-09-23T18:28:40Z"),
				"codex-a.json": proxyTestJSON("account-a", "two", "2026-09-23T18:28:40Z"),
			},
			wantError: "conflicting Codex CLI and proxy files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeTestCredential(t, filepath.Join(dir, name), content)
			}
			bridge := newCredentialBridge(dir)
			created, err := bridge.reconcile()
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("reconcile() error = %v, want %q", err, tt.wantError)
				}
				return
			}
			if err != nil || created != tt.wantCreated {
				t.Fatalf("reconcile() = (%d, %v), want (%d, nil)", created, err, tt.wantCreated)
			}
			for accountID, wantToken := range tt.wantTokens {
				path := filepath.Join(dir, "codex-"+accountID+".json")
				if accountID == "account-a" && tt.name == "does not duplicate or overwrite an existing proxy account" {
					path = filepath.Join(dir, "existing-codex.json")
					if _, errStat := os.Stat(filepath.Join(dir, "codex-account-a.json")); !os.IsNotExist(errStat) {
						t.Fatalf("duplicate account-a proxy file exists: %v", errStat)
					}
				}
				var proxy struct {
					Type         string `json:"type"`
					AccountID    string `json:"account_id"`
					RefreshToken string `json:"refresh_token"`
				}
				readTestCredential(t, path, &proxy)
				if proxy.Type != "codex" || proxy.AccountID != accountID || proxy.RefreshToken != wantToken {
					t.Errorf("proxy account %s has incorrect fields", accountID)
				}
				if info, errStat := os.Stat(path); errStat != nil || info.Mode().Perm() != 0o600 {
					t.Errorf("proxy file mode is not 0600: %v", errStat)
				}
			}
			createdAgain, errAgain := bridge.reconcile()
			if errAgain != nil || createdAgain != 0 {
				t.Fatalf("second reconcile() = (%d, %v), want (0, nil)", createdAgain, errAgain)
			}
			accounts, errList := bridge.List(context.Background())
			if errList != nil || len(accounts) != len(tt.wantTokens) {
				t.Fatalf("List() returned %d accounts, error %v", len(accounts), errList)
			}
			for _, account := range accounts {
				if account.Provider != "codex" {
					t.Errorf("SDK loaded provider %q, want codex", account.Provider)
				}
				accountID, _ := account.Metadata["account_id"].(string)
				wantToken, exists := tt.wantTokens[accountID]
				if !exists || account.Metadata["refresh_token"] != wantToken {
					t.Errorf("SDK loaded incorrect credentials for account %q", accountID)
				}
			}
		})
	}
}

func TestCredentialBridgeReconcileFindsNewAccount(t *testing.T) {
	dir := t.TempDir()
	writeTestCredential(t, filepath.Join(dir, "first.json"), cliTestJSON("account-a", "first", "2026-09-23T18:28:40Z"))
	bridge := newCredentialBridge(dir)
	if created, err := bridge.reconcile(); err != nil || created != 1 {
		t.Fatalf("first reconcile() = (%d, %v), want (1, nil)", created, err)
	}
	writeTestCredential(t, filepath.Join(dir, "second.json"), cliTestJSON("account-b", "second", "2026-09-23T18:28:40Z"))
	if created, err := bridge.reconcile(); err != nil || created != 1 {
		t.Fatalf("second reconcile() = (%d, %v), want (1, nil)", created, err)
	}
	accounts, err := bridge.List(context.Background())
	if err != nil || len(accounts) != 2 {
		t.Fatalf("List() returned %d accounts, error %v", len(accounts), err)
	}
}

func TestCredentialBridgeReconcileHandlesSourceFilenameCollision(t *testing.T) {
	dir := t.TempDir()
	writeTestCredential(t, filepath.Join(dir, "codex-account-a.json"), cliTestJSON("account-a", "source", "2026-09-23T18:28:40Z"))
	bridge := newCredentialBridge(dir)
	if created, err := bridge.reconcile(); err != nil || created != 1 {
		t.Fatalf("reconcile() = (%d, %v), want (1, nil)", created, err)
	}
	var proxy struct {
		Type      string `json:"type"`
		AccountID string `json:"account_id"`
	}
	readTestCredential(t, filepath.Join(dir, "codex-account-a-1.json"), &proxy)
	if proxy.Type != "codex" || proxy.AccountID != "account-a" {
		t.Fatal("collision fallback did not create the account credential")
	}
}

func TestCredentialBridgeReconcileRefreshDirections(t *testing.T) {
	const oldRefresh = "2026-09-23T18:28:40Z"
	const newRefresh = "2026-09-25T18:28:40Z"
	tests := []struct {
		name        string
		updatedFile string
	}{
		{name: "newer CLI tokens update the proxy", updatedFile: "auth.json"},
		{name: "newer proxy tokens update the CLI", updatedFile: "codex-account-a.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cliPath := filepath.Join(dir, "auth.json")
			copyPath := filepath.Join(dir, "auth-copy.json")
			proxyPath := filepath.Join(dir, "codex-account-a.json")
			writeTestCredential(t, cliPath, cliTestJSON("account-a", "old", oldRefresh))
			writeTestCredential(t, copyPath, cliTestJSON("account-a", "old", "2026-09-22T18:28:40Z"))
			writeTestCredential(t, proxyPath, proxyTestJSON("account-a", "old", oldRefresh))
			bridge := newCredentialBridge(dir)
			if created, err := bridge.reconcile(); err != nil || created != 0 {
				t.Fatalf("initial reconcile() = (%d, %v), want (0, nil)", created, err)
			}
			if tt.updatedFile == "auth.json" {
				writeTestCredential(t, cliPath, cliTestJSON("account-a", "new", newRefresh))
			} else {
				writeTestCredential(t, proxyPath, proxyTestJSON("account-a", "new", newRefresh))
			}
			if created, err := bridge.reconcile(); err != nil || created != 0 {
				t.Fatalf("refresh reconcile() = (%d, %v), want (0, nil)", created, err)
			}
			for _, path := range []string{cliPath, copyPath} {
				var cli cliCredential
				readTestCredential(t, path, &cli)
				if cli.Tokens.RefreshToken != "refresh-new" || cli.LastRefresh != newRefresh {
					t.Errorf("CLI credential %s did not receive the newest tokens", filepath.Base(path))
				}
			}
			var proxy struct {
				RefreshToken string `json:"refresh_token"`
				LastRefresh  string `json:"last_refresh"`
				Note         string `json:"note"`
			}
			readTestCredential(t, proxyPath, &proxy)
			if proxy.RefreshToken != "refresh-new" || proxy.LastRefresh != newRefresh || proxy.Note != "keep" {
				t.Errorf("proxy credential did not receive the newest tokens or lost its note")
			}
			accounts, err := bridge.List(context.Background())
			if err != nil || len(accounts) != 1 || accounts[0].Metadata["refresh_token"] != "refresh-new" {
				t.Fatalf("List() did not load the newest proxy credential: %v", err)
			}
		})
	}
}

func TestCredentialBridgeReconcilePreservesUndatedCopy(t *testing.T) {
	dir := t.TempDir()
	undatedPath := filepath.Join(dir, "auth-copy.json")
	writeTestCredential(t, filepath.Join(dir, "auth.json"), cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"))
	undated := cliTestJSON("account-a", "unknown", "")
	writeTestCredential(t, undatedPath, undated)
	writeTestCredential(t, filepath.Join(dir, "codex-account-a.json"), proxyTestJSON("account-a", "new", "2026-09-25T18:28:40Z"))
	bridge := newCredentialBridge(dir)
	if created, err := bridge.reconcile(); err != nil || created != 0 {
		t.Fatalf("reconcile() = (%d, %v), want (0, nil)", created, err)
	}
	data, err := os.ReadFile(undatedPath)
	if err != nil || string(data) != undated {
		t.Fatal("reconcile() overwrote an undated credential")
	}
}

func TestCredentialBridgeSaveUpdatesMatchingCLIAccounts(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "auth.json")
	second := filepath.Join(dir, "auth-copy.json")
	other := filepath.Join(dir, "other.json")
	writeTestCredential(t, first, cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"))
	writeTestCredential(t, second, cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"))
	writeTestCredential(t, other, cliTestJSON("account-b", "other", "2026-09-23T18:28:40Z"))

	bridge := newCredentialBridge(dir)
	if _, err := bridge.reconcile(); err != nil {
		t.Fatal(err)
	}
	accounts, err := bridge.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, account := range accounts {
		if account.Metadata["account_id"] != "account-a" {
			continue
		}
		found = true
		account.Metadata["id_token"] = "id-rotated"
		account.Metadata["access_token"] = "access-rotated"
		account.Metadata["refresh_token"] = "refresh-rotated"
		account.Metadata["last_refresh"] = "2026-09-25T18:28:40Z"
		if _, err = bridge.Save(context.Background(), account); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Fatal("account-a proxy credential was not loaded")
	}
	for _, path := range []string{first, second} {
		var source struct {
			AuthMode     string         `json:"auth_mode"`
			OpenAIAPIKey *string        `json:"OPENAI_API_KEY"`
			Tokens       cliTokens      `json:"tokens"`
			LastRefresh  string         `json:"last_refresh"`
			Extra        map[string]any `json:"extra"`
		}
		readTestCredential(t, path, &source)
		if source.AuthMode != "chatgpt" || source.OpenAIAPIKey != nil || source.Tokens.RefreshToken != "refresh-rotated" || source.Tokens.AccessToken != "access-rotated" || source.LastRefresh != "2026-09-25T18:28:40Z" || source.Extra["keep"] != "yes" {
			t.Errorf("Codex CLI file %s was not synced or lost unrelated fields", filepath.Base(path))
		}
		if info, errStat := os.Stat(path); errStat != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("Codex CLI file mode is not 0600: %v", errStat)
		}
		var raw struct {
			Tokens map[string]string `json:"tokens"`
		}
		readTestCredential(t, path, &raw)
		if raw.Tokens["custom"] != "keep" {
			t.Error("unrelated token field was removed")
		}
	}
	var untouched cliCredential
	readTestCredential(t, other, &untouched)
	if untouched.Tokens.RefreshToken != "refresh-other" {
		t.Error("other account was modified")
	}

	restarted := newCredentialBridge(dir)
	if created, err := restarted.reconcile(); err != nil || created != 0 {
		t.Fatalf("restart reconcile() = (%d, %v), want (0, nil)", created, err)
	}
}

func TestSyncCLIFileRejectsChangedAccount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	original := cliTestJSON("account-b", "original", "2026-09-23T18:28:40Z")
	writeTestCredential(t, path, original)
	err := syncCLIFile(path, cliTokens{IDToken: "id-new", AccessToken: "access-new", RefreshToken: "refresh-new", AccountID: "account-a"}, "2026-09-25T18:28:40Z")
	if err == nil || !strings.Contains(err.Error(), "source account changed") {
		t.Fatalf("syncCLIFile() error = %v, want changed-account error", err)
	}
	data, errRead := os.ReadFile(path)
	if errRead != nil || string(data) != original {
		t.Fatal("source file changed after account mismatch")
	}
}

func TestCredentialBridgeSaveRejectsOlderProxyTokens(t *testing.T) {
	dir := t.TempDir()
	cliPath := filepath.Join(dir, "auth.json")
	writeTestCredential(t, cliPath, cliTestJSON("account-a", "old", "2026-09-23T18:28:40Z"))
	bridge := newCredentialBridge(dir)
	if _, err := bridge.reconcile(); err != nil {
		t.Fatal(err)
	}
	accounts, err := bridge.List(context.Background())
	if err != nil || len(accounts) != 1 {
		t.Fatalf("List() returned %d accounts, error %v", len(accounts), err)
	}
	writeTestCredential(t, cliPath, cliTestJSON("account-a", "new", "2026-09-25T18:28:40Z"))
	accounts[0].Metadata["id_token"] = "id-stale"
	accounts[0].Metadata["access_token"] = "access-stale"
	accounts[0].Metadata["refresh_token"] = "refresh-stale"
	accounts[0].Metadata["last_refresh"] = "2026-09-24T18:28:40Z"
	if _, err := bridge.Save(context.Background(), accounts[0]); err == nil || !strings.Contains(err.Error(), "refreshed more recently") {
		t.Fatalf("Save() error = %v, want newer source conflict", err)
	}
	var cli cliCredential
	readTestCredential(t, cliPath, &cli)
	if cli.Tokens.RefreshToken != "refresh-new" {
		t.Fatal("Save() overwrote newer CLI tokens")
	}
	stored, err := bridge.List(context.Background())
	if err != nil || len(stored) != 1 || stored[0].Metadata["refresh_token"] != "refresh-old" {
		t.Fatalf("Save() persisted stale proxy tokens: %v", err)
	}
	if _, err := bridge.reconcile(); err != nil {
		t.Fatal(err)
	}
	accounts, err = bridge.List(context.Background())
	if err != nil || len(accounts) != 1 || accounts[0].Metadata["refresh_token"] != "refresh-new" {
		t.Fatalf("reconcile() did not repair stale proxy tokens: %v", err)
	}
}

func cliTestJSON(accountID, suffix, lastRefresh string) string {
	data, _ := json.Marshal(map[string]any{
		"auth_mode": "chatgpt", "OPENAI_API_KEY": nil,
		"tokens": map[string]string{
			"id_token": "id-" + suffix, "access_token": "access-" + suffix,
			"refresh_token": "refresh-" + suffix, "account_id": accountID,
			"custom": "keep",
		},
		"last_refresh": lastRefresh, "extra": map[string]string{"keep": "yes"},
	})
	return string(data)
}

func proxyTestJSON(accountID, suffix, lastRefresh string) string {
	data, _ := json.Marshal(map[string]string{
		"type": "codex", "id_token": "id-" + suffix,
		"access_token": "access-" + suffix, "refresh_token": "refresh-" + suffix,
		"account_id": accountID, "last_refresh": lastRefresh, "note": "keep",
	})
	return string(data)
}

func writeTestCredential(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readTestCredential(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		t.Fatal(err)
	}
}
