package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseLink(t *testing.T) {
	for in, want := range map[string][3]any{
		"[[a]]":             {"a", "", true},
		"[[a#b c]]":         {"a", "b c", true},
		"x [[a-b#n]] y":     {"a-b", "n", true},
		"no link":           {"", "", false},
		"[[task:abcd1234]]": {"", "", false},
		"[[a]] and [[b]]":   {"a", "", true},
	} {
		m, n, ok := parseLink(in)
		if m != want[0] || n != want[1] || ok != want[2] {
			t.Errorf("parseLink(%q) = %q %q %v, want %v", in, m, n, ok, want)
		}
	}
}

func testApp(t *testing.T) (*App, string) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "one.hmm"), []byte("one\n\tgo [[two#beta]]\n\tplain\n\tnew [[three]]\n"), 0644)
	os.WriteFile(filepath.Join(d, "two.hmm"), []byte("two\n\talpha\n\tbeta\n"), 0644)
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(100, 30)
	a := &App{s: s, mapDir: d}
	a.openMap(filepath.Join(d, "one.hmm"))
	return a, d
}

func byTitle(m *Map, title string) int {
	for id, n := range m.Nodes {
		if n.Title == title {
			return id
		}
	}
	return -1
}

func TestFollowAndBack(t *testing.T) {
	a, _ := testApp(t)
	a.m.Active = byTitle(a.m, "go [[two#beta]]")
	a.followLink()
	if filepath.Base(a.m.File) != "two.hmm" || a.m.Nodes[a.m.Active].Title != "beta" || len(a.stack) != 1 {
		t.Fatalf("follow: file=%s active=%q stack=%d", a.m.File, a.m.Nodes[a.m.Active].Title, len(a.stack))
	}
	if a.breadcrumb() != "one › two [1]" {
		t.Errorf("breadcrumb %q", a.breadcrumb())
	}
	a.goBack()
	if filepath.Base(a.m.File) != "one.hmm" || a.m.Nodes[a.m.Active].Title != "go [[two#beta]]" || len(a.stack) != 0 || a.breadcrumb() != "" {
		t.Fatalf("back: file=%s active=%q stack=%d", a.m.File, a.m.Nodes[a.m.Active].Title, len(a.stack))
	}
	a.goBack()
	if filepath.Base(a.m.File) != "one.hmm" {
		t.Error("back on empty stack is a no-op")
	}
	a.m.Active = byTitle(a.m, "plain")
	a.followLink()
	if filepath.Base(a.m.File) != "one.hmm" {
		t.Error("no link: nothing happens")
	}
}

func TestOpenLinkedNodeLookup(t *testing.T) {
	a, d := testApp(t)
	a.openLinked(filepath.Join(d, "two.hmm"), "ALPH") // substring, case-insensitive
	if a.m.Nodes[a.m.Active].Title != "alpha" {
		t.Errorf("substring match: %q", a.m.Nodes[a.m.Active].Title)
	}
	a.openLinked(filepath.Join(d, "two.hmm"), "missing")
	if a.m.Active != a.m.Root || a.m.Nodes[a.m.Root].Title != "two" || a.msg != "node not found: missing" {
		t.Errorf("missing target: active=%d msg=%q", a.m.Active, a.msg)
	}
}

func TestFollowCreatesMissingMap(t *testing.T) {
	a, d := testApp(t)
	a.m.Active = byTitle(a.m, "new [[three]]")
	a.s.(tcell.SimulationScreen).InjectKey(tcell.KeyRune, 'n', 0)
	a.followLink()
	if _, err := os.Stat(filepath.Join(d, "three.hmm")); err == nil || filepath.Base(a.m.File) != "one.hmm" {
		t.Fatal("n: not created, stays")
	}
	a.s.(tcell.SimulationScreen).InjectKey(tcell.KeyRune, 'y', 0)
	a.followLink()
	b, err := os.ReadFile(filepath.Join(d, "three.hmm"))
	if err != nil || string(b) != "three\n" || filepath.Base(a.m.File) != "three.hmm" || len(a.stack) != 1 {
		t.Fatalf("y: %v %q file=%s", err, b, a.m.File)
	}
}

func TestEnterKey(t *testing.T) {
	a, _ := testApp(t)
	a.m.Active = byTitle(a.m, "go [[two#beta]]")
	actions["enter_key"](a)
	if filepath.Base(a.m.File) != "two.hmm" {
		t.Error("enter on a link follows it")
	}
}
