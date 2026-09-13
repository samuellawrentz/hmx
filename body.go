package main

import (
	"os"
	"os/exec"
	"strings"
)

// shQuote wraps s in single quotes for safe use inside a sh -c string.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// editBody ports edit_body, ref/hmx.php 4107.
func (a *App) editBody() {
	n := a.m.Nodes[a.m.Active]
	tmp, err := os.CreateTemp("", "hmx*.md")
	if err != nil {
		return
	}
	path := tmp.Name()
	tmp.WriteString(n.Title + "\n\n" + n.Body + "\n")
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command("sh", "-c", editor+" "+shQuote(path))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	a.s.Suspend()
	cmd.Run()
	a.s.Resume()

	b, _ := os.ReadFile(path)
	os.Remove(path)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	title := strings.TrimSpace(lines[0])
	body := strings.TrimSpace(strings.Join(lines[1:], "\n"))
	if title == "" {
		title = n.Title
	}
	a.m.PushChange()
	n.Title = title
	n.Body = body
	a.build()
}
