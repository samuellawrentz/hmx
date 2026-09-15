package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	b, err := os.ReadFile(filepath.Join("test/fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func backend(t *testing.T) *Map { return Parse(fixture(t, "backend.hmm")) }

func ids(m *Map, id int) string {
	var s []string
	for _, c := range m.Nodes[id].Children {
		s = append(s, strconv.Itoa(c))
	}
	return strings.Join(s, ",")
}

func TestParseBackend(t *testing.T) {
	m := backend(t)
	if m.Root != 2 || m.Active != 2 {
		t.Fatalf("root=%d active=%d", m.Root, m.Active)
	}
	want := map[int][2]string{2: {"backend", ""}, 3: {"auth", ""}, 4: {"JWT rotation", "Rotate every 24h."}, 5: {"refresh", ""}, 6: {"[[infra#redis]]", ""}}
	for id, w := range want {
		n := m.Nodes[id]
		if n == nil || n.Title != w[0] || n.Body != w[1] {
			t.Errorf("node %d = %+v, want %v", id, n, w)
		}
	}
	if m.Nodes[3].Parent != 2 || m.Nodes[4].Parent != 3 || m.Nodes[6].Parent != 2 || m.Nodes[2].Parent != 0 {
		t.Error("parents wrong")
	}
	if ids(m, 2) != "3,6" || ids(m, 3) != "4,5" || len(m.Nodes[4].Children) != 0 {
		t.Errorf("children: 2=%s 3=%s", ids(m, 2), ids(m, 3))
	}
	if len(m.Nodes) != 6 { // 0 hidden super-root + 2..6
		t.Errorf("len=%d", len(m.Nodes))
	}
}

func TestParseEdgeCases(t *testing.T) {
	m := Parse("a\n>x\n\t> body one\n\t> \nb\n")
	if m.Root != 1 || m.Nodes[1].Title != "root" || ids(m, 1) != "2,3,4" {
		t.Fatalf("multi-root: root=%d title=%q kids=%s", m.Root, m.Nodes[1].Title, ids(m, 1))
	}
	if m.Nodes[3].Title != ">x" || m.Nodes[3].Body != "body one\n" {
		t.Errorf(">x is a node with body %q, got %+v", "body one\n", m.Nodes[3])
	}
	if m.Nodes[4].Title != "b" || m.Nodes[4].Parent != 1 {
		t.Errorf("b: %+v", m.Nodes[4])
	}
	if Parse("").Nodes[1].Title != "root" {
		t.Error("empty map has root 1")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	files, _ := filepath.Glob("test/fixtures/*.hmm")
	for _, f := range files {
		b, _ := os.ReadFile(f)
		want := strings.TrimRight(string(b), "\n") + "\n" // unlike php, exactly one trailing newline
		if got := Parse(string(b)).Serialize(); got != want {
			t.Errorf("%s: round trip differs\n%s", f, got)
		}
	}
	if got := Parse("a\nb\n").Serialize(); got != "root\n\ta\n\tb\n" {
		t.Errorf("multi-root serialize %q", got)
	}
}

func TestLoadSaveNodeCount(t *testing.T) {
	d := t.TempDir()
	f := filepath.Join(d, "x.hmm")
	os.WriteFile(f, []byte(fixture(t, "backend.hmm")), 0644)
	m, err := Load(f)
	if err != nil || m.File != f || m.Nodes[2].Title != "backend" {
		t.Fatal(err, m)
	}
	m.Nodes[2].Title = "changed"
	m.Modified = true
	if err := m.Save(); err != nil || m.Modified {
		t.Fatal(err, m.Modified)
	}
	if b, _ := os.ReadFile(f); !strings.HasPrefix(string(b), "changed\n\tauth\n") {
		t.Errorf("saved %q", b)
	}
	if n := NodeCount(f); n != 5 {
		t.Errorf("count=%d", n)
	}
	if n := NodeCount("test/fixtures/deep.hmm"); n != 60 {
		t.Errorf("deep count=%d", n)
	}
	m, err = Load(filepath.Join(d, "missing.hmm"))
	if err != nil || m.Root != 1 || m.Nodes[1].Title != "missing" || len(m.Nodes[1].Children) != 0 {
		t.Errorf("missing file → empty map named after file: %v %+v", err, m.Nodes[1])
	}
}

func TestInsert(t *testing.T) {
	m := backend(t)
	m.Active = 3
	if id := m.InsertSibling(); id != 7 || m.Active != 7 || m.Nodes[7].Parent != 2 || m.Nodes[7].Title != "NEW" || ids(m, 2) != "3,7,6" {
		t.Errorf("sibling: id=%d active=%d kids=%s", id, m.Active, ids(m, 2))
	}
	m.Active = 4
	if id := m.InsertChild(); id != 8 || m.Nodes[8].Parent != 4 || ids(m, 4) != "8" {
		t.Errorf("child: id=%d kids=%s", id, ids(m, 4))
	}
	m.Nodes[3].Collapsed = true
	m.Active = 4
	m.InsertSibling()
	if m.Nodes[3].Collapsed {
		t.Error("parent stays collapsed")
	}
	m.Active = 2
	m.InsertSibling() // sibling of root becomes a child
	if m.Nodes[m.Active].Parent != 2 {
		t.Error("sibling of root must be child")
	}
	if !m.Modified {
		t.Error("modified")
	}
}

func TestDelete(t *testing.T) {
	m := backend(t)
	m.Active = 3
	clip := m.Delete()
	if clip != "auth\n\tJWT rotation\n\t> Rotate every 24h.\n\trefresh\n" {
		t.Errorf("clip %q", clip)
	}
	if m.Nodes[3] != nil || m.Nodes[4] != nil || m.Nodes[5] != nil || ids(m, 2) != "6" || m.Active != 6 {
		t.Errorf("after delete: active=%d kids=%s", m.Active, ids(m, 2))
	}
	m.Delete()
	if m.Active != 2 || len(m.Nodes[2].Children) != 0 {
		t.Error("deleting last child activates parent")
	}
	m = backend(t)
	m.Active = 6
	m.Delete()
	if m.Active != 3 {
		t.Error("previous sibling preferred")
	}
	m.Active = 2
	if clip := m.Delete(); clip != "auth\n\tJWT rotation\n\t> Rotate every 24h.\n\trefresh\n" || m.Nodes[2] == nil || len(m.Nodes[2].Children) != 0 {
		t.Errorf("delete on root clears children only: %q", clip)
	}
}

func TestMove(t *testing.T) {
	m := backend(t)
	m.Active = 3
	m.MoveDown()
	if ids(m, 2) != "6,3" {
		t.Errorf("down: %s", ids(m, 2))
	}
	m.MoveDown()
	if ids(m, 2) != "6,3" {
		t.Error("down at end is a no-op")
	}
	m.MoveUp()
	if ids(m, 2) != "3,6" {
		t.Errorf("up: %s", ids(m, 2))
	}
	m.MoveUp()
	if ids(m, 2) != "3,6" {
		t.Error("up at start is a no-op")
	}
}

func TestYankPaste(t *testing.T) {
	m := backend(t)
	m.Active = 3
	text := m.Yank()
	if text != "auth\n\tJWT rotation\n\t> Rotate every 24h.\n\trefresh\n" || m.Modified {
		t.Errorf("yank %q modified=%v", text, m.Modified)
	}
	m.Active = 6
	m.Paste(text, false)
	if ids(m, 6) != "7" || m.Nodes[7].Title != "auth" || m.Nodes[7].Parent != 6 || m.Nodes[8].Body != "Rotate every 24h." || m.Nodes[8].Parent != 7 || m.Active != 7 {
		t.Errorf("paste children: active=%d 6=%s 7=%+v", m.Active, ids(m, 6), m.Nodes[7])
	}
	m.Active = 3
	m.Paste("x\ny\n", true)
	if ids(m, 2) != "3,10,11,6" {
		t.Errorf("paste siblings: %s", ids(m, 2))
	}
	m.Active = 2
	before := ids(m, 0)
	m.Paste("z", true)
	if ids(m, 0) != before || m.Nodes[12] != nil {
		t.Error("paste sibling on root is a no-op")
	}
	m.Nodes[6].Collapsed = true
	m.Active = 6
	m.Paste("w", false)
	if m.Nodes[6].Collapsed {
		t.Error("paste expands target")
	}
}

// TestChildren covers YankChildren/DeleteChildren (yank_children/delete_children, ref/hmx.php 3082/3099).
// Fixture ids are sequential from 2 in file order: root=2, a=3, b=4, c=5, d=6, e=7.
func TestChildren(t *testing.T) {
	orig := "root\n\ta\n\t\tb\n\t\t\tc\n\t\td\n\te\n"
	m := Parse(orig)

	m.Active = 3
	if text := m.YankChildren(); text != "b\n\tc\nd\n" || m.Serialize() != orig || m.Modified {
		t.Errorf("yank children: %q modified=%v", text, m.Modified)
	}

	if text := m.DeleteChildren(); text != "b\n\tc\nd\n" {
		t.Errorf("delete children: %q", text)
	}
	if len(m.Nodes[3].Children) != 0 || m.Nodes[4] != nil || m.Nodes[5] != nil || m.Nodes[6] != nil || m.Active != 3 || !m.Modified {
		t.Errorf("after delete children: active=%d nodes 4,5,6=%v,%v,%v", m.Active, m.Nodes[4], m.Nodes[5], m.Nodes[6])
	}
	m.Undo()
	if m.Serialize() != orig {
		t.Errorf("undo delete children: %q", m.Serialize())
	}

	m.Active = 7
	m.Paste("b\n\tc\nd\n", false)
	want := "root\n\ta\n\t\tb\n\t\t\tc\n\t\td\n\te\n\t\tb\n\t\t\tc\n\t\td\n"
	if got := m.Serialize(); got != want {
		t.Errorf("paste children: %q", got)
	}

	leaf := Parse(orig)
	leaf.Active = 5 // c is a leaf
	if text := leaf.YankChildren(); text != "" {
		t.Errorf("yank leaf: %q", text)
	}
	if text := leaf.DeleteChildren(); text != "" || leaf.Modified {
		t.Errorf("delete leaf: %q modified=%v", text, leaf.Modified)
	}
}

func TestUndo(t *testing.T) {
	m := backend(t)
	m.Undo() // empty stack
	m.PushChange()
	m.Nodes[2].Title = "x"
	m.Active = 3
	m.PushChange()
	m.Nodes[3].Title = "y"
	m.Undo()
	if m.Nodes[3].Title != "auth" || m.Nodes[2].Title != "x" || m.Active != 3 {
		t.Errorf("undo 1: %q %q active=%d", m.Nodes[2].Title, m.Nodes[3].Title, m.Active)
	}
	m.Undo()
	if m.Nodes[2].Title != "backend" || m.Active != 2 {
		t.Errorf("undo 2: %q active=%d", m.Nodes[2].Title, m.Active)
	}
	m.Undo() // stack empty again, no-op
	m.Active = 3
	m.Delete()
	m.Undo()
	if m.Nodes[3] == nil || ids(m, 2) != "3,6" {
		t.Error("undo delete restores subtree")
	}
	m.Active = 3
	m.InsertChild()
	m.Undo()
	if len(m.Nodes[3].Children) != 2 {
		t.Error("undo insert removes the node")
	}
	for i := 0; i < 30; i++ {
		m.PushChange()
	}
	if len(m.undo) != 24 {
		t.Errorf("undo capped at 24, got %d", len(m.undo))
	}
}
