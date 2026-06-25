package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// help_test.go is the permanent suite for the central keybinding registry and the
// contextual help (ticket-07-03). It drives the model with exact keys and asserts:
//   - the registry has no key collisions and one binding per action;
//   - help (?) is reachable in every context (root/area/sub/form/loading/empty/error);
//   - each context's help lists the right actions (a form lists submit/cancel/move,
//     not just navigation);
//   - the same action maps to the same key in all six areas (consistency);
//   - every action a context announces actually runs (announced == real).

// helpText renders a context's short+full help to plain text (ANSI stripped) so a
// test can assert what the user would read.
func helpText(m Model) string {
	short := m.ShortHelp()
	full := m.FullHelp()
	var b strings.Builder
	for _, bnd := range short {
		b.WriteString(bnd.Help().Key + " " + bnd.Help().Desc + "\n")
	}
	for _, group := range full {
		for _, bnd := range group {
			b.WriteString(bnd.Help().Key + " " + bnd.Help().Desc + "\n")
		}
	}
	return b.String()
}

// TestRegistryNoKeyCollisions verifies no two distinct actions share a trigger key,
// so a key never has two divergent meanings (R2). It also confirms every action
// resolves to a non-empty binding.
func TestRegistryNoKeyCollisions(t *testing.T) {
	km := defaultKeymap()
	owner := map[string]keyAction{}
	allActions := []keyAction{
		actionUp, actionDown, actionEnter, actionBack, actionHome, actionRetry,
		actionFilter, actionHelp, actionQuit, actionPageUp, actionPageDown,
		actionTop, actionBottom, actionFormSubmit, actionFormCancel,
		actionFormNextField, actionFormPrevField,
	}

	// Actions that legitimately share keys because they are the SAME gesture in
	// different contexts (never active at once): enter == form submit; esc == back ==
	// form cancel. These are intentional unifications, not collisions.
	allowedShared := map[string]map[keyAction]bool{
		"enter": {actionEnter: true, actionFormSubmit: true},
		"esc":   {actionBack: true, actionFormCancel: true},
	}

	for _, a := range allActions {
		b := km.binding(a)
		if len(b.Keys()) == 0 {
			t.Errorf("action %d resolves to an empty binding", a)
		}
		for _, k := range b.Keys() {
			if prev, ok := owner[k]; ok {
				if allowedShared[k] != nil && allowedShared[k][prev] && allowedShared[k][a] {
					continue
				}
				t.Errorf("key %q is bound to two actions (%d and %d) — collision", k, prev, a)
			}
			owner[k] = a
		}
	}
}

// TestHelpReachableEverywhere covers R4/Check-1/2/8: ? toggles the expanded help in
// every context, including the loading/empty/error states and a form.
func TestHelpReachableEverywhere(t *testing.T) {
	// root
	m := sizedModel(t)
	assertHelpToggles(t, m, "root")

	// area (loaded, populated)
	a := loadedAgentsArea(t)
	assertHelpToggles(t, a, "area")

	// area loading (entered but not yet loaded)
	ld := sizedModel(t)
	ld, _ = enterAreaByIndex(t, ld, indexOf(areaPrompts))
	if !ld.areas[areaPrompts].loading {
		t.Fatal("precondition: prompts area should be loading")
	}
	assertHelpToggles(t, ld, "loading")

	// area empty
	em := sizedModel(t)
	em, _ = enterAreaByIndex(t, em, indexOf(areaPrompts))
	em = deliverAreaLoaded(em, areaPrompts, nil)
	assertHelpToggles(t, em, "empty")

	// area error
	er := sizedModel(t)
	er, _ = enterAreaByIndex(t, er, indexOf(areaBacklog))
	er = deliverAreaLoadedErr(er, areaBacklog)
	assertHelpToggles(t, er, "error")

	// sub-screen
	sub := sizedModel(t)
	sub, _ = enterAreaByIndex(t, sub, indexOf(areaPrompts))
	sub = deliverAreaLoaded(sub, areaPrompts, []areaItem{{key: "p", label: "p", opens: true}})
	sub = update(sub, "enter")
	sub = deliverSubLoaded(sub, areaPrompts, "p", "# body")
	if sub.current() != routeSub {
		t.Fatal("precondition: should be on a sub-screen")
	}
	assertHelpToggles(t, sub, "sub")

	// form
	fm := loadedAgentsArea(t)
	fm = update(fm, "/")
	if fm.current() != routeForm {
		t.Fatal("precondition: should be on a form")
	}
	assertHelpToggles(t, fm, "form")
}

// assertHelpToggles presses ? and checks the expanded-help flag flips on, then off.
func assertHelpToggles(t *testing.T, m Model, ctx string) {
	t.Helper()
	if m.help.ShowAll {
		t.Fatalf("[%s] help should start collapsed", ctx)
	}
	m = update(m, "?")
	if !m.help.ShowAll {
		t.Errorf("[%s] '?' should expand the help", ctx)
	}
	m = update(m, "?")
	if m.help.ShowAll {
		t.Errorf("[%s] '?' should collapse the help again", ctx)
	}
}

// TestFormHelpListsFormActions covers Check-4: a form's help lists submit, cancel
// and move-between-fields — not just generic navigation.
func TestFormHelpListsFormActions(t *testing.T) {
	m := loadedAgentsArea(t)
	m = update(m, "/")
	txt := helpText(m)
	for _, want := range []string{"submit", "cancel", "next field", "prev field"} {
		if !strings.Contains(txt, want) {
			t.Errorf("form help should list %q, got:\n%s", want, txt)
		}
	}
	// It should NOT advertise area-list-only actions like filter/open inside the form.
	if strings.Contains(txt, "filter") {
		t.Errorf("form help should not advertise the list filter, got:\n%s", txt)
	}
}

// TestAreaHelpListsNavigation covers Check-1/2: an area's help lists move/open/
// filter/back/home/help/quit.
func TestAreaHelpListsNavigation(t *testing.T) {
	m := loadedAgentsArea(t)
	txt := helpText(m)
	for _, want := range []string{"up", "down", "open", "filter", "back", "home", "help", "quit"} {
		if !strings.Contains(txt, want) {
			t.Errorf("area help should list %q, got:\n%s", want, txt)
		}
	}
}

// TestSameActionSameKeyAcrossAreas covers R2/Check-3/6: the keys behind each common
// action are identical in all six areas (the registry is shared, so the area help
// context resolves to the same bindings regardless of which area is active).
func TestSameActionSameKeyAcrossAreas(t *testing.T) {
	// Capture each area's resolved short-help bindings and compare key sets per
	// action description. We compare descriptions that appear in more than one area;
	// because every area's help resolves through the ONE central registry, the same
	// action must carry identical keys everywhere — regardless of whether a given area
	// happened to load, be empty, or error in this environment (all contexts draw from
	// the same registry, which is the property under test).
	reference := map[string][]string{}
	for i, id := range areaOrder {
		m := sizedModel(t)
		m, cmd := enterAreaByIndex(t, m, i)
		m = deliver(t, m, cmd)
		got := keysByDesc(m.ShortHelp())
		for desc, keys := range got {
			if ref, ok := reference[desc]; ok {
				if strings.Join(ref, ",") != strings.Join(keys, ",") {
					t.Errorf("action %q uses keys %v in area %v but %v elsewhere — inconsistent",
						desc, keys, id, ref)
				}
			} else {
				reference[desc] = keys
			}
		}
	}
}

// keysByDesc maps each binding's help description to its trigger keys, so two
// contexts can be compared action-by-action.
func keysByDesc(bindings []key.Binding) map[string][]string {
	out := map[string][]string{}
	for _, b := range bindings {
		out[b.Help().Desc] = b.Keys()
	}
	return out
}

// TestAnnouncedEqualsRealArea covers Check-5: every action the area help announces
// actually executes its effect when its key is pressed. We drive a representative
// announced action (filter -> opens the form) and verify the effect.
func TestAnnouncedEqualsRealArea(t *testing.T) {
	m := loadedAgentsArea(t)

	// The area help announces "filter" on "/" — pressing it must open the form.
	announced := announces(m.ShortHelp(), "filter")
	if announced == "" {
		t.Fatal("area help should announce a filter key")
	}
	m2 := pressFirstKey(m, announced)
	if m2.current() != routeForm {
		t.Errorf("the announced filter key %q did not open the form (announced != real)", announced)
	}

	// The area help announces "back" on "esc" — pressing it must pop to the root.
	m3 := loadedAgentsArea(t)
	back := announces(m3.ShortHelp(), "back")
	if back == "" {
		t.Fatal("area help should announce a back key")
	}
	m4 := pressFirstKey(m3, back)
	if m4.current() != routeRoot {
		t.Errorf("the announced back key %q did not return to the root (announced != real)", back)
	}
}

// TestAnnouncedEqualsRealForm covers Check-5 in a form: the announced cancel key
// actually cancels the form back to the area.
func TestAnnouncedEqualsRealForm(t *testing.T) {
	m := loadedAgentsArea(t)
	m = update(m, "/")
	cancel := announces(m.ShortHelp(), "cancel")
	if cancel == "" {
		t.Fatal("form help should announce a cancel key")
	}
	m = pressFirstKey(m, cancel)
	if m.current() != routeArea {
		t.Errorf("the announced cancel key %q did not return to the area (announced != real)", cancel)
	}
}

// TestBuildPreviewSharesRegistryKeys covers R2 reconciliation: the standalone build
// preview (a separate program) honors "same action ⇒ same key" by sourcing its
// shared actions from the central registry. We assert its scroll/jump/help/quit keys
// equal the registry's, so they can never drift apart even though the build preview
// renders different help wording.
func TestBuildPreviewSharesRegistryKeys(t *testing.T) {
	reg := defaultKeymap()
	bk := defaultBuildKeyMap()

	cases := []struct {
		name    string
		action  keyAction
		binding key.Binding
	}{
		{"up", actionUp, bk.Up},
		{"down", actionDown, bk.Down},
		{"pageUp", actionPageUp, bk.PgUp},
		{"pageDown", actionPageDown, bk.PgDn},
		{"top", actionTop, bk.Top},
		{"bottom", actionBottom, bk.Botom},
		{"help", actionHelp, bk.Help},
		{"quit", actionQuit, bk.Quit},
	}
	for _, c := range cases {
		want := strings.Join(reg.binding(c.action).Keys(), ",")
		got := strings.Join(c.binding.Keys(), ",")
		if want != got {
			t.Errorf("build preview %q keys %q diverge from registry %q", c.name, got, want)
		}
	}
}

// announces returns the first trigger key for the binding whose help description is
// desc, or "" if no such binding is advertised.
func announces(bindings []key.Binding, desc string) string {
	for _, b := range bindings {
		if b.Help().Desc == desc {
			ks := b.Keys()
			if len(ks) > 0 {
				return ks[0]
			}
		}
	}
	return ""
}

// pressFirstKey sends a single key (by its string name) to the model, mapping the
// common named keys to their KeyMsg type so esc/enter are delivered correctly.
func pressFirstKey(m Model, k string) Model {
	var msg tea.KeyMsg
	switch k {
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// deliverAreaLoadedErr injects a load error for an area so the error-state help can
// be exercised.
func deliverAreaLoadedErr(m Model, id areaID) Model {
	updated, _ := m.Update(areaLoadedMsg{id: id, err: errHelpTest{}})
	return updated.(Model)
}

type errHelpTest struct{}

func (errHelpTest) Error() string { return "boom" }
