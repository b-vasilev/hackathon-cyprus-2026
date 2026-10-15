package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sdkauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/auth"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

// credentialBridge keeps Codex CLI files in their original format while the SDK
// uses a separate, flat credential file for each account.
type credentialBridge struct {
	mu      sync.Mutex
	dir     string
	store   *sdkauth.FileTokenStore
	sources map[string][]string
}

type cliTokens struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

type cliCredential struct {
	AuthMode    string    `json:"auth_mode"`
	Tokens      cliTokens `json:"tokens"`
	LastRefresh string    `json:"last_refresh"`
}

type cliSource struct {
	path       string
	credential cliCredential
	refreshed  time.Time
}

type proxySource struct {
	path      string
	tokens    cliTokens
	refreshed time.Time
}

func newCredentialBridge(dir string) *credentialBridge {
	bridge := &credentialBridge{store: sdkauth.NewFileTokenStore()}
	bridge.SetBaseDir(dir)
	return bridge
}

func (b *credentialBridge) SetBaseDir(dir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dir = dir
	b.store.SetBaseDir(dir)
}

// reconcile creates one SDK credential per Codex account and copies the newest
// known tokens between CLI and proxy files.
func (b *credentialBridge) reconcile() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sources, proxies, err := scanCredentialFiles(b.dir)
	if err != nil {
		return 0, err
	}
	b.sources = make(map[string][]string, len(sources))
	created := 0
	for accountID, entries := range sources {
		for _, entry := range entries {
			b.sources[accountID] = append(b.sources[accountID], entry.path)
		}
	}

	for accountID, entries := range sources {
		newest := entries[0]
		for _, entry := range entries[1:] {
			if entry.refreshed.After(newest.refreshed) {
				newest = entry
			} else if entry.refreshed.Equal(newest.refreshed) && !sameTokens(entry.credential.Tokens, newest.credential.Tokens) {
				return 0, errors.New("conflicting Codex CLI files for one account with the same last_refresh")
			}
		}
		proxyEntries := proxies[accountID]
		if len(proxyEntries) != 0 {
			// A file without last_refresh might be newer than either side. Leave
			// the whole account alone rather than overwriting an undated copy.
			unknownRefresh := false
			for _, entry := range entries {
				unknownRefresh = unknownRefresh || entry.refreshed.IsZero()
			}
			for _, entry := range proxyEntries {
				unknownRefresh = unknownRefresh || entry.refreshed.IsZero()
			}
			if unknownRefresh {
				continue
			}
			current := proxyEntries[0]
			for _, entry := range proxyEntries[1:] {
				if entry.refreshed.After(current.refreshed) {
					current = entry
				} else if entry.refreshed.Equal(current.refreshed) && !sameTokens(entry.tokens, current.tokens) {
					return 0, errors.New("conflicting proxy files for one account with the same last_refresh")
				}
			}
			if newest.refreshed.Equal(current.refreshed) && !sameTokens(newest.credential.Tokens, current.tokens) {
				return 0, errors.New("conflicting Codex CLI and proxy files with the same last_refresh")
			}
			tokens := newest.credential.Tokens
			lastRefresh := newest.credential.LastRefresh
			if current.refreshed.After(newest.refreshed) {
				tokens = current.tokens
				lastRefresh = current.refreshed.Format(time.RFC3339Nano)
			}
			for _, entry := range entries {
				if errSync := syncCLIFile(entry.path, tokens, lastRefresh); errSync != nil {
					return created, fmt.Errorf("sync Codex CLI file %s: %w", filepath.Base(entry.path), errSync)
				}
			}
			for _, entry := range proxyEntries {
				if errSync := syncProxyFile(entry.path, tokens, lastRefresh); errSync != nil {
					return created, fmt.Errorf("sync proxy file %s: %w", filepath.Base(entry.path), errSync)
				}
			}
			continue
		}

		path, errPath := availableProxyPath(b.dir, accountID)
		if errPath != nil {
			return created, errPath
		}
		data, errMarshal := json.Marshal(map[string]any{
			"type":          "codex",
			"id_token":      newest.credential.Tokens.IDToken,
			"access_token":  newest.credential.Tokens.AccessToken,
			"refresh_token": newest.credential.Tokens.RefreshToken,
			"account_id":    newest.credential.Tokens.AccountID,
			"last_refresh":  newest.credential.LastRefresh,
		})
		if errMarshal != nil {
			return created, fmt.Errorf("encode proxy credential: %w", errMarshal)
		}
		if errWrite := writeNewCredential(path, append(data, '\n')); errWrite != nil {
			return created, fmt.Errorf("create proxy credential: %w", errWrite)
		}
		created++
	}
	return created, nil
}

func availableProxyPath(dir, accountID string) (string, error) {
	for index := 0; index < 1000; index++ {
		name := "codex-" + accountID
		if index > 0 {
			name = fmt.Sprintf("%s-%d", name, index)
		}
		path := filepath.Join(dir, name+".json")
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return path, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("no available filename for proxy credential")
}

func scanCredentialFiles(dir string) (map[string][]cliSource, map[string][]proxySource, error) {
	sources := make(map[string][]cliSource)
	proxies := make(map[string][]proxySource)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(path), ".json") {
			return nil
		}
		data, errRead := os.ReadFile(path)
		if errRead != nil {
			return fmt.Errorf("read %s: %w", filepath.Base(path), errRead)
		}
		trimmed := bytes.TrimSpace(data)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil
		}
		var header struct {
			AuthMode string `json:"auth_mode"`
			Type     string `json:"type"`
		}
		if errJSON := json.Unmarshal(data, &header); errJSON != nil {
			return fmt.Errorf("parse %s: %w", filepath.Base(path), errJSON)
		}
		if header.AuthMode == "chatgpt" {
			var credential cliCredential
			if errJSON := json.Unmarshal(data, &credential); errJSON != nil {
				return fmt.Errorf("parse %s: %w", filepath.Base(path), errJSON)
			}
			if credential.Tokens.IDToken == "" || credential.Tokens.AccessToken == "" || credential.Tokens.RefreshToken == "" || credential.Tokens.AccountID == "" {
				return fmt.Errorf("%s has incomplete Codex tokens", filepath.Base(path))
			}
			accountID, errID := safeAccountID(credential.Tokens.AccountID)
			if errID != nil {
				return fmt.Errorf("%s: %w", filepath.Base(path), errID)
			}
			refreshed, errTime := parseRefreshTime(credential.LastRefresh)
			if errTime != nil {
				return fmt.Errorf("%s has invalid last_refresh", filepath.Base(path))
			}
			sources[accountID] = append(sources[accountID], cliSource{path: path, credential: credential, refreshed: refreshed})
			return nil
		}
		if strings.EqualFold(header.Type, "codex") {
			var credential struct {
				IDToken      string `json:"id_token"`
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				AccountID    string `json:"account_id"`
				LastRefresh  string `json:"last_refresh"`
			}
			if errJSON := json.Unmarshal(data, &credential); errJSON != nil {
				return fmt.Errorf("parse %s: %w", filepath.Base(path), errJSON)
			}
			if credential.AccountID == "" {
				return fmt.Errorf("%s has no account_id", filepath.Base(path))
			}
			if credential.IDToken == "" || credential.AccessToken == "" || credential.RefreshToken == "" {
				return fmt.Errorf("%s has incomplete proxy tokens", filepath.Base(path))
			}
			accountID, errID := safeAccountID(credential.AccountID)
			if errID != nil {
				return fmt.Errorf("%s: %w", filepath.Base(path), errID)
			}
			refreshed, errTime := parseRefreshTime(credential.LastRefresh)
			if errTime != nil {
				return fmt.Errorf("%s has invalid last_refresh", filepath.Base(path))
			}
			proxies[accountID] = append(proxies[accountID], proxySource{
				path: path,
				tokens: cliTokens{
					IDToken: credential.IDToken, AccessToken: credential.AccessToken,
					RefreshToken: credential.RefreshToken, AccountID: credential.AccountID,
				},
				refreshed: refreshed,
			})
		}
		return nil
	})
	return sources, proxies, err
}

func parseRefreshTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

func sameTokens(a, b cliTokens) bool {
	return a.IDToken == b.IDToken && a.AccessToken == b.AccessToken &&
		a.RefreshToken == b.RefreshToken && strings.EqualFold(a.AccountID, b.AccountID)
}

func safeAccountID(value string) (string, error) {
	if len(value) == 0 || len(value) > 128 {
		return "", errors.New("invalid account_id length")
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return "", errors.New("account_id contains unsafe characters")
		}
	}
	return strings.ToLower(value), nil
}

func (b *credentialBridge) List(ctx context.Context) ([]*coreauth.Auth, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	all, err := b.store.List(ctx)
	if err != nil {
		return nil, err
	}
	sourcePaths := make(map[string]bool)
	for _, paths := range b.sources {
		for _, path := range paths {
			sourcePaths[path] = true
		}
	}
	result := make([]*coreauth.Auth, 0, len(all))
	for _, auth := range all {
		if auth.Provider != "unknown" && !sourcePaths[auth.Attributes[coreauth.AttributePath]] {
			result = append(result, auth)
		}
	}
	return result, nil
}

func (b *credentialBridge) Save(ctx context.Context, auth *coreauth.Auth) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var paths []string
	tokens := cliTokens{}
	lastRefresh := ""
	if auth != nil && strings.EqualFold(auth.Provider, "codex") {
		tokens.AccountID, _ = auth.Metadata["account_id"].(string)
		tokens.IDToken, _ = auth.Metadata["id_token"].(string)
		tokens.AccessToken, _ = auth.Metadata["access_token"].(string)
		tokens.RefreshToken, _ = auth.Metadata["refresh_token"].(string)
		lastRefresh, _ = auth.Metadata["last_refresh"].(string)
		if tokens.IDToken != "" && tokens.AccessToken != "" && tokens.RefreshToken != "" {
			paths = b.sources[strings.ToLower(tokens.AccountID)]
			for _, sourcePath := range paths {
				if err := checkCLIFile(sourcePath, tokens, lastRefresh); err != nil {
					return "", fmt.Errorf("check Codex CLI file %s: %w", filepath.Base(sourcePath), err)
				}
			}
		}
	}
	path, err := b.store.Save(ctx, auth)
	if err != nil || path == "" || len(paths) == 0 {
		return path, err
	}
	for _, sourcePath := range paths {
		if errSync := syncCLIFile(sourcePath, tokens, lastRefresh); errSync != nil {
			return path, fmt.Errorf("sync Codex CLI file %s: %w", filepath.Base(sourcePath), errSync)
		}
	}
	return path, nil
}

func checkCLIFile(path string, tokens cliTokens, lastRefresh string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var credential cliCredential
	if err = json.Unmarshal(data, &credential); err != nil {
		return err
	}
	if credential.AuthMode != "chatgpt" || !strings.EqualFold(credential.Tokens.AccountID, tokens.AccountID) {
		return errors.New("source account changed")
	}
	_, err = shouldUpdateCredential(credential.Tokens, credential.LastRefresh, tokens, lastRefresh)
	return err
}

func (b *credentialBridge) Delete(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.store.Delete(ctx, id)
}

func syncCLIFile(path string, tokens cliTokens, lastRefresh string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document map[string]json.RawMessage
	if err = json.Unmarshal(data, &document); err != nil {
		return err
	}
	var credential cliCredential
	if err = json.Unmarshal(data, &credential); err != nil {
		return err
	}
	if credential.AuthMode != "chatgpt" || !strings.EqualFold(credential.Tokens.AccountID, tokens.AccountID) {
		return errors.New("source account changed")
	}
	update, err := shouldUpdateCredential(credential.Tokens, credential.LastRefresh, tokens, lastRefresh)
	if err != nil || !update {
		return err
	}
	var tokenFields map[string]json.RawMessage
	if err = json.Unmarshal(document["tokens"], &tokenFields); err != nil {
		return err
	}
	for key, value := range map[string]string{
		"id_token": tokens.IDToken, "access_token": tokens.AccessToken,
		"refresh_token": tokens.RefreshToken, "account_id": tokens.AccountID,
	} {
		tokenFields[key], _ = json.Marshal(value)
	}
	document["tokens"], err = json.Marshal(tokenFields)
	if err != nil {
		return err
	}
	if lastRefresh != "" {
		document["last_refresh"], _ = json.Marshal(lastRefresh)
	}
	updated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return replaceCredential(path, append(updated, '\n'))
}

func syncProxyFile(path string, tokens cliTokens, lastRefresh string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document map[string]json.RawMessage
	if err = json.Unmarshal(data, &document); err != nil {
		return err
	}
	var credential struct {
		Type         string `json:"type"`
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
		LastRefresh  string `json:"last_refresh"`
	}
	if err = json.Unmarshal(data, &credential); err != nil {
		return err
	}
	if !strings.EqualFold(credential.Type, "codex") || !strings.EqualFold(credential.AccountID, tokens.AccountID) {
		return errors.New("proxy account changed")
	}
	currentTokens := cliTokens{
		IDToken: credential.IDToken, AccessToken: credential.AccessToken,
		RefreshToken: credential.RefreshToken, AccountID: credential.AccountID,
	}
	update, err := shouldUpdateCredential(currentTokens, credential.LastRefresh, tokens, lastRefresh)
	if err != nil || !update {
		return err
	}
	for key, value := range map[string]string{
		"id_token": tokens.IDToken, "access_token": tokens.AccessToken,
		"refresh_token": tokens.RefreshToken, "account_id": tokens.AccountID,
	} {
		document[key], _ = json.Marshal(value)
	}
	if lastRefresh != "" {
		document["last_refresh"], _ = json.Marshal(lastRefresh)
	}
	updated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	return replaceCredential(path, append(updated, '\n'))
}

func shouldUpdateCredential(currentTokens cliTokens, currentRefresh string, tokens cliTokens, lastRefresh string) (bool, error) {
	if sameTokens(currentTokens, tokens) && (lastRefresh == "" || currentRefresh == lastRefresh) {
		return false, nil
	}
	currentTime, err := parseRefreshTime(currentRefresh)
	if err != nil {
		return false, errors.New("existing credential has invalid last_refresh")
	}
	newTime, err := parseRefreshTime(lastRefresh)
	if err != nil {
		return false, errors.New("updated credential has invalid last_refresh")
	}
	if currentTime.After(newTime) {
		if sameTokens(currentTokens, tokens) {
			return false, nil
		}
		return false, errors.New("credential was refreshed more recently")
	}
	if !currentTime.IsZero() && currentTime.Equal(newTime) && !sameTokens(currentTokens, tokens) {
		return false, errors.New("credential has conflicting tokens at the same last_refresh")
	}
	return true, nil
}

func writeNewCredential(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return errors.New("proxy credential filename already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := stageCredential(path, data)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Link(tmp, path)
}

func replaceCredential(path string, data []byte) error {
	tmp, err := stageCredential(path, data)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Rename(tmp, path)
}

func stageCredential(path string, data []byte) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".llm-proxy-*")
	if err != nil {
		return "", err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	if errClose := file.Close(); err == nil {
		err = errClose
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

var _ coreauth.Store = (*credentialBridge)(nil)
