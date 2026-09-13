# hmx

Fork of h-m-m: one PHP file `hmx`, global `$mm`, flat functions, `$keybindings[key] = 'fn'`. Read `foundations.md`, `spec.md`, `architecture.md`, `testing.md` before changing anything. Additions live in the `// {{{ hmx additions` fold.

## Gates
`php -l hmx` · `bash test/run.sh` (unit, includes `hmx` with `HMX_TEST` defined so main never runs) · `bash test/tui.sh` (tmux smoke, fixtures in `test/fixtures/` with real tabs).

## Config
`~/.config/hmx/config` (or `$XDG_CONFIG_HOME/hmx/config`), same `key=value` and `bind…` lines as upstream. Environment overrides use the `hmx_` prefix (`hmx_map_dir=…`), not upstream's `hmm_`. Upstream's `h-m-m.conf` is not read. Map files keep the `.hmm` extension.

## Upstream gotchas
- Layout is horizontal: `l` descends into the first child, `j`/`k` move between siblings. `j` on a single child does nothing (same y as parent).
- `list_to_map()` turns tabs into two spaces before computing depth. Never detect anything by space count; body lines use the `> ` sigil.
- `load_file()` merges into `$mm['nodes']` without clearing it. Reset state via `load_map()`, never call `load_file()` directly to switch maps.
- `magic_readline()` draws on the bottom row and calls `display()` on Esc. `display()` returns early when `$mm['mode'] == 'list'`; keep that guard.
- `display()` and `message()` shell out to `tput`; both are safe under `HMX_TEST` (output is captured by run.sh).
- Node ids from `list_to_map` are sequential from 2 in file order, so tests can address nodes by id.
- `insert_new_sibling` shows `NEW` in the tree while the inline editor is open; tui asserts on that.
- macOS has no `timeout`; smoke tests use tmux (`send-keys`, `capture-pane -p`), see `test/tui.sh`.
- PHP `passthru`/`system` pipe the child's stdout, so `less` dumps and exits and `vim` refuses to start. Anything interactive goes through `run_in_tty()`, which redirects to `/dev/tty` when stdout is a terminal.

## Config
`config($mm, key, default)` resolves CLI `--key=value` → env → conf file → default. Conf file: `~/Library/Preferences/h-m-m/h-m-m.conf` on mac, `~/.config/h-m-m/h-m-m.conf` elsewhere, `key = value` lines. `map_dir` lives there. Manual smoke: `tmux new -d -s t -x 120 -y 30 "php hmm file.hmm"; tmux capture-pane -t t -p`.
