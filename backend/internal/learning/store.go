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

// Put replaces the saved progress after validating it.
func (s Store) Put(ctx context.Context, userID string, state State) (State, error) {
	if err := state.Validate(); err != nil {
		return State{}, err
	}
	state = state.normalized()
	raw, err := json.Marshal(state)
	if err != nil {
		return State{}, fmt.Errorf("encode learning progress: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO learning_progress(user_id,state,updated_at) VALUES(?,?,?)
 ON CONFLICT(user_id) DO UPDATE SET state=excluded.state, updated_at=excluded.updated_at`,
		userID, string(raw), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return State{}, fmt.Errorf("save learning progress: %w", err)
	}
	return state, nil
}
