#!/usr/bin/env bash
# TUI smoke: $HMX_BIN (default ./hmx; "php ref/hmx.php" = reference) inside tmux, send keys, assert on captured text (testing.md §2), 11 scenarios
set -u
repo="$(cd "$(dirname "$0")/.." && pwd)"
bin="${HMX_BIN:-./hmx}"
sess="hmxtest$$"   # unique per run: a concurrent suite on this machine must not share the session
status=0

keys() { tmux send-keys -t "$sess" "$@"; sleep 0.4; }
screen() { tmux capture-pane -t "$sess" -p; }
has() { local s; s=$(screen); for n in "$@"; do grep -qF -- "$n" <<<"$s" || { echo "  missing: $n"; echo "$s"; return 1; }; done; }
lacks() { screen | grep -qF -- "$1" && { echo "  unexpected: $1"; return 1; }; return 0; }

# run <name> <file-or-empty> <body-fn>: fresh map_dir with fixtures, hmx in tmux, kill after
run()
{
	local d; d=$(mktemp -d); cp "$repo"/test/fixtures/*.hmm "$d/"
	tmux kill-session -t "$sess" 2>/dev/null
	tmux new-session -d -c "$repo" -s "$sess" -x 100 -y 30 "${EDITOR:+EDITOR=$(printf '%q' "$EDITOR") }${FAKE_TASK:+PATH=$FAKE_TASK:\$PATH }$bin --map-dir=$d${4:-} ${2:+$d/$2}"
	for _ in $(seq 50); do screen | grep -q '[^[:space:]]' && break; sleep 0.1; done   # first exec of a fresh build is slow on macOS
	if "$3" "$d"; then echo "PASS $1"; else echo "FAIL $1"; status=1; fi
	tmux kill-session -t "$sess" 2>/dev/null
}

open_map() { has backend auth '[[infra#redis]]'; }

follow_and_back()
{
	keys l; keys j; keys Enter
	has redis postgres 'backend › infra [1]' || return 1
	keys BSpace
	has '[[infra#redis]]' && lacks postgres
}

# insert_new_sibling shows 'NEW' in the tree while the inline editor is open
enter_on_plain()
{
	keys l; keys Enter
	has NEW || return 1
	keys z z z; keys Enter
	has zzz
}

extract()
{
	keys l
	keys C-e
	sleep 0.3
	keys Enter
	has '[[backend-auth]]' && [ -f "$1/backend-auth.hmm" ] && grep -q 'JWT rotation' "$1/backend-auth.hmm"
}

list_screen()
{
	printf 'inbox\n' > "$1/inbox.hmm"
	keys q
	screen | grep -A1 '^$' | grep -q '^inbox' || { echo '  inbox not first'; screen; return 1; }
	has backend '  infra' || return 1        # infra nests under backend, which links it
	keys / ; keys i n f; keys Enter
	has backend '  infra' || return 1        # filter keeps matches plus their ancestors
	keys j; keys Enter
	has redis postgres
}

# E on JWT rotation: '…' marker in tree, body pane shows the existing + appended line
body_edit()
{
	keys l; keys l
	keys E
	sleep 0.8
	has '…' 'Rotate every 24h.' 'added line'
}

# missing map_dir is created and the empty list renders
no_map_dir() { [ -d "$1/new" ] && has 'maps in' 'no maps'; }

# 0 expands every branch, 1 collapses back to the root's children
expand_collapse()
{
	screen | grep -q '\[+\]' || return 1
	keys 0
	lacks '[+]' || return 1
	keys 1
	has '[+]'
}

# D cuts the children of a node to the clipboard, p pastes them under a sibling
move_children()
{
	keys 0                # expand all so titles are visible
	keys /
	keys "API contract"
	keys Enter
	has 'API contract review' 'Schema migration' || return 1   # title truncated at this width
	keys D
	lacks 'Schema migration checklist' || return 1
	keys j                # sibling "Service ownership matrix update"
	keys p
	has 'Schema migration checklist' 'Rate limit tuning notes' || return 1
	screen | grep -F 'API contract review' | grep -vq 'Schema' || return 1
	screen | awk '/Service ownership matrix update/{f=1} f && /Schema migration checklist/{found=1} END{exit !found}'
}

# [[ in the inline editor opens the link popup; Enter accepts the highlighted candidate, a second Enter commits
link_complete()
{
	keys l
	keys o
	tmux send-keys -t "$sess" -l '[['; sleep 0.4   # send-keys treats "[[" specially; -l forces literal
	sleep 0.3
	has infra || return 1        # a candidate row is visible
	keys i n
	sleep 0.3
	keys Enter                   # accept the candidate
	keys Enter                   # commit the node
	has '[['
}

# t opens the task picker on the active node; Enter links the top (most urgent) task
task_picker()
{
	keys l
	keys t
	sleep 0.3
	has 'fix login' || return 1
	keys Enter
	sleep 0.3
	has '[[task:'
}

run open_map        backend.hmm open_map
run follow_and_back backend.hmm follow_and_back
run enter_on_plain  backend.hmm enter_on_plain
run list_screen     backend.hmm list_screen
run extract         backend.hmm extract
EDITOR="sh -c 'printf \"\\nadded line\\n\" >> \"\$0\"'" run body_edit backend.hmm body_edit
run no_map_dir      ''          no_map_dir /new
run expand_collapse backend.hmm expand_collapse
run move_children   deep.hmm    move_children
run link_complete   backend.hmm link_complete

FAKE_TASK=$(mktemp -d)
printf '[{"uuid":"aaaaaaaa-0000-0000-0000-000000000000","description":"write spec","project":"hmx","urgency":3.1},
{"uuid":"bbbbbbbb-0000-0000-0000-000000000000","description":"fix login","project":"web","urgency":9.5}]' > "$FAKE_TASK/pending.json"
printf '#!/bin/sh\ncase "$*" in *export*) /bin/cat %s/pending.json ;; esac\n' "$FAKE_TASK" > "$FAKE_TASK/task"
chmod +x "$FAKE_TASK/task"
run task_picker backend.hmm task_picker
exit $status
