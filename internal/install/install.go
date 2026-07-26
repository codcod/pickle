// Package install implements `pickle install`: it lays the ticket flow into a
// project — the embedded skill payload, the tickets/ board scaffold, the
// AGENTS.md/CLAUDE.md marker blocks, the Claude view symlink, and a pickle.toml
// registering the first child-project. It is per-project (never writes to ~/ or
// outside root), idempotent, and safe to re-run: the payload and markers are
// refreshed in place while instance data (BOARD.md, tickets, pickle.toml) is
// preserved once present.
package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// Marker delimiters for the injected AGENTS.md/CLAUDE.md block. Anything between
// them is pickle-managed and replaced on re-run; text outside is preserved.
const (
	MarkerBegin = "<!-- pickle:begin -->"
	MarkerEnd   = "<!-- pickle:end -->"

	// SkillDir is the installed skill payload location (relative to the project root).
	SkillDir = ".agents/skills/ticket-flow"
	// ClaudeSkillLink is the Claude Code view of the skill; ClaudeSkillTarget is the
	// relative symlink target it points at (SkillDir, from inside .claude/skills/).
	ClaudeSkillLink   = ".claude/skills/ticket-flow"
	ClaudeSkillTarget = "../../.agents/skills/ticket-flow"
)

// Options configures a single install run.
type Options struct {
	ProjectName string // first child-project name (required; caller defaults it)
	ProjectPath string // first child path, relative to root (defaults to ".")
	Build       string // optional child commands
	Test        string
	Lint        string
	Docs        string
	Claude      bool // install the Claude view (.claude symlink) + CLAUDE.md
	ClaudeLink  bool // make CLAUDE.md a symlink to AGENTS.md instead of a marker block
}

// Result records what the run created, left in place, or removed, for the CLI
// summary. Removed is populated by Uninstall (and, under UninstallOptions.DryRun,
// records what would be removed without mutating the tree).
type Result struct {
	Created []string
	Skipped []string
	Removed []string
}

func (r *Result) created(f string) { r.Created = append(r.Created, f) }
func (r *Result) skipped(f string) { r.Skipped = append(r.Skipped, f) }
func (r *Result) removed(f string) { r.Removed = append(r.Removed, f) }

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
	if err := injectMarker(filepath.Join(root, "AGENTS.md"), "Ticket flow", markerBlock(cfg), &res); err != nil {
		return res, err
	}
	if opts.Claude {
		if err := ensureSymlink(
			filepath.Join(root, filepath.FromSlash(ClaudeSkillLink)),
			ClaudeSkillTarget, &res); err != nil {
			return res, err
		}
		if opts.ClaudeLink {
			if err := ensureSymlink(filepath.Join(root, "CLAUDE.md"), "AGENTS.md", &res); err != nil {
				return res, err
			}
		} else if err := injectMarker(filepath.Join(root, "CLAUDE.md"), "Ticket flow", markerBlock(cfg), &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

// Upgrade refreshes the installed skill payload and the AGENTS.md/CLAUDE.md
// marker block(s) to payloadVersion, and stamps payloadVersion into pickle.toml
// by rewriting that single line, leaving the rest of the file (comments
// included) untouched. The skill directory, by contrast, is pickle-owned and
// replaced wholesale.
// It never reads or writes anything under tickets/ or the board. Idempotent:
// re-running at the current version still refreshes payload/markers (so drift
// is corrected) and reports the version as unchanged rather than erroring.
func Upgrade(payload fs.FS, root, payloadVersion string) (Result, error) {
	var res Result

	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		return res, err
	}

	// Refresh the skill payload: a real dir is wiped and re-copied so files
	// removed from the new payload don't linger; a self-host symlink is left
	// alone (copyPayload already skips it via the Lstat/ModeSymlink guard).
	dst := filepath.Join(root, filepath.FromSlash(SkillDir))
	if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		if err := os.RemoveAll(dst); err != nil {
			return res, fmt.Errorf("refresh skill payload: %w", err)
		}
	}
	if err := copyPayload(payload, root, &res); err != nil {
		return res, err
	}

	if err := injectMarker(filepath.Join(root, "AGENTS.md"), "Ticket flow", markerBlock(cfg), &res); err != nil {
		return res, err
	}
	claude := filepath.Join(root, "CLAUDE.md")
	if fi, err := os.Lstat(claude); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		if err := injectMarker(claude, "Ticket flow", markerBlock(cfg), &res); err != nil {
			return res, err
		}
	}
	claudeLink := filepath.Join(root, filepath.FromSlash(ClaudeSkillLink))
	if _, err := os.Lstat(claudeLink); err == nil {
		if err := ensureSymlink(claudeLink, ClaudeSkillTarget, &res); err != nil {
			return res, err
		}
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

// Uninstall removes the installed skill dir + Claude symlinks and strips the
// pickle marker block(s), leaving tickets/ and pickle.toml untouched so a later
// install/upgrade re-attaches cleanly. Idempotent: re-running on an already-clean
// tree reports nothing removed.
func Uninstall(root string, opts UninstallOptions) (Result, error) {
	var res Result

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
		text := string(existing)
		if strings.Contains(text, MarkerBegin) && strings.Contains(text, MarkerEnd) {
			res.removed(rel + " (marker, dry-run)")
		} else {
			res.skipped(rel + " (no marker)")
		}
		return nil
	}
	return stripMarker(path, res)
}

// copyPayload writes the embedded skill tree into root/.agents/skills/ticket-flow
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

// scaffoldTickets creates the seven ordered status dirs, each with a .gitkeep so
// empty dirs survive git.
func scaffoldTickets(root string, res *Result) error {
	for _, s := range ticket.Statuses {
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
	res.created("tickets/ (7 status dirs)")
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
	tickets, _ := ticket.LoadAll(root)
	out := board.Render(tickets, cfg, time.Now().Format("2006-01-02"))
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
	bi := strings.Index(text, MarkerBegin)
	ei := strings.Index(text, MarkerEnd)
	if bi >= 0 && ei > bi {
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
	bi := strings.Index(text, MarkerBegin)
	ei := strings.Index(text, MarkerEnd)
	if bi < 0 || ei < bi {
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

// markerBlock builds the pickle-managed AGENTS.md/CLAUDE.md content from cfg.
//
// Everything the block states about the children — their commands, branch
// prefixes, WIP limits — is rendered from pickle.toml rather than hardcoded, so
// regenerating the block cannot silently drop project-specific facts. Content
// that is *not* derivable from the config does not belong between the markers;
// it belongs in the surrounding file, which pickle never touches.
func markerBlock(cfg *config.Config) string {
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
		fmt.Fprintf(&branches, "\n  - `%s`: `%sT-NNN-<slug>`", p.Name, p.BranchPrefix)
		fmt.Fprintf(&wip, "\n  - `%s`: `3-in-development/` ≤ %d · `4-in-review/` ≤ %d",
			p.Name, p.WIPInDevelopment, p.WIPInReview)
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

	return "## Ticket flow (start here)\n" +
		"\n" +
		"**Start at [`tickets/BOARD.md`](tickets/BOARD.md)** — the generated index of every ticket by\n" +
		"status. No feature is built directly from a chat message or a raw idea — work enters only as a\n" +
		"ticket whose Implementation Plan has met the READY gate. A *review finding* is different: it\n" +
		"earns a **disposition** (rules §5), and most are resolved without a new ticket.\n" +
		"\n" +
		"- The flow engine is the **ticket-flow skill** at `.agents/skills/ticket-flow/`. It holds\n" +
		"  the rules (`resources/tickets-README.md`), the ticket template\n" +
		"  (`resources/TEMPLATE.md`), and the review protocol\n" +
		"  (`resources/review-protocol.md`). Claude Code sees it via `.claude/skills/ticket-flow`.\n" +
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
		"  the subject** (e.g. `feat(cli): add board audit (T-2)`). Branch per child:" + branches.String() + "\n" +
		"- **WIP limits** (per child):" + wip.String() + "\n" +
		"- **Commit policy.** " + childPolicy + "\n" +
		"  " + overarching + " (`git add <paths>`, never `git add -A`/`.`).\n" +
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
	"- **The rules, ticket template, and review protocol live in the ticket-flow skill:**\n" +
	"  - rules: `.agents/skills/ticket-flow/resources/tickets-README.md` (so `§N` references\n" +
	"    resolve there)\n" +
	"  - template: `.agents/skills/ticket-flow/resources/TEMPLATE.md`\n" +
	"  - review protocol: `.agents/skills/ticket-flow/resources/review-protocol.md`\n" +
	"- **Build target:** every ticket targets a registered child-project (see `../pickle.toml`).\n\n" +
	"This directory holds **instance data only** (the tickets, the generated board, the notes).\n" +
	"See `../AGENTS.md` for the project configuration and commit policy.\n"
