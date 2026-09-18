# lazy1on1

A lazygit-style terminal UI for continuous 1:1 note taking. Add the people
you meet with, start a new meeting at any time, jot notes as you talk, track
a quick RAG (red/amber/green) status and a Perceived Pulse read per session,
log wins since the last 1:1, keep a running list of action items across
everyone you meet with, and track long-term and short-term goals per
person — all from panels you drive entirely from the keyboard.

## Why plain markdown

Every person and meeting is a plain markdown file on disk. Nothing is locked
in a database:

```
data/
  jane-doe/
    person.yaml              # name / role / cadence
    goals.md                 # long-term / short-term goal checklist
    notes.md                 # free-form global notes, not tied to a meeting
    2026-09-17-2005.md        # one file per meeting
    2026-10-01-1000.md
  john-smith/
    person.yaml
    2026-09-20-0930.md
```

That means you can `grep` across every 1:1 you've ever had, put the whole
directory under git for history and backup, or edit a file by hand in your
usual editor — `lazy1on1` will pick up the change next time it reads the file.

Override the data directory with `-dir` or the `ONETOONES_DIR` environment
variable; it defaults to a `data` folder next to the `lazy1on1` binary.

## A meeting file

```markdown
---
date: 2026-09-17T20:05:37Z
rag: green
pulse: up
tags: performance, career
---
## Notes

Discussed Q3 goals and career growth.
Follow-up scheduled for next sprint.

## Wins
- Shipped the migration a week early

## Action Items
- [ ] Send promo doc by Friday
- [x] Share feedback survey
```

Checklist lines (`- [ ]` / `- [x]`) anywhere in the body are recognized as
action items automatically, whether you typed them yourself or added them
through the app. Plain bullets (`- `) under `## Wins` work the same way for
wins — logged, not checked off.

## A goals file

Each person's `goals.md` uses the same checklist syntax, under two fixed
headings:

```markdown
## Long-term
- [ ] Get promoted to senior by end of year

## Short-term
- [ ] Finish onboarding new hire
- [x] Ship the Q3 report
```

## A notes file

Each person's `notes.md` is plain free-form text — no fixed structure, no
checklist syntax required — for anything worth remembering that isn't scoped
to one particular session: background, career context, running
observations. It's shown in the Global Notes panel, separate from any one
meeting's own notes.

## Running it

```sh
go build -o lazy1on1 .
./lazy1on1                      # uses ./data, next to the binary
./lazy1on1 -dir ./my-1-1-notes  # or point it somewhere else
```

Go 1.21+ is required. There are no runtime dependencies beyond your
terminal and (optionally) `$EDITOR` for full external editing.

## Layout

Like lazygit's multi-pane layout: **People**, **Goals**, **Meetings** and
**Actions** stack vertically in a left-hand sidebar (like lazygit's Branches
panel sitting above its Commits panel). To the right, **Global Notes** stacks
above **Meeting notes**, filling the rest of the width. A third panel —
titled **Detail** — spans the full width along the bottom.

- **People** — everyone you have recurring 1:1s with.
- **Goals** — the selected person's Long-term / Short-term goals checklist
  (per-person, so it stays the same as you switch between that person's
  meetings).
- **Meetings** — every dated session with the selected person, most recent
  first, with a RAG dot, an open-action-item count, and `(locked)` for any
  meeting that's been locked.
- **Actions** — every action item across all of the selected person's
  meetings, most recent meeting first, each one showing which meeting it
  came from.
- **Global Notes** (right, top, keyboard-focusable — `5`) — free-form notes
  about the selected person that persist across all of their meetings, not
  tied to any single one (e.g. background, career context, running
  observations). Separate from a meeting's own notes below it.
- **Meeting notes** (right, bottom, keyboard-focusable — `6`) — the selected
  meeting's RAG and Perceived Pulse status (interactive header rows), wins
  since the last 1:1, and that meeting's own notes.
- **Detail** (bottom, always visible) — RAG and Perceived Pulse over time
  for the selected person, as two bar graphs side by side.

## Keybindings

The bottom help bar is contextual, lazygit-style: it always leads with the
keys that do something in whichever panel currently has focus, followed by
the keys that work no matter where focus is. `n` ("new") is the clearest
example: it has no global meaning of its own and only exists per panel —
new person in People, new goal in Goals, new meeting in Meetings, quick-add
an action item in Actions/Meeting notes — see the per-panel tables below. `e`/`E`
("edit notes" / edit in `$EDITOR`) work the same way: they edit the Global
Notes panel's notes when that panel has focus, and the selected meeting's
notes everywhere else. `d` ("delete") is contextual the same way too —
deletes whatever's selected in the focused panel, always behind a
confirmation prompt since every one of these is a permanent disk write; see
the per-panel tables below.

Global, from any panel:

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Cycle focus between panels |
| `1` / `2` / `3` / `4` / `5` / `6` | Jump straight to People / Goals / Meetings / Actions / Global Notes / Meeting notes |
| `r` | Cycle the selected meeting's RAG status: none → green → amber → red |
| `?` | Help |
| `q` / `ctrl+c` | Quit |

**People** panel:

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Move between people |
| `→`/`l`/`enter` | Open the selected person's meetings |
| `n` | Add a new person |
| `d` | Delete the selected person (and all their meetings, goals, notes) |

**Goals** panel:

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Move between goals |
| `space` | Toggle the selected goal done/open |
| `←`/`h`/`esc` | Back to People |
| `n` | Add a goal to the selected person (long-term or short-term) |
| `d` | Delete the selected goal |

**Meetings** panel:

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Move between meetings |
| `→`/`l`/`enter` | Open the selected meeting |
| `←`/`h`/`esc` | Back to People |
| `n` | Start a new meeting right now, and jump straight into notes |
| `L` | Lock/unlock the selected meeting |
| `d` | Delete the selected meeting |

**Actions** panel:

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Move between action items |
| `space` | Toggle the selected action item done/open |
| `←`/`h`/`esc` | Back to Meetings |
| `n` | Quick-add an action item to the currently selected meeting |
| `d` | Delete the selected action item |

**Global Notes** panel (free-form notes about the selected person, not tied
to any meeting):

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Scroll |
| `←`/`h`/`esc` | Back to Meetings |
| `e` | Edit notes inline (small built-in editor) |
| `E` | Edit `notes.md` in `$EDITOR` |
| `d` | Clear the person's global notes |

**Meeting notes** panel (RAG/Perceived Pulse header rows, wins, and notes):

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | Move the row cursor: RAG → Perceived Pulse → notes body; scrolls once the cursor reaches the notes |
| `enter`/`space` | Cycle the selected row's value (RAG or Perceived Pulse) |
| `←`/`h`/`esc` | Back to Meetings |
| `w` | Log a win since the last 1:1 |
| `e` | Edit notes inline (small built-in editor) |
| `E` | Edit the meeting file in `$EDITOR` |
| `n` | Quick-add an action item to the selected meeting |
| `L` | Lock/unlock the current meeting |
| `d` | Delete the currently selected meeting |

While either inline notes editor is open: `ctrl+s` saves, `esc` discards.

Deleting anything (`d`) always opens a confirmation prompt first: `y`/`enter`
confirms, `esc`/`n` cancels.

## Locking a meeting

`L` toggles a lock on the currently selected meeting, from either the
Meetings panel or the Meeting notes panel. A locked meeting is marked
`(locked)` in the Meetings list and `[locked]` in the Meeting notes header,
and rejects every other mutation until it's unlocked again: RAG/Perceived
Pulse changes, editing notes (inline or in `$EDITOR`), adding/renaming/
toggling/deleting its action items or wins (including from the Actions
panel), and deleting the meeting itself. It's meant for sessions you want to
treat as a finished, frozen record — lock it once you're done, unlock with
`L` again if you need to fix something.

## What "RAG" and "Perceived Pulse" mean here

**RAG** is a deliberately loose red/amber/green signal — use it for whatever
you want to track at a glance across sessions: relationship health, morale,
project status, anything. It's a colored dot, cycled with `r` from anywhere
or from the RAG row in the Meeting notes panel.

**Perceived Pulse** is a separate, independent read on how the person seemed
to be doing in this specific session — down / steady / up, shown as an arrow
so it's never confused with the RAG dot. Cycle it from the Perceived Pulse
row in the Meeting notes panel.

Both are plotted together, oldest to newest, in the **Detail** panel along
the bottom of the screen — one bar graph per metric, side by side, each
with its own date axis, so you can see whether the two track together or
diverge for the selected person.

## Project layout

```
main.go                   entry point, flag/env handling
internal/store/            markdown + frontmatter persistence, action-item
                            and goal checklist parsing, cross-person
                            aggregation (no external YAML dependency — the
                            frontmatter schema is small and fixed, so it's
                            hand-parsed)
internal/ui/                Bubble Tea model: panels, keybindings, modals
```

## Ideas for extending it

- A `tags` filter or fuzzy search across people/meetings.
- Cadence reminders ("you haven't met with X in 3 weeks") using
  `Person.CadenceDays`, which is already tracked but not yet surfaced.
- Exporting a person's full history to a single markdown file.
- A recurring-meeting template (pre-filled agenda sections).
