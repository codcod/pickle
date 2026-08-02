// Package audit implements `pickle board audit`: a pure check of the ticket-flow
// invariants over a tickets/ tree + pickle.toml. It never prints or exits — it
// returns findings so it stays fixture-testable.
package audit

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// Result is the outcome of an audit.
type Result struct {
	NumTickets int
	Errors     []string
	Warnings   []string
}

var requiredKeys = []string{"id", "title", "project", "depends-on", "spawned-by", "impact", "complexity", "cost"}

// optionalKeys lists frontmatter keys that are legal but not required —
// validated only when present (today: T-059's family:). Kept next to
// requiredKeys so the audit's full frontmatter vocabulary has one home;
// TestFrontmatterKeysMatchTemplate (T-040) checks both lists against
// skill/resources/TEMPLATE.md, so drift between the audit and the authoring
// guide fails in CI instead of in a user's project.
var optionalKeys = []string{"family"}

// maxHistoryEntryRunes bounds a status-transition or merge History line (T-040
// D4). Measured over this repo's own 303 History entries (2026-08-01):
// transitions top out at 306 runes and merge lines at 194 — 400 leaves ~25%
// headroom while still catching the field's ~1,900-rune malformed merge line by
// a wide margin. Deliberately NOT applied to `created` lines (provenance prose,
// measured up to 331 runes) or free-form dated notes (up to 2199 runes, and
// actively encouraged — gate records, fold-ins, corrections): neither carries a
// TEMPLATE-prescribed shape to violate. A warning, not a truncation — sibling in
// spirit to board.maxCellRunes, but truncation must never become the way
// malformed history is hidden (see T-049).
const maxHistoryEntryRunes = 400

// Audit checks every invariant for the tickets/ tree under root, using cfg for the
// registered-child and per-child WIP checks.
func Audit(root string, cfg *config.Config) Result {
	var r Result
	auditStatusDirs(&r, root)
	tickets, issues := ticket.LoadAll(root)
	r.Errors = append(r.Errors, issues...)
	r.NumTickets = len(tickets)

	// Index by id; report duplicates.
	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, t := range tickets {
		ref := t.Dir + "/" + filepath.Base(t.Path)
		if prev, ok := byID[t.ID]; ok {
			r.errf("%s: duplicate id — also at %s/%s", ref, prev.Dir, filepath.Base(prev.Path))
			continue
		}
		byID[t.ID] = t
	}

	// Per-ticket frontmatter, id, grade, and project checks.
	for _, t := range tickets {
		ref := t.Dir + "/" + filepath.Base(t.Path)
		for _, k := range requiredKeys {
			if _, ok := t.Front[k]; !ok {
				r.errf("%s: frontmatter missing %q", ref, k)
			}
		}
		// A duplicate key silently overwrites in the parse (last wins, T-033) —
		// malformed however it arrived, so flagged however it arrived.
		for _, k := range t.DuplicateKeys {
			r.errf("%s: frontmatter has duplicate key %q — remove one (the parse keeps the last value)", ref, k)
		}
		if id := t.Front["id"]; id != "" && id != t.ID {
			r.errf("%s: frontmatter id %s != filename id %s", ref, id, t.ID)
		}
		for _, k := range []string{"impact", "complexity", "cost"} {
			if v := t.Front[k]; v != "" && !ticket.ValidGrade(k, v) {
				r.errf("%s: illegal %s value %q (legal: single values or adjacent-pair ranges)", ref, k, v)
			}
		}
		if p := t.Project(); p == "" {
			// missing-key already reported above
		} else if cp, ok := cfg.Project(p); !ok {
			r.errf("%s: project %q is not a registered child", ref, p)
		} else if got, _, ok := ticket.SplitID(t.ID); ok && got != cp.Prefix() {
			// The id's prefix must match its child's configured ticket_prefix
			// (T-058). This catches a ticket filed under the wrong project, and
			// it is the guard that keeps a child from being half-migrated to a new
			// prefix: change ticket_prefix on a populated child and every unrenamed
			// ticket goes red here until `pickle ticket renumber` (T-060) runs.
			r.errf("%s: id prefix %q does not match project %q ticket_prefix %q", ref, got, p, cp.Prefix())
		}
		for _, dep := range t.DependsOn {
			// A ticket citing itself (T-027) audits clean today, then silently
			// self-blocks at pickup: the transition guard demands the dependency
			// be in 6-done/, which it can never be while this ticket is in
			// development. Mirrors the spawned-by/family self-checks below.
			if dep == t.ID {
				r.errf("%s: depends-on lists itself", ref)
				continue
			}
			if _, ok := byID[dep]; !ok {
				r.errf("%s: depends-on %s does not exist", ref, dep)
			}
		}
		// spawned-by is lineage, not a dependency: the only checks are that each
		// parent exists and that a ticket does not cite itself. There is
		// deliberately NO done/merged or transition gate for it (contrast the
		// depends-on gate below) — provenance must never block pickup.
		for _, src := range t.SpawnedBy {
			if src == t.ID {
				r.errf("%s: spawned-by lists itself", ref)
				continue
			}
			if _, ok := byID[src]; !ok {
				r.errf("%s: spawned-by %s does not exist", ref, src)
			}
		}
		// family is lineage like spawned-by (never gates pickup) but with extra shape
		// invariants because the board renders it as a single-parent grouping (T-059):
		// the umbrella must exist, live in the SAME child (the board groups per child,
		// so a cross-child family could not render as one group), the ticket may not be
		// its own umbrella, and families are flat — the umbrella must not itself be a
		// family member (no nesting). It is an optional field, so validated only when
		// set; deliberately NOT in requiredKeys, so the existing backlog stays green.
		if fam := t.Family; fam != "" {
			switch parent, ok := byID[fam]; {
			case fam == t.ID:
				r.errf("%s: family lists itself", ref)
			case !ok:
				r.errf("%s: family %s does not exist", ref, fam)
			case parent.Project() != t.Project():
				r.errf("%s: family %s is in a different child-project", ref, fam)
			case parent.Family != "":
				r.errf("%s: family %s is itself a family member (families do not nest)", ref, fam)
			}
		}
	}

	// Board staleness check (T-044 D6). The board is a generated artifact, so the
	// only board invariant is "the file matches a fresh render" — one
	// byte-comparison after normalising the `Last updated:` line on both sides.
	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	if data, err := os.ReadFile(boardPath); err != nil {
		r.errf("BOARD.md: %v", err)
	} else if board.NormalizeLastUpdated(string(data)) !=
		board.NormalizeLastUpdated(board.Render(tickets, cfg, "")) {
		r.errf("BOARD.md is stale or hand-edited — run pickle board sync")
	}

	// Per-child WIP limits.
	auditWIP(&r, tickets, cfg)

	// History ↔ directory, plus the over-long-entry warning (T-040 D4/D5) —
	// folded into the same per-ticket loop so History is scanned once, not twice.
	for _, t := range tickets {
		ref := t.Dir + "/" + filepath.Base(t.Path)
		st, _ := ticket.StatusByDir(t.Dir)
		switch got := ticket.LastHistoryStatus(t.Text); got {
		case "":
			r.warnf("%s: no parseable status line in ## History", ref)
		case st.Name:
		default:
			r.errf("%s: last History status is %s but ticket sits in %s", ref, got, t.Dir)
		}
		// Only the two TEMPLATE-prescribed forms (a status transition, a merge
		// note) carry a shape to violate. `created` lines and free-form dated
		// notes are exempt — see maxHistoryEntryRunes.
		for _, e := range ticket.HistoryEntries(t.Text) {
			if e.Kind != ticket.HistoryTransition && e.Kind != ticket.HistoryMerged {
				continue
			}
			if n := len([]rune(e.Text)); n > maxHistoryEntryRunes {
				r.warnf("%s: History entry %s is %d runes (limit %d) — move the analysis into "+
					"the Description or tickets/NOTES.md, and keep the History line to the "+
					"prescribed 'OLD → NEW: one-clause reason' / 'merged to <base> (<ref>)' form",
					ref, e.Date, n, maxHistoryEntryRunes)
			}
		}
	}

	// In-development dependency gate. depends-on only: spawned-by parents are
	// intentionally absent here — lineage never gates a pickup.
	for _, t := range tickets {
		if t.Dir != "3-in-development" {
			continue
		}
		ref := t.Dir + "/" + filepath.Base(t.Path)
		for _, dep := range t.DependsOn {
			dt, ok := byID[dep]
			if !ok {
				continue // already reported
			}
			if dt.Dir != "6-done" {
				r.errf("%s: in development but dependency %s is in %s", ref, dep, dt.Dir)
			} else if !ticket.HasMergeLine(dt.Text) {
				r.warnf("%s: dependency %s is DONE but has no 'MERGED' History line — confirm the human merged it in its own child", ref, dep)
			}
		}
	}

	sort.Strings(r.Errors)
	sort.Strings(r.Warnings)
	return r
}

// auditWIP checks each child's per-status WIP limits. The tally itself comes from
// board.WIPCounts — the same counter behind the board's `(n/limit)` sub-headings
// and the dashboard's badges — so a limit cannot look breached in one view and
// satisfied in another.
func auditWIP(r *Result, tickets []*ticket.Ticket, cfg *config.Config) {
	counts := board.WIPCounts(tickets)
	children := make([]string, 0, len(counts))
	for p := range counts {
		children = append(children, p)
	}
	sort.Strings(children)
	for _, p := range children {
		c := counts[p]
		cp, ok := cfg.Project(p)
		if !ok {
			continue // unregistered project already reported per-ticket
		}
		if c.InDevelopment > cp.WIPInDevelopment {
			r.errf("WIP: child %q has %d tickets in 3-in-development (limit %d)", p, c.InDevelopment, cp.WIPInDevelopment)
		}
		if c.InReview > cp.WIPInReview {
			r.errf("WIP: child %q has %d tickets in 4-in-review (limit %d)", p, c.InReview, cp.WIPInReview)
		}
	}
}

// auditStatusDirs checks that all seven status directories exist (T-040 D3).
// ticket.LoadAll deliberately treats an absent directory as empty — git does not
// track empty directories, and the swallowing is load-bearing beyond that
// (internal/move turns any LoadAll issue into a hard failure) — so this is the
// one place the audit looks at the directories directly rather than through
// LoadAll. A missing directory is an error: re-running `pickle install`
// (idempotent) recreates it and its `.gitkeep`. A directory that exists but is
// empty and carries no `.gitkeep` is only a warning — not yet a broken
// invariant, but the exact predictor of the same defect on the next fresh clone
// (git does not track empty directories).
func auditStatusDirs(r *Result, root string) {
	for _, s := range ticket.Statuses {
		dir := filepath.Join(root, "tickets", s.Dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				r.errf("tickets/%s/: status directory is missing — re-run pickle install "+
					"(idempotent) to recreate it, and commit its .gitkeep", s.Dir)
			} else {
				r.errf("tickets/%s/: %v", s.Dir, err)
			}
			continue
		}
		hasKeep, hasTicket := false, false
		for _, e := range entries {
			switch {
			case e.Name() == ".gitkeep":
				hasKeep = true
			case !e.IsDir() && strings.HasSuffix(e.Name(), ".md"):
				hasTicket = true
			}
		}
		if !hasKeep && !hasTicket {
			r.warnf("tickets/%s/: empty and not kept by a .gitkeep — git does not track empty "+
				"directories, so a fresh clone will be missing this status", s.Dir)
		}
	}
}

func (r *Result) errf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}
func (r *Result) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}
