package llm

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

//nolint:gocyclo // The protocol matrix verifies nine wire-format variants.
func TestConfiguredProtocolsAndOutputModes(t *testing.T) {
	t.Parallel()
	for _, protocol := range []string{"openai_responses", "openai_chat_completions", "anthropic_messages"} {
		for _, mode := range []string{"json_schema", "tool_call", "prompt_json"} {
			t.Run(protocol+"/"+mode, func(t *testing.T) {
				t.Parallel()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/custom/endpoint" || r.Header.Get("api-key") != "configured-secret" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Api-Key") != "" || r.URL.Query().Get("api-version") != "test" {
						t.Error("wrong endpoint, query or credential forwarding")
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body["model"] != "same-model" {
						t.Error("model was rewritten")
					}
					if mode == "tool_call" && body["tools"] == nil {
						t.Error("tool output was not requested")
					}
					w.Header().Set("Content-Type", "application/json")
					var response string
					switch protocol {
					case "openai_responses":
						if mode == "tool_call" {
							response = `{"id":"resp_1","status":"completed","output":[{"type":"function_call","name":"emit_result","call_id":"call_1","arguments":"{\"status\":\"ok\"}"}],"usage":{"input_tokens":3,"output_tokens":2}}`
						} else {
							response = `{"id":"resp_1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"{\"status\":\"ok\"}"}]}],"usage":{"input_tokens":3,"output_tokens":2}}`
						}
					case "openai_chat_completions":
						if mode == "tool_call" {
							response = `{"id":"chat_1","choices":[{"index":0,"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"emit_result","arguments":"{\"status\":\"ok\"}"}}]}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`
						} else {
							response = `{"id":"chat_1","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"{\"status\":\"ok\"}"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`
						}
					case "anthropic_messages":
						if body["stream"] != true {
							t.Error("Anthropic request was not streamed")
						}
						w.Header().Set("Content-Type", "text/event-stream")
						if mode == "tool_call" {
							response = anthropicStream(`{"type":"tool_use","id":"call_1","name":"emit_result","input":{}}`, `{"type":"input_json_delta","partial_json":"{\"status\":\"ok\"}"}`, "tool_use")
						} else {
							response = anthropicStream(`{"type":"text","text":""}`, `{"type":"text_delta","text":"{\"status\":\"ok\"}"}`, "end_turn")
						}
					}
					_, _ = w.Write([]byte(response))
				}))
				defer server.Close()
				policy, err := llmgateway.NewNetworkPolicy(server.URL)
				if err != nil {
					t.Fatal(err)
				}
				client := Client{Policy: policy, Limiter: llmgateway.NewLimiter(2, 2, 2, 2)}
				config := llmgateway.Config{Protocol: protocol, EndpointURL: server.URL + "/custom/endpoint", EndpointMode: "full", AuthMode: "header", AuthHeader: "api-key", Secret: "configured-secret", QueryParams: []llmgateway.Parameter{{Name: "api-version", Value: "test"}}, TimeoutSeconds: 15, MaxConcurrent: 1, Models: []llmgateway.Model{{ID: "same-model", OutputMode: mode, MaxOutputTokens: 32000}}}
				result, err := client.Generate(t.Context(), config, Request{Model: "same-model", System: "Return JSON.", Input: "Check.", Schema: map[string]any{"type": "object", "properties": map[string]any{"status": map[string]any{"type": "string"}}, "required": []string{"status"}, "additionalProperties": false}})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(result.JSON), "ok") || result.InputTokens != 3 || result.OutputTokens != 2 {
					t.Fatalf("wrong normalized result: %+v", result)
				}
			})
		}
	}
}

// anthropicStream renders a one-block Messages API event stream.
func anthropicStream(block, delta, stopReason string) string {
	events := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"same-model","content":[],"stop_reason":null,"usage":{"input_tokens":3,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":` + block + `}`,
		`{"type":"content_block_delta","index":0,"delta":` + delta + `}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"` + stopReason + `","stop_sequence":null},"usage":{"output_tokens":2}}`,
		`{"type":"message_stop"}`,
	}
	var out strings.Builder
	for _, event := range events {
		var head struct{ Type string }
		_ = json.Unmarshal([]byte(event), &head)
		out.WriteString("event: " + head.Type + "\ndata: " + event + "\n\n")
	}
	return out.String()
}

func TestAnthropicStreamTruncation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(anthropicStream(`{"type":"text","text":""}`, `{"type":"text_delta","text":"{\"status\":"}`, "max_tokens")))
	}))
	defer server.Close()
	policy, err := llmgateway.NewNetworkPolicy(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := Client{Policy: policy, Limiter: llmgateway.NewLimiter(2, 2, 2, 2)}
	config := llmgateway.Config{Protocol: "anthropic_messages", EndpointURL: server.URL, EndpointMode: "base", AuthMode: "none", TimeoutSeconds: 15, MaxConcurrent: 1, Models: []llmgateway.Model{{ID: "same-model", OutputMode: "json_schema", MaxOutputTokens: 64000}}}
	_, err = client.Generate(t.Context(), config, Request{Model: "same-model", Input: "Check.", Schema: map[string]any{"type": "object"}})
	if failure, ok := errors.AsType[*ProviderError](err); !ok || failure.Code != "truncated" {
		t.Fatalf("truncated stream was not reported: %v", err)
	}
}

func TestOutputFailureStates(t *testing.T) {
	t.Parallel()
	for _, response := range []string{`{"status":"incomplete"}`, `{"stop_reason":"refusal"}`, `{"choices":[{"finish_reason":"length"}]}`, `{"content":[{"type":"text","text":"not json"}]}`} {
		if _, err := parseResponse([]byte(response)); err == nil {
			t.Errorf("accepted failed output: %s", response)
		}
	}
}
