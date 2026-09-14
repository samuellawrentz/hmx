package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

type MapInfo struct {
	Name, File string
	Count      int
	Depth      int
	Mtime      time.Time
}

func listRows(dir, filter string) []MapInfo {
	var rows []MapInfo
	files, _ := filepath.Glob(filepath.Join(dir, "*.hmm"))
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".hmm")
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		rows = append(rows, MapInfo{Name: name, File: f, Count: NodeCount(f), Mtime: info.ModTime()})
	}
	sort.Slice(rows, func(i, j int) bool {
		if (rows[i].Name == "inbox") != (rows[j].Name == "inbox") {
			return rows[i].Name == "inbox"
		}
		return rows[i].Mtime.After(rows[j].Mtime)
	})
	rows = treeOrder(rows)
	if filter == "" {
		return rows
	}
	return filterTree(rows, filter)
}

// treeOrder nests maps by links: A parents B if A's file contains [[B]] or [[B#,
// first such A in flat order wins. Roots first (flat order), then children recursively;
// anything a cycle keeps unplaced is emitted at depth 0 in a final flat-order pass.
func treeOrder(rows []MapInfo) []MapInfo {
	content := make([]string, len(rows))
	for i, r := range rows {
		b, _ := os.ReadFile(r.File)
		content[i] = string(b)
	}
	parent := make([]int, len(rows))
	for i, b := range rows {
		parent[i] = -1
		for j := range rows {
			if i != j && (strings.Contains(content[j], "[["+b.Name+"]]") || strings.Contains(content[j], "[["+b.Name+"#")) {
				parent[i] = j
				break
			}
		}
	}
	placed := make([]bool, len(rows))
	var out []MapInfo
	var walk func(i, depth int)
	walk = func(i, depth int) {
		placed[i] = true
		rows[i].Depth = depth
		out = append(out, rows[i])
		for c := range rows {
			if parent[c] == i && !placed[c] {
				walk(c, depth+1)
			}
		}
	}
	for pass := 0; pass < 2; pass++ { // roots, then whatever a cycle left unplaced
		for i := range rows {
			if !placed[i] && (pass == 1 || parent[i] == -1) {
				walk(i, 0)
			}
		}
	}
	return out
}

// filterTree keeps rows matching filter (case-insensitive substring) plus their ancestors.
func filterTree(rows []MapInfo, filter string) []MapInfo {
	q := strings.ToLower(filter)
	keep := make([]bool, len(rows))
	for i, r := range rows {
		if !strings.Contains(strings.ToLower(r.Name), q) {
			continue
		}
		keep[i] = true
		depth := r.Depth
		for j := i - 1; j >= 0 && depth > 0; j-- {
			if rows[j].Depth < depth {
				keep[j] = true
				depth = rows[j].Depth
			}
		}
	}
	var out []MapInfo
	for i, r := range rows {
		if keep[i] {
			out = append(out, r)
		}
	}
	return out
}

func renameMap(dir, old, new string) (int, bool) {
	newFile := filepath.Join(dir, new+".hmm")
	if _, err := os.Stat(newFile); err == nil {
		return 0, false
	}
	os.Rename(filepath.Join(dir, old+".hmm"), newFile)
	rep := strings.NewReplacer("[["+old+"]]", "[["+new+"]]", "[["+old+"#", "[["+new+"#")
	touched := 0
	files, _ := filepath.Glob(filepath.Join(dir, "*.hmm"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		r := rep.Replace(string(b))
		if err == nil && r != string(b) {
			os.WriteFile(f, []byte(r), 0644)
			touched++
		}
	}
	return touched, true
}

var listKeys = map[string]string{
	"j": "list_down", "k": "list_up", "Up": "list_up", "Down": "list_down",
	"Enter": "list_open", "n": "list_new", "r": "list_rename", "d": "list_delete",
	"/": "list_filter", "q": "list_quit", "Ctrl-C": "list_quit",
}

func ageStr(mt time.Time) string {
	s := int(time.Since(mt).Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	case s < 86400:
		return fmt.Sprintf("%dh", s/3600)
	}
	return fmt.Sprintf("%dd", s/86400)
}

func (a *App) drawList(rows []MapInfo, cursor int, filter string) {
	header := "maps in " + a.mapDir
	if filter != "" {
		header += "  /" + filter
	}
	empty := "no maps"
	if filter != "" {
		empty = "no match for /" + filter
	}
	var lines []string
	for _, r := range rows {
		lines = append(lines, padRight(strings.Repeat("  ", r.Depth)+r.Name, 32)+fmt.Sprintf("%5d  %4s", r.Count, ageStr(r.Mtime)))
	}
	a.drawRows(header, lines, cursor, empty)
}

// drawRows renders a header row, an optional empty-state message, and a cursor-highlighted list of lines.
func (a *App) drawRows(header string, lines []string, cursor int, empty string) {
	a.s.Clear()
	w, h := a.s.Size()
	putStr(a.s, 0, 0, header, tcell.StyleDefault)
	if len(lines) == 0 {
		putStr(a.s, 0, 2, empty, tcell.StyleDefault)
	}
	for i, line := range lines {
		st := tcell.StyleDefault
		if i == cursor {
			st = styleActive
		}
		putStr(a.s, 0, 2+i, line, st)
	}
	putMessage(a.s, w, h, a.msg)
	a.s.Show()
}

func (a *App) listScreen() string {
	filter := ""
	cursor := 0
	for {
		rows := listRows(a.mapDir, filter)
		cursor = max(0, min(cursor, len(rows)-1))
		a.drawList(rows, cursor, filter)
		ev, ok := a.s.PollEvent().(*tcell.EventKey)
		if !ok {
			continue
		}
		a.msg = ""
		switch action := listKeys[keyName(ev)]; action {
		case "list_down":
			cursor = min(cursor+1, len(rows)-1)
		case "list_up":
			cursor = max(cursor-1, 0)
		case "list_quit":
			return ""
		case "list_filter":
			if new, ok := a.readline(filter); ok {
				filter, cursor = new, 0
			}
		case "list_open":
			switch {
			case len(rows) > 0:
				return rows[cursor].File
			case filter != "":
				files, _ := filepath.Glob(filepath.Join(a.mapDir, "*.hmm")) // Glob sorts within a dir
				q := strings.ToLower(filter)
				a.msg = "no match"
				for _, f := range files {
					if b, err := os.ReadFile(f); err == nil && strings.Contains(strings.ToLower(string(b)), q) {
						a.listQuery, a.msg = filter, ""
						return f
					}
				}
			}
		case "list_new":
			name, ok := a.readline("")
			if !ok || name == "" {
				continue
			}
			file := filepath.Join(a.mapDir, name+".hmm")
			if _, err := os.Stat(file); err == nil {
				a.msg = name + ".hmm already exists"
				continue
			}
			os.WriteFile(file, []byte(name+"\n"), 0644)
			return file
		case "list_rename":
			if len(rows) == 0 {
				break
			}
			newName, ok := a.readline(rows[cursor].Name)
			if !ok || newName == "" || newName == rows[cursor].Name {
				break
			}
			a.msg = newName + ".hmm already exists"
			if n, ok := renameMap(a.mapDir, rows[cursor].Name, newName); ok {
				a.msg = fmt.Sprintf("renamed, %d files updated", n)
			}
		case "list_delete":
			if len(rows) == 0 {
				break
			}
			a.msg = "delete " + rows[cursor].Name + ".hmm? [y/N]"
			a.drawList(rows, cursor, filter)
			if ev2, ok := a.s.PollEvent().(*tcell.EventKey); ok && (ev2.Rune() == 'y' || ev2.Rune() == 'Y') {
				os.Remove(rows[cursor].File)
			}
		}
	}
}
