// Package logging provides Daedalus's structured logging baseline.
//
// It wraps the standard library's log/slog to emit key/value records at the
// decision points of the core (init, build, validation). The minimum level is
// configurable via the DAEDALUS_LOG_LEVEL environment variable
// (debug|info|warn|error) and defaults to info. Output is written to an
// explicit io.Writer (stderr in the binary) so it never corrupts the Bubble
// Tea render, which owns stdout.
//
// Sensitive-data policy (enforced by convention, not by the type system):
// callers MUST NOT log secrets, tokens, credentials, PII, or raw file
// contents. Log the identifiers and the decision taken, never the sensitive
// payload behind them. The logging baseline gives later epics a stable, shared
// API so this rule is applied consistently across the project.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// EnvLevel is the environment variable that overrides the default log level.
const EnvLevel = "DAEDALUS_LOG_LEVEL"

// New returns a structured JSON logger writing to w, using the level resolved
// from the DAEDALUS_LOG_LEVEL environment variable (default: info).
func New(w io.Writer) *slog.Logger {
	return NewWithLevel(w, levelFromEnv())
}

// NewWithLevel returns a structured JSON logger writing to w at the given
// minimum level. It avoids hidden global state so the core can inject the
// logger it needs instead of reaching for a package-level singleton.
func NewWithLevel(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// levelFromEnv resolves the minimum level from the environment, defaulting to
// info when unset or unrecognized.
func levelFromEnv() slog.Level {
	return ParseLevel(os.Getenv(EnvLevel))
}

// ParseLevel maps a case-insensitive level name to a slog.Level. Empty or
// unknown values resolve to info so a misconfiguration never silences logging.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
