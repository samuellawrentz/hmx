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
| `c` | center the active node |
| `0` / `1` | expand all / collapse all |
| `o` | new sibling |
| `Tab` | new child |
| `e` | edit title |
| `E` | edit title + body in `$EDITOR`, first line = title |
| `d` | delete subtree to clipboard |
| `y` | yank subtree |
| `D` | cut the children of the node to the clipboard |
| `Y` | yank the children of the node |
| `p` / `P` | paste as children / as siblings |
| `J` / `K` | move node down / up |
| `u` | undo |
| `/` `n` `N` | search, next, prev |
| `Enter` | node has a link → follow it. Else → new sibling |
| `Backspace` | back to previous map |
| `ctrl+e` | extract subtree to its own map |
| `t` | task picker: link or create a Taskwarrior task |
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
- Typing `[[` while editing a title opens a completion popup over every node of every map; arrows or `ctrl+n`/`ctrl+p` move, `Tab` or `Enter` inserts the finished link, `Esc` closes it.
- The popup filters by subsequence over the ancestor path and the title, one term at a time, so `projects red` finds `reduce cx cost` under `projects › Q3`. Closer title matches rank first, so short queries still surface the obvious node.
- `#` scopes the query: `projects#` lists that map's nodes, `proj#red` narrows them. Accepting inserts the whole link, so typing `#` yourself never doubles it.

## Taskwarrior
When a map opens, hmx runs one `task rc.context=none <uuids> export` for every `[[task:uuid8]]` in it and renders `☐` pending, `☑` done, red when overdue. Enter on such a node shows `task <uuid> info` in `$PAGER`. Without a `task` binary everything renders as plain `☐`.

`t` on a node opens a picker over pending tasks (description, project, due, by urgency). Enter links the task: the title becomes `<description> [[task:uuid8]]` and the task gets the annotation `map: <map>#<title>`. `n` creates a task from the node title and links it. Nothing else writes to Taskwarrior.

## Credits
Based on [h-m-m](https://github.com/nadrad/h-m-m) by nadrad. GPL-3, see [LICENSE](LICENSE).
