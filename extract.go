package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slug ports the mb_ereg_replace/mb_strtolower call in extract_to_map, ref/hmx.php 4090.
func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// extractSubtree ports extract_subtree, ref/hmx.php 4055.
func (a *App) extractSubtree(id int, key string) bool {
	file := filepath.Join(a.mapDir, key+".hmm")
	if _, err := os.Stat(file); err == nil {
		a.msg = key + ".hmm exists, pick another name"
		return false
	}
	a.m.PushChange()
	n := a.m.Nodes[id]
	n.Title = key
	os.WriteFile(file, []byte(mapToList(a.m.Nodes, id, false, 0)), 0644)
	n.Title = "[[" + key + "]]"
	n.Body = ""
	a.m.deleteChildren(id)
	n.Children = nil
	a.build()
	return true
}

// extractToMap ports extract_to_map, ref/hmx.php 4084.
func extractToMap(a *App) {
	m := a.m
	id := m.Active
	if id == m.Root {
		return
	}
	def := strings.TrimSuffix(filepath.Base(m.File), ".hmm") + "-" + slug(m.Nodes[id].Title)
	key, ok := a.readline(def)
	if !ok || key == "" {
		return
	}
	a.extractSubtree(id, key)
}
