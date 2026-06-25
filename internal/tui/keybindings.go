package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// keybindings.go is the SINGLE source of truth for every keyboard shortcut in the
// TUI (ticket-07-03, RF-7.3). One action ⇒ one key binding, defined exactly once
// here, so the same action uses the same key everywhere and there are no divergent
// meanings or collisions (R1/R2). Both the actual key handling (Update branches in
// app.go / form.go) and the contextual help (help.go) resolve their keys through
// this registry, which is what makes "what the help announces" identical to "what
// actually runs" (R7/Check-5) — the help text can never drift from the real keys
// because there is no parallel hardcoded help string.
//
// Screens do NOT redefine keys; they DECLARE which subset of actions they expose in
// their context (see help.go's helpContext), and the registry resolves each action
// to its one binding. Adding a screen is choosing a subset, never inventing a key.
//
// User remapping/configurability is intentionally out of scope for Phase 1 (per the
// ticket); this registry is a fixed default set, structured so a future config layer
// could populate it without touching any call site.

// keyAction enumerates every distinct user action the TUI understands. It is the
// stable vocabulary screens reference; the binding for each action lives in keymap.
type keyAction int

const (
	actionUp keyAction = iota
	actionDown
	actionEnter
	actionBack
	actionHome
	actionRetry
	actionFilter
	actionHelp
	actionQuit
	actionPageUp
	actionPageDown
	actionTop
	actionBottom
	// Form actions. They are part of the SAME registry as navigation so a form's
	// help is generated from real bindings too (R7): submit/cancel/move-between-fields
	// are first-class, consistent actions, not ad-hoc per-form strings.
	actionFormSubmit
	actionFormCancel
	actionFormNextField
	actionFormPrevField
)

// keymap is the central registry: every action mapped to its one key.Binding. It
// replaces the per-screen keymaps that existed before 07-03; every screen now reads
// its keys from this one struct, so consistency is structural rather than a
// convention to remember.
type keymap struct {
	bindings map[keyAction]key.Binding
}

// binding returns the key.Binding for an action. Every action defined in
// defaultKeymap has a binding, so this never returns a zero binding for a known
// action; an unknown action returns a disabled binding so a mistaken lookup is inert
// rather than panicking.
func (k keymap) binding(a keyAction) key.Binding {
	if b, ok := k.bindings[a]; ok {
		return b
	}
	return key.NewBinding(key.WithDisabled())
}

// bindings resolves a list of actions to their key.Bindings, preserving order. It
// is how a screen turns its declared action subset into the concrete bindings the
// help renderer and key matching use — the one place actions become keys.
func (k keymap) resolve(actions []keyAction) []key.Binding {
	out := make([]key.Binding, 0, len(actions))
	for _, a := range actions {
		out = append(out, k.binding(a))
	}
	return out
}

// defaultKeymap builds the fixed default registry. Each binding carries BOTH its
// keys and its help text (key.WithHelp), so the help footer is generated from the
// very bindings that drive behavior — announced == real by construction. The key
// choices preserve the muscle memory established in 07-01/07-02 (esc=back, h=home,
// /=filter, ?=help, q=quit, enter=go-in), now centralized.
func defaultKeymap() keymap {
	b := map[keyAction]key.Binding{
		actionUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		actionDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		// enter (and l, vim-style) is the universal "go in": enter an area from the
		// root, open the selected entry inside an area.
		actionEnter: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter", "open"),
		),
		// esc (and backspace) is the universal "go back one level", identical on every
		// screen so the way out is always the same (R2).
		actionBack: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		// h jumps straight to the root from anywhere below it.
		actionHome: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "home"),
		),
		// r retries a failed load so an error state is never a dead end.
		actionRetry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		// / opens the list filter, the conventional search key.
		actionFilter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		// ? toggles between the short help bar and the expanded help view, the same
		// key in every context (R4) — including forms and the loading/empty/error
		// states (Check-8).
		actionHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		// q (and ctrl+c) quit. q is reserved inside sub-screens/forms so it is not
		// consumed while reading or typing; ctrl+c is the always-available escape hatch.
		actionQuit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		actionPageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup", "page up"),
		),
		actionPageDown: key.NewBinding(
			key.WithKeys("pgdown", "f", " "),
			key.WithHelp("pgdn", "page down"),
		),
		actionTop: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "top"),
		),
		actionBottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "bottom"),
		),

		// --- form actions ---
		// These mirror Huh's defaults so the form's real behavior and its announced
		// help agree (Huh submits/advances on enter, goes back a field on shift+tab),
		// while cancel reuses the shell's universal esc so leaving a form matches
		// leaving any other screen (R2).
		actionFormSubmit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		actionFormCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		actionFormNextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		actionFormPrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
	}
	return keymap{bindings: b}
}
