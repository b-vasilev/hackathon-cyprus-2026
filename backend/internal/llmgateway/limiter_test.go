package llmgateway

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

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
