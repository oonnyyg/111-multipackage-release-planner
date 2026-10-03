package planner

import (
	"fmt"
	"regexp"
	"strings"
)

// ParsedCommit is the structured result of parsing a commit message.
type ParsedCommit struct {
	Type        string
	Scope       string
	HasScope    bool
	Breaking    bool
	Description string
}

// headerRe matches "type", optional "(scope)", optional "!", then ": desc".
var headerRe = regexp.MustCompile(`^([a-z]+)(\(([^()]*)\))?(!)?: (.*)$`)

const breakingPrefix = "BREAKING CHANGE:"

// ParseCommitMessage parses a conventional commit message. The first line
// must be "type[(scope)][!]: description" with a lowercase type, a space
// after the colon, and a non-empty description. A "!" marker or a body line
// starting with "BREAKING CHANGE: " (followed by a non-empty note) marks a
// breaking change.
func ParseCommitMessage(commitID, message string) (ParsedCommit, error) {
	var pc ParsedCommit
	lines := strings.Split(message, "\n")
	header := strings.TrimRight(lines[0], "\r")
	m := headerRe.FindStringSubmatch(header)
	if m == nil {
		return pc, fmt.Errorf("commit %q: malformed header %q: expected \"type[(scope)][!]: description\"", commitID, header)
	}
	pc.Type = m[1]
	pc.HasScope = m[2] != ""
	pc.Scope = m[3]
	pc.Breaking = m[4] == "!"
	pc.Description = m[5]
	if strings.TrimSpace(pc.Description) == "" {
		return pc, fmt.Errorf("commit %q: description after colon must not be empty", commitID)
	}
	for _, line := range lines[1:] {
		line = strings.TrimRight(line, "\r")
		if rest, ok := strings.CutPrefix(line, breakingPrefix); ok && (rest == "" || strings.HasPrefix(rest, " ")) {
			if strings.TrimSpace(rest) == "" {
				return pc, fmt.Errorf("commit %q: BREAKING CHANGE note must not be empty", commitID)
			}
			pc.Breaking = true
		}
	}
	return pc, nil
}
