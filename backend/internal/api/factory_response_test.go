package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestRespondCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		cancelRequest  bool
		err            error
		responseStatus int
		logStatus      int
		wantError      bool
	}{
		{"client canceled", true, context.Canceled, 0, 499, false},
		{"wrapped client cancellation", true, fmt.Errorf("read generation: %w", context.Canceled), 0, 499, false},
		{"internal cancellation", false, context.Canceled, 500, 500, true},
		{"deadline exceeded", false, context.DeadlineExceeded, 500, 500, true},
		{"failure after client cancellation", true, errors.New("database failure"), 500, 500, true},
		{"completed response", true, nil, 200, 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelRequest {
				cancel()
			}
			var logs bytes.Buffer
			handler := requestContext(zerolog.New(&logs), accessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respond(w, r, map[string]bool{"ok": true}, tt.err)
			})))
			request := httptest.NewRequest(http.MethodGet, "/api/generations/test", http.NoBody).WithContext(ctx)
			response := httptest.NewRecorder()
			response.Code = 0 // Detect whether the handler writes a response at all.
			handler.ServeHTTP(response, request)
			if response.Code != tt.responseStatus {
				t.Fatalf("response status = %d, want %d", response.Code, tt.responseStatus)
			}
			if tt.responseStatus == 0 && response.Body.Len() != 0 {
				t.Fatal("canceled request wrote a response body")
			}
			type logEntry struct {
				Level   string `json:"level"`
				Message string `json:"message"`
				Status  int    `json:"status"`
			}
			entries := make([]logEntry, 0, 2)
			for line := range bytes.SplitSeq(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
				var entry logEntry
				if err := json.Unmarshal(line, &entry); err != nil {
					t.Fatal(err)
				}
				entries = append(entries, entry)
			}
			wantEntries := 1
			if tt.wantError {
				wantEntries++
			}
			if len(entries) != wantEntries {
				t.Fatalf("got %d log entries, want %d: %s", len(entries), wantEntries, &logs)
			}
			if tt.wantError && (entries[0].Level != "error" || entries[0].Message != "factory request failed") {
				t.Fatalf("missing server error log: %+v", entries[0])
			}
			access := entries[len(entries)-1]
			if access.Level != "info" || access.Message != "http request" || access.Status != tt.logStatus {
				t.Fatalf("unexpected access log: %+v", access)
			}
		})
	}
}
