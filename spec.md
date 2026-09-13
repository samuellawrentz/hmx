# Spec

## Screens

### List (no file argument)
- Rows: `~/maps/*.hmm`. `inbox` pinned first, then mtime desc. Columns: name, node count, age.
- Node count = lines not starting with the body sigil.
- `/` filters names live. Enter on the filter with no name hit → `grep -il` contents, open the map with in-map search pre-filled.

| key | action |
|---|---|
| `j k` ↑↓ | move |
| `Enter` | open |
| `n` | new map |
| `r` | rename. Rewrites `[[old` → `[[new` across all maps, reports files touched |
| `d` | delete, confirm |
| `/` | filter |
| `q` | quit |

### Map
Existing h-m-m view plus: breadcrumb with stack depth, `[[..]]` titles blue, `…` amber when a body exists,
bottom pane shows the body when the active node has one.

| key | action |
|---|---|
| `h j k l` ↑↓←→ | move |
| `Space` | collapse / expand |
| `f` | focus: collapse others, center |
| `0` | expand all |
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

Stage B, only if A is used daily: `t` picker over pending tasks (list screen reused); Enter links,
title becomes the task description, task gets annotation `map: <map>#<node>`. `n` in picker creates from title.
`T` marks done. Writes use uuid + `rc.confirmation=off rc.bulk=0 rc.verbose=nothing`.

## Config
`map_dir` (default `~/maps`). Everything cut from the keymap is bindable via the config file (`~/.config/hmx/config`, or `$XDG_CONFIG_HOME/hmx/config`).
