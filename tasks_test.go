package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

const export = `[{"uuid":"cccccccc-0000-0000-0000-000000000000","status":"pending","due":"20000101T000000Z"},
{"uuid":"bbbbbbbb-0000-0000-0000-000000000000","status":"completed","due":"20000101T000000Z"}]`

func TestTasksFromExport(t *testing.T) {
	ts := tasksFromExport(export)
	if !ts["cccccccc"].overdue || ts["cccccccc"].done || !ts["bbbbbbbb"].done || ts["bbbbbbbb"].overdue {
		t.Errorf("statuses %+v", ts)
	}
	if len(tasksFromExport("[]")) != 0 || len(tasksFromExport("garbage")) != 0 {
		t.Error("empty/invalid export → no tasks")
	}
}

func TestTaskMarkers(t *testing.T) {
	m := Parse("root\n\ta [[task:bbbbbbbb]]\n\tb [[task:cccccccc]]\n")
	m.Tasks = tasksFromExport(export)
	if got := nodeText(m, &Node{Title: "[[task:dddddddd]]"}); got != "☐ [[task:dddddddd]]" {
		t.Errorf("unknown uuid renders pending: %q", got)
	}
	if got := nodeText(m, &Node{Title: "x", Body: "y"}); got != "x …" {
		t.Errorf("body marker: %q", got)
	}
	Build(m, 120, 40)
	screen := m.Screen(120, 40)
	for _, want := range []string{"a ☑ [[task:bbbbbbbb]]", "b ☐ [[task:cccccccc]]"} {
		if !strings.Contains(screen, want) {
			t.Errorf("missing %q in\n%s", want, screen)
		}
	}
}

// one `task rc.context=none <uuids> export` at map open, through a fake task on PATH; plain when task is missing
func TestLoadTasks(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "task"), []byte("#!/bin/sh\n/bin/echo \"$@\" > "+d+"/args\n/bin/cat "+d+"/export.json\n"), 0755)
	os.WriteFile(filepath.Join(d, "export.json"), []byte(export), 0644)
	os.WriteFile(filepath.Join(d, "m.hmm"), []byte("m\n\ta [[task:bbbbbbbb]]\n\tb [[task:cccccccc]] and [[task:bbbbbbbb]]\n"), 0644)
	s := tcell.NewSimulationScreen("")
	s.Init()
	a := &App{s: s, mapDir: d}

	t.Setenv("PATH", d)
	a.openMap(filepath.Join(d, "m.hmm"))
	args, _ := os.ReadFile(filepath.Join(d, "args"))
	if !strings.HasPrefix(string(args), "rc.context=none ") || !strings.HasSuffix(strings.TrimSpace(string(args)), " export") ||
		strings.Count(string(args), "bbbbbbbb") != 1 || !strings.Contains(string(args), "cccccccc") {
		t.Errorf("task args %q", args)
	}
	if !a.m.Tasks["bbbbbbbb"].done || !a.m.Tasks["cccccccc"].overdue {
		t.Errorf("tasks %+v", a.m.Tasks)
	}

	t.Setenv("PATH", filepath.Join(d, "none"))
	a.openMap(filepath.Join(d, "m.hmm"))
	if len(a.m.Tasks) != 0 || !strings.Contains(a.m.Screen(120, 40), "a ☐ [[task:bbbbbbbb]]") {
		t.Errorf("no task binary: plain ☐, got %+v", a.m.Tasks)
	}
}
