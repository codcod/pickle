// Package doctor implements `pickle doctor`: the install-side analogue of
// `board audit`. It is a pure, fixture-testable check of an installed project's
// integrity — the installed skill payload, the .claude view symlink, the
// AGENTS.md/CLAUDE.md marker block, the agent scaffolds, the pre-commit guard,
// pickle.toml, and each registered child's git repo. Like audit, it never prints
// or exits — it returns findings.
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
	"github.com/codcod/pickle/internal/hook"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/vcs"
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
// holds pickle.toml). version is used for a payload-drift warning, unless the
// skill directory is a self-host link (install.SkillLinked), in which case the
// payload is that linked source by construction and the comparison is skipped.
// payload is the binary's embedded payload, used only to drift-check the
// pickle-owned agent scaffolds. Filesystem checks run even when pickle.toml
// fails to parse; per-child checks run only when it loaded.
func Check(root, version string, payload fs.FS) Result {
	var r Result

	selfHost := install.SkillLinked(root)

	checkLegacyPaths(root, &r)
	cfg := checkConfig(root, &r)
	checkSkill(root, selfHost, &r)
	checkClaudeView(root, &r)
	checkMarkers(root, cfg, &r)
	checkAgentScaffolds(root, payload, &r)
	checkHooks(root, &r)
	if cfg != nil {
		checkChildren(root, cfg, &r)
		checkVersion(cfg, version, selfHost, &r)
	}

	sort.Strings(r.Errors)
	sort.Strings(r.Warnings)
	sort.Strings(r.Passed)
	return r
}

// checkLegacyPaths errors while either pre-brine install path (T-074) is still
// present: a project that has upgraded only partially, or not at all, is
// carrying a stale skill copy that some agents (native .agents/skills
// discovery) would prefer over the current one — a state doctor must not wave
// through as green. A pure filesystem read, so it runs even when pickle.toml
// fails to parse. Delete alongside install.sweepLegacySkill at 1.0 (T-074).
func checkLegacyPaths(root string, r *Result) {
	found := false
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(install.LegacySkillDir))); err == nil {
		found = true
		r.errf("skill: legacy %s still present (renamed to %s in T-074) — run `pickle upgrade`",
			install.LegacySkillDir, install.SkillDir)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(install.LegacyClaudeSkillLink))); err == nil {
		found = true
		r.errf("claude: legacy %s still present (renamed to %s in T-074) — run `pickle upgrade`",
			install.LegacyClaudeSkillLink, install.ClaudeSkillLink)
	}
	if !found {
		r.ok("no legacy skill path present (" + install.LegacySkillDir + ")")
	}
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
	r.ok(fmt.Sprintf("flow: %s", cfg.FlowName()))
	return cfg
}

// checkSkill verifies the skill payload resolves (following a dev/self-host
// symlink) to a directory carrying SKILL.md and resources/tickets-README.md.
// When selfHost is true, the passed line names the link and its target instead
// of the generic "skill payload present" text, since there is no installed
// copy to speak of — the payload *is* the linked source.
func checkSkill(root string, selfHost bool, r *Result) {
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
	if selfHost {
		// dir is the link path itself (os.Stat above followed it; Readlink must
		// not). Only a TOCTOU race can fail here — selfHost came from an Lstat
		// on this same path — so the fallback is belt-and-braces, not a case
		// worth naming in the output.
		target, err := os.Readlink(dir)
		if err != nil {
			target = "?"
		}
		r.ok(fmt.Sprintf("skill payload present (%s -> %s, self-host link)", install.SkillDir, target))
		return
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
// resolve; absent is fine). When cfg parsed, a present block is further
// compared against install.MarkerBlock(cfg): a difference is a drift warning,
// never an error — a hand-edited block is a legitimate state until the next
// `pickle upgrade` (or `project add|remove`) re-injects it. A block cannot be
// compared when pickle.toml itself failed to parse (cfg == nil); its absence
// is already reported by checkConfig.
func checkMarkers(root string, cfg *config.Config, r *Result) {
	agents := filepath.Join(root, "AGENTS.md")
	body, ok := install.InstalledMarkerBody(agents)
	if !ok {
		r.errf("markers: AGENTS.md is missing or has no pickle marker block")
	} else {
		r.ok("AGENTS.md marker block present")
		checkMarkerDrift("AGENTS.md", body, cfg, r)
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
	body, ok = install.InstalledMarkerBody(claude)
	if !ok {
		r.errf("markers: CLAUDE.md has no pickle marker block")
		return
	}
	r.ok("CLAUDE.md marker block present")
	checkMarkerDrift("CLAUDE.md", body, cfg, r)
}

// checkMarkerDrift compares an installed marker body (already trimmed by
// install.InstalledMarkerBody) to a fresh install.MarkerBlock(cfg) render,
// byte-exact — the literal question "would pickle upgrade change this file?".
func checkMarkerDrift(name, installed string, cfg *config.Config, r *Result) {
	if cfg == nil {
		return
	}
	want := strings.Trim(install.MarkerBlock(cfg), "\n")
	if installed == want {
		r.ok(name + " marker block current")
		return
	}
	r.warnf("markers: %s block differs from what pickle.toml renders (%d line(s) differ) — run `pickle upgrade`; hand-written content belongs outside the markers",
		name, diffLineCount(installed, want))
}

// diffLineCount counts how many positional lines differ between a and b,
// padding the shorter side with empty lines. Cheap and needs no diff
// algorithm — doctor reports a count, never a unified diff.
func diffLineCount(a, b string) int {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	n := len(al)
	if len(bl) > n {
		n = len(bl)
	}
	count := 0
	for i := 0; i < n; i++ {
		var av, bv string
		if i < len(al) {
			av = al[i]
		}
		if i < len(bl) {
			bv = bl[i]
		}
		if av != bv {
			count++
		}
	}
	return count
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

// checkHooks reports the state of the pre-commit bookkeeping guard.
//
// The guard is opt-in, so its absence is never a finding — only a line saying
// how to arm it, which is also the answer for the case a pickle.toml key was
// once considered for: hooks live in .git/ and are never cloned, so a fresh
// clone of a project that uses the guard starts without one. Only a stale
// pickle-owned shim earns a warning, mirroring the agent-scaffold drift check.
// A tree that is not a git repository says so and stops: `checkChildren` already
// owns that verdict as a real check, and duplicating it here would report the
// same broken install twice.
func checkHooks(root string, r *Result) {
	st, err := hook.Status(root)
	if err != nil {
		r.warnf("hooks: %v", err)
		return
	}
	switch st.Kind {
	case hook.KindNoRepo:
		r.ok("pre-commit guard not applicable (no git repository at the install root)")
	case hook.KindAbsent:
		r.ok("pre-commit guard not installed (optional — `pickle hooks install` arms it; hooks are per-clone)")
	case hook.KindForeign:
		r.ok(fmt.Sprintf("pre-commit hook present but not pickle's (%s) — left alone", st.Path))
	case hook.KindOwned:
		if st.Stale {
			r.warnf("hooks: %s was written by an older pickle (shim v%d, this binary ships v%d) — run `pickle upgrade`",
				st.Path, st.Version, hook.ShimVersion)
			return
		}
		// The shim on disk is current, but that only matters if the `pickle` the
		// shim actually resolves from PATH can run it (T-068) — probed only here,
		// for an owned+current shim, so an absent/foreign/stale guard (already
		// handled above, and free of the exec cost) never pays for it either.
		// The exec itself lives entirely behind internal/hook (T-057 decision 12):
		// this package never spawns a process directly, and still only returns
		// findings rather than printing or exiting. It does now *cause* an exec
		// through hook.Probe, which is why that call is confined to this one
		// branch. checkChildren below causes the only other exec, confined the
		// same way behind internal/vcs (T-051) — every other doctor check
		// remains a pure filesystem read.
		if p := hook.Probe().Problem(); p != "" {
			r.warnf("hooks: %s is installed and current, but %s", st.Path, p)
			return
		}
		r.ok(fmt.Sprintf("pre-commit guard installed and current (%s), and the pickle on PATH can run it", st.Path))
	}
	// Nothing else in the tree earns this treatment (T-068 decision 10): `pickle
	// serve`, the pi scaffolds and opencode.jsonc are all "pickle wrote a file
	// something else reads", but none of them re-executes through PATH the way
	// the shim calls back into `pickle hooks run` — only the hook has a
	// version-coupled contract with the binary.
}

// checkChildren verifies every registered child path resolves to a git repo
// (a .git entry — a directory for a normal clone, or a file for a worktree /
// submodule), then — for a child that is not the repository root itself,
// however that path was spelled (see vcs.IsRepoRoot) — warns
// when the overarching repo would still stage that path whole (T-051): until
// a .gitignore entry (or a deliberate gitlink) exists, the window between a
// child appearing and someone remembering to ignore it is a staging accident
// waiting to happen. The single-repo default (the child *is* the repo) and an
// already-tracked gitlink are both silent — see vcs.ChildState.
func checkChildren(root string, cfg *config.Config, r *Result) {
	for _, p := range cfg.Projects {
		abs := filepath.Join(root, p.Path)
		if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
			r.errf("child %q: %s is not a git repository", p.Name, p.Path)
			continue
		}
		r.ok(fmt.Sprintf("child %q is a git repository (%s)", p.Name, p.Path))

		if vcs.IsRepoRoot(p.Path) {
			continue
		}
		switch st := vcs.ChildState(root, p.Path); st {
		case vcs.Stageable:
			r.warnf("child %q: %s", p.Name, st.Advice(p.Path))
		case vcs.Ignored:
			r.ok(fmt.Sprintf("child %q is git-ignored (%s)", p.Name, p.Path))
		case vcs.Tracked:
			// "Tracked" covers a deliberate gitlink *and* a child whose
			// contents were already committed as ordinary files, which this
			// package cannot tell apart (see vcs's package doc) — so the line
			// says what git reports and nothing more. It must not promise the
			// child is safe: if the contents were committed as ordinary files,
			// `git add <child>` still stages whatever is untracked under it.
			r.ok(fmt.Sprintf("child %q is tracked by this repository (%s)", p.Name, p.Path))
		case vcs.Unknown:
			// no git on PATH, root not a repository, or an unexpected exit
			// code — silent by design (vcs.State's zero value).
		}
	}
}

// checkVersion warns when the installed payload version differs from the
// running binary. Advisory only — never an error. T-026: `pickle upgrade`
// can legitimately refuse to stamp a new version (a handful of pickle.toml
// shapes defeat the line-based writer), so recommending it unconditionally
// would send the user to a command that is going to fail. Probe first — with
// PayloadVersionStampable, which changes nothing — and give whichever advice
// actually applies.
//
// T-046: when selfHost is true, the installed skill directory is a symlink to
// the payload source (this tree's own skill/), not an installed copy — the
// payload_version-vs-binary comparison is meaningless in that arrangement, so
// it is skipped entirely, before either of the two branches above (and ahead
// of the version=={"","dev"} guard and the equal-version early return, since
// the check genuinely does not run rather than happening to agree). `upgrade`
// still stamps payload_version in this mode (it keeps refreshing everything
// else it owns), so this is purely doctor no longer having an opinion about a
// number it has no business comparing — it can never send you to `pickle
// upgrade` in vain.
func checkVersion(cfg *config.Config, version string, selfHost bool, r *Result) {
	if selfHost {
		r.ok(fmt.Sprintf("payload version check skipped (%s is a self-host link, so the payload is this tree, not an installed copy)", install.SkillDir))
		return
	}
	if version == "" || version == "dev" {
		return
	}
	if cfg.PayloadVersion == version {
		return
	}
	if err := config.PayloadVersionStampable(cfg.Path(), version); err != nil {
		r.warnf("payload version %q differs from binary %q, and `pickle upgrade` cannot fix it automatically (%s) — edit payload_version by hand",
			cfg.PayloadVersion, version, err)
		return
	}
	r.warnf("payload version %q differs from binary %q — run `pickle upgrade`", cfg.PayloadVersion, version)
}
