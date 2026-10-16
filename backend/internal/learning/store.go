// Package learning stores each person's Foundations progress, quiz results and achievements.
package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrInvalid identifies progress that is too large or malformed.
var ErrInvalid = errors.New("invalid learning progress")

// Limits keep one person's progress small; real content is far below them.
const (
	maxModules   = 100
	maxLessons   = 200
	maxIDLength  = 100
	maxFlags     = 50
	maxUnlocked  = 100
	maxDateBytes = 40
)

// Module is the progress in one Foundations module.
type Module struct {
	Lessons    []string `json:"lessons"`
	QuizPassed bool     `json:"quiz_passed"`
	Visited    bool     `json:"visited"`
	BestScore  float64  `json:"best_score" jsonschema:"Best quiz result as a share of correct answers, from 0 to 1."`
	FailedQuiz bool     `json:"failed_quiz"`
	Comeback   bool     `json:"comeback" jsonschema:"The quiz was passed after a failed attempt."`
}

// State is everything the learning area remembers for one person.
type State struct {
	Modules  map[string]Module `json:"modules"`
	Flags    []string          `json:"flags" jsonschema:"Easter eggs found, by identifier."`
	Unlocked map[string]string `json:"unlocked" jsonschema:"When each achievement was first unlocked, as an ISO 8601 date."`
}

// Store reads and writes progress per user.
type Store struct{ DB *sql.DB }

func validID(value string) bool { return value != "" && len(value) <= maxIDLength }

// Validate rejects progress that does not fit the expected shape or size.
func (s State) Validate() error {
	if len(s.Modules) > maxModules || len(s.Flags) > maxFlags || len(s.Unlocked) > maxUnlocked {
		return fmt.Errorf("%w: too many entries", ErrInvalid)
	}
	for id, module := range s.Modules {
		if !validID(id) || len(module.Lessons) > maxLessons || module.BestScore < 0 || module.BestScore > 1 {
			return fmt.Errorf("%w: module %q", ErrInvalid, id)
		}
		for _, lesson := range module.Lessons {
			if !validID(lesson) {
				return fmt.Errorf("%w: lesson in module %q", ErrInvalid, id)
			}
		}
	}
	for _, flag := range s.Flags {
		if !validID(flag) {
			return fmt.Errorf("%w: flag", ErrInvalid)
		}
	}
	for id, date := range s.Unlocked {
		if !validID(id) || date == "" || len(date) > maxDateBytes {
			return fmt.Errorf("%w: achievement %q", ErrInvalid, id)
		}
	}
	return nil
}

// normalized returns empty collections instead of nil so clients always get the same shape.
func (s State) normalized() State {
	if s.Modules == nil {
		s.Modules = map[string]Module{}
	}
	for id, module := range s.Modules {
		if module.Lessons == nil {
			module.Lessons = []string{}
			s.Modules[id] = module
		}
	}
	if s.Flags == nil {
		s.Flags = []string{}
	}
	if s.Unlocked == nil {
		s.Unlocked = map[string]string{}
	}
	return s
}

// Get returns the saved progress, or empty progress for someone who has not started.
func (s Store) Get(ctx context.Context, userID string) (State, error) {
	var raw string
	err := s.DB.QueryRowContext(ctx, "SELECT state FROM learning_progress WHERE user_id=?", userID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}.normalized(), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("load learning progress: %w", err)
	}
	var state State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return State{}, fmt.Errorf("decode learning progress: %w", err)
	}
	return state.normalized(), nil
}

// Put merges progress into the saved copy and returns the result. Progress only grows,
// so a device with an older or empty copy can never erase what another device saved.
func (s Store) Put(ctx context.Context, userID string, state State) (State, error) {
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return State{}, fmt.Errorf("begin learning progress update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var raw string
	saved := State{}
	err = tx.QueryRowContext(ctx, "SELECT state FROM learning_progress WHERE user_id=?", userID).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return State{}, fmt.Errorf("load learning progress: %w", err)
	default:
		if decodeErr := json.Unmarshal([]byte(raw), &saved); decodeErr != nil {
			return State{}, fmt.Errorf("decode learning progress: %w", decodeErr)
		}
	}
	merged := merge(saved, state).normalized()
	if validateErr := merged.Validate(); validateErr != nil {
		return State{}, validateErr
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return State{}, fmt.Errorf("encode learning progress: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO learning_progress(user_id,state,updated_at) VALUES(?,?,?)
 ON CONFLICT(user_id) DO UPDATE SET state=excluded.state, updated_at=excluded.updated_at`,
		userID, string(data), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return State{}, fmt.Errorf("save learning progress: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return State{}, fmt.Errorf("commit learning progress: %w", err)
	}
	return merged, nil
}

// merge mirrors frontend/src/foundations/merge.ts: lessons and flags are unioned, flags
// set on either side stay set, the best score wins and the earliest unlock date is kept.
func merge(a, b State) State {
	result := State{Modules: map[string]Module{}, Unlocked: map[string]string{}}
	for _, modules := range []map[string]Module{a.Modules, b.Modules} {
		for id, m := range modules {
			current, seen := result.Modules[id]
			if !seen {
				m.Lessons = union(nil, m.Lessons)
				result.Modules[id] = m
				continue
			}
			result.Modules[id] = Module{
				Lessons:    union(current.Lessons, m.Lessons),
				QuizPassed: current.QuizPassed || m.QuizPassed,
				Visited:    current.Visited || m.Visited,
				BestScore:  max(current.BestScore, m.BestScore),
				FailedQuiz: current.FailedQuiz || m.FailedQuiz,
				Comeback:   current.Comeback || m.Comeback,
			}
		}
	}
	result.Flags = union(a.Flags, b.Flags)
	for _, unlocked := range []map[string]string{a.Unlocked, b.Unlocked} {
		for id, date := range unlocked {
			if current, seen := result.Unlocked[id]; !seen || date < current {
				result.Unlocked[id] = date
			}
		}
	}
	return result
}

func union(a, b []string) []string {
	result := make([]string, 0, len(a)+len(b))
	seen := map[string]bool{}
	for _, item := range append(append([]string{}, a...), b...) {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
