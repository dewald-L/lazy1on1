package store

import "time"

// RAG represents a quick red/amber/green health signal for a 1:1 session
// (relationship health, morale, project status - whatever the user wants it
// to mean). Empty string means "not set".
type RAG string

const (
	RAGNone  RAG = ""
	RAGGreen RAG = "green"
	RAGAmber RAG = "amber"
	RAGRed   RAG = "red"
)

// Next cycles none -> green -> amber -> red -> none.
func (r RAG) Next() RAG {
	switch r {
	case RAGNone:
		return RAGGreen
	case RAGGreen:
		return RAGAmber
	case RAGAmber:
		return RAGRed
	default:
		return RAGNone
	}
}

// Glyph returns a single-character indicator for the status.
func (r RAG) Glyph() string {
	switch r {
	case RAGGreen, RAGAmber, RAGRed:
		return "●"
	default:
		return "○"
	}
}

// PerceivedPulse represents a quick perceived-mood signal for a 1:1
// session: how the other person seemed to be doing, independent of RAG
// (which tracks overall health/status). Empty string means "not set".
type PerceivedPulse string

const (
	PerceivedPulseNone   PerceivedPulse = ""
	PerceivedPulseDown   PerceivedPulse = "down"
	PerceivedPulseSteady PerceivedPulse = "steady"
	PerceivedPulseUp     PerceivedPulse = "up"
)

// Next cycles none -> down -> steady -> up -> none.
func (p PerceivedPulse) Next() PerceivedPulse {
	switch p {
	case PerceivedPulseNone:
		return PerceivedPulseDown
	case PerceivedPulseDown:
		return PerceivedPulseSteady
	case PerceivedPulseSteady:
		return PerceivedPulseUp
	default:
		return PerceivedPulseNone
	}
}

// Glyph returns a single-character arrow indicator, deliberately distinct
// from RAG's dot glyphs so the two signals are never visually confused.
func (p PerceivedPulse) Glyph() string {
	switch p {
	case PerceivedPulseDown:
		return "↓"
	case PerceivedPulseSteady:
		return "→"
	case PerceivedPulseUp:
		return "↑"
	default:
		return "·"
	}
}

// Label returns a human-readable label for display.
func (p PerceivedPulse) Label() string {
	switch p {
	case PerceivedPulseDown:
		return "down"
	case PerceivedPulseSteady:
		return "steady"
	case PerceivedPulseUp:
		return "up"
	default:
		return "not set"
	}
}

// Person is one recurring 1:1 counterpart.
type Person struct {
	Slug        string
	Name        string
	Role        string
	Team        string
	CadenceDays int
	Dir         string // absolute path to the person's folder

	GoalLines []string // raw lines of goals.md, split by line
	Goals     []Goal

	Notes string // free-form global notes (notes.md), not tied to any single meeting
}

// ActionItem is a single checklist line ("- [ ] ..." / "- [x] ...") found in
// a meeting's notes body.
type ActionItem struct {
	Text   string
	Done   bool
	LineNo int // index into Meeting.BodyLines, used to toggle in place
}

// Goal is a single checklist line ("- [ ] ..." / "- [x] ...") found under
// the "## Long-term" or "## Short-term" heading in a person's goals.md.
type Goal struct {
	Text   string
	Term   string // "long" or "short"
	Done   bool
	LineNo int // index into Person.GoalLines, used to toggle in place
}

// Meeting is a single dated 1:1 session with a Person.
type Meeting struct {
	PersonSlug     string
	Path           string // absolute path to the markdown file
	Date           time.Time
	RAG            RAG
	PerceivedPulse PerceivedPulse
	Tags           []string
	BodyLines      []string // raw markdown body, split by line, frontmatter stripped

	ActionItems []ActionItem
}

// Body joins BodyLines back into a single markdown string.
func (m *Meeting) Body() string {
	s := ""
	for i, l := range m.BodyLines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}

// OpenActionItems returns the not-yet-done action items.
func (m *Meeting) OpenActionItems() []ActionItem {
	var out []ActionItem
	for _, a := range m.ActionItems {
		if !a.Done {
			out = append(out, a)
		}
	}
	return out
}
