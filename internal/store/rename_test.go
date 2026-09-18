package store

import (
	"testing"
	"time"
)

func TestRenamePerson(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "Staff Engineer", "Platform", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	origSlug, origDir := p.Slug, p.Dir

	if err := s.RenamePerson(p, "Jane Smith"); err != nil {
		t.Fatalf("RenamePerson: %v", err)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	got := people[0]
	if got.Name != "Jane Smith" {
		t.Fatalf("expected renamed person, got %+v", got)
	}
	// The slug/directory (and thus every file path derived from it) must
	// stay put - only the display name changes.
	if got.Slug != origSlug || got.Dir != origDir {
		t.Fatalf("expected slug/dir unchanged, got slug=%q dir=%q", got.Slug, got.Dir)
	}
	// Role/team/cadence must round-trip untouched.
	if got.Role != "Staff Engineer" || got.Team != "Platform" || got.CadenceDays != 14 {
		t.Fatalf("expected other fields preserved, got %+v", got)
	}
}

func TestRenameActionItem(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	m, err := s.CreateMeeting(p, time.Now())
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	if err := s.AddActionItem(&m, "old text"); err != nil {
		t.Fatalf("AddActionItem: %v", err)
	}
	item := m.ActionItems[0]
	if err := s.ToggleActionItem(&m, item.LineNo); err != nil {
		t.Fatalf("ToggleActionItem: %v", err)
	}

	if err := s.RenameActionItem(&m, item.LineNo, "new text"); err != nil {
		t.Fatalf("RenameActionItem: %v", err)
	}
	if len(m.ActionItems) != 1 || m.ActionItems[0].Text != "new text" {
		t.Fatalf("expected renamed item, got %+v", m.ActionItems)
	}
	if !m.ActionItems[0].Done {
		t.Fatalf("expected done state preserved across rename, got %+v", m.ActionItems[0])
	}

	reloaded, err := s.loadMeeting(p.Slug, m.Path)
	if err != nil {
		t.Fatalf("loadMeeting: %v", err)
	}
	if len(reloaded.ActionItems) != 1 || reloaded.ActionItems[0].Text != "new text" || !reloaded.ActionItems[0].Done {
		t.Fatalf("rename not persisted, got %+v", reloaded.ActionItems)
	}
}

func TestRenameGoal(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	if err := s.AddGoal(p, "short", "old goal text"); err != nil {
		t.Fatalf("AddGoal: %v", err)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	p = people[0]
	goal := p.Goals[0]
	if err := s.ToggleGoal(p, goal.LineNo); err != nil {
		t.Fatalf("ToggleGoal: %v", err)
	}

	if err := s.RenameGoal(p, goal.LineNo, "new goal text"); err != nil {
		t.Fatalf("RenameGoal: %v", err)
	}

	people, err = s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	got := people[0].Goals[0]
	if got.Text != "new goal text" {
		t.Fatalf("expected renamed goal, got %+v", got)
	}
	if got.Term != "short" || !got.Done {
		t.Fatalf("expected term/done state preserved across rename, got %+v", got)
	}
}
