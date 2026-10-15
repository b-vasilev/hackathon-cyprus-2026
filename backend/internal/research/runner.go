package research

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/llmgateway"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

//go:embed prompts/research.md
var prompt string

type Runner struct {
	Policy              llmgateway.NetworkPolicy
	Limiter             *llmgateway.Limiter
	WorkDir, RepomixBin string
}

type Result struct {
	Context                    string
	Collection                 source.Collection
	InputTokens, OutputTokens  int64
	Steps, ToolCalls, MCPReads int
	DurationMS                 int64
}

// Progress reports activity and the usage of each completed model turn.
type Progress struct {
	Message                   string
	Step                      int
	InputTokens, OutputTokens int64
	DurationMS                int64
}

type workspace struct {
	documents           map[string]source.Document
	paths               []string
	mu                  sync.Mutex
	reads               map[string][]source.LineRange
	toolCalls, mcpReads int
	event               func(string) error
}

// Run gives the model access to the full safe source package, with bounded paged tool results.
//
//nolint:gocyclo // Keep workspace setup, model turns and coverage checks in one lifecycle.
func (r *Runner) Run(ctx context.Context, config llmgateway.Config, modelID string, collection source.Collection, progress func(Progress) error) (Result, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	result := Result{}
	event := func(message string) error { return progress(Progress{Message: message}) }
	if err := os.MkdirAll(r.WorkDir, 0o700); err != nil {
		return result, fmt.Errorf("create research workspace: %w", err)
	}
	dir, err := os.MkdirTemp(r.WorkDir, "research-")
	if err != nil {
		return result, fmt.Errorf("create research workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	w := &workspace{documents: map[string]source.Document{}, reads: map[string][]source.LineRange{}, event: event}
	for _, doc := range collection.Documents {
		path := doc.SnapshotID + "/" + doc.Path
		if !filepath.IsLocal(path) {
			return result, errors.New("invalid research source path")
		}
		target := filepath.Join(dir, path)
		if stageErr := os.MkdirAll(filepath.Dir(target), 0o700); stageErr != nil {
			return result, fmt.Errorf("stage source directory: %w", stageErr)
		}
		if stageErr := os.WriteFile(target, []byte(doc.Content), 0o600); stageErr != nil {
			return result, fmt.Errorf("stage source file: %w", stageErr)
		}
		w.documents[path] = doc
		w.paths = append(w.paths, path)
	}
	slices.Sort(w.paths)
	session, err := connectMCP(ctx, r.RepomixBin, dir)
	if err != nil {
		return result, err
	}
	defer session.Close()
	client, err := r.Policy.Client(config, r.Limiter)
	if err != nil {
		return result, fmt.Errorf("research gateway: %w", err)
	}
	client.Timeout = time.Duration(config.TimeoutSeconds) * time.Second
	defer client.CloseIdleConnections()
	model, err := languageModel(ctx, config, modelID, client)
	if err != nil {
		return result, err
	}
	tools := w.tools(session)
	agent := fantasy.NewAgent(model, fantasy.WithSystemPrompt(prompt), fantasy.WithTools(tools...))
	var orientation strings.Builder
	for _, snapshot := range collection.Snapshots {
		fmt.Fprintf(&orientation, "Repository %s at %s; workspace prefix %s/; %d available files\n", snapshot.Locator, snapshot.Revision, snapshot.ID, snapshot.Coverage.FileCount)
	}
	maxOutput := int64(6000)
	reportedSteps := 0
	var stepStarted time.Time
	reportUsage := func(steps []fantasy.StepResult) error {
		for reportedSteps < len(steps) {
			usage := steps[reportedSteps].Usage
			step := reportedSteps + 1
			if checkErr := progress(Progress{
				Message:      fmt.Sprintf("Research model turn %d completed. Recorded usage: %d input tokens and %d output tokens.", step, usage.InputTokens, usage.OutputTokens),
				Step:         step,
				InputTokens:  usage.InputTokens,
				OutputTokens: usage.OutputTokens,
				DurationMS:   time.Since(stepStarted).Milliseconds(),
			}); checkErr != nil {
				return fmt.Errorf("report research usage: %w", checkErr)
			}
			reportedSteps++
		}
		return nil
	}
	response, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: orientation.String() + "Investigate this complete product scope and prepare source-backed findings.", MaxOutputTokens: &maxOutput, ProviderOptions: providerOptions(config, modelID), StopWhen: []fantasy.StopCondition{fantasy.StepCountIs(24)}, PrepareStep: func(stepCtx context.Context, options fantasy.PrepareStepFunctionOptions) (context.Context, fantasy.PrepareStepResult, error) {
		if checkErr := reportUsage(options.Steps); checkErr != nil {
			return stepCtx, fantasy.PrepareStepResult{}, checkErr
		}
		stepStarted = time.Now()
		return stepCtx, fantasy.PrepareStepResult{}, event(fmt.Sprintf("Research model turn %d started.", options.StepNumber+1))
	}})
	if err != nil {
		return result, errors.New("repository research failed through the selected connection; check protocol support, endpoint availability and timeout")
	}
	if checkErr := reportUsage(response.Steps); checkErr != nil {
		return result, checkErr
	}
	notes := response.Response.Content.Text()
	if !strings.Contains(notes, "RESEARCH_COMPLETE") || strings.Contains(notes, "RESEARCH_INCOMPLETE") {
		return result, errors.New("research budget ended before the model completed its coverage checks")
	}
	result.Context, err = w.evidenceContext(notes)
	if err != nil {
		return result, err
	}
	result.Collection = source.Collection{Snapshots: collection.Snapshots, Documents: []source.Document{}}
	for i := range result.Collection.Snapshots {
		snap := &result.Collection.Snapshots[i]
		snap.Coverage.Files = []source.FileCoverage{}
		for _, path := range w.paths {
			doc := w.documents[path]
			if doc.SnapshotID != snap.ID {
				continue
			}
			ranges := w.reads[doc.ID]
			snap.Coverage.Files = append(snap.Coverage.Files, source.FileCoverage{ID: doc.ID, Path: doc.Path, Bytes: len(doc.Content), ReadRanges: ranges})
			if len(ranges) > 0 {
				snap.Coverage.ReadFiles++
				result.Collection.Documents = append(result.Collection.Documents, doc)
			}
		}
	}
	result.InputTokens = response.TotalUsage.InputTokens
	result.OutputTokens = response.TotalUsage.OutputTokens
	result.Steps = len(response.Steps)
	result.ToolCalls = w.toolCalls
	result.MCPReads = w.mcpReads
	result.DurationMS = time.Since(started).Milliseconds()
	return result, nil
}

var citationPattern = regexp.MustCompile(`\[([A-Z0-9]{20,}):L(\d+)-L(\d+)\]`)

func (w *workspace) evidenceContext(notes string) (string, error) {
	var b strings.Builder
	b.WriteString("RESEARCH FINDINGS (verify against the original excerpts below):\n" + notes + "\nORIGINAL SOURCE EXCERPTS:\n")
	byID := map[string]source.Document{}
	for _, doc := range w.documents {
		byID[doc.ID] = doc
	}
	anchors := citationPattern.FindAllStringSubmatch(notes, -1)
	if len(anchors) == 0 {
		return "", errors.New("research produced no original-source evidence anchors")
	}
	seen := map[string]bool{}
	for _, match := range anchors {
		if seen[match[0]] {
			continue
		}
		seen[match[0]] = true
		doc, ok := byID[match[1]]
		start, _ := strconv.Atoi(match[2])
		end, _ := strconv.Atoi(match[3])
		read := false
		for _, span := range w.reads[doc.ID] {
			if start >= span.StartLine && end <= span.EndLine {
				read = true
			}
		}
		if !ok || !read || start < 1 || end < start || end > doc.LineCount {
			return "", errors.New("research cited a source range it did not read")
		}
		fmt.Fprintf(&b, "\n<source_document id=%q path=%q>\n", doc.ID, doc.Path)
		lines := strings.Split(doc.Content, "\n")
		for i := start - 1; i < end; i++ {
			fmt.Fprintf(&b, "%d: %s\n", i+1, lines[i])
		}
		b.WriteString("</source_document>\n")
	}
	return b.String(), nil
}
