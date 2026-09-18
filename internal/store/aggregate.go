package store

import "sort"

// GlobalActionItem is an action item annotated with which person and
// meeting it came from, for cross-person aggregation.
type GlobalActionItem struct {
	Person  Person
	Meeting Meeting
	Item    ActionItem
}

// AllActionItems scans every person's every meeting and returns every
// checklist item, done or not, most recent meeting first. This backs the
// sidebar Actions panel, which lists everyone's action items rather than
// just the currently selected person's.
func (s *Store) AllActionItems() ([]GlobalActionItem, error) {
	people, err := s.ListPeople()
	if err != nil {
		return nil, err
	}
	var out []GlobalActionItem
	for _, p := range people {
		meetings, err := s.ListMeetings(p)
		if err != nil {
			continue
		}
		for _, m := range meetings {
			for _, item := range m.ActionItems {
				out = append(out, GlobalActionItem{Person: p, Meeting: m, Item: item})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Meeting.Date.After(out[j].Meeting.Date)
	})
	return out, nil
}
