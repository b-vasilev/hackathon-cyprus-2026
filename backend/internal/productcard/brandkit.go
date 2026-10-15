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
		return nil, nil
	}
	copy := *kit
	copy.URL = strings.TrimSpace(copy.URL)
	if copy.URL != "" {
		if copy.FileName != "" || copy.Text != "" {
			return nil, errors.New("provide a product card brand kit URL or a file, not both")
		}
		text, err := download(ctx, copy.URL)
		if err != nil {
			return nil, err
		}
		copy.Text = text
	} else {
		if len(copy.Text) > MaxBytes {
			return nil, errors.New("product card brand kit files must be 100 KB or smaller")
		}
		switch strings.ToLower(filepath.Ext(copy.FileName)) {
		case ".txt", ".md", ".json", ".css", ".html":
		default:
			return nil, errors.New("product card brand kit files must be TXT, Markdown, JSON, CSS or HTML")
		}
		if len(copy.FileName) > 255 {
			return nil, errors.New("product card brand kit file name is too long")
		}
		if strings.EqualFold(filepath.Ext(copy.FileName), ".html") {
			copy.Text = htmlText(copy.Text)
		}
	}
	if strings.TrimSpace(copy.Text) == "" || len(copy.Text) > MaxBytes || !utf8.ValidString(copy.Text) || strings.ContainsRune(copy.Text, 0) {
		return nil, errors.New("product card brand kit must contain 1–100000 bytes of UTF-8 text")
	}
	return &copy, nil
}

func download(ctx context.Context, location string) (string, error) {
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(location) > 2048 {
		return "", errors.New("product card brand kit URL must be a public HTTPS URL without credentials")
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
		return "", errors.New("product card brand kit URL must use a public HTTPS hostname")
	}
	defer client.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return "", fmt.Errorf("create product card brand kit request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("could not download product card brand kit; use a public URL or upload a file")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("product card brand kit URL must return a public document directly (no login or redirect)")
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if !strings.HasPrefix(mediaType, "text/") && mediaType != "application/json" && mediaType != "application/xhtml+xml" {
		return "", errors.New("product card brand kit URL must return text, HTML, CSS or JSON")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxBytes+1))
	if err != nil || len(data) > MaxBytes {
		return "", errors.New("product card brand kit download failed or exceeds 100 KB")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return "", errors.New("product card brand kit must contain UTF-8 text")
	}
	if mediaType == "text/html" || mediaType == "application/xhtml+xml" {
		return htmlText(string(data)), nil
	}
	return string(data), nil
}

func htmlText(input string) string {
	tokens := html.NewTokenizer(strings.NewReader(input))
	var text strings.Builder
	script := false
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return text.String()
		case html.StartTagToken:
			name, _ := tokens.TagName()
			if string(name) == "script" {
				script = true
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
