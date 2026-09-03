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
	"github.com/codcod/pickle/internal/ticketset"
)

// Ticket mechanics. `ticket new` lands in P1 (id allocation + template + board
// row); `ticket move` in P3 (state machine, per-child WIP, cross-child merge gate).

func runTicket(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle ticket <new|move|set> ...")
		return exitUsage
	}
	switch args[0] {
	case "new":
		return runTicketNew(args[1:])
	case "move":
		return runTicketMove(args[1:])
	case "set":
		return runTicketSet(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle ticket: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

const (
	ticketNewUsage  = `usage: pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V] [--spawned-by "T-NNN[,T-MMM]"] [--family T-NNN]`
	ticketMoveUsage = `usage: pickle ticket move <T-NNN> <status> [--reason "<why>"]`
	ticketSetUsage  = `usage: pickle ticket set <T-NNN> (--impact V|--complexity V|--cost V|--family T-NNN|--title "<title>")`
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

// runTicketSet implements `pickle ticket set` (T-102): exactly one of
// --impact/--complexity/--cost/--family/--title must be present on the
// command line (decision 3) — checked via fs.Visit rather than a zero-value
// check, since an explicitly empty flag value must still count as "present".
// Validation is the same per-field validator `ticket new` already uses;
// the guarded write itself is internal/ticketset.Set.
func runTicketSet(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, ticketSetUsage)
		return exitUsage
	}
	id := args[0]

	fs := flag.NewFlagSet("ticket set", flag.ContinueOnError)
	impact := fs.String("impact", "", "new impact grade")
	complexity := fs.String("complexity", "", "new complexity grade")
	cost := fs.String("cost", "", "new cost grade")
	family := fs.String("family", "", "new umbrella ticket id")
	title := fs.String("title", "", "new title")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, ticketSetUsage)
		return exitUsage
	}

	valueByField := map[string]*string{
		"impact": impact, "complexity": complexity, "cost": cost, "family": family, "title": title,
	}
	// fs.Visit walks only flags actually named on the command line — an empty
	// default (e.g. --title "") would otherwise be indistinguishable from not
	// passing --title at all, which matters here because "exactly one field"
	// (T-102 decision 3) has to count what the caller typed, not what parsed.
	var touched []string
	fs.Visit(func(f *flag.Flag) {
		if _, ok := valueByField[f.Name]; ok {
			touched = append(touched, f.Name)
		}
	})
	if len(touched) != 1 {
		return errf("exactly one of --impact/--complexity/--cost/--family/--title is required (got %d)", len(touched))
	}
	field := touched[0]
	value := *valueByField[field]

	switch field {
	case "impact", "complexity", "cost":
		if !ticket.ValidGrade(field, value) {
			return errf("illegal %s value %q (legal: single values or adjacent-pair ranges)", field, value)
		}
	case "family":
		if !ticket.ValidID(value) {
			return errf("--family: %q is not a ticket id (expected <PREFIX>-NNN, e.g. T-001)", value)
		}
	case "title":
		if err := ticket.ValidateTitle(value); err != nil {
			return errf("%v", err)
		}
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	res, err := ticketset.Set(cfg.Root(), cfg, id, field, value)
	if err != nil {
		return errf("%v", err)
	}
	if res.Old == res.New {
		fmt.Printf("%s.%s already %q — nothing to do\n", id, res.Field, res.Old)
		return exitOK
	}
	fmt.Printf("set %s.%s: %q → %q  (%s)\n", id, res.Field, res.Old, res.New, res.Path)
	fmt.Printf("  stage:   %s\n", stageLine(res.Path, ""))
	return exitOK
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
