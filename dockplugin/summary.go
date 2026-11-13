package dockplugin

import (
	"fmt"
	"time"

	"github.com/dewald-L/dock/plugin"

	"github.com/dewald-L/lazy1on1/internal/store"
)

// Summary implements plugin.Summarizer for dock's home screen: open action
// items across everyone, and whose 1:1 is due next (by their cadence). It
// only reads the notes, so it is safe to call while lazy1on1 is open.
func (p Plugin) Summary(ctx plugin.Context) (plugin.Summary, error) {
	dir := p.Dir
	if dir == "" {
		dir = ctx.DataDir
	}
	if dir == "" {
		return plugin.Summary{}, fmt.Errorf("no data directory")
	}
	return summary(store.New(dir), time.Now())
}

func summary(s *store.Store, now time.Time) (plugin.Summary, error) {
	people, err := s.ListPeople()
	if err != nil {
		return plugin.Summary{}, err
	}
	if len(people) == 0 {
		return plugin.Summary{Lines: []string{"No people yet"}}, nil
	}
	open := 0
	today := day(now)
	var next *store.Person
	nextDays, nextFirst := 0, false
	for i := range people {
		p := &people[i]
		meetings, err := s.ListMeetings(*p)
		if err != nil {
			return plugin.Summary{}, err
		}
		for _, m := range meetings {
			open += len(m.OpenActionItems())
		}
		// days until the next 1:1 is due; someone never met is due today
		days, first := 0, len(meetings) == 0
		if !first {
			due := day(meetings[0].Date).AddDate(0, 0, p.CadenceDays)
			days = int(due.Sub(today).Hours() / 24)
		}
		if next == nil || days < nextDays {
			next, nextDays, nextFirst = p, days, first
		}
	}

	items := fmt.Sprintf("%d open action items", open)
	switch open {
	case 0:
		items = "No open action items"
	case 1:
		items = "1 open action item"
	}
	who := "1 person"
	if len(people) != 1 {
		who = fmt.Sprintf("%d people", len(people))
	}
	out := plugin.Summary{Lines: []string{items + " · " + who, "Next 1:1: " + next.Name + ", " + dueText(nextDays, nextFirst)}}
	if nextDays < 0 {
		out.Level = plugin.LevelWarn
	}
	return out, nil
}

func dueText(days int, first bool) string {
	switch {
	case first:
		return "first one"
	case days < -1:
		return fmt.Sprintf("%d days overdue", -days)
	case days == -1:
		return "1 day overdue"
	case days == 0:
		return "due today"
	case days == 1:
		return "due tomorrow"
	}
	return fmt.Sprintf("due in %d days", days)
}

// day is t's local calendar day as midnight UTC, so differences are whole days.
func day(t time.Time) time.Time {
	y, m, d := t.In(time.Local).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
