package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/mcpauth"
)

// MCPSettings is the account's connection setup and active apps, without secrets.
type MCPSettings struct {
	URL         string               `json:"url"`
	Connections []mcpauth.Connection `json:"connections"`
}

// MCPConsentInput records an explicit allow or deny decision for a validated request.
type MCPConsentInput struct {
	Authorization mcpauth.Authorization `json:"authorization"`
	Allow         bool                  `json:"allow"`
}

// MCPRedirect is a registered callback with the authorization result and client state.
type MCPRedirect struct {
	URL string `json:"redirect_url"`
}

func (f *Factory) registerMCPOAuth(mux *http.ServeMux) {
	resourceMetadata := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"resource": f.PublicURL + "/mcp", "authorization_servers": []string{f.PublicURL},
			"scopes_supported": []string{mcpauth.Scope}, "bearer_methods_supported": []string{"header"},
		})
	}
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", resourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", resourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer": f.PublicURL, "authorization_endpoint": f.PublicURL + "/oauth/authorize",
			"token_endpoint": f.PublicURL + "/oauth/token", "registration_endpoint": f.PublicURL + "/oauth/register",
			"response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
			"token_endpoint_auth_methods_supported": []string{"none"}, "code_challenge_methods_supported": []string{"S256"},
			"scopes_supported": []string{mcpauth.Scope},
			"authorization_response_iss_parameter_supported": true,
		})
	})
	mux.HandleFunc("POST /oauth/register", f.registerMCPClient)
	mux.HandleFunc("POST /oauth/token", f.mcpToken)
	mux.HandleFunc("GET /oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		if _, err := f.MCPAuth.Validate(r.Context(), authorizationQuery(r.URL.Query()), f.PublicURL+"/mcp"); err != nil {
			oauthError(w, "invalid_request", "Invalid client, callback, resource, scope or PKCE challenge.")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		target := url.URL{Path: "/ai-connections/authorize", RawQuery: r.URL.Query().Encode()}
		http.Redirect(w, r, target.String(), http.StatusSeeOther)
	})
	mux.HandleFunc("GET /api/ai-connections", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		connections, err := f.MCPAuth.Connections(r.Context(), currentUser(r).ID)
		w.Header().Set("Cache-Control", "no-store")
		respond(w, r, MCPSettings{URL: f.PublicURL + "/mcp", Connections: connections}, err)
	}))
	mux.HandleFunc("DELETE /api/ai-connections/{id}", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		noContent(w, r, f.MCPAuth.Disconnect(r.Context(), currentUser(r).ID, r.PathValue("id")))
	}))
	mux.HandleFunc("GET /api/ai-connections/authorize", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		client, err := f.MCPAuth.Validate(r.Context(), authorizationQuery(r.URL.Query()), f.PublicURL+"/mcp")
		if err != nil {
			problem(w, http.StatusBadRequest, "invalid_request", "This connection request is invalid or unsupported. Start again from your AI app.")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, client)
	}))
	mux.HandleFunc("POST /api/ai-connections/authorize", f.signedIn(f.authorizeMCP))
	mux.HandleFunc("GET /api/products/{id}/passport.md", f.product(false, pathProduct, func(w http.ResponseWriter, r *http.Request) {
		export, err := f.Service.ExportMarkdown(r.Context(), r.PathValue("id"))
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": export.Filename}))
		_, _ = io.WriteString(w, export.Markdown)
	}))
}

func authorizationQuery(q url.Values) mcpauth.Authorization {
	return mcpauth.Authorization{
		ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"), ResponseType: q.Get("response_type"),
		CodeChallenge: q.Get("code_challenge"), CodeChallengeMethod: q.Get("code_challenge_method"),
		Resource: q.Get("resource"), Scope: q.Get("scope"), State: q.Get("state"),
	}
}

func oauthError(w http.ResponseWriter, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": description})
}

func (f *Factory) registerMCPClient(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		oauthError(w, "invalid_client_metadata", "Use Content-Type: application/json.")
		return
	}
	var input struct {
		Name         string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		AuthMethod   string   `json:"token_endpoint_auth_method"`
		GrantTypes   []string `json:"grant_types"`
		Responses    []string `json:"response_types"`
	}
	decoder := json.NewDecoder(r.Body)
	var extra any
	if err := decoder.Decode(&input); err != nil {
		oauthError(w, "invalid_client_metadata", "Provide valid client metadata.")
		return
	}
	if err := decoder.Decode(&extra); err != io.EOF || (input.AuthMethod != "" && input.AuthMethod != "none") ||
		slices.ContainsFunc(input.GrantTypes, func(g string) bool { return g != "authorization_code" && g != "refresh_token" }) ||
		slices.ContainsFunc(input.Responses, func(g string) bool { return g != "code" }) {
		oauthError(w, "invalid_client_metadata", "Only public authorization-code clients with PKCE are supported.")
		return
	}
	client, err := f.MCPAuth.Register(r.Context(), input.Name, input.RedirectURIs)
	if err != nil {
		oauthError(w, "invalid_client_metadata", "Use an HTTPS callback or local loopback HTTP callback.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id": client.ID, "client_name": client.Name, "redirect_uris": client.RedirectURIs,
		"token_endpoint_auth_method": "none", "grant_types": []string{"authorization_code", "refresh_token"}, "response_types": []string{"code"},
	})
}

func (f *Factory) authorizeMCP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	input, ok := decode[MCPConsentInput](w, r)
	if !ok {
		return
	}
	a := input.Authorization
	if _, err := f.MCPAuth.Validate(r.Context(), a, f.PublicURL+"/mcp"); err != nil {
		problem(w, http.StatusBadRequest, "invalid_request", "Invalid connection request. Start again from your AI app.")
		return
	}
	u, err := url.Parse(a.RedirectURI)
	if err != nil {
		problem(w, http.StatusBadRequest, "invalid_request", "Invalid callback URL.")
		return
	}
	q := u.Query()
	q.Set("state", a.State)
	q.Set("iss", f.PublicURL)
	if input.Allow {
		code, err := f.MCPAuth.Authorize(r.Context(), currentUser(r).ID, a, f.PublicURL+"/mcp")
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		q.Set("code", code)
	} else {
		q.Set("error", "access_denied")
	}
	u.RawQuery = q.Encode()
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, MCPRedirect{URL: u.String()})
}

func (f *Factory) mcpToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		oauthError(w, "invalid_request", "Provide form-encoded token parameters.")
		return
	}
	var tokens mcpauth.Tokens
	var err error
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		tokens, err = f.MCPAuth.Exchange(r.Context(), r.PostForm, f.PublicURL+"/mcp")
	case "refresh_token":
		tokens, err = f.MCPAuth.Refresh(r.Context(), r.PostForm, f.PublicURL+"/mcp")
	default:
		oauthError(w, "unsupported_grant_type", "Use authorization_code or refresh_token.")
		return
	}
	if err != nil {
		oauthError(w, "invalid_grant", "The authorization is invalid, expired, revoked or already used.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, tokens)
}
