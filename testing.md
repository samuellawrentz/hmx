# Testing

Three levels. Cheap first. Every change must pass level 1; UI changes also level 2.

## 1. Unit (`go test ./...`)
TDD: write the failing test first, then the code, one slice at a time. Fixtures are real `.hmm` files in `test/fixtures/` with real tabs. Never build fixtures from strings with spaces.

**Golden layout tests.** For each fixture, `test/golden/<name>.txt` (expanded) and `<name>.collapsed.txt` are the plain-text 120x40 screens of the PHP v1 (`test/golden/gen.sh`, which needs `git show php-ref:ref/hmx.php > ref/hmx.php`). `layout_test.go` renders the same fixture and diffs byte for byte. Goldens are frozen; never regenerate them from Go.

| area | check |
|---|---|
| parser | example map round-trips byte-identical: load → save. Body `> ` lines attach to the node above, not as children |
| parser | unicode titles keep width; a title starting with `>` but no space is a node, not a body |
| node count | skips body lines |
| links | `[[a]]`, `[[a#b c]]`, `[[task:1a2b3c4d]]` parsed from title first, then body; no link → null |
| extract | key = `<map>-<slug>`; both files reload; parent node title is `[[key]]` with no children; existing target refused |
| rename | `[[old` → `[[new` rewritten only inside `[[..]]`, across every map in a temp dir; count reported |
| keymap | exactly the 23 default map keys and 7 list keys are bound; every bound function exists |
| task status | `task export` JSON fixture → ☐ / ☑ / overdue; missing `task` binary → plain render |

## 2. TUI smoke (`bash test/tui.sh`, tmux)
`HMX_BIN` selects the binary (default `./hmx`). Run it inside a detached tmux pane of fixed size, send keys, capture the screen, assert on text.
```
tmux new-session -d -s hmxtest -x 100 -y 30 "$HMX_BIN /tmp/t/backend.hmm"
tmux send-keys -t hmxtest j j Enter        # move, follow link
sleep 0.3
tmux capture-pane -t hmxtest -p | grep -q "infra"   # breadcrumb / target map rendered
```
Scenarios, one function each, temp `map_dir` per run:
1. open map → root and children rendered, `[[..]]` node present
2. `Enter` on link → target map shown, breadcrumb depth 1. `Backspace` → back, same active node
3. `Enter` on plain node → new sibling `NEW` appears
4. `ctrl+e` on a subtree → new file exists, parent shows `[[key]]`
5. `q` → list screen shows all maps, `inbox` first. `/` filter narrows. `Enter` opens
6. `E` with `EDITOR="sed -i '' '$ a\> added'"` → body line saved, `…` marker and pane shown
7. no `map_dir` → created, empty list renders

Golden text, not golden screenshots. Assert on 1–2 strings per scenario.

## 3. Dogfood
- After phase 4: copy 3 real maps into `~/maps`, use hmm daily for a week. Log friction in `inbox.hmm` under a `hmm-bugs` node.
- After phase 5: extract one real subtree, open both in upstream h-m-m, confirm it still reads them.
- After phase 7: link 2 real tasks, mark one done in `twt`, reopen the map, confirm ☑.

## Gates
`go vet ./...` · `go test ./...` · `bash test/tui.sh` · manual 30-second run on the example map, described in the commit message body.

## Not testing
Rendering geometry (upstream's job), colours, terminal resize, performance.
