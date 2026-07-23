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

	"pickle/internal/config"
	"pickle/internal/ticket"
)

// Marker delimiters for the injected AGENTS.md/CLAUDE.md block. Anything between
// them is pickle-managed and replaced on re-run; text outside is preserved.
const (
	markerBegin = "<!-- pickle:begin -->"
	markerEnd   = "<!-- pickle:end -->"

	skillDir = ".agents/skills/ticket-flow"
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

// Result records what the run created vs. left in place, for the CLI summary.
type Result struct {
	Created []string
	Skipped []string
}

func (r *Result) created(f string) { r.Created = append(r.Created, f) }
func (r *Result) skipped(f string) { r.Skipped = append(r.Skipped, f) }

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
	if err := writeBoard(payload, root, opts.ProjectName, &res); err != nil {
		return res, err
	}
	if err := writeTicketsReadme(root, &res); err != nil {
		return res, err
	}
	cfg, err := writeConfig(root, payloadVersion, opts, &res)
	if err != nil {
		return res, err
	}
	if err := injectMarker(filepath.Join(root, "AGENTS.md"), "Ticket flow", markerBlock(cfg), &res); err != nil {
		return res, err
	}
	if opts.Claude {
		if err := ensureSymlink(
			filepath.Join(root, ".claude", "skills", "ticket-flow"),
			"../../.agents/skills/ticket-flow", &res); err != nil {
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

// copyPayload writes the embedded skill tree into root/.agents/skills/ticket-flow
// as real files. If that path already exists as a symlink (a dev/self-host link),
// it is left untouched.
func copyPayload(payload fs.FS, root string, res *Result) error {
	dst := filepath.Join(root, filepath.FromSlash(skillDir))
	if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		res.skipped(skillDir + " (existing symlink)")
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
	res.created(skillDir + "/")
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

// writeBoard seeds tickets/BOARD.md from the embedded skeleton (substituting the
// date and the first child's name) — only if absent, to preserve instance data.
func writeBoard(payload fs.FS, root, child string, res *Result) error {
	dst := filepath.Join(root, "tickets", "BOARD.md")
	if _, err := os.Stat(dst); err == nil {
		res.skipped("tickets/BOARD.md (exists)")
		return nil
	}
	data, err := fs.ReadFile(payload, "skill/resources/BOARD.md")
	if err != nil {
		return fmt.Errorf("read embedded BOARD.md: %w", err)
	}
	out := strings.ReplaceAll(string(data), "<YYYY-MM-DD>", time.Now().Format("2006-01-02"))
	out = strings.ReplaceAll(out, "<child-project>", child)
	if err := os.WriteFile(dst, []byte(out), 0o644); err != nil {
		return err
	}
	res.created("tickets/BOARD.md")
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
	wrapped := markerBegin + "\n" + block + "\n" + markerEnd

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
	bi := strings.Index(text, markerBegin)
	ei := strings.Index(text, markerEnd)
	if bi >= 0 && ei > bi {
		out := text[:bi] + wrapped + text[ei+len(markerEnd):]
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

// markerBlock builds the pickle-managed AGENTS.md/CLAUDE.md content from cfg.
func markerBlock(cfg *config.Config) string {
	var names []string
	for _, p := range cfg.Projects {
		names = append(names, "`"+p.Name+"`")
	}
	children := strings.Join(names, ", ")
	if children == "" {
		children = "(none yet — register with `pickle project add`)"
	}
	return fmt.Sprintf(`## Ticket flow (start here)

**Start at [`+"`tickets/BOARD.md`"+`](tickets/BOARD.md)** — the live index of every ticket by
status. Nothing is built directly from a chat message, an idea, or a review finding — only from
a ticket whose Implementation Plan has met the READY gate.

- The flow engine is the **ticket-flow skill** at `+"`.agents/skills/ticket-flow/`"+`. It holds
  the rules (`+"`resources/tickets-README.md`"+`), the ticket template
  (`+"`resources/TEMPLATE.md`"+`), and the review protocol
  (`+"`resources/review-protocol.md`"+`). Claude Code sees it via `+"`.claude/skills/ticket-flow`"+`.
- Triggers: "make it a ticket", "refine ticket T-NNN", "implement ticket T-NNN", "rework ticket
  T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board".

### Project configuration

- **Build target.** Every ticket targets one registered child-project via `+"`project:`"+`
  frontmatter (`+"`pickle project list`"+`; commands, branch prefix, and WIP limits live in
  `+"`pickle.toml`"+`). Registered child-projects: %s.
- **Branch & commit.** `+"`<branch_prefix>T-NNN-<slug>`"+` branches; Conventional Commits with the
  **ticket id in brackets at the end of the subject** (e.g. `+"`feat(cli): add board audit (T-2)`"+`).
- **Commit policy.** Child-projects are **publish-gated**: local WIP commits are encouraged; **no
  push / no merge request without explicit user approval**; merging is always the human's.
  Overarching bookkeeping (tickets, board, docs) may be committed automatically, always with
  **explicit pathspecs** (`+"`git add <paths>`"+`, never `+"`git add -A`/`.`"+`).

### Board rule

Every ticket move = move the file + one dated `+"`## History`"+` line + one `+"`tickets/BOARD.md`"+`
edit, in the same change. A move that doesn't touch the board is a bug. Prefer
`+"`pickle ticket move`"+` once available; otherwise do the three edits by hand.`, children)
}

const ticketsReadme = "# `tickets/` — the ticket-based feature flow\n\n" +
	"The live board. Every change flows through one artifact per feature: a **ticket** — a\n" +
	"markdown file whose status is the directory it lives in, targeting a registered\n" +
	"child-project via `project:` frontmatter, with an append-only History.\n\n" +
	"- **Start at [`BOARD.md`](BOARD.md)** — the maintained index of every ticket by status.\n" +
	"- **The rules, ticket template, and review protocol live in the ticket-flow skill:**\n" +
	"  - rules: `.agents/skills/ticket-flow/resources/tickets-README.md` (so `§N` references\n" +
	"    resolve there)\n" +
	"  - template: `.agents/skills/ticket-flow/resources/TEMPLATE.md`\n" +
	"  - review protocol: `.agents/skills/ticket-flow/resources/review-protocol.md`\n" +
	"- **Build target:** every ticket targets a registered child-project (see `../pickle.toml`).\n\n" +
	"This directory holds **instance data only** (the board + the tickets). See `../AGENTS.md`\n" +
	"for the project configuration and commit policy.\n"
