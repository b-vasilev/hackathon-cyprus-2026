package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/passport"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

func TestIncrementalToolsSaveDraftThenRequireReadEvidenceToFinalize(t *testing.T) {
	content := "func Run() {}"
	hash := sha256.Sum256([]byte(content))
	input := Input{
		Collection: source.Collection{
			Snapshots: []source.Snapshot{{ID: "repo", Locator: "fixture", Revision: "pinned"}},
			Documents: []source.Document{{ID: "doc", SnapshotID: "repo", Path: "main.go", Content: content, ContentHash: hex.EncodeToString(hash[:]), LineCount: 1}},
		},
		Tasks: []Task{{ID: "purpose", Title: "Product purpose", Question: "Explain purpose", Status: TaskActive}, {ID: ReconciliationTaskID, Title: "Reconciliation", Question: "Finalize verified facts", Status: TaskPending}},
	}
	var durable Checkpoint
	store, err := NewModelStore(input, func(_ context.Context, next Checkpoint) error { durable = next; return nil })
	if err != nil {
		t.Fatal(err)
	}
	var actions []string
	session := &analysisSession{store: store, input: input, current: input.Tasks[0], progress: func(p Progress) { actions = append(actions, p.Message) }}
	tools, err := session.tools()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]fantasy.AgentTool{}
	for _, tool := range tools {
		byName[tool.Info().Name] = tool
	}
	objectSchema := byName["upsert_model_part"].Info().Parameters["objects"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	kinds := objectSchema["kind"].(map[string]any)["enum"].([]any)
	if !slices.Contains(kinds, any("component")) || slices.Contains(kinds, any("software_system")) {
		t.Fatal("tool schema does not distinguish object kinds from architecture roles")
	}
	call := func(name string, args any, wantError bool) fantasy.ToolResponse {
		t.Helper()
		data, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		response, err := byName[name].Run(t.Context(), fantasy.ToolCall{Name: name, Input: string(data)})
		if err != nil || response.IsError != wantError {
			t.Fatalf("%s: err=%v response=%s", name, err, response.Content)
		}
		return response
	}
	object := passport.Object{ID: "product", Kind: "software_system", Name: "Fixture", Status: "implemented", ArchitectureRole: "software_system"}
	part := ModelPart{AreaID: "purpose", Objects: []passport.Object{object}, Metadata: &Metadata{Title: "Fixture", Summary: "Runs the fixture flow"}}
	call("upsert_model_part", part, true)
	if !strings.Contains(session.lastToolFailure, "kind") || !strings.Contains(actions[len(actions)-1], "software_system") {
		t.Fatal("the rejected field was hidden from the next window and progress log")
	}
	object.Kind = "component"
	part.Objects = []passport.Object{object}
	call("upsert_model_part", part, false)
	if len(durable.Model.Objects) != 1 || session.lastToolFailure != "" || !slices.ContainsFunc(store.Issues(), func(i Issue) bool { return i.Code == "missing_claims" }) {
		t.Fatal("initial object was not saved as a durable, explicitly incomplete draft")
	}
	claim := passport.Claim{ID: "purpose", Text: "The fixture has a Run entrypoint", Basis: "source_backed"}
	call("upsert_model_part", ModelPart{AreaID: "purpose", Claims: []passport.Claim{claim}}, false)
	if !slices.ContainsFunc(store.Issues(), func(i Issue) bool { return i.Code == "missing_evidence" }) {
		t.Fatal("unfinished source backing disappeared from validation issues")
	}
	evidence := ModelPart{AreaID: "purpose", Evidence: []passport.Evidence{{ID: "proof", DocumentID: "doc", StartLine: 1, EndLine: 1}}}
	call("upsert_model_part", evidence, true)
	if len(durable.Model.Objects) != 1 || len(durable.Model.Claims) != 1 || len(durable.Model.Evidence) != 0 {
		t.Fatal("unread citation was accepted or erased earlier saved facts")
	}
	call("read_source", sourceReadInput{DocumentID: "doc", StartLine: 1, EndLine: 1}, false)
	call("upsert_model_part", evidence, false)
	object.ClaimIDs = []string{claim.ID}
	claim.EvidenceIDs = []string{"proof"}
	call("upsert_model_part", ModelPart{AreaID: "purpose", Correction: "Attach the verified source evidence", Objects: []passport.Object{object}, Claims: []passport.Claim{claim}}, false)
	call("finish_analysis_area", finishAreaInput{AreaID: "purpose"}, false)
	if err := store.SetTaskStatus(t.Context(), ReconciliationTaskID, TaskActive); err != nil {
		t.Fatal(err)
	}
	session.current = input.Tasks[1]
	call("finalize_model", finalizeInput{AreaID: ReconciliationTaskID}, false)
	if !durable.Finalized || len(store.Issues()) != 0 {
		t.Fatal("the incrementally saved source-backed model did not finalize")
	}
}
