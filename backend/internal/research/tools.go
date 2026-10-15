package research

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listInput struct {
	Pattern string `json:"pattern"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
}

const toolErrorKey = "error"

type searchQuery struct {
	Pattern string `json:"pattern"`
	Glob    string `json:"glob"`
	Offset  int    `json:"offset"`
	Limit   int    `json:"limit"`
}
type searchInput struct {
	Queries []searchQuery `json:"queries"`
}
type readSpec struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}
type readInput struct {
	Reads []readSpec `json:"reads"`
}

func toolJSON(value any) (fantasy.ToolResponse, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("encode research result: %w", err)
	}
	return fantasy.NewTextResponse(string(data)), nil
}

func (w *workspace) tick(message string) error {
	w.mu.Lock()
	w.toolCalls++
	w.activity = message
	w.execution = "working"
	w.mu.Unlock()
	return w.event(message)
}

func (w *workspace) tools(session *mcp.ClientSession) []fantasy.AgentTool {
	return []fantasy.AgentTool{
		fantasy.NewParallelAgentTool("list_files", "List all indexed source paths matching a glob. Use **/README.md, **/go.mod or a responsibility glob. Results are paged; offset=0 and limit=100 are good defaults.", func(_ context.Context, input listInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if err := w.tick("Listing repository files: " + input.Pattern); err != nil {
				return fantasy.ToolResponse{}, err
			}
			pattern := input.Pattern
			if pattern == "" {
				pattern = "**"
			}
			matches := []string{}
			for _, path := range w.paths {
				ok, matchErr := doublestar.Match(pattern, path)
				if matchErr != nil {
					return toolJSON(map[string]string{toolErrorKey: "Invalid glob pattern: " + matchErr.Error()})
				}
				if ok {
					matches = append(matches, path)
				}
			}
			start := min(max(input.Offset, 0), len(matches))
			end := min(start+min(max(input.Limit, 1), 150), len(matches))
			return toolJSON(map[string]any{"paths": matches[start:end], "total": len(matches), "next_offset": end, "has_more": end < len(matches)})
		}),
		fantasy.NewParallelAgentTool("search_code", "Search all indexed original source text with regular expressions and optional file glob. Batch independent searches. Returns file paths and original line numbers; use read_files to inspect supporting code. Results are paged.", func(_ context.Context, input searchInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if len(input.Queries) > 12 {
				return fantasy.NewTextResponse("Use at most 12 queries per batch"), nil
			}
			if err := w.tick(fmt.Sprintf("Searching repository code (%d queries).", len(input.Queries))); err != nil {
				return fantasy.ToolResponse{}, err
			}
			results := []any{}
			for _, query := range input.Queries {
				results = append(results, w.search(query))
			}
			return toolJSON(results)
		}),
		fantasy.NewParallelAgentTool("read_files", "Read original source ranges from the verified source workspace. Batch independent file ranges. Use 1-based original lines; at most 250 lines per range. Another page can always be requested. Returns exact document IDs for evidence citations.", func(ctx context.Context, input readInput, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if len(input.Reads) > 24 {
				return fantasy.NewTextResponse("Use at most 24 file ranges per batch"), nil
			}
			if err := w.tick(fmt.Sprintf("Reading source code (%d ranges).", len(input.Reads))); err != nil {
				return fantasy.ToolResponse{}, err
			}
			results := []any{}
			for _, spec := range input.Reads {
				results = append(results, w.read(ctx, session, spec))
			}
			return toolJSON(results)
		}),
	}
}

func (w *workspace) search(query searchQuery) any {
	if len(query.Pattern) > 400 {
		return map[string]string{toolErrorKey: "Search pattern exceeds 400 characters"}
	}
	re, err := regexp.Compile(query.Pattern)
	if err != nil {
		return map[string]string{toolErrorKey: "Invalid regular expression"}
	}
	if query.Glob != "" {
		if _, err := doublestar.Match(query.Glob, ""); err != nil {
			return map[string]string{toolErrorKey: "Invalid glob pattern"}
		}
	}
	type match struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	}
	matches := []match{}
	total := 0
	offset := max(query.Offset, 0)
	limit := min(max(query.Limit, 1), 60)
	for _, path := range w.paths {
		if query.Glob != "" {
			ok, err := doublestar.Match(query.Glob, path)
			if err != nil || !ok {
				continue
			}
		}
		for index, line := range strings.Split(w.documents[path].Content, "\n") {
			if !re.MatchString(line) {
				continue
			}
			total++
			if total > offset && len(matches) < limit {
				if len(line) > 320 {
					line = line[:320] + "…"
				}
				matches = append(matches, match{path, index + 1, line})
			}
		}
	}
	w.mu.Lock()
	scope := searchedScope{Pattern: query.Pattern, Glob: query.Glob}
	if !slices.Contains(w.searches, scope) {
		w.searches = append(w.searches, scope)
		w.ledger.Searches = append(w.ledger.Searches, searchRecord{Pattern: query.Pattern, Glob: query.Glob, Matches: total})
	}
	w.mu.Unlock()
	return map[string]any{"pattern": query.Pattern, "glob": query.Glob, "matches": matches, "total": total, "next_offset": offset + len(matches), "has_more": offset+len(matches) < total}
}

func (w *workspace) read(ctx context.Context, session *mcp.ClientSession, spec readSpec) any {
	doc, ok := w.documents[spec.Path]
	if !ok {
		return map[string]string{"path": spec.Path, toolErrorKey: "File is not in the verified source workspace"}
	}
	var view struct {
		Content string `json:"content"`
	}
	if err := callMCP(ctx, session, "file_system_read_file", map[string]any{"path": spec.Path}, &view); err != nil {
		w.mu.Lock()
		if w.readFailures == nil {
			w.readFailures = map[string]string{}
		}
		w.readFailures[spec.Path] = err.Error()
		w.mu.Unlock()
		return map[string]string{"path": spec.Path, toolErrorKey: err.Error()}
	}
	if view.Content != doc.Content {
		return map[string]string{"path": spec.Path, toolErrorKey: "Source bytes did not match the pinned snapshot"}
	}
	lines := strings.Split(view.Content, "\n")
	start := max(spec.StartLine, 1)
	if start > len(lines) {
		return map[string]any{"path": spec.Path, "total_lines": len(lines), toolErrorKey: "Start line is beyond the file"}
	}
	end := spec.EndLine
	if end < start {
		end = start + 119
	}
	end = min(end, start+249, len(lines))
	var text strings.Builder
	for i := start - 1; i < end; i++ {
		if text.Len() > 16000 {
			end = i
			break
		}
		fmt.Fprintf(&text, "%d: %s\n", i+1, lines[i])
	}
	w.mu.Lock()
	delete(w.readFailures, spec.Path)
	w.mcpReads++
	w.reads[doc.ID] = append(w.reads[doc.ID], source.LineRange{StartLine: start, EndLine: end})
	w.mu.Unlock()
	return map[string]any{"document_id": doc.ID, "path": spec.Path, "start_line": start, "end_line": end, "total_lines": len(lines), "has_more": end < len(lines), "content": text.String()}
}
