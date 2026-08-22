// Package audit implements `pickle board audit`: a pure check of the brine flow's
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
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

// Result is the outcome of an audit.
type Result struct {
	NumTickets int
	Errors     []string
	Warnings   []string
}

var requiredKeys = []string{"id", "title", "project", "depends-on", "spawned-by", "impact", "complexity", "cost"}

// optionalKeys lists frontmatter keys that are legal but not required (today:
// T-059's family:, whose shape checks live in the per-ticket loop below and run
// only when the key is set). The list itself validates nothing at runtime — it
// is kept next to requiredKeys so the audit's full frontmatter vocabulary has
// one home instead of the optional-key concept living only inside an `if`;
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
	def := flow.ForName(cfg.FlowName())
	auditStatusDirs(&r, def, root)
	tickets, issues := ticket.LoadAll(def, root)
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
			// ticket goes red here until it is manually renumbered (T-060 considered
			// and dropped a `pickle ticket renumber` command: the one real case is a
			// one-time guided migration, not automation worth shipping).
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
		// Gate table (T-081): every Requirement the ticket's own state declares
		// (flow.State.Requires, read via def.Requirements) is evaluated against
		// its text — a Blocking violation is an error (the same check
		// ticket move refuses a move on, now caught for a ticket already past
		// the gate), an Advisory one a warning. This is also where T-083's
		// original Outcome-presence check lives now: it used to be hand-rolled
		// here as `!st.Terminal && ticket.OutcomeMissing(...)`, gated on
		// Terminal directly; the exemption for 6-done/7-dropped is now the
		// table's own (brine declares no Requires on either terminal state,
		// internal/flow/brine.go), not this file's — a permanent archive simply
		// has nothing to ask, rather than this loop special-casing it.
		for _, v := range ticket.GateViolations(def.Requirements(t.Dir), t.Text) {
			if v.Blocking() {
				r.errf("%s: %s%s", ref, v.Message(), gateRemedy(def, t.Dir))
			} else {
				r.warnf("%s: %s", ref, v.Message())
			}
		}
	}

	// Board staleness check (T-044 D6, two-tiered by T-052). The board is a
	// generated artifact, so the invariant is that its ticket rows match a
	// fresh render — but the file also carries generated scaffolding (preamble,
	// WIP-limit lines, per-child sub-headings and counts) that a registry change
	// or a renderer upgrade can make stale without any row disagreeing. Rows
	// must match (an error otherwise); the layout merely should (a warning
	// otherwise) — see board.Compare for the classification itself.
	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	if data, err := os.ReadFile(boardPath); err != nil {
		r.errf("BOARD.md: %v", err)
	} else {
		switch board.Compare(def, string(data), board.Render(def, tickets, cfg, "")) {
		case board.DriftRows:
			r.errf("BOARD.md does not match the ticket files (rows differ) — run pickle board sync")
		case board.DriftLayout:
			r.warnf("BOARD.md is out of date in its generated layout only (every ticket row matches) — run pickle board sync")
		}
	}

	// Per-child WIP limits.
	auditWIP(&r, def, tickets, cfg)

	// History ↔ directory, plus the over-long-entry warning (T-040 D4/D5) —
	// folded into this loop so the tickets are not walked a third time.
	for _, t := range tickets {
		ref := t.Dir + "/" + filepath.Base(t.Path)
		st, _ := def.ByDir(t.Dir)
		switch got := ticket.LastHistoryStatus(def, t.Text); got {
		case "":
			r.warnf("%s: no parseable status line in ## History", ref)
		case st.Name:
		default:
			r.errf("%s: last History status is %s but ticket sits in %s", ref, got, t.Dir)
		}
		// Only the two TEMPLATE-prescribed forms (a status transition, a merge
		// note) carry a shape to violate. `created` lines and free-form dated
		// notes are exempt — see maxHistoryEntryRunes.
		for _, e := range ticket.HistoryEntries(def, t.Text) {
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

	// In-development dependency gate: applies to def.Pickup() (T-081; for
	// brine, IN DEVELOPMENT) — the same state internal/move gates pickup
	// into. depends-on only: spawned-by parents are intentionally absent here
	// — lineage never gates a pickup.
	pickup := def.Pickup()
	for _, t := range tickets {
		if t.Dir != pickup.Dir {
			continue
		}
		ref := t.Dir + "/" + filepath.Base(t.Path)
		for _, dep := range t.DependsOn {
			dt, ok := byID[dep]
			if !ok {
				continue // already reported
			}
			if dt.Dir != def.DependencySatisfied().Dir {
				r.errf("%s: in development but dependency %s is in %s", ref, dep, dt.Dir)
			} else if !ticket.HasMergeLine(def, dt.Text) {
				r.warnf("%s: dependency %s is DONE but has no 'MERGED' History line — confirm the human merged it in its own child", ref, dep)
			}
		}
	}

	// Unfinalized-merge detection (T-092): the dependency-scoped warning above
	// only fires when some other ticket names a done one in depends-on — a
	// done ticket nobody depends on (the common case) was never checked. Scan
	// every ticket sitting in def.DependencySatisfied() (DONE; never DROPPED —
	// a dropped ticket has nothing to merge) and warn on its own ref when it
	// carries no merge line. This is a WARNING, not an error (rules §3/§4:
	// merging is the human's and may lag) — it stays alongside the
	// dependency-scoped check rather than replacing it, since the two address
	// different readers (rules §4 decision 3).
	done := def.DependencySatisfied().Dir
	for _, t := range tickets {
		if t.Dir != done {
			continue
		}
		if ticket.HasMergeLine(def, t.Text) {
			continue
		}
		ref := t.Dir + "/" + filepath.Base(t.Path)
		r.warnf("%s: DONE but has no 'MERGED' History line — not merged yet, or the merge line was forgotten (rules §4: append it and run pickle board sync)", ref)
	}

	sort.Strings(r.Errors)
	sort.Strings(r.Warnings)
	return r
}

// gateRemedy names the second of a blocking gate violation's "two ways out"
// (T-081 Task 4): writing the missing heading always works, and is named by
// v.Message() itself; this appends the other way, when one actually exists —
// moving the ticket to a state reachable in one legal move from dir that
// carries no Blocking requirement of its own. Derived from def.Allowed(dir)
// rather than hard-coded, so it can never again name an illegal transition
// (rework fix for review finding B1, T-081: the original hard-coded "move the
// ticket back to 1-to-do" is only ever true from 2-ready — brine's transition
// table gives 1-to-do no direct predecessor from 3-in-development/4-in-review
// /5-rework, so the same suffix on those states told a user to run a move
// `ticket move` would then refuse). A terminal target is skipped even when it
// meets the requirement bar: dropping the ticket is not "fixing" the plan, and
// suggesting it here would blur the gate-violation message with the abort
// gate's own sign-off requirement. When no such state exists (3-in-development,
// 4-in-review, 5-rework today — every legal move from them either keeps the
// same gate or is terminal), the remedy is the generic one: fix the text in
// place, since no move clears the violation for free.
func gateRemedy(def *flow.Definition, dir string) string {
	for _, st := range def.Allowed(dir) {
		if st.Terminal {
			continue
		}
		if !hasBlockingRequirement(def.Requirements(st.Dir)) {
			return fmt.Sprintf(" — write it, or move the ticket back to %s until the plan is complete", st.Dir)
		}
	}
	return " — write it to satisfy the gate"
}

// hasBlockingRequirement reports whether any row in reqs is Blocking —
// gateRemedy's test for "does this state's own gate still hold the same
// requirement class".
func hasBlockingRequirement(reqs []flow.Requirement) bool {
	for _, r := range reqs {
		if r.Severity == flow.Blocking {
			return true
		}
	}
	return false
}

// auditWIP checks each child's per-status WIP limits. The tally itself comes from
// board.WIPCounts — the same counter behind the board's `(n/limit)` sub-headings
// and the dashboard's badges — so a limit cannot look breached in one view and
// satisfied in another.
func auditWIP(r *Result, def *flow.Definition, tickets []*ticket.Ticket, cfg *config.Config) {
	counts := board.WIPCounts(def, tickets)
	children := make([]string, 0, len(counts))
	for p := range counts {
		children = append(children, p)
	}
	sort.Strings(children)
	wipStates := def.WIPStates()
	for _, p := range children {
		cp, ok := cfg.Project(p)
		if !ok {
			continue // unregistered project already reported per-ticket
		}
		for _, s := range wipStates {
			limit, ok := cp.WIPLimitFor(s.WIPKey)
			if !ok {
				continue
			}
			if n := counts[p][s.Dir]; n > limit {
				r.errf("WIP: child %q has %d tickets in %s (limit %d)", p, n, s.Dir, limit)
			}
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
func auditStatusDirs(r *Result, def *flow.Definition, root string) {
	for _, s := range def.States() {
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
