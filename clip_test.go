package main

import (
	"os"
	"path/filepath"
	"testing"
)

// yank/cut go through pbcopy, paste through pbpaste; internal string when pbcopy is not on PATH
func TestClipboard(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "pbcopy"), []byte("#!/bin/sh\n/bin/cat > "+d+"/clip\n"), 0755)
	os.WriteFile(filepath.Join(d, "pbpaste"), []byte("#!/bin/sh\n/bin/cat "+d+"/clip\n"), 0755)
	t.Setenv("PATH", d)
	a := &App{}
	a.copyToClipboard("x\n\ty\n")
	if b, _ := os.ReadFile(filepath.Join(d, "clip")); string(b) != "x\n\ty\n" {
		t.Errorf("pbcopy got %q", b)
	}
	if got := a.getFromClipboard(); got != "x\n\ty\n" {
		t.Errorf("pbpaste gave %q", got)
	}
	t.Setenv("PATH", filepath.Join(d, "none"))
	a.copyToClipboard("z")
	if got := a.getFromClipboard(); got != "z" {
		t.Errorf("fallback gave %q", got)
	}
}
