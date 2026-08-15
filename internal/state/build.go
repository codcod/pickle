package state

import (
	"os"
	"path/filepath"
	"time"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/lock"
	"github.com/codcod/pickle/internal/ticket"
)

// Build assembles the whole projection for the tickets/ tree under root,
// using cfg for the registered-child and WIP configuration and version as
// the emitting binary's version (Document.PickleVersion).
//
// The entire traversal runs inside lock.WithShared (T-065 confirmed decision
// 14), the same pattern internal/serve/serve.go's handler.load and
// internal/serve/view.go's buildHealth use — so a concurrent writer's atomic
// rename is either fully visible or not visible yet, never half-written.
// Unlike buildHealth, a lock error here is fatal: this command's contract is
// "a correct document or a non-zero exit," not a degraded page, so the error
// is returned rather than swallowed into a zero-valued result.
func Build(def *flow.Definition, root string, cfg *config.Config, version string) (Document, error) {
	var doc Document
	err := lock.WithShared(root, func() error {
		doc = build(def, root, cfg, version)
		return nil
	})
	if err != nil {
		return Document{}, err
	}
	return doc, nil
}

// build is Build's body, run while the caller holds the tree's shared lock.
func build(def *flow.Definition, root string, cfg *config.Config, version string) Document {
	tickets, _ := ticket.LoadAll(def, root)
	// Structural load problems (a bad filename, a missing frontmatter block)
	// are not surfaced here — they are exactly what audit.Audit's own,
	// separate LoadAll call below reports in Health.Errors, so they are not
	// dropped, only reported in one place instead of two.

	byID := make(map[string]*ticket.Ticket, len(tickets))
	for _, t := range tickets {
		byID[t.ID] = t
	}

	doc := Document{
		Schema:        CurrentSchema,
		PickleVersion: version,
		Flow:          def.Name(),
		Root:          root,
		States:        buildStates(def),
		Children:      buildChildren(def, cfg, tickets),
		Tickets:       buildTickets(def, cfg, tickets, byID),
		Health:        buildHealth(def, root, cfg),
	}
	return doc
}

func buildStates(def *flow.Definition) []State {
	states := make([]State, 0, len(def.BoardStates()))
	for _, st := range def.BoardStates() {
		states = append(states, State{
			Name:     st.Name,
			Dir:      st.Dir,
			Terminal: st.Terminal,
			WIPKey:   st.WIPKey,
		})
	}
	return states
}

func buildChildren(def *flow.Definition, cfg *config.Config, tickets []*ticket.Ticket) []Child {
	wip := board.WIPCounts(def, tickets)
	wipStates := def.WIPStates()
	children := make([]Child, 0, len(cfg.Projects))
	for _, p := range cfg.Projects {
		c := Child{Name: p.Name, WIP: []WIP{}}
		for _, st := range wipStates {
			limit, ok := p.WIPLimitFor(st.WIPKey)
			if !ok {
				continue
			}
			c.WIP = append(c.WIP, WIP{
				Key:    st.WIPKey,
				Dir:    st.Dir,
				Status: st.Name,
				Count:  wip[p.Name][st.Dir],
				Limit:  limit,
			})
		}
		children = append(children, c)
	}
	return children
}

// buildTickets projects every ticket, in the board's own order: def.BoardStates()
// order, then cfg.Projects order within a state, then board.Sort within a
// child's group (T-065 confirmed decision 7) — the same three-level grouping
// internal/serve/view.go's stateChildGroup uses for the dashboard, so the
// array reads in the same order the board and BOARD.md do.
func buildTickets(def *flow.Definition, cfg *config.Config, tickets []*ticket.Ticket, byID map[string]*ticket.Ticket) []Ticket {
	out := make([]Ticket, 0, len(tickets))
	for _, st := range def.BoardStates() {
		for _, p := range cfg.Projects {
			var group []*ticket.Ticket
			for _, t := range tickets {
				if t.Dir == st.Dir && t.Project() == p.Name {
					group = append(group, t)
				}
			}
			board.Sort(group, st, byID)
			for _, t := range group {
				out = append(out, projectTicket(def, t, st.Name))
			}
		}
	}
	return out
}

func projectTicket(def *flow.Definition, t *ticket.Ticket, status string) Ticket {
	prefix, _, _ := ticket.SplitID(t.ID)
	return Ticket{
		ID:            t.ID,
		Num:           t.Num,
		Prefix:        prefix,
		Title:         t.Front["title"],
		Project:       t.Project(),
		Status:        status,
		Dir:           t.Dir,
		File:          filepath.Base(t.Path),
		Slug:          t.Slug,
		Impact:        t.Front["impact"],
		Complexity:    t.Front["complexity"],
		Cost:          t.Front["cost"],
		DependsOn:     nonNil(t.DependsOn),
		SpawnedBy:     nonNil(t.SpawnedBy),
		Family:        t.Family,
		DuplicateKeys: nonNil(t.DuplicateKeys),
		FrontMatter:   t.Front,
		Merged:        ticket.MergeLine(def, t.Text),
		History:       buildHistory(def, t.Text),
		Review:        parseReview(t.Text),
	}
}

func buildHistory(def *flow.Definition, text string) []HistoryEntry {
	entries := ticket.HistoryEntries(def, text)
	out := make([]HistoryEntry, 0, len(entries))
	for _, h := range entries {
		out = append(out, HistoryEntry{
			Date:   h.Date,
			Kind:   string(h.Kind),
			Target: h.Target,
			Text:   h.Text,
		})
	}
	return out
}

// buildHealth wraps audit.Audit (a second, separate full traversal, exactly
// as internal/serve/view.go's buildHealth does — see internal/lock's package
// doc for why this is an accepted, cosmetic-only limitation) and adds the
// board's own drift classification (T-052's board.Drift, reused rather than
// renamed — T-065 confirmed decision 13).
func buildHealth(def *flow.Definition, root string, cfg *config.Config) Health {
	res := audit.Audit(root, cfg)
	h := Health{
		Tickets:  res.NumTickets,
		Errors:   nonNil(res.Errors),
		Warnings: nonNil(res.Warnings),
	}

	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	current := ""
	if data, err := os.ReadFile(boardPath); err == nil {
		current = string(data)
	}
	tickets, issues := ticket.LoadAll(def, root)
	if len(issues) == 0 {
		fresh := board.Render(def, tickets, cfg, time.Now().Format("2006-01-02"))
		h.BoardDrift = driftString(board.Compare(def, current, fresh))
	} else {
		// A tree with structural load problems cannot be rendered fresh
		// (board.Render has nothing sound to build from); the drift errors
		// above already report the underlying problem, so this is not a
		// second error, just an honest "unknown" rather than a guess.
		h.BoardDrift = "unknown"
	}
	return h
}

func driftString(d board.Drift) string {
	switch d {
	case board.DriftLayout:
		return "layout"
	case board.DriftRows:
		return "rows"
	default:
		return "none"
	}
}

// nonNil returns s, or an empty (never nil) slice of the same type when s is
// nil — so an empty field marshals to JSON `[]`, not `null`. A ticket with
// no depends-on, no spawned-by, no duplicate frontmatter keys, or an audit
// with no errors/warnings is common, and a `[]`-returning consumer (e.g. a
// jq `.[]` iteration) should not have to special-case `null`.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
