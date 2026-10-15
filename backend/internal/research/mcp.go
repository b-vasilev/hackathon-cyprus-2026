package research

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectMCP(ctx context.Context, binary, workspace string) (*mcp.ClientSession, error) {
	//nolint:gosec // The executable is operator-configured. Repository code is never executed.
	command := exec.CommandContext(ctx, binary, "--mcp", "--sandbox", workspace)
	command.Dir = workspace
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.Contains(name, "KEY") || strings.Contains(name, "TOKEN") || strings.Contains(name, "SECRET") || strings.Contains(name, "PASSWORD") || name == "TMPDIR" {
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "TMPDIR="+workspace)
	client := mcp.NewClient(&mcp.Implementation{Name: "passport-research", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		return nil, &sourceReaderError{message: "source reader is unavailable", cause: err}
	}
	return session, nil
}

func callMCP(ctx context.Context, session *mcp.ClientSession, name string, args, result any) error {
	response, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return &sourceReaderError{message: "source file could not be read", cause: err}
	}
	if response.IsError {
		return errors.New("source file could not be read; it may be excluded by the source security policy")
	}
	for _, block := range response.Content {
		if value, ok := block.(*mcp.TextContent); ok {
			if err := json.Unmarshal([]byte(value.Text), result); err != nil {
				return &sourceReaderError{message: "source reader returned an invalid result", cause: err}
			}
			return nil
		}
	}
	return errors.New("source reader returned no text")
}

// Keep the infrastructure cause available for diagnostics without exposing
// executable names, local paths or transport details in user-facing errors.
type sourceReaderError struct {
	message string
	cause   error
}

func (e *sourceReaderError) Error() string { return e.message }
func (e *sourceReaderError) Unwrap() error { return e.cause }
