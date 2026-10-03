package planner

import (
	"fmt"
	"strings"
)

var levelTitles = []struct {
	level, title string
}{
	{"major", "Breaking Changes"},
	{"minor", "Features"},
	{"patch", "Bug Fixes"},
}

// RenderMarkdown renders the plan as a Markdown changelog. Packages appear
// in plan order (sorted by name); commits are grouped by level and keep
// their original order within each group.
func RenderMarkdown(plan *Plan) string {
	var b strings.Builder
	b.WriteString("# Release Plan\n")
	for _, p := range plan.Packages {
		fmt.Fprintf(&b, "\n## %s %s -> %s\n", p.Name, p.CurrentVersion, p.NewVersion)
		if p.Level == "none" {
			b.WriteString("\nNo changes.\n")
			continue
		}
		for _, lt := range levelTitles {
			var group []CommitRef
			for _, c := range p.Commits {
				if c.Level == lt.level {
					group = append(group, c)
				}
			}
			if len(group) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\n### %s\n\n", lt.title)
			for _, c := range group {
				scope := ""
				if c.Scope != "" {
					scope = "(" + c.Scope + ")"
				}
				breaking := ""
				if c.Breaking {
					breaking = " BREAKING"
				}
				fmt.Fprintf(&b, "- %s%s: %s (%s)%s\n", c.Type, scope, c.Description, c.ID, breaking)
			}
		}
	}
	return b.String()
}
