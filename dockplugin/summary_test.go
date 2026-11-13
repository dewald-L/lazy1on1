package dockplugin

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dewald-L/dock/plugin"

	"github.com/dewald-L/lazy1on1/internal/store"
)

// TestMain points HOME and XDG_CONFIG_HOME at a throwaway directory; every
// test uses its own data directory, never the real 1:1 notes.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "lazy1on1-dockplugin-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.Unsetenv("ONETOONES_DIR")
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

func at(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 10, 0, 0, 0, time.Local) }

func meeting(t *testing.T, s *store.Store, p store.Person, when time.Time, open, done int) {
	t.Helper()
	m, err := s.CreateMeeting(p, when)
	if err != nil {
		t.Fatal(err)
	}
	for i := range open + done {
		if err := s.AddActionItem(&m, "item"); err != nil {
			t.Fatal(err)
		}
		if i >= open {
			if err := s.ToggleActionItem(&m, m.ActionItems[len(m.ActionItems)-1].LineNo); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func person(t *testing.T, s *store.Store, name string, cadence int) store.Person {
	t.Helper()
	p, err := s.CreatePerson(name, "", "", cadence)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSummaryNoPeople(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	s, err := Plugin{}.Summary(plugin.Context{DataDir: dir})
	if err != nil || len(s.Lines) != 1 || s.Lines[0] != "No people yet" {
		t.Fatalf("empty data dir: %q %v", s.Lines, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("Summary must not create the data directory")
	}
}

func TestSummary(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	ana := person(t, s, "Ana", 7)
	ben := person(t, s, "Ben", 14)
	meeting(t, s, ana, at(2026, 10, 27), 1, 0)
	meeting(t, s, ana, at(2026, 11, 3), 2, 1)
	meeting(t, s, ben, at(2026, 11, 10), 1, 2)

	cases := []struct {
		now   time.Time
		next  string
		level plugin.Level
	}{
		{at(2026, 11, 13), "Next 1:1: Ana, 3 days overdue", plugin.LevelWarn},
		{at(2026, 11, 11), "Next 1:1: Ana, 1 day overdue", plugin.LevelWarn},
		{at(2026, 11, 10), "Next 1:1: Ana, due today", plugin.LevelNone},
		{at(2026, 11, 9), "Next 1:1: Ana, due tomorrow", plugin.LevelNone},
		{at(2026, 11, 5), "Next 1:1: Ana, due in 5 days", plugin.LevelNone},
	}
	for _, c := range cases {
		got, err := summary(s, c.now)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"4 open action items · 2 people", c.next}
		if strings.Join(got.Lines, "|") != strings.Join(want, "|") || got.Level != c.level {
			t.Errorf("%s: got %q %v, want %q %v", c.now.Format("Jan 2"), got.Lines, got.Level, want, c.level)
		}
		if clean := got.Clean(); len(clean.Lines) != len(got.Lines) {
			t.Errorf("summary must be plain text: %q", got.Lines)
		}
	}

	// someone never met is due now, ahead of anyone not yet due
	person(t, s, "Carl", 14)
	if got, _ := summary(s, at(2026, 11, 5)); got.Lines[1] != "Next 1:1: Carl, first one" || got.Lines[0] != "4 open action items · 3 people" {
		t.Errorf("new person: %q", got.Lines)
	}
	// but not ahead of someone overdue
	if got, _ := summary(s, at(2026, 11, 13)); got.Lines[1] != "Next 1:1: Ana, 3 days overdue" {
		t.Errorf("overdue beats new: %q", got.Lines)
	}
}

func TestSummaryCounts(t *testing.T) {
	s := store.New(t.TempDir())
	p := person(t, s, "Solo", 14)
	meeting(t, s, p, at(2026, 11, 12), 0, 2)
	if got, _ := summary(s, at(2026, 11, 13)); got.Lines[0] != "No open action items · 1 person" {
		t.Errorf("none open: %q", got.Lines)
	}
	meeting(t, s, p, at(2026, 11, 13), 1, 0)
	if got, _ := summary(s, at(2026, 11, 13)); got.Lines[0] != "1 open action item · 1 person" || got.Lines[1] != "Next 1:1: Solo, due in 14 days" {
		t.Errorf("one open: %q", got.Lines)
	}
}

func TestSummaryDirAndConcurrency(t *testing.T) {
	ctxDir, ownDir := t.TempDir(), t.TempDir()
	person(t, store.New(ownDir), "Dana", 14)
	// Plugin.Dir (standalone) wins over dock's data dir
	got, err := Plugin{Dir: ownDir}.Summary(plugin.Context{DataDir: ctxDir})
	if err != nil || got.Lines[1] != "Next 1:1: Dana, first one" {
		t.Fatalf("Dir override: %q %v", got.Lines, err)
	}
	if got, _ := (Plugin{}).Summary(plugin.Context{DataDir: ctxDir}); got.Lines[0] != "No people yet" {
		t.Fatalf("dock data dir: %q", got.Lines)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := (Plugin{Dir: ownDir}).Summary(plugin.Context{}); err != nil || len(got.Lines) != 2 {
				t.Errorf("concurrent: %q %v", got.Lines, err)
			}
		}()
	}
	wg.Wait()
}
