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
