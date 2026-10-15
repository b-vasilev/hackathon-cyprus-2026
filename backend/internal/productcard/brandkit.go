// Package productcard freezes brand guidance separately from product evidence.
package productcard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"golang.org/x/net/html"
)

const MaxBytes = 100000

// Brandkit accepts either a public URL or the text of an uploaded file.
// Text also stores the downloaded URL content so generations use a fixed snapshot.
type Brandkit struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Text     string `json:"text"`
}

// ResolveBrandkit validates an upload or downloads a public text/HTML brand guide once.
func ResolveBrandkit(ctx context.Context, kit *Brandkit) (*Brandkit, error) {
	if kit == nil {
		return nil, nil //nolint:nilnil // Branding is optional; nil means no kit was supplied.
	}
	resolved := *kit
	resolved.URL = strings.TrimSpace(resolved.URL)
	if resolved.URL != "" {
		if resolved.FileName != "" || resolved.Text != "" {
			return nil, errors.New("provide a product card brand kit URL or a file, not both")
		}
		text, err := download(ctx, resolved.URL)
		if err != nil {
			return nil, err
		}
		resolved.Text = text
	} else if err := resolveBrandkitUpload(&resolved); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resolved.Text) == "" || len(resolved.Text) > MaxBytes || !utf8.ValidString(resolved.Text) || strings.ContainsRune(resolved.Text, 0) {
		return nil, errors.New("product card brand kit must contain 1–100000 bytes of UTF-8 text")
	}
	return &resolved, nil
}

func resolveBrandkitUpload(kit *Brandkit) error {
	if len(kit.Text) > MaxBytes {
		return errors.New("product card brand kit files must be 100 KB or smaller")
	}
	switch strings.ToLower(filepath.Ext(kit.FileName)) {
	case ".txt", ".md", ".json", ".css", ".html":
	default:
		return errors.New("product card brand kit files must be TXT, Markdown, JSON, CSS or HTML")
	}
	if len(kit.FileName) > 255 {
		return errors.New("product card brand kit file name is too long")
	}
	if strings.EqualFold(filepath.Ext(kit.FileName), ".html") {
		kit.Text = htmlText(kit.Text)
	}
	return nil
}

func download(ctx context.Context, location string) (string, error) {
	data, mediaType, err := fetchPublic(ctx, location, MaxBytes)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(mediaType, "text/") && mediaType != "application/json" && mediaType != "application/xhtml+xml" {
		return "", errors.New("product card brand kit URL must return text, HTML, CSS or JSON")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", errors.New("product card brand kit must contain UTF-8 text")
	}
	if mediaType == "text/html" || mediaType == "application/xhtml+xml" {
		return htmlText(string(data)), nil
	}
	return string(data), nil
}

func fetchPublic(ctx context.Context, location string, limit int64) ([]byte, string, error) {
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(location) > 2048 {
		return nil, "", errors.New("product brand URL must be a public HTTPS URL without credentials")
	}
	// Reuse the gateway's DNS-pinned public-only transport. No credentials,
	// proxies, private-origin exceptions or redirects are permitted here.
	config := llmgateway.Config{EndpointURL: location, EndpointMode: "full", TimeoutSeconds: 15, MaxConcurrent: 1}
	for name, values := range u.Query() {
		for _, value := range values {
			config.QueryParams = append(config.QueryParams, llmgateway.Parameter{Name: name, Value: value})
		}
	}
	client, err := (llmgateway.NetworkPolicy{}).Client(config, llmgateway.NewLimiter(1, 0, 1, 0))
	if err != nil {
		return nil, "", errors.New("product brand URL must use a public HTTPS hostname")
	}
	defer client.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("create product brand request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", errors.New("could not download product brand source; use a public URL or upload a file")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", errors.New("product brand URL must return a public document directly (no login or redirect)")
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, "", errors.New("product brand download failed or exceeds the size limit")
	}
	return data, mediaType, nil
}

func htmlText(input string) string {
	tokens := html.NewTokenizer(strings.NewReader(input))
	var text strings.Builder
	script := false
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return text.String()
		case html.CommentToken, html.DoctypeToken:
			// Comments and declarations do not contribute brand guidance.
		case html.StartTagToken, html.SelfClosingTagToken:
			tag := tokens.Token()
			if tag.Data == "script" {
				script = true
			}
			if !script {
				text.WriteByte('\n')
				for _, attr := range tag.Attr {
					if attr.Key == "href" || attr.Key == "src" || attr.Key == "alt" || attr.Key == "style" {
						fmt.Fprintf(&text, " %s=%q ", attr.Key, attr.Val)
					}
				}
			}
		case html.EndTagToken:
			name, _ := tokens.TagName()
			if string(name) == "script" {
				script = false
			}
		case html.TextToken:
			if !script {
				text.Write(tokens.Text())
				text.WriteByte('\n')
			}
		}
	}
}
