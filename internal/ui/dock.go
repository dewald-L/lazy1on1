package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dewald-L/dock/plugin"
)

// Integration with the dock host (github.com/dewald-L/dock). These methods
// implement dock's optional plugin interfaces; standalone runs ignore them.

// commandMsg is delivered when a palette command is picked in dock.
type commandMsg struct{ name string }

// Commands implements plugin.Commander.
func (a *App) Commands() []plugin.Command {
	return []plugin.Command{
		{Name: "new person", Desc: "add someone you meet with", Msg: commandMsg{"person"}},
		{Name: "new meeting", Desc: "start a 1:1 with the selected person now", Msg: commandMsg{"meeting"}},
		{Name: "new action item", Desc: "on the selected meeting", Msg: commandMsg{"action"}},
		{Name: "keybindings", Desc: "full help screen", Msg: commandMsg{"help"}},
	}
}

func (a *App) runCommand(c commandMsg) (tea.Model, tea.Cmd) {
	if a.modal != modalNone && a.modal != modalHelp && a.modal != modalPeople {
		a.status = "finish the open dialog first"
		return a, nil
	}
	a.closePeople()
	switch c.name {
	case "person":
		a.peopleReturn = false
		return a.startNewPerson()
	case "meeting":
		if a.currentPerson() == nil {
			a.status = "select a person first"
			return a, nil
		}
		a.setFocus(panelMeetings)
		return a.newForFocus()
	case "action":
		if a.currentMeeting() == nil {
			a.status = "select a meeting first"
			return a, nil
		}
		a.setFocus(panelDetail)
		return a.newForFocus()
	case "help":
		a.modal = modalHelp
	}
	return a, nil
}

// Help implements plugin.Helper: the same contextual hints as the
// standalone help bar.
func (a *App) Help() []plugin.KeyHelp {
	pairs := a.helpPairs()
	out := make([]plugin.KeyHelp, len(pairs))
	for i, p := range pairs {
		out[i] = plugin.KeyHelp{Key: p.k, Desc: p.v}
	}
	return out
}
