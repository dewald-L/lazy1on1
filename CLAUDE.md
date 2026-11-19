# lazy1on1

Lazygit-style terminal UI (Bubble Tea) for continuous 1:1 note taking. See
`README.md` for feature/keybinding details.

## Commands

```sh
go build -o lazy1on1 .   # build
go vet ./...              # vet
go test ./...             # test (internal/store and dockplugin's Summary)
./lazy1on1 -dir ./tmp     # run against a scratch data dir instead of the default ./data
```

## Architecture

- Module `github.com/dewald-L/lazy1on1` (was `oto`). It depends on
  `github.com/dewald-L/dock/plugin` at a published commit (bump it with
  `go get github.com/dewald-L/dock@<commit>`). To develop against unpushed dock
  changes, use a temporary `go.work`, not a committed replace. dock does not import
  lazy1on1. `dockplugin`'s `init()` calls `plugin.Register`, and dock compiles
  it in with `dock plugin install [--path ~/Projects/oto]
  github.com/dewald-L/lazy1on1/dockplugin`. After changing the plugin, run
  `dock build`.
- `main.go` — flag/env parsing (`-dir`, `ONETOONES_DIR`). Runs
  `dockplugin.Plugin{Standalone: true}` through `plugin.RunStandalone`, so
  standalone and dock share one code path.
- `dockplugin/` — the dock plugin (id `lazy1on1`, DisplayFull, Persistent,
  min 70x24). Under dock, data lives in dock's plugin data dir
  (`ctx.DataDir`, ~/.local/share/dock/lazy1on1), not the standalone `data/`
  folder or `$ONETOONES_DIR`. `Plugin.Dir` overrides it (main sets it for
  standalone). `summary.go` is dock's home screen `plugin.Summarizer` (on the
  Plugin value): open action items across everyone, and whose 1:1 is due next
  (last meeting + `cadence_days`; never met = "first one"), warn when overdue.
  It runs off the UI goroutine, possibly while lazy1on1 is open: read-only
  through `store`, never `EnsureRoot`. Tests call `summary(store, now)` on
  temp dirs.
- Embedded mode (`App.embedded`, set unless Standalone): our status and help
  bars are hidden (`helpBarHeight()`/`statusBarHeight()` return 0).
  `Update` wraps `update` and forwards `status`/`errMsg` to dock
  (`plugin.Notify`/`plugin.Error`), then clears them. It also sends
  `plugin.SetTitle` when the selected person changes. `internal/ui/dock.go`
  holds `Help()` (built from `helpPairs`) and the palette `Commands()`.
  `$EDITOR` runs through `plugin.Exec`, never `tea.ExecProcess`, because the
  host must route the result back. Don't call `tea.NewProgram` or touch the
  terminal from `ui`; see ../dock/docs/PLUGINS.md.
- `internal/store/` — markdown + front-matter persistence, action-item
  parsing, cross-person aggregation. No UI concerns.
- `internal/ui/` — Bubble Tea model/update/view for the six-panel layout:
  People / Meetings / Goals / Actions stacked in a left sidebar; Global
  Notes (per-person, not tied to any meeting) stacked above Meeting notes
  (per-meeting notes; on-screen label "Meeting notes", but the panel's
  internal identifiers — `panelDetail`, `renderDetailPanel`, `syncDetail`,
  `handleDetailKey`, etc. — still say "Detail", left that way rather than
  renamed across the codebase) on the right. No file I/O of its own; goes
  through `store`.
- Contextual keys and hints: some keys (e.g. `n`, the "new X for this panel"
  key, and `e`/`E`, "edit notes") dispatch on `a.focus` in `update.go`
  rather than doing one fixed thing — see `newForFocus` and
  `editForFocus`/`editExternalForFocus`. The key hints (`helpPairs` in
  `view.go`, which feeds both the standalone help bar and dock's `Help()`)
  mirror that same `a.focus` switch to show the right hint per panel, and
  `renderHelpModal` documents it in the full
  `?` help screen. These three places (dispatch, help bar, help modal) are
  independent hand-written switches, not generated from one table — when
  adding or changing a contextual key, update all three together or the
  on-screen hints will drift from actual behavior.
- Panel numbering (`1`-`4`/`0` jump keys, `tab`/`shift+tab` cycling) is a fixed
  `panelID` enum in `model.go`; adding/removing/reordering a panel means
  updating the enum, `keys.FocusPanel`'s key list, the jump-key `case`s and
  the tab modulo in `update.go`'s `handleKey`, together — they don't derive
  from each other. `panelPeople` (enum 0) is the picker modal, not a panel:
  it has no jump key (`keys.FocusPanel` and `panelShortcut` are offset by one)
  and tab cycling skips it.

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
