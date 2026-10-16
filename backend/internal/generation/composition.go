package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
)

// CompositionTask is a durable unit of work, including queued and retried parts.
type CompositionTask struct {
	ID          string `json:"id"`
	ChapterID   string `json:"chapter_id"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts,omitempty"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
}

type CompositionState struct {
	Action      string            `json:"action"`
	Parallelism int               `json:"parallelism"`
	Tasks       []CompositionTask `json:"tasks"`
}

type chapterPlan struct {
	Title    string            `json:"title"`
	Question string            `json:"question"`
	Parts    []compositionPart `json:"parts"`
}

// Each planned part produces exactly one block. IDs select its source context.
type compositionPart struct {
	Title        string                  `json:"title"`
	Instructions string                  `json:"instructions"`
	Kind         string                  `json:"kind" enum:"markdown,diagram_ref,scenario_ref,object_table"`
	ObjectIDs    []string                `json:"object_ids"`
	ClaimIDs     []string                `json:"claim_ids"`
	ScenarioIDs  []string                `json:"scenario_ids"`
	TableStyle   string                  `json:"table_style,omitempty" enum:"objects,repositories,technologies"`
	Diagram      *compositionDiagramPlan `json:"diagram"`
}

type compositionDiagramPlan struct {
	Kind          string `json:"kind" enum:"repositories,context,containers,components,domain,data,deployment,states"`
	ScopeObjectID string `json:"scope_object_id"`
}

type compositionFragment struct {
	Block   passport.DocumentBlock `json:"block"`
	Diagram *passport.Diagram      `json:"diagram"`
}

type compositionTheme struct {
	CardTheme *passport.ProductCardTheme `json:"card_theme"`
}

type compositionJob struct {
	Task CompositionTask
	Work func(context.Context, generationCall) error
}

const compositionTaskQueued = "queued"
const scenarioReferenceKind = "scenario_ref"

type composer struct {
	service   *Service
	run       Run
	model     passport.Model
	documents map[string]int
	state     CompositionState
	mu        sync.Mutex
}

func (s *Service) compose(ctx context.Context, run Run, model passport.Model, documents map[string]int, brandContext string) (passport.Views, error) {
	views := passport.Views{Sections: []passport.Section{}, Diagrams: []passport.Diagram{}, Document: &passport.Document{Version: 2}}
	if err := passport.ValidateModel(model, documents); err != nil {
		return views, fmt.Errorf("canonical model is not ready for composition: %w", err)
	}
	config, err := s.Connections.Load(ctx, run.ConnectionID, run.ConnectionRevision)
	if err != nil {
		return views, fmt.Errorf("load composition connection: %w", err)
	}
	c := &composer{service: s, run: run, model: model, documents: documents, state: CompositionState{Parallelism: max(1, min(config.MaxConcurrent, 4)), Tasks: []CompositionTask{}}}
	// Details have independent plans and parts. Overview is synthesized last.
	chapters, diagrams, err := c.chapters(ctx, passport.ChapterIDs()[1:], "")
	if err != nil {
		return views, err
	}
	type chapterSummary struct{ ID, Title, Question, Answer string }
	summaries := make([]chapterSummary, 0, len(chapters))
	for _, chapter := range chapters {
		answer := ""
		for _, block := range chapter.Blocks {
			if block.Kind == "markdown" {
				answer = block.Markdown
				break
			}
		}
		summaries = append(summaries, chapterSummary{chapter.ID, chapter.Title, chapter.Question, answer})
	}
	outline, _ := json.Marshal(summaries)
	overview, overviewDiagrams, err := c.chapters(ctx, []string{"overview"}, string(outline))
	if err != nil {
		return views, err
	}
	views.Document.Chapters = append(views.Document.Chapters, overview...)
	views.Document.Chapters = append(views.Document.Chapters, chapters...)
	views.Diagrams = append(views.Diagrams, overviewDiagrams...)
	views.Diagrams = append(views.Diagrams, diagrams...)
	if brandContext != "" {
		job := compositionJob{Task: CompositionTask{ID: "card-theme", Title: "Product card appearance", Kind: "theme"}}
		job.Work = func(ctx context.Context, options generationCall) error {
			theme, err := generateWith[compositionTheme](ctx, s, run, "composing", "Choose only the product card theme from the verified branding context. Return card_theme=null if no product identity is supported. Leave sources empty; the server resolves provenance.\n"+brandContext, options, nil)
			if err == nil {
				views.CardTheme = theme.CardTheme
			}
			return err
		}
		if err := c.work(ctx, []compositionJob{job}); err != nil {
			return views, err
		}
	}
	return views, nil
}

func (c *composer) chapters(ctx context.Context, ids []string, overviewContext string) ([]passport.Chapter, []passport.Diagram, error) {
	plans := make([]chapterPlan, len(ids))
	recipes := make([]string, len(ids))
	planJobs := make([]compositionJob, len(ids))
	modelJSON, _ := json.Marshal(c.model)
	for i, id := range ids {
		recipe, err := sectionPrompts.ReadFile("prompts/sections/" + id + ".md")
		if err != nil {
			return nil, nil, fmt.Errorf("read chapter recipe %s: %w", id, err)
		}
		recipes[i] = string(recipe)
		input := chapterPlanPrompt + "\nCHAPTER ID: " + id + "\n" + informationArchitecturePrompt + "\nRECIPE:\n" + string(recipe) + "\nMODEL:\n" + string(modelJSON)
		if overviewContext != "" {
			input += "\nCOMPLETED CHAPTERS (summarize, do not repeat):\n" + overviewContext
		}
		planJobs[i] = compositionJob{Task: CompositionTask{ID: id + "-plan", ChapterID: id, Title: "Plan: " + id, Kind: "plan"}, Work: func(ctx context.Context, options generationCall) error {
			plan, err := generateWith[chapterPlan](ctx, c.service, c.run, "composing", input, options, func(plan chapterPlan) error { return c.checkPlan(id, plan) })
			plans[i] = plan
			return err
		}}
	}
	if err := c.work(ctx, planJobs); err != nil {
		return nil, nil, err
	}
	var jobs []compositionJob
	fragments := make([][]compositionFragment, len(ids))
	for i, plan := range plans {
		fragments[i] = make([]compositionFragment, len(plan.Parts))
		planJSON, _ := json.Marshal(plan)
		for j, part := range plan.Parts {
			id := fmt.Sprintf("%s-part-%d", ids[i], j+1)
			partJSON, _ := json.Marshal(part)
			contextJSON, _ := json.Marshal(compositionContext(c.model, part))
			input := fragmentPrompt + "\n" + compositionPrompt + "\nCHAPTER ID: " + ids[i] + "\nDETAILED CHAPTER REQUIREMENTS:\n" + recipes[i] + "\nCHAPTER PLAN (order and transitions only):\n" + string(planJSON) + "\nASSIGNED PART:\n" + string(partJSON) + "\nMODEL:\n" + string(contextJSON)
			jobs = append(jobs, compositionJob{Task: CompositionTask{ID: id, ChapterID: ids[i], Title: plan.Title + ": " + part.Title, Kind: part.Kind}, Work: func(ctx context.Context, options generationCall) error {
				var fragment compositionFragment
				var err error
				switch part.Kind {
				case scenarioReferenceKind:
					fragment.Block = passport.DocumentBlock{Kind: part.Kind, ReferenceID: part.ScenarioIDs[0]}
				case "object_table":
					fragment.Block = passport.DocumentBlock{Kind: part.Kind, ObjectIDs: part.ObjectIDs, ClaimIDs: part.ClaimIDs, TableStyle: part.TableStyle}
				default:
					fragment, err = generateWith[compositionFragment](ctx, c.service, c.run, "composing", input, options, func(fragment compositionFragment) error {
						return c.checkFragment(ids[i], id, part, authoredFragment(fragment))
					})
					fragment = authoredFragment(fragment)
				}
				if err != nil {
					return err
				}
				if err := c.checkFragment(ids[i], id, part, fragment); err != nil {
					return err
				}
				if fragment.Diagram != nil {
					fragment.Diagram.ID = id + "-diagram"
					fragment.Block.ReferenceID = fragment.Diagram.ID
				}
				fragments[i][j] = fragment
				return nil
			}})
		}
	}
	if err := c.work(ctx, jobs); err != nil {
		return nil, nil, err
	}
	var chapters []passport.Chapter
	var diagrams []passport.Diagram
	for i, plan := range plans {
		if len(plan.Parts) == 0 {
			continue
		}
		chapter := passport.Chapter{ID: ids[i], Title: plan.Title, Question: plan.Question, Blocks: []passport.DocumentBlock{}}
		for _, fragment := range fragments[i] {
			chapter.Blocks = append(chapter.Blocks, fragment.Block)
			if fragment.Diagram != nil {
				diagrams = append(diagrams, *fragment.Diagram)
			}
		}
		chapters = append(chapters, chapter)
	}
	return chapters, diagrams, nil
}

func (c *composer) checkPlan(id string, plan chapterPlan) error {
	if len(plan.Parts) == 0 {
		if slices.Contains(passport.RequiredSectionIDs(), id) {
			return errors.New("required chapter needs at least one part; state precise evidence gaps when necessary")
		}
		return nil
	}
	if strings.TrimSpace(plan.Title) == "" || strings.TrimSpace(plan.Question) == "" {
		return errors.New("chapter title and question are required")
	}
	if plan.Parts[0].Kind != "markdown" {
		return errors.New("begin the chapter with a short markdown answer")
	}
	scenarios := map[string]bool{}
	for _, part := range plan.Parts {
		if err := c.checkPart(id, part); err != nil {
			return err
		}
		if part.Kind == scenarioReferenceKind {
			if scenarios[part.ScenarioIDs[0]] {
				return errors.New("place each full scenario reference only once")
			}
			scenarios[part.ScenarioIDs[0]] = true
		}
	}
	return c.checkCoverage(id, plan)
}

func (c *composer) checkPart(id string, part compositionPart) error {
	if strings.TrimSpace(part.Title) == "" || strings.TrimSpace(part.Instructions) == "" {
		return errors.New("each part needs a title and instructions")
	}
	if !slices.Contains([]string{"markdown", "diagram_ref", scenarioReferenceKind, "object_table"}, part.Kind) {
		return errors.New("unsupported part kind")
	}
	if part.Kind == "object_table" {
		fragment := compositionFragment{Block: passport.DocumentBlock{Kind: part.Kind, ObjectIDs: part.ObjectIDs, ClaimIDs: part.ClaimIDs, TableStyle: part.TableStyle}}
		if err := c.checkFragment(id, "table-plan-check", part, fragment); err != nil {
			return err
		}
	}
	if part.Kind == scenarioReferenceKind && (id != "scenarios" || len(part.ScenarioIDs) != 1) {
		return errors.New("place each scenario exactly once in the scenarios chapter")
	}
	for _, objectID := range part.ObjectIDs {
		if !slices.ContainsFunc(c.model.Objects, func(o passport.Object) bool { return o.ID == objectID }) {
			return fmt.Errorf("unknown object: %s", objectID)
		}
	}
	for _, claimID := range part.ClaimIDs {
		if !slices.ContainsFunc(c.model.Claims, func(claim passport.Claim) bool { return claim.ID == claimID }) {
			return fmt.Errorf("unknown claim: %s", claimID)
		}
	}
	for _, scenarioID := range part.ScenarioIDs {
		if !slices.ContainsFunc(c.model.Scenarios, func(scenario passport.Scenario) bool { return scenario.ID == scenarioID }) {
			return fmt.Errorf("unknown scenario: %s", scenarioID)
		}
	}
	if part.Kind == "diagram_ref" {
		return c.checkDiagramPlan(id, part)
	}
	if part.Diagram != nil {
		return errors.New("only diagram_ref parts may have a diagram plan; otherwise return diagram=null")
	}
	return nil
}

// Validate the planned nodes before spending requests on individual fragments.
func (c *composer) checkDiagramPlan(chapterID string, part compositionPart) error {
	if part.Diagram == nil {
		return errors.New("diagram_ref needs a diagram plan with kind and scope_object_id")
	}
	scope := part.Diagram.ScopeObjectID
	if scope != "" && !slices.Contains(part.ObjectIDs, scope) {
		return fmt.Errorf("diagram plan %q: include scope object %q in object_ids", part.Title, scope)
	}
	nodes := []string{}
	for _, id := range part.ObjectIDs {
		if id != scope {
			nodes = append(nodes, id)
		}
	}
	if len(nodes) == 0 {
		return fmt.Errorf("diagram plan %q needs at least one node besides its scope", part.Title)
	}
	fragment := compositionFragment{
		Block:   passport.DocumentBlock{Kind: "diagram_ref"},
		Diagram: &passport.Diagram{Kind: part.Diagram.Kind, ScopeObjectID: scope, NodeIDs: nodes},
	}
	if err := c.checkFragment(chapterID, "diagram-plan-check", part, fragment); err != nil {
		return fmt.Errorf("diagram plan %q: %w; split incompatible objects into separate views or explain them in markdown; preserve canonical roles and parents", part.Title, err)
	}
	return nil
}

func (c *composer) checkCoverage(id string, plan chapterPlan) error {
	rows, scenarios := map[string]bool{}, map[string]bool{}
	for _, part := range plan.Parts {
		for _, scenarioID := range part.ScenarioIDs {
			scenarios[scenarioID] = true
		}
		if part.Kind != "object_table" || part.TableStyle != id {
			continue
		}
		for _, objectID := range part.ObjectIDs {
			if rows[objectID] {
				return fmt.Errorf("duplicate table row: %s", objectID)
			}
			rows[objectID] = true
		}
	}
	for _, object := range c.model.Objects {
		required := (id == "repositories" && object.Kind == "repository") || (id == "technologies" && strings.TrimSpace(object.Technology) != "")
		if required && !rows[object.ID] {
			return fmt.Errorf("include a %s table row for %s", id, object.ID)
		}
	}
	if id == "scenarios" {
		for _, scenario := range c.model.Scenarios {
			if !scenarios[scenario.ID] {
				return fmt.Errorf("account for scenario %s in a reference or the variant matrix", scenario.ID)
			}
		}
	}
	return nil
}

// authoredFragment keeps only what the model writes for its block kind: prose and
// claims for markdown, the diagram for diagram_ref. Strict output schemas make the
// model fill every block field, so structural values such as a table style can
// appear on prose; the server assigns those from the plan instead.
func authoredFragment(fragment compositionFragment) compositionFragment {
	block := passport.DocumentBlock{Kind: fragment.Block.Kind}
	if block.Kind == "markdown" {
		block.Markdown, block.ClaimIDs = fragment.Block.Markdown, fragment.Block.ClaimIDs
	}
	return compositionFragment{Block: block, Diagram: fragment.Diagram}
}

func (c *composer) checkFragment(chapterID, taskID string, part compositionPart, fragment compositionFragment) error {
	if fragment.Block.Kind != part.Kind {
		return errors.New("return exactly the assigned block kind")
	}
	if len(strings.Fields(fragment.Block.Markdown)) > 250 {
		return errors.New("this part must stay under 250 words; write only its assigned point")
	}
	if part.Kind == "diagram_ref" {
		if err := validatePlannedDiagram(part, fragment.Diagram); err != nil {
			return err
		}
		diagram := *fragment.Diagram
		diagram.ID = taskID + "-diagram"
		fragment.Diagram = &diagram
		fragment.Block.ReferenceID = diagram.ID
	} else if fragment.Diagram != nil {
		return errors.New("only diagram_ref parts may define a diagram")
	}
	if part.Kind == scenarioReferenceKind && fragment.Block.ReferenceID != part.ScenarioIDs[0] {
		return errors.New("return only the assigned scenario reference")
	}
	chapters := []passport.Chapter{{ID: chapterID, Title: part.Title, Question: part.Title, Blocks: []passport.DocumentBlock{fragment.Block}}}
	for _, id := range passport.RequiredSectionIDs() {
		if id != chapterID {
			chapters = append(chapters, passport.Chapter{ID: id, Title: id, Question: id, Blocks: []passport.DocumentBlock{{Kind: "markdown", Markdown: "Validation placeholder."}}})
		}
	}
	views := passport.Views{Sections: []passport.Section{}, Diagrams: []passport.Diagram{}, Document: &passport.Document{Version: 2, Chapters: chapters}}
	if fragment.Diagram != nil {
		views.Diagrams = append(views.Diagrams, *fragment.Diagram)
	}
	if err := passport.Validate(passport.Passport{Model: c.model, Views: views}, c.documents); err != nil {
		return fmt.Errorf("validate assigned part: %w", err)
	}
	return nil
}

func (c *composer) update(ctx context.Context, index, attempt int, status, action string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	task := &c.state.Tasks[index]
	task.Status, task.Attempt = status, attempt
	if status == "running" && task.StartedAt == "" {
		task.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if status == "completed" || status == "failed" || status == "stopped" {
		task.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	c.state.Action = action
	return c.save(ctx)
}

func (c *composer) save(ctx context.Context) error {
	encoded, err := json.Marshal(c.state)
	if err != nil {
		return fmt.Errorf("encode composition progress: %w", err)
	}
	return c.service.event(ctx, c.run.ID, "composition_progress", "composing", string(encoded))
}

// A bounded worker pool avoids filling the provider queue with a whole document.
// Results use their plan indexes, so completion order never changes reading order.
func (c *composer) work(ctx context.Context, jobs []compositionJob) error {
	if len(jobs) == 0 {
		return nil
	}
	offset := len(c.state.Tasks)
	for _, job := range jobs {
		task := job.Task
		task.Status = compositionTaskQueued
		task.MaxAttempts = maxGenerationAttempts
		c.state.Tasks = append(c.state.Tasks, task)
	}
	c.state.Action = fmt.Sprintf("Queued %d passport tasks; up to %d requests at once.", len(jobs), c.state.Parallelism)
	if err := c.save(ctx); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	fail := func(err error) { once.Do(func() { firstErr = err; cancel() }) }
	queue := make(chan int)
	for range min(c.state.Parallelism, len(jobs)) {
		wg.Go(func() {
			for i := range queue {
				if ctx.Err() != nil {
					return
				}
				job := jobs[i]
				if job.Task.Kind == "object_table" || job.Task.Kind == scenarioReferenceKind {
					if err := c.update(ctx, offset+i, 0, "running", job.Task.Title+": assembling from saved facts."); err != nil {
						fail(err)
						return
					}
				}
				options := generationCall{Label: job.Task.Title, MaxOutputTokens: 16000, RetryProvider: true, OnAttempt: func(attempt int) error {
					return c.update(ctx, offset+i, attempt, "running", fmt.Sprintf("%s: waiting for the model (attempt %d of %d).", job.Task.Title, attempt, maxGenerationAttempts))
				}}
				err := job.Work(ctx, options)
				status, action := "completed", job.Task.Title+": completed."
				if err != nil {
					status, action = "failed", job.Task.Title+": "+err.Error()
					if errors.Is(err, context.Canceled) {
						status = "stopped"
					}
				}
				// Preserve the failed task even when a sibling canceled the worker context.
				saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
				c.mu.Lock()
				attempt := c.state.Tasks[offset+i].Attempt
				c.mu.Unlock()
				saveErr := c.update(saveCtx, offset+i, attempt, status, action)
				saveCancel()
				if err != nil {
					fail(fmt.Errorf("%s: %w", job.Task.Title, err))
					return
				}
				if saveErr != nil {
					fail(saveErr)
					return
				}
			}
		})
	}
dispatch:
	for i := range jobs {
		select {
		case queue <- i:
		case <-ctx.Done():
			break dispatch
		}
	}
	close(queue)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("composition interrupted: %w", err)
	}
	return nil
}

// Part requests receive selected facts and their supporting closure, not the
// whole product model or source bodies. Ancestors preserve diagram hierarchy.
func compositionContext(model passport.Model, part compositionPart) passport.Model {
	result := passport.Model{Title: model.Title, Tagline: model.Tagline, Summary: model.Summary, OpenQuestions: model.OpenQuestions, Objects: []passport.Object{}, Relations: []passport.Relation{}, Scenarios: []passport.Scenario{}, Claims: []passport.Claim{}, Evidence: []passport.Evidence{}}
	objects, claims, evidence := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, id := range part.ObjectIDs {
		objects[id] = true
	}
	for _, id := range part.ClaimIDs {
		claims[id] = true
	}
	result.Scenarios = compositionScenarios(model.Scenarios, part.ScenarioIDs, objects, claims)
	byID := map[string]passport.Object{}
	for _, object := range model.Objects {
		byID[object.ID] = object
	}
	for id := range objects {
		seen := map[string]bool{}
		for parent := byID[id].ParentID; parent != "" && !seen[parent]; parent = byID[parent].ParentID {
			objects[parent] = true
			seen[parent] = true
		}
	}
	for _, object := range model.Objects {
		if !objects[object.ID] {
			continue
		}
		result.Objects = append(result.Objects, object)
		for _, id := range object.ClaimIDs {
			claims[id] = true
		}
	}
	for _, relation := range model.Relations {
		if !objects[relation.From] || !objects[relation.To] {
			continue
		}
		result.Relations = append(result.Relations, relation)
		for _, id := range relation.ClaimIDs {
			claims[id] = true
		}
	}
	for _, claim := range model.Claims {
		if !claims[claim.ID] {
			continue
		}
		result.Claims = append(result.Claims, claim)
		for _, id := range claim.EvidenceIDs {
			evidence[id] = true
		}
	}
	for _, item := range model.Evidence {
		if evidence[item.ID] {
			result.Evidence = append(result.Evidence, item)
		}
	}
	return result
}

func compositionScenarios(all []passport.Scenario, selected []string, objects, claims map[string]bool) []passport.Scenario {
	result := []passport.Scenario{}
	for _, scenario := range all {
		if !slices.Contains(selected, scenario.ID) {
			continue
		}
		result = append(result, scenario)
		for _, step := range scenario.Steps {
			for _, id := range step.ObjectIDs {
				objects[id] = true
			}
			for _, id := range step.ClaimIDs {
				claims[id] = true
			}
		}
	}
	return result
}

func validatePlannedDiagram(part compositionPart, diagram *passport.Diagram) error {
	if diagram == nil {
		return errors.New("diagram_ref needs one diagram")
	}
	if part.Diagram == nil {
		return errors.New("diagram_ref needs a validated diagram plan")
	}
	if diagram.Kind != part.Diagram.Kind || diagram.ScopeObjectID != part.Diagram.ScopeObjectID {
		return fmt.Errorf("diagram must keep planned kind %q and scope_object_id %q; got kind %q and scope_object_id %q", part.Diagram.Kind, part.Diagram.ScopeObjectID, diagram.Kind, diagram.ScopeObjectID)
	}
	for _, id := range diagram.NodeIDs {
		if id == part.Diagram.ScopeObjectID || !slices.Contains(part.ObjectIDs, id) {
			return fmt.Errorf("diagram node %q is not a planned node; use object_ids excluding scope_object_id %q", id, part.Diagram.ScopeObjectID)
		}
	}
	return nil
}
