package main

import (
	"math"
	"os/exec"
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

func (a *App) build() { w, h := a.s.Size(); Build(a.m, w, h) }

var actions = map[string]func(*App){
	"go_left":                func(a *App) { changeActiveNode(a, -1, 0) },
	"go_down":                func(a *App) { changeActiveNode(a, 0, 1) },
	"go_up":                  func(a *App) { changeActiveNode(a, 0, -1) },
	"go_right":               func(a *App) { changeActiveNode(a, 1, 0) },
	"toggle_node":            toggleNode,
	"focus":                  focus,
	"expand_all":             expandAllAction,
	"collapse_all":           collapseAllAction,
	"save":                   save,
	"quit":                   quitAction,
	"help":                   help,
	"insert_new_sibling":     insertNewSibling,
	"insert_new_child":       insertNewChild,
	"edit_node":              editNode,
	"delete_node":            deleteNode,
	"yank_node":              yankNode,
	"paste_as_children":      pasteAsChildren,
	"paste_as_siblings":      pasteAsSiblings,
	"move_node_down":         moveNodeDown,
	"move_node_up":           moveNodeUp,
	"undo":                   undoAction,
	"search":                 search,
	"next_search_result":     func(a *App) { nextSearchResult(a) },
	"previous_search_result": func(a *App) { previousSearchResult(a) },
	"enter_key":              enterKey,
	"go_back":                (*App).goBack,
	"edit_body":              (*App).editBody,
	"extract_to_map":         extractToMap,
}

var mapKeys = map[string]string{
	"h":          "go_left",
	"j":          "go_down",
	"k":          "go_up",
	"l":          "go_right",
	"Left":       "go_left",
	"Down":       "go_down",
	"Up":         "go_up",
	"Right":      "go_right",
	"Space":      "toggle_node",
	"f":          "focus",
	"0":          "expand_all",
	"1":          "collapse_all",
	"s":          "save",
	"q":          "quit",
	"Ctrl-C":     "quit",
	"?":          "help",
	"o":          "insert_new_sibling",
	"Tab":        "insert_new_child",
	"e":          "edit_node",
	"E":          "edit_body",
	"Ctrl-E":     "extract_to_map",
	"d":          "delete_node",
	"y":          "yank_node",
	"p":          "paste_as_children",
	"P":          "paste_as_siblings",
	"J":          "move_node_down",
	"K":          "move_node_up",
	"u":          "undo",
	"/":          "search",
	"n":          "next_search_result",
	"N":          "previous_search_result",
	"Enter":      "enter_key",
	"Backspace":  "go_back",
	"Backspace2": "go_back",
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
	a.build()
}

// focus ports focus/focus_vh, ref/hmx.php 3223.
func focus(a *App) {
	m := a.m
	collapseSiblings(m, m.Active)
	expandSiblings(m, m.Active)
	a.build()
	m.Center(a.s.Size())
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
	a.build()
	a.m.Center(a.s.Size())
}

func collapseAllAction(a *App) {
	a.m.CollapseAll()
	a.build()
	a.m.Center(a.s.Size())
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

// {{{ hmx additions

// insertNewSibling/insertNewChild port insert_new_node, ref/hmx.php 1623.
func insertNewSibling(a *App) { insertNewNode(a, a.m.InsertSibling) }
func insertNewChild(a *App)   { insertNewNode(a, a.m.InsertChild) }

// enterKey ports enter_key, ref/hmx.php 3835.
func enterKey(a *App) {
	n := a.m.Nodes[a.m.Active]
	if mm := taskRe.FindStringSubmatch(n.Title); mm != nil {
		a.showTask(mm[1])
	} else if strings.Contains(n.Title+n.Body, "[[") {
		a.followLink()
	} else {
		insertNewSibling(a)
	}
}

func insertNewNode(a *App, insert func() int) {
	insert()
	a.build()
	a.draw()
	a.m.Nodes[a.m.Active].Title = ""
	editNodeInner(a, "")
}

// editNode ports edit_node, ref/hmx.php 1989.
func editNode(a *App) {
	n := a.m.Nodes[a.m.Active]
	initial := n.Title
	if (a.m.Active == a.m.Root && initial == "root") || initial == "NEW" {
		initial = ""
	}
	editNodeInner(a, initial)
}

func editNodeInner(a *App, initial string) {
	out, ok := a.readline(initial)
	m := a.m
	n := m.Nodes[m.Active]

	if (!ok || out == "") && n.Title == "" && isLeaf(n) {
		m.Delete()
		m.PushChange()
		a.build()
		return
	}
	if !ok {
		a.msg = "Editing cancelled"
		return
	}
	n.Title = out
	m.PushChange()
	a.build()
}

// deleteNode ports delete_node_vh, ref/hmx.php 3093.
func deleteNode(a *App) {
	a.copyToClipboard(a.m.Delete())
	a.build()
	a.msg = "Item(s) are cut and placed into the clipboard."
}

// yankNode ports yank_node, ref/hmx.php 3075.
func yankNode(a *App) {
	a.copyToClipboard(a.m.Yank())
	a.msg = "Item(s) are copied to the clipboard."
}

// pasteAsChildren/pasteAsSiblings port paste_sub_tree, ref/hmx.php 2897.
func pasteAsChildren(a *App) {
	a.m.Paste(a.getFromClipboard(), false)
	a.build()
}

func pasteAsSiblings(a *App) {
	a.m.Paste(a.getFromClipboard(), true)
	a.build()
}

// copyToClipboard/getFromClipboard port copy_to_clipboard/get_from_clipboard, ref/hmx.php 2984-3030:
// shell out to pbcopy/pbpaste when present, else keep the text in a.clip.
func (a *App) copyToClipboard(text string) {
	if path, err := exec.LookPath("pbcopy"); err == nil {
		cmd := exec.Command(path)
		cmd.Stdin = strings.NewReader(text)
		cmd.Run()
		return
	}
	a.clip = text
}

func (a *App) getFromClipboard() string {
	text := a.clip
	if path, err := exec.LookPath("pbpaste"); err == nil {
		if out, err := exec.Command(path).Output(); err == nil {
			text = string(out)
		}
	}
	return strings.Map(func(r rune) rune {
		if r <= 0x08 || (r >= 0x0B && r <= 0x1F) || r == 0x7F || r == 0xFEFF {
			return -1
		}
		return r
	}, text)
}

// moveNodeDown/moveNodeUp port move_node_down/up, ref/hmx.php 2298/2335.
func moveNodeDown(a *App) {
	a.m.MoveDown()
	a.build()
}

func moveNodeUp(a *App) {
	a.m.MoveUp()
	a.build()
}

// undoAction ports undo, ref/hmx.php 2728.
func undoAction(a *App) {
	a.m.Undo()
	a.build()
}

// search ports search, ref/hmx.php 2199.
func search(a *App) {
	q, ok := a.readline("")
	a.query = q
	if !ok || q == "" {
		return
	}
	if !nextSearchResult(a) {
		previousSearchResult(a)
	}
}

// nextSearchResult/previousSearchResult port next_search_result/previous_search_result, ref/hmx.php 2222-2296.
func nextSearchResult(a *App) bool     { return searchResult(a, true) }
func previousSearchResult(a *App) bool { return searchResult(a, false) }

func searchResult(a *App, forward bool) bool {
	m := a.m
	active := m.Nodes[m.Active]
	cy := active.y + active.yo
	query := strings.ToLower(a.query)

	best, bestY, found := 0, 0, false
	for id, n := range m.Nodes {
		if id == 0 || n.y == -1 {
			continue
		}
		ny := n.y + n.yo
		if forward && ny <= cy {
			continue
		}
		if !forward && ny >= cy {
			continue
		}
		if !strings.Contains(strings.ToLower(n.Title), query) {
			continue
		}
		if !found || (forward && ny < bestY) || (!forward && ny > bestY) {
			bestY, best, found = ny, id, true
		}
	}
	if !found {
		return false
	}
	m.Active = best
	return true
}

// readline ports magic_readline/show_line, ref/hmx.php 1678-1978.
func (a *App) readline(title string) (string, bool) {
	buf := []rune(title)
	cursor := len(buf)
	w, _ := a.s.Size()
	shift := adjustShift(0, cursor, w)
	a.showLine(buf, cursor, shift)

	for {
		ev, ok := a.s.PollEvent().(*tcell.EventKey)
		if !ok {
			continue
		}
		mod := ev.Modifiers()
		switch ev.Key() {
		case tcell.KeyEsc:
			a.draw()
			return "", false
		case tcell.KeyEnter:
			return strings.TrimSpace(string(buf)), true
		case tcell.KeyUp, tcell.KeyHome, tcell.KeyCtrlA:
			cursor = 0
		case tcell.KeyDown, tcell.KeyEnd, tcell.KeyCtrlE:
			cursor = len(buf)
		case tcell.KeyRight:
			if mod&tcell.ModCtrl != 0 {
				cursor = wordRight(buf, cursor)
			} else {
				cursor = min(len(buf), cursor+1)
			}
		case tcell.KeyCtrlF:
			cursor = min(len(buf), cursor+1)
		case tcell.KeyLeft:
			if mod&tcell.ModCtrl != 0 {
				cursor = wordLeft(buf, cursor)
			} else {
				cursor = max(0, cursor-1)
			}
		case tcell.KeyCtrlB:
			cursor = max(0, cursor-1)
		case tcell.KeyCtrlW:
			from := wordLeft(buf, cursor)
			buf = rlDelete(buf, from, cursor)
			cursor = from
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if mod&tcell.ModAlt != 0 {
				from := wordLeft(buf, cursor)
				buf = rlDelete(buf, from, cursor)
				cursor = from
			} else if cursor > 0 {
				buf = rlDelete(buf, cursor-1, cursor)
				cursor--
			}
		case tcell.KeyDelete:
			if mod&tcell.ModCtrl != 0 {
				to := wordRight(buf, cursor)
				buf = rlDelete(buf, cursor, to)
			} else if cursor < len(buf) {
				buf = rlDelete(buf, cursor, cursor+1)
			}
		case tcell.KeyCtrlK:
			buf = buf[:cursor]
		case tcell.KeyCtrlU:
			buf = buf[cursor:]
			cursor = 0
		case tcell.KeyTab:
			buf = rlInsert(buf, cursor, []rune("  "))
			cursor += 2
		case tcell.KeyRune:
			r := ev.Rune()
			if mod&tcell.ModAlt != 0 {
				switch r {
				case 'b':
					cursor = wordLeft(buf, cursor)
				case 'f':
					cursor = wordRight(buf, cursor)
				case 'd':
					to := wordRight(buf, cursor)
					buf = rlDelete(buf, cursor, to)
				}
			} else {
				buf = rlInsert(buf, cursor, []rune{r})
				cursor++
			}
		}
		w, _ = a.s.Size()
		shift = adjustShift(shift, cursor, w)
		a.showLine(buf, cursor, shift)
	}
}

func (a *App) showLine(buf []rune, cursor, shift int) {
	w, h := a.s.Size()
	for i := 0; i < w; i++ {
		r := ' '
		if shift+i < len(buf) {
			r = buf[shift+i]
		}
		st := styleActive
		if shift+i == cursor {
			st = st.Reverse(true)
		}
		a.s.SetContent(i, h-1, r, nil, st)
	}
	a.s.Show()
}

func adjustShift(shift, cursor, w int) int {
	if cursor < shift {
		shift = cursor
	}
	if w > 0 && cursor >= shift+w {
		shift = cursor - w + 1
	}
	if shift < 0 {
		shift = 0
	}
	return shift
}

// wordLeft/wordRight find the word boundary before/after cursor (skip spaces, skip word).
func wordLeft(buf []rune, cursor int) int {
	i := cursor
	for i > 0 && buf[i-1] == ' ' {
		i--
	}
	for i > 0 && buf[i-1] != ' ' {
		i--
	}
	return i
}

func wordRight(buf []rune, cursor int) int {
	i := cursor
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	for i < len(buf) && buf[i] == ' ' {
		i++
	}
	return i
}

func rlDelete(buf []rune, from, to int) []rune {
	return append(buf[:from], buf[to:]...)
}

func rlInsert(buf []rune, cursor int, ins []rune) []rune {
	buf = append(buf, ins...)
	copy(buf[cursor+len(ins):], buf[cursor:len(buf)-len(ins)])
	copy(buf[cursor:], ins)
	return buf
}

// }}}
