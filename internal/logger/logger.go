// Package logger provides structured logging with zerolog.
package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// New returns a zerolog.Logger configured for JSON output to stdout.
// Timestamps are in UNIX format for easy parsing in log aggregators.
func New() zerolog.Logger {
	return zerolog.New(os.Stdout).
		With().Timestamp().
		Logger()
}

// NewWithWriter returns a zerolog.Logger writing to the given io.Writer.
// Useful for testing or redirecting output.
func NewWithWriter(w io.Writer) zerolog.Logger {
	return zerolog.New(w).
		With().Timestamp().
		Logger()
}
