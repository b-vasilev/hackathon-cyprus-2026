package llmgateway

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestAdaptiveLimitAfterOverload(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			limiter := NewLimiter(30, 60, 30, 60)
			policy := DefaultPoolPolicy("pool")
			policy.MaxActive = 30
			pool := &targetLimit{limit: 30, policy: policy}
			limiter.targets["pool"] = pool
			now := time.Unix(100, 0)
			meta := RequestMeta{PoolKey: "pool", API: true, StartedAt: now.Add(-time.Second)}
			headers := http.Header{"Retry-After": []string{"10"}}
			limiter.observeResponseLocked(meta, status, headers, now)
			if pool.limit != 15 {
				t.Fatalf("first overload must halve 30 slots, got %d", pool.limit)
			}
			if allowed, wake := limiter.canStart(pool, meta, now); allowed || !wake.Equal(now.Add(10*time.Second)) {
				t.Fatalf("pool did not honor shared cooldown: allowed=%v wake=%v", allowed, wake)
			}
			// Successes and more errors from the original batch must neither
			// undo the reduction nor reduce once for every in-flight request.
			for range 8 {
				limiter.observeResponseLocked(meta, http.StatusOK, nil, now.Add(time.Second))
			}
			for range 21 {
				limiter.observeResponseLocked(meta, status, headers, now.Add(time.Second))
			}
			if pool.limit != 15 || pool.successes != 0 {
				t.Fatalf("in-flight batch changed recovery: limit=%d successes=%d", pool.limit, pool.successes)
			}
			if pool.total429+pool.total503 != 22 {
				t.Fatalf("overload counters lost responses: 429=%d 503=%d", pool.total429, pool.total503)
			}
			after := pool.cooldownTill.Add(time.Second)
			for range 8 {
				limiter.observeResponseLocked(meta, http.StatusOK, nil, after)
			}
			if pool.limit != 15 || pool.successes != 0 {
				t.Fatalf("late old successes raised capacity: limit=%d successes=%d", pool.limit, pool.successes)
			}
			meta.StartedAt = after
			if allowed, _ := limiter.canStart(pool, meta, after); !allowed {
				t.Fatal("pool did not resume after cooldown")
			}
			for range 8 {
				limiter.observeResponseLocked(meta, http.StatusOK, nil, after)
			}
			if pool.limit != 16 {
				t.Fatalf("fresh successes did not recover gradually: %d", pool.limit)
			}
			limiter.observeResponseLocked(meta, status, headers, after)
			if pool.limit != 8 {
				t.Fatalf("new overload did not lower capacity again: %d", pool.limit)
			}
		})
	}
}

func TestAdmissionRecordsDispatchTime(t *testing.T) {
	limiter := NewLimiter(30, 60, 30, 60)
	policy := DefaultPoolPolicy("pool")
	pool := &targetLimit{limit: policy.MaxActive, policy: policy, cooldownTill: time.Now().Add(-time.Second)}
	limiter.targets["pool"] = pool
	meta := RequestMeta{PoolKey: "pool", API: true, StartedAt: time.Now().Add(-time.Hour)}
	finish, err := limiter.AcquireRequest(t.Context(), meta, policy)
	if err != nil {
		t.Fatal(err)
	}
	finish(http.StatusOK, nil)
	if pool.successes != 1 {
		t.Fatal("request admitted after cooldown was counted as an older request")
	}
}

// Signal each blocking select so the test can coordinate without sleeps.
type observedWaitContext struct {
	context.Context
	waiting chan struct{}
}

func (c observedWaitContext) Done() <-chan struct{} {
	select {
	case c.waiting <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

type admissionResult struct {
	finish func(int, string)
	err    error
}

func startBackgroundAdmission(ctx context.Context, limiter *Limiter) (<-chan struct{}, <-chan admissionResult) {
	waiting := make(chan struct{}, 1)
	result := make(chan admissionResult, 1)
	go func() {
		finish, err := limiter.Acquire(WithQueueWait(observedWaitContext{Context: ctx, waiting: waiting}), "pool", true)
		result <- admissionResult{finish: finish, err: err}
	}()
	return waiting, result
}

func fullLimiter(t *testing.T) (*Limiter, func()) {
	t.Helper()
	limiter := NewLimiter(1, 0, 1, 0)
	finish, err := limiter.Acquire(t.Context(), "pool", true)
	if err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release := func() { once.Do(func() { finish(200, "") }) }
	t.Cleanup(release)
	return limiter, release
}

func assertAdmissionCounts(t *testing.T, limiter *Limiter, active int) {
	t.Helper()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if limiter.globalActive != active || limiter.globalWaiting != 0 || len(limiter.waiters) != 0 {
		t.Fatalf("admission leaked capacity: active=%d waiting=%d entries=%d", limiter.globalActive, limiter.globalWaiting, len(limiter.waiters))
	}
	for _, target := range limiter.targets {
		if target.active != active || target.waiting != 0 {
			t.Fatalf("pool leaked capacity: active=%d waiting=%d", target.active, target.waiting)
		}
	}
}

func TestBackgroundQueueBackpressure(t *testing.T) {
	t.Run("foreground rejects while background waits without consuming queue", func(t *testing.T) {
		limiter, release := fullLimiter(t)
		if _, err := limiter.Acquire(t.Context(), "pool", true); !errors.Is(err, ErrQueueFull) {
			t.Fatalf("foreground queue overflow: %v", err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		waiting, result := startBackgroundAdmission(ctx, limiter)
		select {
		case <-waiting:
		case got := <-result:
			t.Fatalf("background admission stopped before waiting: %v", got.err)
		case <-ctx.Done():
			t.Fatal("background admission did not wait")
		}
		assertAdmissionCounts(t, limiter, 1)
		release()
		select {
		case got := <-result:
			if got.err != nil {
				t.Fatal(got.err)
			}
			assertAdmissionCounts(t, limiter, 1)
			got.finish(200, "")
		case <-ctx.Done():
			t.Fatal("background admission did not resume after capacity was released")
		}
		assertAdmissionCounts(t, limiter, 0)
	})

	t.Run("cancellation leaves no queue entries", func(t *testing.T) {
		limiter, release := fullLimiter(t)
		guard, stopGuard := context.WithTimeout(t.Context(), 2*time.Second)
		defer stopGuard()
		ctx, cancel := context.WithCancel(guard)
		defer cancel()
		waiting, result := startBackgroundAdmission(ctx, limiter)
		select {
		case <-waiting:
		case <-guard.Done():
			t.Fatal("background admission did not wait")
		}
		cancel()
		select {
		case got := <-result:
			if !errors.Is(got.err, context.Canceled) {
				t.Fatalf("background cancellation: %v", got.err)
			}
		case <-guard.Done():
			t.Fatal("background admission ignored cancellation")
		}
		assertAdmissionCounts(t, limiter, 1)
		release()
		assertAdmissionCounts(t, limiter, 0)
	})

	t.Run("change notifications preserve the original timeout", func(t *testing.T) {
		limiter, _ := fullLimiter(t)
		limiter.globalWait = 20 * time.Millisecond
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		waiting, result := startBackgroundAdmission(ctx, limiter)
		for {
			select {
			case <-waiting:
				limiter.mu.Lock()
				limiter.signal()
				limiter.mu.Unlock()
			case got := <-result:
				if !errors.Is(got.err, ErrQueueTimeout) {
					t.Fatalf("background queue timeout: %v", got.err)
				}
				assertAdmissionCounts(t, limiter, 1)
				return
			case <-ctx.Done():
				t.Fatal("change notifications extended the queue timeout")
			}
		}
	})
}
