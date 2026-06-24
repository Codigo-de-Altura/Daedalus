package tui

import "github.com/charmbracelet/huh"

// DefaultFormTheme returns the Huh theme reserved for the interactive forms
// that later epics (agent and prompt editors, the init/build wizards) build on
// top of this skeleton. It lives in the foundations so the Huh dependency is
// fixed from the start and every form shares one consistent look.
func DefaultFormTheme() *huh.Theme {
	return huh.ThemeCharm()
}
