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
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
	}
	for _, c := range cases {
		if got := ParseLevel(c.in); got != c.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
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
