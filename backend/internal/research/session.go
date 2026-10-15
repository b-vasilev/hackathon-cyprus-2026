package research

import (
	"encoding/json"
	"slices"

	"charm.land/fantasy"
)

func (s taskSession) messages() []fantasy.Message {
	var messages []fantasy.Message
	for _, turn := range s.Turns {
		messages = append(messages, turn...)
	}
	return messages
}

func (w *workspace) rememberTurn(messages []fantasy.Message) {
	w.conversation.Turns = append(slices.Clone(w.conversation.Turns), slices.Clone(messages))
	// Keep complete assistant/tool turns, never split a tool call from its reply.
	for len(w.conversation.Turns) > 1 {
		data, err := json.Marshal(w.conversation)
		if err == nil && len(data) <= 256<<10 && len(w.conversation.Turns) <= 8 {
			break
		}
		w.conversation.Turns = w.conversation.Turns[1:]
	}
}
