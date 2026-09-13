package main

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

var (
	styleActive = tcell.StyleDefault.Foreground(tcell.PaletteColor(0)).Background(tcell.PaletteColor(172)).Bold(true)
	styleMsg    = tcell.StyleDefault.Foreground(tcell.PaletteColor(0)).Background(tcell.PaletteColor(141)).Bold(true)
)

// draw ports display(), ref/hmx.php 3407-3660: no body pane, no breadcrumb, no logo (phases 5-6).
func (a *App) draw() {
	if a.m == nil { // list mode: no map loaded yet, magic_readline's Esc path calls draw()
		return
	}
	w, h := a.s.Size()
	a.m.MoveWindow(w, h)
	a.s.Clear()

	node := a.m.Nodes[a.m.Active]
	x1 := max(0, node.x+node.xo-1-a.m.Left)
	x2 := node.wl + x1 + 2
	y1 := max(0, node.y+node.yo-a.m.Top)
	y2 := node.lh + y1

	query := []rune(strings.ToLower(a.query))

	for y := 0; y < h; y++ {
		row := rowAt(a.m, y, w)
		styles := styleRow(row, y, x1, x2, y1, y2, query)
		putRow(a.s, 0, y, row, styles)
	}

	if bc := a.breadcrumb(); bc != "" {
		putStr(a.s, 0, 0, bc, tcell.StyleDefault)
	}

	putMessage(a.s, w, h, a.msg)
	a.s.Show()
}

// rowAt slices m.rows the same way Map.Screen does, padded to w runes.
func rowAt(m *Map, y, w int) []rune {
	var row []rune
	if ry := y + m.Top; ry >= 0 && ry < len(m.rows) {
		r := m.rows[ry]
		start := min(m.Left, len(r))
		end := min(start+w, len(r))
		row = append([]rune(nil), r[start:end]...)
	}
	for len(row) < w {
		row = append(row, ' ')
	}
	return row
}

// styleRow assigns a style per rune of one screen row, porting the colouring
// rules in display(): connectors, "[+]", links/ellipsis (non-active rows),
// "(?)"/"???", "{...}" dimming, and query-match reverse video.
func styleRow(row []rune, y, x1, x2, y1, y2 int, query []rune) []tcell.Style {
	n := len(row)
	fg := make([]int, n)
	for i := range fg {
		fg[i] = -1
	}
	dim := make([]bool, n)
	activeRow := y >= y1 && y < y2

	dimOn := false
	for i := 0; i < n; i++ {
		r := row[i]

		if r >= 0x2500 && r <= 0x2571 {
			fg[i] = 95
		}

		if r == '{' {
			dimOn = true
		}
		if dimOn {
			dim[i] = true
		}
		if r == '}' {
			dimOn = false
		}

		if r == '(' && i+2 < n && row[i+1] == '?' && row[i+2] == ')' {
			fg[i], fg[i+1], fg[i+2] = 168, 168, 168
		}
		if r == '?' && i+2 < n && row[i+1] == '?' && row[i+2] == '?' {
			fg[i], fg[i+1], fg[i+2] = 168, 168, 168
		}

		if r == ' ' && i+3 < n && row[i+1] == '[' && row[i+2] == '+' && row[i+3] == ']' {
			fg[i+1], fg[i+2], fg[i+3] = 215, 215, 215
		}

		if !activeRow {
			if r == '…' {
				fg[i] = 214
			}
			if r == '[' && i+1 < n && row[i+1] == '[' {
				j := i + 2
				for j < n && row[j] != ']' {
					j++
				}
				if j+1 < n && row[j+1] == ']' {
					for k := i; k <= j+1; k++ {
						fg[k] = 33
					}
				}
			}
		}
	}

	queried := matchMask(row, query)

	styles := make([]tcell.Style, n)
	for i := 0; i < n; i++ {
		st := tcell.StyleDefault
		if fg[i] >= 0 {
			st = st.Foreground(tcell.PaletteColor(fg[i]))
		}
		if dim[i] {
			st = st.Dim(true)
		}
		if activeRow && i >= x1 && i < x2 {
			st = styleActive
		}
		if queried[i] {
			st = st.Reverse(true)
		}
		styles[i] = st
	}
	return styles
}

// matchMask marks every rune covered by a case-insensitive occurrence of query in row.
func matchMask(row, query []rune) []bool {
	mask := make([]bool, len(row))
	if len(query) == 0 {
		return mask
	}
	lower := make([]rune, len(row))
	for i, r := range row {
		lower[i] = unicode.ToLower(r)
	}
	for i := 0; i+len(query) <= len(lower); i++ {
		match := true
		for j, qr := range query {
			if lower[i+j] != qr {
				match = false
				break
			}
		}
		if match {
			for j := range query {
				mask[i+j] = true
			}
		}
	}
	return mask
}

func putRow(s tcell.Screen, x, y int, row []rune, styles []tcell.Style) {
	cx := x
	for i, r := range row {
		s.SetContent(cx, y, r, nil, styles[i])
		cx += runewidth.RuneWidth(r)
	}
}

func putStr(s tcell.Screen, x, y int, str string, style tcell.Style) {
	cx := x
	for _, r := range str {
		s.SetContent(cx, y, r, nil, style)
		cx += runewidth.RuneWidth(r)
	}
}

// putMessage ports message(), ref/hmx.php 2589: right-aligned on the bottom row, "..." prefix when too long.
func putMessage(s tcell.Screen, w, h int, msg string) {
	if msg == "" {
		return
	}
	body := []rune(msg)
	extralen := w - len(body) - 2
	if extralen < 1 {
		cut := -extralen + 4
		if cut > len(body) {
			cut = len(body)
		}
		if cut < 0 {
			cut = 0
		}
		body = append([]rune("..."), body[len(body)-cut:]...)
	}
	x := max(0, extralen)
	putStr(s, x, h-1, " "+string(body)+" ", styleMsg)
}
