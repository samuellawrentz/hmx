# Architecture (Go)

One `package main`, flat files, one `App` struct where the PHP had `$mm`. No interfaces with one implementation.

## Files
```
main.go      flags, config, mode loop (list | map), tcell screen lifecycle
model.go     Node, Map: parse (.hmm with `> ` bodies) / serialize, ids, tree ops (insert, delete, move, yank/paste), undo stack
layout.go    port of PHP calculate_x_and_lh / calculate_h / calculate_y / calculate_xo / build_map → cell grid with connectors
render.go    grid → tcell (colours, active highlight, link/marker/task styling), viewport, breadcrumb, body pane, message line
keys.go      key → action table (24 map keys, 7 list keys), inline line editor (port of magic_readline)
links.go     parse [[map#node]] / [[task:uuid8]], follow, back (nav stack), confirm-create
list.go      list screen: scan map_dir, inbox first, mtime, node count, filter, content-grep fallback, rename (rewrites inbound links)
extract.go   extract subtree to <map>-<slug>.hmm
body.go      $EDITOR round-trip (suspend tcell), body pane
tasks.go     taskwarrior stage A: one `task export`, ☐/☑/overdue
help.go      help screen generated from the key table
```

## State
```go
type App struct {
    mode    string            // "list" | "map"
    file    string
    nodes   map[int]*Node     // id → node; 0 = hidden super-root, root = 1 or 2 as in PHP
    root, active int
    stack   []NavEntry        // {file, active, viewTop, viewLeft}
    grid    [][]Cell          // laid-out map, rebuilt after every mutation
    view    struct{ top, left int }
    undo    []Snapshot
    tasks   map[string]TaskStatus
    list    struct{ items []MapInfo; filter string; cursor int }
    cfg     Config            // map_dir, colours, widths; file → env hmx_* → flags
    modified bool
}
```
Node ids stay sequential from 2 in file order (tests address nodes by id, same as PHP).

## Flow
```
main
  no arg → mode=list : list.go loop → Enter → openMap
  map loop: tcell.PollEvent → keys.go table → action(app) → layout.Build → render.Draw
    Enter     → link in node? followLink : insertSibling
    Backspace → goBack
    ctrl+e    → extractToMap
    E         → editBody (screen.Suspend, $EDITOR, Resume)
    q         → save if modified, mode=list
```

## Layout port rules
- Ported function by function from the PHP v1 (`git show php-ref:ref/hmx.php`: calculate_x_and_lh 748, calculate_aligned_x 826, calculate_h 863, calculate_y 912, calculate_children_y 922, calculate_height_shift 960, calculate_xo 1221, build_map 1298). Same names, same order, same arithmetic. Do not redesign the algorithm.
- Width = runewidth.StringWidth. Wrap = same word-wrap semantics as PHP `wordwrap` at max_leaf_node_width / max_parent_node_width.
- Golden tests: the PHP binary's plain-text screen (tmux capture, no colours) for each fixture is the expected grid. Go must match byte for byte inside the tree area.

## Invariants (unchanged)
- One map in memory. File on disk is the truth. Every write goes through `Map.Save`; extract and rename are the only functions that write other files.
- Taskwarrior is read-only.
