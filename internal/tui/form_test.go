package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// form_test.go is the permanent test suite for the reusable form layer and the
// wired filter flow (ticket-07-02). It drives the form with the exact keys a user
// types and asserts the lifecycle the validation cares about: open, type, submit
// valid → continue, submit invalid → error and no submit, cancel → return without
// breaking navigation. The form is reached through the real shell, so these also
// prove the form integrates with navigation (no dead end).

// typeString sends each rune of s to the model as an individual key press, like a
// user typing into the focused form field.
func typeString(m Model, s string) Model {
	for _, r := range s {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	return m
}

// sendKeyPumped applies a key and then pumps the resulting commands back into the
// model, the way the Bubble Tea runtime does. Huh advances/submits a form via
// commands (nextFieldMsg / nextGroupMsg), so a bare Update(enter) is not enough to
// reach the completed state in a headless test — the produced command must run and
// its message be fed back. This mirrors what the live program does on every key.
func sendKeyPumped(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, cmd := m.Update(msg)
	m = updated.(Model)
	return pumpCmds(t, m, cmd)
}

// pumpCmds executes cmd (and any commands it yields, including batched ones),
// feeding each produced message back into the model until the commands settle. A
// safety bound prevents an accidental infinite loop in a test.
func pumpCmds(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0 && steps < 100; steps++ {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		msg := c()
		switch mm := msg.(type) {
		case nil:
			continue
		case tea.BatchMsg:
			queue = append(queue, mm...)
		default:
			updated, next := m.Update(msg)
			m = updated.(Model)
			if next != nil {
				queue = append(queue, next)
			}
		}
	}
	return m
}

// loadedAgentsArea enters the agents area and delivers its catalog load, returning
// a model sitting on a populated, deterministic list (the embedded catalog).
func loadedAgentsArea(t *testing.T) Model {
	t.Helper()
	m := sizedModel(t)
	m, cmd := enterAreaByIndex(t, m, indexOf(areaAgents))
	m = deliver(t, m, cmd)
	if len(m.areas[areaAgents].items) == 0 {
		t.Fatal("agents area should be populated from the built-in catalog")
	}
	return m
}

// TestFilterFormOpens covers reaching a form from the TUI (Check precondition): "/"
// on a populated list opens the filter form.
func TestFilterFormOpens(t *testing.T) {
	m := loadedAgentsArea(t)
	m = update(m, "/")
	if m.current() != routeForm {
		t.Fatalf("'/' should open the filter form, route = %v", m.current())
	}
	view := m.View()
	if !strings.Contains(view, "Filter") {
		t.Errorf("form view should show a Filter title/breadcrumb, got:\n%s", view)
	}
	// The submit/cancel affordances must be visible (Check-10).
	if !strings.Contains(view, "submit") || !strings.Contains(view, "cancel") {
		t.Errorf("form should advertise submit/cancel, got:\n%s", view)
	}
}

// TestFilterFormSubmitValid covers Check-4: a valid submission is accepted, the
// flow continues (back to the area), and the filter is applied to the list.
func TestFilterFormSubmitValid(t *testing.T) {
	m := loadedAgentsArea(t)
	total := len(m.areas[areaAgents].visibleItems())

	m = update(m, "/")
	m = typeString(m, "arch")
	m = sendKeyPumped(t, m, keyPress("enter")) // submit (pump huh's completion cmds)

	if m.current() != routeArea {
		t.Fatalf("a valid submit should return to the area, route = %v", m.current())
	}
	st := m.areas[areaAgents]
	if st.filter != "arch" {
		t.Errorf("filter should be applied, got %q", st.filter)
	}
	visible := st.visibleItems()
	if len(visible) == 0 {
		t.Fatal("filter 'arch' should still match the architect agent")
	}
	if len(visible) >= total {
		t.Errorf("filter should narrow the list (%d) below the total (%d)", len(visible), total)
	}
	for _, it := range visible {
		if !strings.Contains(strings.ToLower(it.label+it.badge), "arch") {
			t.Errorf("filtered row %q does not match 'arch'", it.label)
		}
	}
	// The area view shows the active filter banner.
	if !strings.Contains(m.View(), "Filter:") {
		t.Errorf("area should show the active filter banner, got:\n%s", m.View())
	}
}

// TestFilterFormAcceptsLetterQ is the regression test for the "q is eaten in the
// filter input" finding (07-02 minor). In a free-text field, q must be an ordinary
// character — not swallowed by the shell's global quit binding — so queries like
// "queue"/"query" can be typed. It asserts the live input shows the typed q's AND
// that submitting a q-bearing query lands the exact value in the filter.
func TestFilterFormAcceptsLetterQ(t *testing.T) {
	m := loadedAgentsArea(t)

	// A bare 'q' as the very first keystroke in the field must insert, not quit.
	m = update(m, "/")
	m = typeString(m, "q")
	if m.current() != routeForm {
		t.Fatalf("typing 'q' in the field must NOT quit/leave the form, route = %v", m.current())
	}
	if !strings.Contains(visibleText(m.View()), "q") {
		t.Errorf("the typed 'q' should appear in the input, got:\n%s", visibleText(m.View()))
	}

	// A full q-bearing query must be typed in full and submit to the exact value.
	m = loadedAgentsArea(t)
	m = update(m, "/")
	m = typeString(m, "queue")
	if got := visibleText(m.View()); !strings.Contains(got, "queue") {
		t.Errorf("input should contain the full 'queue', got:\n%s", got)
	}
	m = sendKeyPumped(t, m, keyPress("enter"))
	if m.areas[areaAgents].filter != "queue" {
		t.Errorf("submitted q-bearing query should land verbatim, got %q", m.areas[areaAgents].filter)
	}

	// q interleaved with other characters must all survive (no per-key swallowing).
	m = loadedAgentsArea(t)
	m = update(m, "/")
	m = typeString(m, "qaqbqc")
	m = sendKeyPumped(t, m, keyPress("enter"))
	if m.areas[areaAgents].filter != "qaqbqc" {
		t.Errorf("every 'q' should be inserted, got %q", m.areas[areaAgents].filter)
	}
}

// TestFilterFormSubmitInvalidShowsError covers Check-5: a whitespace-only query is
// rejected with a clear error and the submit does not proceed (still on the form).
func TestFilterFormSubmitInvalidShowsError(t *testing.T) {
	m := loadedAgentsArea(t)
	m = update(m, "/")
	m = typeString(m, "   ")                   // whitespace only → invalid
	m = sendKeyPumped(t, m, keyPress("enter")) // attempt submit

	if m.current() != routeForm {
		t.Fatalf("an invalid submit must NOT leave the form, route = %v", m.current())
	}
	if m.areas[areaAgents].filter != "" {
		t.Errorf("an invalid submit must not apply a filter, got %q", m.areas[areaAgents].filter)
	}
	// The validation error message is rendered (Huh shows it in-place).
	if !strings.Contains(m.View(), "non-blank") {
		t.Errorf("a clear validation error should be shown, got:\n%s", m.View())
	}
}

// TestFilterFormCancelReturns covers Check-6: cancelling (esc) mid-entry returns to
// the area without applying anything and without breaking navigation.
func TestFilterFormCancelReturns(t *testing.T) {
	m := loadedAgentsArea(t)
	m = update(m, "/")
	m = typeString(m, "plan")
	m = update(m, "esc") // cancel

	if m.current() != routeArea {
		t.Fatalf("cancel should return to the area, route = %v", m.current())
	}
	if m.areas[areaAgents].filter != "" {
		t.Errorf("cancel must not apply the typed filter, got %q", m.areas[areaAgents].filter)
	}
	// And the area is fully navigable again: esc goes back to the root (no dead end).
	m = update(m, "esc")
	if m.current() != routeRoot {
		t.Error("after cancelling the form, esc should still return to the root")
	}
}

// TestEmptyFilterClears covers the "clear the filter" path: submitting an empty
// query is valid and removes any active filter, restoring the full list.
func TestEmptyFilterClears(t *testing.T) {
	m := loadedAgentsArea(t)
	// First apply a filter.
	m = update(m, "/")
	m = typeString(m, "arch")
	m = sendKeyPumped(t, m, keyPress("enter"))
	if m.areas[areaAgents].filter == "" {
		t.Fatal("precondition: a filter should be set")
	}
	// Reopen and submit empty to clear.
	m = update(m, "/")
	// Clear the seeded value with backspaces (the form seeds the current filter).
	for i := 0; i < len("arch"); i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(Model)
	}
	m = sendKeyPumped(t, m, keyPress("enter"))
	if m.current() != routeArea {
		t.Fatalf("empty submit should be valid and return to the area, route = %v", m.current())
	}
	if m.areas[areaAgents].filter != "" {
		t.Errorf("empty submit should clear the filter, got %q", m.areas[areaAgents].filter)
	}
}

// TestReusableFormBuilders verifies the toolkit's select and confirm builders
// construct valid, themed, navigable form components (the reusable text/select/
// confirm set R5 asks for). They are part of the shared toolkit for later flows;
// this pins them so they keep building real forms rather than rotting.
func TestReusableFormBuilders(t *testing.T) {
	th := defaultTheme()

	var choice string
	sel := newSelectForm(th, "Pick one", []string{"a", "b", "c"}, &choice)
	if sel.form == nil {
		t.Error("select form should be constructed")
	}
	if !strings.Contains(sel.View(), "Pick one") {
		t.Errorf("select form should render its title, got:\n%s", sel.View())
	}

	var ok bool
	conf := newConfirmForm(th, "Proceed?", "Yes", "No", &ok)
	if conf.form == nil {
		t.Error("confirm form should be constructed")
	}
	view := conf.View()
	if !strings.Contains(view, "Proceed?") {
		t.Errorf("confirm form should render its title, got:\n%s", view)
	}
	// The submit/cancel affordance is no longer drawn by the form component itself —
	// the shell renders one contextual help footer (formHelp) for forms, unifying the
	// old double help line (07-03). That affordance is asserted in the help tests; here
	// we only confirm the builders produce valid, themed forms.
}

// TestFilterValidatorUnit checks the validation rule directly so the boundary cases
// are pinned regardless of the form plumbing.
func TestFilterValidatorUnit(t *testing.T) {
	if err := validateFilterQuery(""); err != nil {
		t.Errorf("empty query should be valid (clears filter), got %v", err)
	}
	if err := validateFilterQuery("spec"); err != nil {
		t.Errorf("a normal query should be valid, got %v", err)
	}
	if err := validateFilterQuery("   "); err == nil {
		t.Error("a whitespace-only query should be rejected")
	}
}
