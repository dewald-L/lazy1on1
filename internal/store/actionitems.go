package store

import (
	"regexp"
	"strings"
)

var actionItemRe = regexp.MustCompile(`^(\s*[-*]\s\[)([ xX])(\]\s*)(.*)$`)

// parseActionItems scans body lines for markdown checklist items.
func parseActionItems(lines []string) []ActionItem {
	var items []ActionItem
	for i, l := range lines {
		m := actionItemRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		items = append(items, ActionItem{
			Text:   strings.TrimSpace(m[4]),
			Done:   strings.ToLower(m[2]) == "x",
			LineNo: i,
		})
	}
	return items
}

// toggleLine flips the checkbox state of a single body line in place.
func toggleLine(line string) string {
	m := actionItemRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	mark := " "
	if m[2] == " " {
		mark = "x"
	}
	return m[1] + mark + m[3] + m[4]
}

// setItemText replaces a checklist line's text while preserving its
// checkbox marker, mirroring toggleLine (which preserves the text but flips
// the marker). Shared by RenameActionItem and RenameGoal, since goals.md
// checklist lines use the same syntax as action items.
func setItemText(line, text string) string {
	m := actionItemRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	return m[1] + m[2] + m[3] + text
}

// AppendActionItem appends a new open checklist item to the body lines,
// creating an "## Action Items" section if one doesn't already exist.
func appendActionItem(lines []string, text string) []string {
	heading := "## Action Items"
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), heading) {
			// Insert right after the heading (and any existing items that
			// directly follow it).
			insertAt := i + 1
			for insertAt < len(lines) && actionItemRe.MatchString(lines[insertAt]) {
				insertAt++
			}
			out := make([]string, 0, len(lines)+1)
			out = append(out, lines[:insertAt]...)
			out = append(out, "- [ ] "+text)
			out = append(out, lines[insertAt:]...)
			return out
		}
	}
	out := append([]string{}, lines...)
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, heading, "- [ ] "+text)
	return out
}
