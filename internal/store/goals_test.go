package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoalsMixedLongShort(t *testing.T) {
	lines := []string{
		"## Long-term",
		"- [ ] Get promoted to senior by end of year",
		"- [x] Learn Rust",
		"## Short-term",
		"- [ ] Finish onboarding new hire",
		"- [x] Ship the Q3 report",
	}
	goals := parseGoals(lines)
	if len(goals) != 4 {
		t.Fatalf("expected 4 goals, got %d: %+v", len(goals), goals)
	}

	want := []Goal{
		{Text: "Get promoted to senior by end of year", Term: "long", Done: false, LineNo: 1},
		{Text: "Learn Rust", Term: "long", Done: true, LineNo: 2},
		{Text: "Finish onboarding new hire", Term: "short", Done: false, LineNo: 4},
		{Text: "Ship the Q3 report", Term: "short", Done: true, LineNo: 5},
	}
	for i, g := range goals {
		if g != want[i] {
			t.Fatalf("goal %d = %+v, want %+v", i, g, want[i])
		}
	}
}

func TestAppendGoalCreatesSectionIfMissing(t *testing.T) {
	// Empty file: appending a long-term goal should create the heading.
	lines := appendGoal(nil, "long", "Get promoted to senior by end of year")
	want := []string{"## Long-term", "- [ ] Get promoted to senior by end of year"}
	if !equalLines(lines, want) {
		t.Fatalf("got %v, want %v", lines, want)
	}

	// Appending a short-term goal when only "## Long-term" exists should
	// create a new "## Short-term" section rather than touching the
	// existing one.
	lines = appendGoal(lines, "short", "Finish onboarding new hire")
	want = []string{
		"## Long-term",
		"- [ ] Get promoted to senior by end of year",
		"",
		"## Short-term",
		"- [ ] Finish onboarding new hire",
	}
	if !equalLines(lines, want) {
		t.Fatalf("got %v, want %v", lines, want)
	}

	// Appending a second long-term goal should insert it directly after
	// the existing long-term item, not at the end of the file.
	lines = appendGoal(lines, "long", "Learn Rust")
	want = []string{
		"## Long-term",
		"- [ ] Get promoted to senior by end of year",
		"- [ ] Learn Rust",
		"",
		"## Short-term",
		"- [ ] Finish onboarding new hire",
	}
	if !equalLines(lines, want) {
		t.Fatalf("got %v, want %v", lines, want)
	}
}

func TestToggleGoalLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"- [ ] do the thing", "- [x] do the thing"},
		{"- [x] done thing", "- [ ] done thing"},
		{"## Long-term", "## Long-term"},
	}
	for _, c := range cases {
		got := toggleGoalLine(c.in)
		if got != c.want {
			t.Errorf("toggleGoalLine(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAddGoalAndToggleGoalByLineNumber(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "Staff Engineer", "Platform", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	if err := s.AddGoal(p, "long", "Get promoted to senior by end of year"); err != nil {
		t.Fatalf("AddGoal long: %v", err)
	}
	if err := s.AddGoal(p, "short", "Finish onboarding new hire"); err != nil {
		t.Fatalf("AddGoal short: %v", err)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	p = people[0]
	if len(p.Goals) != 2 {
		t.Fatalf("expected 2 goals, got %d: %+v", len(p.Goals), p.Goals)
	}
	for _, g := range p.Goals {
		if g.Done {
			t.Fatalf("new goal should start open: %+v", g)
		}
	}

	var shortLineNo int
	for _, g := range p.Goals {
		if g.Term == "short" {
			shortLineNo = g.LineNo
		}
	}
	if err := s.ToggleGoal(p, shortLineNo); err != nil {
		t.Fatalf("ToggleGoal: %v", err)
	}

	people, _ = s.ListPeople()
	p = people[0]
	for _, g := range p.Goals {
		if g.Term == "short" && !g.Done {
			t.Fatalf("short-term goal should be done after toggle: %+v", g)
		}
		if g.Term == "long" && g.Done {
			t.Fatalf("long-term goal should remain open: %+v", g)
		}
	}
}

func TestGoalsRoundTripPreservesUntouchedLines(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p, err := s.CreatePerson("Jane Doe", "", "", 14)
	if err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}

	original := "## Long-term\n" +
		"- [ ] Get promoted to senior by end of year\n" +
		"## Short-term\n" +
		"- [ ] Finish onboarding new hire\n" +
		"- [x] Ship the Q3 report\n"
	if err := os.WriteFile(filepath.Join(p.Dir, "goals.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("seed goals.md: %v", err)
	}

	people, err := s.ListPeople()
	if err != nil || len(people) != 1 {
		t.Fatalf("ListPeople: %v people=%v", err, people)
	}
	p = people[0]

	var toggleLineNo int
	for _, g := range p.Goals {
		if g.Text == "Finish onboarding new hire" {
			toggleLineNo = g.LineNo
		}
	}
	if err := s.ToggleGoal(p, toggleLineNo); err != nil {
		t.Fatalf("ToggleGoal: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(p.Dir, "goals.md"))
	if err != nil {
		t.Fatalf("read goals.md: %v", err)
	}

	want := "## Long-term\n" +
		"- [ ] Get promoted to senior by end of year\n" +
		"## Short-term\n" +
		"- [x] Finish onboarding new hire\n" +
		"- [x] Ship the Q3 report\n"
	if string(raw) != want {
		t.Fatalf("round-trip mismatch:\ngot:\n%q\nwant:\n%q", string(raw), want)
	}
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
