// Package install implements `pickle install`: it lays the brine flow into a
// project — the embedded skill payload, the tickets/ board scaffold, the
// AGENTS.md/CLAUDE.md marker blocks, the Claude view symlink, and a pickle.toml
// registering the first child-project. It is per-project (never writes to ~/ or
// outside root), idempotent, and safe to re-run: the payload and markers are
// refreshed in place while instance data (BOARD.md, tickets, pickle.toml) is
// preserved once present.
package install

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/hook"
	"github.com/codcod/pickle/internal/ticket"
)

// hookLabel names a hook path for the created/removed lists: relative to root
// when it lives inside the project (the normal .git/hooks case), absolute when
// core.hooksPath points somewhere else entirely.
func hookLabel(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}

// Marker delimiters for the injected AGENTS.md/CLAUDE.md block. Anything between
// them is pickle-managed and replaced on re-run; text outside is preserved.
const (
	MarkerBegin = "<!-- pickle:begin -->"
	MarkerEnd   = "<!-- pickle:end -->"

	// SkillDir is the installed skill payload location (relative to the project root).
	SkillDir = ".agents/skills/brine"
	// ClaudeSkillLink is the Claude Code view of the skill; ClaudeSkillTarget is the
	// relative symlink target it points at (SkillDir, from inside .claude/skills/).
	ClaudeSkillLink   = ".claude/skills/brine"
	ClaudeSkillTarget = "../../.agents/skills/brine"

	// OpencodeConfigFile is the opencode config `--agent opencode` scaffolds
	// (whole-file, only when absent — pickle never parses or merges JSONC);
	// OpencodeAsset is the embedded template it is written from.
	OpencodeConfigFile = "opencode.jsonc"
	OpencodeAsset      = "agents/opencode/opencode.jsonc"
	// PiExtensionsDir holds the pi extension files `--agent pi` scaffolds.
	PiExtensionsDir = ".pi/extensions"
)

// Legacy install paths from before T-074 renamed the installed skill directory
// to brine. install/upgrade never write these paths; Upgrade and Uninstall
// call sweepLegacySkill to remove them (or, for a self-host symlink, re-link it
// at the new name — see sweepLegacySkill), and doctor errors while either is
// still present. Delete this block, sweepLegacySkill, checkLegacyPaths and
// their call sites and tests at 1.0 (T-074).
const (
	LegacySkillDir        = ".agents/skills/ticket-flow"
	LegacyClaudeSkillLink = ".claude/skills/ticket-flow"
)

// SkillLinked reports whether the installed skill directory is a symlink: the
// dev/self-host arrangement in which .agents/skills/brine points at the
// payload source (this repo's skill/) instead of holding a copy of it. install
// and upgrade never overwrite through such a link, and uninstall removes the
// link itself rather than RemoveAll-ing the tree it points at; doctor uses it
// to skip the payload_version comparison, which would otherwise compare against
// an installed copy that does not exist.
//
// A broken link still counts — it is still not an installed copy, and doctor
// reports the breakage separately. Note the asymmetry: false means "not a
// link", which covers both a real directory and a path that could not be
// stat'ed at all, so a caller using !SkillLinked to guard a destructive branch
// should be sure that is what it wants (Upgrade is: RemoveAll of a missing
// path is a no-op).
func SkillLinked(root string) bool {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(SkillDir)))
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// AgentAsset pairs an installed agent-scaffold path (relative to the project
// root, slash-separated) with its source inside the embedded payload.
type AgentAsset struct {
	Installed string
	Asset     string
}

// PiScaffolds are the pi extension files `--agent pi` lays down. Unlike
// opencode.jsonc they are pickle-owned: refreshed by upgrade (when present),
// removed by uninstall, drift-checked by doctor. User customizations belong in
// sibling extension files, never in these.
var PiScaffolds = []AgentAsset{
	{Installed: ".pi/extensions/docs-readability.ts", Asset: "agents/pi/extensions/docs-readability.ts"},
	{Installed: ".pi/extensions/pickle-guardrails.ts", Asset: "agents/pi/extensions/pickle-guardrails.ts"},
}

// Agents selects which coding agents install wires up (the --agent flag).
// Claude gets the .claude view + CLAUDE.md marker; opencode gets opencode.jsonc
// (docs-readability subagent + bash guardrails); pi gets .pi/extensions/*.
// There is no autodetection — the set is exactly what the user names.
type Agents struct {
	Claude   bool
	Opencode bool
	Pi       bool
}

// ParseAgents parses a --agent value: a comma-separated subset of "claude",
// "opencode", "pi". Empty tokens (an empty flag, stray commas) and unknown
// names are errors — callers apply the "claude" default before parsing.
func ParseAgents(s string) (Agents, error) {
	var a Agents
	for _, tok := range strings.Split(s, ",") {
		switch strings.TrimSpace(tok) {
		case "claude":
			a.Claude = true
		case "opencode":
			a.Opencode = true
		case "pi":
			a.Pi = true
		default:
			return Agents{}, fmt.Errorf("unknown agent %q (legal: claude, opencode, pi)", strings.TrimSpace(tok))
		}
	}
	return a, nil
}

// Options configures a single install run.
type Options struct {
	ProjectName string // first child-project name (required; caller defaults it)
	ProjectPath string // first child path, relative to root (defaults to ".")
	Build       string // optional child commands
	Test        string
	Lint        string
	Docs        string
	Agents      Agents // which agents to wire up (no autodetection)
	ClaudeLink  bool   // make CLAUDE.md a symlink to AGENTS.md instead of a marker block
}

// Result records what the run created, left in place, or removed, for the CLI
// summary. Removed is populated by Uninstall (and, under UninstallOptions.DryRun,
// records what would be removed without mutating the tree).
type Result struct {
	Created []string
	Skipped []string
	Removed []string
	// Notes carries longer, human-directed guidance (e.g. the opencode blocks to
	// merge by hand when opencode.jsonc already exists) for the CLI to print
	// after the created/skipped lists.
	Notes []string
}

func (r *Result) created(f string) { r.Created = append(r.Created, f) }
func (r *Result) skipped(f string) { r.Skipped = append(r.Skipped, f) }
func (r *Result) removed(f string) { r.Removed = append(r.Removed, f) }
func (r *Result) note(n string)    { r.Notes = append(r.Notes, n) }

// Run performs the install into root using the embedded payload FS (rooted at
// the binary's "skill" tree) and payloadVersion (stamped into pickle.toml).
func Run(payload fs.FS, root, payloadVersion string, opts Options) (Result, error) {
	var res Result
	if opts.ProjectName == "" {
		return res, fmt.Errorf("project name is required")
	}
	if opts.ProjectPath == "" {
		opts.ProjectPath = "."
	}

	if err := copyPayload(payload, root, &res); err != nil {
		return res, err
	}
	if err := scaffoldTickets(root, &res); err != nil {
		return res, err
	}
	cfg, err := writeConfig(root, payloadVersion, opts, &res)
	if err != nil {
		return res, err
	}
	if err := writeBoard(root, cfg, &res); err != nil {
		return res, err
	}
	if err := writeNotes(root, &res); err != nil {
		return res, err
	}
	if err := writeTicketsReadme(root, &res); err != nil {
		return res, err
	}
	if err := injectMarker(filepath.Join(root, "AGENTS.md"), "Ticket flow", MarkerBlock(cfg), &res); err != nil {
		return res, err
	}
	if opts.Agents.Claude {
		if err := ensureSymlink(
			filepath.Join(root, filepath.FromSlash(ClaudeSkillLink)),
			ClaudeSkillTarget, &res); err != nil {
			return res, err
		}
		if opts.ClaudeLink {
			if err := ensureSymlink(filepath.Join(root, "CLAUDE.md"), "AGENTS.md", &res); err != nil {
				return res, err
			}
		} else if err := injectMarker(filepath.Join(root, "CLAUDE.md"), "Ticket flow", MarkerBlock(cfg), &res); err != nil {
			return res, err
		}
	}
	if opts.Agents.Opencode {
		if err := installOpencode(payload, root, &res); err != nil {
			return res, err
		}
	}
	if opts.Agents.Pi {
		if err := installPi(payload, root, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

// installOpencode lays down opencode.jsonc — whole-file, only when absent.
// OpenCode picks up AGENTS.md and .agents/skills/ natively, so this config
// (the docs-readability subagent + bash guardrails) is the only
// opencode-specific artifact. An existing opencode.jsonc is user-owned JSONC
// that pickle never parses or merges: it is left untouched and the template is
// returned as a note so the user can merge the blocks by hand.
func installOpencode(payload fs.FS, root string, res *Result) error {
	tmpl, err := fs.ReadFile(payload, OpencodeAsset)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", OpencodeAsset, err)
	}
	dst := filepath.Join(root, OpencodeConfigFile)
	if _, err := os.Lstat(dst); err == nil {
		res.skipped(OpencodeConfigFile + " (exists — left untouched; merge the pickle blocks by hand)")
		res.note(OpencodeConfigFile + " already exists and pickle never merges JSONC. Add the\n" +
			"docs-readability agent and the guardrail rules yourself — the template to copy from:\n\n" +
			string(tmpl))
		return nil
	}
	if err := os.WriteFile(dst, tmpl, 0o644); err != nil {
		return err
	}
	res.created(OpencodeConfigFile)
	return nil
}

// installPi writes the pi extension scaffolds (guardrails + docs-readability
// reviewer). They are pickle-owned: created or refreshed in place.
func installPi(payload fs.FS, root string, res *Result) error {
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(PiExtensionsDir)), 0o755); err != nil {
		return err
	}
	for _, f := range PiScaffolds {
		data, err := fs.ReadFile(payload, f.Asset)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", f.Asset, err)
		}
		dst := filepath.Join(root, filepath.FromSlash(f.Installed))
		if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, data) {
			res.skipped(f.Installed + " (current)")
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		res.created(f.Installed)
	}
	return nil
}

// RefreshMarkers regenerates the pickle-managed marker block in AGENTS.md (and,
// when it is a regular file rather than a CLAUDE.md -> AGENTS.md symlink,
// CLAUDE.md too) from cfg. It is the single entry point every mutator of
// pickle.toml's [[project]] registry re-injects through — pickle upgrade and
// pickle project add|remove — so the block can never describe a config state
// other than the one on disk. (pickle install injects the same block from Run
// instead: only there does --agent decide whether CLAUDE.md becomes a regular
// file or a symlink, which is the one thing this probe-what-exists policy
// cannot infer.) The block interior is
// pickle-owned; hand-written content belongs outside the markers (MarkerBlock's
// doc comment).
func RefreshMarkers(root string, cfg *config.Config) (Result, error) {
	var res Result
	if err := injectMarker(filepath.Join(root, "AGENTS.md"), "Ticket flow", MarkerBlock(cfg), &res); err != nil {
		return res, err
	}
	claude := filepath.Join(root, "CLAUDE.md")
	if fi, err := os.Lstat(claude); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		if err := injectMarker(claude, "Ticket flow", MarkerBlock(cfg), &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

// legacySweep reports what sweepLegacySkill found and removed. LinkTarget is
// non-empty when the legacy skill dir was itself a self-host symlink (so the
// caller can re-create it at the new name instead of letting copyPayload
// write a real directory over it); ClaudeLink reports whether the legacy
// Claude view was present and removed, so a caller can decide whether the new
// Claude view needs (re)creating even though it never existed under the new
// name. Delete alongside sweepLegacySkill at 1.0 (T-074).
type legacySweep struct {
	Found      bool
	LinkTarget string
	ClaudeLink bool
}

// sweepLegacySkill removes the pre-brine install paths (LegacySkillDir,
// LegacyClaudeSkillLink) left by a pickle older than T-074. A legacy real
// directory is RemoveAll'd; a legacy symlink is removed and its target
// reported in LinkTarget rather than deleted-and-recopied, protecting a
// self-host arrangement (SkillLinked) from being silently converted into an
// installed copy. dryRun mirrors Uninstall's convention: labels are recorded
// and nothing is mutated. Delete this function, its call sites, the Legacy*
// constants and their tests at 1.0 (T-074).
func sweepLegacySkill(root string, dryRun bool, res *Result) (legacySweep, error) {
	var sw legacySweep

	legacyDir := filepath.Join(root, filepath.FromSlash(LegacySkillDir))
	if fi, err := os.Lstat(legacyDir); err == nil {
		sw.Found = true
		switch {
		case dryRun:
			res.removed(LegacySkillDir + " (dry-run)")
		case fi.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(legacyDir)
			if err != nil {
				return sw, fmt.Errorf("read legacy skill symlink: %w", err)
			}
			if err := os.Remove(legacyDir); err != nil {
				return sw, fmt.Errorf("remove legacy skill symlink: %w", err)
			}
			sw.LinkTarget = target
			res.removed(LegacySkillDir + " (symlink)")
		default:
			if err := os.RemoveAll(legacyDir); err != nil {
				return sw, fmt.Errorf("remove legacy skill dir: %w", err)
			}
			res.removed(LegacySkillDir + "/")
		}
	}

	legacyClaudeLink := filepath.Join(root, filepath.FromSlash(LegacyClaudeSkillLink))
	if _, err := os.Lstat(legacyClaudeLink); err == nil {
		sw.Found = true
		sw.ClaudeLink = true
		if dryRun {
			res.removed(LegacyClaudeSkillLink + " (dry-run)")
		} else {
			if err := os.Remove(legacyClaudeLink); err != nil {
				return sw, fmt.Errorf("remove legacy claude skill symlink: %w", err)
			}
			res.removed(LegacyClaudeSkillLink)
		}
	}

	return sw, nil
}

// Upgrade refreshes the installed skill payload and the AGENTS.md/CLAUDE.md
// marker block(s) to payloadVersion, and stamps payloadVersion into pickle.toml
// by rewriting that single line, leaving the rest of the file (comments
// included) untouched. The skill directory, by contrast, is pickle-owned and
// replaced wholesale.
// It never reads or writes anything under tickets/ or the board. Idempotent:
// re-running at the current version still refreshes payload/markers (so drift
// is corrected) and reports the version as unchanged rather than erroring.
// It also sweeps away any pre-brine install left by an older pickle
// (sweepLegacySkill, T-074) before refreshing the current-name payload.
func Upgrade(payload fs.FS, root, payloadVersion string) (Result, error) {
	var res Result

	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		return res, err
	}

	// Sweep the legacy install first, so the refresh below writes into a tree
	// with no stale duplicate under the old name.
	swept, err := sweepLegacySkill(root, false, &res)
	if err != nil {
		return res, err
	}

	// Refresh the skill payload: a real dir is wiped and re-copied so files
	// removed from the new payload don't linger; a self-host symlink is left
	// alone (copyPayload already skips it via the Lstat/ModeSymlink guard). A
	// legacy self-host symlink is re-created at the new name instead — the
	// same protection, one name later.
	dst := filepath.Join(root, filepath.FromSlash(SkillDir))
	switch {
	case swept.LinkTarget != "":
		if err := ensureSymlink(dst, swept.LinkTarget, &res); err != nil {
			return res, err
		}
	case !SkillLinked(root):
		if err := os.RemoveAll(dst); err != nil {
			return res, fmt.Errorf("refresh skill payload: %w", err)
		}
	}
	if err := copyPayload(payload, root, &res); err != nil {
		return res, err
	}

	refreshed, err := RefreshMarkers(root, cfg)
	if err != nil {
		return res, err
	}
	res.Created = append(res.Created, refreshed.Created...)
	res.Skipped = append(res.Skipped, refreshed.Skipped...)

	// The new Claude view is (re)created when it already exists under the new
	// name, or when the legacy Claude view just got swept away — an upgraded
	// legacy install must not lose its Claude view entirely.
	claudeLink := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if _, err := os.Lstat(claudeLink); err == nil || swept.ClaudeLink {
		if err := ensureSymlink(claudeLink, ClaudeSkillTarget, &res); err != nil {
			return res, err
		}
	}

	// Agent scaffolds: install choices are not persisted anywhere, so upgrade
	// probes the filesystem (T-006 decision D6) — it refreshes exactly the pi
	// files that are already present. opencode.jsonc is user-owned after
	// creation and is never touched.
	for _, f := range PiScaffolds {
		dst := filepath.Join(root, filepath.FromSlash(f.Installed))
		if _, err := os.Lstat(dst); err != nil {
			continue // not installed — not upgrade's business
		}
		data, err := fs.ReadFile(payload, f.Asset)
		if err != nil {
			return res, fmt.Errorf("read embedded %s: %w", f.Asset, err)
		}
		if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, data) {
			res.skipped(f.Installed + " (current)")
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return res, err
		}
		res.created(f.Installed + " (refreshed)")
	}

	// Pre-commit hook: refresh a pickle-owned shim an older binary wrote. Never
	// installs one that is absent (the guard is opt-in) and never touches a
	// foreign hook. A tree that is not a git repository is a no-op, not an error.
	if hres, err := hook.Refresh(root); err != nil {
		return res, err
	} else if hres.Changed {
		res.created(hookLabel(root, hres.Path) + " (refreshed)")
	}

	if cfg.PayloadVersion == payloadVersion {
		res.skipped(config.FileName + " (already at " + payloadVersion + ")")
		return res, nil
	}
	// Edit the one line rather than re-rendering: pickle.toml is the user's
	// file, and upgrade has no business touching their comments.
	if err = config.SetPayloadVersionInPlace(cfg.Path(), payloadVersion); err != nil {
		return res, err
	}
	// Report the stamp only once it is on disk. A successful write is not the
	// same as an achieved effect, and a version this command claims to have set
	// but did not would stay wrong on every later run.
	if err := verifyStampedVersion(cfg.Path(), payloadVersion); err != nil {
		return res, err
	}
	res.created(config.FileName + " (payload_version -> " + payloadVersion + ")")
	return res, nil
}

// verifyStampedVersion re-reads the config and confirms it now carries want.
func verifyStampedVersion(path, want string) error {
	after, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("re-read %s after stamping %s: %w", config.FileName, want, err)
	}
	if after.PayloadVersion != want {
		return fmt.Errorf("%s still reads payload_version = %q after stamping %q; set it by hand",
			config.FileName, after.PayloadVersion, want)
	}
	return nil
}

// UninstallOptions configures a single uninstall run.
type UninstallOptions struct {
	// DryRun computes and records what would be removed/stripped without
	// mutating the tree.
	DryRun bool
}

// Uninstall removes the installed skill dir, Claude symlinks and agent
// scaffolds, and strips the pickle marker block(s), leaving tickets/ and
// pickle.toml untouched so a later install/upgrade re-attaches cleanly. The
// payload is needed only to recognise a pristine opencode.jsonc (removed iff
// still byte-identical to the shipped template — a user-modified one is theirs
// and stays). Idempotent: re-running on an already-clean tree reports nothing
// removed.
func Uninstall(payload fs.FS, root string, opts UninstallOptions) (Result, error) {
	var res Result

	// Sweep any pre-brine install left by an older pickle, so a new binary can
	// still fully remove an install made by an old one (T-074).
	if _, err := sweepLegacySkill(root, opts.DryRun, &res); err != nil {
		return res, err
	}

	skillDir := filepath.Join(root, filepath.FromSlash(SkillDir))
	if fi, err := os.Lstat(skillDir); err == nil {
		if opts.DryRun {
			res.removed(SkillDir + " (dry-run)")
		} else if fi.Mode()&os.ModeSymlink != 0 {
			// Never RemoveAll a symlink: that would delete the real skill/ tree
			// it points at (self-host). Remove the link itself only.
			if err := os.Remove(skillDir); err != nil {
				return res, fmt.Errorf("remove skill symlink: %w", err)
			}
			res.removed(SkillDir + " (symlink)")
		} else {
			if err := os.RemoveAll(skillDir); err != nil {
				return res, fmt.Errorf("remove skill dir: %w", err)
			}
			res.removed(SkillDir + "/")
		}
	}

	claudeLink := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if _, err := os.Lstat(claudeLink); err == nil {
		if opts.DryRun {
			res.removed(ClaudeSkillLink + " (dry-run)")
		} else {
			if err := os.Remove(claudeLink); err != nil {
				return res, fmt.Errorf("remove claude skill symlink: %w", err)
			}
			res.removed(ClaudeSkillLink)
		}
	}

	if err := uninstallMarkerFile(filepath.Join(root, "AGENTS.md"), opts, &res); err != nil {
		return res, err
	}
	if err := uninstallMarkerFile(filepath.Join(root, "CLAUDE.md"), opts, &res); err != nil {
		return res, err
	}

	// Pi scaffolds (pickle-owned): removed when present. The .pi tree is pruned
	// only when left empty — user-written extensions keep their directories.
	for _, f := range PiScaffolds {
		dst := filepath.Join(root, filepath.FromSlash(f.Installed))
		if _, err := os.Lstat(dst); err != nil {
			continue
		}
		if opts.DryRun {
			res.removed(f.Installed + " (dry-run)")
			continue
		}
		if err := os.Remove(dst); err != nil {
			return res, fmt.Errorf("remove %s: %w", f.Installed, err)
		}
		res.removed(f.Installed)
	}
	if !opts.DryRun {
		// os.Remove fails on a non-empty dir — exactly the contract wanted here.
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(PiExtensionsDir)))
		_ = os.Remove(filepath.Join(root, ".pi"))
	}

	// Pre-commit hook (pickle-owned, recognised by its marker). Absent, foreign
	// and non-git trees are all normal here: uninstall runs on trees that never
	// had a hook, so only an owned one is reported.
	if hres, err := hook.Uninstall(root, opts.DryRun); err != nil {
		return res, err
	} else if hres.Would {
		res.removed(hookLabel(root, hres.Path) + " (dry-run)")
	} else if hres.Changed {
		res.removed(hookLabel(root, hres.Path))
	} else if hres.Kind == hook.KindForeign {
		res.skipped(hookLabel(root, hres.Path) + " (not pickle's, left in place)")
	}

	// opencode.jsonc: removed only while still byte-identical to the shipped
	// template. Anything else is the user's config — pickle wrote it whole at
	// most once and never merges, so an edited file must survive.
	ocDst := filepath.Join(root, OpencodeConfigFile)
	if cur, err := os.ReadFile(ocDst); err == nil {
		tmpl, terr := fs.ReadFile(payload, OpencodeAsset)
		if terr == nil && bytes.Equal(cur, tmpl) {
			if opts.DryRun {
				res.removed(OpencodeConfigFile + " (dry-run)")
			} else if err := os.Remove(ocDst); err != nil {
				return res, fmt.Errorf("remove %s: %w", OpencodeConfigFile, err)
			} else {
				res.removed(OpencodeConfigFile)
			}
		} else {
			res.skipped(OpencodeConfigFile + " (user-modified, left in place)")
		}
	}

	return res, nil
}

// uninstallMarkerFile handles one AGENTS.md/CLAUDE.md: a symlink (CLAUDE.md ->
// AGENTS.md) is removed outright; a regular file has its marker block stripped
// (leaving the rest of the file, and the file itself, in place). Absent files
// are skipped.
func uninstallMarkerFile(path string, opts UninstallOptions, res *Result) error {
	rel := filepath.Base(path)
	fi, err := os.Lstat(path)
	if err != nil {
		return nil // absent — nothing to do
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		if opts.DryRun {
			res.removed(rel + " (symlink, dry-run)")
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s symlink: %w", rel, err)
		}
		res.removed(rel + " (symlink)")
		return nil
	}
	if opts.DryRun {
		existing, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, _, ok := markerSpan(string(existing)); ok {
			res.removed(rel + " (marker, dry-run)")
		} else {
			res.skipped(rel + " (no marker)")
		}
		return nil
	}
	return stripMarker(path, res)
}

// copyPayload writes the embedded skill tree into root/.agents/skills/brine
// as real files. If that path already exists as a symlink (a dev/self-host link),
// it is left untouched.
func copyPayload(payload fs.FS, root string, res *Result) error {
	dst := filepath.Join(root, filepath.FromSlash(SkillDir))
	if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		res.skipped(SkillDir + " (existing symlink)")
		return nil
	}
	sub, err := fs.Sub(payload, "skill")
	if err != nil {
		return fmt.Errorf("locate embedded skill: %w", err)
	}
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copy payload: %w", err)
	}
	res.created(SkillDir + "/")
	return nil
}

// scaffoldTickets creates every status dir the definition names, each with a
// .gitkeep so empty dirs survive git. Install always scaffolds the default
// flow (brine): there is no flow-selection option yet, and pickle.toml has not
// been written at this point in Run, so there is no configured flow to resolve.
func scaffoldTickets(root string, res *Result) error {
	def := flow.Default()
	for _, s := range def.States() {
		dir := filepath.Join(root, "tickets", s.Dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		keep := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(keep); err == nil {
			continue
		}
		if err := os.WriteFile(keep, nil, 0o644); err != nil {
			return err
		}
	}
	res.created(fmt.Sprintf("tickets/ (%d status dirs)", len(def.States())))
	return nil
}

// writeBoard seeds tickets/BOARD.md as a fresh render of the (normally empty)
// ticket tree — only if absent, to preserve instance data. The board is a
// generated artifact (T-044): there is no skeleton to copy.
func writeBoard(root string, cfg *config.Config, res *Result) error {
	dst := filepath.Join(root, "tickets", "BOARD.md")
	if _, err := os.Stat(dst); err == nil {
		res.skipped("tickets/BOARD.md (exists)")
		return nil
	}
	def := flow.ForName(cfg.FlowName())
	tickets, _ := ticket.LoadAll(def, root)
	out := board.Render(def, tickets, cfg, time.Now().Format("2006-01-02"))
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil {
		return err
	}
	res.created("tickets/BOARD.md")
	return nil
}

// notesScaffold is the initial tickets/NOTES.md: the home for the hand-written
// prose the generated board cannot carry.
const notesScaffold = "# Notes\n\n" +
	"Hand-written planning notes live here — triage records, parked-ticket notes,\n" +
	"cross-ticket decisions, dependency rationale. `BOARD.md` is generated from the\n" +
	"ticket files (run `pickle board sync`), so nothing hand-written survives there.\n"

// writeNotes scaffolds tickets/NOTES.md — only if absent, like the board.
func writeNotes(root string, res *Result) error {
	dst := filepath.Join(root, "tickets", "NOTES.md")
	if _, err := os.Stat(dst); err == nil {
		res.skipped("tickets/NOTES.md (exists)")
		return nil
	}
	if err := os.WriteFile(dst, []byte(notesScaffold), 0o644); err != nil {
		return err
	}
	res.created("tickets/NOTES.md")
	return nil
}

// writeTicketsReadme writes the short pointer README into tickets/ — only if absent.
func writeTicketsReadme(root string, res *Result) error {
	dst := filepath.Join(root, "tickets", "README.md")
	if _, err := os.Stat(dst); err == nil {
		res.skipped("tickets/README.md (exists)")
		return nil
	}
	if err := os.WriteFile(dst, []byte(ticketsReadme), 0o644); err != nil {
		return err
	}
	res.created("tickets/README.md")
	return nil
}

// writeConfig writes pickle.toml with the first child registered — only if absent.
// It returns a config to render the marker block from (loaded from disk when the
// file already existed).
func writeConfig(root, payloadVersion string, opts Options, res *Result) (*config.Config, error) {
	dst := filepath.Join(root, config.FileName)
	if _, err := os.Stat(dst); err == nil {
		res.skipped(config.FileName + " (exists)")
		return config.Load(dst)
	}
	cfg := &config.Config{
		PayloadVersion: payloadVersion,
		Commit: config.CommitPolicy{
			OverarchingAuto:   true,
			ChildPublishGated: true,
		},
	}
	if err := cfg.AddProject(config.Project{
		Name:  opts.ProjectName,
		Path:  opts.ProjectPath,
		Build: opts.Build,
		Test:  opts.Test,
		Lint:  opts.Lint,
		Docs:  opts.Docs,
	}); err != nil {
		return nil, err
	}
	if err := cfg.Save(dst); err != nil {
		return nil, err
	}
	res.created(config.FileName)
	return cfg, nil
}

// ensureSymlink creates or repairs a relative symlink at link pointing to target.
// It errors if a real (non-symlink) file or directory blocks the path.
func ensureSymlink(link, target string, res *Result) error {
	rel := link
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, link); err == nil {
			rel = r
		}
	}
	if fi, err := os.Lstat(link); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			if cur, _ := os.Readlink(link); cur == target {
				res.skipped(rel + " (symlink ok)")
				return nil
			}
			if err := os.Remove(link); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%s exists and is not a symlink; refusing to replace", rel)
		}
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(target, link); err != nil {
		return err
	}
	res.created(rel + " -> " + target)
	return nil
}

// markerSpan locates the pickle-managed block within text, returning the byte
// offsets of MarkerBegin and MarkerEnd. ok is false when either delimiter is
// absent, or when they are present but out of order (MarkerEnd before
// MarkerBegin) — a malformed file has no marker block, not a backwards one.
// This is the single predicate every marker-scanning site routes through, so a
// dry-run and its real run can never disagree about whether a block exists.
func markerSpan(text string) (bi, ei int, ok bool) {
	bi = strings.Index(text, MarkerBegin)
	ei = strings.Index(text, MarkerEnd)
	return bi, ei, bi >= 0 && ei > bi
}

// InstalledMarkerBody reads path and returns the trimmed content of its
// pickle-managed block (between MarkerBegin and MarkerEnd, exclusive, with
// leading/trailing newlines trimmed) — exactly what MarkerBlock would need to
// equal for the file to be current. ok is false when the file is unreadable or
// carries no valid marker pair (see markerSpan); doctor uses this for both the
// presence check and the drift comparison.
func InstalledMarkerBody(path string) (body string, ok bool) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	text := string(existing)
	bi, ei, spanOK := markerSpan(text)
	if !spanOK {
		return "", false
	}
	return strings.Trim(text[bi+len(MarkerBegin):ei], "\n"), true
}

// injectMarker replaces the pickle-managed block in path (between markerBegin and
// markerEnd) with block, or appends it if the markers are absent. When the file
// does not exist it is created with a minimal title header.
func injectMarker(path, title, block string, res *Result) error {
	rel := filepath.Base(path)
	wrapped := MarkerBegin + "\n" + block + "\n" + MarkerEnd

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		content := "# " + title + "\n\n" + wrapped + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
		res.created(rel + " (marker)")
		return nil
	}
	if err != nil {
		return err
	}

	text := string(existing)
	if bi, ei, ok := markerSpan(text); ok {
		out := text[:bi] + wrapped + text[ei+len(MarkerEnd):]
		if out == text {
			res.skipped(rel + " (marker current)")
			return nil
		}
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			return err
		}
		res.created(rel + " (marker updated)")
		return nil
	}

	sep := "\n"
	if !strings.HasSuffix(text, "\n") {
		sep = "\n\n"
	} else if !strings.HasSuffix(text, "\n\n") {
		sep = "\n"
	}
	out := text + sep + wrapped + "\n"
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	res.created(rel + " (marker appended)")
	return nil
}

// stripMarker removes the pickle-managed block (MarkerBegin…MarkerEnd, inclusive)
// from path, along with any blank line(s) left orphaned immediately around it —
// the inverse of injectMarker. An absent file, or one without a marker pair, is
// left untouched (skipped, not an error). The file itself is never deleted, even
// if stripping the marker leaves it empty.
func stripMarker(path string, res *Result) error {
	rel := filepath.Base(path)
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		res.skipped(rel + " (absent)")
		return nil
	}
	if err != nil {
		return err
	}

	text := string(existing)
	bi, ei, ok := markerSpan(text)
	if !ok {
		res.skipped(rel + " (no marker)")
		return nil
	}
	end := ei + len(MarkerEnd)

	before := strings.TrimRight(text[:bi], "\n")
	after := strings.TrimLeft(text[end:], "\n")

	var out string
	switch {
	case before == "" && after == "":
		out = ""
	case before == "":
		out = after
	case after == "":
		out = before + "\n"
	default:
		out = before + "\n\n" + after
	}

	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	res.removed(rel + " (marker stripped)")
	return nil
}

// MarkerBlock builds the pickle-managed AGENTS.md/CLAUDE.md content from cfg. It
// is exported so internal/doctor can render the canonical form to compare
// against what is actually installed (drift detection).
//
// Everything the block states about the children — their commands, branch
// prefixes, WIP limits — is rendered from pickle.toml rather than hardcoded, so
// regenerating the block cannot silently drop project-specific facts. Content
// that is *not* derivable from the config does not belong between the markers;
// it belongs in the surrounding file, which pickle never touches.
func MarkerBlock(cfg *config.Config) string {
	var names []string
	for _, p := range cfg.Projects {
		names = append(names, "`"+p.Name+"`")
	}
	children := strings.Join(names, ", ")
	if children == "" {
		children = "(none yet — register with `pickle project add`)"
	}

	// One line per child throughout: uniform for any number of children, and
	// no wording that only reads correctly when there happens to be one.
	def := flow.ForName(cfg.FlowName())
	wipStates := def.WIPStates()
	var commands, branches, wip strings.Builder
	for _, p := range cfg.Projects {
		var cmds []string
		for _, c := range []struct{ label, cmd string }{
			{"build", p.Build}, {"test", p.Test}, {"lint", p.Lint}, {"docs", p.Docs},
		} {
			if c.cmd != "" {
				cmds = append(cmds, c.label+" `"+c.cmd+"`")
			}
		}
		if len(cmds) > 0 {
			fmt.Fprintf(&commands, "\n  - `%s`: %s", p.Name, strings.Join(cmds, " · "))
		}
		fmt.Fprintf(&branches, "\n  - `%s`: `%s%s-NNN-<slug>`", p.Name, p.BranchPrefix, p.Prefix())
		var limits []string
		for _, s := range wipStates {
			if limit, ok := p.WIPLimitFor(s.WIPKey); ok {
				limits = append(limits, fmt.Sprintf("`%s/` ≤ %d", s.Dir, limit))
			}
		}
		fmt.Fprintf(&wip, "\n  - `%s`: %s", p.Name, strings.Join(limits, " · "))
	}
	commandsBullet := ""
	if commands.Len() > 0 {
		commandsBullet = "\n- **Commands** (each child's, from `pickle.toml`):" + commands.String()
	}

	childPolicy := "Child-projects are **publish-gated**: local WIP commits are encouraged;\n" +
		"  **no push / no merge request without explicit user approval**; after approval, finalize\n" +
		"  (squash or keep history) + push + open the MR — **merging is always the human's**."
	if !cfg.Commit.ChildPublishGated {
		childPolicy = "Child-projects are **not publish-gated**: commit and push as the work\n" +
			"  needs, and open the merge request when it is ready — **merging is always the human's**."
	}
	overarching := "Overarching bookkeeping (tickets, board, docs) may be committed automatically,\n" +
		"  always with **explicit pathspecs**"
	if !cfg.Commit.OverarchingAuto {
		overarching = "Overarching bookkeeping (tickets, board, docs) is committed only when the\n" +
			"  user asks, and always with **explicit pathspecs**"
	}

	return "## Brine (start here)\n" +
		"\n" +
		"**Start at [`tickets/BOARD.md`](tickets/BOARD.md)** — the generated index of every ticket by\n" +
		"status. No feature is built directly from a chat message or a raw idea — work enters only as a\n" +
		"ticket whose Implementation Plan has met the READY gate. A *review finding* is different: it\n" +
		"earns a **disposition** (rules §5), and most are resolved without a new ticket.\n" +
		"\n" +
		"- The flow engine is the **brine skill** at `.agents/skills/brine/`. It holds\n" +
		"  the rules (`resources/tickets-README.md`), the ticket template\n" +
		"  (`resources/TEMPLATE.md`), and the review protocol\n" +
		"  (`resources/review-protocol.md`). Claude Code sees it via `.claude/skills/brine`.\n" +
		"  The directory is pickle-owned — `pickle upgrade` replaces it wholesale, so keep\n" +
		"  hand-written notes outside it.\n" +
		"- Triggers: \"make it a ticket\", \"refine ticket T-NNN\", \"implement ticket T-NNN\", \"rework ticket\n" +
		"  T-NNN\", \"validate ticket T-NNN\" (or \"review ticket T-NNN\"), \"audit the board\".\n" +
		"\n" +
		"### Project configuration\n" +
		"\n" +
		"- **Build target.** Every ticket targets one registered child-project via `project:`\n" +
		"  frontmatter (`pickle project list`). Registered child-projects: " + children + "." +
		commandsBullet + "\n" +
		"- **Branch & commit.** Conventional Commits with the **ticket id in brackets at the end of\n" +
		"  the subject** (e.g. `feat(cli): add board audit (T-2)`) for child-project code. Ticket/board\n" +
		"  bookkeeping uses its own `board: T-NNN <verb phrase>` form instead — grammar and scope in\n" +
		"  the rules §0. Branch per child:" + branches.String() + "\n" +
		"- **WIP limits** (per child):" + wip.String() + "\n" +
		"- **Commit policy.** " + childPolicy + "\n" +
		"  " + overarching + " (`git add <paths>`, never `git add -A`/`.`).\n" +
		"- **Where commits land.** Code goes on the child's feature branch; **ticket and board\n" +
		"  bookkeeping is committed on the base branch**, never on a feature branch — a squash-merge\n" +
		"  folds or drops it and the board then disagrees with the tickets it indexes. This covers a\n" +
		"  review's own moves too, and it is why a reviewer on a feature branch reads the ticket from\n" +
		"  the base branch. `pickle hooks install` enforces it locally, once per clone (bypass a\n" +
		"  single commit with `git commit --no-verify`).\n" +
		"\n" +
		"### Board rule\n" +
		"\n" +
		"`tickets/BOARD.md` is **generated** — regenerated wholesale from the ticket files by\n" +
		"`pickle ticket new`, `pickle ticket move` and `pickle board sync`. **Never edit it by\n" +
		"hand**; hand-written planning notes go in `tickets/NOTES.md`. Every ticket move = move\n" +
		"the file + one dated `## History` line, and the board regenerates. Prefer\n" +
		"`pickle ticket move` — it does all of it atomically."
}

const ticketsReadme = "# `tickets/` — the ticket-based feature flow\n\n" +
	"The live board. Every change flows through one artifact per feature: a **ticket** — a\n" +
	"markdown file whose status is the directory it lives in, targeting a registered\n" +
	"child-project via `project:` frontmatter, with an append-only History.\n\n" +
	"- **Start at [`BOARD.md`](BOARD.md)** — the generated index of every ticket by status.\n" +
	"  It is regenerated by `pickle ticket new`, `pickle ticket move` and `pickle board sync`;\n" +
	"  never edit it by hand.\n" +
	"- **Hand-written planning notes live in [`NOTES.md`](NOTES.md)** — the board cannot\n" +
	"  carry them.\n" +
	"- **The rules, ticket template, and review protocol live in the brine skill:**\n" +
	"  - rules: `.agents/skills/brine/resources/tickets-README.md` (so `§N` references\n" +
	"    resolve there)\n" +
	"  - template: `.agents/skills/brine/resources/TEMPLATE.md`\n" +
	"  - review protocol: `.agents/skills/brine/resources/review-protocol.md`\n" +
	"- **Build target:** every ticket targets a registered child-project (see `../pickle.toml`).\n\n" +
	"This directory holds **instance data only** (the tickets, the generated board, the notes).\n" +
	"See `../AGENTS.md` for the project configuration and commit policy.\n"
