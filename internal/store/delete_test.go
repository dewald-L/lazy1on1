package store

import (
	"os"
	"testing"
	"time"
)

func TestDeletePerson(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "Staff Engineer", "Platform", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	if _, err := s.CreateMeeting(p, time.Now()); err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}

	if err := s.DeletePerson(p); err != nil {
		t.Fatalf("DeletePerson: %v", err)
	}
	if _, err := os.Stat(p.Dir); !os.IsNotExist(err) {
		t.Fatalf("expected person dir to be removed, stat err = %v", err)
	}

	people, err := s.ListPeople()
	if err != nil {
		t.Fatalf("ListPeople: %v", err)
	}
	if len(people) != 0 {
		t.Fatalf("expected no people after delete, got %+v", people)
	}
}

func TestDeleteMeeting(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	keep, err := s.CreateMeeting(p, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	gone, err := s.CreateMeeting(p, time.Now())
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}

	if err := s.DeleteMeeting(gone); err != nil {
		t.Fatalf("DeleteMeeting: %v", err)
	}
	if _, err := os.Stat(gone.Path); !os.IsNotExist(err) {
		t.Fatalf("expected meeting file to be removed, stat err = %v", err)
	}

	meetings, err := s.ListMeetings(p)
	if err != nil {
		t.Fatalf("ListMeetings: %v", err)
	}
	if len(meetings) != 1 || meetings[0].Path != keep.Path {
		t.Fatalf("expected only the kept meeting to remain, got %+v", meetings)
	}
}

func TestDeleteActionItem(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	m, err := s.CreateMeeting(p, time.Now())
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	if err := s.AddActionItem(&m, "keep me"); err != nil {
		t.Fatalf("AddActionItem: %v", err)
	}
	if err := s.AddActionItem(&m, "delete me"); err != nil {
		t.Fatalf("AddActionItem: %v", err)
	}
	if len(m.ActionItems) != 2 {
		t.Fatalf("expected 2 action items, got %+v", m.ActionItems)
	}

	// Find "delete me" by text rather than assuming ordering, then delete it.
	var target ActionItem
	for _, it := range m.ActionItems {
		if it.Text == "delete me" {
			target = it
		}
	}
	if err := s.DeleteActionItem(&m, target.LineNo); err != nil {
		t.Fatalf("DeleteActionItem: %v", err)
	}
	if len(m.ActionItems) != 1 || m.ActionItems[0].Text != "keep me" {
		t.Fatalf("expected only 'keep me' to remain, got %+v", m.ActionItems)
	}

	reloaded, err := s.loadMeeting(p.Slug, m.Path)
	if err != nil {
		t.Fatalf("loadMeeting: %v", err)
	}
	if len(reloaded.ActionItems) != 1 || reloaded.ActionItems[0].Text != "keep me" {
		t.Fatalf("deletion not persisted, got %+v", reloaded.ActionItems)
	}
}

func TestDeleteGoal(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	if err := s.AddGoal(p, "long", "keep me"); err != nil {
		t.Fatalf("AddGoal: %v", err)
	}
	if err := s.AddGoal(p, "long", "delete me"); err != nil {
		t.Fatalf("AddGoal: %v", err)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	p = people[0]
	if len(p.Goals) != 2 {
		t.Fatalf("expected 2 goals, got %+v", p.Goals)
	}

	var target Goal
	for _, g := range p.Goals {
		if g.Text == "delete me" {
			target = g
		}
	}
	if err := s.DeleteGoal(p, target.LineNo); err != nil {
		t.Fatalf("DeleteGoal: %v", err)
	}

	people, err = s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	if len(people[0].Goals) != 1 || people[0].Goals[0].Text != "keep me" {
		t.Fatalf("expected only 'keep me' to remain, got %+v", people[0].Goals)
	}
}
