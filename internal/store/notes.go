package store

import (
	"os"
	"path/filepath"
	"strings"
)

const notesFileName = "notes.md"

// loadNotesFile reads a person's persistent global notes (notes.md) - free-
// form prose that isn't tied to any single meeting, unlike a Meeting's
// BodyLines. Returns "" if the file doesn't exist yet.
func (s *Store) loadNotesFile(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, notesFileName))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(raw), "\n")
}

// NotesPath returns the absolute path to a person's global notes file, for
// opening it directly in $EDITOR.
func (s *Store) NotesPath(p Person) string {
	return filepath.Join(p.Dir, notesFileName)
}

// SaveNotes writes a person's global notes back to notes.md.
func (s *Store) SaveNotes(p *Person) error {
	body := p.Notes
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(filepath.Join(p.Dir, notesFileName), []byte(body), 0o644)
}

// UpdateNotes replaces a person's global notes and persists them, mirroring
// Store.UpdateBody for meeting notes.
func (s *Store) UpdateNotes(p *Person, newBody string) error {
	p.Notes = strings.TrimRight(newBody, "\n")
	return s.SaveNotes(p)
}

// ReloadNotes re-reads a person's notes.md from disk (e.g. after external
// $EDITOR use), mirroring Store.Reload for meetings.
func (s *Store) ReloadNotes(p *Person) error {
	p.Notes = s.loadNotesFile(p.Dir)
	return nil
}
