// Package analysis builds a source-backed canonical product model incrementally.
package analysis

import (
	"context"
	"fmt"
	"net/http"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

// languageModel retains the gateway's URL joining, credentials, DNS pinning and limiter.
func languageModel(ctx context.Context, config llmgateway.Config, modelID string, client *http.Client) (fantasy.LanguageModel, error) {
	var provider fantasy.Provider
	var err error
	switch config.Protocol {
	case "openai_responses":
		provider, err = openai.New(openai.WithAPIKey("connection-managed"), openai.WithHTTPClient(client), openai.WithUseResponsesAPI(), openai.WithResponsesAPIFunc(func(string) bool { return true }))
	case "openai_chat_completions":
		provider, err = openaicompat.New(openaicompat.WithAPIKey("connection-managed"), openaicompat.WithHTTPClient(client))
	case "anthropic_messages":
		provider, err = anthropic.New(anthropic.WithAPIKey("connection-managed"), anthropic.WithHTTPClient(client))
	default:
		return nil, fmt.Errorf("unsupported analysis protocol: %s", config.Protocol)
	}
	if err != nil {
		return nil, fmt.Errorf("configure analysis provider: %w", err)
	}
	model, err := provider.LanguageModel(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("configure analysis model: %w", err)
	}
	return model, nil
}

func providerOptions(config llmgateway.Config, modelID string) fantasy.ProviderOptions {
	model, err := llmgateway.ModelByID(config, modelID)
	if err != nil || model.ReasoningEffort == "" {
		return nil
	}
	effort := openai.ReasoningEffort(model.ReasoningEffort)
	if config.Protocol == "openai_responses" {
		store := false
		return openai.NewResponsesProviderOptions(&openai.ResponsesProviderOptions{ReasoningEffort: &effort, Store: &store})
	}
	if config.Protocol == "openai_chat_completions" {
		return openai.NewProviderOptions(&openai.ProviderOptions{ReasoningEffort: &effort})
	}
	return nil
}
