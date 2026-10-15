package applog

import (
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"
)

// New creates a structured logger without changing global state.
func New(output io.Writer, level, format string) (zerolog.Logger, error) {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("parse log level: %w", err)
	}
	if format == "console" {
		output = zerolog.ConsoleWriter{Out: output, TimeFormat: time.RFC3339}
	} else if format != "json" {
		return zerolog.Logger{}, fmt.Errorf("unsupported log format %q", format)
	}
	logger := zerolog.New(output).Level(parsed).With().Timestamp().Str("service", "onboarding-factory").Logger()
	return logger, nil
}
