package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func projApp(t *testing.T) (*App, string) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "proj.hmm"), []byte("proj\n\tbig\n\t> some body\n\t\tleaf1\n\t\tleaf2\n\tother\n"), 0644)
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(100, 30)
	a := &App{s: s, mapDir: d}
	a.openMap(filepath.Join(d, "proj.hmm"))
	return a, d
}

func TestSlug(t *testing.T) {
	if slug("Big Thing! (v2) ") != "big-thing-v2" || slug("ünïcode") != "n-code" {
		t.Errorf("slug: %q %q", slug("Big Thing! (v2) "), slug("ünïcode"))
	}
}

func TestExtractSubtree(t *testing.T) {
	a, d := projApp(t)
	want := "proj-big\n> some body\n\tleaf1\n\tleaf2\n"
	if !a.extractSubtree(3, "proj-big") {
		t.Fatal("extract returned false")
	}
	if b, _ := os.ReadFile(filepath.Join(d, "proj-big.hmm")); string(b) != want {
		t.Errorf("extracted file %q", b)
	}
	n := a.m.Nodes[3]
	if n.Title != "[[proj-big]]" || len(n.Children) != 0 || n.Body != "" || !a.m.Modified || a.m.Nodes[4] != nil {
		t.Errorf("node after extract: %+v modified=%v", n, a.m.Modified)
	}
	a.m.Save()
	if b, _ := os.ReadFile(filepath.Join(d, "proj.hmm")); string(b) != "proj\n\t[[proj-big]]\n\tother\n" {
		t.Errorf("parent saved %q", b)
	}
	a.openMap(filepath.Join(d, "proj-big.hmm"))
	if a.m.Nodes[a.m.Root].Title != "proj-big" || byTitle(a.m, "leaf1") < 0 || byTitle(a.m, "leaf2") < 0 {
		t.Error("extracted map reloads")
	}
	a.openMap(filepath.Join(d, "proj.hmm"))
	a.m.Undo() // undo covers the tree change, not the file
	if a.extractSubtree(byTitle(a.m, "other"), "proj-big") || a.msg != "proj-big.hmm exists, pick another name" {
		t.Errorf("existing target must be refused, msg=%q", a.msg)
	}
	if b, _ := os.ReadFile(filepath.Join(d, "proj-big.hmm")); string(b) != want {
		t.Error("refused extract must not touch the file")
	}
}

func TestExtractToMapKey(t *testing.T) {
	a, d := projApp(t)
	a.m.Active = 3
	a.s.(tcell.SimulationScreen).InjectKey(tcell.KeyEnter, 0, 0) // accept the default key
	actions["extract_to_map"](a)
	if _, err := os.Stat(filepath.Join(d, "proj-big.hmm")); err != nil {
		t.Error("default key is <map>-<slug>")
	}
	a.m.Active = a.m.Root
	actions["extract_to_map"](a) // no readline, no-op on root
	if a.m.Nodes[a.m.Root].Title != "proj" {
		t.Error("root untouched")
	}
}

func TestEditBody(t *testing.T) {
	t.Setenv("EDITOR", `sh -c 'printf "\nadded line\n" >> "$0"'`)
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "backend.hmm"), []byte(fixture(t, "backend.hmm")), 0644)
	s := tcell.NewSimulationScreen("")
	s.Init()
	a := &App{s: s, mapDir: d}
	a.openMap(filepath.Join(d, "backend.hmm"))
	a.m.Active = 5 // refresh, no body
	a.editBody()
	if n := a.m.Nodes[5]; n.Title != "refresh" || n.Body != "added line" || !a.m.Modified {
		t.Errorf("refresh: %+v", n)
	}
	a.m.Active = 4 // JWT rotation, existing body
	a.editBody()
	if n := a.m.Nodes[4]; n.Title != "JWT rotation" || n.Body != "Rotate every 24h.\n\nadded line" {
		t.Errorf("JWT: %+v", n)
	}
	t.Setenv("EDITOR", "true")
	a.editBody()
	if n := a.m.Nodes[4]; n.Body != "Rotate every 24h.\n\nadded line" {
		t.Errorf("unchanged file keeps body: %q", n.Body)
	}
}
