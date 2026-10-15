package llm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

func TestListModels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, protocol, mode, path, wantPath string }{
		{"base", "openai_responses", "base", "/v1", "/v1/models"},
		{"root", "openai_chat_completions", "base", "", "/v1/models"},
		{"responses", "openai_responses", "full", "/gateway/v1/responses", "/gateway/v1/models"},
		{"chat", "openai_chat_completions", "full", "/deployment/chat/completions", "/deployment/models"},
		{"anthropic", "anthropic_messages", "full", "/v1/messages", "/v1/models"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != tc.wantPath || r.Header.Get("Authorization") != "Bearer saved-key" || r.Header.Get("X-Tenant") != "tenant" || r.URL.Query().Get("api-version") != "test" {
					t.Errorf("incorrect discovery request: %s %s", r.Method, r.URL.Path)
				}
				if tc.protocol == "anthropic_messages" && r.Header.Get("Anthropic-Version") == "" {
					t.Error("missing Anthropic version")
				}
				_, _ = w.Write([]byte(`{"data":[{"id":"z"},{"id":"a"},{"id":"a"},{"id":""}]}`))
			}))
			defer upstream.Close()
			policy, err := llmgateway.NewNetworkPolicy("")
			if err != nil {
				t.Fatal(err)
			}
			client := Client{Policy: policy, Limiter: llmgateway.NewLimiter(2, 2, 2, 2)}
			config := llmgateway.Config{Protocol: tc.protocol, EndpointMode: tc.mode, EndpointURL: upstream.URL + tc.path, AuthMode: "header", AuthHeader: "Authorization", AuthPrefix: "Bearer ", Secret: "saved-key", Headers: []llmgateway.Parameter{{Name: "X-Tenant", Value: "tenant"}}, QueryParams: []llmgateway.Parameter{{Name: "api-version", Value: "test"}}, TimeoutSeconds: 10, MaxConcurrent: 1}
			models, err := client.ListModels(t.Context(), config)
			if err != nil || len(models) != 2 || models[0].ID != "a" || models[1].ID != "z" {
				t.Fatalf("models=%v err=%v", models, err)
			}
		})
	}
}

func TestListModelsErrorsStayPrivate(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{"error":"private-provider-secret"}`, `not JSON`, `{"data":null}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if strings.Contains(body, "error") {
					w.WriteHeader(http.StatusUnauthorized)
				}
				_, _ = w.Write([]byte(body))
			}))
			defer upstream.Close()
			client := Client{Limiter: llmgateway.NewLimiter(1, 1, 1, 1)}
			_, err := client.ListModels(t.Context(), llmgateway.Config{EndpointURL: upstream.URL, EndpointMode: "base", TimeoutSeconds: 10, MaxConcurrent: 1})
			if err == nil || strings.Contains(err.Error(), "private-provider-secret") {
				t.Fatalf("unsafe error: %v", err)
			}
		})
	}
}
