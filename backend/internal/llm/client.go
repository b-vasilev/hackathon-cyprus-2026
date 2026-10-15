// Package llm translates a shared generation request into three supported API protocols.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	aoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	openai "github.com/openai/openai-go/v3"
	ooption "github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// Request contains a bounded task and its expected JSON output contract.
type Request struct {
	Model, System, Input string
	Schema               map[string]any
}

// Result preserves usage and provider diagnostics without credentials.
type Result struct {
	JSON         json.RawMessage `json:"json"`
	RequestID    string          `json:"request_id"`
	InputTokens  int64           `json:"input_tokens"`
	OutputTokens int64           `json:"output_tokens"`
}

// Client executes requests through versioned, user-configured connections.
type Client struct {
	Policy  llmgateway.NetworkPolicy
	Limiter *llmgateway.Limiter
}

// ProviderError is safe to return without leaking provider payloads or URLs.
type ProviderError struct{ Code, Message string }

func (e *ProviderError) Error() string { return e.Message }

// Generate invokes exactly the configured protocol, model and destination.
//
//nolint:goconst // Repeated keys and values are literal provider wire contracts.
func (c *Client) Generate(ctx context.Context, config llmgateway.Config, request Request) (Result, error) {
	model, err := llmgateway.ModelByID(config, request.Model)
	if err != nil {
		return Result{}, fmt.Errorf("configure model request: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()
	httpClient, err := c.Policy.Client(config, c.Limiter)
	if err != nil {
		return Result{}, fmt.Errorf("configure model transport: %w", err)
	}
	defer httpClient.CloseIdleConnections()
	input := request.Input
	if model.OutputMode == "prompt_json" {
		schema, _ := json.Marshal(request.Schema)
		input += "\nReturn only JSON matching this schema, without Markdown fences:\n" + string(schema)
	}
	var response any
	switch config.Protocol {
	case "openai_responses":
		client := openai.NewClient(ooption.WithAPIKey("connection-managed"), ooption.WithHTTPClient(httpClient), ooption.WithMaxRetries(1))
		params := responses.ResponseNewParams{Model: request.Model, Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(input)}, Instructions: openai.String(request.System), MaxOutputTokens: openai.Int(int64(model.MaxOutputTokens))}
		opts := []ooption.RequestOption{ooption.WithJSONSet("store", false)}
		if model.ReasoningEffort != "" {
			opts = append(opts, ooption.WithJSONSet("reasoning.effort", model.ReasoningEffort))
		}
		if model.OutputMode == "json_schema" {
			opts = append(opts, ooption.WithJSONSet("text.format", map[string]any{"type": "json_schema", "name": "passport_result", "strict": true, "schema": request.Schema}))
		}
		if model.OutputMode == "tool_call" {
			opts = append(opts, ooption.WithJSONSet("tools", []any{map[string]any{"type": "function", "name": "emit_result", "description": "Return the completed task as structured data.", "strict": true, "parameters": request.Schema}}), ooption.WithJSONSet("tool_choice", map[string]any{"type": "function", "name": "emit_result"}))
		}
		response, err = client.Responses.New(ctx, params, opts...)
	case "openai_chat_completions":
		client := openai.NewClient(ooption.WithAPIKey("connection-managed"), ooption.WithHTTPClient(httpClient), ooption.WithMaxRetries(1))
		params := openai.ChatCompletionNewParams{Model: request.Model, Messages: []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(request.System), openai.UserMessage(input)}}
		opts := []ooption.RequestOption{ooption.WithJSONSet("max_tokens", model.MaxOutputTokens)}
		if model.ReasoningEffort != "" {
			opts = append(opts, ooption.WithJSONSet("reasoning_effort", model.ReasoningEffort))
		}
		if model.OutputMode == "json_schema" {
			opts = append(opts, ooption.WithJSONSet("response_format", map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "passport_result", "strict": true, "schema": request.Schema}}))
		}
		if model.OutputMode == "tool_call" {
			opts = append(opts, ooption.WithJSONSet("tools", []any{map[string]any{"type": "function", "function": map[string]any{"name": "emit_result", "description": "Return the completed task as structured data.", "strict": true, "parameters": request.Schema}}}), ooption.WithJSONSet("tool_choice", map[string]any{"type": "function", "function": map[string]any{"name": "emit_result"}}))
		}
		response, err = client.Chat.Completions.New(ctx, params, opts...)
	case "anthropic_messages":
		client := anthropic.NewClient(aoption.WithAPIKey("connection-managed"), aoption.WithHTTPClient(httpClient), aoption.WithMaxRetries(1))
		params := anthropic.MessageNewParams{Model: request.Model, MaxTokens: int64(model.MaxOutputTokens), System: []anthropic.TextBlockParam{{Text: request.System}}, Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(input))}}
		opts := []aoption.RequestOption{}
		if model.OutputMode == "json_schema" {
			opts = append(opts, aoption.WithJSONSet("output_config.format", map[string]any{"type": "json_schema", "schema": request.Schema}))
		}
		if model.OutputMode == "tool_call" {
			opts = append(opts, aoption.WithJSONSet("tools", []any{map[string]any{"name": "emit_result", "description": "Return the completed task as structured data.", "input_schema": request.Schema}}), aoption.WithJSONSet("tool_choice", map[string]any{"type": "tool", "name": "emit_result"}))
		}
		// Stream so large output budgets are not refused by the SDK's 10-minute non-streaming guard.
		stream := client.Messages.NewStreaming(ctx, params, opts...)
		var message anthropic.Message
		for err == nil && stream.Next() {
			err = message.Accumulate(stream.Current())
		}
		if err == nil {
			err = stream.Err()
		}
		_ = stream.Close()
		response = message
	default:
		return Result{}, &ProviderError{Code: "unsupported_protocol", Message: "Unsupported connection protocol."}
	}
	if err != nil {
		return Result{}, classify(err, ctx)
	}
	if raw, ok := response.(interface{ RawJSON() string }); ok {
		return parseResponse([]byte(raw.RawJSON()))
	}
	return Result{}, errors.New("SDK response does not expose the original JSON payload")
}

func classify(err error, ctx context.Context) error {
	if ctx.Err() != nil {
		return &ProviderError{Code: "timeout", Message: "The model request timed out or was canceled."}
	}
	status := 0
	if oe, ok := errors.AsType[*openai.Error](err); ok {
		status = oe.StatusCode
	}
	if ae, ok := errors.AsType[*anthropic.Error](err); ok {
		status = ae.StatusCode
	}
	switch status {
	case 401, 403:
		return &ProviderError{Code: "authentication", Message: "The endpoint rejected the configured credentials or model access."}
	case 429:
		return &ProviderError{Code: "rate_limited", Message: "The endpoint is rate limited. Try again later."}
	case 400, 404, 422:
		return &ProviderError{Code: "unsupported_request", Message: "The endpoint rejected the API path, model or output schema. Check the connection settings."}
	default:
		return &ProviderError{Code: "provider_error", Message: "The model endpoint could not complete the request."}
	}
}

type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Arguments string          `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	Content   []block         `json:"content"`
}
type wireResponse struct {
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	StopReason string  `json:"stop_reason"`
	Output     []block `json:"output"`
	Content    []block `json:"content"`
	Choices    []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string `json:"content"`
			Refusal   string `json:"refusal"`
			ToolCalls []struct {
				Function struct {
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		InputTokens      int64 `json:"input_tokens"`
		OutputTokens     int64 `json:"output_tokens"`
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

//nolint:goconst // Response discriminators follow three external wire contracts.
func parseResponse(data []byte) (Result, error) {
	var wire wireResponse
	if err := json.Unmarshal(data, &wire); err != nil {
		return Result{}, fmt.Errorf("could not decode model response: %w", err)
	}
	if wire.Status == "incomplete" || wire.StopReason == "max_tokens" {
		return Result{}, &ProviderError{Code: "truncated", Message: "The model reached its output limit. Increase the configured token budget."}
	}
	if wire.StopReason == "refusal" {
		return Result{}, &ProviderError{Code: "refusal", Message: "The model declined this request."}
	}
	var text strings.Builder
	for _, choice := range wire.Choices {
		if choice.FinishReason == "length" {
			return Result{}, &ProviderError{Code: "truncated", Message: "The model reached its output limit."}
		}
		if choice.Message.Refusal != "" {
			return Result{}, &ProviderError{Code: "refusal", Message: "The model declined this request."}
		}
		text.WriteString(choice.Message.Content)
		for _, tool := range choice.Message.ToolCalls {
			text.WriteString(tool.Function.Arguments)
		}
	}
	for _, b := range append(wire.Output, wire.Content...) {
		if b.Type == "refusal" {
			return Result{}, &ProviderError{Code: "refusal", Message: "The model declined this request."}
		}
		text.WriteString(b.Text)
		text.WriteString(b.Arguments)
		if b.Type == "tool_use" {
			text.Write(b.Input)
		}
		for _, content := range b.Content {
			if content.Type == "refusal" {
				return Result{}, &ProviderError{Code: "refusal", Message: "The model declined this request."}
			}
			text.WriteString(content.Text)
		}
	}
	value := strings.TrimSpace(text.String())
	if after, ok := strings.CutPrefix(value, "```json"); ok {
		value = strings.TrimSpace(strings.TrimSuffix(after, "```"))
	} else if after, ok := strings.CutPrefix(value, "```"); ok {
		value = strings.TrimSpace(strings.TrimSuffix(after, "```"))
	}
	if !json.Valid([]byte(value)) {
		return Result{}, &ProviderError{Code: "invalid_json", Message: "The model did not return valid JSON."}
	}
	return Result{JSON: json.RawMessage(value), RequestID: wire.ID, InputTokens: max(wire.Usage.InputTokens, wire.Usage.PromptTokens), OutputTokens: max(wire.Usage.OutputTokens, wire.Usage.CompletionTokens)}, nil
}
