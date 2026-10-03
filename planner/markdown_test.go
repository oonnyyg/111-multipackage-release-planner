package planner

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	in := &Input{
		Packages: []Package{
			{Name: "app", Version: "1.0.0"},
			{Name: "idle", Version: "0.1.0"},
		},
		Commits: []Commit{
			{ID: "c1", Message: "fix: typo", Packages: []string{"app"}},
			{ID: "c2", Message: "feat!: redesign", Packages: []string{"app"}},
		},
	}
	plan, err := BuildPlan(in)
	if err != nil {
		t.Fatal(err)
	}
	md := RenderMarkdown(plan)
	for _, want := range []string{
		"## app 1.0.0 -> 2.0.0",
		"### Breaking Changes",
		"- feat: redesign (c2) BREAKING",
		"### Bug Fixes",
		"- fix: typo (c1)",
		"## idle 0.1.0 -> 0.1.0",
		"No changes.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
	if strings.Index(md, "Breaking Changes") > strings.Index(md, "Bug Fixes") {
		t.Error("breaking group should come before fixes")
	}
}
