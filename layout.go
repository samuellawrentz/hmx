package main

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// constants, ported from ref/hmx.php lines 90-100, 159-173
const (
	connLeftLen        = 6
	connRightLen       = 4
	leftPadding        = 1
	widthTolerance     = 1.3
	verticalOffset     = 4
	maxParentNodeWidth = 25
	maxLeafNodeWidth   = 55
	lineSpacing        = 1
)

var (
	connRight  = strings.Repeat("─", connRightLen-2)
	connLeft   = strings.Repeat("─", connLeftLen-2)
	connSingle = strings.Repeat("─", connLeftLen+connRightLen-3)
)

// wordwrap ports PHP's builtin wordwrap($s, $width, "\n"), byte based, cut=false.
func wordwrap(s string, width int) string {
	text := []byte(s)
	n := len(text)
	out := make([]byte, 0, n)
	laststart, lastspace, current := 0, 0, 0
	for ; current < n; current++ {
		switch text[current] {
		case '\n':
			out = append(out, text[laststart:current+1]...)
			laststart = current + 1
			lastspace = laststart
		case ' ':
			if current-laststart >= width {
				out = append(out, text[laststart:current]...)
				out = append(out, '\n')
				laststart = current + 1
			}
			lastspace = current
		default:
			if current-laststart >= width && current != 0 && lastspace > laststart {
				out = append(out, text[laststart:lastspace]...)
				out = append(out, '\n')
				laststart = lastspace + 1
				lastspace = laststart
			}
		}
	}
	if laststart != current {
		out = append(out, text[laststart:current]...)
	}
	return string(out)
}

// mmput ports mmput 3670.
func mmput(m *Map, x int, y float64, s string) {
	yi := int(math.Round(y))
	if yi < 0 || yi >= len(m.rows) {
		return
	}
	row := m.rows[yi]
	sw := runewidth.StringWidth(s)
	head := row
	if x < len(row) {
		head = row[:x]
	}
	var tail []rune
	if x+sw < len(row) {
		tail = row[x+sw:]
	}
	newRow := make([]rune, 0, len(head)+utf8.RuneCountInString(s)+len(tail))
	newRow = append(newRow, head...)
	newRow = append(newRow, []rune(s)...)
	newRow = append(newRow, tail...)
	m.rows[yi] = newRow
}

// wrapLines: wrap only when mb_strlen(title) > width_tolerance*max_width.
func wrapLines(title string, maxWidth int) []string {
	if float64(utf8.RuneCountInString(title)) > widthTolerance*float64(maxWidth) {
		return strings.Split(wordwrap(title, maxWidth), "\n")
	}
	return []string{title}
}

// atTheEnd ports the repeated `$node['is_leaf'] || $node['collapsed'] || ...` check.
func atTheEnd(node *Node) bool { return isLeaf(node) || node.Collapsed }

func maxWidthFor(node *Node) int {
	if atTheEnd(node) {
		return maxLeafNodeWidth
	}
	return maxParentNodeWidth
}

// calculateXAndLh ports calculate_x_and_lh 748 (align_levels dropped, always 0).
func calculateXAndLh(m *Map, id int) {
	node := m.Nodes[id]
	title := nodeText(m, node)
	parent := m.Nodes[node.Parent]

	node.x = parent.x + parent.w + connLeftLen + connRightLen + 1
	if node.Parent == 0 {
		node.x += 1 - connRightLen - connLeftLen
	}

	lines := wrapLines(title, maxWidthFor(node))
	if len(lines) > 1 {
		node.w, node.wl = 0, 0
		for _, line := range lines {
			t := strings.TrimSpace(line)
			node.w = max(node.w, runewidth.StringWidth(t))
			node.wl = max(node.wl, utf8.RuneCountInString(t))
		}
	} else {
		t := strings.TrimSpace(title)
		node.w = runewidth.StringWidth(t)
		node.wl = utf8.RuneCountInString(t)
	}
	node.lh = len(lines)

	m.mapWidth = max(m.mapWidth, node.x+node.w)

	node.clh = 0
	if atTheEnd(node) {
		node.clh = node.lh
	}
	for _, cid := range node.Children {
		calculateXAndLh(m, cid)
		node.clh += m.Nodes[cid].clh
	}
}

// calculateH ports calculate_h 863.
func calculateH(m *Map) {
	for unfinished := true; unfinished; {
		unfinished = false
		for _, node := range m.Nodes {
			if isLeaf(node) || node.Collapsed {
				node.h = lineSpacing + node.lh
				continue
			}
			h, unready := 0, false
			for _, cid := range node.Children {
				if c := m.Nodes[cid]; c.h >= 0 {
					h += c.h
				} else {
					unready = true
					break
				}
			}
			if unready {
				unfinished = true
			} else {
				node.h = max(h, node.lh+lineSpacing)
			}
		}
	}
}

// calculateY and calculateChildrenY port calculate_y 912 / calculate_children_y 922.
func calculateY(m *Map) {
	m.mapTop, m.mapBottom = 0, 0
	m.mapHeight = m.Nodes[0].h
	m.Nodes[0].y = 0
	calculateChildrenY(m, 0)
}

func calculateChildrenY(m *Map, pid int) {
	node := m.Nodes[pid]
	y := node.y
	node.yo = int(math.Round(float64(node.h-node.lh) / 2))

	if node.Collapsed {
		return
	}
	for _, cid := range node.Children {
		c := m.Nodes[cid]
		c.y = y
		m.mapBottom = max(m.mapBottom, c.lh+lineSpacing+y)
		m.mapTop = min(m.mapTop, y)
		y += c.h
		calculateChildrenY(m, cid)
	}
}

// calculateHeightShift ports calculate_height_shift 960.
func calculateHeightShift(m *Map, id, shift int) {
	node := m.Nodes[id]
	node.yo += shift
	shift += max(0, int(math.Floor(float64(node.lh-node.clh)/2-0.9)))

	if node.Collapsed {
		return
	}
	for _, cid := range node.Children {
		calculateHeightShift(m, cid, shift)
	}
}

// drawConnections ports draw_connections 974 (align_levels dropped, always 0).
func drawConnections(m *Map, id int) {
	node := m.Nodes[id]
	numChildren := len(node.Children)

	if node.Collapsed && numChildren > 0 {
		mmput(m, node.x+node.w+1, float64(node.y+node.yo), " [+]")
		return
	}

	if numChildren == 0 {
		return
	}

	if numChildren == 1 {
		childID := node.Children[0]
		child := m.Nodes[childID]
		y1 := math.Round(float64(node.y+node.yo)) + math.Round(float64(node.lh)/2-0.6)
		y2 := math.Round(float64(child.y+child.yo)) + math.Round(float64(child.lh)/2-0.6)
		x := child.x - connLeftLen - connRightLen

		line := "──" + connSingle

		mn, mx := math.Min(y1, y2), math.Max(y1, y2)
		mmput(m, x, mn, line)

		if math.Abs(mn-y2) > 0 {
			for yy := mn; yy < mx; yy++ {
				mmput(m, child.x-2, yy, "│")
			}
			corner, corner2 := "╭", "╯"
			if y2 > y1 {
				corner, corner2 = "╰", "╮"
			}
			mmput(m, child.x-2, y2, corner)
			mmput(m, child.x-2, mn, corner2)
		}

		drawConnections(m, childID)
		return
	}

	// more than one child
	bottom, bottomChild := 0, 0
	top, topChild := m.mapHeight, 0
	for _, cid := range node.Children {
		c := m.Nodes[cid]
		if c.y+c.yo > bottom {
			bottom, bottomChild = c.y+c.yo, cid
		}
		if c.y+c.yo < top {
			top, topChild = c.y+c.yo, cid
		}
	}

	middle := int(math.Round(float64(node.y+node.yo))) + int(math.Round(float64(node.lh)/2-0.6))
	topC := m.Nodes[topChild]

	mmput(m, topC.x-connLeftLen-connRightLen, float64(middle), "──"+connLeft)

	for i := top; i < bottom; i++ {
		mmput(m, topC.x-connRightLen, float64(i), "│")
	}
	mmput(m, topC.x-connRightLen, float64(top), "╭"+connRight)
	mmput(m, topC.x-connRightLen, float64(bottom), "╰"+connRight)

	if numChildren > 2 {
		for _, cid := range node.Children {
			if cid != topChild && cid != bottomChild {
				c := m.Nodes[cid]
				y := float64(c.y) + float64(c.lh)/2 - 0.2 + float64(c.yo)
				mmput(m, c.x-connRightLen, y, "├"+connRight)
			}
		}
	}

	existingChar := rune(0)
	if px, row := topC.x-connRightLen, m.rows[middle]; px >= 0 && px < len(row) {
		existingChar = row[px]
	}
	switch existingChar {
	case '│':
		mmput(m, topC.x-connRightLen, float64(middle), "┤")
	case '╭':
		mmput(m, topC.x-connRightLen, float64(middle), "┬")
	case '├':
		mmput(m, topC.x-connRightLen, float64(middle), "┼")
	}

	for _, cid := range node.Children {
		drawConnections(m, cid)
	}
}

// calculateXo ports calculate_xo 1221.
func calculateXo(m *Map) {
	for _, node := range m.Nodes {
		node.xo = 0
		for _, fore := range m.Nodes {
			if fore.y+fore.yo == node.y+node.yo && fore.x < node.x {
				node.xo += utf8.RuneCountInString(fore.Title) - runewidth.StringWidth(fore.Title)
			}
		}
	}
}

// addContentToTheMap ports add_content_to_the_map 1246.
func addContentToTheMap(m *Map, id int) {
	node := m.Nodes[id]
	for i, line := range wrapLines(nodeText(m, node), maxWidthFor(node)) {
		mmput(m, node.x+node.xo, float64(node.y+node.yo+i), line+" ")
	}
	if !node.Collapsed {
		for _, cid := range node.Children {
			addContentToTheMap(m, cid)
		}
	}
}

// Build ports build_map 1298 (align_levels dropped, always 0).
func Build(m *Map, termW, termH int) {
	for _, n := range m.Nodes {
		n.x, n.y, n.h, n.lh = -1, -1, -1, -1
	}
	root0 := m.Nodes[0]
	root0.x, root0.xo, root0.w, root0.lh = 0, 0, leftPadding, 1
	m.mapWidth, m.mapHeight, m.mapTop, m.mapBottom = 0, 0, 0, 0

	calculateXAndLh(m, m.Root)
	calculateH(m)
	calculateY(m)
	calculateHeightShift(m, m.Root, 0)
	calculateXo(m)

	height := max(m.mapBottom, termH)
	width := max(m.mapWidth, termW)
	m.rows = make([][]rune, height+1)
	for i := range m.rows {
		m.rows[i] = []rune(strings.Repeat(" ", width))
	}

	drawConnections(m, m.Root)
	addContentToTheMap(m, m.Root)
}

// Center ports center_active_node_vh(false) 2056.
func (m *Map) Center(termW, termH int) {
	node := m.Nodes[m.Active]
	midx := float64(node.w)/2 + float64(node.x)
	midy := float64(node.lh)/2 + float64(node.y+node.yo)

	m.Left = max(0, int(math.Round(midx-float64(termW)/2)))
	m.Top = int(math.Round(midy - float64(termH)/2))
}

// MoveWindow ports move_window 2650.
func (m *Map) MoveWindow(termW, termH int) {
	node := m.Nodes[m.Active]
	x1 := max(0, node.x-connRightLen-2)
	x2 := node.x + node.w + 2
	y1 := max(0, node.y+node.yo-verticalOffset)
	y2 := y1 + node.lh + verticalOffset*2

	m.Left = min(m.Left, x1)
	m.Left = max(m.Left, x2-termW)
	m.Top = min(m.Top, y1)
	m.Top = max(m.Top, y2-termH)
}

// Screen is the plain-text slice of display 3407 used by the tests: no colour,
// no body pane, no breadcrumb, no logo.
func (m *Map) Screen(termW, termH int) string {
	lines := make([]string, termH)
	for y := 0; y < termH; y++ {
		var line []rune
		if ry := y + m.Top; ry >= 0 && ry < len(m.rows) {
			row := m.rows[ry]
			start := min(m.Left, len(row))
			end := min(start+termW, len(row))
			line = row[start:end]
		}
		s := string(line)
		if w := runewidth.StringWidth(s); w < termW {
			s += strings.Repeat(" ", termW-w)
		}
		lines[y] = s
	}
	return strings.Join(lines, "\n")
}
