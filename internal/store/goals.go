package store

import "strings"

const goalHeadingLong = "## Long-term"
const goalHeadingShort = "## Short-term"

// parseGoals scans goals.md lines for markdown checklist items, tagging each
// with whichever of the "## Long-term" / "## Short-term" headings precedes
// it. Reuses actionItemRe so the checklist syntax stays identical to action
// items.
func parseGoals(lines []string) []Goal {
	var goals []Goal
	term := ""
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		switch {
		case strings.EqualFold(trimmed, goalHeadingLong):
			term = "long"
			continue
		case strings.EqualFold(trimmed, goalHeadingShort):
			term = "short"
			continue
		}
		m := actionItemRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		goals = append(goals, Goal{
			Text:   strings.TrimSpace(m[4]),
			Term:   term,
			Done:   strings.ToLower(m[2]) == "x",
			LineNo: i,
		})
	}
	return goals
}

// toggleGoalLine flips the checkbox state of a single goals.md line in
// place, mirroring toggleLine.
func toggleGoalLine(line string) string {
	return toggleLine(line)
}

// appendGoal appends a new open goal under the given term's heading
// ("long" or "short"), creating that heading at the end of the file if it
// isn't already present.
func appendGoal(lines []string, term, text string) []string {
	heading := goalHeadingLong
	if term == "short" {
		heading = goalHeadingShort
	}
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), heading) {
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
