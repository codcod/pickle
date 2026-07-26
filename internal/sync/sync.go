// Package sync implements `pickle board sync`: regenerate tickets/BOARD.md
// wholesale from ground truth (the ticket files + pickle.toml). The board is a
// pure generated artifact (T-044) — nothing in it is preserved across a sync;
// hand-written planning notes belong in tickets/NOTES.md, which sync never
// touches. "In sync" is defined as internal/audit.Audit reporting zero errors,
// so a successful sync leaves `board audit` clean.
package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// Result records the outcome of a sync.
type Result struct {
	Changed bool     // the board differs from a fresh render (Last updated: aside)
	Summary []string // human-readable drift (populated in dry-run and apply)
	Path    string   // board path, relative to root
}

// Sync regenerates the board for the tickets/ tree under root. With dryRun it
// reports drift without writing; otherwise it writes when the file differs from
// a fresh render (normalising the `Last updated:` line, so a date-only
// difference is not drift) and runs an audit self-check.
func Sync(root string, cfg *config.Config, dryRun bool) (Result, error) {
	res := Result{Path: filepath.Join("tickets", "BOARD.md")}

	tickets, issues := ticket.LoadAll(root)
	if len(issues) > 0 {
		return res, fmt.Errorf("cannot sync while the board has load problems: %s", strings.Join(issues, "; "))
	}

	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	newText := board.Render(tickets, cfg, time.Now().Format("2006-01-02"))

	original := ""
	if data, err := os.ReadFile(boardPath); err == nil {
		original = string(data)
	}
	res.Changed = board.NormalizeLastUpdated(original) != board.NormalizeLastUpdated(newText)

	// Drift summary: row membership in the old file vs the rendered truth.
	listed := map[string]string{} // id -> section in the current file
	if rows, err := board.Parse(boardPath); err == nil {
		for _, r := range rows {
			listed[r.ID] = r.Status
		}
	}
	desired := map[string]string{} // id -> section a fresh render puts it in
	for _, t := range tickets {
		st, _ := ticket.StatusByDir(t.Dir)
		desired[t.ID] = st.Name
	}
	res.Summary = drift(listed, desired, res.Changed)

	if dryRun || !res.Changed {
		return res, nil
	}
	if err := os.WriteFile(boardPath, []byte(newText), 0o644); err != nil {
		return res, err
	}
	// Post-op self-check: the tree must be audit-clean (surfaces a ticket-side
	// problem sync cannot fix by rewriting the board).
	if a := audit.Audit(root, cfg); len(a.Errors) > 0 {
		return res, fmt.Errorf("sync applied but board audit still reports %d error(s): %s",
			len(a.Errors), strings.Join(a.Errors, "; "))
	}
	return res, nil
}

// drift compares the current board membership with the desired membership and
// returns a human-readable summary. reformat is true when the text changed but
// no row moved (ordering / WIP counts / spacing only).
func drift(current, desired map[string]string, changed bool) []string {
	var out []string
	ids := map[string]bool{}
	for id := range current {
		ids[id] = true
	}
	for id := range desired {
		ids[id] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	for _, id := range sorted {
		c, inC := current[id]
		d, inD := desired[id]
		switch {
		case !inC && inD:
			out = append(out, fmt.Sprintf("add %s to %s", id, d))
		case inC && !inD:
			out = append(out, fmt.Sprintf("remove %s from %s (no backing ticket file)", id, c))
		case c != d:
			out = append(out, fmt.Sprintf("move %s: %s -> %s", id, c, d))
		}
	}
	if len(out) == 0 && changed {
		out = append(out, "reformat only (ordering / WIP counts / spacing / preamble)")
	}
	return out
}
