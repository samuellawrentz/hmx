package main

import (
	"os"
	"path/filepath"
	"strings"
)

type Node struct {
	Title     string
	Body      string
	Parent    int
	Children  []int
	Collapsed bool

	x, y, w, wl, lh, clh, h, yo, xo int
}

type snapshot struct {
	nodes  map[int]*Node
	active int
}

type Map struct {
	Nodes    map[int]*Node
	Root     int
	Active   int
	File     string
	Modified bool
	undo     []snapshot

	Top, Left                              int
	rows                                   [][]rune
	mapWidth, mapHeight, mapTop, mapBottom int
}

// {{{ list <-> map conversion, ported from list_to_map/map_to_list in ref/hmx.php

func cleanLine(l string) (string, int) {
	l = strings.Map(func(r rune) rune {
		if r <= 0x08 || (r >= 0x0B && r <= 0x1F) || r == 0x7F || r == 0xFEFF {
			return -1
		}
		return r
	}, strings.ReplaceAll(l, "\t", "  "))
	return l, len(l) - len(strings.TrimLeft(l, " "))
}

func listToMap(lines []string, rootID, startID int) map[int]*Node {
	cleaned := make([]string, len(lines))
	indents := make([]int, len(lines))
	minIndent := -1
	for i, raw := range lines {
		l, ind := cleanLine(raw)
		cleaned[i] = l
		indents[i] = ind
		if strings.TrimSpace(l) != "" && (minIndent == -1 || ind < minIndent) {
			minIndent = ind
		}
	}
	if minIndent == -1 {
		minIndent = 0
	}

	nodes := map[int]*Node{}
	id := startID
	prevLevel, level, prevIndent := 1, 1, 0
	levelParent := map[int]int{1: rootID}
	levelIndent := map[int]int{1: 0}

	for i, line := range cleaned {
		if strings.TrimSpace(line) == "" {
			continue
		}
		stripped := strings.TrimLeft(line, " ")
		sigil := ""
		if len(stripped) >= 2 {
			sigil = stripped[:2]
		}
		if sigil == "> " || stripped == ">" {
			if id > startID {
				text := stripped[1:]
				if sigil == "> " {
					text = stripped[2:]
				}
				if n, ok := nodes[id-1]; ok {
					if n.Body != "" {
						n.Body += "\n" + text
					} else {
						n.Body = text
					}
				}
			}
			continue
		}

		indent := indents[i] - minIndent
		if indent > prevIndent {
			level = prevLevel + 1
			levelIndent[level] = indent
		}
		if indent < prevIndent {
			for pl, pind := range levelIndent {
				if pind == indent {
					level = pl
				}
			}
		}
		if level > prevLevel {
			levelParent[level] = id - 1
		}

		nodes[id] = &Node{Title: strings.TrimSpace(line), Parent: levelParent[level]}

		prevIndent = indent
		prevLevel = level
		id++
	}

	for i := startID; i < id; i++ {
		n := nodes[i]
		if p, ok := nodes[n.Parent]; ok {
			p.Children = append(p.Children, i)
		}
	}
	return nodes
}

func mapToList(nodes map[int]*Node, id int, excludeParent bool, base int) string {
	var out strings.Builder
	if !excludeParent {
		out.WriteString(strings.Repeat("\t", base))
		out.WriteString(nodes[id].Title)
		out.WriteString("\n")
		if nodes[id].Body != "" {
			for _, line := range strings.Split(nodes[id].Body, "\n") {
				out.WriteString(strings.Repeat("\t", base))
				out.WriteString("> ")
				out.WriteString(line)
				out.WriteString("\n")
			}
		}
	}
	childBase := base + 1
	if excludeParent {
		childBase = base
	}
	for _, cid := range nodes[id].Children {
		out.WriteString(mapToList(nodes, cid, false, childBase))
	}
	return out.String()
}

// }}}
// {{{ load / save

func NodeCount(file string) int {
	b, _ := os.ReadFile(file)
	count := 0
	for _, line := range strings.Split(string(b), "\n") {
		if t := strings.TrimLeft(line, " \t"); t != "" && t[0] != '>' {
			count++
		}
	}
	return count
}

func loadEmptyMap(filename string) *Map {
	title := "root"
	if filename != "" {
		title = strings.TrimSuffix(filepath.Base(filename), ".hmm")
	}
	return &Map{
		Nodes: map[int]*Node{
			0: {Title: "X", Parent: -1, Children: []int{1}},
			1: {Title: title, Parent: 0},
		},
		Root:   1,
		Active: 1,
	}
}

func parseNodes(text, filename string) *Map {
	newNodes := listToMap(strings.Split(text, "\n"), 0, 2)
	if len(newNodes) == 0 {
		return loadEmptyMap(filename)
	}

	var firstLevel []int
	for id := 2; ; id++ {
		n, ok := newNodes[id]
		if !ok {
			break
		}
		if n.Parent == 0 {
			firstLevel = append(firstLevel, id)
		}
	}

	m := &Map{Nodes: map[int]*Node{0: {Title: "X", Parent: -1}}}
	if len(firstLevel) > 1 {
		m.Root = 1
		m.Nodes[1] = &Node{Title: "root", Parent: 0, Children: firstLevel}
		m.Nodes[0].Children = []int{1}
		for _, id := range firstLevel {
			newNodes[id].Parent = 1
		}
	} else {
		m.Root = 2
		m.Nodes[0].Children = firstLevel
	}
	for id, n := range newNodes {
		m.Nodes[id] = n
	}
	if _, ok := m.Nodes[1]; ok {
		m.Active = 1
	} else {
		m.Active = 2
	}
	return m
}

func Parse(text string) *Map { return parseNodes(text, "") }

func Load(file string) (*Map, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		m := loadEmptyMap(file)
		m.File = file
		return m, nil
	}
	m := parseNodes(string(b), file)
	m.File = file
	return m, nil
}

func (m *Map) Serialize() string {
	return mapToList(m.Nodes, m.Root, false, 0)
}

func (m *Map) Save() error {
	if err := os.WriteFile(m.File, []byte(m.Serialize()), 0644); err != nil {
		return err
	}
	m.Modified = false
	return nil
}

// }}}
// {{{ editing

func (m *Map) maxID() int {
	max := 0
	for id := range m.Nodes {
		if id > max {
			max = id
		}
	}
	return max
}

func (m *Map) insertNode(asSibling bool) int {
	if m.Active == m.Root {
		asSibling = false
	}
	parentID := m.Active
	if asSibling {
		parentID = m.Nodes[m.Active].Parent
	}

	m.PushChange()

	m.Nodes[parentID].Collapsed = false
	newID := m.maxID() + 1
	m.Nodes[newID] = &Node{Title: "NEW", Parent: parentID}

	if asSibling {
		children := make([]int, 0, len(m.Nodes[parentID].Children)+1)
		for _, c := range m.Nodes[parentID].Children {
			children = append(children, c)
			if c == m.Active {
				children = append(children, newID)
			}
		}
		m.Nodes[parentID].Children = children
	} else {
		m.Nodes[parentID].Children = append(m.Nodes[parentID].Children, newID)
	}

	m.Active = newID
	return newID
}

func (m *Map) InsertSibling() int { return m.insertNode(true) }
func (m *Map) InsertChild() int   { return m.insertNode(false) }

func (m *Map) deleteChildren(id int) {
	for _, cid := range m.Nodes[id].Children {
		m.deleteChildren(cid)
		delete(m.Nodes, cid)
	}
}

func (m *Map) deleteInternal(active int, excludeParent bool) {
	m.deleteChildren(active)

	if excludeParent {
		m.Nodes[active].Children = nil
		return
	}

	parentID := m.Nodes[active].Parent
	siblings := m.Nodes[parentID].Children

	prevSib := 0
	passed := false
	for _, cid := range siblings {
		if cid == active {
			if prevSib != 0 {
				break
			}
			passed = true
		} else {
			prevSib = cid
			if passed {
				break
			}
		}
	}

	children := make([]int, 0, len(siblings))
	for _, cid := range siblings {
		if cid != active {
			children = append(children, cid)
		}
	}
	m.Nodes[parentID].Children = children
	delete(m.Nodes, active)

	if len(children) == 0 {
		m.Active = parentID
	} else {
		m.Active = prevSib
	}
}

func (m *Map) Delete() string {
	isRoot := m.Active == m.Root
	clip := mapToList(m.Nodes, m.Active, isRoot, 0)
	m.PushChange()
	m.deleteInternal(m.Active, isRoot)
	return clip
}

func (m *Map) Yank() string {
	return mapToList(m.Nodes, m.Active, false, 0)
}

func (m *Map) Paste(text string, asSibling bool) {
	if asSibling && m.Active == m.Root {
		return
	}

	parentID := m.Active
	if asSibling {
		parentID = m.Nodes[m.Active].Parent
	}

	newID := m.maxID() + 1
	st := listToMap(strings.Split(text, "\n"), parentID, newID)
	if len(st) == 0 {
		return
	}

	m.PushChange()

	m.Nodes[parentID].Collapsed = false

	var subRoots []int
	for id := newID; ; id++ {
		n, ok := st[id]
		if !ok {
			break
		}
		m.Nodes[id] = n
		if n.Parent == parentID {
			subRoots = append(subRoots, id)
		}
	}

	if asSibling {
		children := make([]int, 0, len(m.Nodes[parentID].Children)+len(subRoots))
		for _, cid := range m.Nodes[parentID].Children {
			children = append(children, cid)
			if cid == m.Active {
				children = append(children, subRoots...)
			}
		}
		m.Nodes[parentID].Children = children
	} else {
		m.Nodes[parentID].Children = append(m.Nodes[parentID].Children, subRoots...)
	}

	m.Active = newID
}

func (m *Map) move(dir int) {
	if m.Active == 0 {
		return
	}
	m.PushChange()

	parentID := m.Nodes[m.Active].Parent
	children := m.Nodes[parentID].Children
	for i, c := range children {
		if c == m.Active {
			j := i + dir
			if j >= 0 && j < len(children) {
				children[i], children[j] = children[j], children[i]
			}
			break
		}
	}
}

func (m *Map) MoveDown() { m.move(1) }
func (m *Map) MoveUp()   { m.move(-1) }

// }}}
// {{{ undo

func deepCopyNodes(nodes map[int]*Node) map[int]*Node {
	out := make(map[int]*Node, len(nodes))
	for id, n := range nodes {
		cp := *n
		cp.Children = append([]int(nil), n.Children...)
		out[id] = &cp
	}
	return out
}

func (m *Map) PushChange() {
	m.undo = append(m.undo, snapshot{nodes: deepCopyNodes(m.Nodes), active: m.Active})
	if len(m.undo) > 24 {
		m.undo = m.undo[1:]
	}
	m.Modified = true
}

func (m *Map) Undo() {
	if len(m.undo) == 0 {
		return
	}
	last := m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	m.Nodes = last.nodes
	m.Active = last.active
}

// }}}
// {{{ tree state, ported from expand_all 3241, collapse_all 3322, collapse 3352, collapse_level 3376

func isLeaf(n *Node) bool { return len(n.Children) == 0 }

func (m *Map) ExpandAll() {
	for _, n := range m.Nodes {
		n.Collapsed = false
	}
}

func (m *Map) CollapseAll() {
	for id, n := range m.Nodes {
		if !isLeaf(n) && id != 0 && id != m.Root {
			n.Collapsed = true
		}
	}
	m.Active = m.Root
}

func (m *Map) collapse(id, keep int) {
	n := m.Nodes[id]
	if isLeaf(n) {
		return
	}
	if keep <= 0 {
		n.Collapsed = true
	} else {
		n.Collapsed = false
		for _, cid := range n.Children {
			m.collapse(cid, keep-1)
		}
	}
}

func (m *Map) CollapseLevel(level int) {
	m.collapse(m.Root, level)

	var chain []int
	for current := m.Active; current != m.Root; current = m.Nodes[current].Parent {
		chain = append(chain, current)
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if id := chain[i]; m.Nodes[id].Collapsed {
			m.Active = id
			break
		}
	}
}

// }}}
