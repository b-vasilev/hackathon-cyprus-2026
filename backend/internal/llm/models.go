package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

// AvailableModel identifies a model advertised by the provider.
type AvailableModel struct {
	ID string `json:"id"`
}

// ListModels uses the connection's credentials and network policy for discovery.
func (c *Client) ListModels(ctx context.Context, config llmgateway.Config) ([]AvailableModel, error) {
	endpoint, err := url.Parse(config.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse model discovery endpoint: %w", err)
	}
	if config.EndpointMode == "full" {
		suffix := map[string]string{"openai_responses": "/responses", "openai_chat_completions": "/chat/completions", "anthropic_messages": "/messages"}[config.Protocol]
		endpoint.Path = strings.TrimSuffix(strings.TrimRight(endpoint.Path, "/"), suffix) + "/models"
		endpoint.RawPath = ""
	}
	config.EndpointURL = endpoint.String()
	client, err := c.Policy.Client(config, c.Limiter)
	if err != nil {
		return nil, fmt.Errorf("create model discovery client: %w", err)
	}
	defer client.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://provider.invalid/v1/models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create model discovery request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if config.Protocol == "anthropic_messages" {
		request.Header.Set("Anthropic-Version", "2023-06-01")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, &ProviderError{Code: "models_unavailable", Message: "Could not fetch models from the endpoint."}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, &ProviderError{Code: "models_unavailable", Message: "The endpoint did not return a model list. Check the URL, credentials, and support for /models."}
	}
	const maxBytes = 4 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil || len(data) > maxBytes {
		return nil, &ProviderError{Code: "invalid_models", Message: "The model list could not be read or is too large."}
	}
	var result struct {
		Data []AvailableModel `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil || result.Data == nil {
		return nil, &ProviderError{Code: "invalid_models", Message: "The endpoint returned an invalid model list."}
	}
	models := make([]AvailableModel, 0, len(result.Data))
	seen := make(map[string]bool)
	for _, model := range result.Data {
		if strings.TrimSpace(model.ID) != "" && len(model.ID) <= 200 && !seen[model.ID] {
			models = append(models, model)
			seen[model.ID] = true
		}
	}
	slices.SortFunc(models, func(a, b AvailableModel) int { return strings.Compare(a.ID, b.ID) })
	return models, nil
}
