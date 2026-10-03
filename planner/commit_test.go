package planner

import (
	"strings"
	"testing"
)

func TestParseCommitMessageValid(t *testing.T) {
	cases := []struct {
		msg      string
		typ      string
		scope    string
		breaking bool
		desc     string
	}{
		{"feat: add login", "feat", "", false, "add login"},
		{"fix(api): handle nil", "fix", "api", false, "handle nil"},
		{"feat!: drop v1", "feat", "", true, "drop v1"},
		{"refactor(core)!: rewrite", "refactor", "core", true, "rewrite"},
		{"chore: tidy", "chore", "", false, "tidy"},
	}
	for _, c := range cases {
		pc, err := ParseCommitMessage("c1", c.msg)
		if err != nil {
			t.Fatalf("ParseCommitMessage(%q): %v", c.msg, err)
		}
		if pc.Type != c.typ || pc.Scope != c.scope || pc.Breaking != c.breaking || pc.Description != c.desc {
			t.Errorf("ParseCommitMessage(%q) = %+v", c.msg, pc)
		}
	}
}

func TestParseCommitMessageInvalid(t *testing.T) {
	for _, msg := range []string{
		"Feat: upper type",
		"feat:no space",
		"feat: ",
		"feat(scope",
		": no type",
		"feat!:",
	} {
		if _, err := ParseCommitMessage("c9", msg); err == nil {
			t.Errorf("expected error for %q", msg)
		} else if got := err.Error(); !strings.Contains(got, "c9") {
			t.Errorf("error %q should locate commit c9", got)
		}
	}
}

func TestParseBreakingChangeFooter(t *testing.T) {
	pc, err := ParseCommitMessage("c1", "fix: patch\n\nBREAKING CHANGE: drops support")
	if err != nil {
		t.Fatal(err)
	}
	if !pc.Breaking {
		t.Error("expected breaking from footer")
	}

	pc, err = ParseCommitMessage("c2", "docs: mention BREAKING CHANGE: inline")
	if err != nil {
		t.Fatal(err)
	}
	if pc.Breaking {
		t.Error("inline mention in header description is not breaking")
	}

	pc, err = ParseCommitMessage("c3", "docs: note\n\nsee BREAKING CHANGE: for details")
	if err != nil {
		t.Fatal(err)
	}
	if pc.Breaking {
		t.Error("mid-line mention in body is not breaking")
	}

	if _, err := ParseCommitMessage("c4", "fix: x\n\nBREAKING CHANGE:"); err == nil {
		t.Error("empty BREAKING CHANGE note should be rejected")
	}
}
