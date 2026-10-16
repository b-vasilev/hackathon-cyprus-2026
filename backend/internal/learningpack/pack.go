// Package learningpack turns a passport version into product quizzes and a first-task mission.
package learningpack

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Question matches the Foundations choice question so the portal reuses one quiz component.
type Question struct {
	ID          string   `json:"id" jsonschema:"Short unique kebab-case ID."`
	ChapterID   string   `json:"chapter_id" jsonschema:"ID of the passport chapter that teaches the answer, or empty."`
	Kind        string   `json:"kind" jsonschema:"single (exactly one correct option) or multiple (two or more correct options)."`
	Prompt      string   `json:"prompt" jsonschema:"The question, answerable from the passport alone."`
	Options     []string `json:"options" jsonschema:"Three to five distinct, plausible options."`
	Correct     []int    `json:"correct" jsonschema:"Zero-based indexes of the correct options."`
	Explanation string   `json:"explanation" jsonschema:"Why the answer is right, naming the passport fact it comes from."`
}

// Step is one action in the first-task mission.
type Step struct {
	ID        string   `json:"id" jsonschema:"Short unique kebab-case ID."`
	Title     string   `json:"title" jsonschema:"Imperative step title, for example Clone the repository."`
	Body      string   `json:"body" jsonschema:"Markdown instructions for this step: what to do, what to look at and why."`
	ChapterID string   `json:"chapter_id" jsonschema:"ID of the passport chapter to read for this step, or empty."`
	Commands  []string `json:"commands" jsonschema:"Shell commands to run, taken from the passport; empty when the step has none."`
	DoneWhen  string   `json:"done_when" jsonschema:"What the learner sees when the step is complete."`
}

// Issue is a real closed issue and the merged pull request that resolved it.
type Issue struct {
	Repository string   `json:"repository"`
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	PullNumber int      `json:"pull_number"`
	PullTitle  string   `json:"pull_title"`
	PullURL    string   `json:"pull_url"`
	Files      []string `json:"files"`
	Why        string   `json:"why"`
}

// Mission guides a newcomer from cloning the code to understanding a real change.
type Mission struct {
	Title  string     `json:"title"`
	Intro  string     `json:"intro"`
	Issue  *Issue     `json:"issue,omitempty"`
	Steps  []Step     `json:"steps"`
	Check  []Question `json:"check"`
	Finale string     `json:"finale"`
}

// Pack is what reviewers check and learners take.
type Pack struct {
	Quiz    []Question `json:"quiz"`
	Mission Mission    `json:"mission"`
}

// generated is the model's answer; the server fills in issue details from GitHub.
type generated struct {
	Quiz    []Question       `json:"quiz" jsonschema:"Six to eight questions about the product, spread across the passport chapters."`
	Mission generatedMission `json:"mission"`
}

type generatedMission struct {
	Title       string     `json:"title" jsonschema:"Short mission title."`
	Intro       string     `json:"intro" jsonschema:"Two or three sentences: what the learner will do and what they will be able to do after."`
	IssueNumber int        `json:"issue_number" jsonschema:"Number of the chosen candidate issue, or 0 when no candidates were given."`
	IssueWhy    string     `json:"issue_why" jsonschema:"One or two sentences on why this issue is a good first study, or empty when there is no issue."`
	Steps       []Step     `json:"steps" jsonschema:"Five to nine steps in order."`
	Check       []Question `json:"check" jsonschema:"Two or three questions the learner answers after studying the issue and its pull request."`
	Finale      string     `json:"finale" jsonschema:"Closing message that ends with: You are ready to go!"`
}

func validate(pack generated, chapters []string, candidates []Issue) error {
	ids := map[string]bool{}
	problems := quizProblems(pack.Quiz, chapters, ids)
	problems = append(problems, missionProblems(pack.Mission, chapters, candidates, ids)...)
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// uniqueID records an ID shared by questions and steps and reports a missing or repeated one.
func uniqueID(ids map[string]bool, where, id string) []string {
	if id == "" || ids[id] {
		return []string{where + ": id must be unique and non-empty"}
	}
	ids[id] = true
	return nil
}

func quizProblems(quiz []Question, chapters []string, ids map[string]bool) []string {
	var problems []string
	if n := len(quiz); n < 6 || n > 8 {
		problems = append(problems, fmt.Sprintf("quiz has %d questions; return six to eight", n))
	}
	for _, q := range quiz {
		problems = append(problems, uniqueID(ids, "quiz question", q.ID)...)
		problems = append(problems, questionProblems("quiz question "+q.ID, q, chapters)...)
	}
	return problems
}

func missionProblems(m generatedMission, chapters []string, candidates []Issue, ids map[string]bool) []string {
	var problems []string
	if strings.TrimSpace(m.Title) == "" || strings.TrimSpace(m.Intro) == "" {
		problems = append(problems, "mission needs a title and an intro")
	}
	if !strings.Contains(m.Finale, "You are ready to go!") {
		problems = append(problems, "mission finale must end with: You are ready to go!")
	}
	if n := len(m.Steps); n < 5 || n > 9 {
		problems = append(problems, fmt.Sprintf("mission has %d steps; return five to nine", n))
	}
	for _, step := range m.Steps {
		problems = append(problems, uniqueID(ids, "mission step", step.ID)...)
		problems = append(problems, stepProblems(step, chapters)...)
	}
	if n := len(m.Check); n < 2 || n > 3 {
		problems = append(problems, fmt.Sprintf("mission check has %d questions; return two or three", n))
	}
	for _, q := range m.Check {
		problems = append(problems, uniqueID(ids, "mission check question", q.ID)...)
		problems = append(problems, questionProblems("mission check question "+q.ID, q, chapters)...)
	}
	switch {
	case len(candidates) == 0 && m.IssueNumber != 0:
		problems = append(problems, "no candidate issues were given; issue_number must be 0")
	case len(candidates) > 0 && !slices.ContainsFunc(candidates, func(i Issue) bool { return i.Number == m.IssueNumber }):
		problems = append(problems, fmt.Sprintf("issue_number %d is not one of the candidate issues", m.IssueNumber))
	}
	return problems
}

func stepProblems(step Step, chapters []string) []string {
	var problems []string
	if strings.TrimSpace(step.Title) == "" || strings.TrimSpace(step.Body) == "" || strings.TrimSpace(step.DoneWhen) == "" {
		problems = append(problems, "mission step "+step.ID+": title, body and done_when are required")
	}
	if step.ChapterID != "" && !slices.Contains(chapters, step.ChapterID) {
		problems = append(problems, "mission step "+step.ID+": unknown chapter_id "+step.ChapterID)
	}
	return problems
}

func questionProblems(where string, q Question, chapters []string) []string {
	var problems []string
	if strings.TrimSpace(q.Prompt) == "" || strings.TrimSpace(q.Explanation) == "" {
		problems = append(problems, where+": prompt and explanation are required")
	}
	if q.ChapterID != "" && !slices.Contains(chapters, q.ChapterID) {
		problems = append(problems, where+": unknown chapter_id "+q.ChapterID)
	}
	if n := len(q.Options); n < 3 || n > 5 {
		problems = append(problems, where+": give three to five options")
	}
	seen := map[string]bool{}
	for _, option := range q.Options {
		key := strings.ToLower(strings.TrimSpace(option))
		if key == "" || seen[key] {
			problems = append(problems, where+": options must be distinct and non-empty")
			break
		}
		seen[key] = true
	}
	correct := slices.Clone(q.Correct)
	slices.Sort(correct)
	if len(slices.Compact(correct)) != len(q.Correct) || slices.ContainsFunc(q.Correct, func(i int) bool { return i < 0 || i >= len(q.Options) }) {
		problems = append(problems, where+": correct must list distinct valid option indexes")
	}
	switch {
	case q.Kind == "single" && len(q.Correct) != 1:
		problems = append(problems, where+": a single question has exactly one correct option")
	case q.Kind == "multiple" && (len(q.Correct) < 2 || len(q.Correct) == len(q.Options)):
		problems = append(problems, where+": a multiple question has at least two correct options and at least one wrong one")
	case q.Kind != "single" && q.Kind != "multiple":
		problems = append(problems, where+": kind must be single or multiple")
	}
	return problems
}

// pack attaches the chosen issue's real GitHub details to the model's answer.
func (g generated) pack(candidates []Issue) Pack {
	m := g.Mission
	mission := Mission{Title: m.Title, Intro: m.Intro, Steps: m.Steps, Check: m.Check, Finale: m.Finale}
	for _, candidate := range candidates {
		if candidate.Number == m.IssueNumber {
			issue := candidate
			issue.Why = m.IssueWhy
			mission.Issue = &issue
		}
	}
	return Pack{Quiz: g.Quiz, Mission: mission}
}
