package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/githubconnection"
)

const githubOAuthCookie = "github_oauth_state"

// OAuthURL is the browser destination returned when authorization starts.
type OAuthURL struct {
	URL string `json:"url"`
}

func (f *Factory) registerGitHub(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/git-connection", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if f.GitHub == nil {
			writeJSON(w, http.StatusOK, githubconnection.Status{})
			return
		}
		value, err := f.GitHub.Status(r.Context())
		value.OAuthEnabled = f.GitHubOAuth.Enabled()
		respond(w, r, value, err)
	}))
	mux.HandleFunc("PUT /api/git-connection", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		if !f.requireGitHub(w) {
			return
		}
		input, ok := decode[githubconnection.TokenInput](w, r)
		if !ok {
			return
		}
		value, err := f.GitHub.Connect(r.Context(), input.Token, "token", time.Time{})
		value.OAuthEnabled = f.GitHubOAuth.Enabled()
		respond(w, r, value, err)
	}))
	mux.HandleFunc("DELETE /api/git-connection", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		if !f.requireGitHub(w) {
			return
		}
		err := f.GitHub.Disconnect(r.Context())
		respond(w, r, githubconnection.Status{Enabled: true, OAuthEnabled: f.GitHubOAuth.Enabled()}, err)
	}))
	mux.HandleFunc("POST /api/git-connection/oauth/start", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		if !f.requireGitHub(w) {
			return
		}
		if !f.GitHubOAuth.Enabled() {
			problem(w, http.StatusServiceUnavailable, "github_oauth_disabled", "GitHub browser sign-in is not configured on this server.")
			return
		}
		authorization, cookie, err := f.GitHub.Begin(r.Context(), f.GitHubOAuth)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		f.setGitHubCookie(w, cookie, 600)
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, OAuthURL{URL: authorization})
	}))
	mux.HandleFunc("GET /api/git-connection/oauth/callback", f.signedIn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		f.setGitHubCookie(w, "", -1)
		cookie, err := r.Cookie(githubOAuthCookie)
		if f.GitHub == nil || !f.GitHubOAuth.Enabled() || err != nil || r.URL.Query().Get("error") != "" {
			http.Redirect(w, r, "/connections?github=error", http.StatusSeeOther)
			return
		}
		if err := f.GitHub.Complete(r.Context(), f.GitHubOAuth, cookie.Value, r.URL.Query().Get("state"), r.URL.Query().Get("code")); err != nil {
			http.Redirect(w, r, "/connections?github=error", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/connections?github=connected", http.StatusSeeOther)
	}))
}

func (f *Factory) requireGitHub(w http.ResponseWriter) bool {
	w.Header().Set("Cache-Control", "no-store")
	if f.GitHub != nil {
		return true
	}
	problem(w, http.StatusServiceUnavailable, "github_disabled", "Set a server master key to store GitHub credentials.")
	return false
}

func (f *Factory) setGitHubCookie(w http.ResponseWriter, value string, maxAge int) {
	//nolint:gosec // Secure is required for HTTPS; local HTTP callbacks are permitted only on loopback.
	http.SetCookie(w, &http.Cookie{Name: githubOAuthCookie, Value: value, Path: "/api/git-connection/oauth", HttpOnly: true, Secure: strings.HasPrefix(f.GitHubOAuth.RedirectURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
