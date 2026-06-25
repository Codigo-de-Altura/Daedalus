package compile

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// logEvent is one decoded slog JSON record. Only the fields the instrumentation
// asserts on are named; the rest land in Extra so a test can read any structured
// attribute (e.g. "definition", "result") without binding to a fixed schema.
type logEvent struct {
	Level string
	Msg   string
	Extra map[string]any
}

// captureBuild runs Build with a JSON logger writing to a buffer and returns the
// decoded decision-point events. It is the idiomatic way to assert on injected
// logging: drive the operation, parse the JSON lines, inspect the structured
// attributes — no global logger, no string matching on rendered text.
func captureBuild(t *testing.T, opts Options) (*Outcome, error, []logEvent) {
	t.Helper()
	var buf bytes.Buffer
	opts.Logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	out, err := Build(opts)
	return out, err, decodeEvents(t, &buf)
}

// decodeEvents parses every JSON line the logger emitted into a logEvent, failing
// the test on a malformed line so a broken record can never pass silently.
func decodeEvents(t *testing.T, buf *bytes.Buffer) []logEvent {
	t.Helper()
	var events []logEvent
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			t.Fatalf("log line is not valid JSON: %s: %v", line, err)
		}
		ev := logEvent{Extra: map[string]any{}}
		for k, v := range raw {
			switch k {
			case "level":
				ev.Level, _ = v.(string)
			case "msg":
				ev.Msg, _ = v.(string)
			case "time":
				// Volatile; ignored so assertions stay deterministic.
			default:
				ev.Extra[k] = v
			}
		}
		events = append(events, ev)
	}
	return events
}

// findEvent returns the first event whose msg equals want and whose "definition"
// attribute equals def, or fails the test. It is how a test pins the per-definition
// decision-point event (CA3) for a specific source.
func findEvent(t *testing.T, events []logEvent, msg, def string) logEvent {
	t.Helper()
	for _, ev := range events {
		if ev.Msg == msg && ev.Extra["definition"] == def {
			return ev
		}
	}
	t.Fatalf("no %q event for definition %q; events=%+v", msg, def, events)
	return logEvent{}
}

// writeRawAgent materializes an agent directly from raw file contents so a test can
// craft a malformed or schema-invalid definition the built-in catalog never emits.
func writeRawAgent(t *testing.T, root, id, agentYAML, prompt string) {
	t.Helper()
	dir := filepath.Join(root, workspace.Name, catalog.AgentsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, catalog.DefinitionFileName), []byte(agentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, catalog.PromptFileName), []byte(prompt), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBuildLogsValidAgentAtInfo covers CA3/CA4: a valid agent emits a
// "definition accepted" event at info, carrying its kind, id, workspace-relative
// path and a "valid" result.
func TestBuildLogsValidAgentAtInfo(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files:   []Artifact{{RelPath: ".claude/agents/analyst.md", Content: "x\n"}},
	}}

	_, err, events := captureBuild(t, Options{Root: root, Registry: registryWith(fake)})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	ev := findEvent(t, events, "definition accepted", ".daedalus/agents/analyst")
	if ev.Level != "INFO" {
		t.Errorf("level = %q, want INFO", ev.Level)
	}
	if ev.Extra["kind"] != "agent" {
		t.Errorf("kind = %v, want agent", ev.Extra["kind"])
	}
	if ev.Extra["result"] != "valid" {
		t.Errorf("result = %v, want valid", ev.Extra["result"])
	}
	if ev.Extra["phase"] != "validate" {
		t.Errorf("phase = %v, want validate", ev.Extra["phase"])
	}
	if ev.Extra["id"] != "analyst" {
		t.Errorf("id = %v, want analyst", ev.Extra["id"])
	}
}

// TestBuildLogsInvalidAgentWithReason covers CA3: a schema-invalid agent emits a
// "definition rejected" event at warn with an "invalid" result and a reason, so
// the trace shows which definition failed and why.
func TestBuildLogsInvalidAgentWithReason(t *testing.T) {
	root := initWorkspace(t)
	// Parses fine (id + role present) but the prompt body is empty ⇒ schema-invalid.
	writeRawAgent(t, root, "hollow", "id: hollow\nrole: tester\nprompt: prompt.md\n", "")

	fake := fakeCompiler{backend: workspace.DefaultBackend}
	_, err, events := captureBuild(t, Options{Root: root, Registry: registryWith(fake)})
	if !IsDefinitionInvalid(err) {
		t.Fatalf("err = %v, want *DefinitionError", err)
	}

	ev := findEvent(t, events, "definition rejected", ".daedalus/agents/hollow")
	if ev.Level != "WARN" {
		t.Errorf("level = %q, want WARN", ev.Level)
	}
	if ev.Extra["result"] != "invalid" {
		t.Errorf("result = %v, want invalid", ev.Extra["result"])
	}
	reason, _ := ev.Extra["reason"].(string)
	if reason == "" {
		t.Error("rejected event carries no reason")
	}
}

// TestBuildLogsMalformedAgent covers CA3: an unparsable agent.yaml emits a
// "definition rejected" event at warn with a "malformed" result.
func TestBuildLogsMalformedAgent(t *testing.T) {
	root := initWorkspace(t)
	// Missing the required "role" key ⇒ the loader cannot parse it as an agent.
	writeRawAgent(t, root, "broken", "id: broken\n", "body\n")

	fake := fakeCompiler{backend: workspace.DefaultBackend}
	_, err, events := captureBuild(t, Options{Root: root, Registry: registryWith(fake)})
	if !IsDefinitionInvalid(err) {
		t.Fatalf("err = %v, want *DefinitionError", err)
	}

	ev := findEvent(t, events, "definition rejected", ".daedalus/agents/broken")
	if ev.Level != "WARN" {
		t.Errorf("level = %q, want WARN", ev.Level)
	}
	if ev.Extra["result"] != "malformed" {
		t.Errorf("result = %v, want malformed", ev.Extra["result"])
	}
}

// TestBuildLogsNoAbsolutePathsOrPromptBody covers CA5/R5: the per-definition
// decision-point events log artifact paths RELATIVE to the workspace (never
// absolute) and never leak the prompt body. The agent's prompt body is a unique
// sentinel; it must never appear in any log attribute. The "definition" attribute
// — the path this ticket introduces — must always be workspace-relative.
func TestBuildLogsNoAbsolutePathsOrPromptBody(t *testing.T) {
	root := initWorkspace(t)
	const secretBody = "TOP-SECRET-PROMPT-BODY-do-not-log"
	writeRawAgent(t, root, "leaky", "id: leaky\nrole: tester\nprompt: prompt.md\n", secretBody)

	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files:   []Artifact{{RelPath: ".claude/agents/leaky.md", Content: "x\n"}},
	}}
	_, _, events := captureBuild(t, Options{Root: root, Registry: registryWith(fake)})

	for _, ev := range events {
		// The "definition" attribute (the artifact path this ticket owns) must be
		// workspace-relative — anchored at the .daedalus/ workspace name, never an
		// absolute filesystem path that would leak the user's directory layout.
		if def, ok := ev.Extra["definition"].(string); ok && def != "" {
			if filepath.IsAbs(def) {
				t.Errorf("event %q logs an absolute definition path: %q", ev.Msg, def)
			}
			if !bytesContains(def, workspace.Name+"/") {
				t.Errorf("event %q definition path is not workspace-relative: %q", ev.Msg, def)
			}
		}
		// No attribute, anywhere, may carry the prompt body (R5/CA5).
		for k, v := range ev.Extra {
			if s, ok := v.(string); ok && bytesContains(s, secretBody) {
				t.Errorf("event %q attr %q leaks the prompt body: %q", ev.Msg, k, s)
			}
		}
	}
}

// bytesContains is a tiny substring helper kept local so the assertion above reads
// as one expression.
func bytesContains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}

// TestBuildDeterministicWithAndWithoutLogging covers CA6/RNF-5: build produces
// byte-identical artifacts whether logging is active (debug to a buffer) or silent
// (the default no-op logger). Logging is observational only.
func TestBuildDeterministicWithAndWithoutLogging(t *testing.T) {
	// Two independent workspaces with identical inputs so the only difference is
	// whether logging is active; comparing them proves the logging does not perturb
	// the written artifacts.
	build := func(t *testing.T, withLogging bool) map[string]string {
		t.Helper()
		root := initWorkspace(t)
		addAgent(t, root, "analyst")
		fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
			Backend: workspace.DefaultBackend,
			Files: []Artifact{
				{RelPath: ".claude/agents/analyst.md", Content: "agent body\n"},
				{RelPath: ".claude/commands/x.md", Content: "command body\n"},
			},
		}}
		opts := Options{Root: root, Registry: registryWith(fake)}
		if withLogging {
			var buf bytes.Buffer
			opts.Logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
		if _, err := Build(opts); err != nil {
			t.Fatalf("Build: %v", err)
		}
		return readClaudeTree(t, root)
	}

	silent := build(t, false)
	logged := build(t, true)

	if len(silent) != len(logged) {
		t.Fatalf("artifact count differs: silent=%d logged=%d", len(silent), len(logged))
	}
	for rel, content := range silent {
		if logged[rel] != content {
			t.Errorf("artifact %q differs with logging on:\n silent=%q\n logged=%q", rel, content, logged[rel])
		}
	}
}

// readClaudeTree reads every file under <root>/.claude into a rel-path→content map
// so two builds can be compared byte-for-byte.
func readClaudeTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	claudeRoot := filepath.Join(root, ".claude")
	err := filepath.Walk(claudeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		rel, rerr := filepath.Rel(claudeRoot, path)
		if rerr != nil {
			return rerr
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walk .claude: %v", err)
	}
	return out
}
