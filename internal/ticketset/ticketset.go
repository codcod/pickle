// Package ticketset implements `pickle ticket set`: a guarded, single-field
// edit to one ticket's frontmatter or title+heading (T-102).
//
// Like internal/move, the whole operation — load, the duplicate-key
// precondition, the edit itself, and the post-write board regeneration —
// runs behind the tree's exclusive lock (internal/lock), so a concurrent
// pickle process never reads the tree mid-decision or mid-write. The lock
// spans load→check→write→regenerate, not just the write, for the same
// reason move.Move's doc comment gives: a lock around the write alone would
// leave every check it guards racing against a concurrent writer.
//
// What this package refuses, and why: internal/ticket.SetField already
// refuses an edit that would touch more than the one intended line (plus,
// for a title edit, the one H1 line) or that would not read back as
// intended — its own parse-back guard, checked per call. This package adds
// the one precondition that belongs to the whole ticket rather than to a
// single field: a ticket whose frontmatter already carries ANY duplicate
// key is refused outright, regardless of which field is being set (T-040's
// DuplicateKeys). The Description this ships against is explicit that a
// duplicate must be refused, never silently repaired by picking a winner.
package ticketset

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/codcod/pickle/internal/atomicfile"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/lock"
	"github.com/codcod/pickle/internal/ticket"
)

// Result records a completed (or no-op) set for the CLI summary.
type Result struct {
	Path  string // ticket path, relative to root
	Field string
	Old   string // the field's value before this call
	New   string // the field's value after this call (== Old for a no-op)
}

// Set rewrites id's field to value. It takes the tree's exclusive lock for
// the whole operation (package doc). Setting a field to the value it
// already has is a no-op: Result is still populated (Old == New == value)
// but nothing is written and the board is not regenerated.
func Set(root string, cfg *config.Config, id, field, value string) (Result, error) {
	var res Result
	var err error
	lockErr := lock.WithExclusive(root, func() error {
		res, err = set(root, cfg, id, field, value)
		return nil // set's own error is returned via err, not the lock's
	})
	if lockErr != nil {
		return res, lockErr
	}
	return res, err
}

// set is Set's body, run while the caller holds the exclusive tree lock.
func set(root string, cfg *config.Config, id, field, value string) (Result, error) {
	var res Result

	found := false
	for _, f := range ticket.SettableFields {
		if f == field {
			found = true
			break
		}
	}
	if !found {
		return res, fmt.Errorf("field %q is not settable (legal: %s)", field, strings.Join(ticket.SettableFields, ", "))
	}

	def := flow.ForName(cfg.FlowName())
	tickets, issues := ticket.LoadAll(def, root)
	if len(issues) > 0 {
		return res, fmt.Errorf("cannot set a field while the board has load problems: %s", strings.Join(issues, "; "))
	}
	var t *ticket.Ticket
	for _, c := range tickets {
		if c.ID == id {
			t = c
			break
		}
	}
	if t == nil {
		return res, fmt.Errorf("ticket %s not found", id)
	}
	if len(t.DuplicateKeys) > 0 {
		return res, fmt.Errorf("%s: frontmatter has duplicate key(s) %s; refusing to set any field until fixed by hand",
			id, strings.Join(t.DuplicateKeys, ", "))
	}

	res.Path = filepath.Join("tickets", t.Dir, filepath.Base(t.Path))
	res.Field = field
	res.Old = t.Front[field]
	res.New = value

	if res.Old == value {
		return res, nil // nothing to write, nothing to regenerate
	}

	updated, err := ticket.SetField(t.Text, id, field, value)
	if err != nil {
		return res, fmt.Errorf("%s: %w", id, err)
	}
	if err := atomicfile.WriteFile(t.Path, []byte(updated)); err != nil {
		return res, err
	}
	if err := board.Regenerate(def, root, cfg); err != nil {
		return res, err
	}
	return res, nil
}
