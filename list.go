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
	Mtime      time.Time
}

func listRows(dir, filter string) []MapInfo {
	var rows []MapInfo
	files, _ := filepath.Glob(filepath.Join(dir, "*.hmm"))
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".hmm")
		info, err := os.Stat(f)
		if err != nil || (filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter))) {
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
	return rows
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
	a.s.Clear()
	w, h := a.s.Size()
	header := "maps in " + a.mapDir
	if filter != "" {
		header += "  /" + filter
	}
	putStr(a.s, 0, 0, header, tcell.StyleDefault)
	if len(rows) == 0 {
		msg := "no maps"
		if filter != "" {
			msg = "no match for /" + filter
		}
		putStr(a.s, 0, 2, msg, tcell.StyleDefault)
	}
	for i, r := range rows {
		st := tcell.StyleDefault
		if i == cursor {
			st = styleActive
		}
		line := padRight(r.Name, 32) + fmt.Sprintf("%5d  %4s", r.Count, ageStr(r.Mtime))
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
