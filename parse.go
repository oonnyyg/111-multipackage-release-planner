package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Commit is a parsed conventional commit.
type Commit struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Scope       string   `json:"scope,omitempty"`
	Description string   `json:"description"`
	Breaking    bool     `json:"breaking"`
	Packages    []string `json:"packages"`
}

// First line: type, optional (scope), optional !, then ": " and a non-empty
// description. Type is lowercase English letters.
var headerRe = regexp.MustCompile(`^([a-z]+)(\(([^()]*)\))?(!)?: (.*)$`)

const breakingPrefix = "BREAKING CHANGE: "

// ParseCommit parses the conventional commit message of one input commit.
// Errors are located to the specific commit id.
func ParseCommit(c CommitInput) (Commit, error) {
	out := Commit{ID: c.ID, Packages: c.Packages}
	first, rest, _ := strings.Cut(c.Message, "\n")
	m := headerRe.FindStringSubmatch(first)
	if m == nil {
		return out, fmt.Errorf("commit %q: malformed header %q: expected \"type[(scope)][!]: description\"", c.ID, first)
	}
	if m[5] == "" {
		return out, fmt.Errorf("commit %q: description after \": \" must not be empty", c.ID)
	}
	if m[2] != "" && m[3] == "" {
		return out, fmt.Errorf("commit %q: scope must not be empty", c.ID)
	}
	out.Type = m[1]
	out.Scope = m[3]
	out.Description = m[5]
	out.Breaking = m[4] == "!"
	// Body: a standalone line starting with "BREAKING CHANGE: " and a
	// non-empty note also marks the commit as breaking.
	for _, line := range strings.Split(rest, "\n") {
		if note, ok := strings.CutPrefix(line, breakingPrefix); ok && strings.TrimSpace(note) != "" {
			out.Breaking = true
		}
	}
	return out, nil
}
