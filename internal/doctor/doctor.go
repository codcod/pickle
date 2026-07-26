// Package doctor implements `pickle doctor`: the install-side analogue of
// `board audit`. It is a pure, fixture-testable check of an installed project's
// integrity — the installed skill payload, the .claude view symlink, the
// AGENTS.md/CLAUDE.md marker block, the agent scaffolds, pickle.toml, and each
// registered child's git repo. Like audit, it never prints or exits — it
// returns findings.
package doctor

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/install"
)

// Result is the outcome of a doctor run. Errors mean a broken install (the CLI
// exits non-zero); Warnings are advisory (exit 0). Passed lists the checks that
// held, for verbose output.
type Result struct {
	Errors   []string
	Warnings []string
	Passed   []string
}

func (r *Result) errf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}
func (r *Result) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}
func (r *Result) ok(msg string) { r.Passed = append(r.Passed, msg) }

// Check inspects an installed pickle project rooted at root (the directory that
// holds pickle.toml). version is the running binary's version, used only for a
// payload-drift warning; payload is the binary's embedded payload, used only to
// drift-check the pickle-owned agent scaffolds. Filesystem checks run even when
// pickle.toml fails to parse; per-child checks run only when it loaded.
func Check(root, version string, payload fs.FS) Result {
	var r Result

	cfg := checkConfig(root, &r)
	checkSkill(root, &r)
	checkClaudeView(root, &r)
	checkMarkers(root, &r)
	checkAgentScaffolds(root, payload, &r)
	if cfg != nil {
		checkChildren(root, cfg, &r)
		checkVersion(cfg, version, &r)
	}

	sort.Strings(r.Errors)
	sort.Strings(r.Warnings)
	sort.Strings(r.Passed)
	return r
}

// checkConfig loads pickle.toml. A parse/validate failure is a finding, not a
// crash: it returns nil and the filesystem checks still run.
func checkConfig(root string, r *Result) *config.Config {
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		r.errf("pickle.toml: %v", err)
		return nil
	}
	r.ok(fmt.Sprintf("pickle.toml parses (%d child-project(s))", len(cfg.Projects)))
	return cfg
}

// checkSkill verifies the skill payload resolves (following a dev/self-host
// symlink) to a directory carrying SKILL.md and resources/tickets-README.md.
func checkSkill(root string, r *Result) {
	dir := filepath.Join(root, filepath.FromSlash(install.SkillDir))
	fi, err := os.Stat(dir) // follows symlinks
	if err != nil || !fi.IsDir() {
		r.errf("skill: %s does not resolve to a directory", install.SkillDir)
		return
	}
	for _, f := range []string{"SKILL.md", filepath.FromSlash("resources/tickets-README.md")} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			r.errf("skill: %s missing %s", install.SkillDir, filepath.ToSlash(f))
			return
		}
	}
	r.ok(fmt.Sprintf("skill payload present (%s)", install.SkillDir))
}

// checkClaudeView verifies the Claude view. The Claude artifacts are optional
// (install can run with --no-claude): an absent link is not a finding, but a
// present one must be a symlink that resolves.
func checkClaudeView(root string, r *Result) {
	link := filepath.Join(root, filepath.FromSlash(install.ClaudeSkillLink))
	fi, err := os.Lstat(link)
	if err != nil {
		return // absent — Claude view not installed
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		r.errf("claude: %s exists but is not a symlink", install.ClaudeSkillLink)
		return
	}
	if _, err := os.Stat(link); err != nil { // resolve the target
		r.errf("claude: %s is a broken symlink", install.ClaudeSkillLink)
		return
	}
	r.ok(fmt.Sprintf("claude view resolves (%s)", install.ClaudeSkillLink))
}

// checkMarkers verifies the pickle-managed marker block. AGENTS.md is required;
// CLAUDE.md is optional (regular file must carry the markers; a symlink must
// resolve; absent is fine).
func checkMarkers(root string, r *Result) {
	agents := filepath.Join(root, "AGENTS.md")
	if !hasMarkerBlock(agents) {
		r.errf("markers: AGENTS.md is missing or has no pickle marker block")
	} else {
		r.ok("AGENTS.md marker block present")
	}

	claude := filepath.Join(root, "CLAUDE.md")
	fi, err := os.Lstat(claude)
	if err != nil {
		return // absent — fine
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(claude); err != nil {
			r.errf("markers: CLAUDE.md is a broken symlink")
			return
		}
		r.ok("CLAUDE.md symlink resolves")
		return
	}
	if !hasMarkerBlock(claude) {
		r.errf("markers: CLAUDE.md has no pickle marker block")
		return
	}
	r.ok("CLAUDE.md marker block present")
}

// hasMarkerBlock reports whether path contains a MarkerBegin…MarkerEnd pair.
func hasMarkerBlock(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := string(b)
	bi := strings.Index(text, install.MarkerBegin)
	ei := strings.Index(text, install.MarkerEnd)
	return bi >= 0 && ei > bi
}

// checkAgentScaffolds verifies the pi agent scaffolds. They are optional (laid
// down only by `--agent pi`) and pickle-owned: an absent file is not a finding,
// but a present one that differs from this binary's embedded copy earns a
// drift warning — `pickle upgrade` refreshes it. opencode.jsonc is user-owned
// after creation (pickle writes it whole at most once and never merges JSONC),
// so doctor deliberately performs no opencode checks.
func checkAgentScaffolds(root string, payload fs.FS, r *Result) {
	if payload == nil {
		return
	}
	for _, f := range install.PiScaffolds {
		cur, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Installed)))
		if err != nil {
			continue // absent — pi scaffolds not installed
		}
		want, err := fs.ReadFile(payload, f.Asset)
		if err != nil {
			r.errf("agents: embedded asset %s unreadable: %v", f.Asset, err)
			continue
		}
		if !bytes.Equal(cur, want) {
			r.warnf("agents: %s differs from the shipped version — run `pickle upgrade` (customizations belong in sibling files)", f.Installed)
			continue
		}
		r.ok(fmt.Sprintf("agent scaffold current (%s)", f.Installed))
	}
}

// checkChildren verifies every registered child path resolves to a git repo
// (a .git entry — a directory for a normal clone, or a file for a worktree /
// submodule).
func checkChildren(root string, cfg *config.Config, r *Result) {
	for _, p := range cfg.Projects {
		abs := filepath.Join(root, p.Path)
		if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
			r.errf("child %q: %s is not a git repository", p.Name, p.Path)
			continue
		}
		r.ok(fmt.Sprintf("child %q is a git repository (%s)", p.Name, p.Path))
	}
}

// checkVersion warns when the installed payload version differs from the running
// binary (a `pickle upgrade` is available). Advisory only — never an error.
func checkVersion(cfg *config.Config, version string, r *Result) {
	if version == "" || version == "dev" {
		return
	}
	if cfg.PayloadVersion != version {
		r.warnf("payload version %q differs from binary %q — run `pickle upgrade`", cfg.PayloadVersion, version)
	}
}
