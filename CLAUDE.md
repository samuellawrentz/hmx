# hmm

Fork of h-m-m: one PHP file `hmm`, global `$mm`, flat functions, `$keybindings[key] = 'fn'`. Read `foundations.md`, `spec.md`, `architecture.md`, `testing.md` before changing anything. Additions live in the `// {{{ hmm additions` fold.

## Gates
`php -l hmm` · `bash test/run.sh` (unit, includes `hmm` with `HMM_TEST` defined so main never runs) · `bash test/tui.sh` (tmux smoke, fixtures in `test/fixtures/` with real tabs).

## Upstream gotchas
- Layout is horizontal: `l` descends into the first child, `j`/`k` move between siblings. `j` on a single child does nothing (same y as parent).
- `list_to_map()` turns tabs into two spaces before computing depth. Never detect anything by space count; body lines use the `> ` sigil.
- `load_file()` merges into `$mm['nodes']` without clearing it. Reset state via `load_map()`, never call `load_file()` directly to switch maps.
- `magic_readline()` draws on the bottom row and calls `display()` on Esc. `display()` returns early when `$mm['mode'] == 'list'`; keep that guard.
- `display()` and `message()` shell out to `tput`; both are safe under `HMM_TEST` (output is captured by run.sh).
- Node ids from `list_to_map` are sequential from 2 in file order, so tests can address nodes by id.
- `insert_new_sibling` shows `NEW` in the tree while the inline editor is open; tui asserts on that.
- macOS has no `timeout`; smoke tests use tmux (`send-keys`, `capture-pane -p`), see `test/tui.sh`.
- PHP `passthru`/`system` pipe the child's stdout, so `less` dumps and exits and `vim` refuses to start. Anything interactive goes through `run_in_tty()`, which redirects to `/dev/tty` when stdout is a terminal.
