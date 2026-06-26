// Package logging provides Daedalus's structured logging baseline.
//
// It wraps the standard library's log/slog to emit key/value records at the
// decision points of the core (init, build, validation). Two things are
// configurable through the environment:
//
//   - DAEDALUS_LOG_LEVEL (debug|info|warn|error) — the minimum level. It
//     defaults to warn so a normal CLI run stays quiet: the human-readable
//     command output owns the terminal, and info/debug telemetry is opt-in.
//   - DAEDALUS_LOG_FORMAT (text|json) — the rendering. It defaults to text, a
//     compact, human-friendly console line (colored on a TTY). json emits the
//     machine-parseable structured records meant for telemetry and tooling.
//
// Output is written to an explicit io.Writer (stderr in the binary) so it never
// corrupts the Bubble Tea render, which owns stdout.
//
// Sensitive-data policy (enforced by convention, not by the type system):
// callers MUST NOT log secrets, tokens, credentials, PII, or raw file
// contents. Log the identifiers and the decision taken, never the sensitive
// payload behind them. The logging baseline gives later epics a stable, shared
// API so this rule is applied consistently across the project.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Environment variables that tune the logger.
const (
	// EnvLevel overrides the minimum log level (debug|info|warn|error).
	EnvLevel = "DAEDALUS_LOG_LEVEL"
	// EnvFormat selects the rendering (text|json).
	EnvFormat = "DAEDALUS_LOG_FORMAT"
)

// Format is the on-the-wire rendering of a log record.
type Format int

const (
	// FormatText is the human-friendly console line (the default).
	FormatText Format = iota
	// FormatJSON is the machine-parseable structured record (telemetry).
	FormatJSON
)

// New returns a logger configured from the environment, writing to w. The level
// comes from DAEDALUS_LOG_LEVEL (default warn) and the rendering from
// DAEDALUS_LOG_FORMAT (default text). This is what the binary uses, so normal
// runs are quiet and readable while telemetry stays one env var away.
func New(w io.Writer) *slog.Logger {
	level := levelFromEnv()
	if formatFromEnv() == FormatJSON {
		return NewWithLevel(w, level)
	}
	return slog.New(newConsoleHandler(w, level))
}

// NewWithLevel returns a structured JSON logger writing to w at the given
// minimum level. It avoids hidden global state so the core can inject the
// logger it needs instead of reaching for a package-level singleton, and it is
// the explicit way to ask for machine-parseable output regardless of the
// environment (the compile/validate instrumentation tests rely on it).
func NewWithLevel(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// levelFromEnv resolves the minimum level from the environment, defaulting to
// warn when unset or unrecognized.
func levelFromEnv() slog.Level {
	return ParseLevel(os.Getenv(EnvLevel))
}

// formatFromEnv resolves the rendering from the environment, defaulting to text.
func formatFromEnv() Format {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvFormat))) {
	case "json":
		return FormatJSON
	default:
		return FormatText
	}
}

// ParseLevel maps a case-insensitive level name to a slog.Level. Empty or
// unknown values resolve to warn, keeping the default output quiet and
// production-friendly; info and debug are opt-in via DAEDALUS_LOG_LEVEL.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

// consoleHandler renders records as a compact, human-friendly line:
//
//	warning: seeding default workflow failed  name=sdd-default err=...
//
// The level is a colored word label (color only when w is a terminal, via
// lipgloss/termenv profile detection — piped or NO_COLOR output stays plain),
// the time is omitted (noise for an interactive CLI), and attributes trail as
// dimmed key=value pairs. It is deliberately small; groups are flattened with a
// dotted prefix and there is no buffering beyond the per-record line.
type consoleHandler struct {
	w      io.Writer
	level  slog.Leveler
	styles consoleStyles
	attrs  []slog.Attr
	group  string
}

type consoleStyles struct {
	warn  lipgloss.Style
	err   lipgloss.Style
	info  lipgloss.Style
	debug lipgloss.Style
	key   lipgloss.Style
}

func newConsoleHandler(w io.Writer, level slog.Level) *consoleHandler {
	// Bind the renderer to w so color is enabled only when w itself is a capable
	// terminal; a buffer or a pipe degrades to plain text automatically.
	r := lipgloss.NewRenderer(w)
	return &consoleHandler{
		w:     w,
		level: level,
		styles: consoleStyles{
			warn:  r.NewStyle().Bold(true).Foreground(lipgloss.Color("#ef9c34")),
			err:   r.NewStyle().Bold(true).Foreground(lipgloss.Color("#e5484d")),
			info:  r.NewStyle().Foreground(lipgloss.Color("#7aa2cf")),
			debug: r.NewStyle().Faint(true),
			key:   r.NewStyle().Faint(true),
		},
	}
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *consoleHandler) Handle(_ context.Context, rec slog.Record) error {
	var b strings.Builder
	b.WriteString(h.label(rec.Level))
	b.WriteByte(' ')
	b.WriteString(rec.Message)

	write := func(a slog.Attr) {
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return
		}
		key := a.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		b.WriteByte(' ')
		b.WriteString(h.styles.key.Render(key + "="))
		b.WriteString(a.Value.String())
	}
	for _, a := range h.attrs {
		write(a)
	}
	rec.Attrs(func(a slog.Attr) bool {
		write(a)
		return true
	})

	b.WriteByte('\n')
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *consoleHandler) label(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return h.styles.err.Render("error:")
	case level >= slog.LevelWarn:
		return h.styles.warn.Render("warning:")
	case level >= slog.LevelInfo:
		return h.styles.info.Render("info:")
	default:
		return h.styles.debug.Render("debug:")
	}
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nh := *h
	nh.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &nh
}

func (h *consoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := *h
	if h.group != "" {
		nh.group = h.group + "." + name
	} else {
		nh.group = name
	}
	return &nh
}
