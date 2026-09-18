package store

import (
	"regexp"
	"strings"
)

const winsHeading = "## Wins"
const actionItemsHeadingText = "## Action Items"

var winBulletRe = regexp.MustCompile(`^\s*[-*]\s`)

// appendWin appends a new "win since last 1:1" bullet to the body lines,
// creating a "## Wins" section if one doesn't already exist. Wins are plain
// bullets ("- text"), not checklist items - a win is logged, not completed.
//
// A brand-new "## Wins" heading is inserted directly above "## Action Items"
// when that section exists, rather than at the end of the file: the Detail
// panel's notes viewport stops rendering at "## Action Items" (see
// notesOnly in internal/ui), so a heading appended after it would silently
// never show up on screen.
func appendWin(lines []string, text string) []string {
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), winsHeading) {
			insertAt := i + 1
			for insertAt < len(lines) && winBulletRe.MatchString(lines[insertAt]) {
				insertAt++
			}
			out := make([]string, 0, len(lines)+1)
			out = append(out, lines[:insertAt]...)
			out = append(out, "- "+text)
			out = append(out, lines[insertAt:]...)
			return out
		}
	}

	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), actionItemsHeadingText) {
			out := make([]string, 0, len(lines)+3)
			out = append(out, lines[:i]...)
			out = append(out, winsHeading, "- "+text, "")
			out = append(out, lines[i:]...)
			return out
		}
	}

	out := append([]string{}, lines...)
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, winsHeading, "- "+text)
	return out
}
