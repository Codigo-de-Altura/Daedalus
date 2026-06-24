// Package tui contains the Bubble Tea bootstrap for Daedalus.
//
// This is the minimal Elm-architecture skeleton established by epic-00: it
// starts, renders a minimal identifiable Daedalus view, and quits cleanly on
// "q" or Ctrl+C. Product screens, navigation, and styling live in later epics
// (epic-07). Keeping the skeleton here lets the foundations prove the Charm
// stack is wired correctly without committing to any product UX.
package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// welcome is the minimal, identifiable Daedalus view rendered by the skeleton.
const welcome = "# Daedalus\n\nBackend-agnostic scaffolding for your project's AI structure.\n"

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))

// keyMap declares the key bindings the skeleton understands. It implements
// help.KeyMap so the footer help is generated from the bindings themselves.
type keyMap struct {
	Quit key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding { return []key.Binding{k.Quit} }

func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.Quit}} }

// Model is the minimal Bubble Tea model for the Daedalus skeleton.
type Model struct {
	keys keyMap
	help help.Model
}

// New returns an initialized skeleton Model.
func New() Model {
	return Model{keys: defaultKeyMap(), help: help.New()}
}

// Init implements tea.Model. The skeleton has no startup command.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model. It only handles the quit binding; everything
// else is a no-op until later epics add real behavior.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keys.Quit) {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model. It renders the welcome markdown with Glamour when
// possible, falling back to the raw text so the view never breaks.
func (m Model) View() string {
	body := welcome
	if rendered, err := glamour.Render(welcome, "dark"); err == nil {
		body = rendered
	}
	return titleStyle.Render("Daedalus") + "\n" + body + "\n" + m.help.View(m.keys)
}
