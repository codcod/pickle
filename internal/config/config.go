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
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/codcod/pickle/internal/atomicfile"
	"github.com/codcod/pickle/internal/vcs"
)

// ticketPrefixRE is the legal shape for a per-child ticket_prefix: an uppercase
// letter followed by up to seven more uppercase letters/digits (1–8 chars). It
// deliberately excludes the '-' that separates prefix from number in an id, so
// SplitID's last-'-' rule is unambiguous.
var ticketPrefixRE = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,7}$`)

// FileName is the fixed config filename, located at the overarching project root.
const FileName = "pickle.toml"

// Per-child defaults applied when a field is omitted.
const (
	DefaultBranchPrefix     = "feat/"
	DefaultTicketPrefix     = "T"
	DefaultWIPInDevelopment = 1
	DefaultWIPInReview      = 1
	DefaultSpecsRoot        = "docs/specs"
)

// Overarching commit-policy defaults, applied when the key is absent. Both are
// the cautious choice, and both match what `pickle install` writes: a child is
// not pushed without approval, and only the overarching project's own
// bookkeeping is committed automatically.
const (
	DefaultOverarchingAuto   = true
	DefaultChildPublishGated = true
)

// DefaultFlowName is the flow's name when the overarching pickle.toml has no
// explicit flow key. It is also, today, the only legal value: no second flow
// exists yet (see FlowName and Validate).
const DefaultFlowName = "brine"

// Layout selects where the board (tickets/ and pickle.toml) lives relative to
// its child-projects (T-108).
//
// LayoutUmbrella is the default and primary mode: the overarching project and
// its children are separate repositories, so a child's feature branches
// cannot fork the board — it is simply not in that repository.
//
// LayoutInTree is the exception: the sole child and the overarching project
// are the same repository (a root-path child, path "."). This makes tickets
// visible to anyone who clones the code, at a known price — the board is now
// inside a branching medium, so every ticket-reading command can report a
// stale copy when run from a feature branch cut before the latest
// bookkeeping commit on the base branch.
const (
	LayoutUmbrella = "umbrella"
	LayoutInTree   = "in-tree"
)

// WIPKeyInDevelopment and WIPKeyInReview name the two pickle.toml keys a WIP
// limit is configured under. They must equal Project's own `wip_in_development`
// / `wip_in_review` struct tags below — Go has no way to derive a constant
// from a struct tag, so the two are kept in sync by hand; a flow definition
// (internal/flow) names a WIP-limited state's key with one of these two
// strings, and Project.WIPLimitFor is the only place that key is resolved to
// a number.
const (
	WIPKeyInDevelopment = "wip_in_development"
	WIPKeyInReview      = "wip_in_review"
)

// Config is the whole pickle.toml.
type Config struct {
	PayloadVersion string       `toml:"payload_version"`
	ReviewAddendum string       `toml:"review_addendum,omitempty"`
	Flow           string       `toml:"flow,omitempty"`
	Layout         string       `toml:"layout,omitempty"`
	Commit         CommitPolicy `toml:"commit"`
	Projects       []Project    `toml:"project"`

	path string // where this config was loaded from (not serialised)
}

// ResolvedLayout is the effective layout: the recorded Layout when set, or —
// for a config written before this key existed — the decision-5 inference
// (T-108): a child registered at "." means in-tree, otherwise umbrella. Every
// caller that needs the layout calls this rather than reading Layout
// directly, so the inference lives in exactly one place.
func (c *Config) ResolvedLayout() string {
	if c.Layout != "" {
		return c.Layout
	}
	for i := range c.Projects {
		if vcs.IsRepoRoot(c.Projects[i].Path) {
			return LayoutInTree
		}
	}
	return LayoutUmbrella
}

// FlowName is the effective name of the flow this project runs: the
// configured flow key, or DefaultFlowName ("brine") when unset. Callers use
// this rather than reading Flow directly, the same pattern as Project.Prefix.
func (c *Config) FlowName() string {
	if c.Flow == "" {
		return DefaultFlowName
	}
	return c.Flow
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
	TicketPrefix     string `toml:"ticket_prefix"`
	BranchPrefix     string `toml:"branch_prefix"`
	WIPInDevelopment int    `toml:"wip_in_development"`
	WIPInReview      int    `toml:"wip_in_review"`
	ReviewAddendum   string `toml:"review_addendum,omitempty"`
	// Rick opts this child into rick interop (T-076): pickle will shell out to
	// `rick status --json` for it. Off by default, and never auto-detected
	// (DESIGN.md §3 decision 6) — a child that has never heard of rick sees no
	// behaviour change at all.
	Rick bool `toml:"rick,omitempty"`
	// SpecsRoot is the child-relative directory rick's artifacts live under.
	// Stored and validated here; not read by this package's own code (T-076
	// decision 3) — Query invokes `rick status --json` with the child's
	// directory as its cwd and lets rick find its own tree. Still unread by
	// any pickle code as of T-077 (review finding F1): that ticket's
	// `/specs/{key}/{name}` route trusts rick's own reported artifact paths
	// as its whitelist instead of an independent containment check against
	// this field (T-077 decision 2's deliberate choice).
	SpecsRoot string `toml:"specs_root,omitempty"`
}

// Prefix is the effective ticket-id prefix for this child: the configured
// ticket_prefix, or DefaultTicketPrefix ("T") when unset. Callers use this
// rather than reading TicketPrefix directly so the fallback lives in one place
// (a config loaded through Load already has the default applied, but Prefix is
// safe on a zero-valued Project too).
func (p *Project) Prefix() string {
	if p.TicketPrefix == "" {
		return DefaultTicketPrefix
	}
	return p.TicketPrefix
}

// Specs is the effective rick specs directory for this child: the configured
// specs_root, or DefaultSpecsRoot ("docs/specs") when unset — the same
// fallback shape as Prefix for TicketPrefix, and named differently from its
// field for the same reason (a method cannot share its field's name).
func (p *Project) Specs() string {
	if p.SpecsRoot == "" {
		return DefaultSpecsRoot
	}
	return p.SpecsRoot
}

// WIPLimitFor resolves a flow state's WIPKey (WIPKeyInDevelopment or
// WIPKeyInReview) to this project's configured limit. Any other key —
// including "" for a state that is not WIP-limited — returns (0, false).
// This is the only place a WIP key is resolved to a number: internal/move,
// internal/audit, internal/board and internal/serve all call it rather than
// switching on the key themselves.
func (p *Project) WIPLimitFor(key string) (int, bool) {
	switch key {
	case WIPKeyInDevelopment:
		return p.WIPInDevelopment, true
	case WIPKeyInReview:
		return p.WIPInReview, true
	default:
		return 0, false
	}
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
		if p.TicketPrefix == "" {
			p.TicketPrefix = DefaultTicketPrefix
		}
		if p.WIPInDevelopment == 0 {
			p.WIPInDevelopment = DefaultWIPInDevelopment
		}
		if p.WIPInReview == 0 {
			p.WIPInReview = DefaultWIPInReview
		}
	}
}

// Validate checks the invariants (unique non-empty names, resolvable paths, WIP >= 1,
// and that every string field is valid UTF-8). The UTF-8 check exists because Render
// has to quote every one of these fields as a TOML string: a value that isn't valid
// UTF-8 either fails to round-trip at all or round-trips as something silently
// different, and Validate is the one place that can refuse it before it ever reaches
// the file (AddProject calls this after appending, so `pickle project add` cannot
// write a value that cannot come back out the way it went in).
func (c *Config) Validate() error {
	if !utf8.ValidString(c.PayloadVersion) {
		return errors.New("pickle.toml: payload_version is not valid UTF-8")
	}
	if !utf8.ValidString(c.ReviewAddendum) {
		return errors.New("pickle.toml: review_addendum is not valid UTF-8")
	}
	if !utf8.ValidString(c.Flow) {
		return errors.New("pickle.toml: flow is not valid UTF-8")
	}
	if c.Flow != "" && c.Flow != DefaultFlowName {
		return fmt.Errorf("pickle.toml: flow %q is not a known flow (legal: %s)", c.Flow, DefaultFlowName)
	}
	if !utf8.ValidString(c.Layout) {
		return errors.New("pickle.toml: layout is not valid UTF-8")
	}
	if c.Layout != "" && c.Layout != LayoutUmbrella && c.Layout != LayoutInTree {
		return fmt.Errorf("pickle.toml: layout %q is not a known layout (legal: %s, %s)", c.Layout, LayoutUmbrella, LayoutInTree)
	}
	// A config with zero registered children is legal (T-108 decision 2): a
	// freshly installed umbrella project has none until `pickle project add`
	// registers the first one. The layout/children invariant itself — in-tree
	// implies exactly one child at "." — is enforced by `pickle doctor`
	// (decision 7), not here, so an in-tree config mid-migration still loads.
	seen := make(map[string]bool, len(c.Projects))
	// seenPrefix guards against two children sharing a non-default prefix (their
	// ids would collide). The default "T" is exempt: children that omit
	// ticket_prefix all share the one legacy global "T" namespace, which is legal
	// and must stay legal so pre-prefix multi-child workspaces still load.
	seenPrefix := make(map[string]bool, len(c.Projects))
	root := c.Root()
	for i := range c.Projects {
		p := &c.Projects[i]
		badField := invalidUTF8Field(p)
		switch {
		case strings.TrimSpace(p.Name) == "":
			return fmt.Errorf("pickle.toml: [[project]] #%d has an empty name", i+1)
		case seen[p.Name]:
			return fmt.Errorf("pickle.toml: duplicate project name %q", p.Name)
		case strings.TrimSpace(p.Path) == "":
			return fmt.Errorf("pickle.toml: project %q has an empty path", p.Name)
		case !ticketPrefixRE.MatchString(p.Prefix()):
			return fmt.Errorf("pickle.toml: project %q ticket_prefix %q is illegal (want %s)", p.Name, p.Prefix(), ticketPrefixRE)
		case p.WIPInDevelopment < 1 || p.WIPInReview < 1:
			return fmt.Errorf("pickle.toml: project %q WIP limits must be >= 1", p.Name)
		case badField != "":
			return fmt.Errorf("pickle.toml: project %q field %s is not valid UTF-8", p.Name, badField)
		}
		if pfx := p.Prefix(); pfx != DefaultTicketPrefix {
			if seenPrefix[pfx] {
				return fmt.Errorf("pickle.toml: duplicate ticket_prefix %q (each child needs a distinct prefix)", pfx)
			}
			seenPrefix[pfx] = true
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

// invalidUTF8Field returns the name of the first field of p that is not valid UTF-8,
// or "" if every field is. Render quotes each of these as a TOML string, so this is
// the set Validate has to check.
func invalidUTF8Field(p *Project) string {
	for _, f := range []struct{ name, value string }{
		{"name", p.Name},
		{"path", p.Path},
		{"build", p.Build},
		{"test", p.Test},
		{"lint", p.Lint},
		{"docs", p.Docs},
		{"ticket_prefix", p.TicketPrefix},
		{"branch_prefix", p.BranchPrefix},
		{"review_addendum", p.ReviewAddendum},
		{"specs_root", p.SpecsRoot},
	} {
		if !utf8.ValidString(f.value) {
			return f.name
		}
	}
	return ""
}

// AddProject appends a child after applying defaults; errors on duplicate/empty name,
// or if the resulting config would fail Validate (e.g. a field that is not valid
// UTF-8, or an illegal ticket_prefix) — the append is rolled back in that case, so an
// invalid value is never left in c.Projects for a caller to Save.
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
	if p.TicketPrefix == "" {
		p.TicketPrefix = DefaultTicketPrefix
	}
	if p.WIPInDevelopment == 0 {
		p.WIPInDevelopment = DefaultWIPInDevelopment
	}
	if p.WIPInReview == 0 {
		p.WIPInReview = DefaultWIPInReview
	}
	c.Projects = append(c.Projects, p)
	if err := c.Validate(); err != nil {
		c.Projects = c.Projects[:len(c.Projects)-1]
		return err
	}
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

// tomlQuote renders s as a TOML basic string. Go's %q is the wrong tool for this:
// it escapes a control character or an invalid-UTF-8 byte the Go way (\a, \v,
// \xNN), and TOML has no such escapes — \a and \v don't exist at any TOML version,
// and \xNN is a *decoder* extension (TOML 1.1, github.com/BurntSushi/toml v1.6.0)
// meant for a value that was already a legal escape, not a way to mask a byte that
// was never valid UTF-8 to begin with. tomlQuote instead uses TOML's own short
// escapes where one exists and \uXXXX otherwise, and assumes s is already valid
// UTF-8 — Validate rejects anything else before it can reach here.
func tomlQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
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
	fmt.Fprintf(&b, "payload_version = %s\n", tomlQuote(c.PayloadVersion))
	if c.ReviewAddendum != "" {
		fmt.Fprintf(&b, "review_addendum = %s\n", tomlQuote(c.ReviewAddendum))
	}
	if c.Flow != "" {
		fmt.Fprintf(&b, "flow = %s\n", tomlQuote(c.Flow))
	}
	if c.Layout != "" {
		fmt.Fprintf(&b, "layout = %s\n", tomlQuote(c.Layout))
	}
	b.WriteString("\n[commit]\n")
	fmt.Fprintf(&b, "overarching_auto = %t\n", c.Commit.OverarchingAuto)
	fmt.Fprintf(&b, "child_publish_gated = %t\n", c.Commit.ChildPublishGated)
	for _, p := range c.Projects {
		b.WriteString("\n[[project]]\n")
		fmt.Fprintf(&b, "name = %s\n", tomlQuote(p.Name))
		fmt.Fprintf(&b, "path = %s\n", tomlQuote(p.Path))
		for _, kv := range []struct {
			k, v string
		}{{"build", p.Build}, {"test", p.Test}, {"lint", p.Lint}, {"docs", p.Docs}} {
			if kv.v != "" {
				fmt.Fprintf(&b, "%s = %s\n", kv.k, tomlQuote(kv.v))
			}
		}
		fmt.Fprintf(&b, "ticket_prefix = %s\n", tomlQuote(p.Prefix()))
		fmt.Fprintf(&b, "branch_prefix = %s\n", tomlQuote(p.BranchPrefix))
		fmt.Fprintf(&b, "wip_in_development = %d\n", p.WIPInDevelopment)
		fmt.Fprintf(&b, "wip_in_review = %d\n", p.WIPInReview)
		if p.Rick {
			fmt.Fprintf(&b, "rick = true\n")
			fmt.Fprintf(&b, "specs_root = %s\n", tomlQuote(p.Specs()))
		}
		if p.ReviewAddendum != "" {
			fmt.Fprintf(&b, "review_addendum = %s\n", tomlQuote(p.ReviewAddendum))
		}
	}
	return b.String()
}

// Save writes the canonical render to path (or the loaded path if empty), atomically
// and preserving the file's existing permission bits (see atomicfile.WriteFile).
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
	return atomicfile.WriteFile(path, []byte(c.Render()))
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
	return atomicfile.WriteFile(path, []byte(updated))
}

// PayloadVersionStampable reports whether SetPayloadVersionInPlace(path,
// version) would succeed, without writing anything. `pickle doctor` uses it
// to decide what to tell the user: a plain "run `pickle upgrade`" is only
// useful advice when upgrade could actually succeed. err, when non-nil, is
// the same refusal SetPayloadVersionInPlace would return.
func PayloadVersionStampable(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if _, err := setPayloadVersion(string(data), version); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// SetLayoutInPlace back-fills a "layout" key into the pickle.toml at path
// when the key is entirely absent, using the same surgical, comment-preserving
// edit as SetPayloadVersionInPlace (T-108 decision 6): `pickle upgrade` calls
// this so no separate migration command is needed. Unlike payload_version,
// layout is never rewritten when already present — the recorded value always
// wins over inference (decision 5), so an existing key, of any value, is left
// untouched byte-for-byte.
func SetLayoutInPlace(path, layout string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	mark := ""
	text := string(data)
	if strings.HasPrefix(text, bom) {
		mark, text = bom, strings.TrimPrefix(text, bom)
	}
	if topLevelKeyPresent(text, layoutKeySpellings[:]) {
		return nil // present already — decision 5: never overwritten here
	}
	updated := insertTopLevelKey(strings.Split(text, "\n"), topLevelInsertAt(text), "layout", layout)
	if updated == text {
		return nil
	}
	return atomicfile.WriteFile(path, []byte(mark+updated))
}

// topLevelKeyPresent reports whether text defines any of spellings as a
// top-level key (i.e. before the first table header, and not inside a string
// or array) — the presence half of the scan replacePayloadVersionLine also
// performs.
func topLevelKeyPresent(text string, spellings []string) bool {
	var st scanState
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		topLevel := st.atTopLevel()
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		body := line[indent:]
		if topLevel {
			switch {
			case body == "" || strings.HasPrefix(body, "#"):
			case strings.HasPrefix(body, "["):
				return false // table header reached — key never appeared above it
			default:
				if _, ok := matchKey(body, spellings); ok {
					return true
				}
			}
		}
		advance(line, &st)
	}
	return false
}

// topLevelInsertAt finds the line index a new top-level key belongs at: right
// before the first real table header, or at EOF when the file has none — the
// same rule replacePayloadVersionLine's insert path applies.
func topLevelInsertAt(text string) int {
	lines := strings.Split(text, "\n")
	var st scanState
	for i, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		if st.atTopLevel() {
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			body := line[indent:]
			if strings.HasPrefix(body, "[") {
				return i
			}
		}
		advance(line, &st)
	}
	return len(lines)
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
	if !treeEqual(b, a) {
		return fmt.Errorf("setting %s would change other values in the file; set it by hand",
			payloadVersionKey)
	}
	return nil
}

// treeEqual compares two decoded TOML trees, treating two NaN float64s as
// equal. reflect.DeepEqual does not — IEEE 754 defines NaN != NaN — which
// would make the safety gate above unpassable by construction for any file
// that legally contains `= nan` somewhere, no matter how harmless the edit
// actually is. Every other value keeps reflect.DeepEqual's semantics.
func treeEqual(a, b any) bool {
	if af, ok := a.(float64); ok {
		bf, ok := b.(float64)
		if !ok {
			return false
		}
		if math.IsNaN(af) || math.IsNaN(bf) {
			return math.IsNaN(af) && math.IsNaN(bf)
		}
		return af == bf
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvv, ok := bv[k]
			if !ok || !treeEqual(v, bvv) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !treeEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

func decodeTree(text string) (map[string]any, error) {
	var m map[string]any
	_, err := toml.Decode(text, &m)
	return m, err
}

// scanState is the lexical context carried from one line to the next while
// scanning for the top-level payload_version key: whether a triple-quoted
// string opened on an earlier line is still open, and how many '['..']'
// pairs (array literals) are currently unclosed. Inline tables are out of
// scope — TOML requires them on a single line, so a bare payload_version
// inside one would never reach the line-start prefix check below.
type scanState struct {
	multilineDelim string // "", `"""`, or `'''`
	bracketDepth   int
}

// atTopLevel reports whether a key definition or table header found at the
// start of a line, given this state, is really one — as opposed to text that
// merely looks like one inside a string or an array.
func (st scanState) atTopLevel() bool {
	return st.multilineDelim == "" && st.bracketDepth == 0
}

// advance scans one raw line's non-string, non-comment content and updates
// st for the line that follows: it tracks bracket depth, skips over
// single-line quoted strings (so a '[' or '#' inside one is never mistaken
// for syntax), and opens or closes a multi-line string. Every line is passed
// through it exactly once, regardless of how that line was classified.
func advance(line string, st *scanState) {
	i, n := 0, len(line)
	for i < n {
		if st.multilineDelim != "" {
			if st.multilineDelim == "'''" {
				// A literal string has no escapes at all, so the delimiter can
				// never be masked by one: a plain substring search is exact.
				idx := strings.Index(line[i:], st.multilineDelim)
				if idx < 0 {
					return // the rest of the line is string content
				}
				i += idx + len(st.multilineDelim)
				st.multilineDelim = ""
				continue
			}
			// A basic multi-line string does have escapes, so `\` + the next
			// byte must be skipped before testing for the closing delimiter —
			// otherwise an escaped `"` inside the string (e.g. `\"""`, an
			// escaped quote followed by two more) is misread as opening the
			// three-quote delimiter one byte early. A `\` as the line's very
			// last byte is a line-continuation, not an escape (there is no next
			// byte on this line to pair it with), so it does not consume
			// anything here; an *escaped* backslash immediately before the
			// delimiter (`\\"""`) is consumed as its own pair first and so
			// still lets the real delimiter close the string.
			j := i
			closed := false
			for j < n {
				if line[j] == '\\' && j+1 < n {
					j += 2
					continue
				}
				if strings.HasPrefix(line[j:], st.multilineDelim) {
					closed = true
					break
				}
				j++
			}
			if !closed {
				return // the rest of the line is string content
			}
			i = j + len(st.multilineDelim)
			st.multilineDelim = ""
			continue
		}
		switch {
		case line[i] == '#':
			return // the rest of the line is a comment
		case strings.HasPrefix(line[i:], `"""`):
			st.multilineDelim = `"""`
			i += 3
		case strings.HasPrefix(line[i:], "'''"):
			st.multilineDelim = "'''"
			i += 3
		case line[i] == '"':
			j := i + 1
			for j < n && line[j] != '"' {
				if line[j] == '\\' {
					j++
				}
				j++
			}
			i = j + 1
		case line[i] == '\'':
			if idx := strings.IndexByte(line[i+1:], '\''); idx >= 0 {
				i += idx + 2
			} else {
				i = n
			}
		case line[i] == '[':
			st.bracketDepth++
			i++
		case line[i] == ']':
			if st.bracketDepth > 0 {
				st.bracketDepth--
			}
			i++
		default:
			i++
		}
	}
}

// payloadVersionKeySpellings are the token forms TOML allows for the key.
// pickle always writes the bare form; a hand-edited file may use either
// quoted spelling.
var payloadVersionKeySpellings = [...]string{
	payloadVersionKey,
	`"` + payloadVersionKey + `"`,
	`'` + payloadVersionKey + `'`,
}

// layoutKeySpellings mirrors payloadVersionKeySpellings for the "layout" key
// (T-108): pickle always writes the bare form; a hand-edited file may quote it.
var layoutKeySpellings = [...]string{
	"layout",
	`"layout"`,
	"'layout'",
}

// matchKey reports whether body — a line already stripped of its leading
// indentation — opens with one of spellings followed by optional whitespace
// and '=', and if so the byte offset of that '=' within body. It deliberately
// does not match a longer identifier that merely starts with the key, such as
// payload_version_note.
func matchKey(body string, spellings []string) (eqOffset int, ok bool) {
	for _, spelling := range spellings {
		rest, cut := strings.CutPrefix(body, spelling)
		if !cut {
			continue
		}
		ws := len(rest) - len(strings.TrimLeft(rest, " \t"))
		if !strings.HasPrefix(rest[ws:], "=") {
			continue
		}
		return len(spelling) + ws, true
	}
	return 0, false
}

// replacePayloadVersionLine performs the line edit. It is deliberately naive
// about anything past bracket depth and multi-line-string state —
// verifyOnlyPayloadVersion is what makes the result safe regardless.
func replacePayloadVersionLine(text, version string) (string, error) {
	lines := strings.Split(text, "\n")
	insertAt := len(lines)
	var st scanState

	for i, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		topLevel := st.atTopLevel()
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		body := line[indent:]

		if topLevel {
			switch {
			case body == "" || strings.HasPrefix(body, "#"):
				// nothing to classify
			case strings.HasPrefix(body, "["):
				// A table header ends the top-level scope: a payload_version
				// below this point belongs to [commit] or a [[project]].
				insertAt = i
				return insertPayloadVersion(lines, insertAt, version)
			default:
				if eqOffset, ok := matchKey(body, payloadVersionKeySpellings[:]); ok {
					return rewriteFoundKey(lines, i, raw, line, indent, eqOffset, version)
				}
			}
		}
		advance(line, &st)
	}

	return insertPayloadVersion(lines, insertAt, version)
}

// rewriteFoundKey builds the edited text once the key's line and the byte
// offset of its '=' within that line have been found.
func rewriteFoundKey(lines []string, i int, raw, line string, indent, eqOffset int, version string) (string, error) {
	start := indent + eqOffset + 1
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}
	switch {
	case strings.HasPrefix(line[start:], `"""`) || strings.HasPrefix(line[start:], "'''"):
		// A multi-line string may continue past this line; rewriting it
		// blind could corrupt the file. Refuse loudly instead.
		return "", fmt.Errorf("line %d: payload_version's value is a multi-line string; set it by hand", i+1)
	case strings.HasPrefix(line[start:], "["):
		// payload_version's own value is an array. pickle never writes this
		// shape; rewriting it as a single `= "value"` line would silently
		// discard it rather than merely fail to touch it, so refuse instead.
		return "", fmt.Errorf("line %d: payload_version's value is an array; rewriting it as a single line could leave the file unparseable — set it by hand", i+1)
	}
	end := valueEnd(line, start)
	lines[i] = line[:indent] + payloadVersionKey + " = " + tomlQuote(version) + line[end:]
	if raw != line {
		lines[i] += "\r"
	}
	return strings.Join(lines, "\n"), nil
}

// insertPayloadVersion is the not-present path for payload_version: insert
// the key at insertAt. A thin wrapper over insertTopLevelKey, kept so its two
// call sites below need no change.
func insertPayloadVersion(lines []string, insertAt int, version string) (string, error) {
	return insertTopLevelKey(lines, insertAt, payloadVersionKey, version), nil
}

// insertTopLevelKey inserts `key = "value"` at insertAt, keeping any blank
// lines attached to whatever follows rather than orphaning them above the new
// key. Shared by insertPayloadVersion and SetLayoutInPlace (T-108) so the two
// keys this file ever back-fills use one insertion rule.
func insertTopLevelKey(lines []string, insertAt int, key, value string) string {
	for insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}
	entry := key + " = " + tomlQuote(value)
	// Only match the file's CRLF style when another line actually follows the
	// insert point: appending \r to what becomes the file's last line adds a
	// bare trailing \r with no \n after it, which the parse-back gate then
	// refuses as unparseable (a file whose last line was never newline-
	// terminated to begin with, e.g. "\r\n#", still has nothing to be
	// consistent with at that point).
	if usesCRLF(lines) && insertAt < len(lines) {
		entry += "\r" // match the file rather than leaving one lone LF line
	}
	lines = append(lines[:insertAt], append([]string{entry}, lines[insertAt:]...)...)
	return strings.Join(lines, "\n")
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

// The atomic write primitive itself (create-temp, write, fsync, chmod,
// rename) lives in internal/atomicfile as of T-101 — it was never
// config-specific and internal/serve's write paths need it too. See
// atomicfile.WriteFile for the full contract this package relies on.
