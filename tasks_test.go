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

const pendingJSON = `[
{"uuid":"aaaaaaaa-0000-0000-0000-000000000000","description":"write spec","project":"hmx","urgency":3.1,"due":"20300101T000000Z"},
{"uuid":"bbbbbbbb-0000-0000-0000-000000000000","description":"fix login","project":"web","urgency":9.5},
{"uuid":"cccccccc-0000-0000-0000-000000000000","description":"buy milk","project":"","urgency":1.0,"due":"20250601T120000Z"}
]`

// fakeTask writes a `task` script to d that logs its args and answers export/add/_uuids; mode "fail" makes it exit 1.
func fakeTask(t *testing.T, d string, mode string) {
	t.Helper()
	body := "#!/bin/sh\n/bin/echo \"$@\" >> " + d + "/args\n"
	if mode == "fail" {
		body += "echo boom 1>&2\nexit 1\n"
	} else {
		body += `case "$*" in
*export*) /bin/cat ` + d + `/pending.json ;;
*" add "*) echo "Created task 7." ;;
*_uuids*) echo "dddddddd-1111-2222-3333-444444444444" ;;
esac
`
	}
	os.WriteFile(filepath.Join(d, "task"), []byte(body), 0755)
	os.WriteFile(filepath.Join(d, "pending.json"), []byte(pendingJSON), 0644)
}

func TestPendingTasks(t *testing.T) {
	rows := pendingTasks(pendingJSON)
	if len(rows) != 3 || rows[0].Description != "fix login" || rows[1].Description != "write spec" || rows[2].Description != "buy milk" {
		t.Errorf("order %+v", rows)
	}
	if len(pendingTasks("garbage")) != 0 {
		t.Error("invalid export → no rows")
	}
}

func TestTaskPicker(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "m.hmm"), []byte("m\n\talpha\n\tbeta [[task:aaaaaaaa]]\n"), 0644)
	fakeTask(t, d, "")

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

	t.Setenv("PATH", d)
	a.openMap(filepath.Join(d, "m.hmm"))

	// 1: browse, sorted by urgency desc, no changes on quit
	a.m.Active = 3 // alpha (node 2 is the root "m")
	inject("q")
	taskPicker(a)
	txt := screenText(s)
	for _, want := range []string{"fix login", "write spec", "buy milk", "web", "2030-01-01"} {
		if !strings.Contains(txt, want) {
			t.Errorf("missing %q in\n%s", want, txt)
		}
	}
	if strings.Index(txt, "fix login") > strings.Index(txt, "write spec") || strings.Index(txt, "write spec") > strings.Index(txt, "buy milk") {
		t.Errorf("order wrong:\n%s", txt)
	}
	if a.m.Nodes[3].Title != "alpha" {
		t.Errorf("title changed: %q", a.m.Nodes[3].Title)
	}

	// 2: filter + link a plain node
	inject("/spec", tcell.KeyEnter, tcell.KeyEnter)
	taskPicker(a)
	if a.m.Nodes[3].Title != "write spec [[task:aaaaaaaa]]" {
		t.Errorf("linked title: %q", a.m.Nodes[3].Title)
	}
	if !a.m.Modified {
		t.Error("not marked modified")
	}
	args, _ := os.ReadFile(filepath.Join(d, "args"))
	if !strings.Contains(string(args), "rc.confirmation=off rc.verbose=nothing aaaaaaaa-0000-0000-0000-000000000000 annotate -- map: m#write spec") {
		t.Errorf("annotate args: %q", args)
	}
	if _, ok := a.m.Tasks["aaaaaaaa"]; !ok {
		t.Error("export not re-run after link")
	}

	// 3: replace an existing link
	a.m.Active = 4 // beta, already linked to aaaaaaaa
	inject("jj", tcell.KeyEnter)
	taskPicker(a)
	if a.m.Nodes[4].Title != "beta [[task:cccccccc]]" {
		t.Errorf("relinked title: %q", a.m.Nodes[4].Title)
	}
	args, _ = os.ReadFile(filepath.Join(d, "args"))
	if !strings.Contains(string(args), "cccccccc-0000-0000-0000-000000000000 annotate -- map: m#beta") {
		t.Errorf("relink annotate args: %q", args)
	}

	// 4: create new task from title
	a.m.Active = 3
	a.m.Nodes[3].Title = "write spec [[task:aaaaaaaa]]"
	inject("n")
	taskPicker(a)
	args, _ = os.ReadFile(filepath.Join(d, "args"))
	if !strings.Contains(string(args), "rc.confirmation=off rc.verbose=nothing add -- write spec") ||
		!strings.Contains(string(args), "rc.context=none +LATEST _uuids") ||
		!strings.Contains(string(args), "dddddddd-1111-2222-3333-444444444444 annotate -- map: m#write spec") {
		t.Errorf("create args: %q", args)
	}
	if a.m.Nodes[3].Title != "write spec [[task:dddddddd]]" {
		t.Errorf("created+linked title: %q", a.m.Nodes[3].Title)
	}

	// 5: missing binary
	title := a.m.Nodes[3].Title
	t.Setenv("PATH", filepath.Join(d, "none"))
	taskPicker(a)
	if a.msg != "task binary not found" {
		t.Errorf("msg = %q", a.msg)
	}
	if a.m.Nodes[3].Title != title {
		t.Error("title changed on missing binary")
	}

	// 6: non-zero exit
	d2 := t.TempDir()
	fakeTask(t, d2, "fail")
	t.Setenv("PATH", d2)
	taskPicker(a)
	if !strings.HasPrefix(a.msg, "task: ") {
		t.Errorf("msg = %q", a.msg)
	}
	if a.m.Nodes[3].Title != title {
		t.Error("title changed on task failure")
	}
}
