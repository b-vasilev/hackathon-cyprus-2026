package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

const taskTurns = 16

type researchTask struct {
	Key          string      `json:"key"`
	Boundaries   []string    `json:"responsibilities,omitempty"`
	Handoffs     []string    `json:"handoffs,omitempty"`
	Candidates   []Candidate `json:"candidates,omitempty"`
	base         planInput
	priority     int
	Question     string   `json:"question"`
	Exclude      []string `json:"other_worker_scopes,omitempty"`
	historyStart int
	questions    []string
}

func (t *researchTask) ownsCandidate(id string) bool {
	return slices.ContainsFunc(t.Candidates, func(c Candidate) bool { return c.ID == id })
}

func (t *researchTask) ownsQuestion(q researchQuestion) bool {
	if t.ownsCandidate(q.Key) {
		return true
	}
	for _, name := range append(slices.Clone(t.Boundaries), t.Handoffs...) {
		if separator := strings.LastIndex(q.Key, "/"); separator >= 0 && q.Key[:separator] == name {
			return true
		}
	}
	return false
}

// Workers may read dependencies anywhere, but only write their assigned work.
// Existing responsibilities outside the assignment are read-only references.
func (w *workspace) validateTaskPatch(input planInput) error {
	t := w.task
	if t == nil {
		return nil
	}
	if input.Replace {
		return errors.New("a worker cannot replace the shared research plan")
	}
	if strings.HasPrefix(t.Key, "scope/") {
		names := map[string]bool{}
		for _, name := range t.Boundaries {
			names[name] = true
		}
		for _, b := range append(slices.Clone(w.plan), input.Boundaries...) {
			if !slices.ContainsFunc(t.base.Boundaries, func(old researchBoundary) bool { return old.Name == b.Name }) {
				names[b.Name] = true
			}
		}
		if len(names) > 1 {
			return errors.New("save one subsystem summary for the assigned source area; keep files, helpers and variants as evidence under that responsibility")
		}
	}
	for _, b := range input.Boundaries {
		if err := w.validateTaskBoundary(b); err != nil {
			return err
		}
	}
	for _, d := range input.Candidates {
		if !t.ownsCandidate(d.CandidateID) {
			return fmt.Errorf("candidate %s belongs to another task", d.CandidateID)
		}
	}
	for _, h := range input.Handoffs {
		if slices.ContainsFunc(t.base.Handoffs, func(old repositoryHandoff) bool { return old.ID == h.ID }) && !slices.Contains(t.Handoffs, h.ID) {
			return fmt.Errorf("handoff %s belongs to another task", h.ID)
		}
	}
	return nil
}

func (t *researchTask) ownsPath(file string) bool {
	for _, excluded := range t.Exclude {
		if file == excluded || strings.HasPrefix(file, excluded+"/") {
			return false
		}
	}
	for _, c := range t.Candidates {
		prefix := path.Clean(c.Path)
		if file == prefix || ((c.Kind == "source_area" || c.Kind == "repository") && strings.HasPrefix(file, prefix+"/")) {
			return true
		}
	}
	return false
}

// Research architecture source areas, not every file hint. Keep each
// responsibility's proof roles together to avoid conflicting writers.
func (w *workspace) researchTasks() []researchTask {
	groups := map[string]*researchTask{}
	group := func(key string, q researchQuestion) *researchTask {
		if groups[key] == nil {
			groups[key] = &researchTask{Key: key, priority: q.priority}
		} else {
			groups[key].priority = min(groups[key].priority, q.priority)
		}
		groups[key].questions = append(groups[key].questions, q.Key)
		return groups[key]
	}
	for _, q := range w.pendingQuestions() {
		if !q.Actionable {
			continue
		}
		if q.Kind == "proof_obligation" {
			w.groupProofQuestion(q, group)
			continue
		}
		w.groupCandidateQuestion(q, group)
	}
	result := make([]researchTask, 0, len(groups))
	for _, task := range groups {
		switch {
		case len(task.Boundaries) > 0:
			task.Question = "Resolve the remaining implementation, caller, contract and validation questions for " + strings.Join(task.Boundaries, ", ")
		case len(task.Handoffs) > 0:
			task.Question = "Verify producer, consumer and wiring for " + strings.Join(task.Handoffs, ", ")
		default:
			task.Question = "Explain the architecture role of " + w.taskLabel(*task) + " using one representative input -> processing -> output flow, its caller and external contracts. Save one subsystem summary; do not catalog individual files or helpers."
		}
		for _, other := range groups {
			if other.Key != task.Key && strings.HasPrefix(other.Key, task.Key+"/") {
				task.Exclude = append(task.Exclude, strings.TrimPrefix(other.Key, "scope/"))
			}
		}
		slices.Sort(task.Exclude)
		result = append(result, *task)
	}
	slices.SortFunc(result, func(a, b researchTask) int {
		if a.priority != b.priority {
			return a.priority - b.priority
		}
		return strings.Compare(a.Key, b.Key)
	})
	return result
}

func (w *workspace) taskLabel(task researchTask) string {
	if len(task.Boundaries) > 0 {
		return strings.Join(task.Boundaries, ", ")
	}
	if len(task.Handoffs) > 0 {
		return "Cross-repository contract: " + strings.Join(task.Handoffs, ", ")
	}
	scope := strings.TrimPrefix(task.Key, "scope/")
	for _, repo := range w.inventory.Repositories {
		if scope == repo.ID || strings.HasPrefix(scope, repo.ID+"/") {
			name := strings.TrimSuffix(path.Base(strings.TrimRight(repo.Locator, "/")), ".git")
			area := strings.TrimPrefix(strings.TrimPrefix(scope, repo.ID), "/")
			if area == "" {
				return name + " · Repository overview"
			}
			return name + "/" + area
		}
	}
	return scope
}

func (w *workspace) taskContext() string {
	var boundaries []researchBoundary
	var handoffs []repositoryHandoff
	documents := map[string]bool{}
	include := func(refs []roleEvidence) {
		for _, ref := range refs {
			documents[w.documents[ref.Path].ID] = true
		}
	}
	for _, h := range w.ledger.Plan.Handoffs {
		if slices.Contains(w.task.Handoffs, h.ID) {
			handoffs = append(handoffs, h)
		}
	}
	for _, b := range w.plan {
		if slices.Contains(w.task.Boundaries, b.Name) || slices.ContainsFunc(handoffs, func(h repositoryHandoff) bool { return h.FromBoundary == b.Name || h.ToBoundary == b.Name }) {
			boundaries = append(boundaries, b)
			for _, refs := range b.roles() {
				include(refs)
			}
		}
	}
	for _, c := range w.task.Candidates {
		for file, doc := range w.documents {
			if path.Clean(c.Path) == file || strings.HasPrefix(file, path.Clean(c.Path)+"/") {
				documents[doc.ID] = true
			}
		}
	}
	var findings []researchFinding
	for _, finding := range w.ledger.Findings {
		if slices.ContainsFunc(citationPattern.FindAllStringSubmatch(finding.Notes, -1), func(match []string) bool { return documents[match[1]] }) {
			findings = append(findings, finding)
		}
	}
	readRanges := map[string]any{}
	for file, doc := range w.documents {
		if documents[doc.ID] && len(w.reads[doc.ID]) > 0 {
			readRanges[file] = map[string]any{"document_id": doc.ID, "ranges": mergeRanges(w.reads[doc.ID])}
		}
	}
	remaining := w.pendingTaskQuestions()
	data, _ := json.Marshal(map[string]any{"assignment": w.task, "remaining_questions": remaining, "existing_responsibilities": boundaries, "existing_handoffs": handoffs, "retained_findings": findings, "already_read": readRanges})
	return string(data)
}

func (w *workspace) forkTask(task researchTask, turns int) (*workspace, error) {
	data, err := json.Marshal(w.ledger)
	if err != nil {
		return nil, fmt.Errorf("copy research ledger: %w", err)
	}
	var ledger Ledger
	if decodeErr := json.Unmarshal(data, &ledger); decodeErr != nil {
		return nil, fmt.Errorf("copy research ledger: %w", decodeErr)
	}
	base, err := clonePlan(ledger.Plan)
	if err != nil {
		return nil, err
	}
	task.base = base
	task.historyStart = len(ledger.History)
	local := &workspace{
		limiter:   w.limiter,
		documents: w.documents, paths: w.paths, inventory: w.inventory,
		sources: w.sourceIndex(),
		reads:   map[string][]source.LineRange{}, readFailures: map[string]string{},
		ledger: ledger, plan: ledger.Plan.Boundaries, searches: slices.Clone(w.searches),
		summaryNotes: w.summaryNotes, submittedNotes: w.submittedNotes,
		turnBudget: turns, deadline: w.deadline, task: &task,
		conversation: w.conversation,
	}
	for id, ranges := range w.reads {
		local.reads[id] = slices.Clone(ranges)
	}
	maps.Copy(local.readFailures, w.readFailures)
	return local, nil
}

// Publish an immutable copy between turns. Later corrections use the last
// published plan as their merge base, so they do not conflict with themselves.
func (w *workspace) taskSnapshot() (*workspace, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, b := range w.plan {
		if !slices.ContainsFunc(w.task.base.Boundaries, func(old researchBoundary) bool { return old.Name == b.Name }) && !slices.Contains(w.task.Boundaries, b.Name) {
			w.task.Boundaries = append(w.task.Boundaries, b.Name)
		}
	}
	for _, h := range w.ledger.Plan.Handoffs {
		if !slices.ContainsFunc(w.task.base.Handoffs, func(old repositoryHandoff) bool { return old.ID == h.ID }) && !slices.Contains(w.task.Handoffs, h.ID) {
			w.task.Handoffs = append(w.task.Handoffs, h.ID)
		}
	}
	snapshot, err := w.forkTask(*w.task, w.turnBudget)
	if err != nil {
		return nil, err
	}
	snapshot.task.base, snapshot.task.historyStart = w.task.base, w.task.historyStart
	w.task.base, err = clonePlan(w.ledger.Plan)
	if err != nil {
		return nil, err
	}
	w.task.historyStart = len(w.ledger.History)
	return snapshot, nil
}

// Compare against the worker's starting revision before applying a patch. A
// late result must never overwrite another worker's newer evidence.
func changedEntries[T any](base, updated, current []T, key func(T) string) ([]T, error) {
	var changes []T
	for _, item := range updated {
		id := key(item)
		before := slices.IndexFunc(base, func(v T) bool { return key(v) == id })
		if before >= 0 && reflect.DeepEqual(base[before], item) {
			continue
		}
		now := slices.IndexFunc(current, func(v T) bool { return key(v) == id })
		if now >= 0 && !reflect.DeepEqual(current[now], item) && (before < 0 || !reflect.DeepEqual(base[before], current[now])) {
			return nil, fmt.Errorf("concurrent research conflict for %s; keep saved evidence and requeue the task", id)
		}
		changes = append(changes, item)
	}
	return changes, nil
}

// Only the coordinator calls this method. Independent findings and source
// reads survive even if a stale plan patch needs another pass.
func (w *workspace) mergeTask(local *workspace) error {
	if w.sessions == nil {
		w.sessions = map[string]taskSession{}
	}
	w.sessions[local.task.Key] = local.conversation
	for _, b := range local.plan {
		if slices.Contains(local.task.Boundaries, b.Name) {
			w.sessions["responsibility/"+b.Repository+"/"+b.Name] = local.conversation
		}
	}
	for id, ranges := range local.reads {
		w.reads[id] = mergeRanges(append(w.reads[id], ranges...))
	}
	for file, reason := range local.readFailures {
		if len(w.reads[w.documents[file].ID]) == 0 {
			w.readFailures[file] = reason
		}
	}
	for _, record := range local.ledger.Searches {
		scope := searchedScope{Pattern: record.Pattern, Glob: record.Glob}
		if !slices.Contains(w.searches, scope) {
			w.searches = append(w.searches, scope)
			w.ledger.Searches = append(w.ledger.Searches, record)
		}
	}
	w.toolCalls += local.toolCalls
	w.mcpReads += local.mcpReads
	for _, finding := range local.ledger.Findings {
		if err := w.saveFinding(finding); err != nil {
			return err
		}
	}
	base, updated, current := local.task.base, local.ledger.Plan, w.ledger.Plan
	boundaries, err := changedEntries(base.Boundaries, updated.Boundaries, current.Boundaries, func(b researchBoundary) string { return b.Name })
	if err != nil {
		return err
	}
	candidates, err := changedEntries(base.Candidates, updated.Candidates, current.Candidates, func(d candidateDisposition) string { return d.CandidateID })
	if err != nil {
		return err
	}
	handoffs, err := changedEntries(base.Handoffs, updated.Handoffs, current.Handoffs, func(h repositoryHandoff) string { return h.ID })
	if err != nil {
		return err
	}
	if len(boundaries)+len(candidates)+len(handoffs) == 0 {
		return nil
	}
	var reclassifications []reclassification
	for _, patch := range local.ledger.History[local.task.historyStart:] {
		reclassifications = append(reclassifications, patch.Reclassifications...)
	}
	return w.setPlan(planInput{Boundaries: boundaries, Candidates: candidates, Handoffs: handoffs, Reclassifications: reclassifications})
}

type taskUpdate struct {
	id         string
	progress   Progress
	local      *workspace
	snapshot   *workspace
	result     Result
	err        error
	key        string
	allowance  int
	progressed bool
}

type taskAttempts struct {
	count     int
	stalled   int
	questions string
	repaired  bool
}

func queueResearchTasks(tasks []researchTask, active map[string]researchTask, attempts map[string]taskAttempts) []researchTask {
	queued := make([]researchTask, 0, len(tasks))
	for _, task := range tasks {
		claimed := false
		for _, running := range active {
			if tasksOverlap(task, running) {
				claimed = true
				break
			}
		}
		if claimed {
			continue
		}
		attempt := attempts[task.Key]
		questions := strings.Join(task.questions, "\x00")
		if attempt.questions != questions {
			attempt.stalled, attempt.questions = 0, questions
			attempt.repaired = false
			attempts[task.Key] = attempt
		}
		if attempt.stalled < 2 || (attempt.stalled == 2 && !attempt.repaired) {
			queued = append(queued, task)
		}
	}
	return queued
}

func tasksOverlap(a, b researchTask) bool {
	if a.Key == b.Key {
		return true
	}
	for _, name := range a.Boundaries {
		if slices.Contains(b.Boundaries, name) {
			return true
		}
	}
	for _, id := range a.Handoffs {
		if slices.Contains(b.Handoffs, id) {
			return true
		}
	}
	for _, candidate := range a.Candidates {
		if b.ownsCandidate(candidate.ID) {
			return true
		}
	}
	return false
}

// The event loop owns the shared ledger, budget, queue and progress callback.
// Each goroutine owns a deep copy of mutable research state. The MCP source
// session and indexed documents are shared read-only.
func (w *workspace) investigateParallel(ctx context.Context, workers int, newAgent func(*workspace) fantasy.Agent, config llmgateway.Config, modelID, orientation string, progress func(Progress) error, result *Result) error {
	if workers == 1 {
		return w.investigate(ctx, newAgent(w), config, modelID, orientation, progress, result)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	w.startParallelBudget(workers)
	if w.sessions == nil {
		w.sessions = map[string]taskSession{}
	}
	w.parallel = ParallelProgress{Limit: workers, Workers: []WorkerProgress{}}
	updates := make(chan taskUpdate, workers*4)
	// Respect the selected connection's request limit without overflowing its
	// admission queue when the workers reach a model step together.
	stepGate := make(chan struct{}, min(workers, max(config.MaxConcurrent, 1)))
	active := map[string]researchTask{}
	attempts := map[string]taskAttempts{}
	reserved, sequence := 0, 0
	var fatal error
	for {
		tasks := w.researchTasks()
		slices.SortStableFunc(tasks, func(a, b researchTask) int {
			if a.priority != b.priority {
				return a.priority - b.priority
			}
			return attempts[a.Key].count - attempts[b.Key].count
		})
		queued := queueResearchTasks(tasks, active, attempts)
		w.refillForParallelTasks(workers, len(active), len(queued), reserved)
		for w.parallelCanStart(ctx, workers-len(active), len(queued), reserved, fatal) {
			task := queued[0]
			queued = queued[1:]
			available := w.activeLimit() - w.turn - reserved
			slots := min(workers-len(active), len(queued)+1)
			allowance := min(taskTurns, max(1, available/slots))
			local, err := w.forkTask(task, allowance)
			if err != nil {
				fatal = err
				cancel()
				break
			}
			sequence++
			local.stepGate = stepGate
			local.conversation = w.sessions[task.Key]
			local.ledger.Focus = Focus{Area: w.taskLabel(task), Question: task.Question, Reason: "Investigate this independently assigned product responsibility."}
			id := fmt.Sprintf("research-%d", sequence)
			active[task.Key] = task
			reserved += allowance
			attempt := attempts[task.Key]
			attempt.count++
			repair := attempt.stalled == 2 && !attempt.repaired
			if repair {
				attempt.repaired = true
				local.conversation = taskSession{}
			}
			attempts[task.Key] = attempt
			recovering := w.budget.decisions[len(w.budget.decisions)-1].Kind == "recovery"
			w.parallel.Workers = append(w.parallel.Workers, WorkerProgress{ID: id, Task: w.taskLabel(task), Question: task.Question, Focus: local.ledger.Focus, Status: "waiting"})
			go runParallelTask(ctx, config, modelID, newAgent, updates, taskUpdate{id: id, key: task.Key, local: local, allowance: allowance}, recovering, repair)
		}
		w.refreshParallelCounts(len(queued))
		if len(active) == 0 {
			stop, idleErr := w.parallelIdle(ctx, len(queued), fatal)
			if idleErr != nil {
				return idleErr
			}
			if stop {
				break
			}
			continue
		}
		if err := w.publish(progress, "Parallel source research is running."); err != nil {
			fatal = err
			cancel()
		}
		update := <-updates
		if update.snapshot != nil {
			active[update.key] = *update.snapshot.task
			if err := w.mergeTask(update.snapshot); err != nil {
				w.activity = fmt.Sprintf("Task %s has a conflicting plan patch; source findings are retained for continuation.", update.key)
			}
		}
		if update.local != nil {
			released, taskErr := w.completeParallelTask(ctx, update, active, attempts, fatal)
			reserved -= released
			fatal = taskErr
			if fatal != nil {
				cancel()
			}
			continue
		}
		spent, progressErr := w.applyParallelProgress(update, result, progress)
		reserved -= spent
		if progressErr != nil {
			fatal = progressErr
			cancel()
		}
	}
	return w.finishParallelResearch(progress)
}

func (w *workspace) validateTaskBoundary(b researchBoundary) error {
	t := w.task
	if (strings.HasPrefix(t.Key, "responsibility/") || strings.HasPrefix(t.Key, "handoff/")) && !slices.Contains(t.Boundaries, b.Name) {
		return fmt.Errorf("finish the assigned responsibility instead of creating %s; save helper and variant details as evidence under the existing responsibility", b.Name)
	}
	if slices.ContainsFunc(t.base.Boundaries, func(old researchBoundary) bool { return old.Name == b.Name }) && !slices.Contains(t.Boundaries, b.Name) {
		return fmt.Errorf("responsibility %s belongs to another task; reference it without changing it", b.Name)
	}
	if !slices.Contains(t.Boundaries, b.Name) {
		owned := false
		for _, refs := range b.roles() {
			for _, ref := range refs {
				if t.ownsPath(ref.Path) {
					owned = true
				}
			}
		}
		if !owned {
			return fmt.Errorf("responsibility %s has no evidence in this assignment; other source areas belong to other workers", b.Name)
		}
	}
	return nil
}

func (w *workspace) groupProofQuestion(q researchQuestion, group func(string, researchQuestion) *researchTask) {
	for _, b := range w.plan {
		separator := strings.LastIndex(q.Key, "/")
		if separator >= 0 && q.Key[:separator] == b.Name && roleQuestions[q.Key[separator+1:]] != "" {
			t := group("responsibility/"+b.Repository+"/"+b.Name, q)
			if !slices.Contains(t.Boundaries, b.Name) {
				t.Boundaries = append(t.Boundaries, b.Name)
			}
		}
	}
	for _, h := range w.ledger.Plan.Handoffs {
		separator := strings.LastIndex(q.Key, "/")
		if separator >= 0 && q.Key[:separator] == h.ID {
			t := group("handoff/"+h.ID, q)
			if !slices.Contains(t.Handoffs, h.ID) {
				t.Handoffs = append(t.Handoffs, h.ID)
			}
		}
	}
	if strings.HasPrefix(q.Key, "repository/") {
		group("scope/"+q.Repository, q)
	}
}

func (w *workspace) groupCandidateQuestion(q researchQuestion, group func(string, researchQuestion) *researchTask) {
	for _, c := range w.inventory.Candidates {
		if c.ID != q.Key {
			continue
		}
		var owner *researchTask
		for _, b := range w.plan {
			if b.Repository == c.Repository && boundaryTouches(b, c) {
				owner = group("responsibility/"+b.Repository+"/"+b.Name, q)
				if !slices.Contains(owner.Boundaries, b.Name) {
					owner.Boundaries = append(owner.Boundaries, b.Name)
				}
				break
			}
		}
		if owner != nil {
			owner.Candidates = append(owner.Candidates, c)
			continue
		}
		scope := w.candidateTaskScope(c)
		t := group("scope/"+scope, q)
		t.Candidates = append(t.Candidates, c)
	}
}

func (w *workspace) candidateTaskScope(c Candidate) string {
	scope := path.Clean(c.Path)
	if c.Kind != "source_area" && c.Kind != "repository" {
		for _, area := range w.inventory.Candidates {
			prefix := path.Clean(area.Path)
			if area.Kind == "source_area" && isArchitectureScope(area) && area.Repository == c.Repository && (path.Clean(c.Path) == prefix || strings.HasPrefix(c.Path, prefix+"/")) {
				if scope == path.Clean(c.Path) || len(prefix) > len(scope) {
					scope = prefix
				}
			}
		}
	}
	return scope
}

func (w *workspace) pendingTaskQuestions() []researchQuestion {
	var remaining []researchQuestion
	for _, question := range w.pendingQuestions() {
		if question.Actionable && w.task.ownsQuestion(question) {
			remaining = append(remaining, question)
		}
	}
	return remaining
}

func runParallelTask(ctx context.Context, config llmgateway.Config, modelID string, newAgent func(*workspace) fantasy.Agent, updates chan<- taskUpdate, launch taskUpdate, recovering, repair bool) {
	local, id, key, allowance := launch.local, launch.id, launch.key, launch.allowance
	before := local.workFingerprint()
	send := func(p Progress) error {
		update := taskUpdate{id: id, key: key, progress: p}
		if p.Step > 0 {
			var err error
			update.snapshot, err = local.taskSnapshot()
			if err != nil {
				return err
			}
		}
		select {
		case updates <- update:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	local.event = func(message string) error { return local.publish(send, message) }
	instructions := "PARALLEL ASSIGNMENT:\n" + local.taskContext() + "\nInvestigate ONLY this assigned question. Its candidate map and existing results are already supplied above; start with a focused source search or read, without reviewing inventory or selecting another question. Other workers exclusively own other_worker_scopes; read them only as dependencies, never independently investigate or change their responsibilities. Keep dependencies outside the assignment read-only. Use save_research_answer immediately after verifying EACH role; it saves the answer, exact citations and matching plan evidence together. Do not wait to finish all four roles. Continue with remaining roles after saving the first. Batch independent reads. With three turns left, prioritize saving known evidence. Return after saving; your conversation, read ranges and results are retained for continuation. Never return an unsaved prose-only answer or decide global completion."
	if recovering {
		instructions += "\nThe shared research budget is in its final recovery window. Close existing questions and save known evidence. Do not create new responsibilities or broaden discovery."
	}
	if repair {
		instructions += "\nThis is a fresh closure attempt after two stalled windows. Use remaining_questions and the retained read ranges/findings. Save the missing role evidence, or record a precise role research_gap supported by actual prior searches when evidence cannot be established. Do not repeat unrelated discovery, create responsibilities or return only prose."
	}
	var usage Result
	err := local.modelPass(ctx, newAgent(local), config, modelID, instructions, send, &usage)
	updates <- taskUpdate{id: id, key: key, local: local, result: usage, err: err, allowance: allowance, progressed: before != local.workFingerprint()}
}

func (w *workspace) refreshParallelCounts(queued int) {
	w.parallel.Active, w.parallel.Waiting, w.parallel.Requests = 0, 0, 0
	for _, worker := range w.parallel.Workers {
		if worker.Status == "waiting" {
			w.parallel.Waiting++
		} else {
			w.parallel.Active++
		}
		if worker.Status == "requesting" {
			w.parallel.Requests++
		}
	}
	w.parallel.Queued = queued
}

func (w *workspace) completeParallelTask(ctx context.Context, update taskUpdate, active map[string]researchTask, attempts map[string]taskAttempts, fatal error) (int, error) {
	mergeErr := w.mergeTask(update.local)
	// A request/tool failure is not evidence that research is exhausted.
	// Preserve accepted work, then expose the failure for a resumable retry.
	if update.err != nil && ctx.Err() == nil && fatal == nil {
		fatal = fmt.Errorf("research task %s failed: %w", w.taskLabel(active[update.key]), update.err)
	}
	delete(active, update.key)
	released := update.allowance - update.result.Steps
	w.parallel.Workers = slices.DeleteFunc(w.parallel.Workers, func(p WorkerProgress) bool { return p.ID == update.id })
	attempt := attempts[update.key]
	// A failed later model request must not erase actual saved progress.
	if !update.progressed || mergeErr != nil {
		attempt.stalled++
	} else {
		attempt.stalled = 0
	}
	attempts[update.key] = attempt
	if update.err != nil && fatal != nil {
		w.activity = fatal.Error()
	} else if mergeErr != nil {
		w.activity = fmt.Sprintf("Task %s saved available evidence and remains open for continuation.", update.key)
	}
	return released, fatal
}

func (w *workspace) applyParallelProgress(update taskUpdate, result *Result, progress func(Progress) error) (int, error) {
	spent := 0
	for i := range w.parallel.Workers {
		worker := &w.parallel.Workers[i]
		if worker.ID == update.id && update.progress.State != nil {
			worker.Status = update.progress.State.Execution
			if update.progress.Step > 0 {
				worker.Turns = update.progress.Step
			}
			worker.Focus = update.progress.State.Focus
			worker.Activity = update.progress.State.Activity
			w.ledger.Focus, w.activity = worker.Focus, worker.Activity
		}
	}
	if update.progress.Step > 0 || update.progress.Retry {
		if update.progress.Step > 0 {
			w.turn++
			spent++
			result.Steps++
			result.InputTokens += update.progress.InputTokens
			result.OutputTokens += update.progress.OutputTokens
		}
		state := w.state()
		p := update.progress
		p.State = &state
		if p.Step > 0 {
			p.Step = result.Steps
			p.Message = fmt.Sprintf("Research model turn %d completed across parallel workers.", result.Steps)
		}
		if err := progress(p); err != nil {
			return spent, fmt.Errorf("publish parallel research progress: %w", err)
		}
	}
	return spent, nil
}

func (w *workspace) parallelCanStart(ctx context.Context, slots, queued, reserved int, fatal error) bool {
	return fatal == nil && ctx.Err() == nil && !w.budget.stopped && slots > 0 && queued > 0 && w.activeLimit()-w.turn-reserved > 0 && (w.deadline.IsZero() || time.Until(w.deadline) > 30*time.Second)
}

func (w *workspace) refillForParallelTasks(workers, active, queued, reserved int) {
	if active < workers && queued > 0 && w.activeLimit()-w.turn-reserved == 0 {
		w.refillParallelBudget()
	}
}

func (w *workspace) parallelIdle(ctx context.Context, queued int, fatal error) (bool, error) {
	if fatal != nil {
		return false, fatal
	}
	select {
	case <-ctx.Done():
		return true, nil
	default:
	}
	if queued == 0 {
		w.budget.stopped = true
		w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: "stopped", Reason: "No independent tasks remain after stalled assignments received a fresh closure attempt. Unresolved evidence remains explicit research gaps."})
		return true, nil
	}
	return !w.reviewBudget(), nil
}

func (w *workspace) finishParallelResearch(progress func(Progress) error) error {
	w.parallel.Active, w.parallel.Waiting, w.parallel.Requests, w.parallel.Queued = 0, 0, 0, len(w.researchTasks())
	if !w.budget.stopped {
		w.budget.stopped = true
		w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: "stopped", Reason: "Research time limit reached; remaining tasks are saved for continuation."})
	}
	if err := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchComplete}); err != nil {
		if partialErr := w.checkCoverage(coverageInput{Notes: w.summaryNotes, Status: researchPartial, StopReason: w.budget.decisions[len(w.budget.decisions)-1].Reason}); partialErr != nil {
			return fmt.Errorf("parallel research saved unchecked evidence: %w", partialErr)
		}
	}
	return w.finishProgress(progress)
}
