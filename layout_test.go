package main

import (
	"os"
	"strings"
	"testing"
)

// Goldens are 120x40 tmux captures of ref/hmx.php (test/golden/gen.sh):
// <name>.collapsed.txt as opened (collapse_all + collapse_level 1, centred on root),
// <name>.txt after pressing 0 (expand_all, centred). Compared row by row, trailing blanks ignored.
func TestGolden(t *testing.T) {
	for _, name := range []string{"backend", "infra", "todo", "nixie"} {
		for _, expanded := range []bool{false, true} {
			file := "test/golden/" + name + ".collapsed.txt"
			if expanded {
				file = "test/golden/" + name + ".txt"
			}
			m, _ := Load("test/fixtures/" + name + ".hmm")
			m.CollapseAll()
			m.CollapseLevel(1)
			if expanded {
				m.ExpandAll()
			}
			Build(m, 120, 40)
			m.Center(120, 40)
			m.MoveWindow(120, 40)
			got := rows(m.Screen(120, 40))
			b, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			want := rows(string(b))
			for i := 0; i < len(want) || i < len(got); i++ {
				g, w := "", ""
				if i < len(got) {
					g = got[i]
				}
				if i < len(want) {
					w = want[i]
				}
				if g != w {
					t.Errorf("%s row %d\n got: %q\nwant: %q", file, i+1, g, w)
				}
			}
		}
	}
}

func rows(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		out = append(out, strings.TrimRight(l, " "))
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func TestWordWrapLikePHP(t *testing.T) {
	// php wordwrap($s, 25, "\n"): greedy break at spaces, words longer than the width overflow
	cases := map[string]string{
		"short":                              "short",
		"aaaa bbbb cccc dddd eeee ffff gggg": "aaaa bbbb cccc dddd eeee\nffff gggg",
		"abcdefghijklmnopqrstuvwxyz0123 x":   "abcdefghijklmnopqrstuvwxyz0123\nx",
		"a b":                                "a b",
	}
	for in, want := range cases {
		if got := wordwrap(in, 25); got != want {
			t.Errorf("wordwrap(%q) = %q, want %q", in, got, want)
		}
	}
	if got := wordwrap("aaaa bbbb cccc dddd eeeee ffff", 25); got != "aaaa bbbb cccc dddd eeeee\nffff" {
		t.Errorf("exact fit: %q", got)
	}
}
