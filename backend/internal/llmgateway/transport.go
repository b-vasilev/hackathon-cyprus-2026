package llmgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Settings and PoolPolicy retain the admission-control contract from Antares.
type Settings struct{ MaxActive, MaxQueue, MaxWaitMS int }

// PoolPolicy limits HTTP requests sharing an upstream credential or quota group.
type PoolPolicy struct {
	PoolKey                                                            string
	MaxActive, MaxQueue, MaxWaitMS, RequestsPerMinute, TokensPerMinute int
	Mode                                                               string
}

// DefaultPoolPolicy supplies conservative per-credential limits.
func DefaultPoolPolicy(key string) PoolPolicy {
	return PoolPolicy{PoolKey: key, MaxActive: 2, MaxQueue: 8, MaxWaitMS: 300000, Mode: "auto"}
}

// NetworkPolicy permits local destinations and public HTTPS, with explicit overrides.
type NetworkPolicy struct{ allowed map[string]bool }

// NewNetworkPolicy parses a comma-separated list such as http://127.0.0.1:11434.
func NewNetworkPolicy(origins string) (NetworkPolicy, error) {
	p := NetworkPolicy{allowed: make(map[string]bool)}
	for origin := range strings.SplitSeq(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
			return p, errors.New("LLM_ALLOWED_PRIVATE_ORIGINS must contain exact HTTP(S) origins")
		}
		p.allowed[strings.ToLower(u.Scheme+"://"+u.Host)] = true
	}
	return p, nil
}

// ValidateEndpoint checks URL syntax; resolved destinations are checked when dialing.
func (p NetworkPolicy) ValidateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: invalid endpoint", ErrInvalid)
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !p.allowed[strings.ToLower(u.Scheme+"://"+u.Host)] && !localIP(ip.Unmap()) && (u.Scheme != "https" || !publicIP(ip.Unmap())) {
		return fmt.Errorf("%w: this endpoint requires an exact origin in LLM_ALLOWED_PRIVATE_ORIGINS", ErrInvalid)
	}
	return nil
}

// Client builds an SDK HTTP client without inheriting unrelated process credentials or proxies.
func (p NetworkPolicy) Client(c Config, limiter *Limiter) (*http.Client, error) {
	if err := p.ValidateEndpoint(c.EndpointURL); err != nil {
		return nil, err
	}
	u, err := url.Parse(c.EndpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse connection endpoint: %w", err)
	}
	allowedPrivate := p.allowed[strings.ToLower(u.Scheme+"://"+u.Host)]
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.ResponseHeaderTimeout = time.Duration(c.TimeoutSeconds) * time.Second
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("split endpoint address: %w", err)
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, errors.New("endpoint DNS lookup failed")
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		for _, address := range addresses {
			ip, ok := netip.AddrFromSlice(address.IP)
			if !ok || (!allowedPrivate && !localIP(ip.Unmap()) && (u.Scheme != "https" || !publicIP(ip.Unmap()))) {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
			if err == nil {
				return conn, nil
			}
		}
		return nil, errors.New("endpoint is unreachable or blocked by the network policy")
	}
	return &http.Client{Transport: &connectionTransport{config: c, upstream: u, base: transport, limiter: limiter}, Timeout: time.Duration(c.TimeoutSeconds) * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

type connectionTransport struct {
	config   Config
	upstream *url.URL
	base     http.RoundTripper
	limiter  *Limiter
}

func (t *connectionTransport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t *connectionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	request := r.Clone(r.Context())
	request.URL.Scheme = t.upstream.Scheme
	request.URL.Host = t.upstream.Host
	request.Host = t.upstream.Host
	path := t.upstream.EscapedPath()
	if t.config.EndpointMode == "base" {
		path = joinPath(path, r.URL.EscapedPath())
	}
	request.URL.Path, _ = url.PathUnescape(path)
	request.URL.RawPath = path
	query := url.Values{}
	for _, p := range t.config.QueryParams {
		query.Add(p.Name, p.Value)
	}
	request.URL.RawQuery = query.Encode()
	for name := range request.Header {
		lower := strings.ToLower(name)
		if lower == "authorization" || lower == "x-api-key" || lower == "api-key" || lower == "cookie" || lower == "forwarded" || strings.HasPrefix(lower, "proxy-") || strings.HasPrefix(lower, "x-forwarded-") {
			request.Header.Del(name)
		}
	}
	for _, h := range t.config.Headers {
		request.Header.Set(h.Name, h.Value)
	}
	if t.config.AuthMode == "header" {
		request.Header.Set(t.config.AuthHeader, t.config.AuthPrefix+t.config.Secret)
	}
	model, estimated := inspectTokenRequest(request)
	pool := connectionPoolKey(t.config)
	policy := DefaultPoolPolicy(pool)
	policy.MaxActive = t.config.MaxConcurrent
	meta := RequestMeta{PoolKey: pool, API: true, Model: model, EstimatedTokens: estimated, AzureOpenAI: isAzureOpenAI(t.config.EndpointURL)}
	finish, err := t.limiter.AcquireRequest(r.Context(), meta, policy)
	if err != nil {
		return nil, fmt.Errorf("admit LLM request: %w", err)
	}
	meta.StartedAt = time.Now()
	response, err := t.base.RoundTrip(request)
	if err != nil {
		finish(0, nil)
		return nil, errors.New("LLM endpoint request failed")
	}
	t.limiter.ObserveResponse(meta, response.StatusCode, response.Header)
	response.Body = &releaseBody{ReadCloser: response.Body, finish: func() { finish(0, nil) }}
	return response, nil
}

func connectionPoolKey(config Config) string {
	if config.QuotaGroup != "" {
		return "group:" + config.QuotaGroup
	}
	sum := sha256.Sum256([]byte(config.EndpointURL + "\x00" + config.AuthHeader + "\x00" + config.AuthPrefix + "\x00" + config.Secret))
	return hex.EncodeToString(sum[:])
}

type releaseBody struct {
	io.ReadCloser
	once   sync.Once
	finish func()
}

func (b *releaseBody) Close() error {
	b.once.Do(b.finish)
	if err := b.ReadCloser.Close(); err != nil {
		return fmt.Errorf("close provider response: %w", err)
	}
	return nil
}
func (b *releaseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.once.Do(b.finish)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("read provider response: %w", err)
	}
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}
	return n, nil
}

// joinPath preserves full endpoints and avoids repeated /v1 segments, as in Antares.
func joinPath(base, incoming string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" || base == "/" {
		return incoming
	}
	a := strings.Split(strings.Trim(base, "/"), "/")
	b := strings.Split(strings.TrimPrefix(incoming, "/"), "/")
	for overlap := min(len(a), len(b)); overlap > 0; overlap-- {
		match := true
		for i := range overlap {
			if a[len(a)-overlap+i] != b[i] {
				match = false
				break
			}
		}
		if match {
			if overlap == len(b) {
				return base
			}
			return base + "/" + strings.Join(b[overlap:], "/")
		}
	}
	return base + incoming
}

func localIP(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate()
}

func publicIP(ip netip.Addr) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15", "2001:db8::/32"} {
		if netip.MustParsePrefix(prefix).Contains(ip) {
			return false
		}
	}
	return true
}
