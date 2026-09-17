package main

import (
	"os"
	"regexp"
	"strings"
)

// Candidate is one completable link target: a map root or a node inside a map.
type Candidate struct{ Title, Context, Insert string }

var bodyLineRe = regexp.MustCompile(`^\t*> `)

// linkCandidates scans dir's maps once: every non-body, non-blank line becomes a candidate.
func linkCandidates(dir string) []Candidate {
	var out []Candidate
	for _, r := range listRows(dir, "") {
		b, err := os.ReadFile(r.File)
		if err != nil {
			continue
		}
		var stack []string
		for _, line := range strings.Split(string(b), "\n") {
			if bodyLineRe.MatchString(line) || strings.TrimSpace(line) == "" {
				continue
			}
			depth := 0
			for depth < len(line) && line[depth] == '\t' {
				depth++
			}
			title := strings.TrimSpace(line[depth:])
			stack = append(stack[:min(depth, len(stack))], title)

			if depth == 0 {
				out = append(out, Candidate{Title: title, Context: "map", Insert: "[[" + r.Name + "]]"})
				continue
			}
			ctx := r.Name
			if depth > 1 {
				ctx = r.Name + " › " + strings.Join(stack[1:depth], " › ")
			}
			out = append(out, Candidate{Title: title, Context: ctx, Insert: "[[" + r.Name + "#" + title + "]]"})
		}
	}
	return out
}

// fuzzyMatch reports whether q's runes appear in s in order, case-insensitive.
func fuzzyMatch(q, s string) bool {
	qr := []rune(strings.ToLower(q))
	if len(qr) == 0 {
		return true
	}
	i := 0
	for _, r := range strings.ToLower(s) {
		if r == qr[i] {
			i++
			if i == len(qr) {
				return true
			}
		}
	}
	return false
}

// filterCandidates keeps candidates whose "Title Context" fuzzy-matches every whitespace-separated
// term of q, in source order. Per term, so "q3 red" finds a title under a matching ancestor path.
func filterCandidates(all []Candidate, q string) []Candidate {
	var out []Candidate
	for _, c := range all {
		hay := c.Title + " " + c.Context
		ok := true
		for _, term := range strings.Fields(q) {
			ok = ok && fuzzyMatch(term, hay)
		}
		if ok {
			out = append(out, c)
		}
	}
	return out
}
