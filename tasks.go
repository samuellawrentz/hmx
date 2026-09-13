package main

import (
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"time"
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
