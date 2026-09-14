package main

import (
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// {{{ hmx additions
// TaskStatus mirrors tasks_from_export's per-uuid result, ref/hmx.php 3881.
type TaskStatus struct{ done, overdue bool }

var taskRe = regexp.MustCompile(`\[\[task:([0-9a-z]{8})\]\]`)

// tasksFromExport ports tasks_from_export, ref/hmx.php 3881.
func tasksFromExport(js string) map[string]TaskStatus {
	tasks := map[string]TaskStatus{}
	var raw []struct{ UUID, Status, Due string }
	if json.Unmarshal([]byte(js), &raw) != nil {
		return tasks
	}
	for _, t := range raw {
		done := t.Status == "completed"
		overdue := false
		if due, err := time.Parse("20060102T150405Z", t.Due); t.Due != "" && !done && err == nil {
			overdue = due.Before(time.Now())
		}
		tasks[t.UUID[:min(8, len(t.UUID))]] = TaskStatus{done, overdue}
	}
	return tasks
}

// loadTasks ports load_tasks, ref/hmx.php 3896.
func (m *Map) loadTasks() {
	var uuids []string
	for _, id := range slices.Sorted(maps.Keys(m.Nodes)) {
		for _, mm := range taskRe.FindAllStringSubmatch(m.Nodes[id].Title, -1) {
			if !slices.Contains(uuids, mm[1]) {
				uuids = append(uuids, mm[1])
			}
		}
	}
	m.Tasks = nil
	if _, err := exec.LookPath("task"); len(uuids) == 0 || err != nil {
		return
	}
	out, _ := exec.Command("task", append(append([]string{"rc.context=none", "rc.verbose=nothing"}, uuids...), "export")...).Output()
	m.Tasks = tasksFromExport(string(out))
}

// showTask ports show_task, ref/hmx.php 3848.
func (a *App) showTask(uuid string) {
	if _, err := exec.LookPath("task"); err != nil {
		a.msg = "task binary not found"
		return
	}
	cmd := exec.Command("sh", "-c", "task rc.context=none "+shQuote(uuid)+" info | ${PAGER:-less}")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	a.s.Suspend()
	cmd.Run()
	a.s.Resume()
}

// nodeText ports node_text, ref/hmx.php 3868.
func nodeText(m *Map, node *Node) string {
	title := taskRe.ReplaceAllStringFunc(node.Title, func(s string) string {
		if m.Tasks[taskRe.FindStringSubmatch(s)[1]].done {
			return "☑ " + s
		}
		return "☐ " + s
	})
	if node.Body != "" {
		return title + " …"
	}
	return title
}

type TaskRow struct {
	UUID, Description, Project, Due string
	Urgency                         float64
}

// pendingTasks parses `task status:pending export` JSON, sorted by urgency desc.
func pendingTasks(js string) []TaskRow {
	var rows []TaskRow
	if json.Unmarshal([]byte(js), &rows) != nil {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Urgency > rows[j].Urgency })
	return rows
}

// task runs the task binary with args, surfacing failures on a.msg.
func (a *App) task(args ...string) (out string, ok bool) {
	if _, err := exec.LookPath("task"); err != nil {
		a.msg = "task binary not found"
		return "", false
	}
	cmd := exec.Command("task", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	res, err := cmd.Output()
	if err != nil {
		msg := strings.SplitN(stderr.String(), "\n", 2)[0]
		if msg == "" {
			msg = err.Error()
		}
		a.msg = "task: " + msg
		return "", false
	}
	return strings.TrimSpace(string(res)), true
}

// taskPicker is the `t` action: pick a pending Taskwarrior task and link it to the active node.
func taskPicker(a *App) {
	js, ok := a.task("rc.context=none", "rc.verbose=nothing", "status:pending", "export")
	if !ok {
		return
	}
	all := pendingTasks(js)

	filter := ""
	cursor := 0
	for {
		var rows []TaskRow
		q := strings.ToLower(filter)
		for _, r := range all {
			if strings.Contains(strings.ToLower(r.Description), q) || strings.Contains(strings.ToLower(r.Project), q) {
				rows = append(rows, r)
			}
		}
		cursor = max(0, min(cursor, len(rows)-1))

		header := "tasks"
		if filter != "" {
			header += "  /" + filter
		}
		var lines []string
		for _, r := range rows {
			due := ""
			if t, err := time.Parse("20060102T150405Z", r.Due); err == nil {
				due = t.Format("2006-01-02")
			}
			lines = append(lines, padRight(r.Description, 50)+padRight(r.Project, 16)+due)
		}
		a.drawRows(header, lines, cursor, "no pending tasks")

		ev, ok := a.s.PollEvent().(*tcell.EventKey)
		if !ok {
			continue
		}
		a.msg = ""
		switch listKeys[keyName(ev)] {
		case "list_down":
			cursor = min(cursor+1, len(rows)-1)
		case "list_up":
			cursor = max(cursor-1, 0)
		case "list_filter":
			if new, ok := a.readline(filter); ok {
				filter, cursor = new, 0
			}
		case "list_quit":
			return
		case "list_open":
			if len(rows) > 0 {
				a.linkTask(rows[cursor].UUID, rows[cursor].Description)
			}
			return
		case "list_new":
			a.createTask()
			return
		}
	}
}

// linkTask annotates the Taskwarrior task with this node and rewrites the node's title to link it.
func (a *App) linkTask(uuid, desc string) {
	short := uuid[:min(8, len(uuid))]
	link := "[[task:" + short + "]]"
	n := a.m.Nodes[a.m.Active]
	newTitle := desc + " " + link
	if taskRe.MatchString(n.Title) {
		newTitle = taskRe.ReplaceAllString(n.Title, link)
	}
	plain := strings.TrimSpace(taskRe.ReplaceAllString(newTitle, ""))
	mapName := strings.TrimSuffix(filepath.Base(a.m.File), ".hmm")

	if _, ok := a.task("rc.confirmation=off", "rc.verbose=nothing", uuid, "annotate", "--", "map: "+mapName+"#"+plain); !ok {
		return
	}
	n.Title = newTitle
	a.m.PushChange()
	a.m.loadTasks()
	a.build()
	a.msg = "linked " + short
}

// createTask adds a new Taskwarrior task from the active node's title, then links it.
func (a *App) createTask() {
	title := strings.TrimSpace(taskRe.ReplaceAllString(a.m.Nodes[a.m.Active].Title, ""))
	if title == "" {
		a.msg = "empty title"
		return
	}
	if _, ok := a.task("rc.confirmation=off", "rc.verbose=nothing", "add", "--", title); !ok {
		return
	}
	out, ok := a.task("rc.context=none", "+LATEST", "_uuids")
	if !ok || out == "" {
		a.msg = "task: no uuid for new task"
		return
	}
	uuid := strings.Fields(out)[0]
	a.linkTask(uuid, title)
}
