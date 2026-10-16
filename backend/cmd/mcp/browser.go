package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func browserAuth(ctx context.Context) (*auth.AuthorizationCodeHandler, func(), error) {
	var listen net.ListenConfig
	listener, err := listen.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("listen for browser sign-in: %w", err)
	}
	callback := "http://" + listener.Addr().String() + "/callback"
	results := make(chan *auth.AuthorizationResult, 1)
	failures := make(chan error, 1)
	var stateMu sync.Mutex
	var expectedState string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		stateMu.Lock()
		valid := expectedState != "" && q.Get("state") == expectedState
		if valid {
			expectedState = ""
		}
		stateMu.Unlock()
		if !valid {
			http.Error(w, "Invalid sign-in state.", http.StatusBadRequest)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if q.Get("error") != "" || q.Get("code") == "" {
			select {
			case failures <- errors.New("browser sign-in was canceled or failed"):
			default:
			}
			_, _ = fmt.Fprintln(w, "Connection canceled. You can close this tab.")
			return
		}
		select {
		case results <- &auth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}:
		default:
		}
		_, _ = fmt.Fprintln(w, "Wayfinder connected. You can close this tab.")
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	closeBrowser := func() { _ = server.Close() }
	handler, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName: "Wayfinder local MCP adapter", RedirectURIs: []string{callback},
				GrantTypes: []string{"authorization_code", "refresh_token"}, TokenEndpointAuthMethod: "none",
			},
		},
		RedirectURL: callback, RequestRefreshToken: true,
		AuthorizationCodeFetcher: func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
			u, parseErr := url.Parse(args.URL)
			if parseErr != nil {
				return nil, fmt.Errorf("parse browser sign-in URL: %w", parseErr)
			}
			if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return nil, errors.New("browser sign-in requires an HTTP or HTTPS URL")
			}
			stateMu.Lock()
			expectedState = u.Query().Get("state")
			stateMu.Unlock()
			fmt.Fprintln(os.Stderr, "Sign in to Wayfinder in your browser:", args.URL)
			openBrowser(ctx, args.URL)
			timer := time.NewTimer(5 * time.Minute)
			defer timer.Stop()
			select {
			case result := <-results:
				return result, nil
			case failure := <-failures:
				return nil, failure
			case <-ctx.Done():
				return nil, fmt.Errorf("browser sign-in canceled: %w", ctx.Err())
			case <-timer.C:
				return nil, errors.New("browser sign-in timed out")
			}
		},
	})
	if err != nil {
		closeBrowser()
		return nil, nil, fmt.Errorf("configure browser sign-in: %w", err)
	}
	return handler, closeBrowser, nil
}

//nolint:gosec // A validated HTTP(S) URL is an argument to a fixed browser opener, without a shell.
func openBrowser(ctx context.Context, target string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.CommandContext(ctx, "open", target).Run()
	case "windows":
		_ = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", target).Run()
	default:
		_ = exec.CommandContext(ctx, "xdg-open", target).Run()
	}
}
