// Command lazy1on1 is a lazygit-style terminal UI for continuous 1:1 note taking:
// add the people you meet with, start a new meeting at any time, jot notes,
// track a RAG (red/amber/green) status per session, and keep a running list
// of action items across everyone you meet with.
//
// Data is stored as plain markdown files under -dir (default: a "data"
// folder next to the lazy1on1 binary, or $ONETOONES_DIR), one folder per person
// and one file per meeting, so it stays readable, greppable, and easy to
// back up with git.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"lazy1on1/internal/store"
	"lazy1on1/internal/ui"
)

func defaultDir() string {
	if d := os.Getenv("ONETOONES_DIR"); d != "" {
		return d
	}
	exe, err := os.Executable()
	if err != nil {
		return "data"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), "data")
}

func main() {
	dir := flag.String("dir", defaultDir(), "directory to store people & meeting notes in")
	flag.Parse()

	s := store.New(*dir)
	if err := s.EnsureRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "lazy1on1: could not create data directory %s: %v\n", *dir, err)
		os.Exit(1)
	}

	app := ui.New(s)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazy1on1: %v\n", err)
		os.Exit(1)
	}
}
