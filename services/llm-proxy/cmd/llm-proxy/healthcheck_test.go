package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeModels(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		status     int
		body       string
		wantErr    string
		wantHeader string
	}{
		{name: "available model", key: "service-key", status: http.StatusOK, body: `{"data":[{"id":"model"}]}`, wantHeader: "Bearer service-key"},
		{name: "missing key", status: http.StatusOK, body: `{"data":[{"id":"model"}]}`, wantErr: "key is missing"},
		{name: "unauthorized", key: "service-key", status: http.StatusUnauthorized, wantErr: "status 401"},
		{name: "no models", key: "service-key", status: http.StatusOK, body: `{"data":[]}`, wantErr: "no available models"},
		{name: "invalid response", key: "service-key", status: http.StatusOK, body: `{`, wantErr: "read subscription models"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if tt.wantHeader != "" && r.Header.Get("Authorization") != tt.wantHeader {
					t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), tt.wantHeader)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := probeModels(server.Client(), server.URL+"/v1/models", tt.key)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("probeModels() error = %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("probeModels() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
