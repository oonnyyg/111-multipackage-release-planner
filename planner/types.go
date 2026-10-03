package planner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
)

// Input is the top-level JSON document consumed by the planner.
type Input struct {
	Packages []Package `json:"packages"`
	Commits  []Commit  `json:"commits"`
}

// Package describes one releasable package and its current version.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Commit is a single conventional commit with its explicit package set.
type Commit struct {
	ID       string   `json:"id"`
	Message  string   `json:"message"`
	Packages []string `json:"packages"`
}

// Version is a three-segment semantic version without leading zeros.
type Version struct {
	Major, Minor, Patch int
}

var versionRe = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ParseVersion validates and parses a "major.minor.patch" version string.
func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version %q: must be three non-negative integers without leading zeros", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ValidateInput checks the whole batch: package names and versions must be
// valid, commit IDs unique, and referenced packages must exist. Any problem
// rejects the entire input.
func ValidateInput(in *Input) error {
	seen := map[string]bool{}
	for _, p := range in.Packages {
		if p.Name == "" {
			return fmt.Errorf("package name must not be empty")
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate package %q", p.Name)
		}
		seen[p.Name] = true
		if _, err := ParseVersion(p.Version); err != nil {
			return fmt.Errorf("package %q: %w", p.Name, err)
		}
	}
	ids := map[string]bool{}
	for _, c := range in.Commits {
		if c.ID == "" {
			return fmt.Errorf("commit id must not be empty")
		}
		if ids[c.ID] {
			return fmt.Errorf("duplicate commit id %q", c.ID)
		}
		ids[c.ID] = true
		if len(c.Packages) == 0 {
			return fmt.Errorf("commit %q: must affect at least one package", c.ID)
		}
		for _, name := range c.Packages {
			if !seen[name] {
				return fmt.Errorf("commit %q: unknown package %q", c.ID, name)
			}
		}
	}
	return nil
}

// DecodeInput parses JSON input and validates it as a whole batch.
func DecodeInput(data []byte) (*Input, error) {
	var in Input
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	if err := ValidateInput(&in); err != nil {
		return nil, err
	}
	return &in, nil
}
