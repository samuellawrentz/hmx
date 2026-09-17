package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
)

// Candidate is one completable link target: a map root or a node inside a map.
type Candidate struct {
	Map, Path, Title, Insert string // Path = ancestor titles below the root, " › " joined, "" at depth 0 and 1
	Node                     bool   // false for the map's root line
}

// Context is the dim right-hand column: "map" for a root line, else the map name plus the ancestor path.
func (c Candidate) Context() string {
	if !c.Node {
		return "map"
	}
	if c.Path == "" {
		return c.Map
	}
	return c.Map + " › " + c.Path
}

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
				out = append(out, Candidate{Map: r.Name, Title: title, Insert: "[[" + r.Name + "]]"})
				continue
			}
			path := ""
			if depth > 1 {
				path = strings.Join(stack[1:depth], " › ")
			}
			out = append(out, Candidate{Map: r.Name, Path: path, Title: title, Insert: "[[" + r.Name + "#" + title + "]]", Node: true})
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

// allTermsMatch reports whether every whitespace-separated term of q is a fuzzy subsequence of hay.
func allTermsMatch(q, hay string) bool {
	for _, term := range strings.Fields(q) {
		if !fuzzyMatch(term, hay) {
			return false
		}
	}
	return true
}

// candScore ranks a surviving candidate against q for display order only; it never decides which
// candidates survive filterCandidates. Higher is better: an exact title match, then a title prefix,
// then a title substring, then a title subsequence, then a candidate that only matched via the path.
func candScore(c Candidate, q string) int {
	q = strings.ToLower(strings.TrimSpace(q))
	title := strings.ToLower(c.Title)
	switch {
	case title == q:
		return 4
	case strings.HasPrefix(title, q):
		return 3
	case strings.Contains(title, q):
		return 2
	case fuzzyMatch(q, title):
		return 1
	default:
		return 0
	}
}

// filterCandidates keeps candidates matching every whitespace-separated term of q, then ranks the
// survivors by candScore (source order breaks ties). "#" scopes the query to one map: the left side
// matches the map name, the right side matches the node's path+title, and only node rows survive (a
// map row has no "[[map#]]" form). Without "#", the match is against "Context() Title", so a query
// can mix a context term with a title term.
func filterCandidates(all []Candidate, q string) []Candidate {
	var out []Candidate
	scope, rest, hasHash := strings.Cut(q, "#")
	scoreQ := q
	if hasHash {
		scoreQ = rest
	}
	for _, c := range all {
		if hasHash {
			if !c.Node || !allTermsMatch(scope, c.Map) || !allTermsMatch(rest, strings.TrimSpace(c.Path+" "+c.Title)) {
				continue
			}
		} else if !allTermsMatch(q, c.Context()+" "+c.Title) {
			continue
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return candScore(out[i], scoreQ) > candScore(out[j], scoreQ) })
	return out
}
