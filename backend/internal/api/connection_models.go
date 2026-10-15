package api

import (
	"context"
	"net/http"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

// ModelDiscoveryInput carries a draft and an optional saved connection for masked secrets.
type ModelDiscoveryInput struct {
	ConnectionID string            `json:"connection_id"`
	Config       llmgateway.Config `json:"config"`
}

func (f *Factory) listConnectionModels(w http.ResponseWriter, r *http.Request) {
	if !f.requireConnections(w) {
		return
	}
	client, ok := f.Service.LLM.(interface {
		ListModels(context.Context, llmgateway.Config) ([]llm.AvailableModel, error)
	})
	if !ok {
		problem(w, http.StatusServiceUnavailable, "setup_required", "Model discovery is unavailable.")
		return
	}
	input, ok := decode[ModelDiscoveryInput](w, r)
	if !ok {
		return
	}
	// Discovery does not require a name or a configured generation model.
	input.Config.Name = "Model discovery"
	input.Config.Models = []llmgateway.Model{{ID: "discovery", OutputMode: "json_schema", MaxOutputTokens: 256}}
	config, err := f.Service.Connections.Prepare(r.Context(), input.ConnectionID, input.Config)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	models, err := client.ListModels(ctx, config)
	respond(w, r, models, err)
}
