package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendServesClientRoutes(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("app"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := frontend(directory)
	cases := map[string]int{
		"/":                         http.StatusOK,
		"/foundations":              http.StatusOK,
		"/foundations/git":          http.StatusOK,
		"/foundations/achievements": http.StatusOK,
		"/products/abc":             http.StatusOK,
		"/products":                 http.StatusOK,
		"/foundations/git/extra":    http.StatusNotFound,
		"/unknown":                  http.StatusNotFound,
	}
	for route, want := range cases {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, route, http.NoBody))
		if recorder.Code != want {
			t.Errorf("%s: got %d, want %d", route, recorder.Code, want)
		}
	}
}
