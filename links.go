package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// NavEntry is one entry of the back-navigation stack pushed by openLinked.
type NavEntry struct {
	file      string
	active    int
	top, left int
}

var linkRe = regexp.MustCompile(`\[\[([^\]]*)\]\]`)

// parseLink ports parse_link, ref/hmx.php 3911.
func parseLink(text string) (mapName, node string, ok bool) {
	m := linkRe.FindStringSubmatch(text)
	if m == nil || strings.HasPrefix(m[1], "task:") {
		return "", "", false
	}
	parts := strings.SplitN(m[1], "#", 2)
	if len(parts) > 1 {
		node = parts[1]
	}
	return parts[0], node, true
}

// findByTitle returns the lowest id (id != 0) whose title matches pred, or 0.
func (m *Map) findByTitle(pred func(string) bool) int {
	ids := make([]int, 0, len(m.Nodes))
	for id := range m.Nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if id != 0 && pred(m.Nodes[id].Title) {
			return id
		}
	}
	return 0
}

// goToNode ports go_to_node, ref/hmx.php 3945.
func (a *App) goToNode(id int) {
	a.m.Active = id
	for p := a.m.Nodes[id].Parent; p > 0; p = a.m.Nodes[p].Parent {
		a.m.Nodes[p].Collapsed = false
	}
}

// openLinked ports open_map, ref/hmx.php 3953.
func (a *App) openLinked(file, node string) {
	a.stack = append(a.stack, NavEntry{a.m.File, a.m.Active, a.m.Top, a.m.Left})
	if a.m.Modified {
		a.m.Save()
	}
	a.openMap(file)

	if node != "" {
		id := a.m.findByTitle(func(t string) bool { return t == node })
		if id == 0 {
			nl := strings.ToLower(node)
			id = a.m.findByTitle(func(t string) bool { return strings.Contains(strings.ToLower(t), nl) })
		}
		if id == 0 {
			a.msg = "node not found: " + node
		} else {
			a.goToNode(id)
		}
	}

	a.build()
	a.m.Center(a.s.Size())
}

// followLink ports follow_link, ref/hmx.php 3992.
func (a *App) followLink() {
	n := a.m.Nodes[a.m.Active]
	mapName, node, ok := parseLink(n.Title)
	if !ok {
		mapName, node, ok = parseLink(n.Body)
	}
	if !ok {
		return
	}

	file := filepath.Join(a.mapDir, mapName+".hmm")

	if _, err := os.Stat(file); err != nil {
		a.msg = "create " + mapName + ".hmm? [y/N]"
		a.draw()

		ev, ok := a.s.PollEvent().(*tcell.EventKey)
		if !ok || (ev.Rune() != 'y' && ev.Rune() != 'Y') {
			return
		}

		os.MkdirAll(a.mapDir, 0755)
		os.WriteFile(file, []byte(mapName+"\n"), 0644)
	}

	a.openLinked(file, node)
}

// goBack ports go_back, ref/hmx.php 4030.
func (a *App) goBack() {
	if len(a.stack) == 0 {
		return
	}
	prev := a.stack[len(a.stack)-1]
	a.stack = a.stack[:len(a.stack)-1]

	if a.m.Modified {
		a.m.Save()
	}
	a.openMap(prev.file)

	if _, ok := a.m.Nodes[prev.active]; ok {
		a.goToNode(prev.active)
		a.m.Top, a.m.Left = prev.top, prev.left
		a.build()
	}
}

// breadcrumb ports the breadcrumb line in display(), ref/hmx.php 3640-3650.
func (a *App) breadcrumb() string {
	if len(a.stack) == 0 {
		return ""
	}
	names := make([]string, 0, len(a.stack)+1)
	for _, e := range a.stack {
		names = append(names, strings.TrimSuffix(filepath.Base(e.file), ".hmm"))
	}
	names = append(names, strings.TrimSuffix(filepath.Base(a.m.File), ".hmm"))
	return strings.Join(names, " › ") + " [" + strconv.Itoa(len(a.stack)) + "]"
}
