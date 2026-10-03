package planner

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for _, bad := range []string{"1.2", "1.2.3.4", "01.2.3", "1.02.3", "a.b.c", "1.2.-3", ""} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
	v, err := ParseVersion("0.20.300")
	if err != nil || v != (Version{0, 20, 300}) {
		t.Errorf("ParseVersion ok case: %v %v", v, err)
	}
}

func TestValidateInputRejects(t *testing.T) {
	base := func() *Input {
		return &Input{
			Packages: []Package{{Name: "a", Version: "1.0.0"}},
			Commits:  []Commit{{ID: "c1", Message: "fix: x", Packages: []string{"a"}}},
		}
	}
	dup := base()
	dup.Commits = append(dup.Commits, Commit{ID: "c1", Message: "fix: y", Packages: []string{"a"}})
	unknown := base()
	unknown.Commits[0].Packages = []string{"ghost"}
	badVer := base()
	badVer.Packages[0].Version = "1.01.0"
	for name, in := range map[string]*Input{"dup id": dup, "unknown pkg": unknown, "bad version": badVer} {
		if err := ValidateInput(in); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if err := ValidateInput(base()); err != nil {
		t.Errorf("valid input rejected: %v", err)
	}
}

func TestBuildPlanCrossPackageAndCarry(t *testing.T) {
	in := &Input{
		Packages: []Package{
			{Name: "web", Version: "1.9.9"},
			{Name: "core", Version: "0.2.3"},
			{Name: "docs", Version: "2.0.0"},
		},
		Commits: []Commit{
			{ID: "c1", Message: "fix: shared patch", Packages: []string{"web", "core"}},
			{ID: "c2", Message: "feat(api): new endpoint", Packages: []string{"web"}},
			{ID: "c3", Message: "chore: bump deps", Packages: []string{"core"}},
			{ID: "c4", Message: "refactor!: rewrite engine", Packages: []string{"core"}},
		},
	}
	plan, err := BuildPlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(plan.Packages))
	}
	// Sorted by name: core, docs, web.
	core, docs, web := plan.Packages[0], plan.Packages[1], plan.Packages[2]
	if core.Name != "core" || docs.Name != "docs" || web.Name != "web" {
		t.Fatalf("packages not sorted: %v", plan.Packages)
	}
	// Zero major follows the same major-bump rule; lower segments reset.
	if core.NewVersion != "1.0.0" || core.Level != "major" {
		t.Errorf("core = %+v", core)
	}
	// Minor bump carries: 1.9.9 -> 1.10.0.
	if web.NewVersion != "1.10.0" || web.Level != "minor" {
		t.Errorf("web = %+v", web)
	}
	// Unchanged package keeps its version.
	if docs.NewVersion != "2.0.0" || docs.Level != "none" || len(docs.Commits) != 0 {
		t.Errorf("docs = %+v", docs)
	}
	// Cross-package commit c1 appears once in each affected package.
	if countID(core.Commits, "c1") != 1 || countID(web.Commits, "c1") != 1 {
		t.Errorf("c1 should appear once per affected package")
	}
	// Non-bumping chore commit is omitted.
	if countID(core.Commits, "c3") != 0 {
		t.Errorf("chore commit should not appear in plan")
	}
	// Core commits grouped by level: major (c4) before patch (c1).
	if core.Commits[0].ID != "c4" || core.Commits[1].ID != "c1" {
		t.Errorf("core commits not grouped by level: %+v", core.Commits)
	}
	// Scope is recorded but does not change the affected packages.
	if web.Commits[0].ID != "c2" || web.Commits[0].Scope != "api" {
		t.Errorf("web commits = %+v", web.Commits)
	}
}

func TestBuildPlanBreakingFooterOnOtherType(t *testing.T) {
	in := &Input{
		Packages: []Package{{Name: "lib", Version: "3.4.5"}},
		Commits: []Commit{
			{ID: "c1", Message: "perf: faster\n\nBREAKING CHANGE: new API", Packages: []string{"lib"}},
		},
	}
	plan, err := BuildPlan(in)
	if err != nil {
		t.Fatal(err)
	}
	p := plan.Packages[0]
	if p.NewVersion != "4.0.0" || p.Level != "major" {
		t.Errorf("breaking footer on perf should bump major: %+v", p)
	}
}

func TestBuildPlanRejectsMalformedCommit(t *testing.T) {
	in := &Input{
		Packages: []Package{{Name: "lib", Version: "1.0.0"}},
		Commits:  []Commit{{ID: "bad1", Message: "not conventional", Packages: []string{"lib"}}},
	}
	_, err := BuildPlan(in)
	if err == nil || !strings.Contains(err.Error(), "bad1") {
		t.Errorf("expected error locating commit bad1, got %v", err)
	}
}

func countID(refs []CommitRef, id string) int {
	n := 0
	for _, r := range refs {
		if r.ID == id {
			n++
		}
	}
	return n
}
