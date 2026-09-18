// Package store persists people and 1:1 meeting notes as plain markdown
// files on disk, so the data stays human-readable, greppable, and
// git-friendly outside of this tool.
//
// Layout:
//
//	<root>/
//	  <person-slug>/
//	    person.yaml            # name / role / team / cadence
//	    goals.md               # long-term / short-term goal checklist
//	    notes.md               # free-form global notes, not tied to a meeting
//	    2026-09-17-1000.md      # one file per meeting
//	    2026-09-30-1000.md
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02-1504"
const timeLayoutRFC = time.RFC3339

// Store is a handle to a root directory of people/meetings.
type Store struct {
	Root string
}

func New(root string) *Store {
	return &Store{Root: root}
}

func (s *Store) EnsureRoot() error {
	return os.MkdirAll(s.Root, 0o755)
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9-]+`)

func Slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonSlugChars.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "person"
	}
	return slug
}

// uniqueSlug appends -2, -3, ... if the base slug's directory already exists.
func (s *Store) uniqueSlug(base string) string {
	slug := base
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(s.Root, slug)); os.IsNotExist(err) {
			return slug
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

// ListPeople returns every person under Root, sorted by name.
func (s *Store) ListPeople() ([]Person, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var people []Person
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(s.Root, e.Name())
		p := s.readPersonMeta(dir, e.Name())
		people = append(people, p)
	}
	sort.Slice(people, func(i, j int) bool {
		return strings.ToLower(people[i].Name) < strings.ToLower(people[j].Name)
	})
	return people, nil
}

func (s *Store) readPersonMeta(dir, slug string) Person {
	p := Person{Slug: slug, Name: slug, CadenceDays: 14, Dir: dir}
	if raw, err := os.ReadFile(filepath.Join(dir, "person.yaml")); err == nil {
		fm, _ := splitFrontMatter("---\n" + string(raw) + "\n---\n")
		if v, ok := fm["name"]; ok && v != "" {
			p.Name = v
		}
		if v, ok := fm["role"]; ok {
			p.Role = v
		}
		if v, ok := fm["team"]; ok {
			p.Team = v
		}
		if v, ok := fm["cadence_days"]; ok {
			if n, err := strconv.Atoi(v); err == nil {
				p.CadenceDays = n
			}
		}
	}
	p.GoalLines, p.Goals = s.loadGoalsFile(dir)
	p.Notes = s.loadNotesFile(dir)
	return p
}

// loadGoalsFile reads and parses a person's goals.md, returning nil, nil if
// it doesn't exist yet.
func (s *Store) loadGoalsFile(dir string) ([]string, []Goal) {
	raw, err := os.ReadFile(filepath.Join(dir, "goals.md"))
	if err != nil {
		return nil, nil
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	return lines, parseGoals(lines)
}

func (s *Store) writePersonMeta(p Person) error {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", p.Name)
	fmt.Fprintf(&b, "role: %s\n", p.Role)
	fmt.Fprintf(&b, "team: %s\n", p.Team)
	fmt.Fprintf(&b, "cadence_days: %d\n", p.CadenceDays)
	return os.WriteFile(filepath.Join(p.Dir, "person.yaml"), []byte(b.String()), 0o644)
}

// CreatePerson creates a new person folder + metadata file.
func (s *Store) CreatePerson(name, role, team string, cadenceDays int) (Person, error) {
	if err := s.EnsureRoot(); err != nil {
		return Person{}, err
	}
	slug := s.uniqueSlug(Slugify(name))
	dir := filepath.Join(s.Root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Person{}, err
	}
	p := Person{Slug: slug, Name: name, Role: role, Team: team, CadenceDays: cadenceDays, Dir: dir}
	if p.CadenceDays <= 0 {
		p.CadenceDays = 14
	}
	if err := s.writePersonMeta(p); err != nil {
		return Person{}, err
	}
	return p, nil
}

// ListMeetings returns every meeting for a person, newest first.
func (s *Store) ListMeetings(person Person) ([]Meeting, error) {
	entries, err := os.ReadDir(person.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meetings []Meeting
	for _, e := range entries {
		if e.IsDir() || e.Name() == "person.yaml" || e.Name() == "goals.md" || e.Name() == notesFileName || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(person.Dir, e.Name())
		m, err := s.loadMeeting(person.Slug, path)
		if err != nil {
			continue
		}
		meetings = append(meetings, m)
	}
	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].Date.After(meetings[j].Date)
	})
	return meetings, nil
}

func (s *Store) loadMeeting(personSlug, path string) (Meeting, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Meeting{}, err
	}
	fm, body := splitFrontMatter(string(raw))

	m := Meeting{PersonSlug: personSlug, Path: path}

	if v, ok := fm["date"]; ok {
		if t, err := time.Parse(timeLayoutRFC, v); err == nil {
			m.Date = t
		}
	}
	if m.Date.IsZero() {
		// Fall back to the filename, e.g. 2026-09-17-1000.md
		base := strings.TrimSuffix(filepath.Base(path), ".md")
		if t, err := time.ParseInLocation(dateLayout, base, time.Local); err == nil {
			m.Date = t
		}
	}
	m.RAG = RAG(fm["rag"])
	m.PerceivedPulse = PerceivedPulse(fm["pulse"])
	m.Tags = splitList(fm["tags"])
	m.BodyLines = strings.Split(strings.TrimRight(body, "\n"), "\n")
	m.ActionItems = parseActionItems(m.BodyLines)
	return m, nil
}

// DeletePerson permanently removes a person's entire directory: every
// meeting file alongside person.yaml, goals.md, and notes.md.
func (s *Store) DeletePerson(p Person) error {
	return os.RemoveAll(p.Dir)
}

// RenamePerson updates a person's display name in person.yaml. Their
// directory and slug (and every file path derived from it) are left
// unchanged - this only changes what's shown on screen.
func (s *Store) RenamePerson(p Person, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("name cannot be empty")
	}
	p.Name = newName
	return s.writePersonMeta(p)
}

// CreateMeeting creates a new dated meeting file for a person and returns it.
func (s *Store) CreateMeeting(person Person, at time.Time) (Meeting, error) {
	if err := os.MkdirAll(person.Dir, 0o755); err != nil {
		return Meeting{}, err
	}
	fname := at.Format(dateLayout) + ".md"
	path := filepath.Join(person.Dir, fname)
	// Avoid clobbering an existing meeting created in the same minute.
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		fname = fmt.Sprintf("%s-%d.md", at.Format(dateLayout), i)
		path = filepath.Join(person.Dir, fname)
	}

	m := Meeting{
		PersonSlug:     person.Slug,
		Path:           path,
		Date:           at,
		RAG:            RAGNone,
		PerceivedPulse: PerceivedPulseNone,
		BodyLines:      []string{"## Notes", "", "", "## Wins", "", "## Action Items", ""},
	}
	m.ActionItems = parseActionItems(m.BodyLines)
	if err := s.SaveMeeting(&m); err != nil {
		return Meeting{}, err
	}
	return m, nil
}

// SaveMeeting serializes front matter + body back to disk.
func (s *Store) SaveMeeting(m *Meeting) error {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "date: %s\n", m.Date.Format(timeLayoutRFC))
	fmt.Fprintf(&b, "rag: %s\n", string(m.RAG))
	fmt.Fprintf(&b, "pulse: %s\n", string(m.PerceivedPulse))
	fmt.Fprintf(&b, "tags: %s\n", joinList(m.Tags))
	b.WriteString("---\n")
	b.WriteString(m.Body())
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	return os.WriteFile(m.Path, []byte(b.String()), 0o644)
}

// DeleteMeeting permanently removes a single meeting's markdown file.
func (s *Store) DeleteMeeting(m Meeting) error {
	return os.Remove(m.Path)
}

// SetRAG updates and persists a meeting's RAG status.
func (s *Store) SetRAG(m *Meeting, rag RAG) error {
	m.RAG = rag
	return s.SaveMeeting(m)
}

// SetPerceivedPulse updates and persists a meeting's perceived-pulse status.
func (s *Store) SetPerceivedPulse(m *Meeting, pulse PerceivedPulse) error {
	m.PerceivedPulse = pulse
	return s.SaveMeeting(m)
}

// ToggleActionItem flips a checklist line's done state and persists it.
func (s *Store) ToggleActionItem(m *Meeting, lineNo int) error {
	if lineNo < 0 || lineNo >= len(m.BodyLines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	m.BodyLines[lineNo] = toggleLine(m.BodyLines[lineNo])
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// AddActionItem appends a new open action item to the meeting notes.
func (s *Store) AddActionItem(m *Meeting, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	m.BodyLines = appendActionItem(m.BodyLines, text)
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// RenameActionItem replaces a checklist line's text in place (as opposed to
// ToggleActionItem/DeleteActionItem, which flip or remove it) and persists
// the result.
func (s *Store) RenameActionItem(m *Meeting, lineNo int, newText string) error {
	newText = strings.TrimSpace(newText)
	if newText == "" {
		return fmt.Errorf("text cannot be empty")
	}
	if lineNo < 0 || lineNo >= len(m.BodyLines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	m.BodyLines[lineNo] = setItemText(m.BodyLines[lineNo], newText)
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// DeleteActionItem removes a checklist line entirely (as opposed to
// ToggleActionItem, which only flips its done state) and persists the
// result.
func (s *Store) DeleteActionItem(m *Meeting, lineNo int) error {
	if lineNo < 0 || lineNo >= len(m.BodyLines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	out := make([]string, 0, len(m.BodyLines)-1)
	out = append(out, m.BodyLines[:lineNo]...)
	out = append(out, m.BodyLines[lineNo+1:]...)
	m.BodyLines = out
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// AddWin appends a new "win since last 1:1" bullet to the meeting notes.
// Like AddActionItem, it re-parses ActionItems afterward, since inserting a
// "## Wins" heading ahead of "## Action Items" (see appendWin) shifts the
// line numbers those items are indexed by.
func (s *Store) AddWin(m *Meeting, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	m.BodyLines = appendWin(m.BodyLines, text)
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// SaveGoals serializes a person's goal checklist lines back to goals.md.
func (s *Store) SaveGoals(p *Person) error {
	body := strings.Join(p.GoalLines, "\n")
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(filepath.Join(p.Dir, "goals.md"), []byte(body), 0o644)
}

// AddGoal appends a new open goal under the given term ("long" or "short")
// to a person's goals.md, creating the section heading if needed. It
// re-reads the file fresh from disk first (Person is passed by value, so a
// caller's copy can go stale across calls) rather than trusting p.GoalLines.
func (s *Store) AddGoal(p Person, term, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	lines, _ := s.loadGoalsFile(p.Dir)
	p.GoalLines = appendGoal(lines, term, text)
	p.Goals = parseGoals(p.GoalLines)
	return s.SaveGoals(&p)
}

// ToggleGoal flips a goal checklist line's done state and persists it. Like
// AddGoal, it re-reads goals.md fresh from disk before applying lineNo.
func (s *Store) ToggleGoal(p Person, lineNo int) error {
	lines, _ := s.loadGoalsFile(p.Dir)
	if lineNo < 0 || lineNo >= len(lines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	lines[lineNo] = toggleGoalLine(lines[lineNo])
	p.GoalLines = lines
	p.Goals = parseGoals(p.GoalLines)
	return s.SaveGoals(&p)
}

// RenameGoal replaces a goal's text in place, keeping its checkbox state and
// term unchanged, mirroring RenameActionItem. Like AddGoal/ToggleGoal/
// DeleteGoal, it re-reads goals.md fresh from disk before applying lineNo.
func (s *Store) RenameGoal(p Person, lineNo int, newText string) error {
	newText = strings.TrimSpace(newText)
	if newText == "" {
		return fmt.Errorf("text cannot be empty")
	}
	lines, _ := s.loadGoalsFile(p.Dir)
	if lineNo < 0 || lineNo >= len(lines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	lines[lineNo] = setItemText(lines[lineNo], newText)
	p.GoalLines = lines
	p.Goals = parseGoals(p.GoalLines)
	return s.SaveGoals(&p)
}

// DeleteGoal removes a goal checklist line entirely (as opposed to
// ToggleGoal, which only flips its done state), mirroring DeleteActionItem.
// Like AddGoal/ToggleGoal, it re-reads goals.md fresh from disk before
// applying lineNo.
func (s *Store) DeleteGoal(p Person, lineNo int) error {
	lines, _ := s.loadGoalsFile(p.Dir)
	if lineNo < 0 || lineNo >= len(lines) {
		return fmt.Errorf("line %d out of range", lineNo)
	}
	out := make([]string, 0, len(lines)-1)
	out = append(out, lines[:lineNo]...)
	out = append(out, lines[lineNo+1:]...)
	p.GoalLines = out
	p.Goals = parseGoals(p.GoalLines)
	return s.SaveGoals(&p)
}

// UpdateBody replaces a meeting's notes body (e.g. after inline editing) and
// persists it.
func (s *Store) UpdateBody(m *Meeting, newBody string) error {
	m.BodyLines = strings.Split(strings.TrimRight(newBody, "\n"), "\n")
	m.ActionItems = parseActionItems(m.BodyLines)
	return s.SaveMeeting(m)
}

// Reload re-reads a single meeting from disk (e.g. after external $EDITOR use).
func (s *Store) Reload(m *Meeting) error {
	fresh, err := s.loadMeeting(m.PersonSlug, m.Path)
	if err != nil {
		return err
	}
	*m = fresh
	return nil
}
