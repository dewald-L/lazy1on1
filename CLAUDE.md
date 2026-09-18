# lazy1on1

Lazygit-style terminal UI (Bubble Tea) for continuous 1:1 note taking. See
`README.md` for feature/keybinding details.

## Commands

```sh
go build -o lazy1on1 .   # build
go vet ./...              # vet
go test ./...             # test (internal/store only, currently)
./lazy1on1 -dir ./tmp     # run against a scratch data dir instead of the default ./data
```

## Architecture

- `main.go` — flag/env parsing (`-dir`, `ONETOONES_DIR`), wires `store` into
  the Bubble Tea program.
- `internal/store/` — markdown + front-matter persistence, action-item
  parsing, cross-person aggregation. No UI concerns.
- `internal/ui/` — Bubble Tea model/update/view for the six-panel layout:
  People / Goals / Meetings / Actions stacked in a left sidebar; Global
  Notes (per-person, not tied to any meeting) stacked above Meeting notes
  (per-meeting notes; on-screen label "Meeting notes", but the panel's
  internal identifiers — `panelDetail`, `renderDetailPanel`, `syncDetail`,
  `handleDetailKey`, etc. — still say "Detail", left that way rather than
  renamed across the codebase) on the right. No file I/O of its own; goes
  through `store`.
- Contextual keys and hints: some keys (e.g. `n`, the "new X for this panel"
  key, and `e`/`E`, "edit notes") dispatch on `a.focus` in `update.go`
  rather than doing one fixed thing — see `newForFocus` and
  `editForFocus`/`editExternalForFocus`. The bottom help bar
  (`renderHelpBar` in `view.go`) mirrors that same `a.focus` switch to show
  the right hint per panel, and `renderHelpModal` documents it in the full
  `?` help screen. These three places (dispatch, help bar, help modal) are
  independent hand-written switches, not generated from one table — when
  adding or changing a contextual key, update all three together or the
  on-screen hints will drift from actual behavior.
- Panel numbering (`1`-`6` jump keys, `tab`/`shift+tab` cycling) is a fixed
  `panelID` enum in `model.go`; adding/removing/reordering a panel means
  updating the enum, `keys.FocusPanel`'s key list, the jump-key `case`s and
  the tab modulo in `update.go`'s `handleKey`, together — they don't derive
  from each other.

## Gotchas

- `internal/store/frontmatter.go` hand-parses front matter — it is
  deliberately **not** a real YAML parser (flat `key: value` lines only, no
  nesting, no lists beyond `a, b, c`). This is intentional so the app has no
  external YAML dependency; don't "fix" it by reaching for a YAML library
  without checking whether the schema actually needs one.
- Data on disk is the source of truth: one folder per person under the data
  dir, one markdown file per meeting, action items detected from
  `- [ ]` / `- [x]` checklist syntax anywhere in the body. Round-tripping
  (read → edit → write) must not reformat content it didn't touch.
- Every per-person aggregate file (`person.yaml`, `goals.md`, `notes.md`)
  lives in the same directory as that person's meeting files and must be
  named in `ListMeetings`' exclusion list in `store.go` — otherwise it gets
  parsed as a meeting. Adding a new one of these files means updating that
  list too.
- There are two distinct panels that render the title/word "Detail":
  the per-meeting notes panel (right side, keyboard-focusable — now labeled
  "Meeting notes" on screen) and the always-visible RAG/Perceived-Pulse bar
  graph panel along the bottom (`renderStatusGraphsPanel` in `view.go`,
  still labeled "Detail"). They are unrelated code paths that happened to
  share a label; when changing one panel's on-screen text, check which of
  the two you actually mean before search-and-replacing "Detail".
- `panelBorder(focused, title, w, h)` in `view.go` takes a `title` argument
  but never renders it — the border itself has no label. Every panel's
  visible heading comes from a `panelTitleStyle.Render(...)` line written
  into the panel's own content (first line(s) of the `strings.Builder`),
  not from the string passed to `panelBorder`. When changing a panel's
  on-screen label, find and edit that content line, not the `panelBorder`
  call — editing only the `panelBorder` title argument changes nothing
  visible.
