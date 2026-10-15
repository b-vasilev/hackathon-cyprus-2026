package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Pinger is the database contract used by readiness checks.
type Pinger interface {
	PingContext(context.Context) error
}

//go:embed openapi.json
var openAPI []byte

// New creates the API and its process-wide middleware.
func New(database Pinger, logger zerolog.Logger, origins []string, factories ...*Factory) http.Handler {
	mux := http.NewServeMux()
	for _, factory := range factories {
		factory.register(mux)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := database.PingContext(ctx); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("readiness check failed")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(openAPI)
	})
	return requestContext(logger, accessLog(recoverPanic(securityHeaders(cors(origins, mux)))))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// A response write failure usually means the client has disconnected.
	_ = json.NewEncoder(w).Encode(value)
}
