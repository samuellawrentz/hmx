#!/usr/bin/env bash
# TUI smoke: php hmm inside tmux, send keys, assert on captured text (testing.md §2)
set -u
repo="$(cd "$(dirname "$0")/.." && pwd)"
status=0

keys() { tmux send-keys -t hmmtest "$@"; sleep 0.4; }
screen() { tmux capture-pane -t hmmtest -p; }
has() { local s; s=$(screen); for n in "$@"; do grep -qF -- "$n" <<<"$s" || { echo "  missing: $n"; echo "$s"; return 1; }; done; }
lacks() { screen | grep -qF -- "$1" && { echo "  unexpected: $1"; return 1; }; return 0; }

# run <name> <file-or-empty> <body-fn>: fresh map_dir with fixtures, hmm in tmux, kill after
run()
{
	local d; d=$(mktemp -d); cp "$repo"/test/fixtures/*.hmm "$d/"
	tmux kill-session -t hmmtest 2>/dev/null
	tmux new-session -d -s hmmtest -x 100 -y 30 "${EDITOR:+EDITOR=$(printf '%q' "$EDITOR") }php $repo/hmm --map-dir=$d${4:-} ${2:+$d/$2}"
	sleep 0.6
	if "$3" "$d"; then echo "PASS $1"; else echo "FAIL $1"; status=1; fi
	tmux kill-session -t hmmtest 2>/dev/null
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
	has backend infra || return 1
	keys / ; keys i n f; keys Enter
	lacks backend && has infra || return 1
	keys Enter
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

run open_map        backend.hmm open_map
run follow_and_back backend.hmm follow_and_back
run enter_on_plain  backend.hmm enter_on_plain
run list_screen     backend.hmm list_screen
run extract         backend.hmm extract
EDITOR="sh -c 'printf \"\\nadded line\\n\" >> \"\$0\"'" run body_edit backend.hmm body_edit
run no_map_dir      ''          no_map_dir /new
exit $status
