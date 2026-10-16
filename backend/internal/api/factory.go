package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learning"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/learningpack"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/mcpauth"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/product"
	"github.com/rs/zerolog"
)

// Factory exposes product, connection and generation operations.
type Factory struct {
	Access      *access.Store
	MCPAuth     *mcpauth.Store
	PublicURL   string
	Learning    *learning.Store
	GitHub      *githubconnection.Store
	GitHubOAuth githubconnection.OAuthConfig
	Service     *generation.Service
	Packs       *learningpack.Service
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
	f.registerAccess(mux)
	f.registerMCP(mux)
	f.registerLearning(mux)
	f.registerGitHub(mux)
	f.registerReview(mux)
	if f.Packs != nil {
		f.registerLearningPacks(mux)
	}
	if f.FrontendDir != "" {
		mux.Handle("GET /", frontend(f.FrontendDir))
	}
	runProduct := func(r *http.Request) (string, error) { return f.Service.RunProduct(r.Context(), r.PathValue("id")) }
	revisionProduct := func(r *http.Request) (string, error) {
		return f.Service.RevisionProduct(r.Context(), r.PathValue("id"))
	}
	mux.HandleFunc("GET /api/factory/status", f.signedIn(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, FactoryStatus{ConnectionsEnabled: f.Service.Connections != nil})
	}))
	// Every signed-in user manages only their own connections.
	mux.HandleFunc("GET /api/llm-connections", f.signedIn(f.connections))
	mux.HandleFunc("POST /api/llm-connections", f.signedIn(f.createConnection))
	mux.HandleFunc("POST /api/llm-connections/models", f.signedIn(f.listConnectionModels))
	mux.HandleFunc("GET /api/llm-connections/{id}", f.signedIn(f.connection))
	mux.HandleFunc("PATCH /api/llm-connections/{id}", f.signedIn(f.updateConnection))
	mux.HandleFunc("DELETE /api/llm-connections/{id}", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		if !f.requireConnections(w) {
			return
		}
		value, err := f.Service.Connections.Delete(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("POST /api/llm-connections/{id}/test", f.signedIn(f.testConnection))
	mux.HandleFunc("GET /api/products", f.signedIn(f.products))
	mux.HandleFunc("POST /api/products", f.admin(f.createProduct))
	mux.HandleFunc("DELETE /api/products/{id}", f.admin(func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.DeleteProduct(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/products/{id}", f.product(false, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Products.Get(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("PUT /api/products/{id}/icon", withInput(f, pathProduct, func(r *http.Request, input product.IconInput) (any, error) {
		return f.Service.Products.SetIcon(r.Context(), r.PathValue("id"), input)
	}))
	mux.HandleFunc("DELETE /api/products/{id}/icon", f.product(true, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Products.RemoveIcon(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("POST /api/products/{id}/generations", f.product(true, pathProduct, f.startGeneration))
	mux.HandleFunc("POST /api/generations/{id}/cancel", f.product(true, runProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Cancel(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/generations/{id}/events", f.product(true, runProduct, f.generationEvents))
	mux.HandleFunc("GET /api/generations/{id}", f.product(true, runProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.GetRun(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/products/{id}/passport-revisions", f.product(true, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.History(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/passport-revisions/{id}", f.product(true, revisionProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Revision(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/products/{id}/passport", f.product(false, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.PublishedRevision(r.Context(), r.PathValue("id"))
		respond(w, r, value, err)
	}))
	mux.HandleFunc("GET /api/passport-revisions/{id}/evidence/{evidenceId}", f.product(false, f.publishedEvidenceProduct, func(w http.ResponseWriter, r *http.Request) {
		value, err := f.Service.Evidence(r.Context(), r.PathValue("id"), r.PathValue("evidenceId"))
		respond(w, r, value, err)
	}))
}

// products lists every product for admins and member or shared products for other users.
func (f *Factory) products(w http.ResponseWriter, r *http.Request) {
	value, err := f.visibleProducts(r.Context(), currentUser(r))
	respond(w, r, value, err)
}

func (f *Factory) visibleProducts(ctx context.Context, user access.User) ([]product.Product, error) {
	value, err := f.Service.Products.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	if user.Role == access.RoleAdmin {
		return value, nil
	}
	memberships, err := f.Access.Memberships(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list product memberships: %w", err)
	}
	visible := make([]product.Product, 0, len(memberships))
	for _, item := range value {
		if slices.ContainsFunc(memberships, func(m access.Membership) bool { return m.ProductID == item.ID }) {
			visible = append(visible, item)
		}
	}
	return visible, nil
}

func (f *Factory) requireConnections(w http.ResponseWriter) bool {
	if f.Service.Connections == nil {
		problem(w, http.StatusServiceUnavailable, "setup_required", "Set CONNECTION_MASTER_KEY to a stable secret of at least 32 characters, then restart the API.")
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
	if errors.Is(err, context.Canceled) && errors.Is(r.Context().Err(), context.Canceled) {
		// The client no longer needs a response; this is not a server failure.
		return
	}
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
	case errors.Is(err, generation.ErrFlagged):
		problem(w, http.StatusConflict, "review_flagged", err.Error())
	case errors.Is(err, generation.ErrReviewInput), errors.Is(err, passport.ErrInvalidEdit):
		problem(w, http.StatusBadRequest, "invalid_review", err.Error())
	case errors.Is(err, llmgateway.ErrInUse):
		problem(w, http.StatusConflict, "connection_in_use", err.Error())
	case errors.Is(err, learning.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_learning_progress", err.Error())
	case errors.Is(err, product.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_product", err.Error())
	case errors.Is(err, llmgateway.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_connection", err.Error())
	case errors.Is(err, access.ErrSession):
		problem(w, http.StatusUnauthorized, "unauthenticated", err.Error())
	case errors.Is(err, access.ErrCredentials):
		problem(w, http.StatusUnauthorized, "invalid_credentials", "Incorrect email or password.")
	case errors.Is(err, access.ErrInvalid):
		problem(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, access.ErrConflict):
		problem(w, http.StatusConflict, "email_taken", "An account with this email already exists.")
	case errors.Is(err, access.ErrLastAdmin):
		problem(w, http.StatusConflict, "last_admin", "At least one admin must remain.")
	case errors.Is(err, access.ErrUnknownUser):
		problem(w, http.StatusNotFound, "unknown_user", "No account uses this email. Ask the person to register first.")
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
