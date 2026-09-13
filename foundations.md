# Foundations

What hmx is, what it is not, and the rules every change is checked against.

## Purpose
A keyboard-only terminal mind mapper for one person, many maps. Fork of h-m-m.
Maps are plain text in one folder, synced with git.

## Non-negotiables
1. **Plain text.** One `.hmm` per map. Tabs for structure. Readable by upstream h-m-m and by `cat`. No index, no ids, no frontmatter.
2. **One map in memory.** Switching saves and reloads. No tabs, no splits.
3. **Links are text.** `[[map]]`, `[[map#node]]`, `[[task:uuid8]]`. Grep is the backlink engine.
4. **Fewest keys.** 24 defaults. A cut key returns only as a user config binding.
5. **No editor inside the editor.** Bodies go to `$EDITOR`.
6. **Sync is git.** hmx never talks to a network.
7. **Shortest diff wins** once the problem is understood. Ask "does this need to exist" before "how".

## Out of scope
Graph model, node ids, backlink panel, drawn node-to-node arrows, merge-back after extract,
recent/jump screen, templates, timestamps, archive, today view, in-TUI multiline editor, sync.

## Stack
PHP fork of h-m-m for v1 (~320 new lines). Go + tcell rewrite only if the PHP runtime dependency hurts in practice.
Not OpenTUI (flexbox model fights absolute-positioned trees, swaps php dep for bun dep).

## Prior art checked (2026-09-13)
h-m-m (base), tmmpr (Rust free-canvas whiteboard, different model), tui-mindmap (render only).
Nothing does list + link + extract in a terminal.
