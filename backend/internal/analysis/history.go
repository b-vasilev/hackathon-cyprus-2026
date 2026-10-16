package analysis

import "charm.land/fantasy"

// pairedToolTurns keeps only complete tool turns. Providers such as Anthropic reject a
// request when a tool call is not followed by its result in the next message, or when a
// result has no call before it. Trimming a carried window or stopping a window right after
// a model response can leave either half behind, so unpaired parts are dropped.
func pairedToolTurns(messages []fantasy.Message) []fantasy.Message {
	result := make([]fantasy.Message, 0, len(messages))
	for index, message := range messages {
		switch message.Role {
		case fantasy.MessageRoleAssistant:
			answered := map[string]bool{}
			if index+1 < len(messages) && messages[index+1].Role == fantasy.MessageRoleTool {
				for _, part := range messages[index+1].Content {
					if toolResult, ok := fantasy.AsMessagePart[fantasy.ToolResultPart](part); ok {
						answered[toolResult.ToolCallID] = true
					}
				}
			}
			kept := keepParts(message, func(part fantasy.MessagePart) bool {
				call, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part)
				return !ok || call.ProviderExecuted || answered[call.ToolCallID]
			})
			if len(kept.Content) > 0 {
				result = append(result, kept)
			}
		case fantasy.MessageRoleTool:
			called := map[string]bool{}
			if index > 0 && messages[index-1].Role == fantasy.MessageRoleAssistant {
				for _, part := range messages[index-1].Content {
					if call, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part); ok {
						called[call.ToolCallID] = true
					}
				}
			}
			kept := keepParts(message, func(part fantasy.MessagePart) bool {
				toolResult, ok := fantasy.AsMessagePart[fantasy.ToolResultPart](part)
				return !ok || called[toolResult.ToolCallID]
			})
			if len(kept.Content) > 0 {
				result = append(result, kept)
			}
		default:
			result = append(result, message)
		}
	}
	return result
}

func keepParts(message fantasy.Message, keep func(fantasy.MessagePart) bool) fantasy.Message {
	parts := make([]fantasy.MessagePart, 0, len(message.Content))
	for _, part := range message.Content {
		if keep(part) {
			parts = append(parts, part)
		}
	}
	message.Content = parts
	return message
}
