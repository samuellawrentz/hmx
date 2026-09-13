package main

import (
	"math"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func keyName(ev *tcell.EventKey) string {
	if ev.Key() == tcell.KeyRune {
		if ev.Rune() == ' ' {
			return "Space"
		}
		return string(ev.Rune())
	}
	return tcell.KeyNames[ev.Key()]
}

var actions = map[string]func(*App){
	"go_left":      func(a *App) { changeActiveNode(a, -1, 0) },
	"go_down":      func(a *App) { changeActiveNode(a, 0, 1) },
	"go_up":        func(a *App) { changeActiveNode(a, 0, -1) },
	"go_right":     func(a *App) { changeActiveNode(a, 1, 0) },
	"toggle_node":  toggleNode,
	"focus":        focus,
	"expand_all":   expandAllAction,
	"collapse_all": collapseAllAction,
	"save":         save,
	"quit":         quitAction,
	"help":         help,
}

var mapKeys = map[string]string{
	"h":      "go_left",
	"j":      "go_down",
	"k":      "go_up",
	"l":      "go_right",
	"Left":   "go_left",
	"Down":   "go_down",
	"Up":     "go_up",
	"Right":  "go_right",
	"Space":  "toggle_node",
	"f":      "focus",
	"0":      "expand_all",
	"1":      "collapse_all",
	"s":      "save",
	"q":      "quit",
	"Ctrl-C": "quit",
	"?":      "help",
}

// changeActiveNode ports change_active_node, ref/hmx.php 2763-2890.
func changeActiveNode(a *App, dx, dy int) {
	m := a.m
	node := m.Nodes[m.Active]

	if dx > 0 {
		if len(node.Children) == 0 {
			return
		}
		if node.Collapsed {
			toggleNode(a)
			node = m.Nodes[m.Active]
		}
		best, bestDist := 0, math.MaxFloat64
		for _, cid := range node.Children {
			c := m.Nodes[cid]
			d := math.Abs(float64(node.y+node.yo) + float64(node.lh)/2 - float64(c.y+c.yo) - float64(c.lh)/2)
			if d < bestDist {
				bestDist, best = d, cid
			}
		}
		m.Active = best
		return
	}

	if m.Active == 0 {
		return
	}

	if dx < 0 {
		if m.Active == m.Root {
			return
		}
		m.Active = node.Parent
		return
	}

	if dy < 0 {
		siblings := m.Nodes[node.Parent].Children
		for i := len(siblings) - 1; i >= 0; i-- {
			c := m.Nodes[siblings[i]]
			if c.y+c.yo < node.y+node.yo {
				m.Active = siblings[i]
				return
			}
		}
	}

	if dy > 0 {
		for _, cid := range m.Nodes[node.Parent].Children {
			c := m.Nodes[cid]
			if c.y+c.yo > node.y+node.yo {
				m.Active = cid
				return
			}
		}
	}

	best, bestDist, found := 0, 0.0, false
	for id, nd := range m.Nodes {
		if id == m.Active || nd.y == -1 {
			continue
		}
		dyv := float64(nd.y+nd.yo) + float64(nd.lh)/2 - float64(node.y+node.yo) - float64(node.lh)/2
		if (dy > 0 && dyv > 0) || (dy < 0 && dyv < 0) {
			dxv := float64(nd.x) + float64(nd.w)/2 - float64(node.x) - float64(node.w)/2
			dist := dyv*dyv*225 + dxv*dxv
			if !found || dist < bestDist {
				bestDist, best, found = dist, id, true
			}
		}
	}
	if found {
		m.Active = best
	}
}

// toggleNode ports toggle_node, ref/hmx.php 1502.
func toggleNode(a *App) {
	m := a.m
	if isLeaf(m.Nodes[m.Active]) {
		m.Active = m.Nodes[m.Active].Parent
	}
	n := m.Nodes[m.Active]
	n.Collapsed = !n.Collapsed
	w, h := a.s.Size()
	Build(m, w, h)
}

// focus ports focus/focus_vh, ref/hmx.php 3223.
func focus(a *App) {
	m := a.m
	collapseSiblings(m, m.Active)
	expandSiblings(m, m.Active)
	w, h := a.s.Size()
	Build(m, w, h)
	m.Center(w, h)
}

// collapseSiblings ports collapse_siblings, ref/hmx.php 3253.
func collapseSiblings(m *Map, id int) {
	if id <= m.Root {
		return
	}
	parent := m.Nodes[id].Parent
	for _, cid := range m.Nodes[parent].Children {
		if cid != id {
			m.Nodes[cid].Collapsed = true
		}
	}
	collapseSiblings(m, parent)
}

// expandSiblings ports expand_siblings, ref/hmx.php 3268.
func expandSiblings(m *Map, id int) {
	n := m.Nodes[id]
	if isLeaf(n) {
		return
	}
	n.Collapsed = false
	for _, cid := range n.Children {
		expandSiblings(m, cid)
	}
}

func expandAllAction(a *App) {
	a.m.ExpandAll()
	w, h := a.s.Size()
	Build(a.m, w, h)
	a.m.Center(w, h)
}

func collapseAllAction(a *App) {
	a.m.CollapseAll()
	w, h := a.s.Size()
	Build(a.m, w, h)
	a.m.Center(w, h)
}

func save(a *App) {
	if err := a.m.Save(); err != nil {
		a.msg = err.Error()
		return
	}
	a.msg = "saved"
}

// quitAction ports quit, ref/hmx.php 2620.
func quitAction(a *App) {
	if a.m.Modified {
		a.m.Save()
	}
	a.quit = true
}

// help ports help(), ref/hmx.php 2145.
func help(a *App) {
	commands := map[string][]string{}
	for key, action := range mapKeys {
		commands[action] = append(commands[action], key)
	}

	var output []string
	for action, keys := range commands {
		sort.Strings(keys)
		output = append(output, padRight(action+" ", 32)+" "+strings.Join(keys, ", "))
	}
	sort.Strings(output)

	a.s.Clear()
	breakpoint := (len(output) - 1) / 2
	for i := 0; i <= breakpoint; i++ {
		right := ""
		if j := i + breakpoint + 1; j < len(output) {
			right = output[j]
		}
		putStr(a.s, 0, i, " "+padRight(output[i], 56)+right, tcell.StyleDefault)
	}

	putStr(a.s, 0, breakpoint+2, "list screen: j/k move, Enter open, n new, r rename, d delete, / filter, q quit", tcell.StyleDefault)

	w, h := a.s.Size()
	putMessage(a.s, w, h, "Press any key to exit this help screen.")
	a.s.Show()

	for {
		if _, ok := a.s.PollEvent().(*tcell.EventKey); ok {
			break
		}
	}
}

func padRight(s string, n int) string {
	for len([]rune(s)) < n {
		s += " "
	}
	return s
}
