package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"lazy1on1/internal/store"
)

func (a *App) View() string {
	if !a.ready {
		return "loading..."
	}

	left := lipgloss.JoinVertical(
		lipgloss.Left,
		a.renderPeoplePanel(),
		a.renderMeetingsPanel(),
		a.renderGoalsPanel(),
		a.renderActionsPanel(),
	)
	right := lipgloss.JoinVertical(
		lipgloss.Left,
		a.renderGlobalNotesPanel(),
		a.renderDetailPanel(),
	)
	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		right,
	)

	view := lipgloss.JoinVertical(lipgloss.Left, body, a.renderStatusGraphsPanel(), a.renderStatusBar(), a.renderHelpBar())

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
	ks := keys.FocusPanel.Keys()
	if int(id) >= len(ks) {
		return ""
	}
	return "[" + ks[id] + "] "
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

func panelBorder(focused bool, title string, width, height int) lipgloss.Style {
	s := panelStyle
	if focused {
		s = panelStyleFocused
	}
	return s.Width(width).Height(height)
}

func relDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

// --- people panel ---------------------------------------------------------

func (a *App) renderPeoplePanel() string {
	w, h := leftColumnWidth(), a.peopleHeight()
	focused := a.focus == panelPeople

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render(panelShortcut(panelPeople)+"People") + "\n\n")

	if len(a.people) == 0 {
		b.WriteString(dimItemStyle.Render("no one yet\npress n to add"))
	}
	for i, p := range a.people {
		style := rowStyle(focused, focused && i == a.peopleCursor, i == a.peopleIdx)
		b.WriteString(style.Render(p.Name))
		b.WriteString("\n")
	}

	return panelBorder(focused, "People", w, h).Render(b.String())
}

// --- goals panel ---------------------------------------------------------

func (a *App) renderGoalsPanel() string {
	w, h := leftColumnWidth(), a.goalsHeight()
	focused := a.focus == panelGoals

	var b strings.Builder
	p := a.currentPerson()
	shortcut := panelShortcut(panelGoals)
	title := shortcut + "Goals"
	if p != nil {
		title = shortcut + "Goals — " + p.Name
	}
	b.WriteString(panelTitleStyle.Render(title) + "\n\n")

	if p == nil {
		b.WriteString(dimItemStyle.Render("select a person"))
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	goalIdx := 0
	renderGoal := func(g store.Goal) {
		box := "[ ]"
		style := openItemStyle
		if g.Done {
			box = "[x]"
			style = doneItemStyle
		}
		line := fmt.Sprintf("%s %s", box, g.Text)
		if focused && goalIdx == a.goalIdx {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(style.Render(line))
		}
		b.WriteString("\n")
		goalIdx++
	}

	b.WriteString(dimItemStyle.Render("Long-term") + "\n")
	hasLong := false
	for _, g := range p.Goals {
		if g.Term != "long" {
			continue
		}
		hasLong = true
		renderGoal(g)
	}
	if !hasLong {
		b.WriteString(dimItemStyle.Render("  none") + "\n")
	}

	b.WriteString(dimItemStyle.Render("Short-term") + "\n")
	hasShort := false
	for _, g := range p.Goals {
		if g.Term != "short" {
			continue
		}
		hasShort = true
		renderGoal(g)
	}
	if !hasShort {
		b.WriteString(dimItemStyle.Render("  none") + "\n")
	}
	b.WriteString(dimItemStyle.Render("press 'g' to add a goal"))

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- meetings panel ---------------------------------------------------------

func (a *App) renderMeetingsPanel() string {
	w, h := leftColumnWidth(), a.meetingsHeight()
	focused := a.focus == panelMeetings

	var b strings.Builder
	p := a.currentPerson()
	shortcut := panelShortcut(panelMeetings)
	title := shortcut + "Meetings"
	if p != nil {
		title = shortcut + "Meetings — " + p.Name
	}
	b.WriteString(panelTitleStyle.Render(title) + "\n\n")

	if p == nil {
		b.WriteString(dimItemStyle.Render("select a person"))
	} else if len(a.meetings) == 0 {
		b.WriteString(dimItemStyle.Render("no meetings yet\npress N to add one now"))
	}

	for i, m := range a.meetings {
		open := len(m.OpenActionItems())
		style := rowStyle(focused, focused && i == a.meetingsCursor, i == a.meetingsIdx)
		// Rendered as two segments sharing style's background (rather than
		// embedding ragStyledGlyph's independently-Rendered string, whose own
		// trailing reset would cancel style's background partway through the
		// line, leaving only the glyph highlighted) so a selected row's
		// highlight covers the whole line.
		glyphStyle := style.Foreground(ragColor(string(m.RAG)))
		line := glyphStyle.Render(ragGlyphChar(string(m.RAG))) + style.Render(" "+relDate(m.Date))
		b.WriteString(line)
		if m.Locked {
			b.WriteString(dimItemStyle.Render("  (locked)"))
		}
		if open > 0 {
			b.WriteString(dimItemStyle.Render(fmt.Sprintf("  (%d open)", open)))
		}
		if len(m.Tags) > 0 {
			b.WriteString("\n  " + tagStyle.Render(strings.Join(m.Tags, ", ")))
		}
		b.WriteString("\n")
	}

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- actions panel ---------------------------------------------------------

// renderActionsPanel lists every action item across every person, most
// recent meeting first, each one annotated with who it belongs to (as
// reference — this panel isn't scoped to the currently selected person).
func (a *App) renderActionsPanel() string {
	w, h := leftColumnWidth(), a.actionsHeight()
	focused := a.focus == panelActions

	var b strings.Builder
	shortcut := panelShortcut(panelActions)
	title := shortcut + "Actions"
	b.WriteString(panelTitleStyle.Render(title) + "\n\n")

	items := a.actionItems
	if len(items) == 0 {
		b.WriteString(dimItemStyle.Render("none — press 'a' to add one"))
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	// Each item takes two lines (the item, then who it belongs to); clip
	// to what actually fits in h so an unbounded item count can't grow
	// this panel past its share (see the comment on actionsHeight()).
	shown, more := items, 0
	if maxItems := (h - 2) / 2; maxItems < 1 {
		shown, more = items[:0], len(items)
	} else if len(items) > maxItems {
		keep := maxItems
		if maxItems > 1 {
			keep = maxItems - 1 // leave a line for the "N more" note
		}
		shown, more = items[:keep], len(items)-keep
	}

	for i, row := range shown {
		box := "[ ]"
		style := openItemStyle
		if row.Item.Done {
			box = "[x]"
			style = doneItemStyle
		}
		line := fmt.Sprintf("%s %s", box, row.Item.Text)
		if focused && i == a.actionIdx {
			b.WriteString(selectedItemStyle.Render(line))
		} else {
			b.WriteString(style.Render(line))
		}
		b.WriteString("\n")
		b.WriteString(dimItemStyle.Render("  " + row.Person.Name + " · " + relDate(row.Meeting.Date)))
		b.WriteString("\n")
	}
	if more > 0 {
		b.WriteString(dimItemStyle.Render(fmt.Sprintf("…and %d more (press 'A')", more)))
	}

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- global notes panel -----------------------------------------------------

// renderGlobalNotesPanel shows a single person-scoped, free-form notes
// block that persists across all of that person's meetings — distinct from
// the Detail panel's notes body below it, which is scoped to one meeting.
func (a *App) renderGlobalNotesPanel() string {
	w := a.width - leftColumnWidth() - 6
	if w < 10 {
		w = 10
	}
	h := a.globalNotesHeight()
	focused := a.focus == panelGlobalNotes

	p := a.currentPerson()
	shortcut := panelShortcut(panelGlobalNotes)
	title := shortcut + "Global Notes"
	if p != nil {
		title = shortcut + "Global Notes — " + p.Name
	}

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render(title) + "\n")

	if p == nil {
		b.WriteString(dimItemStyle.Render("select a person"))
		return panelBorder(focused, title, w, h).Render(b.String())
	}

	if a.modal == modalEditGlobalNotes {
		b.WriteString(a.globalNotesEditArea.View())
	} else {
		b.WriteString(a.globalNotesVP.View())
	}

	return panelBorder(focused, title, w, h).Render(b.String())
}

// --- detail panel ---------------------------------------------------------

func (a *App) renderDetailPanel() string {
	w := a.width - leftColumnWidth() - 6
	if w < 10 {
		w = 10
	}
	h := a.detailHeight()
	focused := a.focus == panelDetail

	p := a.currentPerson()
	m := a.currentMeeting()

	var b strings.Builder
	detailShortcut := panelShortcut(panelDetail)
	if p == nil {
		b.WriteString(panelTitleStyle.Render(detailShortcut+"Meeting notes") + "\n\n")
		b.WriteString(dimItemStyle.Render("nothing selected yet"))
		return panelBorder(focused, "Meeting notes", w, h).Render(b.String())
	}

	if m != nil {
		header := fmt.Sprintf("%sMeeting notes — %s  %s  %s", detailShortcut, p.Name, ragStyledGlyph(string(m.RAG)), m.Date.Format("Mon, Jan 2 2006 · 15:04"))
		headerLine := panelTitleStyle.Render(header)
		if m.Locked {
			headerLine += "  " + lockedBadgeStyle.Render("[locked]")
		}
		b.WriteString(headerLine + "\n")
		if len(m.Tags) > 0 {
			b.WriteString(tagStyle.Render(strings.Join(m.Tags, ", ")) + "\n")
		}
		b.WriteString(strings.Repeat("─", min(w-4, 60)) + "\n")

		b.WriteString(renderMetricRow(focused && a.detailCursor == detailRowRAG,
			"RAG", ragGlyphChar(string(m.RAG)), ragColor(string(m.RAG)), ragLabel(string(m.RAG))) + "\n")
		b.WriteString(renderMetricRow(focused && a.detailCursor == detailRowPerceivedPulse,
			"Perceived Pulse", perceivedPulseGlyphChar(string(m.PerceivedPulse)), perceivedPulseColor(string(m.PerceivedPulse)), m.PerceivedPulse.Label()) + "\n")
		b.WriteString(strings.Repeat("─", min(w-4, 60)) + "\n")

		if a.modal == modalEditNotes {
			b.WriteString(a.editArea.View())
		} else {
			b.WriteString(a.notesVP.View())
		}
	} else {
		b.WriteString(panelTitleStyle.Render(detailShortcut+"Meeting notes — "+p.Name) + "\n\n")
		b.WriteString(dimItemStyle.Render("no meetings yet — press 'N' to add one") + "\n")
	}

	return panelBorder(focused, "Meeting notes", w, h).Render(b.String())
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
	w := a.width - 4
	if w < 10 {
		w = 10
	}
	h := statusGraphsContentHeight

	p := a.currentPerson()
	title := "Detail"
	if p != nil {
		title = "Detail — " + p.Name
	}

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render(title))

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

// renderHelpBar builds a lazygit-style contextual footer: keys that only do
// something in the currently focused panel/section come first, followed by
// the keys that work no matter what has focus.
func (a *App) renderHelpBar() string {
	type kv struct{ k, v string }
	var pairs []kv

	if a.modal == modalEditNotes || a.modal == modalEditGlobalNotes {
		pairs = append(pairs, kv{"ctrl+s", "save"}, kv{"esc", "cancel"})
		var parts []string
		for _, p := range pairs {
			parts = append(parts, helpKeyStyle.Render(p.k)+" "+p.v)
		}
		return helpBarStyle.Width(a.width).Render(strings.Join(parts, "  "))
	}

	if a.modal == modalConfirmDelete {
		pairs = append(pairs, kv{"y/enter", "confirm"}, kv{"esc/n", "cancel"})
		var parts []string
		for _, p := range pairs {
			parts = append(parts, helpKeyStyle.Render(p.k)+" "+p.v)
		}
		return helpBarStyle.Width(a.width).Render(strings.Join(parts, "  "))
	}

	if a.modal == modalRename {
		pairs = append(pairs, kv{"enter", "save"}, kv{"esc", "cancel"})
		var parts []string
		for _, p := range pairs {
			parts = append(parts, helpKeyStyle.Render(p.k)+" "+p.v)
		}
		return helpBarStyle.Width(a.width).Render(strings.Join(parts, "  "))
	}

	switch a.focus {
	case panelPeople:
		pairs = append(pairs,
			kv{"space", "select"}, kv{"→/l/enter", "open person"}, kv{"n", "new person"}, kv{"r", "rename person"}, kv{"d", "delete person"},
		)
	case panelGoals:
		pairs = append(pairs,
			kv{"space", "toggle"}, kv{"←/h/esc", "back"}, kv{"n", "new goal"}, kv{"r", "rename goal"}, kv{"d", "delete goal"},
		)
	case panelMeetings:
		pairs = append(pairs,
			kv{"→/l/enter", "open meeting"}, kv{"←/h/esc", "back"},
			kv{"r", "cycle RAG"}, kv{"n", "new meeting"}, kv{"L", "lock/unlock"}, kv{"d", "delete meeting"},
		)
	case panelActions:
		pairs = append(pairs,
			kv{"space", "toggle"}, kv{"←/h/esc", "back"}, kv{"n", "new action item"}, kv{"r", "rename action item"}, kv{"d", "delete action item"},
		)
	case panelGlobalNotes:
		pairs = append(pairs,
			kv{"←/h/esc", "back"}, kv{"e", "edit notes"}, kv{"E", "$EDITOR"}, kv{"d", "clear notes"},
		)
	case panelDetail:
		pairs = append(pairs,
			kv{"enter/space", "cycle RAG/Pulse"}, kv{"←/h/esc", "back"},
			kv{"w", "add win"}, kv{"n", "new action item"}, kv{"e", "edit notes"}, kv{"E", "$EDITOR"},
			kv{"L", "lock/unlock"}, kv{"d", "delete meeting"},
		)
	}

	pairs = append(pairs,
		kv{"tab", "panel"},
		kv{"?", "help"}, kv{"q", "quit"},
	)

	var parts []string
	for _, p := range pairs {
		parts = append(parts, helpKeyStyle.Render(p.k)+" "+p.v)
	}
	return helpBarStyle.Width(a.width).Render(strings.Join(parts, "  "))
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
		{"Navigation", "tab / shift+tab     cycle panel\n1/2/3/4/5/0         jump to People / Meetings / Global Notes / Goals / Actions / Meeting notes\n↑/k, ↓/j            move cursor within panel\nspace               confirm cursor as selection (People)\n                    (Meetings syncs the Meeting notes panel as you move, no space needed)\n→/l/enter           confirm selection and drill into panel\n←/h/esc             back out"},
		{"People & meetings", "n   new — contextual per panel:\n      People        new person\n      Meetings      new meeting now (opens editor)\n      Goals         new goal (long-term or short-term)\n      Actions,\n      Meeting notes quick-add an action item to the selected meeting\nd   delete — contextual per panel, always asks to confirm:\n      People        delete the selected person and all their data\n      Meetings,\n      Meeting notes delete the selected/current meeting\n      Goals         delete the selected goal\n      Actions       delete the selected action item\n      Global Notes  clear the person's global notes\nr   rename/cycle RAG — contextual per panel:\n      People        rename the selected person\n      Meetings      cycle RAG status (none→green→amber→red)\n      Goals         rename the selected goal\n      Actions       rename the selected action item\nL   lock/unlock — Meetings, Meeting notes:\n      toggle the selected/current meeting's locked state. A locked\n      meeting shows \"(locked)\"/\"[locked]\" and rejects RAG/Pulse\n      changes, notes edits, action items/wins, and deletion — from\n      any panel, including action items reached via the Actions\n      panel — until unlocked again."},
		{"Meeting notes panel", "↑↓/jk       move between the RAG row, Perceived Pulse row, and notes body\n            (scrolls once the cursor reaches the notes)\nenter/space cycle the selected row's value:\n              RAG              none→green→amber→red\n              Perceived Pulse  none→down→steady→up\nw           log a win since the last 1:1 (appended under \"## Wins\")"},
		{"Global Notes panel", "a free-form, per-person notes block that persists across all of that\nperson's meetings — unlike Meeting notes, which belong to one meeting.\n↑↓/jk       scroll"},
		{"Notes", "e   edit notes inline — the selected meeting's notes, or the Global\n    Notes panel's notes if that panel has focus\nE   edit in $EDITOR (same target as e)"},
		{"Action items", "space toggle selected action item in the Actions panel"},
		{"Goals", "space toggle selected goal in the Goals panel"},
		{"Misc", "?   toggle this help\nq / ctrl+c   quit"},
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
