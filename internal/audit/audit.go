// Package audit implements `pickle board audit`: a pure check of the ticket-flow
// invariants over a tickets/ tree + pickle.toml. It never prints or exits — it
// returns findings so it stays fixture-testable.
package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

// Audit checks every invariant for the tickets/ tree under root, using cfg for the
// registered-child and per-child WIP checks.
func Audit(root string, cfg *config.Config) Result {
	var r Result
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
		} else if _, ok := cfg.Project(p); !ok {
			r.errf("%s: project %q is not a registered child", ref, p)
		}
		for _, dep := range t.DependsOn {
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

	// History ↔ directory.
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

func (r *Result) errf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}
func (r *Result) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}
