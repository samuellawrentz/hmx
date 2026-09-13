#!/usr/bin/env bash
# Regenerate goldens from the PHP reference only (testing.md). 120x40 tmux pane (own socket,
# status bar off so the pane really is 40 rows), one per fixture:
#   <name>.txt           after pressing 0 (expand all)
#   <name>.collapsed.txt as opened
set -eu
repo="$(cd "$(dirname "$0")/../.." && pwd)"
d=$(mktemp -d); cp "$repo"/test/fixtures/*.hmm "$d/"
printf 'set -g status off\n' > "$d/tmux.conf"
tmux() { command tmux -L hmxgold -f "$d/tmux.conf" "$@"; }
for f in "$repo"/test/fixtures/*.hmm; do
	n=$(basename "$f" .hmm)
	for mode in collapsed expanded; do
		tmux kill-session -t hmxgold 2>/dev/null || true
		tmux new-session -d -c "$repo" -s hmxgold -x 120 -y 40 "php ref/hmx.php --map-dir=$d $d/$n.hmm"
		sleep 0.8
		[ $mode = expanded ] && { tmux send-keys -t hmxgold 0; sleep 0.5; }
		out="$repo/test/golden/$n.txt"; [ $mode = collapsed ] && out="$repo/test/golden/$n.collapsed.txt"
		tmux capture-pane -t hmxgold -p > "$out"
	done
done
tmux kill-session -t hmxgold 2>/dev/null || true
rm -rf "$d"
ls -1 "$repo"/test/golden
