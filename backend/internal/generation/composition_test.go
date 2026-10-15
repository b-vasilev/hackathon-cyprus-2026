package generation

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llm"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
)

type compositionInvocation struct {
	key     string
	release chan error
}

type controlledComposer struct {
	fixture fixtureGenerator
	started chan compositionInvocation
	mu      sync.Mutex
	calls   map[string]int
	active  int
	peak    int
}

func (g *controlledComposer) Generate(ctx context.Context, config llmgateway.Config, request llm.Request) (llm.Result, error) {
	if !strings.HasPrefix(request.Input, fragmentPrompt) {
		return g.fixture.Generate(ctx, config, request)
	}
	chapter, _, _ := strings.Cut(strings.Split(request.Input, "CHAPTER ID: ")[1], "\n")
	part, _, _ := strings.Cut(strings.Split(request.Input, "ASSIGNED PART:\n")[1], "\nMODEL:\n")
	key := chapter + ":" + part
	g.mu.Lock()
	g.calls[key]++
	g.active++
	g.peak = max(g.peak, g.active)
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.active--
		g.mu.Unlock()
	}()
	invocation := compositionInvocation{key: key, release: make(chan error, 1)}
	select {
	case g.started <- invocation:
	case <-ctx.Done():
		return llm.Result{}, fmt.Errorf("queue fixture fragment: %w", ctx.Err())
	}
	select {
	case err := <-invocation.release:
		if err != nil {
			return llm.Result{}, err
		}
	case <-ctx.Done():
		return llm.Result{}, fmt.Errorf("wait for fixture fragment: %w", ctx.Err())
	}
	return g.fixture.Generate(ctx, config, request)
}

//nolint:gocyclo // One coordinated execution checks bounded concurrency, isolated retry and stable order.
func TestCompositionParallelPartsRetryIndependentlyAndKeepReadingOrder(t *testing.T) {
	f := newRetryFixture(t)
	generator := &controlledComposer{started: make(chan compositionInvocation, 8), calls: map[string]int{}}
	f.service.LLM = generator
	c := &composer{service: f.service, run: f.run, model: fixtureModel("Payment service", "source"), documents: map[string]int{"source": 1}, state: CompositionState{Parallelism: 2, Tasks: []CompositionTask{}}}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	type outcome struct {
		chapters []string
		blocks   []string
		err      error
	}
	finished := make(chan outcome, 1)
	go func() {
		chapters, _, err := c.chapters(ctx, []string{"architecture", "repositories"}, "")
		result := outcome{err: err}
		for _, chapter := range chapters {
			result.chapters = append(result.chapters, chapter.ID)
			for _, block := range chapter.Blocks {
				result.blocks = append(result.blocks, block.Kind+":"+block.Markdown)
			}
		}
		finished <- result
	}()
	next := func() compositionInvocation {
		t.Helper()
		select {
		case invocation := <-generator.started:
			return invocation
		case result := <-finished:
			t.Fatalf("composition stopped before expected request: %v", result.err)
		case <-ctx.Done():
			t.Fatal("composition did not start concurrent parts")
		}
		return compositionInvocation{}
	}
	first, second := next(), next()
	// Keep the first planned point blocked while a later point completes.
	if strings.Contains(second.key, "Fixture block 0") {
		first, second = second, first
	}
	if !strings.Contains(first.key, "Fixture block 0") || !strings.HasPrefix(first.key, "architecture:") {
		t.Fatalf("unexpected first point: %s", first.key)
	}
	second.release <- nil
	third := next()
	// A third point starts only after a worker finishes the second point.
	first.release <- &llm.ProviderError{Code: "timeout", Message: "fixture timeout"}
	third.release <- nil
	for range 2 {
		invocation := next()
		invocation.release <- nil
	}
	var result outcome
	select {
	case result = <-finished:
	case <-ctx.Done():
		t.Fatal("composition did not finish")
	}
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !reflect.DeepEqual(result.chapters, []string{"architecture", "repositories"}) {
		t.Fatalf("chapter order changed: %v", result.chapters)
	}
	wantBlocks := []string{}
	for _, id := range []string{"architecture", "repositories"} {
		for _, chapter := range fixtureViews().Document.Chapters {
			if chapter.ID == id {
				for _, block := range chapter.Blocks {
					wantBlocks = append(wantBlocks, block.Kind+":"+block.Markdown)
				}
			}
		}
	}
	if !reflect.DeepEqual(result.blocks, wantBlocks) {
		t.Fatalf("point order changed: got %v, want %v", result.blocks, wantBlocks)
	}
	generator.mu.Lock()
	defer generator.mu.Unlock()
	if generator.peak != 2 {
		t.Fatalf("expected exactly two concurrent requests, got %d", generator.peak)
	}
	if len(generator.calls) != 4 {
		t.Fatalf("expected four distinct fragments, got %d", len(generator.calls))
	}
	for key, calls := range generator.calls {
		want := 1
		if key == first.key {
			want = 2
		}
		if calls != want {
			t.Errorf("fragment %s called %d times, want %d", key, calls, want)
		}
	}
}
