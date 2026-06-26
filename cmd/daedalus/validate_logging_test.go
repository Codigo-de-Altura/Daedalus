package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// runValidateInDir runs `daedalus validate` against dir, capturing stdout/stderr
// (the logger writes to stderr), so a test can assert on the emitted log events.
// These tests verify the structured telemetry, so they opt into JSON at info
// level explicitly — the CLI's default is the quiet, human-readable console.
func runValidateInDir(t *testing.T, dir string) (code int, stdout, stderr string) {
	t.Helper()
	t.Setenv("DAEDALUS_LOG_FORMAT", "json")
	t.Setenv("DAEDALUS_LOG_LEVEL", "info")
	var outBuf, errBuf bytes.Buffer
	code = runValidate([]string{"--path", dir}, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// decodeLogLines parses the JSON log records the CLI emitted on stderr into
// generic maps, skipping any non-JSON line so the test is robust to mixed output.
func decodeLogLines(t *testing.T, stderr string) []map[string]any {
	t.Helper()
	var events []map[string]any
	for _, line := range bytes.Split([]byte(stderr), []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(line, &ev); err != nil {
			t.Fatalf("log line is not valid JSON: %s: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}

// findLogByDefinition returns the first event with the given msg and "definition"
// attribute, or fails. It pins the per-finding decision-point event (CA3).
func findLogByDefinition(t *testing.T, events []map[string]any, msg, def string) map[string]any {
	t.Helper()
	for _, ev := range events {
		if ev["msg"] == msg && ev["definition"] == def {
			return ev
		}
	}
	t.Fatalf("no %q event for definition %q; events=%+v", msg, def, events)
	return nil
}

// TestValidateLogsPerFinding covers CA3/CA4: a workspace with a hard convention
// violation (a missing required directory) emits a per-finding "convention
// violated" event at error level, anchored to the workspace-relative location,
// naming the convention and carrying a reason — plus the framing "validation
// started"/"workspace validated" events.
func TestValidateLogsPerFinding(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init failed (%d): %s", code, stderr)
	}

	// Force a hard violation: remove the required agents/ directory so the structure
	// check reports a "required-directory" error finding.
	if err := os.RemoveAll(filepath.Join(dir, ".daedalus", "agents")); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runValidateInDir(t, dir)
	if code != 1 {
		t.Fatalf("validate exit = %d, want 1 (hard violations present)", code)
	}

	events := decodeLogLines(t, stderr)

	// Framing events: the operation announced its start and its aggregate outcome.
	if !hasMsg(events, "validation started") {
		t.Error("missing 'validation started' event")
	}
	if !hasMsg(events, "workspace validated") {
		t.Error("missing 'workspace validated' event")
	}

	// The per-finding decision-point event for the missing agents/ directory.
	ev := findLogByDefinition(t, events, "convention violated", ".daedalus/agents")
	if ev["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR (hard violation)", ev["level"])
	}
	if ev["result"] != "invalid" {
		t.Errorf("result = %v, want invalid", ev["result"])
	}
	if ev["family"] != "structure" {
		t.Errorf("family = %v, want structure", ev["family"])
	}
	if ev["convention"] != "required-directory" {
		t.Errorf("convention = %v, want required-directory", ev["convention"])
	}
	if reason, _ := ev["reason"].(string); reason == "" {
		t.Error("convention violated event carries no reason")
	}
}

// TestValidateConformantLogsNoFindings covers CA4: a conformant workspace emits the
// framing events but no per-finding "convention violated" event, so info-level
// noise is bounded to the operation course (no false rejections logged).
func TestValidateConformantLogsNoFindings(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init failed (%d): %s", code, stderr)
	}

	code, _, stderr := runValidateInDir(t, dir)
	events := decodeLogLines(t, stderr)

	for _, ev := range events {
		if ev["msg"] == "convention violated" {
			t.Errorf("conformant workspace logged a violation (exit=%d): %+v", code, ev)
		}
	}
	if !hasMsg(events, "workspace validated") {
		t.Error("missing 'workspace validated' event")
	}
}

// hasMsg reports whether any event carries the given msg.
func hasMsg(events []map[string]any, msg string) bool {
	for _, ev := range events {
		if ev["msg"] == msg {
			return true
		}
	}
	return false
}
