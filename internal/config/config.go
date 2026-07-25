// Package config loads, validates, and writes pickle.toml — the overarching
// project configuration and the registry of connected child-projects.
//
// TOML is decoded with github.com/BurntSushi/toml (a build-time dependency
// compiled into the static binary; nothing is fetched at runtime).
//
// There are two ways the file is written, and the difference is what a user's
// comments depend on:
//
//   - Render/Save produce the canonical layout, discarding comments and any
//     hand-tuned spacing. Used by `pickle project add|remove`, which change the
//     file's structure, and by `pickle install`, which only ever writes the file
//     when it does not yet exist.
//   - SetPayloadVersionInPlace rewrites a single line and leaves every other
//     byte alone. Used by `pickle upgrade`, so refreshing the payload never
//     costs a user their comments.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// Overarching commit-policy defaults, applied when the key is absent. Both are
// the cautious choice, and both match what `pickle install` writes: a child is
// not pushed without approval, and only the overarching project's own
// bookkeeping is committed automatically.
const (
	DefaultOverarchingAuto   = true
	DefaultChildPublishGated = true
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
	md, err := toml.DecodeFile(path, &c)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.path = path
	c.applyDefaults(md)
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

// applyDefaults fills in omitted fields. It needs the decoder's metadata
// because the commit policy is expressed as booleans: an absent key and an
// explicit `false` both decode to false, yet they must mean opposite things —
// absent has to fall back to the cautious default, while an explicit `false`
// has to survive. Only the decoder knows which one it saw.
func (c *Config) applyDefaults(md toml.MetaData) {
	if !md.IsDefined("commit", "overarching_auto") {
		c.Commit.OverarchingAuto = DefaultOverarchingAuto
	}
	if !md.IsDefined("commit", "child_publish_gated") {
		c.Commit.ChildPublishGated = DefaultChildPublishGated
	}
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
	b.WriteString("# Hand-edits are preserved on load. `pickle upgrade` rewrites only the\n")
	b.WriteString("# payload_version line: comments and every other line survive, while that one\n")
	b.WriteString("# line comes back as payload_version = \"value\" (any inline comment kept), so\n")
	b.WriteString("# its own alignment and quoting style do not. `pickle project add|remove`\n")
	b.WriteString("# re-render this file to the canonical layout below and drop comments entirely.\n\n")
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
//
// This drops comments and any hand-tuned layout, so it is only for the commands
// that change the file's structure (project add|remove) and for creating the
// file from scratch. To stamp a new payload_version onto an existing file, use
// SetPayloadVersionInPlace instead.
func (c *Config) Save(path string) error {
	if path == "" {
		path = c.path
	}
	if path == "" {
		return errors.New("no path to save config to")
	}
	return os.WriteFile(path, []byte(c.Render()), 0o644)
}

const payloadVersionKey = "payload_version"

// bom is the UTF-8 byte-order mark, which some editors put at the head of a
// file. It is not part of any line's content and has to be held aside before
// the text is scanned, or it hides the key on the first line.
const bom = "\ufeff"

// SetPayloadVersionInPlace sets payload_version in the file at path, touching
// exactly one line: comments, blank lines, key order and every other line's
// spacing survive verbatim. The key is rewritten where it already exists and
// inserted before the first table header where it does not.
//
// The rewritten line itself is normalised to `payload_version = "value"`, so
// hand-tuned alignment on that one line is lost and a 'literal' value becomes a
// "basic" one; any inline comment after the value is kept.
//
// The transformation is line-based and therefore cannot see every TOML shape
// (multi-line strings, quoted keys). Rather than trust it, the result is parsed
// and compared against the original before anything is written: if the edit
// would fail to parse, would not set the version, or would change any other
// value, the file is left untouched and an error is returned. So the failure
// mode is a loud refusal, never a silent loss of hand-written content.
//
// This is the counterpart to Save for `pickle upgrade`, whose only job is to
// stamp a new version onto a file the user owns.
func SetPayloadVersionInPlace(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := setPayloadVersion(string(data), version)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if updated == string(data) {
		return nil
	}
	return writePreservingMode(path, []byte(updated))
}

// setPayloadVersion is the pure text transformation behind
// SetPayloadVersionInPlace, including the parse-back check that guards it.
func setPayloadVersion(text, version string) (string, error) {
	mark := ""
	if strings.HasPrefix(text, bom) {
		mark, text = bom, strings.TrimPrefix(text, bom)
	}
	updated, err := replacePayloadVersionLine(text, version)
	if err != nil {
		return "", err
	}
	if err := verifyOnlyPayloadVersion(text, updated, version); err != nil {
		return "", err
	}
	return mark + updated, nil
}

// verifyOnlyPayloadVersion is the safety gate: it decodes both texts and
// insists the edit did exactly what it claims — set payload_version to version,
// and move nothing else. Anything the line scanner mis-read shows up here as a
// parse error, a missing version, or a changed value, and turns into a refusal.
//
// Decoding into a map rather than into Config is deliberate: keys pickle knows
// nothing about are still the user's content, and a mis-read line could corrupt
// them just as easily. That whole-tree comparison is defence-in-depth, not a
// load-bearing check: no input is known to reach it, because every shape that
// would move another value fails the parse or the version check first. It is
// here so that a future change to the scanner cannot quietly start doing so.
//
// Comments are not decoded and so are not covered here; the transformation only
// ever rewrites one line or inserts one, which is what makes that gap tolerable.
func verifyOnlyPayloadVersion(before, after, version string) error {
	b, err := decodeTree(before)
	if err != nil {
		return fmt.Errorf("does not parse, so it cannot be edited safely: %w", err)
	}
	a, err := decodeTree(after)
	if err != nil {
		return fmt.Errorf("setting %s would leave the file unparseable (%w); set it by hand",
			payloadVersionKey, err)
	}
	if got, _ := a[payloadVersionKey].(string); got != version {
		return fmt.Errorf("could not set %s (it would end up %q, not %q); set it by hand",
			payloadVersionKey, got, version)
	}
	delete(b, payloadVersionKey)
	delete(a, payloadVersionKey)
	if !reflect.DeepEqual(b, a) {
		return fmt.Errorf("setting %s would change other values in the file; set it by hand",
			payloadVersionKey)
	}
	return nil
}

func decodeTree(text string) (map[string]any, error) {
	var m map[string]any
	_, err := toml.Decode(text, &m)
	return m, err
}

// replacePayloadVersionLine performs the line edit. It is deliberately naive —
// verifyOnlyPayloadVersion is what makes it safe.
func replacePayloadVersionLine(text, version string) (string, error) {
	lines := strings.Split(text, "\n")
	insertAt := len(lines)
	for i, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		body := line[indent:]
		if body == "" || strings.HasPrefix(body, "#") {
			continue
		}
		if strings.HasPrefix(body, "[") {
			// A table header ends the top-level scope: a payload_version
			// below this point belongs to [commit] or a [[project]].
			insertAt = i
			break
		}
		rest, ok := strings.CutPrefix(body, payloadVersionKey)
		if !ok {
			continue
		}
		value := strings.TrimLeft(rest, " \t")
		if !strings.HasPrefix(value, "=") {
			continue // a different key that merely starts with payload_version
		}
		start := indent + len(body) - len(value) + 1
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}
		if strings.HasPrefix(line[start:], `"""`) || strings.HasPrefix(line[start:], "'''") {
			// A multi-line string may continue past this line; rewriting it
			// blind could corrupt the file. Refuse loudly instead.
			return "", errors.New("payload_version uses a multi-line string; set it by hand")
		}
		end := valueEnd(line, start)
		lines[i] = line[:indent] + payloadVersionKey + " = " + fmt.Sprintf("%q", version) + line[end:]
		if raw != line {
			lines[i] += "\r"
		}
		return strings.Join(lines, "\n"), nil
	}

	// Not present: insert it, keeping any blank lines attached to whatever
	// follows rather than orphaning them above the new key.
	for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	entry := payloadVersionKey + " = " + fmt.Sprintf("%q", version)
	if usesCRLF(lines) {
		entry += "\r" // match the file rather than leaving one lone LF line
	}
	lines = append(lines[:insertAt], append([]string{entry}, lines[insertAt:]...)...)
	return strings.Join(lines, "\n"), nil
}

// usesCRLF reports whether every terminated line ends with CR, i.e. the file is
// consistently CRLF. A mixed file gets a plain LF and is left as it was found.
func usesCRLF(lines []string) bool {
	if len(lines) < 2 {
		return false
	}
	for _, l := range lines[:len(lines)-1] {
		if !strings.HasSuffix(l, "\r") {
			return false
		}
	}
	return true
}

// valueEnd returns the index just past the TOML value starting at i in line, so
// that any trailing inline comment can be preserved verbatim.
func valueEnd(line string, i int) int {
	if i >= len(line) {
		return len(line)
	}
	switch line[i] {
	case '"':
		for j := i + 1; j < len(line); j++ {
			if line[j] == '\\' {
				j++
			} else if line[j] == '"' {
				return j + 1
			}
		}
	case '\'':
		if k := strings.IndexByte(line[i+1:], '\''); k >= 0 {
			return i + 1 + k + 1
		}
	default:
		if j := strings.IndexAny(line[i:], " \t#"); j >= 0 {
			return i + j
		}
	}
	return len(line)
}

// writePreservingMode replaces path atomically, keeping its current permissions.
//
// A symlinked config is followed rather than replaced: renaming onto the link
// would turn it into a regular file and leave the real target stale, which is
// the opposite of preserving the user's file.
func writePreservingMode(path string, data []byte) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }() // no-op once the rename succeeds
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return os.Rename(name, path)
}
