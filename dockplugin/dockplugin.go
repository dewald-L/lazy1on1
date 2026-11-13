// Package dockplugin exposes lazy1on1 as a plugin for the dock terminal
// host (github.com/dewald-L/dock). The same plugin also runs standalone:
// see main.go, which uses plugin.RunStandalone.
package dockplugin

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dewald-L/dock/plugin"

	"github.com/dewald-L/lazy1on1/internal/store"
	"github.com/dewald-L/lazy1on1/internal/ui"
)

// Importing this package registers lazy1on1 with dock:
//
//	dock plugin install github.com/dewald-L/lazy1on1/dockplugin
func init() { plugin.Register(Plugin{}) }

// Plugin is lazy1on1 as a dock plugin.
type Plugin struct {
	// Dir overrides the data directory. Empty means dock's plugin data
	// directory (ctx.DataDir, ~/.local/share/dock/lazy1on1).
	Dir string
	// Standalone keeps lazy1on1's own status and help bars (used by main).
	Standalone bool
}

func (Plugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:          "lazy1on1",
		Name:        "lazy1on1",
		Description: "Continuous 1:1 notes: people, meetings, goals, action items",
		Version:     "0.2.0",
		Display:     plugin.DisplayFull,
		Lifecycle:   plugin.Persistent,
		MinWidth:    70,
		MinHeight:   24,
	}
}

func (p Plugin) New(ctx plugin.Context) (tea.Model, error) {
	dir := p.Dir
	if dir == "" {
		dir = ctx.DataDir
	}
	if dir == "" {
		return nil, fmt.Errorf("no data directory")
	}
	s := store.New(dir)
	if err := s.EnsureRoot(); err != nil {
		return nil, fmt.Errorf("could not create data directory %s: %w", dir, err)
	}
	app := ui.New(s)
	app.SetEmbedded(!p.Standalone)
	return app, nil
}
