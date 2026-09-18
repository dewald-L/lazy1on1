package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazy1on1/internal/store"
)

type panelID int

const (
	panelPeople panelID = iota
	panelMeetings
	panelGlobalNotes
	panelGoals
	panelActions
	panelDetail
)

type modalID int

const (
	modalNone modalID = iota
	modalNewPerson
	modalQuickAdd
	modalAddGoal
	modalEditNotes
	modalEditGlobalNotes
	modalConfirmDelete
	modalRename
	modalHelp
)

const actionHeading = "## action items"

// detailRow identifies which row of the Detail panel's interactive header
// strip (or the scrollable notes/wins body below it) currently has the
// row-navigation cursor.
type detailRow int

const (
	detailRowRAG detailRow = iota
	detailRowPerceivedPulse
	detailRowNotes
)

// quickAddKind distinguishes what the quick-add modal (shared textinput)
// is currently adding, since the same modal/keystroke path is reused for
// both action items and wins.
type quickAddKind int

const (
	quickAddAction quickAddKind = iota
	quickAddWin
)

// App is the top-level Bubble Tea model for the whole TUI.
type App struct {
	store *store.Store

	width, height int
	ready         bool

	focus panelID
	modal modalID

	people       []store.Person
	peopleIdx    int // confirmed selection: drives the Meetings panel
	peopleCursor int // navigational highlight in the People panel, independent until confirmed with space

	meetings       []store.Meeting
	meetingsIdx    int // confirmed selection: drives the Detail panel
	meetingsCursor int // navigational highlight in the Meetings panel, independent until confirmed with space

	actionItems []store.GlobalActionItem // every action item across everyone, most recent first
	actionIdx   int                      // selected index within actionItems
	goalIdx     int                      // selected index within personGoalsOrdered(currentPerson())

	notesVP      viewport.Model
	detailCursor detailRow // row cursor within the Detail panel's header strip (RAG/Perceived Pulse) or notes body

	globalNotesVP              viewport.Model
	globalNotesEditArea        textarea.Model
	externalEditingGlobalNotes bool // set while $EDITOR is open on notes.md, so editorFinishedMsg knows what to reload

	nameInput    textinput.Model
	quickInput   textinput.Model
	quickAddKind quickAddKind
	goalInput    textinput.Model
	editArea     textarea.Model

	addGoalStep int // 0 = text, 1 = term picker
	goalTermIdx   int // 0 = long-term, 1 = short-term

	// confirmMsg/confirmDelete back the shared modalConfirmDelete modal: the
	// message to show and the action to run if the user confirms, set by
	// whichever confirmDelete* helper opened the modal (see deleteForFocus
	// in update.go). Cleared once the modal closes, confirmed or not.
	confirmMsg    string
	confirmDelete func()

	// renameInput/renameTitle/renameApply back the shared modalRename modal,
	// mirroring confirmMsg/confirmDelete: the title to show and the closure
	// to run with the entered text on confirm, set by whichever
	// renamePrompt* helper opened the modal (see renameForFocus in
	// update.go). Cleared once the modal closes, confirmed or not.
	renameInput textinput.Model
	renameTitle string
	renameApply func(text string)

	status string
	errMsg string
}

func New(s *store.Store) *App {
	ni := textinput.New()
	ni.Placeholder = "Full name"
	ni.CharLimit = 80

	qi := textinput.New()
	qi.Placeholder = "Action item text"
	qi.CharLimit = 200

	gi := textinput.New()
	gi.Placeholder = "Goal text"
	gi.CharLimit = 200

	ri := textinput.New()
	ri.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Notes..."
	ta.ShowLineNumbers = false

	gnta := textarea.New()
	gnta.Placeholder = "Global notes about this person (not tied to any meeting)..."
	gnta.ShowLineNumbers = false

	a := &App{
		store:               s,
		focus:               panelPeople,
		notesVP:             viewport.New(10, 10),
		globalNotesVP:       viewport.New(10, 10),
		globalNotesEditArea: gnta,
		nameInput:           ni,
		quickInput:          qi,
		goalInput:           gi,
		renameInput:         ri,
		editArea:            ta,
	}
	a.reloadPeople()
	return a
}

func (a *App) Init() tea.Cmd {
	return nil
}

// --- data helpers -----------------------------------------------------

func (a *App) currentPerson() *store.Person {
	if a.peopleIdx < 0 || a.peopleIdx >= len(a.people) {
		return nil
	}
	return &a.people[a.peopleIdx]
}

func (a *App) currentMeeting() *store.Meeting {
	if a.meetingsIdx < 0 || a.meetingsIdx >= len(a.meetings) {
		return nil
	}
	return &a.meetings[a.meetingsIdx]
}

func (a *App) reloadPeople() {
	people, err := a.store.ListPeople()
	if err != nil {
		a.errMsg = err.Error()
		return
	}
	a.people = people
	a.peopleIdx = clampIdx(a.peopleIdx, len(a.people))
	a.peopleCursor = clampIdx(a.peopleCursor, len(a.people))
	a.reloadMeetings()
	a.reloadActionItems()
}

func (a *App) reloadMeetings() {
	p := a.currentPerson()
	if p == nil {
		a.meetings = nil
		a.meetingsIdx = 0
		a.syncDetail()
		a.syncGlobalNotes()
		return
	}
	meetings, err := a.store.ListMeetings(*p)
	if err != nil {
		a.errMsg = err.Error()
		return
	}
	a.meetings = meetings
	a.meetingsIdx = clampIdx(a.meetingsIdx, len(a.meetings))
	a.meetingsCursor = clampIdx(a.meetingsCursor, len(a.meetings))
	a.syncDetail()
	a.syncGlobalNotes()
	a.layout()
}

// reloadActionItems refreshes the Actions panel's list from disk: every
// action item across every person, not just the currently selected one
// (the person each item belongs to is shown as reference alongside it).
func (a *App) reloadActionItems() {
	items, err := a.store.AllActionItems()
	if err != nil {
		a.errMsg = err.Error()
		return
	}
	a.actionItems = items
	a.actionIdx = clampIdx(a.actionIdx, len(a.actionItems))
}

// clampIdx bounds i to a valid index into a slice of length n (0 if n is 0).
func clampIdx(i, n int) int {
	if i >= n {
		i = n - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

// selectPersonBySlug re-points peopleIdx (and its cursor) at the person with
// the given slug, if present.
func (a *App) selectPersonBySlug(slug string) {
	for i, p := range a.people {
		if p.Slug == slug {
			a.peopleIdx = i
			a.peopleCursor = i
			return
		}
	}
}

// selectMeetingByPath re-points meetingsIdx (and its cursor) at the meeting
// with the given path, if present in the current meetings slice.
func (a *App) selectMeetingByPath(path string) {
	for i, m := range a.meetings {
		if m.Path == path {
			a.meetingsIdx = i
			a.meetingsCursor = i
			return
		}
	}
}

func (a *App) syncDetail() {
	a.goalIdx = 0
	a.detailCursor = detailRowRAG
	m := a.currentMeeting()
	if m == nil {
		a.notesVP.SetContent("")
		return
	}
	a.notesVP.SetContent(renderNotes(m))
	a.notesVP.GotoTop()
}

// syncGlobalNotes refreshes the Global Notes panel's viewport from the
// currently selected person, mirroring syncDetail for the meeting notes
// viewport.
func (a *App) syncGlobalNotes() {
	p := a.currentPerson()
	if p == nil {
		a.globalNotesVP.SetContent("")
		return
	}
	a.globalNotesVP.SetContent(renderGlobalNotes(p))
	a.globalNotesVP.GotoTop()
}

// personGoalsOrdered returns p's goals in the fixed display order used
// throughout the UI: all Long-term goals, then all Short-term goals,
// each in the order they appear in goals.md. goalIdx indexes into this
// slice, so rendering and navigation/toggling always agree on ordering
// regardless of how the two headings happen to be arranged on disk.
func personGoalsOrdered(p *store.Person) []store.Goal {
	if p == nil {
		return nil
	}
	var out []store.Goal
	for _, g := range p.Goals {
		if g.Term == "long" {
			out = append(out, g)
		}
	}
	for _, g := range p.Goals {
		if g.Term == "short" {
			out = append(out, g)
		}
	}
	return out
}

// notesOnly returns the body lines up to (not including) the "## Action
// Items" heading, so that section can be rendered separately as an
// interactive checklist instead of being duplicated.
func notesOnly(m *store.Meeting) []string {
	for i, l := range m.BodyLines {
		if strings.EqualFold(strings.TrimSpace(l), "## Action Items") {
			return m.BodyLines[:i]
		}
	}
	return m.BodyLines
}

// renderMarkdownLines applies the same light styling (## / # headings,
// everything else as plain text) used by both the Detail panel's meeting
// notes and the Global Notes panel's per-person notes.
func renderMarkdownLines(lines []string) string {
	var b strings.Builder
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(trimmed, "## "):
			b.WriteString(panelTitleStyle.Render(strings.TrimPrefix(trimmed, "## ")))
		case strings.HasPrefix(trimmed, "# "):
			b.WriteString(titleStyle.Render(strings.TrimPrefix(trimmed, "# ")))
		default:
			b.WriteString(normalItemStyle.Render(l))
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderNotes(m *store.Meeting) string {
	return renderMarkdownLines(notesOnly(m))
}

// renderGlobalNotes renders a person's persistent notes.md content (free-
// form, not scoped to any single meeting).
func renderGlobalNotes(p *store.Person) string {
	if strings.TrimSpace(p.Notes) == "" {
		return dimItemStyle.Render("no notes yet — press 'e' to add some")
	}
	return renderMarkdownLines(strings.Split(p.Notes, "\n"))
}
