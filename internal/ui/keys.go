package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap only covers bindings that are genuinely global (same action
// regardless of which panel has focus). Contextual actions (e.g. "new" per
// panel) live in update.go's per-panel handlers and newForFocus instead —
// they have no single binding to describe here.
type keyMap struct {
	Up, Down     key.Binding
	Left, Right  key.Binding
	Tab          key.Binding
	FocusPanel   key.Binding
	Edit         key.Binding
	EditExternal key.Binding
	Rename       key.Binding
	CycleRAG     key.Binding
	ToggleItem   key.Binding
	Help         key.Binding
	Quit         key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
}

var keys = keyMap{
	Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:         key.NewBinding(key.WithKeys("left", "h", "esc"), key.WithHelp("←/h/esc", "back")),
	Right:        key.NewBinding(key.WithKeys("right", "l", "enter"), key.WithHelp("→/l/enter", "drill in")),
	Tab:          key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	FocusPanel:   key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6"), key.WithHelp("1/2/3/4/5/6", "jump to panel")),
	Edit:         key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit notes")),
	EditExternal: key.NewBinding(key.WithKeys("E"), key.WithHelp("E", "edit in $EDITOR")),
	Rename:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
	CycleRAG:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "cycle RAG")),
	ToggleItem:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle item")),
	Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Confirm:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Cancel:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}
