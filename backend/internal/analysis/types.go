// Package analysis builds a canonical passport model through small durable edits.
package analysis

import (
	"context"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

const (
	CheckpointVersion = 1
	MaxPartBytes      = 64 * 1024
	TaskPending       = "pending"
	TaskActive        = "active"
	TaskDone          = "done"
)

// Finding is a small research record; source excerpts are fetched separately.
type Finding struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Notes string `json:"notes"`
}

// Task is one material area or reconciliation question, not a fixed model cap.
type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Question   string   `json:"question"`
	FindingIDs []string `json:"finding_ids"`
	Status     string   `json:"status" enum:"pending,active,done" jsonschema:"One of pending, active, done."`
}

// Input contains pinned sources, bounded research records and optional saved state.
type Input struct {
	Collection   source.Collection
	Findings     []Finding
	Tasks        []Task
	Instructions string
	Checkpoint   *Checkpoint
}

// Checkpoint contains model state and read provenance, never source bodies.
type Checkpoint struct {
	Version           int                           `json:"version"`
	SourceFingerprint string                        `json:"source_fingerprint"`
	Revision          int                           `json:"revision"`
	Model             passport.Model                `json:"model"`
	Tasks             []Task                        `json:"tasks"`
	ReadRanges        map[string][]source.LineRange `json:"read_ranges"`
	Gaps              []string                      `json:"gaps"`
	Finalized         bool                          `json:"finalized"`
	LastChange        *Change                       `json:"last_change,omitempty"`
}

// Metadata is written during reconciliation without reprinting the full model.
type Metadata struct {
	Title   string `json:"title"`
	Tagline string `json:"tagline"`
	Summary string `json:"summary"`
}

// ModelPart upserts by stable ID. A changed existing entity requires Correction.
// Nil OpenQuestions/Gaps retains the old list; a non-nil list replaces it and
// requires Correction when changing an existing nonempty list. There is no
// implicit deletion of model entities; Removals requires Correction.
type ModelPart struct {
	AreaID        string              `json:"area_id"`
	Correction    string              `json:"correction,omitempty"`
	Metadata      *Metadata           `json:"metadata,omitempty"`
	Objects       []passport.Object   `json:"objects,omitempty"`
	Relations     []passport.Relation `json:"relations,omitempty"`
	Scenarios     []passport.Scenario `json:"scenarios,omitempty"`
	Claims        []passport.Claim    `json:"claims,omitempty"`
	Evidence      []passport.Evidence `json:"evidence,omitempty"`
	OpenQuestions []string            `json:"open_questions,omitempty"`
	Gaps          []string            `json:"gaps,omitempty"`
	Removals      []EntityRemoval     `json:"removals,omitempty"`
}

// EntityRemoval explicitly removes a mistaken entity, leaving references visible.
type EntityRemoval struct {
	Kind string `json:"kind" enum:"object,relation,scenario,claim,evidence" jsonschema:"One of object, relation, scenario, claim, evidence."`
	ID   string `json:"id"`
}

// Change explains the mutation in this immutable checkpoint revision.
type Change struct {
	Kind      string   `json:"kind"`
	AreaID    string   `json:"area_id"`
	Reason    string   `json:"reason"`
	EntityIDs []string `json:"entity_ids"`
}

// Issue is a current unresolved reference or finalization requirement.
type Issue struct {
	Code        string `json:"code"`
	EntityID    string `json:"entity_id"`
	ReferenceID string `json:"reference_id"`
	Message     string `json:"message"`
}

// Progress reports real actions and accepted state, without a fake percentage.
type Progress struct {
	Message, AreaID                        string
	Step                                   int
	InputTokens, OutputTokens, DurationMS  int64
	ObjectCount, ClaimCount, RelationCount int
}

// Result contains the server-assembled model and merged source read coverage.
type Result struct {
	Model      passport.Model
	Collection source.Collection
	Checkpoint Checkpoint
}

// Persist must finish durable storage before returning nil. ModelStore publishes
// no mutation until this callback succeeds; Revision increases once per mutation.
type Persist func(context.Context, Checkpoint) error

// ProgressFunc receives only actual provider usage and accepted model counts.
type ProgressFunc func(Progress)

// ModelStore API (implemented in model_store.go):
//
//	NewModelStore(Input, Persist) (*ModelStore, error)
//	Checkpoint() Checkpoint
//	Collection() source.Collection
//	Apply(context.Context, ModelPart) error
//	RecordRead(context.Context, string, source.LineRange) error
//	AddTasks(context.Context, []Task) error
//	SetTaskStatus(context.Context, string, string) error
//	Issues() []Issue
//	Finalize(context.Context) (passport.Model, error)
//
// Task statuses are pending, active, done. ModelPart is limited to MaxPartBytes;
// the accumulated model is not capped. Missing cross-part references are exposed
// by Issues and block Finalize. Invalid local fields and unread citations are
// rejected by Apply. Changed entities need an explicit Correction explanation.
