package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// ModelStore publishes only accepted, durable edits. The persist callback must
// not re-enter this store; mutations are serialized through durable publication.
type ModelStore struct {
	mu            sync.RWMutex
	state         Checkpoint
	collection    source.Collection
	lines         map[string]int
	findings      map[string]bool
	persist       Persist
	areaRevisions map[string]int
}

// NewModelStore validates the pinned sources and restores the last accepted state.
func NewModelStore(input Input, persist Persist) (*ModelStore, error) {
	if persist == nil {
		return nil, errors.New("analysis requires durable checkpoint persistence")
	}
	fingerprint, err := SourceFingerprint(input.Collection)
	if err != nil {
		return nil, err
	}
	store := &ModelStore{
		collection: cloneCollection(input.Collection), lines: make(map[string]int),
		findings: make(map[string]bool), persist: persist,
		state: Checkpoint{Version: CheckpointVersion, SourceFingerprint: fingerprint, ReadRanges: make(map[string][]source.LineRange)},
	}
	documents := make(map[string]source.Document, len(input.Collection.Documents))
	for _, document := range input.Collection.Documents {
		store.lines[document.ID] = document.LineCount
		documents[document.ID] = document
	}
	for _, finding := range input.Findings {
		if strings.TrimSpace(finding.ID) == "" || store.findings[finding.ID] {
			return nil, errors.New("duplicate or empty analysis finding ID")
		}
		store.findings[finding.ID] = true
	}
	if err := store.restoreCheckpoint(input, fingerprint); err != nil {
		return nil, err
	}
	if err := store.restoreSourceReads(input.Collection, documents); err != nil {
		return nil, err
	}
	if err := passport.ValidateModelDraft(store.state.Model, store.lines); err != nil {
		return nil, fmt.Errorf("invalid saved analysis model: %w", err)
	}
	if err := store.validateCitations(store.state); err != nil {
		return nil, err
	}
	if store.state.Finalized {
		if issues := store.issuesLocked(); len(issues) > 0 {
			return nil, fmt.Errorf("invalid finalized checkpoint: %s", issues[0].Message)
		}
		if err := passport.ValidateModel(store.state.Model, store.lines); err != nil {
			return nil, fmt.Errorf("invalid finalized model: %w", err)
		}
	}
	return store, nil
}

// SourceFingerprint verifies source bytes and hashes a stable identity index.
// Read coverage is excluded: new verified reads do not change source identity.
func SourceFingerprint(collection source.Collection) (string, error) {
	type snapshotIdentity struct{ ID, Locator, Revision string }
	type documentIdentity struct {
		ID, SnapshotID, Path, Hash string
		Lines                      int
	}
	identities := struct {
		Snapshots []snapshotIdentity
		Documents []documentIdentity
	}{}
	knownSnapshots, knownDocuments := make(map[string]bool), make(map[string]bool)
	paths := make(map[struct{ snapshot, path string }]bool)
	for _, snapshot := range collection.Snapshots {
		if snapshot.ID == "" || snapshot.Locator == "" || snapshot.Revision == "" || knownSnapshots[snapshot.ID] {
			return "", errors.New("invalid or duplicate source snapshot identity")
		}
		knownSnapshots[snapshot.ID] = true
		identities.Snapshots = append(identities.Snapshots, snapshotIdentity{snapshot.ID, snapshot.Locator, snapshot.Revision})
	}
	for _, document := range collection.Documents {
		hash := sha256.Sum256([]byte(document.Content))
		path := struct{ snapshot, path string }{document.SnapshotID, document.Path}
		if document.ID == "" || document.Path == "" || knownDocuments[document.ID] || paths[path] || !knownSnapshots[document.SnapshotID] || document.LineCount != strings.Count(document.Content, "\n")+1 || document.ContentHash != hex.EncodeToString(hash[:]) {
			return "", fmt.Errorf("source document identity or content differs: %s", document.ID)
		}
		knownDocuments[document.ID] = true
		paths[path] = true
		identities.Documents = append(identities.Documents, documentIdentity{document.ID, document.SnapshotID, document.Path, document.ContentHash, document.LineCount})
	}
	if len(identities.Documents) == 0 {
		return "", errors.New("analysis requires pinned source documents")
	}
	slices.SortFunc(identities.Snapshots, func(a, b snapshotIdentity) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(identities.Documents, func(a, b documentIdentity) int { return strings.Compare(a.ID, b.ID) })
	data, err := json.Marshal(identities)
	if err != nil {
		return "", fmt.Errorf("encode source identities: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Checkpoint returns an independent copy, suitable for bounded caller-side views.
func (store *ModelStore) Checkpoint() Checkpoint {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return cloneCheckpoint(store.state)
}

// Apply atomically validates and persists a bounded canonical model part.
func (store *ModelStore) Apply(ctx context.Context, part ModelPart) error {
	if err := boundedPart(part); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	area := slices.IndexFunc(store.state.Tasks, func(task Task) bool { return task.ID == part.AreaID })
	if area < 0 || store.state.Tasks[area].Status != TaskActive {
		return errors.New("model part must belong to the active analysis area")
	}
	part.Evidence = repairDocumentIDs(part.Evidence, store.lines)
	next := cloneCheckpoint(store.state)
	ids, err := applyPart(&next, part)
	if err != nil {
		return err
	}
	if err := passport.ValidateModelDraft(next.Model, store.lines); err != nil {
		return fmt.Errorf("reject analysis part: %w", err)
	}
	if err := store.validateCitations(next); err != nil {
		return err
	}
	if reflect.DeepEqual(next, store.state) {
		return nil
	}
	next.LastChange = &Change{Kind: "model_part", AreaID: part.AreaID, Reason: part.Correction, EntityIDs: ids}
	return store.publish(ctx, next)
}

func applyPart(next *Checkpoint, part ModelPart) ([]string, error) {
	ids := make([]string, 0)
	if err := removeEntities(&next.Model, part); err != nil {
		return nil, err
	}
	var err error
	next.Model.Objects, err = upsertEntities(next.Model.Objects, part.Objects, func(value passport.Object) string { return value.ID }, part.Correction)
	if err != nil {
		return nil, err
	}
	next.Model.Relations, err = upsertEntities(next.Model.Relations, part.Relations, func(value passport.Relation) string { return value.ID }, part.Correction)
	if err != nil {
		return nil, err
	}
	next.Model.Scenarios, err = upsertEntities(next.Model.Scenarios, part.Scenarios, func(value passport.Scenario) string { return value.ID }, part.Correction)
	if err != nil {
		return nil, err
	}
	next.Model.Claims, err = upsertEntities(next.Model.Claims, part.Claims, func(value passport.Claim) string { return value.ID }, part.Correction)
	if err != nil {
		return nil, err
	}
	next.Model.Evidence, err = upsertEntities(next.Model.Evidence, part.Evidence, func(value passport.Evidence) string { return value.ID }, part.Correction)
	if err != nil {
		return nil, err
	}
	if part.Metadata != nil {
		old := Metadata{Title: next.Model.Title, Tagline: next.Model.Tagline, Summary: next.Model.Summary}
		if strings.TrimSpace(part.Metadata.Title) == "" || strings.TrimSpace(part.Metadata.Summary) == "" {
			return nil, errors.New("model metadata requires a title and summary")
		}
		if old != *part.Metadata && old != (Metadata{}) && strings.TrimSpace(part.Correction) == "" {
			return nil, errors.New("changed model metadata requires a correction explanation")
		}
		next.Model.Title, next.Model.Tagline, next.Model.Summary = part.Metadata.Title, part.Metadata.Tagline, part.Metadata.Summary
		ids = append(ids, "model_metadata")
	}
	if err := updateModelQuestions(next, part); err != nil {
		return nil, err
	}
	for _, object := range part.Objects {
		ids = append(ids, object.ID)
	}
	for _, relation := range part.Relations {
		ids = append(ids, relation.ID)
	}
	for _, scenario := range part.Scenarios {
		ids = append(ids, scenario.ID)
	}
	for _, claim := range part.Claims {
		ids = append(ids, claim.ID)
	}
	for _, evidence := range part.Evidence {
		ids = append(ids, evidence.ID)
	}
	for _, removal := range part.Removals {
		ids = append(ids, removal.ID)
	}
	return ids, nil
}

func upsertEntities[T any](current, incoming []T, id func(T) string, correction string) ([]T, error) {
	positions, seen := make(map[string]int, len(current)), make(map[string]bool, len(incoming))
	for index, value := range current {
		positions[id(value)] = index
	}
	for _, value := range incoming {
		key := id(value)
		if key == "" || seen[key] {
			return nil, fmt.Errorf("duplicate or empty ID in model part: %s", key)
		}
		seen[key] = true
		if position, exists := positions[key]; exists {
			if !reflect.DeepEqual(current[position], value) && strings.TrimSpace(correction) == "" {
				return nil, fmt.Errorf("changing %s requires a correction explanation", key)
			}
			current[position] = value
		} else {
			current = append(current, value)
			positions[key] = len(current) - 1
		}
	}
	return current, nil
}

func removeEntities(model *passport.Model, part ModelPart) error {
	if len(part.Removals) == 0 {
		return nil
	}
	if strings.TrimSpace(part.Correction) == "" {
		return errors.New("entity removal requires a correction explanation")
	}
	incoming := make(map[string]bool)
	for _, value := range part.Objects {
		incoming[value.ID] = true
	}
	for _, value := range part.Relations {
		incoming[value.ID] = true
	}
	for _, value := range part.Scenarios {
		incoming[value.ID] = true
	}
	for _, value := range part.Claims {
		incoming[value.ID] = true
	}
	for _, value := range part.Evidence {
		incoming[value.ID] = true
	}
	seen := make(map[string]bool)
	for _, removal := range part.Removals {
		if removal.ID == "" || seen[removal.ID] || incoming[removal.ID] {
			return errors.New("ambiguous duplicate removal or upsert/removal combination")
		}
		seen[removal.ID] = true
		var found bool
		switch removal.Kind {
		case "object":
			model.Objects, found = removeEntity(model.Objects, removal.ID, func(value passport.Object) string { return value.ID })
		case "relation":
			model.Relations, found = removeEntity(model.Relations, removal.ID, func(value passport.Relation) string { return value.ID })
		case "scenario":
			model.Scenarios, found = removeEntity(model.Scenarios, removal.ID, func(value passport.Scenario) string { return value.ID })
		case "claim":
			model.Claims, found = removeEntity(model.Claims, removal.ID, func(value passport.Claim) string { return value.ID })
		case "evidence":
			model.Evidence, found = removeEntity(model.Evidence, removal.ID, func(value passport.Evidence) string { return value.ID })
		default:
			return fmt.Errorf("unknown entity removal kind: %s", removal.Kind)
		}
		if !found {
			return fmt.Errorf("unknown %s removal: %s", removal.Kind, removal.ID)
		}
	}
	return nil
}

func removeEntity[T any](values []T, target string, id func(T) string) ([]T, bool) {
	index := slices.IndexFunc(values, func(value T) bool { return id(value) == target })
	if index < 0 {
		return values, false
	}
	return slices.Delete(values, index, index+1), true
}

// RecordRead persists an exact original-source range before a tool returns it.
func (store *ModelStore) RecordRead(ctx context.Context, documentID string, span source.LineRange) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.validateRange(documentID, span); err != nil {
		return err
	}
	ranges := mergeRanges(append(slices.Clone(store.state.ReadRanges[documentID]), span))
	if slices.Equal(ranges, store.state.ReadRanges[documentID]) {
		return nil
	}
	next := cloneCheckpoint(store.state)
	next.ReadRanges[documentID] = ranges
	next.LastChange = &Change{Kind: "source_read", EntityIDs: []string{documentID}}
	return store.publish(ctx, next)
}

// prepareSynthesis replaces an old research queue while retaining saved facts
// and read provenance. Retries of the fixed plan retain their completed steps.
func (store *ModelStore) prepareSynthesis(ctx context.Context, tasks []Task) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	next := cloneCheckpoint(store.state)
	samePlan := len(next.Tasks) == len(tasks)
	for _, old := range next.Tasks {
		if !slices.ContainsFunc(tasks, func(task Task) bool { return task.ID == old.ID }) {
			samePlan = false
		}
	}
	updated := cloneTasks(tasks)
	if samePlan {
		for i := range updated {
			for _, old := range next.Tasks {
				if updated[i].ID == old.ID {
					updated[i].Status = old.Status
				}
			}
		}
	}
	if err := store.validateTasks(updated); err != nil {
		return err
	}
	if reflect.DeepEqual(next.Tasks, updated) {
		return nil
	}
	next.Tasks = updated
	next.LastChange = &Change{Kind: "synthesis_plan", Reason: "Synthesize saved research in three steps; preserve accepted model facts and citations."}
	return store.publish(ctx, next)
}

// AddTasks adds discovered work without erasing prior areas or completion state.
func (store *ModelStore) AddTasks(ctx context.Context, tasks []Task) error {
	if err := boundedPart(tasks); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	next := cloneCheckpoint(store.state)
	for _, task := range tasks {
		if task.Status == "" {
			task.Status = TaskPending
		}
		index := slices.IndexFunc(next.Tasks, func(existing Task) bool { return existing.ID == task.ID })
		if index >= 0 {
			existing := next.Tasks[index]
			task.Status = existing.Status
			if !reflect.DeepEqual(existing, task) {
				return fmt.Errorf("cannot silently replace existing analysis task: %s", task.ID)
			}
			continue
		}
		if task.Status != TaskPending {
			return errors.New("new analysis tasks must start pending")
		}
		next.Tasks = append(next.Tasks, task)
	}
	if err := store.validateTasks(next.Tasks); err != nil {
		return err
	}
	if reflect.DeepEqual(next.Tasks, store.state.Tasks) {
		return nil
	}
	next.LastChange = &Change{Kind: "add_tasks"}
	return store.publish(ctx, next)
}

// SetTaskStatus explicitly starts, finishes or reopens one material area.
func (store *ModelStore) SetTaskStatus(ctx context.Context, id, status string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	next := cloneCheckpoint(store.state)
	index := slices.IndexFunc(next.Tasks, func(task Task) bool { return task.ID == id })
	if index < 0 {
		return fmt.Errorf("unknown analysis area: %s", id)
	}
	if next.Tasks[index].Status == status {
		return nil
	}
	if status == TaskDone && next.Tasks[index].Status != TaskActive {
		return errors.New("finish an analysis area only after starting it")
	}
	next.Tasks[index].Status = status
	if err := store.validateTasks(next.Tasks); err != nil {
		return err
	}
	next.LastChange = &Change{Kind: "task_status", AreaID: id}
	return store.publish(ctx, next)
}

func (store *ModelStore) validateTasks(tasks []Task) error {
	seen, active := make(map[string]bool), 0
	for _, task := range tasks {
		if strings.TrimSpace(task.ID) == "" || strings.TrimSpace(task.Title) == "" || strings.TrimSpace(task.Question) == "" || seen[task.ID] || !slices.Contains([]string{TaskPending, TaskActive, TaskDone}, task.Status) {
			return errors.New("invalid or duplicate analysis task")
		}
		seen[task.ID] = true
		if task.Status == TaskActive {
			active++
		}
		for _, finding := range task.FindingIDs {
			if !store.findings[finding] {
				return fmt.Errorf("analysis task references an unknown finding: %s", finding)
			}
		}
	}
	if active > 1 {
		if active != 2 || slices.ContainsFunc(tasks, func(task Task) bool {
			return task.Status == TaskActive && task.ID != ArchitectureTaskID && task.ID != PassportTaskID
		}) {
			return errors.New("only architecture and passport synthesis may run together; reconciliation requires exclusive access")
		}
	}
	return nil
}

func (store *ModelStore) validateRange(id string, span source.LineRange) error {
	lines, known := store.lines[id]
	if !known || span.StartLine < 1 || span.EndLine < span.StartLine || span.EndLine > lines {
		return fmt.Errorf("invalid analysis source range: %s", id)
	}
	return nil
}

func (store *ModelStore) validateCitations(state Checkpoint) error {
	for _, evidence := range state.Model.Evidence {
		covered := slices.ContainsFunc(state.ReadRanges[evidence.DocumentID], func(span source.LineRange) bool {
			return span.StartLine <= evidence.StartLine && span.EndLine >= evidence.EndLine
		})
		if !covered {
			return fmt.Errorf("evidence cites unread source lines: %s", evidence.ID)
		}
	}
	return nil
}

// Issues exposes every unresolved cross-part reference and completion requirement.
func (store *ModelStore) Issues() []Issue {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.issuesLocked()
}

func (store *ModelStore) issuesLocked() []Issue {
	return ValidationIssues(store.state)
}

// ValidationIssues reports the work still required to seal a saved model.
func ValidationIssues(checkpoint Checkpoint) []Issue {
	issues := make([]Issue, 0)
	if checkpoint.Model.Title == "" || checkpoint.Model.Summary == "" {
		issues = append(issues, Issue{Code: "missing_metadata", EntityID: "model", Message: "Reconcile the product title and summary."})
	}
	if len(checkpoint.Model.Objects) == 0 || len(checkpoint.Model.Claims) == 0 {
		issues = append(issues, Issue{Code: "empty_model", EntityID: "model", Message: "The canonical model needs objects and supporting claims."})
	}
	if len(checkpoint.Tasks) == 0 {
		issues = append(issues, Issue{Code: "missing_plan", Message: "Discover and complete the material analysis areas."})
	}
	for _, task := range checkpoint.Tasks {
		if task.Status != TaskDone {
			issues = append(issues, Issue{Code: "unfinished_area", EntityID: task.ID, Message: "Analysis area is not done: " + task.Title})
		}
	}
	return append(issues, modelReferenceIssues(checkpoint.Model)...)
}

func modelReferenceIssues(model passport.Model) []Issue {
	issues := make([]Issue, 0)
	evidence, claims, objects := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	for _, value := range model.Evidence {
		evidence[value.ID] = true
	}
	for _, value := range model.Claims {
		claims[value.ID] = true
	}
	for _, value := range model.Objects {
		objects[value.ID] = true
	}
	refs := func(entity, kind string, ids []string, known map[string]bool) {
		for _, id := range ids {
			if !known[id] {
				issues = append(issues, Issue{Code: "unknown_reference", EntityID: entity, ReferenceID: id, Message: fmt.Sprintf("%s references unknown %s %s", entity, kind, id)})
			}
		}
	}
	for _, value := range model.Claims {
		if value.Basis == "source_backed" && len(value.EvidenceIDs) == 0 {
			issues = append(issues, Issue{Code: "missing_evidence", EntityID: value.ID, Message: "Source-backed claim needs exact read source evidence: " + value.ID})
		}
		refs(value.ID, "evidence", value.EvidenceIDs, evidence)
	}
	for _, value := range model.Objects {
		if len(value.ClaimIDs) == 0 {
			issues = append(issues, Issue{Code: "missing_claims", EntityID: value.ID, Message: "Object needs supporting claims: " + value.ID})
		}
		refs(value.ID, "claim", value.ClaimIDs, claims)
		if value.ParentID != "" {
			refs(value.ID, "parent object", []string{value.ParentID}, objects)
		}
	}
	for _, value := range model.Relations {
		if len(value.ClaimIDs) == 0 {
			issues = append(issues, Issue{Code: "missing_claims", EntityID: value.ID, Message: "Relation needs supporting claims: " + value.ID})
		}
		refs(value.ID, "object", []string{value.From, value.To}, objects)
		refs(value.ID, "claim", value.ClaimIDs, claims)
	}
	for _, scenario := range model.Scenarios {
		for _, step := range scenario.Steps {
			refs(step.ID, "object", step.ObjectIDs, objects)
			refs(step.ID, "claim", step.ClaimIDs, claims)
		}
	}
	return issues
}

// Finalize assembles and seals a globally valid model without an LLM reprint.
func (store *ModelStore) Finalize(ctx context.Context) (passport.Model, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if issues := store.issuesLocked(); len(issues) > 0 {
		return passport.Model{}, fmt.Errorf("analysis is not ready: %s", issues[0].Message)
	}
	if err := passport.ValidateModel(store.state.Model, store.lines); err != nil {
		return passport.Model{}, fmt.Errorf("final canonical model: %w", err)
	}
	if err := store.validateCitations(store.state); err != nil {
		return passport.Model{}, err
	}
	if !store.state.Finalized {
		next := cloneCheckpoint(store.state)
		next.Finalized = true
		next.LastChange = &Change{Kind: "finalize"}
		if err := store.publish(ctx, next); err != nil {
			return passport.Model{}, err
		}
	}
	return cloneModel(store.state.Model), nil
}

func (store *ModelStore) publish(ctx context.Context, next Checkpoint) error {
	if store.state.Finalized {
		return errors.New("finalized analysis cannot be changed")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("persist analysis checkpoint: %w", err)
	}
	next.Revision = store.state.Revision + 1
	if err := store.persist(ctx, cloneCheckpoint(next)); err != nil {
		return fmt.Errorf("persist analysis checkpoint: %w", err)
	}
	store.state = cloneCheckpoint(next)
	if next.LastChange != nil && next.LastChange.AreaID != "" {
		if store.areaRevisions == nil {
			store.areaRevisions = make(map[string]int)
		}
		store.areaRevisions[next.LastChange.AreaID]++
	}
	return nil
}

// AreaRevision ignores peer writes when detecting a stalled synthesis worker.
func (store *ModelStore) AreaRevision(id string) int {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.areaRevisions[id]
}

// Collection merges accepted analysis reads and citations into source coverage.
func (store *ModelStore) Collection() source.Collection {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.collectionLocked()
}

func (store *ModelStore) collectionLocked() source.Collection {
	collection := cloneCollection(store.collection)
	cited := make(map[string]bool)
	for _, evidence := range store.state.Model.Evidence {
		cited[evidence.DocumentID] = true
	}
	for index := range collection.Snapshots {
		snapshot := &collection.Snapshots[index]
		files := make(map[string]int)
		for position, file := range snapshot.Coverage.Files {
			files[file.ID] = position
			snapshot.Coverage.Files[position].Cited = cited[file.ID]
		}
		for _, document := range collection.Documents {
			if document.SnapshotID != snapshot.ID {
				continue
			}
			position, exists := files[document.ID]
			if !exists {
				position = len(snapshot.Coverage.Files)
				files[document.ID] = position
				snapshot.Coverage.Files = append(snapshot.Coverage.Files, source.FileCoverage{ID: document.ID, Path: document.Path, Bytes: len(document.Content)})
			}
			snapshot.Coverage.Files[position].ReadRanges = slices.Clone(store.state.ReadRanges[document.ID])
			snapshot.Coverage.Files[position].Cited = cited[document.ID]
		}
		snapshot.Coverage.ReadFiles, snapshot.Coverage.CitedFiles = 0, 0
		for _, file := range snapshot.Coverage.Files {
			if len(file.ReadRanges) > 0 {
				snapshot.Coverage.ReadFiles++
			}
			if file.Cited {
				snapshot.Coverage.CitedFiles++
			}
		}
	}
	return collection
}

func boundedPart(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode analysis part: %w", err)
	}
	if len(data) > MaxPartBytes {
		return fmt.Errorf("analysis part exceeds %d bytes; split the edit", MaxPartBytes)
	}
	return nil
}

func validateQuestions(values []string) error {
	seen := make(map[string]bool)
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			return errors.New("questions and gaps must be nonempty and unique")
		}
		seen[value] = true
	}
	return nil
}

func mergeRanges(ranges []source.LineRange) []source.LineRange {
	result := slices.Clone(ranges)
	slices.SortFunc(result, func(a, b source.LineRange) int {
		if a.StartLine != b.StartLine {
			return a.StartLine - b.StartLine
		}
		return a.EndLine - b.EndLine
	})
	merged := make([]source.LineRange, 0, len(result))
	for _, span := range result {
		if len(merged) > 0 && span.StartLine <= merged[len(merged)-1].EndLine+1 {
			merged[len(merged)-1].EndLine = max(merged[len(merged)-1].EndLine, span.EndLine)
		} else {
			merged = append(merged, span)
		}
	}
	return merged
}

func cloneTasks(tasks []Task) []Task {
	result := slices.Clone(tasks)
	for index := range result {
		result[index].FindingIDs = slices.Clone(result[index].FindingIDs)
	}
	return result
}

func cloneCheckpoint(state Checkpoint) Checkpoint {
	state.Model = cloneModel(state.Model)
	state.Tasks = cloneTasks(state.Tasks)
	state.Gaps = slices.Clone(state.Gaps)
	ranges := make(map[string][]source.LineRange, len(state.ReadRanges))
	for id, spans := range state.ReadRanges {
		ranges[id] = slices.Clone(spans)
	}
	state.ReadRanges = ranges
	if state.LastChange != nil {
		change := *state.LastChange
		change.EntityIDs = slices.Clone(change.EntityIDs)
		state.LastChange = &change
	}
	return state
}

func cloneModel(model passport.Model) passport.Model {
	model.Objects = slices.Clone(model.Objects)
	for index := range model.Objects {
		object := &model.Objects[index]
		object.Inputs, object.Outputs, object.Contracts, object.ClaimIDs = slices.Clone(object.Inputs), slices.Clone(object.Outputs), slices.Clone(object.Contracts), slices.Clone(object.ClaimIDs)
	}
	model.Relations = slices.Clone(model.Relations)
	for index := range model.Relations {
		model.Relations[index].ClaimIDs = slices.Clone(model.Relations[index].ClaimIDs)
	}
	model.Claims = slices.Clone(model.Claims)
	for index := range model.Claims {
		model.Claims[index].EvidenceIDs = slices.Clone(model.Claims[index].EvidenceIDs)
	}
	model.Evidence = slices.Clone(model.Evidence)
	model.Scenarios = slices.Clone(model.Scenarios)
	for index := range model.Scenarios {
		scenario := &model.Scenarios[index]
		scenario.Inputs, scenario.Outputs, scenario.FailureModes = slices.Clone(scenario.Inputs), slices.Clone(scenario.Outputs), slices.Clone(scenario.FailureModes)
		scenario.Steps = slices.Clone(scenario.Steps)
		for stepIndex := range scenario.Steps {
			step := &scenario.Steps[stepIndex]
			step.ObjectIDs, step.ClaimIDs = slices.Clone(step.ObjectIDs), slices.Clone(step.ClaimIDs)
		}
	}
	model.OpenQuestions = slices.Clone(model.OpenQuestions)
	return model
}

func cloneCollection(collection source.Collection) source.Collection {
	collection.Documents = slices.Clone(collection.Documents)
	collection.Snapshots = slices.Clone(collection.Snapshots)
	for index := range collection.Snapshots {
		coverage := &collection.Snapshots[index].Coverage
		coverage.Files = slices.Clone(coverage.Files)
		for fileIndex := range coverage.Files {
			coverage.Files[fileIndex].ReadRanges = slices.Clone(coverage.Files[fileIndex].ReadRanges)
		}
	}
	return collection
}

func (store *ModelStore) restoreCheckpoint(input Input, fingerprint string) error {
	if input.Checkpoint != nil {
		store.state = cloneCheckpoint(*input.Checkpoint)
		if store.state.Version != CheckpointVersion || store.state.SourceFingerprint != fingerprint || store.state.Revision < 0 {
			return errors.New("analysis checkpoint does not match the pinned sources or version")
		}
	} else {
		store.state.Tasks = cloneTasks(input.Tasks)
		for i := range store.state.Tasks {
			if store.state.Tasks[i].Status == "" {
				store.state.Tasks[i].Status = TaskPending
			}
		}
	}
	if store.state.ReadRanges == nil {
		store.state.ReadRanges = make(map[string][]source.LineRange)
	}
	if err := store.validateTasks(store.state.Tasks); err != nil {
		return err
	}
	if err := validateQuestions(store.state.Gaps); err != nil {
		return err
	}
	if err := validateQuestions(store.state.Model.OpenQuestions); err != nil {
		return err
	}
	return nil
}

func (store *ModelStore) restoreSourceReads(collection source.Collection, documents map[string]source.Document) error {
	for id, ranges := range store.state.ReadRanges {
		for _, span := range ranges {
			if err := store.validateRange(id, span); err != nil {
				return fmt.Errorf("restore analysis reads: %w", err)
			}
		}
		if _, known := store.lines[id]; !known {
			return fmt.Errorf("checkpoint has an unknown source document: %s", id)
		}
		store.state.ReadRanges[id] = mergeRanges(ranges)
	}
	for _, snapshot := range collection.Snapshots {
		seenFiles := make(map[string]bool)
		for _, file := range snapshot.Coverage.Files {
			if file.ID != "" && seenFiles[file.ID] {
				return fmt.Errorf("duplicate source coverage entry: %s", file.ID)
			}
			seenFiles[file.ID] = true
			if len(file.ReadRanges) > 0 {
				document, known := documents[file.ID]
				if !known || document.SnapshotID != snapshot.ID || document.Path != file.Path {
					return fmt.Errorf("research reads do not match source ownership: %s", file.ID)
				}
			}
			for _, span := range file.ReadRanges {
				if err := store.validateRange(file.ID, span); err != nil {
					return fmt.Errorf("restore research reads: %w", err)
				}
				store.state.ReadRanges[file.ID] = mergeRanges(append(store.state.ReadRanges[file.ID], span))
			}
		}
	}
	return nil
}

func updateModelQuestions(next *Checkpoint, part ModelPart) error {
	for _, pair := range []struct {
		target *[]string
		values []string
	}{{&next.Model.OpenQuestions, part.OpenQuestions}, {&next.Gaps, part.Gaps}} {
		if pair.values == nil {
			continue
		}
		if err := validateQuestions(pair.values); err != nil {
			return err
		}
		if len(*pair.target) > 0 && !slices.Equal(*pair.target, pair.values) && strings.TrimSpace(part.Correction) == "" {
			return errors.New("changed questions or gaps require a correction explanation")
		}
		*pair.target = slices.Clone(pair.values)
	}
	return nil
}

// repairDocumentIDs fixes source IDs the model copied with one or two wrong characters
// (whitespace is already removed by the tool). Document IDs are random, so a single near
// match is the intended source.
func repairDocumentIDs(evidence []passport.Evidence, lines map[string]int) []passport.Evidence {
	repaired := slices.Clone(evidence)
	for i := range repaired {
		repaired[i].DocumentID = resolveDocumentID(repaired[i].DocumentID, lines)
	}
	return repaired
}

func resolveDocumentID(id string, lines map[string]int) string {
	if _, ok := lines[id]; ok {
		return id
	}
	match := ""
	for known := range lines {
		if withinEdits(id, known, 2) {
			if match != "" {
				return id // Ambiguous: keep the original so validation explains the problem.
			}
			match = known
		}
	}
	if match == "" {
		return id
	}
	return match
}

// withinEdits reports whether a and b differ by at most limit insertions, deletions or substitutions.
func withinEdits(a, b string, limit int) bool {
	if a == "" || b == "" || len(a)-len(b) > limit || len(b)-len(a) > limit {
		return false
	}
	previous := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		best := current[0]
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
			best = min(best, current[j])
		}
		if best > limit {
			return false
		}
		previous = current
	}
	return previous[len(b)] <= limit
}
