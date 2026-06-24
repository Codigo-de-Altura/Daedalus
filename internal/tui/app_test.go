package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
)

// sizedModel returns a model that has already received a window size, so the
// preview viewport has real dimensions for tests that exercise scrolling/render.
func sizedModel(t *testing.T) Model {
	t.Helper()
	m := New(".")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return updated.(Model)
}

func keyPress(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// TestPromptsLoadedPopulatesList verifies that a promptsLoadedMsg moves the model
// out of the loading state and into a populated list.
func TestPromptsLoadedPopulatesList(t *testing.T) {
	m := sizedModel(t)
	if !m.loadingList {
		t.Fatal("model should start in the loading state")
	}

	entries := []prompts.Entry{
		{ID: "glossary", Kind: prompts.KindShared, Title: "Glossary"},
		{ID: "style", Kind: prompts.KindGlobal, Title: "Project Style"},
	}
	updated, _ := m.Update(promptsLoadedMsg{entries: entries})
	m = updated.(Model)

	if m.loadingList {
		t.Error("loadingList should be false after prompts load")
	}
	if len(m.entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(m.entries))
	}
	view := m.View()
	if !strings.Contains(view, "glossary") || !strings.Contains(view, "style") {
		t.Errorf("list view should contain both prompt ids, got:\n%s", view)
	}
}

// TestEmptyStateRendered verifies the empty list renders a clear message rather
// than a blank screen.
func TestEmptyStateRendered(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{}})
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "No prompts found") {
		t.Errorf("empty state message missing, got:\n%s", view)
	}
}

// TestListLoadErrorRendered verifies a listing failure surfaces as a readable
// error instead of crashing or showing nothing.
func TestListLoadErrorRendered(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{err: errors.New("permission denied")})
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "Could not read prompts") {
		t.Errorf("list error message missing, got:\n%s", view)
	}
}

// TestOpenPreviewStartsResolve verifies enter on a selected prompt switches to
// the preview screen, marks it loading, and returns a command (the async
// resolve).
func TestOpenPreviewStartsResolve(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "glossary", Kind: prompts.KindShared, Title: "Glossary"},
	}})
	m = updated.(Model)

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(Model)

	if m.screen != screenPreview {
		t.Fatal("enter should switch to the preview screen")
	}
	if m.previewState != previewLoading {
		t.Error("preview should be in the loading state right after opening")
	}
	if m.previewID != "glossary" {
		t.Errorf("previewID = %q, want glossary", m.previewID)
	}
	if cmd == nil {
		t.Error("opening a preview should return a resolve command")
	}
}

// TestResolvedContentFillsViewport verifies a successful resolve fills the
// viewport and marks the preview ready.
func TestResolvedContentFillsViewport(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "glossary", Kind: prompts.KindShared, Title: "Glossary"},
	}})
	m = updated.(Model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(Model)

	updated, _ = m.Update(promptResolvedMsg{id: "glossary", content: "# Heading\n\nbody text"})
	m = updated.(Model)

	if m.previewState != previewReady {
		t.Fatalf("preview state = %v, want ready", m.previewState)
	}
	if m.viewport.TotalLineCount() == 0 {
		t.Error("viewport should hold the rendered content")
	}
	view := m.View()
	if !strings.Contains(view, "Preview") {
		t.Errorf("preview header missing, got:\n%s", view)
	}
}

// TestResolveErrorShowsMessage verifies a composition error drives the preview
// into the error state with a readable, typed message (cycle vs not-found), and
// never crashes.
func TestResolveErrorShowsMessage(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "cycle",
			err:  &prompts.IncludeCycleError{Chain: []string{"a", "b", "a"}},
			want: "inclusion cycle",
		},
		{
			name: "missing",
			err:  &prompts.IncludeNotFoundError{MissingID: "ghost", ReferencedBy: "a"},
			want: "missing include reference",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := sizedModel(t)
			updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
				{ID: "a", Kind: prompts.KindGlobal, Title: "A"},
			}})
			m = updated.(Model)
			updated, _ = m.Update(keyPress("enter"))
			m = updated.(Model)

			updated, _ = m.Update(promptResolvedMsg{id: "a", err: tc.err})
			m = updated.(Model)

			if m.previewState != previewErrored {
				t.Fatalf("preview state = %v, want errored", m.previewState)
			}
			view := m.View()
			if !strings.Contains(view, tc.want) {
				t.Errorf("error view missing %q, got:\n%s", tc.want, view)
			}
		})
	}
}

// TestStaleResolveIgnored verifies a resolved message for a prompt the user has
// navigated away from does not overwrite the current preview.
func TestStaleResolveIgnored(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "current", Kind: prompts.KindGlobal, Title: "Current"},
	}})
	m = updated.(Model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(Model)

	// A result for a different (older) prompt must be ignored.
	updated, _ = m.Update(promptResolvedMsg{id: "old", content: "stale"})
	m = updated.(Model)

	if m.previewState != previewLoading {
		t.Errorf("stale resolve should not change the preview state, got %v", m.previewState)
	}
}

// TestEscReturnsToList verifies esc from the preview goes back to the list (no
// dead end), while q does not quit from within the preview.
func TestEscReturnsToList(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "a", Kind: prompts.KindGlobal, Title: "A"},
	}})
	m = updated.(Model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(Model)

	// q inside the preview must not quit.
	_, cmd := m.Update(keyPress("q"))
	if cmd != nil {
		t.Error("q inside the preview should be a no-op, not quit")
	}

	updated, _ = m.Update(keyPress("esc"))
	m = updated.(Model)
	if m.screen != screenList {
		t.Error("esc should return to the list screen")
	}
}

// TestQuitFromList verifies q quits from the list screen.
func TestQuitFromList(t *testing.T) {
	m := sizedModel(t)
	_, cmd := m.Update(keyPress("q"))
	if cmd == nil {
		t.Fatal("q on the list should return the quit command")
	}
	if msg := cmd(); msg == nil {
		t.Error("quit command should produce a message")
	}
}

// TestCursorNavigation verifies up/down move the selection within bounds.
func TestCursorNavigation(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}})
	m = updated.(Model)

	if m.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", m.cursor)
	}
	// Up at the top is clamped.
	updated, _ = m.Update(keyPress("up"))
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
	updated, _ = m.Update(keyPress("down"))
	m = updated.(Model)
	updated, _ = m.Update(keyPress("down"))
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor should be 2 after two downs, got %d", m.cursor)
	}
	// Down at the bottom is clamped.
	updated, _ = m.Update(keyPress("down"))
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor should clamp at last index, got %d", m.cursor)
	}
}

// TestHelpContainsBindings verifies the contextual help lists the key actions on
// each screen (R5) — generated from the bindings, so it can never drift.
func TestHelpContainsBindings(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(promptsLoadedMsg{entries: []prompts.Entry{
		{ID: "a", Kind: prompts.KindGlobal, Title: "A"},
	}})
	m = updated.(Model)

	listView := m.View()
	if !strings.Contains(listView, "open preview") || !strings.Contains(listView, "quit") {
		t.Errorf("list help should advertise open/quit, got:\n%s", listView)
	}

	updated, _ = m.Update(keyPress("enter"))
	m = updated.(Model)
	previewView := m.View()
	if !strings.Contains(previewView, "back to list") {
		t.Errorf("preview help should advertise back-to-list, got:\n%s", previewView)
	}
}
