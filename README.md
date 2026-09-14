# hmx

Terminal mind maps. Many maps, one folder, plain text, links between them. Fork of [h-m-m](https://github.com/nadrad/h-m-m).

- [foundations.md](foundations.md) rules and scope
- [spec.md](spec.md) screens, keys, file format
- [architecture.md](architecture.md) state, functions, flow
- [docs/plan.html](docs/plan.html) wireframes and review log

## Install
Needs Go 1.26 or newer.
```
go install github.com/samuellawrentz/hmx@latest
```
The binary lands in `$(go env GOPATH)/bin`. Or build from a clone:
```
git clone https://github.com/samuellawrentz/hmx && cd hmx && go build -o hmx .
```

## Quick start
- `hmx` opens the list of maps in `map_dir`. `hmx file.hmm` opens one map.
- First run with no config: `~/maps` is created if missing and the list shows `no maps`. Press `n`, type a name, Enter: the map opens with its root node. `q` saves and returns to the list.
- Nodes are titles in a tree. `l` goes into a node, `h` back out, `j`/`k` between siblings, `o` adds a sibling, `Tab` a child, `e` edits the title.

## Config
`~/.config/hmx/config` (or `$XDG_CONFIG_HOME/hmx/config`), one `key = value` per line:
- `map_dir` — folder of `.hmm` files, default `~/maps`.
- `bind<key> = action` — rebind a key, e.g. `bindx = expand_all`. Key names are tcell's: `Space`, `Tab`, `Ctrl-E`, `Backspace2`. Action names are the ones shown by `?`.

Every key can also be set by environment (`hmx_map_dir=…`) or flag (`--map-dir=…`); flag beats env beats file.

## Keys
List screen:

| key | action |
|---|---|
| `j k` ↑↓ | move |
| `Enter` | open |
| `n` | new map |
| `r` | rename. Rewrites `[[old` → `[[new` across all maps, reports files touched |
| `d` | delete, confirm |
| `/` | filter |
| `q` | quit |

Map:

| key | action |
|---|---|
| `h j k l` ↑↓←→ | move |
| `Space` | collapse / expand |
| `f` | focus: collapse others, center |
| `0` / `1` | expand all / collapse all |
| `o` | new sibling |
| `Tab` | new child |
| `e` | edit title |
| `E` | edit title + body in `$EDITOR`, first line = title |
| `d` | delete subtree to clipboard |
| `y` | yank subtree |
| `p` / `P` | paste as children / as siblings |
| `J` / `K` | move node down / up |
| `u` | undo |
| `/` `n` `N` | search, next, prev |
| `Enter` | node has a link → follow it. Else → new sibling |
| `Backspace` | back to previous map |
| `ctrl+e` | extract subtree to its own map |
| `s` | save |
| `q` | back to list, saves if modified |
| `?` | help |

## File format
One `.hmm` per map, readable by upstream h-m-m and by `cat`:
- One node per line, depth = number of leading tabs.
- A body line is `> ` after the tabs, directly under its node.
- `[[map]]` links to `map.hmm` in `map_dir`; Enter follows it, Backspace comes back. Missing targets are created after a `[y/N]` prompt.
- `[[map#node title]]` lands on that node (exact title, else substring).
- `[[task:uuid8]]` marks a Taskwarrior task by the first 8 characters of its uuid.

## Taskwarrior
Read-only. When a map opens, hmx runs one `task rc.context=none <uuids> export` for every `[[task:uuid8]]` in it and renders `☐` pending, `☑` done, red when overdue. Enter on such a node shows `task <uuid> info` in `$PAGER`. Without a `task` binary everything renders as plain `☐`.

## Credits
Based on [h-m-m](https://github.com/nadrad/h-m-m) by nadrad. GPL-3, see [LICENSE](LICENSE).
