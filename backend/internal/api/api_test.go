package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

type testDatabase struct{ err error }

func (d testDatabase) PingContext(context.Context) error { return d.err }

func TestAPI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, method, path, origin string
		databaseErr                error
		status                     int
	}{
		{name: "live", method: "GET", path: "/healthz", status: 200},
		{name: "ready", method: "GET", path: "/readyz", status: 200},
		{name: "unavailable", method: "GET", path: "/readyz", databaseErr: errors.New("private database detail"), status: 503},
		{name: "contract", method: "GET", path: "/api/openapi.json", status: 200},
		{name: "unknown", method: "GET", path: "/missing", status: 404},
		{name: "method", method: "POST", path: "/healthz", status: 405},
		{name: "allowed origin", method: "GET", path: "/healthz", origin: "http://localhost:5173", status: 200},
		{name: "production portal", method: "GET", path: "/api/openapi.json", origin: "https://wayfinder-t1a.eastus2.cloudapp.azure.com", status: 200},
		{name: "production preflight", method: "OPTIONS", path: "/api/ai-connections", origin: "https://wayfinder-t1a.eastus2.cloudapp.azure.com", status: 204},
		{name: "denied origin", method: "GET", path: "/healthz", origin: "https://untrusted.example", status: 403},
		{name: "preflight", method: "OPTIONS", path: "/healthz", origin: "http://localhost:5173", status: 204},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := New(testDatabase{err: tt.databaseErr}, zerolog.New(io.Discard), []string{"http://localhost:5173", "https://wayfinder-t1a.eastus2.cloudapp.azure.com"})
			request := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			request.Header.Set("X-Request-ID", "untrusted-client-id")
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			if tt.method == "OPTIONS" {
				request.Header.Set("Access-Control-Request-Method", "GET")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if id := response.Header().Get("X-Request-ID"); id == "" || id == "untrusted-client-id" {
				t.Errorf("invalid request ID: %q", id)
			}
			if response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("missing security headers")
			}
			if tt.origin != "" && tt.status != 403 && response.Header().Get("Access-Control-Allow-Origin") != tt.origin {
				t.Error("missing CORS origin")
			}
			if tt.status == 403 && response.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Error("untrusted origin accepted")
			}
			if strings.Contains(response.Body.String(), "private database detail") {
				t.Error("database error leaked")
			}
			if tt.path == "/api/openapi.json" && !json.Valid(response.Body.Bytes()) {
				t.Error("invalid OpenAPI JSON")
			}
		})
	}
}

func TestPanicRecovery(t *testing.T) {
	t.Parallel()
	handler := requestContext(zerolog.New(io.Discard), accessLog(recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("private panic detail")
	}))))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "private panic detail") {
		t.Fatalf("unexpected panic response: %d %s", response.Code, response.Body.String())
	}
}
