package learning

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/access"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

func newStores(t *testing.T) (Store, access.Store) {
	t.Helper()
	database, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "learning.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if checkErr := db.RunMigrations(database); checkErr != nil {
		t.Fatal(checkErr)
	}
	return Store{DB: database}, access.Store{DB: database}
}

//nolint:gocyclo // One account lifecycle checks replacement, isolation and deletion of saved progress.
func TestProgressIsSavedPerUserAndRemovedWithTheUser(t *testing.T) {
	store, accounts := newStores(t)
	ctx := t.Context()
	ana, err := accounts.Register(ctx, access.Credentials{Email: "ana@t1a.com", Password: "long-enough-password"})
	if err != nil {
		t.Fatal(err)
	}
	ben, err := accounts.Register(ctx, access.Credentials{Email: "ben@t1a.com", Password: "long-enough-password"})
	if err != nil {
		t.Fatal(err)
	}

	empty, err := store.Get(ctx, ana.ID)
	if err != nil || empty.Modules == nil || empty.Flags == nil || empty.Unlocked == nil || len(empty.Modules) != 0 {
		t.Fatalf("empty progress: %+v %v", empty, err)
	}
	saved := State{
		Modules:  map[string]Module{"git": {Lessons: []string{"commits-and-branches"}, QuizPassed: true, BestScore: 1}},
		Flags:    []string{"curious"},
		Unlocked: map[string]string{"first-step": "2026-10-15T20:00:00Z"},
	}
	if _, checkErr := store.Put(ctx, ana.ID, saved); checkErr != nil {
		t.Fatal(checkErr)
	}
	// A second save replaces the first.
	saved.Modules["http"] = Module{Lessons: []string{"requests"}}
	if _, checkErr := store.Put(ctx, ana.ID, saved); checkErr != nil {
		t.Fatal(checkErr)
	}
	loaded, err := store.Get(ctx, ana.ID)
	if err != nil || len(loaded.Modules) != 2 || !loaded.Modules["git"].QuizPassed || loaded.Flags[0] != "curious" {
		t.Fatalf("loaded: %+v %v", loaded, err)
	}
	other, err := store.Get(ctx, ben.ID)
	if err != nil || len(other.Modules) != 0 {
		t.Fatalf("another user sees progress: %+v %v", other, err)
	}

	// Ana is the first account and therefore the admin; delete Ben, a regular user.
	if _, checkErr := store.Put(ctx, ben.ID, State{Flags: []string{"off-route"}}); checkErr != nil {
		t.Fatal(checkErr)
	}
	if checkErr := accounts.DeleteUser(ctx, ben.ID); checkErr != nil {
		t.Fatal(checkErr)
	}
	var rows int
	if checkErr := store.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM learning_progress WHERE user_id=?", ben.ID).Scan(&rows); checkErr != nil || rows != 0 {
		t.Fatalf("progress left after deleting the user: %d %v", rows, checkErr)
	}
	if kept, checkErr := store.Get(ctx, ana.ID); checkErr != nil || len(kept.Modules) != 2 {
		t.Fatalf("another user's progress was affected: %+v %v", kept, checkErr)
	}
}

func TestPutRejectsInvalidProgress(t *testing.T) {
	store, accounts := newStores(t)
	user, err := accounts.Register(t.Context(), access.Credentials{Email: "ana@t1a.com", Password: "long-enough-password"})
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("x", maxIDLength+1)
	for name, state := range map[string]State{
		"score above one": {Modules: map[string]Module{"git": {BestScore: 1.5}}},
		"negative score":  {Modules: map[string]Module{"git": {BestScore: -0.1}}},
		"empty module id": {Modules: map[string]Module{"": {}}},
		"long lesson id":  {Modules: map[string]Module{"git": {Lessons: []string{long}}}},
		"empty flag":      {Flags: []string{""}},
		"empty date":      {Unlocked: map[string]string{"first-step": ""}},
	} {
		if _, checkErr := store.Put(t.Context(), user.ID, state); !errors.Is(checkErr, ErrInvalid) {
			t.Errorf("%s: got %v, want ErrInvalid", name, checkErr)
		}
	}
}
