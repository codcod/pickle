// Package audit implements `pickle board audit`: a pure check of the ticket-flow
// invariants over a tickets/ tree + pickle.toml. It never prints or exits — it
// returns findings so it stays fixture-testable.
package audit

import (
	"fmt"
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

var requiredKeys = []string{"id", "title", "project", "depends-on", "impact", "complexity", "cost"}

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
	}

	// Board cross-check.
	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	rows, err := board.Parse(boardPath)
	if err != nil {
		r.errf("BOARD.md: %v", err)
	} else {
		seen := map[string]board.Row{}
		for _, row := range rows {
			if prev, ok := seen[row.ID]; ok {
				r.errf("BOARD.md: %s listed twice (%s and %s)", row.ID, prev.Status, row.Status)
			}
			seen[row.ID] = row
			t, ok := byID[row.ID]
			if !ok {
				r.errf("BOARD.md: %s listed under %s but no ticket file exists", row.ID, row.Status)
				continue
			}
			st, _ := ticket.StatusByDir(t.Dir)
			if row.Status != st.Name {
				r.errf("BOARD.md: %s is in %s but listed under %s", row.ID, t.Dir, row.Status)
			}
			if p := t.Project(); p != "" && row.Child != "" && row.Child != p {
				r.errf("BOARD.md: %s is under child %q but its project is %q", row.ID, row.Child, p)
			}
		}
		for _, t := range tickets {
			st, _ := ticket.StatusByDir(t.Dir)
			row, listed := seen[t.ID]
			switch {
			case !listed && !st.Terminal:
				r.errf("BOARD.md: %s (%s) missing from the board", t.ID, t.Dir)
			case listed && row.Status != st.Name:
				// already reported above
			}
		}
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

	// In-development dependency gate.
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

func auditWIP(r *Result, tickets []*ticket.Ticket, cfg *config.Config) {
	type counts struct{ dev, rev int }
	byChild := map[string]*counts{}
	for _, t := range tickets {
		p := t.Project()
		if p == "" {
			continue
		}
		c := byChild[p]
		if c == nil {
			c = &counts{}
			byChild[p] = c
		}
		switch t.Dir {
		case "3-in-development":
			c.dev++
		case "4-in-review":
			c.rev++
		}
	}
	children := make([]string, 0, len(byChild))
	for p := range byChild {
		children = append(children, p)
	}
	sort.Strings(children)
	for _, p := range children {
		c := byChild[p]
		cp, ok := cfg.Project(p)
		if !ok {
			continue // unregistered project already reported per-ticket
		}
		if c.dev > cp.WIPInDevelopment {
			r.errf("WIP: child %q has %d tickets in 3-in-development (limit %d)", p, c.dev, cp.WIPInDevelopment)
		}
		if c.rev > cp.WIPInReview {
			r.errf("WIP: child %q has %d tickets in 4-in-review (limit %d)", p, c.rev, cp.WIPInReview)
		}
	}
}

func (r *Result) errf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}
func (r *Result) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}
