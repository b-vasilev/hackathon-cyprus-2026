package analysis

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestRecordReadSkipsCoveredRangesAndKeepsFailedWritesPrivate(t *testing.T) {
	initial := []source.LineRange{{StartLine: 2, EndLine: 5}}
	writes := 0
	store := &ModelStore{
		lines: map[string]int{"doc": 10},
		state: Checkpoint{Revision: 1, ReadRanges: map[string][]source.LineRange{"doc": slices.Clone(initial)}},
		persist: func(context.Context, Checkpoint) error {
			writes++
			return errors.New("storage unavailable")
		},
	}
	if err := store.RecordRead(t.Context(), "doc", source.LineRange{StartLine: 3, EndLine: 4}); err != nil || writes != 0 {
		t.Fatalf("covered read was persisted: writes=%d err=%v", writes, err)
	}
	if err := store.RecordRead(t.Context(), "doc", source.LineRange{StartLine: 5, EndLine: 7}); err == nil {
		t.Fatal("failed persistence was accepted")
	}
	if writes != 1 || store.state.Revision != 1 || !slices.Equal(store.state.ReadRanges["doc"], initial) {
		t.Fatal("failed persistence changed the accepted read ranges")
	}
	store.persist = func(_ context.Context, cp Checkpoint) error {
		if cp.Revision != 2 || !slices.Equal(cp.ReadRanges["doc"], []source.LineRange{{StartLine: 2, EndLine: 7}}) {
			t.Fatal("wrong merged checkpoint")
		}
		return nil
	}
	if err := store.RecordRead(t.Context(), "doc", source.LineRange{StartLine: 5, EndLine: 7}); err != nil {
		t.Fatal(err)
	}
}
