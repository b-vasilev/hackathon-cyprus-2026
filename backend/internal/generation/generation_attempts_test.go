package generation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

type attemptGenerator struct {
	steps  []string
	inputs []string
	cancel context.CancelFunc
}

func (g *attemptGenerator) Generate(_ context.Context, _ llmgateway.Config, request llm.Request) (llm.Result, error) {
	i := len(g.inputs)
	g.inputs = append(g.inputs, request.Input)
	if i >= len(g.steps) {
		return llm.Result{}, errors.New("unexpected extra request")
	}
	switch step := g.steps[i]; step {
	case "api", "timeout", "rate_limited", "authentication", "invalid_json":
		code := step
		if code == "api" {
			code = "provider_error"
		}
		return llm.Result{}, &llm.ProviderError{Code: code, Message: step}
	case "cancel":
		g.cancel()
		return llm.Result{}, context.Canceled
	case "schema":
		return llm.Result{JSON: []byte(`{}`)}, nil
	case "semantic":
		return llm.Result{JSON: []byte(`{"title":"wrong","question":"question","parts":[]}`)}, nil
	default:
		return llm.Result{JSON: []byte(`{"title":"valid","question":"question","parts":[]}`)}, nil
	}
}

func TestGenerationAttemptBudgets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		steps   string
		wantErr string
	}{
		{name: "sixth API attempt succeeds", steps: strings.Repeat("api ", 5) + "ok"},
		{name: "sixth rate limited attempt succeeds", steps: strings.Repeat("rate_limited ", 5) + "ok"},
		{name: "sixth response succeeds", steps: strings.Repeat("semantic ", 5) + "ok"},
		{name: "independent mixed budgets", steps: strings.Repeat("api schema ", 5) + "ok"},
		{name: "mixed invalid JSON and timeout", steps: strings.Repeat("invalid_json timeout ", 5) + "ok"},
		{name: "API budget exhausted", steps: strings.Repeat("api ", 6), wantErr: "api"},
		{name: "invalid JSON budget exhausted", steps: strings.Repeat("invalid_json ", 6), wantErr: "failed validation"},
		{name: "schema budget exhausted", steps: strings.Repeat("schema ", 6), wantErr: "failed validation"},
		{name: "semantic budget exhausted", steps: strings.Repeat("semantic ", 6), wantErr: "wrong title"},
		{name: "API budget does not reset", steps: strings.Repeat("api schema ", 5) + "api", wantErr: "api"},
		{name: "response budget does not reset", steps: strings.Repeat("schema api ", 5) + "schema", wantErr: "failed validation"},
		{name: "authentication fails immediately", steps: "authentication", wantErr: "authentication"},
		{name: "cancellation stops retries", steps: "cancel", wantErr: "context canceled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRetryFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			generator := &attemptGenerator{steps: strings.Fields(tc.steps), cancel: cancel}
			f.service.LLM = generator
			attempts := 0
			options := generationCall{RetryProvider: true, OnAttempt: func(attempt int) error {
				attempts++
				if attempt != attempts {
					t.Errorf("attempt = %d, want %d", attempt, attempts)
				}
				return nil
			}}
			_, err := generateWith[chapterPlan](ctx, f.service, f.run, "composing", "original input", options, func(plan chapterPlan) error {
				if plan.Title != "valid" {
					return errors.New("wrong title")
				}
				return nil
			})
			if tc.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
			if attempts != len(generator.steps) || len(generator.inputs) != attempts {
				t.Fatalf("requests = %d, callbacks = %d, want %d", len(generator.inputs), attempts, len(generator.steps))
			}
			for _, input := range generator.inputs {
				if !strings.HasPrefix(input, "original input") || strings.Count(input, schemaRepairPrompt) > 1 {
					t.Fatal("repair prompt lost the original input or accumulated earlier responses")
				}
			}
		})
	}
}
