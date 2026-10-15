package research

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	for _, b := range input.Boundaries {
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

// Group proof roles for one responsibility together. Discovery candidates go
// to the deepest indexed source area, so a root directory cannot swallow the
// whole repository and leave nineteen workers idle.
func (w *workspace) researchTasks() []researchTask {
	groups := map[string]*researchTask{}
	group := func(key string, priority int) *researchTask {
		if groups[key] == nil {
			groups[key] = &researchTask{Key: key, priority: priority}
		} else {
			groups[key].priority = min(groups[key].priority, priority)
		}
		return groups[key]
	}
	for _, q := range w.pendingQuestions() {
		if !q.Actionable {
			continue
		}
		if q.Kind == "proof_obligation" {
			for _, b := range w.plan {
				separator := strings.LastIndex(q.Key, "/")
				if separator >= 0 && q.Key[:separator] == b.Name && roleQuestions[q.Key[separator+1:]] != "" {
					t := group("responsibility/"+b.Repository+"/"+b.Name, q.priority)
					if !slices.Contains(t.Boundaries, b.Name) {
						t.Boundaries = append(t.Boundaries, b.Name)
					}
				}
			}
			for _, h := range w.ledger.Plan.Handoffs {
				separator := strings.LastIndex(q.Key, "/")
				if separator >= 0 && q.Key[:separator] == h.ID {
					t := group("handoff/"+h.ID, q.priority)
					if !slices.Contains(t.Handoffs, h.ID) {
						t.Handoffs = append(t.Handoffs, h.ID)
					}
				}
			}
			if strings.HasPrefix(q.Key, "repository/") {
				group("scope/"+q.Repository, q.priority)
			}
			continue
		}
		for _, c := range w.inventory.Candidates {
			if c.ID != q.Key {
				continue
			}
			var owner *researchTask
			for _, b := range w.plan {
				if b.Repository == c.Repository && boundaryTouches(b, c) {
					owner = group("responsibility/"+b.Repository+"/"+b.Name, q.priority)
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
			scope := c.Repository
			for _, area := range w.inventory.Candidates {
				prefix := path.Clean(area.Path)
				if area.Kind == "source_area" && area.Repository == c.Repository && (path.Clean(c.Path) == prefix || strings.HasPrefix(c.Path, prefix+"/")) && len(prefix) > len(scope) {
					scope = prefix
				}
			}
			t := group("scope/"+scope, q.priority)
			t.Candidates = append(t.Candidates, c)
		}
	}
	result := make([]researchTask, 0, len(groups))
	for _, task := range groups {
		if len(task.Boundaries) > 0 {
			task.Question = "Resolve the remaining implementation, caller, contract and validation questions for " + strings.Join(task.Boundaries, ", ")
		} else if len(task.Handoffs) > 0 {
			task.Question = "Verify producer, consumer and wiring for " + strings.Join(task.Handoffs, ", ")
		} else {
			task.Question = "Identify and explain the concrete behavior owned by " + w.taskLabel(*task) + "; save one source-backed answer before broadening the investigation"
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
	data, _ := json.Marshal(map[string]any{"assignment": w.task, "existing_responsibilities": boundaries, "existing_handoffs": handoffs, "retained_findings": findings, "already_read": readRanges})
	return string(data)
}

func (w *workspace) forkTask(task researchTask, turns int) (*workspace, error) {
	data, err := json.Marshal(w.ledger)
	if err != nil {
		return nil, fmt.Errorf("copy research ledger: %w", err)
	}
	var ledger Ledger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return nil, fmt.Errorf("copy research ledger: %w", err)
	}
	base, err := clonePlan(ledger.Plan)
	if err != nil {
		return nil, err
	}
	task.base = base
	task.historyStart = len(ledger.History)
	local := &workspace{
		documents: w.documents, paths: w.paths, inventory: w.inventory,
		reads: map[string][]source.LineRange{}, readFailures: map[string]string{},
		ledger: ledger, plan: ledger.Plan.Boundaries, searches: slices.Clone(w.searches),
		summaryNotes: w.summaryNotes, submittedNotes: w.submittedNotes,
		turnBudget: turns, deadline: w.deadline, task: &task,
		conversation: w.conversation,
	}
	for id, ranges := range w.reads {
		local.reads[id] = slices.Clone(ranges)
	}
	for file, reason := range w.readFailures {
		local.readFailures[file] = reason
	}
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
	count   int
	stalled int
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
		queued := make([]researchTask, 0, len(tasks))
		for _, task := range tasks {
			claimed := false
			for _, running := range active {
				if tasksOverlap(task, running) {
					claimed = true
					break
				}
			}
			if !claimed && attempts[task.Key].stalled < 2 {
				queued = append(queued, task)
			}
		}
		for fatal == nil && ctx.Err() == nil && !w.budget.stopped && len(active) < workers && len(queued) > 0 && w.activeLimit()-w.turn-reserved > 0 && (w.deadline.IsZero() || time.Until(w.deadline) > 30*time.Second) {
			task := queued[0]
			queued = queued[1:]
			allowance := min(taskTurns, w.activeLimit()-w.turn-reserved)
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
			attempts[task.Key] = attempt
			w.parallel.Workers = append(w.parallel.Workers, WorkerProgress{ID: id, Task: w.taskLabel(task), Question: task.Question, Focus: local.ledger.Focus, Status: "waiting"})
			go func() {
				before := local.workFingerprint()
				send := func(p Progress) error {
					update := taskUpdate{id: id, key: task.Key, progress: p}
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
				var usage Result
				err := local.modelPass(ctx, newAgent(local), config, modelID, instructions, send, &usage)
				updates <- taskUpdate{id: id, key: task.Key, local: local, result: usage, err: err, allowance: allowance, progressed: before != local.workFingerprint()}
			}()
		}
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
		w.parallel.Queued = len(queued)
		if len(active) == 0 {
			if fatal != nil {
				return fatal
			}
			if ctx.Err() != nil {
				break
			}
			if len(queued) == 0 {
				w.budget.stopped = true
				w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: "stopped", Reason: "No unattempted reachable tasks remain. Tasks that made no progress twice are saved for continuation."})
				break
			}
			if !w.reviewBudget() {
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
			mergeErr := w.mergeTask(update.local)
			delete(active, update.key)
			reserved -= update.allowance - update.result.Steps
			w.parallel.Workers = slices.DeleteFunc(w.parallel.Workers, func(p WorkerProgress) bool { return p.ID == update.id })
			attempt := attempts[update.key]
			if !update.progressed || mergeErr != nil || update.err != nil {
				attempt.stalled++
			} else {
				attempt.stalled = 0
			}
			attempts[update.key] = attempt
			if mergeErr != nil || update.err != nil {
				w.activity = fmt.Sprintf("Task %s saved available evidence and remains open for continuation.", update.key)
			}
			continue
		}
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
		if update.progress.Step > 0 {
			w.turn++
			reserved--
			result.Steps++
			result.InputTokens += update.progress.InputTokens
			result.OutputTokens += update.progress.OutputTokens
			state := w.state()
			p := update.progress
			p.Step, p.State = result.Steps, &state
			p.Message = fmt.Sprintf("Research model turn %d completed across parallel workers.", result.Steps)
			if err := progress(p); err != nil {
				fatal = err
				cancel()
			}
		}
	}
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
