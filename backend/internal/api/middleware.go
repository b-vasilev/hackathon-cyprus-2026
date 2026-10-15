package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/applog"
	"github.com/rs/zerolog"
)

func requestContext(logger zerolog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate our own ID instead of trusting client-supplied log metadata.
		id := rand.Text()
		w.Header().Set("X-Request-ID", id)
		requestLogger := logger.With().Str("request_id", id).Logger()
		ctx := applog.WithRequestID(r.Context(), id)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		next.ServeHTTP(w, r.WithContext(requestLogger.WithContext(ctx)))
	})
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &responseWriter{ResponseWriter: w}
		defer func() {
			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}
			// Route patterns exclude query strings, credentials and dynamic path values.
			zerolog.Ctx(r.Context()).Info().Str("method", r.Method).Str("route", r.Pattern).
				Int("status", status).Dur("duration_ms", time.Since(started)).Msg("http request")
		}()
		next.ServeHTTP(writer, r)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recoveredErr, ok := recovered.(error); ok && errors.Is(recoveredErr, http.ErrAbortHandler) {
					panic(http.ErrAbortHandler)
				}
				zerolog.Ctx(r.Context()).Error().Bytes("stack", debug.Stack()).Msg("http panic recovered")
				if writer, ok := w.(*responseWriter); ok && writer.status != 0 {
					panic(http.ErrAbortHandler)
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func cors(origins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !slices.Contains(origins, origin) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, X-Request-ID")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if status >= 100 && status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	count, err := w.ResponseWriter.Write(data)
	if err != nil {
		return count, fmt.Errorf("write response: %w", err)
	}
	return count, nil
}

// Unwrap lets http.ResponseController reach flushing for future SSE endpoints.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
