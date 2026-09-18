package ui

import "github.com/charmbracelet/lipgloss"

// Palette - a small, calm set of colors reused across the whole UI.
var (
	colorAccent      = lipgloss.Color("#7aa2f7") // selection / titles / tags
	colorBorderFocus = lipgloss.Color("#ffffff") // active panel frame: max contrast against colorBg, distinct from colorAccent
	colorMuted       = lipgloss.Color("#565f89") // unfocused borders / dim text
	colorFg          = lipgloss.Color("#c0caf5")
	colorFgDim       = lipgloss.Color("#7982a9")
	colorGreen       = lipgloss.Color("#9ece6a")
	colorAmber       = lipgloss.Color("#e0af68")
	colorRed         = lipgloss.Color("#f7768e")
	colorGray        = lipgloss.Color("#565f89")
	colorBg          = lipgloss.Color("#1a1b26")
	colorBgSelect    = lipgloss.Color("#283457")
)

func ragColor(rag string) lipgloss.Color {
	switch rag {
	case "green":
		return colorGreen
	case "amber":
		return colorAmber
	case "red":
		return colorRed
	default:
		return colorGray
	}
}

// perceivedPulseColor maps a PerceivedPulse status to a color, reusing the
// RAG palette so red still reads as "bad" and green as "good" - but pulse is
// rendered with arrow glyphs (see perceivedPulseGlyphChar), never RAG's dot,
// so the two signals are never visually confused.
func perceivedPulseColor(pulse string) lipgloss.Color {
	switch pulse {
	case "down":
		return colorRed
	case "up":
		return colorGreen
	case "steady":
		return colorFgDim
	default:
		return colorGray
	}
}

func perceivedPulseGlyphChar(pulse string) string {
	switch pulse {
	case "down":
		return "↓"
	case "steady":
		return "→"
	case "up":
		return "↑"
	default:
		return "·"
	}
}

// ragLabel returns a human-readable label for a RAG status string, for
// display next to the dot glyph in the Detail panel's header row.
func ragLabel(rag string) string {
	if rag == "" {
		return "not set"
	}
	return rag
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorFg)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	// panelStyleFocused marks the active panel with both a heavier border
	// weight and a color change, so the active frame reads clearly even in
	// terminals/themes where colorAccent and colorMuted don't contrast much.
	panelStyleFocused = panelStyle.
				Border(lipgloss.ThickBorder()).
				BorderForeground(colorBorderFocus)

	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)

	// selectedItemStyle marks the confirmed selection (the row currently
	// "open" in another panel) with an inverted, solid-color block rather
	// than a subtle background tint, so it stays visually distinct from
	// cursorItemStyle's plain colored text at a glance.
	selectedItemStyle = lipgloss.NewStyle().
				Background(colorAccent).
				Foreground(colorBg).
				Bold(true)

	// selectedItemStyleDim is selectedItemStyle's quiet counterpart: the
	// confirmed selection in a panel that doesn't currently have focus. It
	// keeps a background tint so the selection stays visible, but drops the
	// inverted block and bold weight so it reads as secondary to whichever
	// panel is actually focused.
	selectedItemStyleDim = lipgloss.NewStyle().
				Background(colorBgSelect).
				Foreground(colorFgDim)

	normalItemStyle = lipgloss.NewStyle().Foreground(colorFg)
	dimItemStyle    = lipgloss.NewStyle().Foreground(colorFgDim)

	// cursorItemStyle marks the navigational highlight in a list panel:
	// where the cursor currently sits, before it's confirmed with space.
	// Underlined in addition to colored/bold so it doesn't rely on hue
	// alone to read as distinct from selectedItemStyle's solid block.
	cursorItemStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Underline(true)

	// selectedCursorItemStyle marks a row that is both the confirmed
	// selection and currently under the cursor: the selection's solid
	// block plus the cursor's underline, so both states remain legible
	// together instead of collapsing into one look.
	selectedCursorItemStyle = selectedItemStyle.Underline(true)

	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorFgDim).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(colorBgSelect).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	doneItemStyle = lipgloss.NewStyle().Foreground(colorFgDim).Strikethrough(true)
	openItemStyle = lipgloss.NewStyle().Foreground(colorFg)

	tagStyle = lipgloss.NewStyle().Foreground(colorAccent)

	errStyle = lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	// lockedBadgeStyle marks a locked meeting, reusing the RAG amber (rather
	// than red) since a lock is a deliberate, reversible restriction, not a
	// warning.
	lockedBadgeStyle = lipgloss.NewStyle().Foreground(colorAmber).Bold(true)
)

func ragGlyphChar(rag string) string {
	if rag == "" {
		return "○"
	}
	return "●"
}

func ragStyledGlyph(rag string) string {
	return lipgloss.NewStyle().Foreground(ragColor(rag)).Render(ragGlyphChar(rag))
}

// ragLevel maps a RAG status to a bar height for the RAG-over-time graph:
// green is healthiest (tallest), red is least healthy, unset is a gap.
func ragLevel(rag string) int {
	switch rag {
	case "green":
		return 3
	case "amber":
		return 2
	case "red":
		return 1
	default:
		return 0
	}
}

// perceivedPulseLevel maps a PerceivedPulse status to a bar height for the
// Perceived Pulse-over-time graph, mirroring ragLevel: up is tallest, down
// is shortest, unset is a gap.
func perceivedPulseLevel(pulse string) int {
	switch pulse {
	case "up":
		return 3
	case "steady":
		return 2
	case "down":
		return 1
	default:
		return 0
	}
}
