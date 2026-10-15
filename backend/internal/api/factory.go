package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/rs/zerolog"
)

// Factory exposes product, connection and generation operations.
type Factory struct {
	GitHub      *githubconnection.Store
	GitHubOAuth githubconnection.OAuthConfig
	Service     *generation.Service
	Network     llmgateway.NetworkPolicy
	FrontendDir string
}

// Problem is the consistent API error envelope.
type Problem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// Probe selects a configured model for a small live connection check.
type Probe struct {
	Model string `json:"model"`
}

// ProbeResult records the provider's real response to a schema-bound check.
type ProbeResult struct {
	OK           bool   `json:"ok"`
	RequestID    string `json:"request_id"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

// FactoryStatus reports server setup without exposing credentials.
type FactoryStatus struct {
	ConnectionsEnabled bool `json:"connections_enabled"`
}

func (f *Factory) register(mux *http.ServeMux) {
	f.registerGitHub(mux)
	if f.FrontendDir != "" {
		mux.Handle("GET /", frontend(f.FrontendDir))
	}
	mux.HandleFunc("GET /api/factory/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, FactoryStatus{ConnectionsEnabled: f.Service.Connections != nil})
	})
	mux.HandleFunc("GET /api/llm-connections", f.connections)
	mux.HandleFunc("POST /api/llm-connections", f.createConnection)
	mux.HandleFunc("GET /api/llm-connections/{id}", f.connection)
	mux.HandleFunc("PATCH /api/llm-connections/{id}", f.updateConnection)
	mux.HandleFunc("POST /api/llm-connections/{id}/test", f.testConnection)
	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Products.List(r.Context())
		respond(w, r, value, err)
	})
	mux.HandleFunc("POST /api/products", f.createProduct)
	mux.HandleFunc("GET /api/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Products.Get(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	})
	mux.HandleFunc("POST /api/products/{id}/generations", f.startGeneration)
	mux.HandleFunc("GET /api/generations/{id}/events", f.generationEvents)
	mux.HandleFunc("GET /api/generations/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.GetRun(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	})
	mux.HandleFunc("GET /api/products/{id}/passport-revisions", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.History(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	})
	mux.HandleFunc("GET /api/passport-revisions/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Revision(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	})
	mux.HandleFunc("GET /api/products/{id}/passport", func(w http.ResponseWriter, r *http.Request) {
		history, err := f.Service.History(r.Context(), r.PathValue("id"))
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if len(history) == 0 {
			respond(w, r, nil, sql.ErrNoRows)
			return
		}
		value, err := f.Service.Revision(r.Context(), history[0].ID)
		respond(w, r, value, err)
	})
	mux.HandleFunc("GET /api/passport-revisions/{id}/evidence/{evidenceId}", func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Evidence(r.Context(), r.PathValue("id"), r.PathValue("evidenceId"))
		respond(w, r, value, err)
	})
}

func (f *Factory) requireConnections(w http.ResponseWriter) bool {
	if f.Service.Connections == nil {
		problem(w, http.StatusServiceUnavailable, "setup_required", "Set LLM_MASTER_KEY to a stable secret of at least 32 characters, then restart the API.")
		return false
	}
	return true
}

func (f *Factory) connections(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	value, err := f.Service.Connections.List(r.Context())
	respond(w, r, value, err)
}
func (f *Factory) connection(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	value, err := f.Service.Connections.Get(r.Context(), r.PathValue("id"))
	respond(w, r, value, err)
}

func (f *Factory) createConnection(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	input, ok := decode[llmgateway.Config](w, r)
	if !ok {
		return
	}
	if checkErr := f.Network.ValidateEndpoint(input.EndpointURL); checkErr != nil {
		respond(w, r, nil, checkErr)
		return
	}
	value, err := f.Service.Connections.Save(r.Context(), "", input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (f *Factory) updateConnection(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	patch, ok := decode[map[string]json.RawMessage](w, r)
	if !ok {
		return
	}
	current, err := f.Service.Connections.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	//nolint:gosec // Public projection has all secret values redacted.
	encoded, _ := json.Marshal(current.Config)
	var fields map[string]json.RawMessage
	if checkErr := json.Unmarshal(encoded, &fields); checkErr != nil {
		respond(w, r, nil, checkErr)
		return
	}
	for key, value := range patch {
		if _, exists := fields[key]; !exists {
			problem(w, http.StatusBadRequest, "invalid_input", "Unknown connection field.")
			return
		}
		fields[key] = value
	}
	encoded, err = json.Marshal(fields)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	var config llmgateway.Config
	if checkErr := json.Unmarshal(encoded, &config); checkErr != nil {
		problem(w, http.StatusBadRequest, "invalid_input", "Invalid connection configuration.")
		return
	}
	if checkErr := f.Network.ValidateEndpoint(config.EndpointURL); checkErr != nil {
		respond(w, r, nil, checkErr)
		return
	}
	value, err := f.Service.Connections.Save(r.Context(), current.ID, config)
	respond(w, r, value, err)
}

func (f *Factory) testConnection(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	input, ok := decode[Probe](w, r)
	if !ok {
		return
	}
	c, err := f.Service.Connections.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	config, err := f.Service.Connections.Load(r.Context(), c.ID, c.Revision)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	for i := range config.Models {
		config.Models[i].MaxOutputTokens = 1024
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	format := map[string]any{schemaTypeKey: "object", "properties": map[string]any{"status": map[string]any{schemaTypeKey: "string", "enum": []string{"ok"}}}, "required": []string{"status"}, "additionalProperties": false}
	result, err := f.Service.LLM.Generate(ctx, config, llm.Request{Model: input.Model, System: "Return only the requested JSON result.", Input: "Check this connection by returning status equal to ok.", Schema: format})
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	var value struct {
		Status string `json:"status"`
	}
	if checkErr := json.Unmarshal(result.JSON, &value); checkErr != nil || value.Status != "ok" {
		problem(w, http.StatusUnprocessableEntity, "invalid_output", "The endpoint responded, but did not complete the structured-output check.")
		return
	}
	writeJSON(w, http.StatusOK, ProbeResult{OK: true, RequestID: result.RequestID, InputTokens: result.InputTokens, OutputTokens: result.OutputTokens})
}

func (f *Factory) createProduct(w http.ResponseWriter, r *http.Request) {
	input, ok := decode[product.Input](w, r)
	if !ok {
		return
	}
	value, err := f.Service.Products.Create(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (f *Factory) startGeneration(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	input, ok := decode[generation.Input](w, r)
	if !ok {
		return
	}
	value, err := f.Service.Start(r.Context(), r.PathValue("id"), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	w.Header().Set("Location", "/api/generations/"+value.ID)
	writeJSON(w, http.StatusAccepted, value)
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var value T
	mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaErr != nil || mediaType != "application/json" {
		problem(w, http.StatusUnsupportedMediaType, "content_type", "Use Content-Type: application/json.")
		return value, false
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if checkErr := decoder.Decode(&value); checkErr != nil {
		problem(w, http.StatusBadRequest, "invalid_json", "Invalid JSON request or unknown fields.")
		return value, false
	}
	var extra any
	if checkErr := decoder.Decode(&extra); checkErr != io.EOF {
		problem(w, http.StatusBadRequest, "invalid_json", "Provide exactly one JSON value.")
		return value, false
	}
	return value, true
}

func respond(w http.ResponseWriter, r *http.Request, value any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, value)
		return
	}
	var failure *llm.ProviderError
	switch {
	case errors.Is(err, sql.ErrNoRows):
		problem(w, http.StatusNotFound, "not_found", "The requested item was not found or is disabled.")
	case errors.Is(err, generation.ErrActive):
		problem(w, http.StatusConflict, "active_generation", err.Error())
	case errors.Is(err, product.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_product", err.Error())
	case errors.Is(err, llmgateway.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_connection", err.Error())
	case errors.Is(err, githubconnection.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_github_connection", err.Error())
	case errors.As(err, &failure):
		problem(w, http.StatusBadGateway, failure.Code, failure.Message)
	default:
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("factory request failed")
		problem(w, http.StatusInternalServerError, "internal_error", "The operation could not be completed.")
	}
}

func problem(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Problem{Code: code, Message: message, RequestID: w.Header().Get("X-Request-ID")})
}
