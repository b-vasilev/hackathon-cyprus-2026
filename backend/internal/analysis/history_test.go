package analysis

import (
	"testing"

	"charm.land/fantasy"
)

func call(id string) fantasy.ToolCallPart {
	return fantasy.ToolCallPart{ToolCallID: id, ToolName: "read_finding", Input: "{}"}
}

func result(id string) fantasy.ToolResultPart {
	return fantasy.ToolResultPart{ToolCallID: id, Output: fantasy.ToolResultOutputContentText{Text: "ok"}}
}

func toolIDs(messages []fantasy.Message) []string {
	var ids []string
	for _, message := range messages {
		for _, part := range message.Content {
			if c, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part); ok {
				ids = append(ids, "call:"+c.ToolCallID)
			}
			if r, ok := fantasy.AsMessagePart[fantasy.ToolResultPart](part); ok {
				ids = append(ids, "result:"+r.ToolCallID)
			}
		}
	}
	return ids
}

func TestPairedToolTurnsDropsUnpairedCallsAndResults(t *testing.T) {
	messages := []fantasy.Message{
		// A window trimmed between a call and its result starts with an orphan result.
		{Role: fantasy.MessageRoleTool, Content: []fantasy.MessagePart{result("trimmed")}},
		fantasy.NewUserMessage("continue"),
		{Role: fantasy.MessageRoleAssistant, Content: []fantasy.MessagePart{call("a"), call("b")}},
		{Role: fantasy.MessageRoleTool, Content: []fantasy.MessagePart{result("a")}},
		// A window stopped right after the model answered leaves a call without a result.
		{Role: fantasy.MessageRoleAssistant, Content: []fantasy.MessagePart{fantasy.TextPart{Text: "Reading."}, call("dangling")}},
		fantasy.NewUserMessage("next window"),
	}
	got := pairedToolTurns(messages)
	ids := toolIDs(got)
	want := []string{"call:a", "result:a"}
	if len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] {
		t.Fatalf("tool parts = %v, want %v", ids, want)
	}
	// Text is kept even when its message loses a dangling call; user messages stay.
	if len(got) != 5 || got[0].Role != fantasy.MessageRoleUser || got[3].Role != fantasy.MessageRoleAssistant || len(got[3].Content) != 1 {
		t.Fatalf("unexpected messages: %+v", got)
	}
}

func TestPairedToolTurnsKeepsCompleteHistory(t *testing.T) {
	messages := []fantasy.Message{
		fantasy.NewUserMessage("start"),
		{Role: fantasy.MessageRoleAssistant, Content: []fantasy.MessagePart{call("a")}},
		{Role: fantasy.MessageRoleTool, Content: []fantasy.MessagePart{result("a")}},
	}
	if got := pairedToolTurns(messages); len(got) != 3 || len(toolIDs(got)) != 2 {
		t.Fatalf("complete history changed: %+v", got)
	}
}
