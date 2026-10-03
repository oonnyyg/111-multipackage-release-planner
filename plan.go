package main

import "sort"

// Level is a bump level: 0 = none, 1 = patch, 2 = minor, 3 = major.
type Level int

const (
	LevelNone Level = iota
	LevelPatch
	LevelMinor
	LevelMajor
)

func (l Level) String() string {
	switch l {
	case LevelPatch:
		return "patch"
	case LevelMinor:
		return "minor"
	case LevelMajor:
		return "major"
	default:
		return "none"
	}
}

// bumpLevel maps a parsed commit to the bump it implies.
func bumpLevel(c Commit) Level {
	switch {
	case c.Breaking:
		return LevelMajor
	case c.Type == "feat":
		return LevelMinor
	case c.Type == "fix":
		return LevelPatch
	default:
		return LevelNone
	}
}

// CommitRef is a commit contributing to a package's bump, in input order.
type CommitRef struct {
	Commit Commit `json:"commit"`
	Level  string `json:"level"`
}

// Group holds commits of one bump level for a package.
type Group struct {
	Level   string      `json:"level"`
	Commits []CommitRef `json:"commits"`
}

// PackagePlan is the release plan for one package.
type PackagePlan struct {
	Name           string  `json:"name"`
	CurrentVersion string  `json:"currentVersion"`
	NewVersion     string  `json:"newVersion"`
	Bump           string  `json:"bump"`
	Groups         []Group `json:"groups,omitempty"`
}

// Plan is the full release plan, packages sorted by name.
type Plan struct {
	Packages []PackagePlan `json:"packages"`
}

// BuildPlan computes the release plan for the batch.
func BuildPlan(in *Input) (*Plan, error) {
	commits := make([]Commit, 0, len(in.Commits))
	for _, ci := range in.Commits {
		c, err := ParseCommit(ci)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}

	type pkgState struct {
		level Level
		refs  []CommitRef
		seen  map[string]bool
	}
	states := make(map[string]*pkgState, len(in.Packages))
	for _, p := range in.Packages {
		states[p.Name] = &pkgState{seen: map[string]bool{}}
	}
	for _, c := range commits {
		lvl := bumpLevel(c)
		if lvl == LevelNone {
			continue
		}
		for _, name := range c.Packages {
			st := states[name]
			if lvl > st.level {
				st.level = lvl
			}
			if !st.seen[c.ID] {
				st.seen[c.ID] = true
				st.refs = append(st.refs, CommitRef{Commit: c, Level: lvl.String()})
			}
		}
	}

	names := make([]string, 0, len(in.Packages))
	for _, p := range in.Packages {
		names = append(names, p.Name)
	}
	sort.Strings(names)

	plan := &Plan{Packages: make([]PackagePlan, 0, len(names))}
	for _, name := range names {
		st := states[name]
		cur, _ := ParseVersion(versionOf(in, name))
		next := bump(cur, st.level)
		pp := PackagePlan{
			Name:           name,
			CurrentVersion: cur.String(),
			NewVersion:     next.String(),
			Bump:           st.level.String(),
		}
		for _, lvl := range []Level{LevelMajor, LevelMinor, LevelPatch} {
			var g *Group
			for _, ref := range st.refs {
				if ref.Level != lvl.String() {
					continue
				}
				if g == nil {
					pp.Groups = append(pp.Groups, Group{Level: lvl.String()})
					g = &pp.Groups[len(pp.Groups)-1]
				}
				g.Commits = append(g.Commits, ref)
			}
		}
		plan.Packages = append(plan.Packages, pp)
	}
	return plan, nil
}

func versionOf(in *Input, name string) string {
	for _, p := range in.Packages {
		if p.Name == name {
			return p.Version
		}
	}
	return ""
}

// bump applies the level, zeroing lower parts. Zero major versions follow
// the same rules (a breaking change on 0.x.y yields 1.0.0).
func bump(v Version, lvl Level) Version {
	switch lvl {
	case LevelMajor:
		return Version{Major: v.Major + 1}
	case LevelMinor:
		return Version{Major: v.Major, Minor: v.Minor + 1}
	case LevelPatch:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v
	}
}
