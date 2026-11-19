package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dewald-L/dock/plugin"

	"github.com/dewald-L/lazy1on1/internal/store"
)

type editorFinishedMsg struct{ err error }

// helpBarHeight and statusBarHeight are 0 when embedded in dock: the host's
// status bar shows our messages and key hints instead.
func (a *App) helpBarHeight() int {
	if a.embedded {
		return 0
	}
	return 1
}

func (a *App) statusBarHeight() int { return a.helpBarHeight() }

// statusGraphsBoxHeight is the total rendered height (content + border) of
// the "Detail" graph panel along the bottom of the screen: a title line,
// plus the RAG and Perceived Pulse graphs side by side (see
// metricGraphLines in view.go), plus the panel's border.
const statusGraphsContentHeight = 1 + metricGraphLines
const statusGraphsBoxHeight = statusGraphsContentHeight + 2

// Update handles msg. When embedded in dock it also forwards status and
// error messages to the host's status bar and keeps the tab title in step
// with the selected person.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := a.update(msg)
	if !a.embedded {
		return m, cmd
	}
	cmds := []tea.Cmd{cmd}
	if a.errMsg != "" {
		cmds = append(cmds, plugin.Error(errors.New(a.errMsg)))
		a.errMsg = ""
	} else if a.status != "" {
		cmds = append(cmds, plugin.Notify(a.status))
	}
	a.status = ""
	title := "lazy1on1"
	if p := a.currentPerson(); p != nil {
		title = "1:1 · " + p.Name
	}
	if title != a.title {
		a.title = title
		cmds = append(cmds, plugin.SetTitle(title))
	}
	return m, tea.Batch(cmds...)
}

func (a *App) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case commandMsg:
		return a.runCommand(msg)

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.ready = true
		a.layout()
		return a, nil

	case editorFinishedMsg:
		if msg.err != nil {
			a.errMsg = msg.err.Error()
		} else if a.externalEditingGlobalNotes {
			a.externalEditingGlobalNotes = false
			if p := a.currentPerson(); p != nil {
				if err := a.store.ReloadNotes(p); err != nil {
					a.errMsg = err.Error()
				} else {
					a.syncGlobalNotes()
					a.status = "reloaded from $EDITOR"
				}
			}
		} else if m := a.currentMeeting(); m != nil {
			_ = a.store.Reload(m)
			a.syncDetail()
			a.reloadActionItems()
			a.status = "reloaded from $EDITOR"
		}
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	return a, nil
}

func (a *App) layout() {
	if a.height <= 0 {
		return
	}
	detailWidth := a.width - leftColumnWidth() - 6
	if detailWidth < 10 {
		detailWidth = 10
	}
	a.notesVP.Width = detailWidth - 2
	a.notesVP.Height = a.notesBodyHeight()
	a.prevVP.Width = detailWidth - 2
	if a.prevVisible() {
		// Split the body: current notes on top, then a one-line title and
		// the read-only previous meeting below.
		prevH := (a.notesVP.Height - 1) / 2
		a.prevVP.Height = prevH
		a.notesVP.Height -= prevH + 1
	}
	a.editArea.SetWidth(a.notesVP.Width)
	a.editArea.SetHeight(a.notesVP.Height)

	// Global Notes panel: same width as Detail (same right column), minus
	// just its own title line (see renderGlobalNotesPanel).
	a.globalNotesVP.Width = detailWidth - 2
	a.globalNotesVP.Height = a.globalNotesHeight() - 1
	if a.globalNotesVP.Height < 2 {
		a.globalNotesVP.Height = 2
	}
	a.globalNotesEditArea.SetWidth(a.globalNotesVP.Width)
	a.globalNotesEditArea.SetHeight(a.globalNotesVP.Height)
}

func leftColumnWidth() int { return 34 }

// totalBodyHeight is the vertical space available for the body row (the
// People/Meetings/Detail panels) before any individual panel's own border
// is subtracted.
func (a *App) totalBodyHeight() int {
	h := a.height - a.helpBarHeight() - a.statusBarHeight() - statusGraphsBoxHeight
	if h < 5 {
		h = 5
	}
	return h
}

// globalNotesHeight and detailHeight split the right column's vertical space
// between the Global Notes panel and the Detail panel, each with its own
// border - mirrors how peopleHeight/goalsHeight/etc. split the left column.
// Global Notes gets a smaller fixed share since it's usually a short
// freeform blurb about the person; Detail gets whatever's left.
func (a *App) globalNotesHeight() int {
	avail := a.totalBodyHeight() - 4 // two panels' worth of borders
	h := avail / 4
	if h < 3 {
		h = 3
	}
	return h
}

// detailHeight is the Detail panel's content height, sharing the right
// column with the Global Notes panel above it (both borders come out of the
// same budget - see globalNotesHeight).
func (a *App) detailHeight() int {
	avail := a.totalBodyHeight() - 4
	h := avail - a.globalNotesHeight()
	if h < 3 {
		h = 3
	}
	return h
}

// peopleHeight, goalsHeight, actionsHeight and meetingsHeight split the left
// column's vertical space between the People, Meetings, Goals and Actions
// panels, lazygit-sidebar style: four panels stacked in one column, each
// with its own border, so all four borders come out of the shared budget.
// People and Goals get a smaller fixed share since their rows are short;
// Actions gets a bigger one since each of its rows takes two lines (the
// item and, under it, the meeting it came from); Meetings — usually the
// longest list — gets whatever's left.
//
// lipgloss's Height() only pads short content up to that height, it never
// truncates content that's taller, so a panel whose actual content exceeds
// its share will silently grow and push everything below and beside the
// left column out of sync. renderActionsPanel clips its item list to
// actionsHeight() for exactly this reason.
func (a *App) goalsHeight() int {
	avail := a.totalBodyHeight() - 6
	h := avail / 4
	if h < 3 {
		h = 3
	}
	return h
}

func (a *App) actionsHeight() int {
	avail := a.totalBodyHeight() - 6
	h := avail / 4
	if h < 4 {
		h = 4
	}
	return h
}

func (a *App) meetingsHeight() int {
	avail := a.totalBodyHeight() - 6
	h := avail - a.goalsHeight() - a.actionsHeight()
	if h < 3 {
		h = 3
	}
	return h
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal input takes over the keyboard entirely first.
	switch a.modal {
	case modalNewPerson:
		return a.returnToPeople(a.handleNewPersonKey(msg))
	case modalQuickAdd:
		return a.handleQuickAddKey(msg)
	case modalAddGoal:
		return a.handleAddGoalKey(msg)
	case modalEditNotes:
		return a.handleEditNotesKey(msg)
	case modalEditGlobalNotes:
		return a.handleEditGlobalNotesKey(msg)
	case modalConfirmDelete:
		return a.returnToPeople(a.handleConfirmDeleteKey(msg))
	case modalRename:
		return a.returnToPeople(a.handleRenameKey(msg))
	case modalHelp:
		a.modal = modalNone
		return a, nil
	case modalPeople:
		return a.handlePeopleKey(msg)
	}

	// Global keys, available from any panel.
	switch {
	case isKey(msg, "ctrl+c"):
		return a, tea.Quit
	case isKey(msg, "q"):
		// q backs out of the notes view to the person picker (the picker
		// is where q quits/closes).
		return a.openPeople()
	case isKey(msg, "?"):
		a.modal = modalHelp
		return a, nil
	case isKey(msg, "n"):
		return a.newForFocus()
	case isKey(msg, "d"):
		return a.deleteForFocus()
	case isKey(msg, "r"):
		if a.focus == panelMeetings {
			return a.cycleRAG()
		}
		return a.renameForFocus()
	case isKey(msg, "L"):
		return a.toggleLockForFocus()
	case isKey(msg, "tab"):
		a.cyclePanel(1)
		return a, nil
	case isKey(msg, "shift+tab"):
		a.cyclePanel(-1)
		return a, nil
	case isKey(msg, "2"):
		a.setFocus(panelMeetings)
		return a, nil
	case isKey(msg, "3"):
		a.setFocus(panelGlobalNotes)
		return a, nil
	case isKey(msg, "4"):
		a.setFocus(panelGoals)
		return a, nil
	case isKey(msg, "5"):
		a.setFocus(panelActions)
		return a, nil
	case isKey(msg, "0"):
		a.setFocus(panelDetail)
		return a, nil
	case isKey(msg, "e"):
		return a.editForFocus()
	case isKey(msg, "E"):
		return a.editExternalForFocus()
	}

	switch a.focus {
	case panelGoals:
		return a.handleGoalsKey(msg)
	case panelMeetings:
		return a.handleMeetingsKey(msg)
	case panelActions:
		return a.handleActionsKey(msg)
	case panelGlobalNotes:
		return a.handleGlobalNotesKey(msg)
	case panelDetail:
		return a.handleDetailKey(msg)
	}
	return a, nil
}

// setFocus moves focus to panel p, resetting that panel's navigational
// cursor to match its currently confirmed selection so entering a panel
// always starts the cursor where the last selection landed.
func (a *App) setFocus(p panelID) {
	switch p {
	case panelMeetings:
		a.meetingsCursor = a.meetingsIdx
	}
	a.focus = p
}

// newForFocus runs the "create a new X" action for whichever panel currently
// has focus, lazygit-style: "n" always means "new" but what it creates
// depends on context. This is the only way to reach these actions — there
// are no standalone global keys for them, so "new" only ever means "new
// thing in the panel you're looking at."
func (a *App) newForFocus() (tea.Model, tea.Cmd) {
	switch a.focus {
	case panelGoals:
		return a.startAddGoal()
	case panelMeetings:
		return a.newMeetingNow()
	case panelActions, panelDetail:
		if m := a.currentMeeting(); m != nil && !a.blockIfLocked(m) {
			a.quickInput.Reset()
			a.quickInput.Placeholder = "Action item text"
			a.quickAddKind = quickAddAction
			cmd := a.quickInput.Focus()
			a.modal = modalQuickAdd
			return a, cmd
		}
	}
	return a, nil
}

// deleteForFocus runs the "delete X" action for whichever panel currently
// has focus, mirroring newForFocus: "d" always means "delete" but what it
// deletes depends on context. Every branch opens the shared
// modalConfirmDelete modal rather than deleting immediately - every one of
// these is a permanent, unrecoverable disk write, unlike the toggles/cycles
// elsewhere in this file. Meeting notes reuses Meetings' action (both view
// the same currently-selected meeting), the way newForFocus already shares
// its Actions/Meeting notes branch.
func (a *App) deleteForFocus() (tea.Model, tea.Cmd) {
	switch a.focus {
	case panelGoals:
		return a.confirmDeleteGoal()
	case panelMeetings, panelDetail:
		return a.confirmDeleteMeeting()
	case panelActions:
		return a.confirmDeleteActionItem()
	case panelGlobalNotes:
		return a.confirmClearGlobalNotes()
	}
	return a, nil
}

// renameForFocus runs the "rename X" action for whichever panel currently
// has focus, mirroring newForFocus/deleteForFocus: "r" means "rename" in
// most panels, but what it renames depends on context. Only People, Goals,
// and Actions have a single piece of text worth renaming in place; other
// panels are a no-op. The Meetings panel is handled separately in
// handleKey, where "r" instead cycles RAG.
func (a *App) renameForFocus() (tea.Model, tea.Cmd) {
	switch a.focus {
	case panelGoals:
		return a.renameGoalPrompt()
	case panelActions:
		return a.renameActionItemPrompt()
	}
	return a, nil
}

// startRename opens the shared modalRename modal, prefilled with current,
// with apply set as the closure to run with the entered text on confirm.
func (a *App) startRename(title, current string, apply func(text string)) (tea.Model, tea.Cmd) {
	a.renameInput.Reset()
	a.renameInput.SetValue(current)
	a.renameInput.CursorEnd()
	a.renameTitle = title
	a.renameApply = apply
	cmd := a.renameInput.Focus()
	a.modal = modalRename
	return a, cmd
}

// renamePersonPrompt targets the People panel's cursor row (which may
// differ from peopleIdx, the last-confirmed selection - see setFocus),
// mirroring confirmDeletePerson.
func (a *App) renamePersonPrompt() (tea.Model, tea.Cmd) {
	if a.peopleCursor < 0 || a.peopleCursor >= len(a.people) {
		return a, nil
	}
	p := a.people[a.peopleCursor]
	return a.startRename("Rename person", p.Name, func(text string) {
		if err := a.store.RenamePerson(p, text); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadPeople()
		a.status = "renamed to " + text
	})
}

func (a *App) renameGoalPrompt() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	goals := personGoalsOrdered(p)
	if p == nil || a.goalIdx < 0 || a.goalIdx >= len(goals) {
		return a, nil
	}
	person := *p
	g := goals[a.goalIdx]
	return a.startRename("Rename goal", g.Text, func(text string) {
		if err := a.store.RenameGoal(person, g.LineNo, text); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.refreshGoalsKeepSelection()
		a.status = "renamed goal"
	})
}

func (a *App) renameActionItemPrompt() (tea.Model, tea.Cmd) {
	items := a.actionItems
	if a.actionIdx < 0 || a.actionIdx >= len(items) {
		return a, nil
	}
	row := items[a.actionIdx]
	if a.blockIfLocked(&row.Meeting) {
		return a, nil
	}
	meeting := row.Meeting
	return a.startRename("Rename action item", row.Item.Text, func(text string) {
		if err := a.store.RenameActionItem(&meeting, row.Item.LineNo, text); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadActionItems()
		a.reloadMeetings()
		a.status = "renamed action item"
	})
}

// confirmDeletePerson targets the People panel's cursor row (which may
// differ from peopleIdx, the last-confirmed selection - see setFocus).
func (a *App) confirmDeletePerson() (tea.Model, tea.Cmd) {
	if a.peopleCursor < 0 || a.peopleCursor >= len(a.people) {
		return a, nil
	}
	p := a.people[a.peopleCursor]
	a.confirmMsg = fmt.Sprintf("Delete %s? This permanently removes all their meetings, goals, and notes.", p.Name)
	a.confirmDelete = func() {
		if err := a.store.DeletePerson(p); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadPeople()
		a.status = "deleted " + p.Name
	}
	a.modal = modalConfirmDelete
	return a, nil
}

func (a *App) confirmDeleteGoal() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	goals := personGoalsOrdered(p)
	if p == nil || a.goalIdx < 0 || a.goalIdx >= len(goals) {
		return a, nil
	}
	person := *p
	g := goals[a.goalIdx]
	a.confirmMsg = fmt.Sprintf("Delete goal %q?", g.Text)
	a.confirmDelete = func() {
		if err := a.store.DeleteGoal(person, g.LineNo); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadPeople()
		a.goalIdx = clampIdx(a.goalIdx, len(personGoalsOrdered(a.currentPerson())))
		a.status = "deleted goal"
	}
	a.modal = modalConfirmDelete
	return a, nil
}

// confirmDeleteMeeting targets the currently selected meeting - shared by
// the Meetings panel and the Meeting notes (Detail) panel, which always
// agree on what's selected (Meetings confirms its cursor immediately on
// every move via selectMeeting, unlike People's cursor/idx split).
func (a *App) confirmDeleteMeeting() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil || a.blockIfLocked(m) {
		return a, nil
	}
	meeting := *m
	a.confirmMsg = fmt.Sprintf("Delete the meeting on %s? This permanently removes its notes file.", meeting.Date.Format("Jan 2, 2006"))
	a.confirmDelete = func() {
		if err := a.store.DeleteMeeting(meeting); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadMeetings()
		a.status = "deleted meeting"
	}
	a.modal = modalConfirmDelete
	return a, nil
}

func (a *App) confirmDeleteActionItem() (tea.Model, tea.Cmd) {
	items := a.actionItems
	if a.actionIdx < 0 || a.actionIdx >= len(items) {
		return a, nil
	}
	row := items[a.actionIdx]
	if a.blockIfLocked(&row.Meeting) {
		return a, nil
	}
	meeting := row.Meeting
	a.confirmMsg = fmt.Sprintf("Delete action item %q?", row.Item.Text)
	a.confirmDelete = func() {
		if err := a.store.DeleteActionItem(&meeting, row.Item.LineNo); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadActionItems()
		a.reloadMeetings()
		a.status = "deleted action item"
	}
	a.modal = modalConfirmDelete
	return a, nil
}

// confirmClearGlobalNotes is the Global Notes panel's delete action: unlike
// the other panels, Global Notes has no discrete list of items to select
// from, just one person-scoped text blob, so "delete" clears that blob
// instead of removing a row.
func (a *App) confirmClearGlobalNotes() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	if p == nil || strings.TrimSpace(p.Notes) == "" {
		return a, nil
	}
	person := *p
	a.confirmMsg = fmt.Sprintf("Clear %s's global notes? This permanently deletes the text.", person.Name)
	a.confirmDelete = func() {
		if err := a.store.UpdateNotes(&person, ""); err != nil {
			a.errMsg = err.Error()
			return
		}
		a.reloadPeople()
		a.syncGlobalNotes()
		a.status = "cleared global notes"
	}
	a.modal = modalConfirmDelete
	return a, nil
}

func isKey(msg tea.KeyMsg, keys ...string) bool {
	s := msg.String()
	for _, k := range keys {
		if s == k {
			return true
		}
	}
	return false
}

// --- panel: people ------------------------------------------------------

// openPeople opens the centered People modal with the cursor on the
// currently selected person.
func (a *App) openPeople() (tea.Model, tea.Cmd) {
	a.peopleCursor = a.peopleIdx
	a.peopleReturn = true
	a.modal = modalPeople
	return a, nil
}

// returnToPeople reopens the People modal when a dialog launched from it
// (new/rename/delete) has just closed.
func (a *App) returnToPeople(m tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if a.modal == modalNone && a.peopleReturn {
		a.modal = modalPeople
	}
	return m, cmd
}

// startNewPerson opens the new-person dialog.
func (a *App) startNewPerson() (tea.Model, tea.Cmd) {
	a.nameInput.Reset()
	cmd := a.nameInput.Focus()
	a.modal = modalNewPerson
	return a, cmd
}

// cyclePanel moves focus through the sidebar/right panels in order,
// skipping panelPeople (it is a modal, not a panel).
func (a *App) cyclePanel(delta int) {
	const n = int(panelDetail) // panels 1..5 -> n of them
	i := int(a.focus) - 1
	if i < 0 {
		i = 0
	}
	i = ((i+delta)%n + n) % n
	a.setFocus(panelID(i + 1))
}

func (a *App) closePeople() {
	a.peopleReturn = false
	a.modal = modalNone
}

func (a *App) handlePeopleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "up", "k"):
		if a.peopleCursor > 0 {
			a.peopleCursor--
		}
	case isKey(msg, "down", "j"):
		if a.peopleCursor < len(a.people)-1 {
			a.peopleCursor++
		}
	case isKey(msg, " ", "right", "l", "enter"):
		if len(a.people) > 0 {
			a.selectPerson()
			a.closePeople()
			a.setFocus(panelMeetings)
		}
	case len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9':
		// number keys pick the nth person directly
		if i := int(msg.String()[0] - '1'); i < len(a.people) {
			a.peopleCursor = i
			a.selectPerson()
			a.closePeople()
			a.setFocus(panelMeetings)
		}
	case isKey(msg, "q", "ctrl+c"):
		return a, tea.Quit
	case isKey(msg, "n"):
		return a.startNewPerson()
	case isKey(msg, "r"):
		return a.renamePersonPrompt()
	case isKey(msg, "d"):
		return a.confirmDeletePerson()
	}
	return a, nil
}

// selectPerson confirms the People panel's cursor row as the actual
// selection, loading that person's meetings.
func (a *App) selectPerson() {
	a.peopleIdx = a.peopleCursor
	a.meetingsIdx = 0
	a.meetingsCursor = 0
	a.reloadMeetings()
}

// --- panel: meetings ------------------------------------------------------

func (a *App) handleMeetingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "up", "k"):
		if a.meetingsCursor > 0 {
			a.meetingsCursor--
			a.selectMeeting()
		}
	case isKey(msg, "down", "j"):
		if a.meetingsCursor < len(a.meetings)-1 {
			a.meetingsCursor++
			a.selectMeeting()
		}
	case isKey(msg, "esc"):
		return a.openPeople()
	case isKey(msg, "right", "l", "enter"):
		if len(a.meetings) > 0 {
			a.selectMeeting()
			a.focus = panelDetail
		}
	}
	return a, nil
}

// selectMeeting confirms the Meetings panel's cursor row as the actual
// selection, syncing the Detail panel to it.
func (a *App) selectMeeting() {
	a.meetingsIdx = a.meetingsCursor
	a.syncDetail()
}

// --- panel: detail (notes) -------------------------------------------------

func (a *App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "left", "h", "esc"):
		a.setFocus(panelMeetings)
		return a, nil
	case isKey(msg, "up", "k"):
		return a.detailUp()
	case isKey(msg, "down", "j"):
		return a.detailDown()
	case isKey(msg, "pgup"):
		a.notesVP.ViewUp()
		return a, nil
	case isKey(msg, "pgdown"):
		a.notesVP.ViewDown()
		return a, nil
	case isKey(msg, " ", "enter"):
		return a.activateDetailRow()
	case isKey(msg, "w"):
		return a.startAddWin()
	case isKey(msg, "p"):
		return a.togglePrev()
	case isKey(msg, "alt+up"):
		a.prevVP.LineUp(1)
		return a, nil
	case isKey(msg, "alt+down"):
		a.prevVP.LineDown(1)
		return a, nil
	}
	return a, nil
}

// detailUp moves the Detail panel's row cursor up through RAG -> Perceived
// Pulse -> notes, or scrolls the notes viewport up once the cursor has
// reached it - stepping back up to the Perceived Pulse row only once the
// viewport is already at its top.
func (a *App) detailUp() (tea.Model, tea.Cmd) {
	switch a.detailCursor {
	case detailRowPerceivedPulse:
		a.detailCursor = detailRowRAG
	case detailRowNotes:
		if a.notesVP.AtTop() {
			a.detailCursor = detailRowPerceivedPulse
		} else {
			a.notesVP.LineUp(1)
		}
	}
	return a, nil
}

// detailDown is detailUp's mirror: RAG -> Perceived Pulse -> notes, then
// scrolling.
func (a *App) detailDown() (tea.Model, tea.Cmd) {
	switch a.detailCursor {
	case detailRowRAG:
		a.detailCursor = detailRowPerceivedPulse
	case detailRowPerceivedPulse:
		a.detailCursor = detailRowNotes
	case detailRowNotes:
		a.notesVP.LineDown(1)
	}
	return a, nil
}

// activateDetailRow handles space/enter on whichever Detail panel row is
// currently under the cursor: cycling RAG or Perceived Pulse. It's a no-op
// once the cursor has moved into the notes body (nothing to "activate"
// there).
func (a *App) activateDetailRow() (tea.Model, tea.Cmd) {
	switch a.detailCursor {
	case detailRowRAG:
		return a.cycleRAG()
	case detailRowPerceivedPulse:
		return a.cyclePerceivedPulse()
	}
	return a, nil
}

// --- panel: global notes ---------------------------------------------------

// handleGlobalNotesKey drives the Global Notes panel: a single scrollable,
// person-scoped block of free-form notes that persist across all of that
// person's meetings (unlike the Detail panel's per-meeting notes), so it
// only needs scroll handling, not Detail's RAG/Pulse/notes row cursor.
func (a *App) handleGlobalNotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "left", "h", "esc"):
		a.setFocus(panelMeetings)
	case isKey(msg, "up", "k"):
		a.globalNotesVP.LineUp(1)
	case isKey(msg, "down", "j"):
		a.globalNotesVP.LineDown(1)
	case isKey(msg, "pgup"):
		a.globalNotesVP.ViewUp()
	case isKey(msg, "pgdown"):
		a.globalNotesVP.ViewDown()
	}
	return a, nil
}

// --- panel: actions ---------------------------------------------------

// handleActionsKey drives the Actions panel: every action item across
// every person, flattened into one list by reloadActionItems().
func (a *App) handleActionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := a.actionItems

	switch {
	case isKey(msg, "left", "h", "esc"):
		a.setFocus(panelMeetings)
		return a, nil
	case isKey(msg, "up", "k"):
		if a.actionIdx > 0 {
			a.actionIdx--
		}
		return a, nil
	case isKey(msg, "down", "j"):
		if a.actionIdx < len(items)-1 {
			a.actionIdx++
		}
		return a, nil
	case isKey(msg, " "):
		if a.actionIdx >= 0 && a.actionIdx < len(items) {
			row := items[a.actionIdx]
			if err := a.store.ToggleActionItem(&row.Meeting, row.Item.LineNo); err != nil {
				a.errMsg = err.Error()
			} else {
				a.reloadActionItems()
				a.reloadMeetings()
			}
		}
		return a, nil
	}
	return a, nil
}

// --- panel: goals ---------------------------------------------------------

func (a *App) handleGoalsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	goals := personGoalsOrdered(p)

	switch {
	case isKey(msg, "esc"):
		return a.openPeople()
	case isKey(msg, "up", "k"):
		if a.goalIdx > 0 {
			a.goalIdx--
		}
		return a, nil
	case isKey(msg, "down", "j"):
		if a.goalIdx < len(goals)-1 {
			a.goalIdx++
		}
		return a, nil
	case isKey(msg, " "):
		if p != nil && a.goalIdx >= 0 && a.goalIdx < len(goals) {
			lineNo := goals[a.goalIdx].LineNo
			if err := a.store.ToggleGoal(*p, lineNo); err != nil {
				a.errMsg = err.Error()
			} else {
				a.refreshGoalsKeepSelection()
			}
		}
		return a, nil
	}
	return a, nil
}

// refreshGoalsKeepSelection reloads people from disk (refreshing goal state)
// without resetting the Goals panel's current cursor position.
func (a *App) refreshGoalsKeepSelection() {
	keepGoal := a.goalIdx
	a.reloadPeople()
	a.goalIdx = keepGoal
}

// --- actions shared across panels ---------------------------------------

func (a *App) newMeetingNow() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	if p == nil {
		a.status = "add a person first (n)"
		return a, nil
	}
	m, err := a.store.CreateMeeting(*p, time.Now())
	if err != nil {
		a.errMsg = err.Error()
		return a, nil
	}
	a.reloadMeetings()
	a.selectMeetingByPath(m.Path)
	a.syncDetail()
	a.focus = panelDetail
	return a.startEditNotes()
}

// startAddGoal opens the two-step add-goal modal: goal text, then a
// long-term/short-term term picker.
func (a *App) startAddGoal() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	if p == nil {
		a.status = "add a person first (n)"
		return a, nil
	}
	a.goalInput.Reset()
	a.addGoalStep = 0
	a.goalTermIdx = 0
	cmd := a.goalInput.Focus()
	a.modal = modalAddGoal
	return a, cmd
}

func (a *App) cycleRAG() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil {
		return a, nil
	}
	if err := a.store.SetRAG(m, m.RAG.Next()); err != nil {
		a.errMsg = err.Error()
		return a, nil
	}
	a.refreshDetailKeepCursor()
	return a, nil
}

func (a *App) cyclePerceivedPulse() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil {
		return a, nil
	}
	if err := a.store.SetPerceivedPulse(m, m.PerceivedPulse.Next()); err != nil {
		a.errMsg = err.Error()
		return a, nil
	}
	a.refreshDetailKeepCursor()
	return a, nil
}

// refreshDetailKeepCursor reloads meetings from disk (refreshing RAG/
// Perceived Pulse state) without resetting the Detail panel's row cursor
// back to the top, mirroring refreshGoalsKeepSelection for the Goals panel.
func (a *App) refreshDetailKeepCursor() {
	keep := a.detailCursor
	a.reloadMeetings()
	a.detailCursor = keep
}

// toggleLockForFocus runs "lock/unlock the current meeting" for whichever
// panel currently has focus, mirroring deleteForFocus: Meetings and Meeting
// notes are the only two panels that operate on "the currently selected
// meeting" (see confirmDeleteMeeting), so 'L' is a no-op everywhere else.
func (a *App) toggleLockForFocus() (tea.Model, tea.Cmd) {
	switch a.focus {
	case panelMeetings, panelDetail:
		return a.toggleLock()
	}
	return a, nil
}

func (a *App) toggleLock() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil {
		return a, nil
	}
	locked := !m.Locked
	if err := a.store.SetLocked(m, locked); err != nil {
		a.errMsg = err.Error()
		return a, nil
	}
	a.refreshDetailKeepCursor()
	if locked {
		a.status = "locked meeting"
	} else {
		a.status = "unlocked meeting"
	}
	return a, nil
}

// blockIfLocked is the front-line guard for actions that would otherwise
// open an editor or a text-input modal against a locked meeting: it stops
// the action before any input is collected, rather than letting the store's
// own lock check (see errMeetingLocked in store.go) reject the save
// afterward and silently discard whatever the user just typed. The store
// methods still enforce the same rule independently, since some call paths
// (e.g. the Actions panel) reach a meeting other than a.currentMeeting().
func (a *App) blockIfLocked(m *store.Meeting) bool {
	if m == nil || !m.Locked {
		return false
	}
	a.status = "meeting is locked — press L to unlock"
	return true
}

// startAddWin opens the quick-add modal for logging a win since the last
// 1:1, reusing the same textinput/modal path as quick-adding an action
// item (see newForFocus) but tagged with quickAddWin so handleQuickAddKey
// appends it under "## Wins" instead.
func (a *App) startAddWin() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil || a.blockIfLocked(m) {
		return a, nil
	}
	a.quickInput.Reset()
	a.quickInput.Placeholder = "Win text"
	a.quickAddKind = quickAddWin
	cmd := a.quickInput.Focus()
	a.modal = modalQuickAdd
	return a, cmd
}

// notesForEditing returns a meeting's body text, guaranteeing there's a
// blank separator line directly above the "## Action Items" heading (adding
// one if it's missing, e.g. because a previous edit's note text ran right up
// against it). That blank line is what startEditNotes lands the cursor on,
// and keeping it self-healing means notes never start getting typed into
// the heading line itself.
func notesForEditing(m *store.Meeting) string {
	lines := append([]string{}, m.BodyLines...)
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), "## Action Items") {
			if i == 0 || strings.TrimSpace(lines[i-1]) != "" {
				out := make([]string, 0, len(lines)+1)
				out = append(out, lines[:i]...)
				out = append(out, "")
				out = append(out, lines[i:]...)
				lines = out
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}

// startEditNotes opens the inline notes editor. The cursor is placed right
// before the "## Action Items" heading (on the blank line above it) so
// typing immediately extends the notes prose, whether this is a brand-new
// meeting or one that already has notes - never inside the action items
// section itself. If there's no such heading, the cursor lands at the very
// end of the document instead.
func (a *App) startEditNotes() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil || a.blockIfLocked(m) {
		return a, nil
	}
	a.setFocus(panelDetail)
	a.editArea.SetValue(notesForEditing(m))

	// SetValue may sanitize away a trailing blank line, so re-derive line
	// positions from what's actually loaded rather than from m.BodyLines.
	loaded := strings.Split(a.editArea.Value(), "\n")
	target := len(loaded) - 1
	for i, l := range loaded {
		if strings.EqualFold(strings.TrimSpace(l), "## Action Items") {
			target = i
			if i > 0 && strings.TrimSpace(loaded[i-1]) == "" {
				target = i - 1
			}
			break
		}
	}

	// Walk the cursor to row 0 (safe/idempotent once already there), then
	// down to the target row.
	for i := 0; i < len(loaded)+5; i++ {
		a.editArea.CursorUp()
	}
	a.editArea.CursorStart()
	for i := 0; i < target; i++ {
		a.editArea.CursorDown()
	}
	a.editArea.CursorEnd()

	cmd := a.editArea.Focus()
	a.modal = modalEditNotes
	return a, cmd
}

func (a *App) openExternalEditor() (tea.Model, tea.Cmd) {
	m := a.currentMeeting()
	if m == nil || a.blockIfLocked(m) {
		return a, nil
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, m.Path)
	return a, plugin.Exec(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// editForFocus and editExternalForFocus make 'e'/'E' contextual on which
// panel has focus, lazygit-style (see newForFocus): the Global Notes panel
// edits the selected person's persistent notes, everywhere else edits the
// currently selected meeting's notes exactly as before this panel existed.
func (a *App) editForFocus() (tea.Model, tea.Cmd) {
	if a.focus == panelGlobalNotes {
		return a.startEditGlobalNotes()
	}
	return a.startEditNotes()
}

func (a *App) editExternalForFocus() (tea.Model, tea.Cmd) {
	if a.focus == panelGlobalNotes {
		return a.openExternalEditorGlobalNotes()
	}
	return a.openExternalEditor()
}

// startEditGlobalNotes opens the inline editor for the selected person's
// global notes, mirroring startEditNotes for meeting notes.
func (a *App) startEditGlobalNotes() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	if p == nil {
		return a, nil
	}
	a.setFocus(panelGlobalNotes)
	a.globalNotesEditArea.SetValue(p.Notes)
	cmd := a.globalNotesEditArea.Focus()
	a.modal = modalEditGlobalNotes
	return a, cmd
}

func (a *App) openExternalEditorGlobalNotes() (tea.Model, tea.Cmd) {
	p := a.currentPerson()
	if p == nil {
		return a, nil
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	a.externalEditingGlobalNotes = true
	c := exec.Command(editor, a.store.NotesPath(*p))
	return a, plugin.Exec(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// --- modal: new person ----------------------------------------------------

func (a *App) handleNewPersonKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "esc"):
		a.modal = modalNone
		return a, nil
	case isKey(msg, "enter"):
		name := a.nameInput.Value()
		if name == "" {
			return a, nil
		}
		p, err := a.store.CreatePerson(name, "", "", 14)
		if err != nil {
			a.errMsg = err.Error()
			a.modal = modalNone
			return a, nil
		}
		if err := a.store.UpdateNotes(&p, newPersonNotesTemplate); err != nil {
			a.errMsg = err.Error()
		}
		a.reloadPeople()
		a.selectPersonBySlug(p.Slug)
		a.reloadMeetings()
		a.modal = modalNone
		a.focus = panelMeetings
		a.status = "added " + p.Name
		return a, nil
	}
	var cmd tea.Cmd
	a.nameInput, cmd = a.nameInput.Update(msg)
	return a, cmd
}

// newPersonNotesTemplate seeds a new person's global notes (notes.md) with a
// heading for what they currently own, so there's a place to jot it down
// right away instead of starting from a blank file.
const newPersonNotesTemplate = "## Current projects / responsibilities\n"

// --- modal: quick add action item ------------------------------------------

func (a *App) handleQuickAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "esc"):
		a.modal = modalNone
		return a, nil
	case isKey(msg, "enter"):
		m := a.currentMeeting()
		if m != nil && a.quickInput.Value() != "" {
			var err error
			if a.quickAddKind == quickAddWin {
				err = a.store.AddWin(m, a.quickInput.Value())
			} else {
				err = a.store.AddActionItem(m, a.quickInput.Value())
			}
			if err != nil {
				a.errMsg = err.Error()
			} else {
				a.syncDetail()
				a.reloadActionItems()
			}
		}
		a.modal = modalNone
		return a, nil
	}
	var cmd tea.Cmd
	a.quickInput, cmd = a.quickInput.Update(msg)
	return a, cmd
}

// --- modal: add goal ------------------------------------------------------

func (a *App) handleAddGoalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "esc"):
		a.modal = modalNone
		return a, nil
	case isKey(msg, "enter"):
		if a.addGoalStep == 0 {
			if a.goalInput.Value() == "" {
				return a, nil
			}
			a.addGoalStep = 1
			a.goalInput.Blur()
			return a, nil
		}
		p := a.currentPerson()
		if p != nil {
			term := "long"
			if a.goalTermIdx == 1 {
				term = "short"
			}
			if err := a.store.AddGoal(*p, term, a.goalInput.Value()); err != nil {
				a.errMsg = err.Error()
			} else {
				a.reloadPeople()
				a.status = "added goal for " + p.Name
			}
		}
		a.modal = modalNone
		return a, nil
	}
	if a.addGoalStep == 0 {
		var cmd tea.Cmd
		a.goalInput, cmd = a.goalInput.Update(msg)
		return a, cmd
	}
	switch {
	case isKey(msg, "left", "up", "h", "k"):
		a.goalTermIdx = 0
	case isKey(msg, "right", "down", "l", "j"):
		a.goalTermIdx = 1
	}
	return a, nil
}

// --- modal: confirm delete --------------------------------------------------

// handleConfirmDeleteKey drives the single confirm-delete modal shared by
// every panel's "d" action (see deleteForFocus): the modal itself doesn't
// know what it's deleting, just the message to show and the closure to run
// on confirm, set by whichever confirmDelete* helper opened it.
func (a *App) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "y", "enter"):
		fn := a.confirmDelete
		a.modal = modalNone
		a.confirmMsg = ""
		a.confirmDelete = nil
		if fn != nil {
			fn()
		}
	case isKey(msg, "n", "esc"):
		a.modal = modalNone
		a.confirmMsg = ""
		a.confirmDelete = nil
	}
	return a, nil
}

// --- modal: rename ----------------------------------------------------------

// handleRenameKey drives the single rename modal shared by People/Goals/
// Actions' "r" action (see renameForFocus): the modal itself doesn't know
// what it's renaming, just the title to show and the closure to run with
// the entered text on confirm, set by whichever renamePrompt* helper opened
// it, mirroring handleConfirmDeleteKey.
func (a *App) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "esc"):
		a.modal = modalNone
		a.renameApply = nil
		return a, nil
	case isKey(msg, "enter"):
		text := a.renameInput.Value()
		fn := a.renameApply
		a.modal = modalNone
		a.renameApply = nil
		if fn != nil && text != "" {
			fn(text)
		}
		return a, nil
	}
	var cmd tea.Cmd
	a.renameInput, cmd = a.renameInput.Update(msg)
	return a, cmd
}

// --- modal: edit notes (inline textarea) -----------------------------------

// togglePrev shows/hides the previous-meeting section next to the notes.
func (a *App) togglePrev() (tea.Model, tea.Cmd) {
	a.showPrev = !a.showPrev
	a.layout()
	switch {
	case a.currentMeeting() == nil:
	case a.previousMeeting() == nil:
		a.status = "no previous meeting"
	case a.showPrev && !a.prevVisible():
		a.status = "not enough room for the previous meeting"
	}
	return a, nil
}

func (a *App) handleEditNotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "ctrl+p"):
		return a.togglePrev()
	case isKey(msg, "alt+up"):
		a.prevVP.LineUp(1)
		return a, nil
	case isKey(msg, "alt+down"):
		a.prevVP.LineDown(1)
		return a, nil
	case isKey(msg, "esc"):
		a.modal = modalNone
		a.editArea.Blur()
		return a, nil
	case isKey(msg, "ctrl+s"):
		m := a.currentMeeting()
		if m != nil {
			if err := a.store.UpdateBody(m, a.editArea.Value()); err != nil {
				a.errMsg = err.Error()
			} else {
				a.syncDetail()
				a.reloadActionItems()
				a.status = "saved"
			}
		}
		a.modal = modalNone
		a.editArea.Blur()
		return a, nil
	}
	var cmd tea.Cmd
	a.editArea, cmd = a.editArea.Update(msg)
	return a, cmd
}

// --- modal: edit global notes (inline textarea) ----------------------------

func (a *App) handleEditGlobalNotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, "esc"):
		a.modal = modalNone
		a.globalNotesEditArea.Blur()
		return a, nil
	case isKey(msg, "ctrl+s"):
		p := a.currentPerson()
		if p != nil {
			if err := a.store.UpdateNotes(p, a.globalNotesEditArea.Value()); err != nil {
				a.errMsg = err.Error()
			} else {
				a.syncGlobalNotes()
				a.status = "saved"
			}
		}
		a.modal = modalNone
		a.globalNotesEditArea.Blur()
		return a, nil
	}
	var cmd tea.Cmd
	a.globalNotesEditArea, cmd = a.globalNotesEditArea.Update(msg)
	return a, cmd
}
