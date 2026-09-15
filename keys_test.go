package main

import (
	"github.com/gdamore/tcell/v2"
	"sort"
	"strings"
	"testing"
)

// spec.md: the 28 map bindings (arrows, Ctrl-C, Backspace2 are aliases), 32 distinct actions
func TestKeymap(t *testing.T) {
	want := strings.Fields("h j k l Up Down Left Right Space f c 0 1 o Tab e E d y Y D p P J K u / n N Enter Backspace Backspace2 Ctrl-E s q Ctrl-C ? t")
	check(t, mapKeys, want, 32)
	for k, name := range mapKeys {
		if actions[name] == nil {
			t.Errorf("%q bound to unknown action %q", k, name)
		}
	}
}

func check(t *testing.T, table map[string]string, want []string, distinct int) {
	var got []string
	funcs := map[string]bool{}
	for k, name := range table {
		got = append(got, k)
		funcs[name] = true
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("keys\n got %v\nwant %v", got, want)
	}
	if len(funcs) != distinct {
		t.Errorf("distinct functions = %d, want %d", len(funcs), distinct)
	}
}

func TestKeyName(t *testing.T) {
	// config `bind<name> = fn` and the tables use the same names
	k := func(key tcell.Key, r rune) string { return keyName(tcell.NewEventKey(key, r, 0)) }
	if k(tcell.KeyRune, 'x') != "x" || k(tcell.KeyRune, ' ') != "Space" || k(tcell.KeyCtrlE, 0) != "Ctrl-E" || k(tcell.KeyEnter, 0) != "Enter" || k(tcell.KeyTab, 0) != "Tab" {
		t.Error("keyName")
	}
}

// inline editor (port of magic_readline) driven through a tcell simulation screen
func TestReadline(t *testing.T) {
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(40, 10)
	a := &App{s: s, m: Parse("x\n")}
	type step struct {
		keys []*tcell.EventKey
		want string
		ok   bool
	}
	r := func(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, 0) }
	k := func(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, 0) }
	for _, st := range []step{
		{[]*tcell.EventKey{r('z'), r('z'), r('z'), k(tcell.KeyEnter)}, "zzz", true},
		{[]*tcell.EventKey{r('b'), k(tcell.KeyLeft), r('a'), k(tcell.KeyEnter)}, "ab", true},
		{[]*tcell.EventKey{r('a'), r('b'), k(tcell.KeyBackspace2), k(tcell.KeyEnter)}, "a", true},
		{[]*tcell.EventKey{r('a'), r(' '), r('b'), k(tcell.KeyCtrlW), r('c'), k(tcell.KeyEnter)}, "a c", true},
		{[]*tcell.EventKey{r('a'), k(tcell.KeyEscape)}, "", false},
		{[]*tcell.EventKey{r('ü'), r(' '), k(tcell.KeyEnter)}, "ü", true},
	} {
		for _, ev := range st.keys {
			s.InjectKey(ev.Key(), ev.Rune(), ev.Modifiers())
		}
		got, ok := a.readline("")
		if got != st.want || ok != st.ok {
			t.Errorf("readline = %q,%v want %q,%v", got, ok, st.want, st.ok)
		}
	}
	s.InjectKey(tcell.KeyEnter, 0, 0)
	if got, _ := a.readline("old"); got != "old" {
		t.Errorf("initial title kept: %q", got)
	}
}
