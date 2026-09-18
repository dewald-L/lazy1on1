package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreatePersonAndMeetingRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "Staff Engineer", "Platform", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	if p.Slug != "jane-doe" {
		t.Fatalf("expected slug jane-doe, got %q", p.Slug)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	if people[0].Role != "Staff Engineer" {
		t.Fatalf("role not persisted: %+v", people[0])
	}

	at := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	m, err := s.CreateMeeting(p, at)
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}
	if filepath.Base(m.Path) != "2026-09-17-1000.md" {
		t.Fatalf("unexpected filename: %s", filepath.Base(m.Path))
	}

	meetings, err := s.ListMeetings(p)
	if err != nil || len(meetings) != 1 {
		t.Fatalf("ListMeetings: %v meetings=%v", err, meetings)
	}

	if err := s.AddActionItem(&m, "Follow up on promo doc"); err != nil {
		t.Fatalf("AddActionItem: %v", err)
	}
	if len(m.ActionItems) != 1 || m.ActionItems[0].Text != "Follow up on promo doc" {
		t.Fatalf("action item not parsed: %+v", m.ActionItems)
	}
	if m.ActionItems[0].Done {
		t.Fatalf("new action item should start open")
	}

	if err := s.ToggleActionItem(&m, m.ActionItems[0].LineNo); err != nil {
		t.Fatalf("ToggleActionItem: %v", err)
	}
	if !m.ActionItems[0].Done {
		t.Fatalf("action item should be done after toggle")
	}
	if len(m.OpenActionItems()) != 0 {
		t.Fatalf("expected no open action items after toggle")
	}

	if err := s.SetRAG(&m, RAGGreen); err != nil {
		t.Fatalf("SetRAG: %v", err)
	}

	// Reload from disk to make sure everything round-trips through the file.
	reloaded, err := s.loadMeeting(p.Slug, m.Path)
	if err != nil {
		t.Fatalf("loadMeeting: %v", err)
	}
	if reloaded.RAG != RAGGreen {
		t.Fatalf("RAG not persisted, got %q", reloaded.RAG)
	}
	if !reloaded.Date.Equal(at) {
		t.Fatalf("date not persisted: got %v want %v", reloaded.Date, at)
	}
	if len(reloaded.ActionItems) != 1 || !reloaded.ActionItems[0].Done {
		t.Fatalf("action item state not persisted: %+v", reloaded.ActionItems)
	}
}

func TestToggleActionItemLineFormat(t *testing.T) {
	cases := []struct{ in, want string }{
		{"- [ ] do the thing", "- [x] do the thing"},
		{"- [x] done thing", "- [ ] done thing"},
		{"  - [ ] indented", "  - [x] indented"},
		{"not a checklist line", "not a checklist line"},
	}
	for _, c := range cases {
		got := toggleLine(c.in)
		if got != c.want {
			t.Errorf("toggleLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRAGNextCycle(t *testing.T) {
	seq := []RAG{RAGNone, RAGGreen, RAGAmber, RAGRed, RAGNone}
	cur := RAGNone
	for i := 1; i < len(seq); i++ {
		cur = cur.Next()
		if cur != seq[i] {
			t.Fatalf("step %d: got %q want %q", i, cur, seq[i])
		}
	}
}

func TestPerceivedPulseNextCycle(t *testing.T) {
	seq := []PerceivedPulse{PerceivedPulseNone, PerceivedPulseDown, PerceivedPulseSteady, PerceivedPulseUp, PerceivedPulseNone}
	cur := PerceivedPulseNone
	for i := 1; i < len(seq); i++ {
		cur = cur.Next()
		if cur != seq[i] {
			t.Fatalf("step %d: got %q want %q", i, cur, seq[i])
		}
	}
}

func TestAddWinRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, _ := s.CreatePerson("Jane Doe", "", "", 14)
	m, err := s.CreateMeeting(p, time.Now())
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}

	if err := s.AddWin(&m, "Shipped the migration"); err != nil {
		t.Fatalf("AddWin: %v", err)
	}
	if err := s.AddWin(&m, "Positive design review"); err != nil {
		t.Fatalf("AddWin: %v", err)
	}
	if err := s.SetPerceivedPulse(&m, PerceivedPulseUp); err != nil {
		t.Fatalf("SetPerceivedPulse: %v", err)
	}

	reloaded, err := s.loadMeeting(p.Slug, m.Path)
	if err != nil {
		t.Fatalf("loadMeeting: %v", err)
	}
	if reloaded.PerceivedPulse != PerceivedPulseUp {
		t.Fatalf("pulse not persisted, got %q", reloaded.PerceivedPulse)
	}

	body := reloaded.Body()
	if !strings.Contains(body, "## Wins") {
		t.Fatalf("Wins heading missing from body:\n%s", body)
	}
	if !strings.Contains(body, "- Shipped the migration") || !strings.Contains(body, "- Positive design review") {
		t.Fatalf("win bullets missing from body:\n%s", body)
	}
	// The Wins section must land above Action Items, so it's still inside
	// what the Detail panel's notes viewport renders.
	if strings.Index(body, "## Wins") > strings.Index(body, "## Action Items") {
		t.Fatalf("Wins heading should come before Action Items:\n%s", body)
	}
}

func TestGlobalNotesRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "", "", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	// A person with no notes.md yet should read back as empty, not error.
	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	if people[0].Notes != "" {
		t.Fatalf("expected no notes yet, got %q", people[0].Notes)
	}

	if err := s.UpdateNotes(&p, "Prefers async updates.\n\nCareer goal: staff engineer."); err != nil {
		t.Fatalf("UpdateNotes: %v", err)
	}

	people, err = s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	if people[0].Notes != "Prefers async updates.\n\nCareer goal: staff engineer." {
		t.Fatalf("notes not persisted, got %q", people[0].Notes)
	}

	// notes.md must not be mistaken for a meeting file.
	meetings, err := s.ListMeetings(people[0])
	if err != nil {
		t.Fatalf("ListMeetings: %v", err)
	}
	if len(meetings) != 0 {
		t.Fatalf("expected notes.md to be excluded from meetings, got %+v", meetings)
	}
}

func TestAllActionItemsAcrossPeople(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	jane, _ := s.CreatePerson("Jane Doe", "", "", 14)
	john, _ := s.CreatePerson("John Smith", "", "", 14)

	m1, _ := s.CreateMeeting(jane, time.Now().Add(-48*time.Hour))
	s.AddActionItem(&m1, "Jane task 1")

	m2, _ := s.CreateMeeting(john, time.Now())
	s.AddActionItem(&m2, "John task 1")
	s.AddActionItem(&m2, "John task 2")
	s.ToggleActionItem(&m2, m2.ActionItems[1].LineNo) // close "John task 2"

	items, err := s.AllActionItems()
	if err != nil {
		t.Fatalf("AllActionItems: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items (open and closed), got %d: %+v", len(items), items)
	}
	// Most recent meeting (John's) should come first.
	if items[0].Person.Slug != "john-smith" {
		t.Fatalf("expected john-smith first, got %s", items[0].Person.Slug)
	}
}
