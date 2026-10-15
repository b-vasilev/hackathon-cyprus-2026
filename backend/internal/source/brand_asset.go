package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

// FindBrandAssets finds bounded logo candidates independently of text selection.
func (c *Collector) FindBrandAssets(ctx context.Context, snapshot Snapshot) ([]string, error) {
	if err := Validate([]Input{{Kind: "github", Locator: snapshot.Locator, Ref: snapshot.Revision}}); err != nil {
		return nil, err
	}
	if !commitPattern.MatchString(snapshot.Revision) {
		return nil, errors.New("product logo search requires a pinned revision")
	}
	if c.Token != nil {
		token, err := c.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("load GitHub connection for product logos: %w", err)
		}
		if token != "" {
			ctx = context.WithValue(ctx, githubTokenKey{}, token)
		}
	}
	tree, err := c.repositoryTree(ctx, snapshot.Locator, snapshot.Revision)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, file := range tree {
		if !brandAssetFile(file) {
			continue
		}
		paths = append(paths, file.Path)
	}
	slices.Sort(paths)
	return paths[:min(len(paths), 10)], nil
}

func brandAssetFile(file treeFile) bool {
	if file.Size > 256000 || !slices.Contains([]string{".svg", ".png", ".jpg", ".jpeg", ".gif"}, strings.ToLower(path.Ext(file.Path))) {
		return false
	}
	// Reuse source exclusions for dependencies, build outputs, secrets and symlinks.
	probe := file
	probe.Path = strings.TrimSuffix(file.Path, path.Ext(file.Path)) + ".txt"
	if !eligibleFile(probe) {
		return false
	}
	name := strings.ToLower(file.Path)
	if strings.Contains(name, "_test.") || strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
		return false
	}
	for _, part := range []string{"logo", "wordmark", "logotype", "brand", "/mark/"} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return false
}

// ReadBrandAsset reads a referenced logo at the analyzed revision. GitHub
// credentials stay on api.github.com, including for private repository logos.
func (c *Collector) ReadBrandAsset(ctx context.Context, snapshot Snapshot, file string) ([]byte, error) {
	if err := Validate([]Input{{Kind: "github", Locator: snapshot.Locator, Ref: snapshot.Revision}}); err != nil {
		return nil, err
	}
	if !commitPattern.MatchString(snapshot.Revision) || path.Clean(file) != file || strings.HasPrefix(file, "/") || strings.HasPrefix(file, "../") || !slices.Contains([]string{".svg", ".png", ".jpg", ".jpeg", ".gif"}, strings.ToLower(path.Ext(file))) {
		return nil, errors.New("invalid pinned product logo reference")
	}
	u, _ := url.Parse(snapshot.Locator)
	u.Host = "api.github.com"
	u.Path = "/repos" + strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git") + "/contents/" + file
	u.RawQuery = url.Values{"ref": {snapshot.Revision}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create product logo request: %w", err)
	}
	token := c.githubToken(ctx)
	if c.Token != nil {
		if connected, tokenErr := c.Token(ctx); tokenErr != nil {
			return nil, fmt.Errorf("load GitHub connection for product logo: %w", tokenErr)
		} else if connected != "" {
			token = connected
		}
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Accept", "application/vnd.github.raw+json")
	client := http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("could not download the pinned product logo")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product logo unavailable (HTTP %d)", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 256001))
	if err != nil || len(data) > 256000 {
		return nil, errors.New("product logo unavailable or exceeds 256 KB")
	}
	return data, nil
}
