package main

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders the plan as a changelog document. Packages are
// already sorted; groups are ordered major, minor, patch with commits in
// their original order.
func RenderMarkdown(plan *Plan) string {
	var b strings.Builder
	b.WriteString("# Release Plan\n")
	for _, p := range plan.Packages {
		fmt.Fprintf(&b, "\n## %s %s -> %s (%s)\n", p.Name, p.CurrentVersion, p.NewVersion, p.Bump)
		if len(p.Groups) == 0 {
			b.WriteString("\nNo changes.\n")
			continue
		}
		for _, g := range p.Groups {
			fmt.Fprintf(&b, "\n### %s\n", groupTitle(g.Level))
			for _, ref := range g.Commits {
				c := ref.Commit
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

func groupTitle(level string) string {
	switch level {
	case "major":
		return "Breaking Changes"
	case "minor":
		return "Features"
	case "patch":
		return "Bug Fixes"
	default:
		return "Other"
	}
}
