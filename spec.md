# Spec

## Screens

### List (no file argument)
- Rows: `~/maps/*.hmm`. `inbox` pinned first, then mtime desc. Columns: name, node count, age.
- Rows nest by links: a map is indented two spaces under the first map (in row order) whose file contains `[[map]]` or `[[map#`; maps nothing links to are roots; a cycle no root reaches is appended at root level. Each map appears once, so the cursor stays flat.
- Node count = lines not starting with the body sigil.
- `/` filters names live, keeping matches and their ancestors. Enter on the filter with no name hit → `grep -il` contents, open the map with in-map search pre-filled.

| key | action |
|---|---|
| `j k` ↑↓ | move |
| `Enter` | open |
| `n` | new map |
| `r` | rename. Rewrites `[[old` → `[[new` across all maps, reports files touched |
| `d` | delete, confirm |
| `/` | filter |
| `q` | quit |

Task picker (`t` on a map node): same screen, rows are pending tasks (`description`, `project`, `due`) by urgency desc, `/` filters description and project.

| key | action |
|---|---|
| `j k` ↑↓ | move |
| `Enter` | link: title becomes `<description> [[task:uuid8]]` (an existing link is replaced, rest of the title kept), task annotated `map: <map>#<title>` |
| `n` | create a task from the node title, then link it |
| `/` | filter |
| `q` | cancel |

### Map
Existing h-m-m view plus: breadcrumb with stack depth, `[[..]]` titles blue, `…` amber when a body exists,
bottom pane shows the body when the active node has one.

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

### Link autocomplete
Typing `[[` while the inline editor is open (`e`, `o`, `Tab`) opens a filter popup above the edit line.
Candidates are every non-body line of every map in `map_dir`, scanned once per popup: the root line inserts
`[[map]]` and shows `map` as its context, deeper lines insert `[[map#title]]` and show the ancestor path
(`projects › Q3`). Maps come in list-screen order, their nodes in file order.
Typing filters by subsequence over title and context (`q3 red` finds `reduce cx cost` under `projects › Q3`).

| key | action |
|---|---|
| any rune, backspace | edit the line and refilter |
| `ctrl+n` ↓ / `ctrl+p` ↑ | next / previous candidate |
| `Tab` `Enter` | accept, inserting the whole link |
| `Esc` | close the popup, keep what was typed |

## Links
- Syntax: `[[map]]`, `[[map#node title]]`. Looked up in the title first, then body, first match wins.
- Follow: push `{file, active_node, viewport}`, save if modified, load target.
  `#node`: exact title, else substring, else root + message.
- Missing target: `create <map>.hmm? [y/N]`.
- Backspace pops the stack. Empty stack → nothing.

## Extract (`ctrl+e`)
- Key = `<parent-map>-<slug(title)>`. Same key for filename, `[[link]]`, root title. Prompt allows override.
- Existing file → refuses with a message.
- Subtree written via the existing serializer, node title replaced by `[[key]]`, children dropped. Undo covers the tree change, not the file.
- Reverse is manual: open child, `y` root, Backspace, `p`, delete link node.

## Large nodes
- Node = title + optional body. Tree shows title only.
- `E` writes `title\n\nbody` to a temp file, runs `$EDITOR`, reparses. Modified flag set.
- Body pane: bottom 30% when active node has a body. No toggle.

## File format
```
auth
	Tokens
		JWT rotation
		> Rotate every 24h. Old key stays valid 1h.
		> See [[security]].
		refresh
	[[infra#redis]]
```
- Node: N tabs + title. Body: N tabs + `> ` + text, directly under its node.
- Tabs only. Normalised on save. Upstream h-m-m shows body lines as children titled `> …`.

## Taskwarrior (later)
Stage A, read-only: `[[task:uuid8]]` in a title. One `task rc.context=none <uuids> export` at map open.
Render `☐` pending, `☑` done, red when overdue. Enter → `task <uuid> info` in a pager.

Stage B: `t` picker over pending tasks (list screen reused); Enter links, title becomes the task description,
task gets annotation `map: <map>#<node>`. `n` in picker creates from title (`add`, uuid via `+LATEST _uuids`).
Writes use uuid + `rc.confirmation=off rc.verbose=nothing`. `task` missing or failing → one-line message, node untouched.
Later, only if B is used daily: `T` marks done.

## Config
`map_dir` (default `~/maps`). Everything cut from the keymap is bindable via the config file (`~/.config/hmx/config`, or `$XDG_CONFIG_HOME/hmx/config`).
