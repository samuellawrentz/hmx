# hmx

Go + tcell port of h-m-m: one `package main`, flat files, one `App` struct, `actions` (name → func) and `mapKeys`/`listKeys` (key → name) tables. Read `foundations.md`, `spec.md`, `architecture.md`, `testing.md` before changing anything. Files per architecture.md; ported functions keep the upstream PHP names in camelCase.

## Build and gates
`go build -o hmx .` · `go vet ./...` · `go test ./...` (unit + golden layout) · `bash test/tui.sh` (tmux smoke, 9 scenarios, `HMX_BIN` defaults to `./hmx`). Fixtures in `test/fixtures/` use real tabs. Deps: tcell/v2 and go-runewidth only; ask before adding one.

## Config
`~/.config/hmx/config` (or `$XDG_CONFIG_HOME/hmx/config`), `key = value` lines, `bind<key> = action` rebinds (key names are tcell's: `Ctrl-E`, `Space`, `Backspace2`). Env `hmx_<key>`, flag `--key=value`; flag > env > file > default. `map_dir` defaults to `~/maps`. Map files keep the `.hmm` extension.

## Gotchas
- Layout is horizontal: `l` descends into the nearest child, `j`/`k` move between siblings. `j` on a single child does nothing (same y as parent).
- `listToMap` turns tabs into two spaces before computing depth. Never detect anything by space count; body lines use the `> ` sigil.
- Node ids from `listToMap` are sequential from 2 in file order, so tests address nodes by id. Node 0 is the hidden super-root; a file with several top-level lines gets a synthetic `root` node 1.
- `Serialize` ends with exactly one newline (the PHP v1 wrote a trailing blank line; files saved by v1 lose it once).
- Layout goldens in `test/golden/` are frozen 120x40 captures of the PHP v1 (`test/golden/gen.sh`, private tmux socket with the status bar off because `tmux -y N` gives an N-1 row pane). The PHP reference lives in git history: `git show php-ref:ref/hmx.php`. Never regenerate goldens from Go.
- `readline` (inline editor), the list screen, prompts, and `$EDITOR` round-trips are unit-tested through `tcell.NewSimulationScreen` + `InjectKey`; `Suspend`/`Resume` are no-ops there.
- Anything interactive (`$EDITOR`, `task … info | $PAGER`) runs between `screen.Suspend()` and `Resume()` with os.Stdin/Stdout.
- Clipboard: yank/cut pipe to `pbcopy`, paste reads `pbpaste`; internal string when `pbcopy` is not on PATH. Tests put fake scripts on PATH and must use absolute `/bin/cat` inside them.
- `insertNewSibling` shows `NEW` in the tree while the inline editor is open; tui asserts on that.
- macOS has no `timeout`; smoke tests use tmux (`send-keys`, `capture-pane -p`). `run()` polls until the screen is non-blank because the first exec of a fresh build is slow.
- Manual smoke: `tmux new -d -s t -x 120 -y 30 "./hmx --map-dir=/tmp/x /tmp/x/file.hmm"; tmux capture-pane -t t -p`. Never point it at `test/fixtures/` directly: `q` saves.
