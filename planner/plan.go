package planner

import "sort"

// Level is a version bump level: 0 none, 1 patch, 2 minor, 3 major.
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

// bumpLevel maps a parsed commit to its bump level.
func bumpLevel(pc ParsedCommit) Level {
	switch {
	case pc.Breaking:
		return LevelMajor
	case pc.Type == "feat":
		return LevelMinor
	case pc.Type == "fix":
		return LevelPatch
	default:
		return LevelNone
	}
}

// applyBump raises v by the given level, zeroing lower segments.
func applyBump(v Version, l Level) Version {
	switch l {
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

// CommitRef explains why a commit contributed to a package bump.
type CommitRef struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Scope       string `json:"scope,omitempty"`
	Breaking    bool   `json:"breaking,omitempty"`
	Description string `json:"description"`
	Level       string `json:"level"`
}

// PackagePlan is the release plan for one package.
type PackagePlan struct {
	Name           string      `json:"name"`
	CurrentVersion string      `json:"currentVersion"`
	NewVersion     string      `json:"newVersion"`
	Level          string      `json:"level"`
	Commits        []CommitRef `json:"commits,omitempty"`
}

// Plan is the full release plan, packages sorted by name.
type Plan struct {
	Packages []PackagePlan `json:"packages"`
}

// BuildPlan parses all commits and computes the per-package release plan.
// Commits that do not bump a version are omitted from the plan. Each commit
// appears at most once per package, at its own bump level.
func BuildPlan(in *Input) (*Plan, error) {
	if err := ValidateInput(in); err != nil {
		return nil, err
	}
	current := map[string]Version{}
	for _, p := range in.Packages {
		v, _ := ParseVersion(p.Version)
		current[p.Name] = v
	}
	levels := map[string]Level{}
	refs := map[string][]CommitRef{}
	for _, c := range in.Commits {
		pc, err := ParseCommitMessage(c.ID, c.Message)
		if err != nil {
			return nil, err
		}
		lvl := bumpLevel(pc)
		if lvl == LevelNone {
			continue
		}
		for _, name := range c.Packages {
			if lvl > levels[name] {
				levels[name] = lvl
			}
			refs[name] = append(refs[name], CommitRef{
				ID:          c.ID,
				Type:        pc.Type,
				Scope:       pc.Scope,
				Breaking:    pc.Breaking,
				Description: pc.Description,
				Level:       lvl.String(),
			})
		}
	}
	plan := &Plan{}
	for _, p := range in.Packages {
		lvl := levels[p.Name]
		pp := PackagePlan{
			Name:           p.Name,
			CurrentVersion: p.Version,
			NewVersion:     applyBump(current[p.Name], lvl).String(),
			Level:          lvl.String(),
		}
		// Group commits by level (major, minor, patch) while preserving the
		// original commit order inside each group.
		pp.Commits = groupByLevel(refs[p.Name])
		plan.Packages = append(plan.Packages, pp)
	}
	sort.Slice(plan.Packages, func(i, j int) bool {
		return plan.Packages[i].Name < plan.Packages[j].Name
	})
	return plan, nil
}

func groupByLevel(refs []CommitRef) []CommitRef {
	rank := map[string]int{"major": 0, "minor": 1, "patch": 2}
	out := make([]CommitRef, len(refs))
	copy(out, refs)
	sort.SliceStable(out, func(i, j int) bool {
		return rank[out[i].Level] < rank[out[j].Level]
	})
	return out
}
