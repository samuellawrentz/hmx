package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func listDir(t *testing.T) string {
	d := t.TempDir()
	files := map[string]string{"inbox": "inbox\n\tx\n", "a": "a\n\tsee [[b]]\n\tand [[b#x]]\n", "b": "b\n\tx\n", "c": "c\n\t> body only\n\tnope [[bb]]\n"}
	for i, n := range []string{"inbox", "a", "b", "c"} {
		f := filepath.Join(d, n+".hmm")
		os.WriteFile(f, []byte(files[n]), 0644)
		mt := time.Now().Add(-time.Duration(400-100*i) * time.Second)
		os.Chtimes(f, mt, mt)
	}
	return d
}

func names(rows []MapInfo) string {
	var s []string
	for _, r := range rows {
		s = append(s, strings.Repeat(" ", r.Depth)+r.Name)
	}
	return strings.Join(s, ",")
}

func TestListRows(t *testing.T) {
	d := listDir(t)
	rows := listRows(d, "") // a links [[b]], so b nests under a
	if names(rows) != "inbox,c,a, b" {
		t.Errorf("order %s", names(rows))
	}
	if names(listRows(d, "b")) != "inbox,a, b" { // matches plus their ancestors
		t.Errorf("filter %s", names(listRows(d, "b")))
	}
	if rows[1].Count != 2 {
		t.Errorf("c count %d", rows[1].Count)
	}
	if len(listRows(filepath.Join(d, "none"), "")) != 0 {
		t.Error("missing dir → no rows")
	}
}

// nesting is derived from links: a map sits under the first map (row order) whose file links to it,
// roots are maps nothing links to, a cycle that no root reaches is appended at root level
func TestListTree(t *testing.T) {
	d := t.TempDir()
	files := map[string]string{
		"root": "root\n\tsee [[kid1]]\n\t> and [[kid2#x]]\n",
		"kid1": "kid1\n\tback to [[root]]\n", // link back to root does not re-parent root
		"kid2": "kid2\n",
		"cyc1": "cyc1\n\t[[cyc2]]\n",
		"cyc2": "cyc2\n\t[[cyc1]]\n",
	}
	for i, n := range []string{"root", "kid1", "kid2", "cyc1", "cyc2"} { // mtime desc = this order
		f := filepath.Join(d, n+".hmm")
		os.WriteFile(f, []byte(files[n]), 0644)
		mt := time.Now().Add(-time.Duration(100*i) * time.Second)
		os.Chtimes(f, mt, mt)
	}
	if got := names(listRows(d, "")); got != "root, kid1, kid2,cyc1, cyc2" {
		t.Errorf("tree %s", got)
	}
	if got := names(listRows(d, "kid2")); got != "root, kid2" {
		t.Errorf("filter keeps ancestors %s", got)
	}
}

func TestRenameMap(t *testing.T) {
	d := listDir(t)
	if n, ok := renameMap(d, "b", "z"); !ok || n != 1 {
		t.Errorf("rename: %d %v", n, ok)
	}
	a, _ := os.ReadFile(filepath.Join(d, "a.hmm"))
	c, _ := os.ReadFile(filepath.Join(d, "c.hmm"))
	if _, err := os.Stat(filepath.Join(d, "z.hmm")); err != nil {
		t.Error("z.hmm missing")
	}
	if _, err := os.Stat(filepath.Join(d, "b.hmm")); err == nil {
		t.Error("b.hmm still there")
	}
	if string(a) != "a\n\tsee [[z]]\n\tand [[z#x]]\n" || string(c) != "c\n\t> body only\n\tnope [[bb]]\n" {
		t.Errorf("links: %q %q", a, c)
	}
	if _, ok := renameMap(d, "z", "a"); ok {
		t.Error("existing target must be refused")
	}
}

func TestListKeymap(t *testing.T) {
	check(t, listKeys, strings.Fields("j k Up Down Enter n r d / q Ctrl-C"), 8)
}

// list screen driven through a simulation screen: returns the file to open, "" on quit
func TestListScreen(t *testing.T) {
	d := listDir(t)
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(100, 30)
	a := &App{s: s, mapDir: d}
	inject := func(keys string, special ...tcell.Key) {
		for _, r := range keys {
			s.InjectKey(tcell.KeyRune, r, 0)
		}
		for _, k := range special {
			s.InjectKey(k, 0, 0)
		}
	}
	inject("jjj", tcell.KeyEnter) // rows: inbox, c, a, b
	if f := a.listScreen(); filepath.Base(f) != "b.hmm" {
		t.Errorf("jjj Enter → %s", f)
	}
	inject("/b", tcell.KeyEnter) // rows: inbox, a, b
	inject("jj", tcell.KeyEnter)
	if f := a.listScreen(); filepath.Base(f) != "b.hmm" {
		t.Errorf("filter then open → %s", f)
	}
	inject("/nope", tcell.KeyEnter)
	inject("", tcell.KeyEnter)
	if f := a.listScreen(); filepath.Base(f) != "c.hmm" || a.listQuery != "nope" {
		t.Errorf("content grep fallback → %s query=%q", f, a.listQuery)
	}
	inject("nzz", tcell.KeyEnter)
	if f := a.listScreen(); filepath.Base(f) != "zz.hmm" {
		t.Errorf("new map → %s", f)
	} else if b, _ := os.ReadFile(f); string(b) != "zz\n" {
		t.Errorf("new map content %q", b)
	}
	inject("jdyq")
	if f := a.listScreen(); f != "" {
		t.Errorf("q → %q", f)
	}
	if _, err := os.Stat(filepath.Join(d, "zz.hmm")); err == nil {
		t.Error("d y deletes the row under the cursor (zz is newest after inbox)")
	}
	inject("q")
	a.listScreen()
	if !strings.Contains(screenText(s), "maps in "+d) {
		t.Errorf("header missing:\n%s", screenText(s))
	}
}

func screenText(s tcell.SimulationScreen) string {
	cells, w, h := s.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.WriteString(string(cells[y*w+x].Runes))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
