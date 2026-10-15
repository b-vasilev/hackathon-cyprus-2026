package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestSourceToolsPreserveBlankLinesAndPagination(t *testing.T) {
	content := "first\n\n"
	hash := sha256.Sum256([]byte(content))
	doc := source.Document{ID: "doc", SnapshotID: "repo", Path: "file.go", Content: content, ContentHash: hex.EncodeToString(hash[:]), LineCount: 3}
	var recorded []source.LineRange
	tools, err := SourceTools(source.Collection{Snapshots: []source.Snapshot{{ID: "repo"}}, Documents: []source.Document{doc}}, func(_ context.Context, id string, span source.LineRange) error {
		if id != doc.ID {
			t.Fatal("wrong document recorded")
		}
		recorded = append(recorded, span)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	call := func(name, input string, output any) {
		t.Helper()
		for _, tool := range tools {
			if tool.Info().Name != name {
				continue
			}
			response, callErr := tool.Run(t.Context(), fantasy.ToolCall{Name: name, Input: input})
			if callErr != nil || response.IsError {
				t.Fatalf("%s: err=%v response=%s", name, callErr, response.Content)
			}
			if decodeErr := json.Unmarshal([]byte(response.Content), output); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			return
		}
		t.Fatalf("missing tool %s", name)
	}
	var matches analysisPage[sourceMatch]
	call("search_sources", `{"pattern":"^$","offset":1,"limit":1}`, &matches)
	if matches.Total != 2 || len(matches.Items) != 1 || matches.Items[0].Line != 3 || matches.NextOffset != 2 || matches.HasMore || len(recorded) != 0 {
		t.Fatalf("blank line search changed pagination or recorded a read: %+v", matches)
	}
	var read struct {
		Content string `json:"content"`
	}
	call("read_source", `{"document_id":"doc","start_line":2,"end_line":3}`, &read)
	if read.Content != "2: \n3: \n" || len(recorded) != 1 || recorded[0] != (source.LineRange{StartLine: 2, EndLine: 3}) {
		t.Fatalf("blank source lines or recorded range changed: content=%q ranges=%v", read.Content, recorded)
	}
}
