// Command oto is a lazygit-style terminal UI for continuous 1:1 note taking:
// add the people you meet with, start a new meeting at any time, jot notes,
// track a RAG (red/amber/green) status per session, and keep a running list
// of action items across everyone you meet with.
//
// Data is stored as plain markdown files under -dir (default
// ~/.onetoones or $ONETOONES_DIR), one folder per person and one file per
// meeting, so it stays readable, greppable, and easy to back up with git.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"oto/internal/store"
	"oto/internal/ui"
)

func defaultDir() string {
	if d := os.Getenv("ONETOONES_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".onetoones"
	}
	return filepath.Join(home, ".onetoones")
}

func main() {
	dir := flag.String("dir", defaultDir(), "directory to store people & meeting notes in")
	flag.Parse()

	s := store.New(*dir)
	if err := s.EnsureRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "oto: could not create data directory %s: %v\n", *dir, err)
		os.Exit(1)
	}

	app := ui.New(s)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "oto: %v\n", err)
		os.Exit(1)
	}
}
