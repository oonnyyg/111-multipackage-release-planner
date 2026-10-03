package main

import (
	"strings"
	"testing"
)

func buildPlan(t *testing.T, in string) *Plan {
	t.Helper()
	input, err := ReadInput(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ReadInput: %v", err)
	}
	plan, err := BuildPlan(input)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	return plan
}

func pkg(t *testing.T, p *Plan, name string) PackagePlan {
	t.Helper()
	for _, pp := range p.Packages {
		if pp.Name == name {
			return pp
		}
	}
	t.Fatalf("package %q not in plan", name)
	return PackagePlan{}
}

func TestCrossPackageCommitAndHighestLevel(t *testing.T) {
	plan := buildPlan(t, `{
	  "packages": [
	    {"name": "web", "version": "1.2.3"},
	    {"name": "core", "version": "0.9.0"},
	    {"name": "docs", "version": "2.0.0"}
	  ],
	  "commits": [
	    {"id": "c1", "message": "fix(core): patch bug", "packages": ["core", "web"]},
	    {"id": "c2", "message": "feat: add feature", "packages": ["core"]},
	    {"id": "c3", "message": "chore: tidy up", "packages": ["web"]}
	  ]
	}`)
	// Packages sorted by name.
	names := []string{plan.Packages[0].Name, plan.Packages[1].Name, plan.Packages[2].Name}
	if names[0] != "core" || names[1] != "docs" || names[2] != "web" {
		t.Fatalf("packages not sorted: %v", names)
	}
	core := pkg(t, plan, "core")
	if core.NewVersion != "0.10.0" || core.Bump != "minor" {
		t.Fatalf("core: got %s (%s)", core.NewVersion, core.Bump)
	}
	// Highest level wins; both commits listed, fix before feat (input order
	// within groups), grouped minor before patch.
	if len(core.Groups) != 2 || core.Groups[0].Level != "minor" || core.Groups[1].Level != "patch" {
		t.Fatalf("core groups: %+v", core.Groups)
	}
	web := pkg(t, plan, "web")
	if web.NewVersion != "1.2.4" || web.Bump != "patch" {
		t.Fatalf("web: got %s (%s)", web.NewVersion, web.Bump)
	}
	// Cross-package commit c1 appears once in web; chore contributes nothing.
	if len(web.Groups) != 1 || len(web.Groups[0].Commits) != 1 || web.Groups[0].Commits[0].Commit.ID != "c1" {
		t.Fatalf("web groups: %+v", web.Groups)
	}
	docs := pkg(t, plan, "docs")
	if docs.NewVersion != "2.0.0" || docs.Bump != "none" || len(docs.Groups) != 0 {
		t.Fatalf("docs should be unchanged: %+v", docs)
	}
}

func TestBreakingMarkers(t *testing.T) {
	plan := buildPlan(t, `{
	  "packages": [{"name": "a", "version": "1.0.0"}],
	  "commits": [
	    {"id": "bang", "message": "feat!: drop old API", "packages": ["a"]},
	    {"id": "footer", "message": "fix: tweak\n\nBREAKING CHANGE: config renamed", "packages": ["a"]},
	    {"id": "other", "message": "refactor!: internal rework", "packages": ["a"]}
	  ]
	}`)
	a := pkg(t, plan, "a")
	if a.NewVersion != "2.0.0" || a.Bump != "major" {
		t.Fatalf("got %s (%s)", a.NewVersion, a.Bump)
	}
	if len(a.Groups) != 1 || a.Groups[0].Level != "major" || len(a.Groups[0].Commits) != 3 {
		t.Fatalf("groups: %+v", a.Groups)
	}
}

func TestQuotedBreakingWordIsNotBreaking(t *testing.T) {
	plan := buildPlan(t, `{
	  "packages": [{"name": "a", "version": "1.0.0"}],
	  "commits": [
	    {"id": "q", "message": "fix: mention BREAKING CHANGE: in docs\n\nSee the BREAKING CHANGE: section of the guide.\nBREAKING CHANGE:", "packages": ["a"]}
	  ]
	}`)
	a := pkg(t, plan, "a")
	if a.Bump != "patch" || a.NewVersion != "1.0.1" {
		t.Fatalf("quoted or empty BREAKING CHANGE must not bump major: %s (%s)", a.NewVersion, a.Bump)
	}
}

func TestVersionCarryOver(t *testing.T) {
	cases := []struct {
		version, msg, want string
	}{
		{"1.9.9", "feat: x", "1.10.0"},
		{"9.9.9", "fix!: x", "10.0.0"},
		{"0.1.2", "feat!: x", "1.0.0"},
		{"0.0.0", "fix: x", "0.0.1"},
		{"2.3.4", "feat: x", "2.4.0"},
	}
	for _, tc := range cases {
		plan := buildPlan(t, `{"packages":[{"name":"a","version":"`+tc.version+`"}],
		  "commits":[{"id":"c","message":`+jsonString(tc.msg)+`,"packages":["a"]}]}`)
		if got := pkg(t, plan, "a").NewVersion; got != tc.want {
			t.Errorf("%s + %q: got %s, want %s", tc.version, tc.msg, got, tc.want)
		}
	}
}

func jsonString(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func TestInvalidInputsRejected(t *testing.T) {
	cases := []struct {
		name, in, wantErr string
	}{
		{"unknown package", `{"packages":[{"name":"a","version":"1.0.0"}],"commits":[{"id":"c","message":"fix: x","packages":["b"]}]}`, "unknown package"},
		{"duplicate commit", `{"packages":[{"name":"a","version":"1.0.0"}],"commits":[{"id":"c","message":"fix: x","packages":["a"]},{"id":"c","message":"fix: y","packages":["a"]}]}`, "duplicate commit id"},
		{"leading zero version", `{"packages":[{"name":"a","version":"01.0.0"}],"commits":[]}`, "invalid version"},
		{"two-part version", `{"packages":[{"name":"a","version":"1.0"}],"commits":[]}`, "invalid version"},
		{"duplicate package", `{"packages":[{"name":"a","version":"1.0.0"},{"name":"a","version":"1.0.0"}],"commits":[]}`, "duplicate package"},
	}
	for _, tc := range cases {
		if _, err := ReadInput(strings.NewReader(tc.in)); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: err=%v, want substring %q", tc.name, err, tc.wantErr)
		}
	}
}

func TestMalformedHeaderLocatedToCommit(t *testing.T) {
	bad := []string{
		"Feat: uppercase type",
		"feat:missing space",
		"feat: ",
		"feat(scope: unbalanced",
		"feat(): empty scope",
	}
	for _, msg := range bad {
		in := `{"packages":[{"name":"a","version":"1.0.0"}],"commits":[{"id":"bad1","message":` + jsonString(msg) + `,"packages":["a"]}]}`
		input, err := ReadInput(strings.NewReader(in))
		if err != nil {
			continue // rejected at read time is also acceptable
		}
		_, err = BuildPlan(input)
		if err == nil || !strings.Contains(err.Error(), `"bad1"`) {
			t.Errorf("message %q: err=%v, want error naming commit bad1", msg, err)
		}
	}
}

func TestScopeDoesNotChangeAffectedPackages(t *testing.T) {
	plan := buildPlan(t, `{
	  "packages": [{"name": "a", "version": "1.0.0"}, {"name": "b", "version": "1.0.0"}],
	  "commits": [{"id": "c", "message": "fix(b): scoped fix", "packages": ["a"]}]
	}`)
	if got := pkg(t, plan, "a").NewVersion; got != "1.0.1" {
		t.Fatalf("a: got %s", got)
	}
	if got := pkg(t, plan, "b").NewVersion; got != "1.0.0" {
		t.Fatalf("b must stay unchanged despite scope: got %s", got)
	}
}

func TestMarkdownRender(t *testing.T) {
	plan := buildPlan(t, `{
	  "packages": [{"name": "a", "version": "1.0.0"}],
	  "commits": [{"id": "c1", "message": "feat(ui)!: redesign", "packages": ["a"]}]
	}`)
	md := RenderMarkdown(plan)
	for _, want := range []string{"## a 1.0.0 -> 2.0.0 (major)", "### Breaking Changes", "- feat(ui): redesign (c1) BREAKING"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}
