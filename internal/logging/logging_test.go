package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// TestStructuredJSONOutput verifies that records are emitted as parseable
// key/value JSON with the expected fields.
func TestStructuredJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithLevel(&buf, slog.LevelInfo)

	logger.Info("daedalus starting", "version", "0.1.0-dev")

	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &rec); err != nil {
		t.Fatalf("expected a JSON log record, got error %v (output: %q)", err, buf.String())
	}
	if rec["msg"] != "daedalus starting" {
		t.Errorf("msg = %v, want %q", rec["msg"], "daedalus starting")
	}
	if rec["version"] != "0.1.0-dev" {
		t.Errorf("version = %v, want %q", rec["version"], "0.1.0-dev")
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v, want %q", rec["level"], "INFO")
	}
}

// TestLevelFiltering verifies that records below the configured level are
// suppressed and records at or above it are emitted.
func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithLevel(&buf, slog.LevelInfo)

	logger.Debug("debug detail")
	if buf.Len() != 0 {
		t.Fatalf("debug record must be suppressed at info level, got: %q", buf.String())
	}

	logger.Info("info event")
	if !strings.Contains(buf.String(), "info event") {
		t.Fatalf("info record must be emitted at info level, got: %q", buf.String())
	}
}

// TestParseLevel covers the level resolution, including the default for empty
// and unknown values.
func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		// Empty and unknown values resolve to warn: the quiet, production
		// default. info/debug must be requested explicitly.
		{"", slog.LevelWarn},
		{"nonsense", slog.LevelWarn},
	}
	for _, c := range cases {
		if got := ParseLevel(c.in); got != c.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestConsoleHandlerHumanReadable verifies the default text rendering: a record
// becomes a single human-friendly line (level word + message + key=value), not
// JSON. Writing to a buffer (not a TTY) keeps the output plain, so the assertion
// is deterministic and color-free.
func TestConsoleHandlerHumanReadable(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newConsoleHandler(&buf, slog.LevelWarn))

	logger.Warn("seeding default workflow failed", "name", "sdd-default", "attempt", 2)

	out := strings.TrimSpace(buf.String())
	if strings.HasPrefix(out, "{") {
		t.Fatalf("console output must not be JSON, got: %q", out)
	}
	for _, want := range []string{"warning:", "seeding default workflow failed", "name=sdd-default", "attempt=2"} {
		if !strings.Contains(out, want) {
			t.Errorf("console output %q missing %q", out, want)
		}
	}
}

// TestConsoleHandlerRespectsLevel verifies the console handler suppresses records
// below its configured level, so the default warn keeps info/debug quiet.
func TestConsoleHandlerRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newConsoleHandler(&buf, slog.LevelWarn))

	logger.Info("backend selection resolved", "backends", "claude-code")
	if buf.Len() != 0 {
		t.Fatalf("info record must be suppressed at warn level, got: %q", buf.String())
	}

	logger.Error("init failed", "phase", "apply")
	if !strings.Contains(buf.String(), "error:") {
		t.Fatalf("error record must be emitted at warn level, got: %q", buf.String())
	}
}

// TestNewFormatSelection verifies New honors DAEDALUS_LOG_FORMAT: text (default)
// is the human console line, json is the structured record.
func TestNewFormatSelection(t *testing.T) {
	t.Setenv(EnvLevel, "info")

	t.Run("default text", func(t *testing.T) {
		var buf bytes.Buffer
		New(&buf).Info("workspace initialized", "path", ".daedalus")
		if strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
			t.Errorf("default format must be text, got JSON: %q", buf.String())
		}
	})

	t.Run("json opt-in", func(t *testing.T) {
		t.Setenv(EnvFormat, "json")
		var buf bytes.Buffer
		New(&buf).Info("workspace initialized", "path", ".daedalus")
		if !strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
			t.Errorf("DAEDALUS_LOG_FORMAT=json must emit JSON, got: %q", buf.String())
		}
	})
}

// TestNoSensitiveFields documents and guards the logging policy: the canonical
// startup event must not carry secrets, tokens, credentials, or PII.
func TestNoSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithLevel(&buf, slog.LevelInfo)

	logger.Info("daedalus starting", "version", "0.1.0-dev", "interactive", false)

	out := strings.ToLower(buf.String())
	for _, forbidden := range []string{"password", "secret", "token", "credential"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("log output must not contain sensitive field %q: %s", forbidden, buf.String())
		}
	}
}
