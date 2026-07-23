// Package config loads, validates, and writes pickle.toml — the overarching
// project configuration and the registry of connected child-projects.
//
// TOML is decoded with github.com/BurntSushi/toml (a build-time dependency
// compiled into the static binary; nothing is fetched at runtime). Writing goes
// through a canonical renderer (Render) so the file layout is deterministic and
// tool-managed: hand-edits are preserved on load but normalised to the canonical
// layout on the next mutation (project add/remove).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the fixed config filename, located at the overarching project root.
const FileName = "pickle.toml"

// Per-child defaults applied when a field is omitted.
const (
	DefaultBranchPrefix     = "feat/"
	DefaultWIPInDevelopment = 1
	DefaultWIPInReview      = 1
)

// Config is the whole pickle.toml.
type Config struct {
	PayloadVersion string       `toml:"payload_version"`
	ReviewAddendum string       `toml:"review_addendum,omitempty"`
	Commit         CommitPolicy `toml:"commit"`
	Projects       []Project    `toml:"project"`

	path string // where this config was loaded from (not serialised)
}

// CommitPolicy is the overarching commit policy.
type CommitPolicy struct {
	OverarchingAuto   bool `toml:"overarching_auto"`
	ChildPublishGated bool `toml:"child_publish_gated"`
}

// Project is one registered child-project.
type Project struct {
	Name             string `toml:"name"`
	Path             string `toml:"path"`
	Build            string `toml:"build,omitempty"`
	Test             string `toml:"test,omitempty"`
	Lint             string `toml:"lint,omitempty"`
	Docs             string `toml:"docs,omitempty"`
	BranchPrefix     string `toml:"branch_prefix"`
	WIPInDevelopment int    `toml:"wip_in_development"`
	WIPInReview      int    `toml:"wip_in_review"`
	ReviewAddendum   string `toml:"review_addendum,omitempty"`
}

// Find returns the path to the nearest pickle.toml at or above startDir.
func Find(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, FileName)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or any parent directory", FileName, startDir)
		}
		dir = parent
	}
}

// Load decodes and validates the config at path.
func Load(path string) (*Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.path = path
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Root is the directory the config lives in (the overarching project root); child
// project paths are resolved relative to it.
func (c *Config) Root() string {
	if c.path == "" {
		return ""
	}
	return filepath.Dir(c.path)
}

// Path is where the config was loaded from.
func (c *Config) Path() string { return c.path }

func (c *Config) applyDefaults() {
	for i := range c.Projects {
		p := &c.Projects[i]
		if p.BranchPrefix == "" {
			p.BranchPrefix = DefaultBranchPrefix
		}
		if p.WIPInDevelopment == 0 {
			p.WIPInDevelopment = DefaultWIPInDevelopment
		}
		if p.WIPInReview == 0 {
			p.WIPInReview = DefaultWIPInReview
		}
	}
}

// Validate checks the invariants (unique non-empty names, resolvable paths, WIP >= 1).
func (c *Config) Validate() error {
	if len(c.Projects) == 0 {
		return errors.New("pickle.toml: at least one [[project]] (child-project) is required")
	}
	seen := make(map[string]bool, len(c.Projects))
	root := c.Root()
	for i := range c.Projects {
		p := &c.Projects[i]
		switch {
		case strings.TrimSpace(p.Name) == "":
			return fmt.Errorf("pickle.toml: [[project]] #%d has an empty name", i+1)
		case seen[p.Name]:
			return fmt.Errorf("pickle.toml: duplicate project name %q", p.Name)
		case strings.TrimSpace(p.Path) == "":
			return fmt.Errorf("pickle.toml: project %q has an empty path", p.Name)
		case p.WIPInDevelopment < 1 || p.WIPInReview < 1:
			return fmt.Errorf("pickle.toml: project %q WIP limits must be >= 1", p.Name)
		}
		seen[p.Name] = true
		if root != "" {
			abs := filepath.Join(root, p.Path)
			if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
				return fmt.Errorf("pickle.toml: project %q path %q does not resolve to a directory", p.Name, p.Path)
			}
		}
	}
	return nil
}

// Project returns the registered child by name.
func (c *Config) Project(name string) (*Project, bool) {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			return &c.Projects[i], true
		}
	}
	return nil, false
}

// AddProject appends a child after applying defaults; errors on duplicate/empty name.
func (c *Config) AddProject(p Project) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("project name is required")
	}
	if _, ok := c.Project(p.Name); ok {
		return fmt.Errorf("project %q is already registered", p.Name)
	}
	if p.BranchPrefix == "" {
		p.BranchPrefix = DefaultBranchPrefix
	}
	if p.WIPInDevelopment == 0 {
		p.WIPInDevelopment = DefaultWIPInDevelopment
	}
	if p.WIPInReview == 0 {
		p.WIPInReview = DefaultWIPInReview
	}
	c.Projects = append(c.Projects, p)
	return nil
}

// RemoveProject drops a child by name; errors if it is not registered.
func (c *Config) RemoveProject(name string) error {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("project %q is not registered", name)
}

// Render produces the canonical pickle.toml text.
func (c *Config) Render() string {
	var b strings.Builder
	b.WriteString("# pickle configuration for this overarching project.\n")
	b.WriteString("# Managed by `pickle`. Hand-edits are preserved on load but normalised to this\n")
	b.WriteString("# layout on the next `pickle project add|remove`.\n\n")
	fmt.Fprintf(&b, "payload_version = %q\n", c.PayloadVersion)
	if c.ReviewAddendum != "" {
		fmt.Fprintf(&b, "review_addendum = %q\n", c.ReviewAddendum)
	}
	b.WriteString("\n[commit]\n")
	fmt.Fprintf(&b, "overarching_auto = %t\n", c.Commit.OverarchingAuto)
	fmt.Fprintf(&b, "child_publish_gated = %t\n", c.Commit.ChildPublishGated)
	for _, p := range c.Projects {
		b.WriteString("\n[[project]]\n")
		fmt.Fprintf(&b, "name = %q\n", p.Name)
		fmt.Fprintf(&b, "path = %q\n", p.Path)
		for _, kv := range []struct {
			k, v string
		}{{"build", p.Build}, {"test", p.Test}, {"lint", p.Lint}, {"docs", p.Docs}} {
			if kv.v != "" {
				fmt.Fprintf(&b, "%s = %q\n", kv.k, kv.v)
			}
		}
		fmt.Fprintf(&b, "branch_prefix = %q\n", p.BranchPrefix)
		fmt.Fprintf(&b, "wip_in_development = %d\n", p.WIPInDevelopment)
		fmt.Fprintf(&b, "wip_in_review = %d\n", p.WIPInReview)
		if p.ReviewAddendum != "" {
			fmt.Fprintf(&b, "review_addendum = %q\n", p.ReviewAddendum)
		}
	}
	return b.String()
}

// Save writes the canonical render to path (or the loaded path if empty).
func (c *Config) Save(path string) error {
	if path == "" {
		path = c.path
	}
	if path == "" {
		return errors.New("no path to save config to")
	}
	return os.WriteFile(path, []byte(c.Render()), 0o644)
}
