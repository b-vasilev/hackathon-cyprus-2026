package llmgateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Azure mode recognizes only direct Azure OpenAI v1 endpoints. An admin can
// select it explicitly for a custom gateway that forwards to Azure.
func isAzureOpenAI(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	direct := strings.HasSuffix(host, ".openai.azure.com") || strings.HasSuffix(host, ".services.ai.azure.com")
	path := strings.TrimRight(u.Path, "/")
	return direct && (path == "/openai/v1" || strings.HasPrefix(path, "/openai/v1/"))
}

// inspectTokenRequest reads only small JSON requests and restores the exact
// bytes for forwarding. Other bodies pass through unchanged.
func inspectTokenRequest(r *http.Request) (string, int) {
	const maxInspectBytes = 1 << 20
	if r.Body == nil || r.ContentLength <= 0 || r.ContentLength > maxInspectBytes ||
		!strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return "", 0
	}
	original := r.Body
	data, err := io.ReadAll(io.LimitReader(original, maxInspectBytes+1))
	if err != nil || len(data) > maxInspectBytes {
		r.Body = struct {
			io.Reader
			io.Closer
		}{io.MultiReader(bytes.NewReader(data), original), original}
		return "", 0
	}
	_ = original.Close()
	r.Body = io.NopCloser(bytes.NewReader(data))
	var payload struct {
		Model           string `json:"model"`
		MaxOutputTokens int    `json:"max_output_tokens"`
		MaxTokens       int    `json:"max_tokens"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", 0
	}
	if len(payload.Model) > 128 {
		return "", 0
	}
	output := max(max(payload.MaxOutputTokens, payload.MaxTokens), 0)
	return payload.Model, max(1, len(data)/4+output)
}

func updateAzureRate(t *targetLimit, meta RequestMeta, headers http.Header, now time.Time) {
	if meta.Model == "" {
		return
	}
	requestLimit, hasRequestLimit := positiveHeaderInt(headers, "X-Ratelimit-Limit-Requests")
	remainingRequests, hasRemainingRequests := nonnegativeHeaderInt(headers, "X-Ratelimit-Remaining-Requests")
	tokenLimit, hasTokenLimit := positiveHeaderInt(headers, "X-Ratelimit-Limit-Tokens")
	remainingTokens, hasRemainingTokens := nonnegativeHeaderInt(headers, "X-Ratelimit-Remaining-Tokens")
	if !hasRemainingRequests && !hasRemainingTokens {
		return
	}
	state := t.models[meta.Model]
	if state == nil {
		completeRequestBudget := hasRequestLimit && hasRemainingRequests
		completeTokenBudget := hasTokenLimit && hasRemainingTokens
		if !completeRequestBudget && !completeTokenBudget {
			return
		}
		if len(t.models) >= 256 {
			t.reclaimUnusedAzure(now)
			if len(t.models) >= 256 {
				return
			}
		}
		state = &rateState{}
		t.models[meta.Model] = state
	}
	started := meta.StartedAt
	if started.IsZero() {
		started = now
	}
	observeRateWindow(&state.requestLimit, &state.remainingRequests, &state.requestReset, &state.lastRequestStart,
		requestLimit, remainingRequests, hasRequestLimit, hasRemainingRequests,
		resetDuration(headers.Get("X-Ratelimit-Reset-Requests")), max(0, state.inFlight-1), started, now)
	observeRateWindow(&state.tokenLimit, &state.remainingTokens, &state.tokenReset, &state.lastTokenStart,
		tokenLimit, remainingTokens, hasTokenLimit, hasRemainingTokens,
		resetDuration(headers.Get("X-Ratelimit-Reset-Tokens")),
		max(0, state.inFlightTokens-max(0, meta.EstimatedTokens)), started, now)
}

func observeRateWindow(currentLimit, currentRemaining *int, currentReset, lastStart *time.Time,
	limit, remaining int, hasLimit, hasRemaining bool, reset time.Duration, reserved int, started, now time.Time,
) {
	if !hasRemaining {
		return
	}
	activeWindow := *currentLimit > 0 && now.Before(*currentReset)
	adjusted := max(0, remaining-reserved)
	// A late older response may establish an exhausted budget after expiry.
	if !activeWindow && started.Before(*lastStart) && (!hasLimit || adjusted > 0) {
		return
	}
	switch {
	case activeWindow:
		*currentRemaining = min(*currentRemaining, adjusted)
		if hasLimit {
			*currentLimit = min(*currentLimit, limit)
		}
	case hasLimit:
		*currentLimit, *currentRemaining = limit, adjusted
	default:
		return
	}
	if started.After(*lastStart) {
		*lastStart = started
	}
	if reset == 0 {
		reset = time.Minute
	}
	deadline := now.Add(reset)
	if !activeWindow || deadline.After(*currentReset) {
		*currentReset = deadline
	}
}

func positiveHeaderInt(headers http.Header, name string) (int, bool) {
	n, ok := nonnegativeHeaderInt(headers, name)
	return n, ok && n > 0
}

func nonnegativeHeaderInt(headers http.Header, name string) (int, bool) {
	n, err := strconv.Atoi(headers.Get(name))
	return n, err == nil && n >= 0
}

func resetDuration(value string) time.Duration {
	if value == "" {
		return 0
	}
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return min(d, time.Hour)
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds <= 0 {
		return 0
	}
	return min(time.Duration(seconds*float64(time.Second)), time.Hour)
}
