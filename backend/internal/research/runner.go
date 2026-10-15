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
	// MaxTurns is the hard cap for one attempt; zero uses 120 turns.
	MaxTurns int
}

type Result struct {
	Context                    string
	Plan                       []researchBoundary
	Ledger                     Ledger
	Notes                      string
	Collection                 source.Collection
	InputTokens, OutputTokens  int64
	Steps, ToolCalls, MCPReads int
	DurationMS                 int64
	Progress                   *State
}

// Progress reports activity and the usage of each completed model turn.
type Progress struct {
	Message                   string
	Step                      int
	InputTokens, OutputTokens int64
	DurationMS                int64
	State                     *State
}

type workspace struct {
	documents           map[string]source.Document
	paths               []string
	mu                  sync.Mutex
	reads               map[string][]source.LineRange
	toolCalls, mcpReads int
	event               func(string) error
	plan                []researchBoundary
	checkedNotes        string
	submittedNotes      string
	inventory           Inventory
	ledger              Ledger
	searches            []searchedScope
	summaryNotes        string
	turn, turnBudget    int
	deadline            time.Time
	readFailures        map[string]string
	budget              *attemptBudget
	activity            string
}

// run gives the model access to pinned source files with bounded paged tools.
//
//nolint:gocyclo // Keep workspace, checkpoint and adaptive research lifecycle cleanup together.
func (r *Runner) run(ctx context.Context, config llmgateway.Config, modelID string, collection source.Collection, checkpoint *Checkpoint, progress func(Progress) error) (result Result, err error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var w *workspace
	event := func(message string) error {
		if w != nil && strings.HasPrefix(message, "Investigating ") {
			return w.publish(progress, message)
		}
		return progress(Progress{Message: message})
	}
	if setupErr := os.MkdirAll(r.WorkDir, 0o700); setupErr != nil {
		return result, fmt.Errorf("create research workspace: %w", setupErr)
	}
	dir, err := os.MkdirTemp(r.WorkDir, "research-")
	if err != nil {
		return result, fmt.Errorf("create research workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	w = &workspace{documents: map[string]source.Document{}, reads: map[string][]source.LineRange{}, readFailures: map[string]string{}, event: event, turnBudget: r.MaxTurns}
	if w.turnBudget <= 0 {
		w.turnBudget = 120
	}
	w.deadline, _ = ctx.Deadline()
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
	w.inventory = buildInventory(collection)
	w.ledger = Ledger{Inventory: w.inventory, Status: "in_progress"}
	if checkpoint != nil {
		if restoreErr := w.restore(*checkpoint); restoreErr != nil {
			return result, restoreErr
		}
	}
	defer func() {
		result.Ledger = w.ledger
		result.Plan = w.plan
		result.Notes = w.submittedNotes
		result.ToolCalls, result.MCPReads = w.toolCalls, w.mcpReads
		result.DurationMS = time.Since(started).Milliseconds()
		result.Collection = w.readCollection(collection)
		if err != nil {
			result.Ledger.Status = researchPartial
			result.Ledger.Frontier = w.unfinishedFrontier()
			result.Ledger.Frontier = append(result.Ledger.Frontier, "Research stopped before an accepted coverage check: "+err.Error())
			w.ledger = result.Ledger
			if w.budget != nil {
				w.budget.stopped = true
				w.budget.decisions = append(w.budget.decisions, BudgetDecision{Turn: w.turn, Limit: w.activeLimit(), Kind: "stopped", Reason: "Research stopped before coverage was accepted: " + err.Error()})
				// The service callback uses the parent context, so it can save the
				// final state even when this attempt's time budget has expired.
				_ = w.publish(progress, "Research stopped; findings and open questions are retained.")
			}
		}
		state := w.state()
		result.Progress = &state
	}()
	orientation := w.orientation(collection)
	if w.checkedNotes != "" && w.ledger.Status == researchComplete {
		w.budget = &attemptBudget{stopped: true, decisions: []BudgetDecision{{Kind: researchComplete, Reason: "Reused completed research at the same source revisions. No new model turns were needed."}}}
		result.Context, err = w.resultContext(orientation)
		return result, err
	}
	w.ledger.Focus = Focus{}
	w.startBudget()
	if publishErr := w.publish(progress, "Research plan opened. Source areas will be refined into product responsibilities."); publishErr != nil {
		return result, publishErr
	}
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
	tools := append(w.tools(session), w.coverageTools()...)
	agent := fantasy.NewAgent(model, fantasy.WithSystemPrompt(prompt), fantasy.WithTools(tools...))
	if loopErr := w.investigate(ctx, agent, config, modelID, orientation, progress, &result); loopErr != nil {
		return result, loopErr
	}
	result.Context, err = w.resultContext(orientation)
	return result, err
}

func (w *workspace) orientation(collection source.Collection) string {
	var b strings.Builder
	for _, snapshot := range collection.Snapshots {
		fmt.Fprintf(&b, "Repository %s at %s; workspace prefix %s/; %d available files\n", snapshot.Locator, snapshot.Revision, snapshot.ID, snapshot.Coverage.FileCount)
	}
	fmt.Fprintf(&b, "Independent inventory: %d candidates. Review list_research_inventory and next_research_questions. This candidate map does not prove completeness.\n", len(w.inventory.Candidates))
	return b.String()
}

func (w *workspace) resultContext(orientation string) (string, error) {
	contextText, err := w.evidenceContext(w.checkedNotes)
	if err != nil {
		return "", err
	}
	ledgerText, err := w.ledgerContext()
	if err != nil {
		return "", err
	}
	return "REPOSITORY CATALOG:\n" + orientation + contextText + ledgerText, nil
}

func (w *workspace) readCollection(collection source.Collection) source.Collection {
	result := source.Collection{Snapshots: slices.Clone(collection.Snapshots), Documents: []source.Document{}}
	for i := range result.Snapshots {
		snap := &result.Snapshots[i]
		snap.Coverage.Files = []source.FileCoverage{}
		snap.Coverage.ReadFiles, snap.Coverage.CitedFiles = 0, 0
		for _, path := range w.paths {
			doc := w.documents[path]
			if doc.SnapshotID != snap.ID {
				continue
			}
			ranges := slices.Clone(w.reads[doc.ID])
			snap.Coverage.Files = append(snap.Coverage.Files, source.FileCoverage{ID: doc.ID, Path: doc.Path, Bytes: len(doc.Content), ReadRanges: ranges})
			if len(ranges) > 0 {
				snap.Coverage.ReadFiles++
				result.Documents = append(result.Documents, doc)
			}
		}
	}
	return result
}

var citationPattern = regexp.MustCompile(`\[([A-Z0-9]{20,}):L(\d+)-L(\d+)\]`)

func mergeRanges(ranges []source.LineRange) []source.LineRange {
	ordered := slices.Clone(ranges)
	slices.SortFunc(ordered, func(a, b source.LineRange) int {
		if a.StartLine < b.StartLine {
			return -1
		}
		if a.StartLine > b.StartLine {
			return 1
		}
		return a.EndLine - b.EndLine
	})
	var merged []source.LineRange
	for _, span := range ordered {
		if len(merged) == 0 || span.StartLine > merged[len(merged)-1].EndLine+1 {
			merged = append(merged, span)
			continue
		}
		merged[len(merged)-1].EndLine = max(merged[len(merged)-1].EndLine, span.EndLine)
	}
	return merged
}

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
	ranges := map[string][]source.LineRange{}
	for _, match := range anchors {
		doc, ok := byID[match[1]]
		start, _ := strconv.Atoi(match[2])
		end, _ := strconv.Atoi(match[3])
		if !ok || !RangeCovered(w.reads[doc.ID], start, end) || end > doc.LineCount {
			return "", errors.New("research cited a source range it did not read")
		}
		ranges[doc.ID] = append(ranges[doc.ID], source.LineRange{StartLine: start, EndLine: end})
	}
	ids := make([]string, 0, len(ranges))
	for id := range ranges {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		doc := byID[id]
		lines := strings.Split(doc.Content, "\n")
		for _, span := range mergeRanges(ranges[id]) {
			fmt.Fprintf(&b, "\n<source_document id=%q snapshot_id=%q path=%q start_line=\"%d\" end_line=\"%d\">\n", doc.ID, doc.SnapshotID, doc.Path, span.StartLine, span.EndLine)
			for i := span.StartLine - 1; i < span.EndLine; i++ {
				fmt.Fprintf(&b, "%d: %s\n", i+1, lines[i])
			}
			b.WriteString("</source_document>\n")
		}
	}
	return b.String(), nil
}
