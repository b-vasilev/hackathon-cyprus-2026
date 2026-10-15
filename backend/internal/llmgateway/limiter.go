package llmgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	ErrQueueFull    = errors.New("LLM gateway queue is full")
	ErrQueueTimeout = errors.New("LLM gateway queue wait expired")
	ErrTokenBudget  = errors.New("request exceeds LLM gateway token budget")
)

type RequestMeta struct {
	PoolKey         string
	API             bool
	AzureOpenAI     bool
	Model           string
	EstimatedTokens int
	StartedAt       time.Time
}

type rateState struct {
	requestLimit      int
	remainingRequests int
	requestReset      time.Time
	tokenLimit        int
	remainingTokens   int
	tokenReset        time.Time
	inFlight          int
	inFlightTokens    int
	lastRequestStart  time.Time
	lastTokenStart    time.Time
}

func (r *rateState) hasUsableBudget(now time.Time) bool {
	return (r.requestLimit > 0 && now.Before(r.requestReset)) ||
		(r.tokenLimit > 0 && now.Before(r.tokenReset))
}

type targetLimit struct {
	active       int
	waiting      int
	limit        int
	successes    int
	throttles    int
	total429     int
	total503     int
	cooldownTill time.Time
	policy       PoolPolicy
	requests     []time.Time
	tokens       []tokenReservation
	models       map[string]*rateState
}

func (t *targetLimit) reserveAzure(meta RequestMeta) bool {
	if !meta.AzureOpenAI || meta.Model == "" {
		return false
	}
	rate := t.models[meta.Model]
	if rate == nil {
		if len(t.models) >= 256 {
			t.reclaimUnusedAzure(time.Now())
			if len(t.models) >= 256 {
				return false
			}
		}
		rate = &rateState{}
		t.models[meta.Model] = rate
	}
	rate.inFlight++
	rate.inFlightTokens += max(0, meta.EstimatedTokens)
	if rate.requestLimit > 0 && rate.remainingRequests > 0 {
		rate.remainingRequests--
	}
	if rate.tokenLimit > 0 {
		rate.remainingTokens = max(0, rate.remainingTokens-meta.EstimatedTokens)
	}
	return true
}

func (t *targetLimit) releaseAzure(meta RequestMeta) {
	if !meta.AzureOpenAI || meta.Model == "" {
		return
	}
	if rate := t.models[meta.Model]; rate != nil {
		rate.inFlight--
		rate.inFlightTokens -= max(0, meta.EstimatedTokens)
		if rate.inFlight == 0 && !rate.hasUsableBudget(time.Now()) {
			delete(t.models, meta.Model)
		}
	}
}

func (t *targetLimit) reclaimUnusedAzure(now time.Time) {
	for model, rate := range t.models {
		if rate.inFlight == 0 && !rate.hasUsableBudget(now) {
			delete(t.models, model)
		}
	}
}

type tokenReservation struct {
	at     time.Time
	tokens int
}

type RateStatus struct {
	RequestLimit      int       `json:"request_limit"`
	RemainingRequests int       `json:"remaining_requests"`
	RequestReset      time.Time `json:"request_reset,omitzero"`
	TokenLimit        int       `json:"token_limit"`
	RemainingTokens   int       `json:"remaining_tokens"`
	TokenReset        time.Time `json:"token_reset,omitzero"`
}

type PoolStatus struct {
	PoolKey        string                `json:"-"`
	Active         int                   `json:"active"`
	Waiting        int                   `json:"waiting"`
	EffectiveLimit int                   `json:"effective_limit"`
	CooldownUntil  time.Time             `json:"cooldown_until,omitzero"`
	Total429       int                   `json:"total_429"`
	Total503       int                   `json:"total_503"`
	Models         map[string]RateStatus `json:"models,omitempty"`
}

type Status struct {
	Active  int          `json:"active"`
	Waiting int          `json:"waiting"`
	Pools   []PoolStatus `json:"pools"`
}

type waiter struct {
	target *targetLimit
	meta   RequestMeta
}

type Limiter struct {
	mu            sync.Mutex
	changed       chan struct{}
	globalActive  int
	globalWaiting int
	lookupWaiting int
	globalMax     int
	globalQueue   int
	globalWait    time.Duration
	apiMax        int
	apiQueue      int
	targets       map[string]*targetLimit
	waiters       []*waiter
}

type queueWaitKey struct{}

// WithQueueWait makes bounded background workers wait for queue space instead
// of treating local admission pressure as a failed provider request. The normal
// admission deadline and context cancellation still apply.
func WithQueueWait(ctx context.Context) context.Context {
	return context.WithValue(ctx, queueWaitKey{}, true)
}

func NewLimiter(globalMax, globalQueue, apiMax, apiQueue int) *Limiter {
	return &Limiter{changed: make(chan struct{}), globalMax: globalMax,
		globalQueue: globalQueue, globalWait: time.Minute, apiMax: apiMax, apiQueue: apiQueue,
		targets: make(map[string]*targetLimit)}
}

func (l *Limiter) SetGlobal(v Settings) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.globalMax, l.globalQueue = v.MaxActive, v.MaxQueue
	l.globalWait = time.Duration(v.MaxWaitMS) * time.Millisecond
	l.signal()
}

// ReserveLookupWait shares the configured global queue with requests whose
// credentials have already been resolved. The returned release must be called
// when a lookup slot is acquired or the wait ends.
func (l *Limiter) ReserveLookupWait() (time.Duration, func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.globalWaiting+l.lookupWaiting >= l.globalQueue {
		return 0, nil, false
	}
	l.lookupWaiting++
	return l.globalWait, func() {
		l.mu.Lock()
		l.lookupWaiting--
		l.mu.Unlock()
	}, true
}

func (l *Limiter) SetPoolPolicy(v PoolPolicy) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if t := l.targets[v.PoolKey]; t != nil {
		l.applyPolicy(t, v)
		l.signal()
	}
}

func (l *Limiter) applyPolicy(t *targetLimit, v PoolPolicy) {
	if t.limit == t.policy.MaxActive {
		t.limit = v.MaxActive
	} else if t.limit > v.MaxActive {
		t.limit = v.MaxActive
	}
	t.policy = v
}

// Acquire preserves the original interface for local test callers.
func (l *Limiter) Acquire(ctx context.Context, targetID string, api bool) (func(int, string), error) {
	policy := DefaultPoolPolicy(targetID)
	policy.MaxActive, policy.MaxQueue = l.apiMax, l.apiQueue
	finish, err := l.AcquireRequest(ctx, RequestMeta{PoolKey: targetID, API: api}, policy)
	if err != nil {
		return nil, err
	}
	return func(status int, retryAfter string) {
		finish(status, http.Header{"Retry-After": []string{retryAfter}})
	}, nil
}

func (l *Limiter) AcquireRequest(ctx context.Context, meta RequestMeta, policy PoolPolicy) (func(int, http.Header), error) { //nolint:gocyclo // Admission keeps capacity, queue, timeout, and cancellation transitions together.
	waitForQueue, _ := ctx.Value(queueWaitKey{}).(bool)
	if meta.StartedAt.IsZero() {
		meta.StartedAt = time.Now()
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("acquire LLM gateway request: %w", err)
	}
	if meta.API && policy.TokensPerMinute > 0 && meta.EstimatedTokens > policy.TokensPerMinute {
		return nil, ErrTokenBudget
	}
	l.mu.Lock()
	var target *targetLimit
	if meta.API {
		target = l.targets[meta.PoolKey]
		if target == nil {
			target = &targetLimit{limit: policy.MaxActive, policy: policy, models: make(map[string]*rateState)}
			l.targets[meta.PoolKey] = target
		} else {
			l.applyPolicy(target, policy)
		}
	}
	waitLimit := l.globalWait
	if target != nil && time.Duration(policy.MaxWaitMS)*time.Millisecond < waitLimit {
		waitLimit = time.Duration(policy.MaxWaitMS) * time.Millisecond
	}
	deadline := time.Now().Add(waitLimit)
	var w *waiter
	waited := false
	for {
		now := time.Now()
		if waited && !now.Before(deadline) {
			l.removeWaiter(w)
			l.signal()
			l.mu.Unlock()
			return nil, ErrQueueTimeout
		}
		if target != nil && target.policy.TokensPerMinute > 0 && meta.EstimatedTokens > target.policy.TokensPerMinute {
			if w != nil {
				l.removeWaiter(w)
				l.signal()
			}
			l.mu.Unlock()
			return nil, ErrTokenBudget
		}
		allowed, next := l.canStart(target, meta, now)
		if allowed && ((w == nil && l.firstEligibleWaiter(now) == nil && !l.hasPoolWaiter(target)) || //nolint:nestif // Queue ownership must be checked before reserving capacity.
			(w != nil && l.firstEligibleWaiter(now) == w)) {
			if w != nil {
				l.removeWaiter(w)
			}
			l.globalActive++
			azureReserved := false
			if target != nil {
				target.active++
				if target.policy.RequestsPerMinute > 0 {
					target.requests = append(target.requests, now)
				}
				if target.policy.TokensPerMinute > 0 && meta.EstimatedTokens > 0 {
					target.tokens = append(target.tokens, tokenReservation{at: now, tokens: meta.EstimatedTokens})
				}
				azureReserved = target.reserveAzure(meta)
			}
			l.signal()
			l.mu.Unlock()
			return func(status int, headers http.Header) { l.complete(meta, status, headers, azureReserved) }, nil
		}
		if w == nil {
			full := l.globalWaiting+l.lookupWaiting >= l.globalQueue || (target != nil && target.waiting >= target.policy.MaxQueue)
			if full && !waitForQueue {
				l.mu.Unlock()
				return nil, ErrQueueFull
			}
			if !full {
				w = &waiter{target: target, meta: meta}
				l.waiters = append(l.waiters, w)
				l.globalWaiting++
				if target != nil {
					target.waiting++
				}
			}
		}
		if !now.Before(deadline) {
			l.removeWaiter(w)
			l.signal()
			l.mu.Unlock()
			return nil, ErrQueueTimeout
		}
		wake := deadline
		if !next.IsZero() && next.Before(wake) {
			wake = next
		}
		changed := l.changed
		waited = true
		l.mu.Unlock()
		timer := time.NewTimer(time.Until(wake))
		select {
		case <-ctx.Done():
		case <-changed:
		case <-timer.C:
		}
		timer.Stop()
		l.mu.Lock()
		if err := ctx.Err(); err != nil {
			l.removeWaiter(w)
			l.signal()
			l.mu.Unlock()
			return nil, fmt.Errorf("wait for LLM gateway request: %w", err)
		}
	}
}

func (l *Limiter) canStart(t *targetLimit, meta RequestMeta, now time.Time) (bool, time.Time) { //nolint:gocyclo // Each active rate window has its own deadline and budget check.
	if l.globalActive >= l.globalMax {
		return false, time.Time{}
	}
	if t == nil {
		return true, time.Time{}
	}
	if t.active >= t.limit {
		return false, time.Time{}
	}
	if now.Before(t.cooldownTill) {
		return false, t.cooldownTill
	}
	cutoff := now.Add(-time.Minute)
	i := 0
	for i < len(t.requests) && !t.requests[i].After(cutoff) {
		i++
	}
	t.requests = t.requests[i:]
	if t.policy.RequestsPerMinute > 0 && len(t.requests) >= t.policy.RequestsPerMinute {
		return false, t.requests[0].Add(time.Minute)
	}
	usedTokens := 0
	i = 0
	for i < len(t.tokens) && !t.tokens[i].at.After(cutoff) {
		i++
	}
	t.tokens = t.tokens[i:]
	if t.policy.TokensPerMinute > 0 && meta.EstimatedTokens > 0 {
		for _, entry := range t.tokens {
			usedTokens += entry.tokens
		}
		if usedTokens+meta.EstimatedTokens > t.policy.TokensPerMinute && len(t.tokens) > 0 {
			return false, t.tokens[0].at.Add(time.Minute)
		}
	}
	if meta.AzureOpenAI { //nolint:nestif // Azure request and token windows are independent.
		if rate := t.models[meta.Model]; rate != nil {
			if !rate.requestReset.IsZero() && !now.Before(rate.requestReset) {
				rate.requestLimit = 0
			}
			if !rate.tokenReset.IsZero() && !now.Before(rate.tokenReset) {
				rate.tokenLimit = 0
			}
			if rate.requestLimit > 0 && rate.remainingRequests <= 0 && now.Before(rate.requestReset) {
				return false, rate.requestReset
			}
			if rate.tokenLimit > 0 && rate.remainingTokens < meta.EstimatedTokens && now.Before(rate.tokenReset) {
				return false, rate.tokenReset
			}
		}
	}
	return true, time.Time{}
}

func (l *Limiter) hasPoolWaiter(t *targetLimit) bool {
	for _, w := range l.waiters {
		if w.target == t {
			return true
		}
	}
	return false
}

func (l *Limiter) firstEligibleWaiter(now time.Time) *waiter {
	seen := make(map[*targetLimit]bool)
	for _, w := range l.waiters {
		if seen[w.target] {
			continue
		}
		seen[w.target] = true
		if ok, _ := l.canStart(w.target, w.meta, now); ok {
			return w
		}
	}
	return nil
}

func (l *Limiter) removeWaiter(w *waiter) {
	for i, queued := range l.waiters {
		if queued == w {
			l.waiters = append(l.waiters[:i], l.waiters[i+1:]...)
			l.globalWaiting--
			if w.target != nil {
				w.target.waiting--
			}
			return
		}
	}
}

func (l *Limiter) complete(meta RequestMeta, status int, headers http.Header, azureReserved bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if status != 0 {
		l.observeResponseLocked(meta, status, headers, time.Now())
	}
	l.globalActive--
	if meta.API {
		t := l.targets[meta.PoolKey]
		t.active--
		if azureReserved {
			t.releaseAzure(meta)
		}
	}
	l.signal()
}

// ObserveResponse applies upstream limits as soon as headers arrive, while
// the request still occupies an execution slot during body streaming.
func (l *Limiter) ObserveResponse(meta RequestMeta, status int, headers http.Header) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.observeResponseLocked(meta, status, headers, time.Now())
	l.signal()
}

func (l *Limiter) observeResponseLocked(meta RequestMeta, status int, headers http.Header, now time.Time) {
	if !meta.API {
		return
	}
	t := l.targets[meta.PoolKey]
	if meta.AzureOpenAI {
		updateAzureRate(t, meta, headers, now)
	}
	switch status {
	case http.StatusTooManyRequests:
		t.total429++
		t.throttles++
		t.successes = 0
		if t.throttles >= 2 && t.limit > 1 {
			t.limit = (t.limit + 1) / 2
			t.throttles = 0
		}
		l.cooldown(t, headers)
	case http.StatusServiceUnavailable:
		t.total503++
		t.successes = 0
		if t.limit > 1 {
			t.limit = (t.limit + 1) / 2
		}
		l.cooldown(t, headers)
	default:
		if status >= 200 && status < 400 {
			t.throttles = 0
			t.successes++
			if t.successes >= 8 && t.limit < t.policy.MaxActive {
				t.limit++
				t.successes = 0
			}
		}
	}
}

func (l *Limiter) cooldown(t *targetLimit, headers http.Header) {
	delay := parseRetryAfterMS(headers.Get("Retry-After-Ms"))
	if delay == 0 {
		delay = parseRetryAfter(headers.Get("Retry-After"))
	}
	if delay == 0 {
		delay = 5 * time.Second
	}
	until := time.Now().Add(delay)
	if until.After(t.cooldownTill) {
		t.cooldownTill = until
	}
}

func (l *Limiter) signal() {
	close(l.changed)
	l.changed = make(chan struct{})
}

func (l *Limiter) Snapshot() Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := Status{Active: l.globalActive, Waiting: l.globalWaiting + l.lookupWaiting, Pools: make([]PoolStatus, 0, len(l.targets))}
	for key, t := range l.targets {
		pool := PoolStatus{PoolKey: key, Active: t.active, Waiting: t.waiting,
			EffectiveLimit: t.limit, CooldownUntil: t.cooldownTill,
			Total429: t.total429, Total503: t.total503}
		if len(t.models) > 0 {
			pool.Models = make(map[string]RateStatus, len(t.models))
			for model, rate := range t.models {
				if rate.requestLimit == 0 && rate.tokenLimit == 0 {
					continue
				}
				pool.Models[model] = RateStatus{RequestLimit: rate.requestLimit,
					RemainingRequests: rate.remainingRequests, RequestReset: rate.requestReset,
					TokenLimit: rate.tokenLimit, RemainingTokens: rate.remainingTokens,
					TokenReset: rate.tokenReset}
			}
		}
		result.Pools = append(result.Pools, pool)
	}
	return result
}

func parseRetryAfterMS(value string) time.Duration {
	ms, err := strconv.Atoi(value)
	if err != nil || ms <= 0 {
		return 0
	}
	return min(time.Duration(ms)*time.Millisecond, time.Hour)
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return min(time.Duration(seconds)*time.Second, time.Hour)
	}
	if t, err := http.ParseTime(value); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return min(delay, time.Hour)
		}
	}
	return 0
}
