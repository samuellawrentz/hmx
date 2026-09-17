package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// completeDir writes 3 small maps: projects.hmm has a depth-3 node under a depth-1
// node, a body line, and a blank line; alpha/beta are unrelated filler maps.
func completeDir(t *testing.T) string {
	d := t.TempDir()
	files := map[string]string{
		"projects": "projects\n\tQ3\n\t\tCost review\n\t\t\treduce cx cost\n\t\t\t> a body line, skip me\n\n\tQ4 goals\n",
		"alpha":    "alpha\n\tfirst note\n",
		"beta":     "beta\n\tsecond note\n",
	}
	for i, n := range []string{"projects", "alpha", "beta"} { // projects newest -> sorts first
		f := filepath.Join(d, n+".hmm")
		os.WriteFile(f, []byte(files[n]), 0644)
		mt := time.Now().Add(-time.Duration(100*i) * time.Second)
		os.Chtimes(f, mt, mt)
	}
	return d
}

func findCand(cands []Candidate, title string) (Candidate, bool) {
	for _, c := range cands {
		if c.Title == title {
			return c, true
		}
	}
	return Candidate{}, false
}

func TestLinkCandidates(t *testing.T) {
	d := completeDir(t)
	all := linkCandidates(d)

	if _, ok := findCand(all, "a body line, skip me"); ok {
		t.Error("body line must be excluded")
	}
	for _, c := range all {
		if c.Title == "" {
			t.Error("blank line produced a candidate")
		}
	}

	root, ok := findCand(all, "projects")
	if !ok || root.Node || root.Context() != "map" || root.Insert != "[[projects]]" {
		t.Errorf("map row = %+v", root)
	}

	q3, ok := findCand(all, "Q3")
	if !ok || !q3.Node || q3.Path != "" || q3.Context() != "projects" || q3.Insert != "[[projects#Q3]]" {
		t.Errorf("depth-1 row = %+v", q3)
	}

	leaf, ok := findCand(all, "reduce cx cost")
	if !ok || !leaf.Node || leaf.Path != "Q3 › Cost review" || leaf.Context() != "projects › Q3 › Cost review" || leaf.Insert != "[[projects#reduce cx cost]]" {
		t.Errorf("depth-3 row = %+v", leaf)
	}
}

func TestFuzzyMatch(t *testing.T) {
	if !fuzzyMatch("", "anything") {
		t.Error("empty query matches everything")
	}
	if !fuzzyMatch("rcc", "reduce cx cost") {
		t.Error("subsequence should match")
	}
	if fuzzyMatch("zzz", "reduce cx cost") {
		t.Error("no zzz in reduce cx cost")
	}
}

func TestFilterCandidates(t *testing.T) {
	d := completeDir(t)
	all := linkCandidates(d)

	// every term matches independently, so a context term may come before a title term
	for _, q := range []string{"q3 red", "reduce q3"} {
		got := filterCandidates(all, q)
		if _, ok := findCand(got, "reduce cx cost"); !ok {
			t.Errorf("%q should match reduce cx cost, got %+v", q, got)
		}
	}

	if got := filterCandidates(all, "zzz"); len(got) != 0 {
		t.Errorf("zzz should match nothing, got %+v", got)
	}

	if got := filterCandidates(all, ""); len(got) != len(all) {
		t.Error("empty query keeps everything")
	}
}

// TestFilterCandidatesScope covers "#" as a map-scope separator: left of "#" filters the map,
// right of "#" filters node path+title, and only node rows (never map rows) survive.
func TestFilterCandidatesScope(t *testing.T) {
	d := completeDir(t)
	all := linkCandidates(d)

	scoped := filterCandidates(all, "projects#")
	if len(scoped) != 4 { // Q3, Cost review, reduce cx cost, Q4 goals
		t.Errorf("projects# count = %d, want 4: %+v", len(scoped), scoped)
	}
	for _, c := range scoped {
		if !c.Node {
			t.Errorf("projects# returned a map row: %+v", c)
		}
	}

	if _, ok := findCand(filterCandidates(all, "proj#red"), "reduce cx cost"); !ok {
		t.Error("proj#red should resolve to reduce cx cost")
	}

	if _, ok := findCand(filterCandidates(all, "projects red"), "reduce cx cost"); !ok {
		t.Error("projects red (no #, map name first) should still match reduce cx cost")
	}
}

func injectStr(s tcell.SimulationScreen, str string) {
	for _, r := range str {
		s.InjectKey(tcell.KeyRune, r, 0)
	}
}

// TestLinkComplete drives the "[[ " popup through readline via a simulation screen.
func TestLinkComplete(t *testing.T) {
	d := completeDir(t)
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(40, 10)
	a := &App{s: s, mapDir: d}

	// First Enter accepts the highlighted candidate, second commits the line.
	injectStr(s, "x[[redu") // simscreen's event channel holds 10; keep each batch within that
	s.InjectKey(tcell.KeyEnter, 0, 0)
	s.InjectKey(tcell.KeyEnter, 0, 0)
	if got, ok := a.readline(""); !ok || got != "x[[projects#reduce cx cost]]" {
		t.Errorf("accept top candidate: got %q, %v", got, ok)
	}

	injectStr(s, "[[proj")
	s.InjectKey(tcell.KeyEnter, 0, 0)
	s.InjectKey(tcell.KeyEnter, 0, 0)
	if got, ok := a.readline(""); !ok || got != "[[projects]]" {
		t.Errorf("accept map row: got %q, %v", got, ok)
	}

	// Esc closes the popup but keeps the typed text; editing continues.
	injectStr(s, "[[ab")
	s.InjectKey(tcell.KeyEsc, 0, 0)
	s.InjectKey(tcell.KeyEnter, 0, 0)
	if got, ok := a.readline(""); !ok || got != "[[ab" {
		t.Errorf("esc keeps typed text: got %q, %v", got, ok)
	}

	// "[[proj#red" is longer than the simscreen event buffer (10), so drain concurrently
	// with injection instead of shrinking the string.
	type result struct {
		s  string
		ok bool
	}
	ch := make(chan result, 1)
	go func() {
		got, ok := a.readline("")
		ch <- result{got, ok}
	}()
	injectStr(s, "[[proj#red")
	s.InjectKey(tcell.KeyEnter, 0, 0)
	s.InjectKey(tcell.KeyEnter, 0, 0)
	if r := <-ch; !r.ok || r.s != "[[projects#reduce cx cost]]" {
		t.Errorf("typed # is not doubled: got %q, %v", r.s, r.ok)
	}
}

// TestShowLinePopup checks the popup renders one row per candidate directly above the edit line.
func TestShowLinePopup(t *testing.T) {
	d := completeDir(t)
	s := tcell.NewSimulationScreen("")
	s.Init()
	s.SetSize(40, 10)
	a := &App{s: s, mapDir: d}

	all := linkCandidates(d)
	cands := filterCandidates(all, "")
	a.showLine([]rune("[["), 2, 0, cands, 0)

	lines := strings.Split(screenText(s), "\n")
	_, h := s.Size()
	if !strings.Contains(lines[h-2], cands[0].Title) {
		t.Errorf("row above edit line = %q, want title %q", lines[h-2], cands[0].Title)
	}
	if !strings.HasPrefix(lines[h-1], "[[") {
		t.Errorf("edit line changed: %q", lines[h-1])
	}
}
