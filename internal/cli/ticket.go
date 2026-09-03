package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/lock"
	"github.com/codcod/pickle/internal/move"
	"github.com/codcod/pickle/internal/ticket"
)

// Ticket mechanics. `ticket new` lands in P1 (id allocation + template + board
// row); `ticket move` in P3 (state machine, per-child WIP, cross-child merge gate).

func runTicket(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle ticket <new|move> ...")
		return exitUsage
	}
	switch args[0] {
	case "new":
		return runTicketNew(args[1:])
	case "move":
		return runTicketMove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle ticket: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

const (
	ticketNewUsage  = `usage: pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V] [--spawned-by "T-NNN[,T-MMM]"] [--family T-NNN]`
	ticketMoveUsage = `usage: pickle ticket move <T-NNN> <status> [--reason "<why>"]`
)

func runTicketMove(args []string) int {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") || strings.HasPrefix(args[1], "-") {
		fmt.Fprintln(os.Stderr, ticketMoveUsage)
		return exitUsage
	}
	id, status := args[0], args[1]

	fs := flag.NewFlagSet("ticket move", flag.ContinueOnError)
	reason := fs.String("reason", "", "why the move is happening (required for backward/rework/drop moves)")
	if err := fs.Parse(args[2:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, ticketMoveUsage)
		return exitUsage
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	res, err := move.Move(cfg.Root(), cfg, id, status, *reason)
	if err != nil {
		return errf("%v", err)
	}
	fmt.Printf("moved %s: %s → %s  (%s)\n", id, res.From, res.To, res.Path)
	fmt.Printf("  removed  %s\n", res.OldPath)
	fmt.Printf("  stage:   %s\n", stageLine(res.Path, res.OldPath))
	for _, w := range res.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	return exitOK
}

// stageLine renders the ready-to-paste `git add` for the paths a writing
// command just touched — rules §0 requires bookkeeping commits to name explicit
// pathspecs, and naming them from memory is how a rename lands add-without-delete
// (T-091). Order is fixed: new path, removed path (if any), then the board.
func stageLine(newPath, oldPath string) string {
	paths := []string{newPath}
	if oldPath != "" {
		paths = append(paths, oldPath)
	}
	paths = append(paths, filepath.Join("tickets", "BOARD.md"))
	return "git add " + strings.Join(paths, " ")
}

func runTicketNew(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, ticketNewUsage)
		return exitUsage
	}
	title := args[0]

	fs := flag.NewFlagSet("ticket new", flag.ContinueOnError)
	project := fs.String("project", "", "target child-project (required)")
	impact := fs.String("impact", "medium", "impact grade")
	complexity := fs.String("complexity", "medium", "complexity grade")
	cost := fs.String("cost", "M", "cost grade")
	spawnedBy := fs.String("spawned-by", "", "lineage: ticket id(s) this one was born from, comma-separated (non-gating)")
	family := fs.String("family", "", "umbrella ticket id this one groups under for board ordering (single id, same child; non-gating)")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, ticketNewUsage)
		return exitUsage
	}
	if *project == "" {
		return errf("--project is required")
	}
	for _, g := range []struct{ kind, v string }{{"impact", *impact}, {"complexity", *complexity}, {"cost", *cost}} {
		if !ticket.ValidGrade(g.kind, g.v) {
			return errf("illegal %s value %q (legal: single values or adjacent-pair ranges)", g.kind, g.v)
		}
	}
	if err := ticket.ValidateTitle(title); err != nil {
		return errf("%v", err)
	}
	// Shape-checked and de-duplicated here, before anything is written. Whether the
	// ids *exist* is still the audit's job: --spawned-by T-999 for a ticket not yet
	// filed is legal input, and `pickle board audit` is what flags it.
	lineage, err := ticket.ParseIDList(*spawnedBy)
	if err != nil {
		return errf("--spawned-by: %v", err)
	}
	// family is a single umbrella id; same shape-check-now, existence-later split as
	// --spawned-by. It groups this ticket on the board and never gates pickup, so the
	// same-child / existence / no-nesting invariants are the audit's job, not this
	// write path's.
	fam := strings.TrimSpace(*family)
	if fam != "" && !ticket.ValidID(fam) {
		return errf("--family: %q is not a ticket id (expected <PREFIX>-NNN, e.g. T-001)", fam)
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	cp, ok := cfg.Project(*project)
	if !ok {
		return errf("project %q is not a registered child", *project)
	}

	root := cfg.Root()
	def := flow.ForName(cfg.FlowName())
	prefix := cp.Prefix()

	// T-101: id allocation (NextNum), the existence check and the create are
	// one critical section under the tree's exclusive lock, spanning
	// load→check→write — not just the write — so two concurrent `ticket new`
	// invocations can never compute the same NextNum and both succeed. The
	// O_EXCL below is belt-and-braces on top of the lock, not a replacement
	// for it: it is what still catches a collision from an older binary, or
	// any future caller that forgets to take the lock.
	var id, rel string
	err = lock.WithExclusive(root, func() error {
		id = fmt.Sprintf("%s-%03d", prefix, ticket.NextNum(def, root, prefix))
		slug := ticket.Slugify(title)
		rel = filepath.Join("tickets", def.Initial().Dir, id+"-"+slug+".md")
		path := filepath.Join(root, rel)
		if err := createExclusive(path, ticket.Scaffold(id, title, *project, *impact, *complexity, *cost, lineage, fam)); err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("%s already exists", rel)
			}
			return err
		}
		return board.Regenerate(def, root, cfg)
	})
	if err != nil {
		return errf("%v", err)
	}
	fmt.Printf("created %s  (%s)\n", id, rel)
	fmt.Printf("  stage:   %s\n", stageLine(rel, ""))
	return exitOK
}

// createExclusive creates path with content, refusing (with an os.IsExist
// error, unwrapped) if it already exists. This is T-101 Task 5: it replaces
// the former os.Stat-then-os.WriteFile pair, which left a window between the
// check and the write for a second writer to land in. It is
// belt-and-braces on top of runTicketNew's tree lock (which already makes
// this collision structurally unreachable through pickle's own code paths —
// two concurrent `ticket new` calls now serialise and always get distinct
// ids), not a replacement for it: this is what still catches a collision
// from an older binary, or any future caller that writes into tickets/
// without taking the lock.
func createExclusive(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := f.Write([]byte(content))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// maxTitleRuneLen and validateTitle moved to internal/ticket (T-102 Task 1),
// exported as ticket.MaxTitleRuneLen / ticket.ValidateTitle, so ticket new
// and ticket set share one validator instead of each carrying its own copy.
