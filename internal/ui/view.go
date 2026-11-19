package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dewald-L/lazy1on1/internal/store"
)

func (a *App) View() string {
	if !a.ready {
		return "loading..."
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, a.renderMeetingsPanel(), a.renderActionsPanel())
	middle := a.renderDetailPanel()
	if a.prevVisible() {
		middle = lipgloss.JoinHorizontal(lipgloss.Top, a.renderPrevPanel(), middle)
	}
	bottom := lipgloss.JoinHorizontal(lipgloss.Top, a.renderGlobalNotesPanel(), a.renderGoalsPanel())
	body := lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom)

	view := lipgloss.JoinVertical(lipgloss.Left, body, a.renderStatusGraphsPanel())
	if !a.embedded {
		view = lipgloss.JoinVertical(lipgloss.Left, view, a.renderStatusBar(), a.renderHelpBar())
	}

	switch a.modal {
	case modalNewPerson:
		return a.overlay(view, a.renderNewPersonModal())
	case modalQuickAdd:
		return a.overlay(view, a.renderQuickAddModal())
	case modalAddGoal:
		return a.overlay(view, a.renderAddGoalModal())
	case modalConfirmDelete:
		return a.overlay(view, a.renderConfirmDeleteModal())
	case modalRename:
		return a.overlay(view, a.renderRenameModal())
	case modalHelp:
		return a.overlay(view, a.renderHelpModal())
	case modalPeople:
		return a.overlay(view, a.renderPeopleModal())
	}
	return view
}

func (a *App) overlay(base, modal string) string {
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceChars(" "))
}

// panelShortcut returns the bracketed jump key for a panel (e.g. "[1]"),
// derived from keys.FocusPanel so the UI stays in sync with the keymap.
func panelShortcut(id panelID) string {
	// panelPeople (0) is the picker, not a panel: the jump keys cover the
	// panels after it, in enum order.
	ks := keys.FocusPanel.Keys()
	i := int(id) - 1
	if i < 0 || i >= len(ks) {
		return ""
	}
	return "[" + ks[i] + "] "
}

// rowStyle picks a list row's style based on whether it's the confirmed
// selection, the panel's navigational cursor (only meaningful while that
// panel has focus), both, or neither. A confirmed selection in a panel that
// isn't focused gets the dim variant, so the one active panel's highlight
// doesn't have to compete with every other panel's own remembered selection.
func rowStyle(focused, isCursor, isSelected bool) lipgloss.Style {
	switch {
	case isCursor && isSelected:
		return selectedCursorItemStyle
	case isCursor:
		return cursorItemStyle
	case isSelected && focused:
		return selectedItemStyle
	case isSelected:
		return selectedItemStyleDim
	default:
		return normalItemStyle
	}
}

// renderMetricRow renders one Detail-panel header row (RAG or Perceived
// Pulse): a left-padded label, a status glyph in its own color, and a
// value. Only the label picks up the cursor highlight when the row is under
// the cursor - the glyph and value stay in their normal/status colors, so
// the cursor marks which field you'd edit without recoloring the value
// you'd be changing. Each segment is rendered independently and
// concatenated rather than nested inside another Render call: lipgloss's
// Underline handling (part of cursorItemStyle) walks a string rune by rune
// to skip styling whitespace, which would mangle escape codes from an
// already-rendered segment nested inside it.
func renderMetricRow(isCursor bool, label, glyph string, glyphColor lipgloss.Color, value string) string {
	labelCol := fmt.Sprintf("%-17s", label)
	labelStyle := normalItemStyle
	if isCursor {
		labelStyle = cursorItemStyle
	}
	return labelStyle.Render(labelCol) +
		lipgloss.NewStyle().Foreground(glyphColor).Render(glyph) +
		normalItemStyle.Render(" "+value)
}

// panelFrame renders a bordered panel whose title sits inline in the top
// border (like dock's home page boxes) instead of taking a content row.
type panelFrame struct {
	style   lipgloss.Style
	title   string
	focused bool
}

func panelBorder(focused bool, title string, width, height int) panelFrame {
	s := panelStyle
	if focused {
		s = panelStyleFocused
	}
	if !strings.Contains(title, "\x1b") {
		title = panelTitleStyle.Render(title)
	}
	return panelFrame{style: s.Width(width).Height(height), title: title, focused: focused}
}

func (f panelFrame) Render(content string) string {
	out := f.style.Render(content)
	lines := strings.Split(out, "\n")
	w := lipgloss.Width(lines[0])
	if w < 6 || f.title == "" {
		return out
	}
	border := lipgloss.RoundedBorder()
	if f.focused {
		border = lipgloss.ThickBorder()
	}
	bs := lipgloss.NewStyle().Foreground(colorMuted)
	if f.focused {
		bs = lipgloss.NewStyle().Foreground(colorBorderFocus)
	}
	t := ansi.Truncate(" "+f.title+" ", w-4, "…")
	fill := w - 3 - lipgloss.Width(t)
	if fill < 0 {
		fill = 0
	}
	lines[0] = bs.Render(border.TopLeft+border.Top) + t + bs.Render(strings.Repeat(border.Top, fill)+border.TopRight)
	return strings.Join(lines, "\n")
}

func relDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

// --- people modal ---------------------------------------------------------

func (a *App) renderPeopleModal() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("People") + "\n\n")

	if len(a.people) == 0 {
		b.WriteString(dimItemStyle.Render("no one yet\npress n to add") + "\n")
	}
	for i, p := range a.people {
		// Only the cursor row is highlighted; everyone else is styled
		// identically (no marker for the person currently open).
		style := normalItemStyle
		if i == a.peopleCursor {
			style = selectedItemStyle
		}
		label := p.Name
		if i < 9 {
			label = fmt.Sprintf("%d  %s", i+1, p.Name)
		}
		b.WriteString(style.Render(label))
		b.WriteString("\n")
	}
	b.WriteString("\n" + dimItemStyle.Render("1-9 pick · ↑/↓ move · space/enter open · n new · r rename · d delete · q "+a.quitWord()))
	return modalStyle.Width(60).Render(b.String())
}

// --- goals panel ---------------------------------------------------------

func (a *App) renderGoalsPanel() string {
	w, h := a.goalsWidth(), a.goalsHeight()
	focused := a.focus == panelGoals

	var b strings.Builder
	p := a.currentPerson()
	shortcut := panelShortcut(panelGoals)
	title := shortcut + "Goals"
	if p != nil {
		title = shortcut + "Goals — " + p.Name
	}

	if p == nil {
		b.WriteString("\n" + dimItemStyle.Render("select a person"))
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	// Open / Completed tabs (toggle with t).
	nOpen, nDone := goalCounts(p)
	openTab, doneTab := fmt.Sprintf("Open (%d)", nOpen), fmt.Sprintf("Completed (%d)", nDone)
	active := selectedItemStyleDim // quiet while this panel isn't focused
	if focused {
		active = selectedItemStyle
	}
	if a.goalsDone {
		openTab, doneTab = dimItemStyle.Render(openTab), active.Render(doneTab)
	} else {
		openTab, doneTab = active.Render(openTab), dimItemStyle.Render(doneTab)
	}
	b.WriteString(openTab + dimItemStyle.Render(" │ ") + doneTab + dimItemStyle.Render("   t switch") + "\n")

	var body strings.Builder
	goalIdx, selLine := 0, 0
	renderGoal := func(g store.Goal) {
		box := "[ ]"
		style := openItemStyle
		if g.Done {
			box = "[x]"
			style = doneItemStyle
		}
		line := fmt.Sprintf("%s %s", box, g.Text)
		if goalIdx == a.goalIdx {
			selLine = strings.Count(body.String(), "\n")
		}
		if focused && goalIdx == a.goalIdx {
			body.WriteString(selectedItemStyle.Render(line))
		} else {
			body.WriteString(style.Render(line))
		}
		body.WriteString("\n")
		goalIdx++
	}

	body.WriteString(dimItemStyle.Render("Long-term") + "\n")
	hasLong := false
	for _, g := range p.Goals {
		if g.Term != "long" || g.Done != a.goalsDone {
			continue
		}
		hasLong = true
		renderGoal(g)
	}
	if !hasLong {
		body.WriteString(dimItemStyle.Render("  none") + "\n")
	}

	body.WriteString(dimItemStyle.Render("Short-term") + "\n")
	hasShort := false
	for _, g := range p.Goals {
		if g.Term != "short" || g.Done != a.goalsDone {
			continue
		}
		hasShort = true
		renderGoal(g)
	}
	if !hasShort {
		body.WriteString(dimItemStyle.Render("  none") + "\n")
	}
	body.WriteString(dimItemStyle.Render("press 'n' to add a goal"))

	// Scroll the goal lines so the selected goal stays visible.
	all := strings.Split(body.String(), "\n")
	capacity := h - 1
	if capacity < 1 {
		capacity = 1
	}
	start := 0
	if selLine >= capacity {
		start = selLine - capacity + 1
	}
	end := start + capacity
	if end > len(all) {
		end = len(all)
	}
	b.WriteString(withScrollbar(strings.Join(all[start:end], "\n"), w-3, scrollbar(len(all), end-start, start, capacity)))

	return panelBorder(focused, title, w, h).Render(clipLines(b.String(), h))
}

// --- meetings panel ---------------------------------------------------------

func (a *App) renderMeetingsPanel() string {
	w, h := a.meetingsWidth(), a.meetingsHeight()
	focused := a.focus == panelMeetings

	var b strings.Builder
	p := a.currentPerson()
	shortcut := panelShortcut(panelMeetings)
	title := shortcut + "Meetings"
	if p != nil {
		title = shortcut + "Meetings — " + p.Name
	}

	if p == nil {
		b.WriteString(dimItemStyle.Render("select a person"))
	} else if len(a.meetings) == 0 {
		b.WriteString(dimItemStyle.Render("no meetings yet\npress N to add one now"))
	}

	// Window the list so the cursor stays visible. Rows with tags take two
	// lines, so the clip below is the backstop for the visible count.
	capacity := h
	start := 0
	if capacity > 0 && a.meetingsCursor >= capacity {
		start = a.meetingsCursor - capacity + 1
	}
	var rows strings.Builder
	shown := 0
	for i, m := range a.meetings {
		if i < start {
			continue
		}
		open := len(m.OpenActionItems())
		style := rowStyle(focused, focused && i == a.meetingsCursor, i == a.meetingsIdx)
		// Rendered as two segments sharing style's background (rather than
		// embedding ragStyledGlyph's independently-Rendered string, whose own
		// trailing reset would cancel style's background partway through the
		// line, leaving only the glyph highlighted) so a selected row's
		// highlight covers the whole line.
		glyphStyle := style.Foreground(ragColor(string(m.RAG)))
		line := glyphStyle.Render(ragGlyphChar(string(m.RAG))) + style.Render(" "+relDate(m.Date))
		rows.WriteString(line)
		if m.Locked {
			rows.WriteString(dimItemStyle.Render("  (locked)"))
		}
		if open > 0 {
			rows.WriteString(dimItemStyle.Render(fmt.Sprintf("  (%d open)", open)))
		}
		if len(m.Tags) > 0 {
			rows.WriteString("\n  " + tagStyle.Render(strings.Join(m.Tags, ", ")))
		}
		rows.WriteString("\n")
		shown++
	}
	if len(a.meetings) > 0 && capacity > 0 {
		body := clipLines(rows.String(), capacity)
		b.WriteString(withScrollbar(body, w-3, scrollbar(len(a.meetings), shown, start, capacity)))
	}

	return panelBorder(focused, title, w, h).Render(clipLines(b.String(), h))
}

// scrollbar draws a one-column vertical scrollbar, height lines tall, for a
// list of total lines/items of which visible are shown starting at offset.
// When everything fits it is a blank column, so the content width doesn't
// change as content grows past the window.
func scrollbar(total, visible, offset, height int) string {
	if height < 1 {
		return ""
	}
	lines := make([]string, height)
	if total <= visible || total <= 0 {
		for i := range lines {
			lines[i] = " "
		}
		return strings.Join(lines, "\n")
	}
	thumb := height * visible / total
	if thumb < 1 {
		thumb = 1
	}
	if thumb > height {
		thumb = height
	}
	pos := 0
	if maxOff := total - visible; maxOff > 0 {
		pos = (offset*(height-thumb) + maxOff/2) / maxOff
	}
	for i := range lines {
		if i >= pos && i < pos+thumb {
			lines[i] = scrollThumbStyle.Render("█")
		} else {
			lines[i] = scrollTrackStyle.Render("░")
		}
	}
	return strings.Join(lines, "\n")
}

// withScrollbar puts a scrollbar to the right of block, which is padded to
// width columns first.
func withScrollbar(block string, width int, bar string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(width).Render(block), bar)
}

// inert restyles block in the previous-meeting window's permanent muted
// palette, dropping any colors it already had.
func inert(block string) string {
	lines := strings.Split(ansi.Strip(block), "\n")
	for i, l := range lines {
		lines[i] = inertTextStyle.Render(l)
	}
	return strings.Join(lines, "\n")
}

// vpScrollbar is the scrollbar for a viewport.
func vpScrollbar(vp viewport.Model) string {
	return scrollbar(vp.TotalLineCount(), vp.Height, vp.YOffset, vp.Height)
}

// clipLines keeps at most n lines of s (see the comment on topHeight).
func clipLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// --- actions panel ---------------------------------------------------------

// renderActionsPanel lists the selected person's action items, most recent
// meeting first, each one annotated with the meeting it came from.
func (a *App) renderActionsPanel() string {
	w, h := a.actionsWidth(), a.actionsHeight()
	focused := a.focus == panelActions

	var b strings.Builder
	shortcut := panelShortcut(panelActions)
	title := shortcut + "Actions"
	if p := a.currentPerson(); p != nil {
		title = shortcut + "Actions — " + p.Name
	}

	// Open / Closed tabs (toggle with t).
	nOpen, nClosed := a.actionCounts()
	openTab, closedTab := fmt.Sprintf("Open (%d)", nOpen), fmt.Sprintf("Closed (%d)", nClosed)
	active := selectedItemStyleDim // quiet while this panel isn't focused
	if focused {
		active = selectedItemStyle
	}
	if a.actionsClosed {
		openTab, closedTab = dimItemStyle.Render(openTab), active.Render(closedTab)
	} else {
		openTab, closedTab = active.Render(openTab), dimItemStyle.Render(closedTab)
	}
	b.WriteString(openTab + dimItemStyle.Render(" │ ") + closedTab + dimItemStyle.Render("   t switch") + "\n")

	items := a.actionItems
	if len(items) == 0 {
		if a.actionsClosed {
			b.WriteString(dimItemStyle.Render("no closed action items"))
		} else {
			b.WriteString(dimItemStyle.Render("no open action items — press 'n' to add one"))
		}
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	// Each item takes two lines (the item, then who it belongs to). Window
	// the list to what fits in h so an unbounded item count can't grow this
	// panel past its share, keeping the selected item visible.
	capacity := (h - 1) / 2
	if capacity < 1 {
		capacity = 1
	}
	start := 0
	if a.actionIdx >= capacity {
		start = a.actionIdx - capacity + 1
	}
	end := start + capacity
	if end > len(items) {
		end = len(items)
	}

	var rows strings.Builder
	for i := start; i < end; i++ {
		row := items[i]
		box := "[ ]"
		style := openItemStyle
		if row.Item.Done {
			box = "[x]"
			style = doneItemStyle
		}
		line := fmt.Sprintf("%s %s", box, row.Item.Text)
		if focused && i == a.actionIdx {
			rows.WriteString(selectedItemStyle.Render(line))
		} else {
			rows.WriteString(style.Render(line))
		}
		rows.WriteString("\n")
		rows.WriteString(dimItemStyle.Render("  " + relDate(row.Meeting.Date)))
		rows.WriteString("\n")
	}
	barH := capacity * 2
	b.WriteString(withScrollbar(clipLines(rows.String(), barH), w-3, scrollbar(len(items), end-start, start, barH)))

	return panelBorder(focused, title, w, h).Render(clipLines(b.String(), h))
}

// --- global notes panel -----------------------------------------------------

// renderGlobalNotesPanel shows a single person-scoped, free-form notes
// block that persists across all of that person's meetings — distinct from
// the Detail panel's notes body below it, which is scoped to one meeting.
func (a *App) renderGlobalNotesPanel() string {
	w := a.globalNotesWidth()
	h := a.globalNotesHeight()
	focused := a.focus == panelGlobalNotes

	p := a.currentPerson()
	shortcut := panelShortcut(panelGlobalNotes)
	title := shortcut + "Global Notes"
	if p != nil {
		title = shortcut + "Global Notes — " + p.Name
	}

	var b strings.Builder

	if p == nil {
		b.WriteString(dimItemStyle.Render("select a person"))
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	if a.modal == modalEditGlobalNotes {
		b.WriteString(a.globalNotesEditArea.View())
	} else {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, a.globalNotesVP.View(), vpScrollbar(a.globalNotesVP)))
	}

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- detail panel ---------------------------------------------------------

// renderPrevPanel is the read-only previous-meeting window shown beside the
// current meeting notes.
func (a *App) renderPrevPanel() string {
	prev := a.previousMeeting()
	title := "Previous — " + prev.Date.Format("Mon, Jan 2 · 15:04")
	var b strings.Builder
	b.WriteString(inertTextStyle.Bold(true).Render(truncate(title, a.prevWidth()-2)) + "\n")
	b.WriteString(inertTextStyle.Render("read-only · alt+↑/↓ scroll") + "\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, a.prevVP.View(), inert(vpScrollbar(a.prevVP))))
	return panelStyleInert.Width(a.prevWidth()).Height(a.detailHeight()).Render(b.String())
}

func (a *App) renderDetailPanel() string {
	w := a.detailWidth()
	h := a.detailHeight()
	focused := a.focus == panelDetail

	p := a.currentPerson()
	m := a.currentMeeting()

	var b strings.Builder
	detailShortcut := panelShortcut(panelDetail)
	title := detailShortcut + "Meeting notes"
	if p == nil {
		b.WriteString(dimItemStyle.Render("nothing selected yet"))
		return panelBorder(focused, detailShortcut+"Meeting notes", w, h).Render(b.String())
	}

	if m != nil {
		header := fmt.Sprintf("%sMeeting notes — %s", detailShortcut, m.Date.Format("Mon, Jan 2 2006 · 15:04"))
		headerLine := panelTitleStyle.Render(header)
		if m.Locked {
			headerLine += "  " + lockedBadgeStyle.Render("[locked]")
		}
		if len(m.Tags) > 0 {
			headerLine += "  " + tagStyle.Render(strings.Join(m.Tags, ", "))
		}
		title = headerLine

		if a.modal == modalEditNotes {
			b.WriteString(a.editArea.View())
		} else {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, a.notesVP.View(), vpScrollbar(a.notesVP)))
		}
	} else {
		title = detailShortcut + "Meeting notes — " + p.Name
		b.WriteString(dimItemStyle.Render("no meetings yet — press 'N' to add one") + "\n")
	}

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- RAG / Perceived Pulse over-time panel ---------------------------------

const graphBarRows = 3

// maxGraphMeetings caps each status graph to the most recent N meetings,
// regardless of how many columns the panel width would otherwise fit.
const maxGraphMeetings = 5

// renderBarRows draws graphBarRows lines of block characters, one column per
// level (tallest bars at the top), colored via colorAt(level). A zero level
// still draws a bare dot on the bottom row rather than a blank column, so an
// explicitly-unset value reads as a point on the axis, not just a gap.
func renderBarRows(levels []int, colorAt func(level int) lipgloss.Color) string {
	rows := make([]string, 0, graphBarRows)
	for row := graphBarRows; row >= 1; row-- {
		var line strings.Builder
		for _, lvl := range levels {
			switch {
			case lvl >= row:
				line.WriteString(lipgloss.NewStyle().Foreground(colorAt(lvl)).Render("█"))
			case lvl == 0 && row == 1:
				line.WriteString(dimItemStyle.Render("·"))
			default:
				line.WriteString(" ")
			}
			line.WriteString(" ")
		}
		rows = append(rows, line.String())
	}
	return strings.Join(rows, "\n")
}

// metricGraphLines is renderMetricGraph's fixed line count (a label line,
// graphBarRows bar rows, and an axis line), so the RAG and Perceived Pulse
// columns always come out the same height and line up cleanly when joined
// side by side.
const metricGraphLines = 1 + graphBarRows + 1

// renderMetricGraph renders one small time-series bar graph - a label line,
// bar rows (tallest at the top), and an axis line - sized to fit within
// maxWidth columns. It's meant to sit beside a sibling metric's graph via
// lipgloss.JoinHorizontal, which is why it always returns exactly
// metricGraphLines lines.
func renderMetricGraph(label string, chron []store.Meeting, maxWidth int, levelOf func(store.Meeting) int, colorAt func(level int) lipgloss.Color) string {
	const colWidth = 2 // one bar column + one gap column
	maxCols := maxWidth / colWidth
	if maxCols < 1 {
		maxCols = 1
	}
	if maxCols > maxGraphMeetings {
		maxCols = maxGraphMeetings
	}
	shown := chron
	if len(shown) > maxCols {
		shown = shown[len(shown)-maxCols:]
	}

	levels := make([]int, len(shown))
	for i, m := range shown {
		levels[i] = levelOf(m)
	}

	var b strings.Builder
	b.WriteString(dimItemStyle.Render(label))
	b.WriteString("\n" + renderBarRows(levels, colorAt))

	axisWidth := len(levels) * colWidth
	b.WriteString("\n" + dimItemStyle.Render(strings.Repeat("─", axisWidth)))

	return b.String()
}

// renderStatusGraphsPanel draws two small bar graphs of the current
// person's meetings in chronological order (oldest to newest, left to
// right), side by side: RAG health (green tallest, red shortest) beside
// Perceived Pulse (up tallest, down shortest), each with its own axis,
// since halving the width also halves how many meetings each graph can
// show.
func (a *App) renderStatusGraphsPanel() string {
	w := a.rowWidth()
	h := statusGraphsContentHeight

	p := a.currentPerson()
	title := "Detail"
	if p != nil {
		title = "Detail — " + p.Name
	}

	var b strings.Builder
	if m := a.currentMeeting(); m != nil {
		focused := a.focus == panelDetail
		metric := func(on bool, label, glyph string, c lipgloss.Color, val string) string {
			l := label
			if on {
				l = selectedItemStyle.Render(label)
			}
			return l + " " + lipgloss.NewStyle().Foreground(c).Render(glyph) + " " + val
		}
		b.WriteString(metric(focused && a.detailCursor == detailRowRAG, "RAG",
			ragGlyphChar(string(m.RAG)), ragColor(string(m.RAG)), ragLabel(string(m.RAG))))
		b.WriteString("   " + metric(focused && a.detailCursor == detailRowPerceivedPulse, "Perceived Pulse",
			perceivedPulseGlyphChar(string(m.PerceivedPulse)), perceivedPulseColor(string(m.PerceivedPulse)), m.PerceivedPulse.Label()))
	}

	if p == nil || len(a.meetings) == 0 {
		b.WriteString("\n" + dimItemStyle.Render("no meetings yet"))
		return panelBorder(false, title, w, h).Render(b.String())
	}

	// a.meetings is newest-first; the graphs read left-to-right oldest-first.
	n := len(a.meetings)
	chron := make([]store.Meeting, n)
	for i, m := range a.meetings {
		chron[n-1-i] = m
	}

	const gapWidth = 4
	halfW := (w - gapWidth) / 2
	if halfW < 6 {
		halfW = 6
	}

	ragBlock := renderMetricGraph("RAG", chron, halfW,
		func(m store.Meeting) int { return ragLevel(string(m.RAG)) },
		func(lvl int) lipgloss.Color { return ragColor(levelToRAG(lvl)) })
	pulseBlock := renderMetricGraph("Perceived Pulse", chron, halfW,
		func(m store.Meeting) int { return perceivedPulseLevel(string(m.PerceivedPulse)) },
		func(lvl int) lipgloss.Color { return perceivedPulseColor(levelToPerceivedPulse(lvl)) })

	dividerLines := make([]string, metricGraphLines)
	for i := range dividerLines {
		dividerLines[i] = dimItemStyle.Render("│")
	}
	divider := strings.Join(dividerLines, "\n")

	b.WriteString("\n" + lipgloss.JoinHorizontal(lipgloss.Top, ragBlock, "  ", divider, "  ", pulseBlock))

	return panelBorder(false, title, w, h).Render(b.String())
}

// levelToRAG inverts ragLevel, for reusing ragColor's status→color mapping.
func levelToRAG(level int) string {
	switch level {
	case 3:
		return "green"
	case 2:
		return "amber"
	case 1:
		return "red"
	default:
		return ""
	}
}

// levelToPerceivedPulse inverts perceivedPulseLevel, for reusing
// perceivedPulseColor's status→color mapping.
func levelToPerceivedPulse(level int) string {
	switch level {
	case 3:
		return "up"
	case 2:
		return "steady"
	case 1:
		return "down"
	default:
		return ""
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- bars -------------------------------------------------------------

func (a *App) renderStatusBar() string {
	msg := a.status
	if a.errMsg != "" {
		msg = "error: " + a.errMsg
	}
	left := "lazy1on1 — continuous 1:1 notes"
	right := msg
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	content := left + strings.Repeat(" ", gap) + right
	style := statusBarStyle
	if a.errMsg != "" {
		style = style.Foreground(colorRed)
	}
	return style.Width(a.width).Render(content)
}

type kv struct{ k, v string }

// renderHelpBar builds a lazygit-style contextual footer from helpPairs.
// Not shown when embedded in dock: the host renders Help() instead.
func (a *App) renderHelpBar() string {
	var parts []string
	for _, p := range a.helpPairs() {
		parts = append(parts, helpKeyStyle.Render(p.k)+" "+p.v)
	}
	return helpBarStyle.Width(a.width).Render(strings.Join(parts, "  "))
}

// helpPairs lists the keys for the current context: keys that only do
// something in the currently focused panel/section come first, followed by
// the keys that work no matter what has focus. It feeds both the help bar
// (standalone) and Help() (embedded in dock).
func (a *App) helpPairs() []kv {
	var pairs []kv

	switch a.modal {
	case modalEditNotes, modalEditGlobalNotes:
		return append(pairs, kv{"ctrl+s", "save"}, kv{"ctrl+p", "previous meeting"}, kv{"esc", "cancel"})
	case modalConfirmDelete:
		return append(pairs, kv{"y/enter", "confirm"}, kv{"esc/n", "cancel"})
	case modalRename:
		return append(pairs, kv{"enter", "save"}, kv{"esc", "cancel"})
	case modalPeople:
		return append(pairs, kv{"1-9", "pick person"}, kv{"space/enter", "open person"}, kv{"n", "new"}, kv{"r", "rename"}, kv{"d", "delete"}, kv{"q", a.quitWord()})
	}

	switch a.focus {
	case panelGoals:
		pairs = append(pairs,
			kv{"space", "toggle"}, kv{"t", "open/completed tab"}, kv{"esc", "people"}, kv{"n", "new goal"}, kv{"r", "rename goal"}, kv{"d", "delete goal"},
		)
	case panelMeetings:
		pairs = append(pairs,
			kv{"→/l/enter", "open meeting"}, kv{"esc", "people"},
			kv{"r", "cycle RAG"}, kv{"n", "new meeting"}, kv{"L", "lock/unlock"}, kv{"d", "delete meeting"},
		)
	case panelActions:
		pairs = append(pairs,
			kv{"space", "toggle"}, kv{"t", "open/closed tab"}, kv{"←/h/esc", "back"}, kv{"n", "new action item"}, kv{"r", "rename action item"}, kv{"d", "delete action item"},
		)
	case panelGlobalNotes:
		pairs = append(pairs,
			kv{"←/h/esc", "back"}, kv{"e", "edit notes"}, kv{"E", "$EDITOR"}, kv{"d", "clear notes"},
		)
	case panelDetail:
		pairs = append(pairs,
			kv{"enter/space", "cycle RAG/Pulse"}, kv{"←/h/esc", "back"},
			kv{"w", "add win"}, kv{"p", "previous meeting"}, kv{"n", "new action item"}, kv{"e", "edit notes"}, kv{"E", "$EDITOR"},
			kv{"L", "lock/unlock"}, kv{"d", "delete meeting"},
		)
	}

	return append(pairs,
		kv{"tab", "panel"},
		kv{"?", "help"}, kv{"q", "people"},
	)
}

// quitWord is what "q" does: quit the program, or close the dock plugin.
func (a *App) quitWord() string {
	if a.embedded {
		return "close"
	}
	return "quit"
}

// --- modals -------------------------------------------------------------

func (a *App) renderNewPersonModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add a person") + "\n\n")
	b.WriteString("Name:  " + a.nameInput.View() + "\n")
	b.WriteString("\n" + dimItemStyle.Render("enter to add · esc to cancel"))
	return modalStyle.Width(50).Render(b.String())
}

func (a *App) renderQuickAddModal() string {
	title := "New action item"
	if a.quickAddKind == quickAddWin {
		title = "New win"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title) + "\n\n")
	b.WriteString(a.quickInput.View() + "\n")
	b.WriteString("\n" + dimItemStyle.Render("enter to add · esc to cancel"))
	return modalStyle.Width(50).Render(b.String())
}

func (a *App) renderAddGoalModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add a goal") + "\n\n")
	b.WriteString("Goal:  " + a.goalInput.View() + "\n")
	if a.addGoalStep >= 1 {
		b.WriteString("\n" + dimItemStyle.Render("Term:") + "\n")
		options := []string{"Long-term", "Short-term"}
		for i, opt := range options {
			if i == a.goalTermIdx {
				b.WriteString(selectedItemStyle.Render("> "+opt) + "\n")
			} else {
				b.WriteString(normalItemStyle.Render("  "+opt) + "\n")
			}
		}
		b.WriteString("\n" + dimItemStyle.Render("←/→ choose · enter confirm · esc cancel"))
	} else {
		b.WriteString("\n" + dimItemStyle.Render("enter to continue · esc to cancel"))
	}
	return modalStyle.Width(50).Render(b.String())
}

func (a *App) renderRenameModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(a.renameTitle) + "\n\n")
	b.WriteString(a.renameInput.View() + "\n")
	b.WriteString("\n" + dimItemStyle.Render("enter to save · esc to cancel"))
	return modalStyle.Width(50).Render(b.String())
}

func (a *App) renderConfirmDeleteModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Confirm delete") + "\n\n")
	b.WriteString(a.confirmMsg + "\n")
	b.WriteString("\n" + dimItemStyle.Render("y/enter to confirm · esc/n to cancel"))
	return modalStyle.Width(50).Render(b.String())
}

func (a *App) renderHelpModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("lazy1on1 — keybindings") + "\n\n")
	sections := [][2]string{
		{"Navigation", "tab / shift+tab     cycle panel\nq (or esc in Meetings/Goals)  back to the People picker, a separate\n                    screen with its own keys (the picker's q quits)\n1/2/3/4/5           jump to Meetings / Actions / Meeting notes / Global Notes / Goals\n↑/k, ↓/j            move cursor within panel\n                    (Meetings syncs the Meeting notes panel as you move, no space needed)\n→/l/enter           confirm selection and drill into panel\n←/h/esc             back out"},
		{"People & meetings", "n   new — contextual per panel:\n      People        new person\n      Meetings      new meeting now (opens editor)\n      Goals         new goal (long-term or short-term)\n      Actions,\n      Meeting notes quick-add an action item to the selected meeting\nd   delete — contextual per panel, always asks to confirm:\n      People        delete the selected person and all their data\n      Meetings,\n      Meeting notes delete the selected/current meeting\n      Goals         delete the selected goal\n      Actions       delete the selected action item\n      Global Notes  clear the person's global notes\nr   rename/cycle RAG — contextual per panel:\n      People        rename the selected person\n      Meetings      cycle RAG status (none→green→amber→red)\n      Goals         rename the selected goal\n      Actions       rename the selected action item\nL   lock/unlock — Meetings, Meeting notes:\n      toggle the selected/current meeting's locked state. A locked\n      meeting shows \"(locked)\"/\"[locked]\" and rejects RAG/Pulse\n      changes, notes edits, action items/wins, and deletion — from\n      any panel, including action items reached via the Actions\n      panel — until unlocked again."},
		{"Meeting notes panel", "↑↓/jk       move between the RAG row, Perceived Pulse row, and notes body\n            (scrolls once the cursor reaches the notes)\nenter/space cycle the selected row's value:\n              RAG              none→green→amber→red\n              Perceived Pulse  none→down→steady→up\nw           log a win since the last 1:1 (appended under \"## Wins\")\np           show/hide the previous meeting's notes below the current ones\n            (also ctrl+p while editing; alt+↑/↓ scrolls it)"},
		{"Global Notes panel", "a free-form, per-person notes block that persists across all of that\nperson's meetings — unlike Meeting notes, which belong to one meeting.\n↑↓/jk       scroll"},
		{"Notes", "e   edit notes inline — the selected meeting's notes, or the Global\n    Notes panel's notes if that panel has focus\nE   edit in $EDITOR (same target as e)"},
		{"Action items", "space toggle selected action item in the Actions panel\nt     switch the Actions panel between its Open and Closed tabs"},
		{"Goals", "space toggle selected goal in the Goals panel\nt     switch the Goals panel between its Open and Completed tabs"},
		{"Misc", "?   toggle this help\nq / ctrl+c   quit"},
	}
	if a.embedded {
		sections[len(sections)-1][1] = "?   toggle this help\nq   close lazy1on1 (ctrl+c quits dock)"
	}
	for _, s := range sections {
		b.WriteString(panelTitleStyle.Render(s[0]) + "\n")
		b.WriteString(s[1] + "\n\n")
	}
	b.WriteString(dimItemStyle.Render("press any key to close"))
	return modalStyle.Width(60).Render(b.String())
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
