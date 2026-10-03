package main

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

// Version is a three-part non-negative semantic version without leading zeros.
type Version struct {
	Major, Minor, Patch int
}

var versionRe = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version %q: must be three non-negative integers without leading zeros", s)
	}
	var v Version
	fmt.Sscanf(m[1], "%d", &v.Major)
	fmt.Sscanf(m[2], "%d", &v.Minor)
	fmt.Sscanf(m[3], "%d", &v.Patch)
	return v, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// PackageInput describes one package and its current released version.
type PackageInput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CommitInput is a single commit in chronological order.
type CommitInput struct {
	ID       string   `json:"id"`
	Message  string   `json:"message"`
	Packages []string `json:"packages"`
}

// Input is the whole batch read from JSON.
type Input struct {
	Packages []PackageInput `json:"packages"`
	Commits  []CommitInput  `json:"commits"`
}

// ReadInput decodes and validates the batch. Any validation failure rejects
// the entire batch.
func ReadInput(r io.Reader) (*Input, error) {
	var in Input
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	if len(in.Packages) == 0 {
		return nil, fmt.Errorf("input must declare at least one package")
	}
	known := make(map[string]bool, len(in.Packages))
	for _, p := range in.Packages {
		if p.Name == "" {
			return nil, fmt.Errorf("package name must not be empty")
		}
		if known[p.Name] {
			return nil, fmt.Errorf("duplicate package %q", p.Name)
		}
		known[p.Name] = true
		if _, err := ParseVersion(p.Version); err != nil {
			return nil, fmt.Errorf("package %q: %w", p.Name, err)
		}
	}
	seen := make(map[string]bool, len(in.Commits))
	for _, c := range in.Commits {
		if c.ID == "" {
			return nil, fmt.Errorf("commit id must not be empty")
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("duplicate commit id %q", c.ID)
		}
		seen[c.ID] = true
		if len(c.Packages) == 0 {
			return nil, fmt.Errorf("commit %q must declare at least one affected package", c.ID)
		}
		for _, name := range c.Packages {
			if !known[name] {
				return nil, fmt.Errorf("commit %q affects unknown package %q", c.ID, name)
			}
		}
	}
	return &in, nil
}
