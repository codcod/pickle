package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/decisions"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/lock"
	"github.com/codcod/pickle/internal/state"
	"github.com/codcod/pickle/internal/sync"
)

// Board mechanics. `board audit` is the keystone (P1): a pure check over tickets/
// + pickle.toml. `board sync` (P3) regenerates the board from ticket state.
// `board state --json` (T-065) is the read-only, versioned JSON projection —
// no mechanics of its own, it only reads what audit/sync/board already own.
// `board decisions` (T-105) is the same shape one level down: a read-only
// query over every ticket's `### Confirmed design decisions` subsection.

func runBoard(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle board <audit|sync|state|decisions> ...")
		return exitUsage
	}
	switch args[0] {
	case "audit":
		return runBoardAudit(args[1:])
	case "sync":
		return runBoardSync(args[1:])
	case "state":
		return runBoardState(args[1:])
	case "decisions":
		return runBoardDecisions(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle board: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

const boardSyncUsage = "usage: pickle board sync [--dry-run]"

func runBoardSync(args []string) int {
	dryRun := false
	for _, a := range args {
		switch a {
		case "--dry-run", "-n":
			dryRun = true
		default:
			fmt.Fprintf(os.Stderr, "pickle board sync: unknown argument %q\n%s\n", a, boardSyncUsage)
			return exitUsage
		}
	}
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	res, err := sync.Sync(cfg.Root(), cfg, dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle board sync: %v\n", err)
		return exitError
	}
	for _, s := range res.Summary {
		fmt.Printf("  %s\n", s)
	}
	if dryRun {
		if res.Changed {
			fmt.Printf("board sync --dry-run: %s is OUT OF SYNC (%d change(s))\n", res.Path, len(res.Summary))
			return exitError
		}
		fmt.Printf("board sync --dry-run: %s is in sync\n", res.Path)
		return exitOK
	}
	if res.Changed {
		fmt.Printf("board sync: rebuilt %s (%d change(s))\n", res.Path, len(res.Summary))
	} else {
		fmt.Printf("board sync: %s already in sync\n", res.Path)
	}
	return exitOK
}

const boardStateUsage = "usage: pickle board state --json"

// runBoardState implements `pickle board state --json` (T-065): the whole
// ticket tree as one versioned JSON document. --json is mandatory, not a
// default (T-065 confirmed decision 1) — a bare `pickle board state` prints
// usage and exits 2, rather than dumping the full document at a human
// exploring the CLI; it also means a later human-readable default is not a
// breaking change for anything already scripted against the flag.
func runBoardState(args []string) int {
	jsonFlag := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonFlag = true
		default:
			fmt.Fprintf(os.Stderr, "pickle board state: unknown argument %q\n%s\n", a, boardStateUsage)
			return exitUsage
		}
	}
	if !jsonFlag {
		fmt.Fprintln(os.Stderr, boardStateUsage)
		return exitUsage
	}
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	def := flow.ForName(cfg.FlowName())
	doc, err := state.Build(def, cfg.Root(), cfg, Version)
	if err != nil {
		return errf("board state: %v", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	// Ticket prose is full of "&", "<" and "→" — HTML-escaping them would help
	// no consumer of a CLI-only, non-HTML wire format.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return errf("board state: %v", err)
	}
	return exitOK
}

const boardDecisionsUsage = "usage: pickle board decisions [--project <child>] [--status <dir>] [--grep <regex>] [--json]"

// runBoardDecisions implements `pickle board decisions` (T-105): every
// confirmed design decision recorded in the ticket tree, in citable
// `<ID> decision <N>` form, filterable by child-project, status directory and
// a regex over each decision's full text (statement + rationale). Unlike
// `board state`, --json is not mandatory here (T-105 decision 2) — the
// default output is a short, human-shaped table, not a full document dump.
func runBoardDecisions(args []string) int {
	fs := flag.NewFlagSet("board decisions", flag.ContinueOnError)
	project := fs.String("project", "", "filter by registered child-project name")
	status := fs.String("status", "", "filter by status directory, e.g. 6-done")
	grep := fs.String("grep", "", "filter: a regexp matched against each decision's statement and rationale")
	jsonFlag := fs.Bool("json", false, "print the machine-readable JSON form")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, boardDecisionsUsage)
		return exitUsage
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	def := flow.ForName(cfg.FlowName())

	var res decisions.Result
	// The whole read runs behind the tree's shared lock (T-105 decision 14),
	// the same pattern internal/state/build.go uses for `board state --json`:
	// a concurrent writer's atomic rename is either fully visible or not
	// visible yet, never half-written.
	err := lock.WithShared(cfg.Root(), func() error {
		var qerr error
		res, qerr = decisions.Query(def, cfg.Root(), cfg, decisions.Filter{
			Project: *project, Status: *status, Grep: *grep,
		})
		return qerr
	})
	if err != nil {
		return errf("board decisions: %v", err)
	}

	if *jsonFlag {
		doc := decisions.Document{
			Schema:        decisions.CurrentSchema,
			PickleVersion: Version,
			Filters:       decisions.Filters{Project: *project, Status: *status, Grep: *grep},
			Decisions:     res.Decisions,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		// Same reasoning as runBoardState: ticket prose is full of "&", "<" and
		// "→", which HTML-escaping would help no consumer of this CLI-only,
		// non-HTML wire format.
		enc.SetEscapeHTML(false)
		if err := enc.Encode(doc); err != nil {
			return errf("board decisions: %v", err)
		}
		return exitOK
	}

	renderDecisionsText(res)
	return exitOK
}

// renderDecisionsText prints res as a pasteable table, exactly the shape
// runProjectList uses (text/tabwriter). FORM is the closed "bold"/"raw" pair
// (T-105 decision 7/8): STATEMENT shows the projected bold-run statement for
// a structured decision, or the raw first line otherwise — the rationale
// itself is never printed, in either case. The closing summary line mirrors
// `board audit`'s own summary line, so the zero case prints a clear 0 rather
// than silence.
func renderDecisionsText(res decisions.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "DECISION\tPROJECT\tSTATUS\tFORM\tSTATEMENT")
	tickets := map[string]bool{}
	for _, d := range res.Decisions {
		form, text := "bold", d.Statement
		if !d.Structured {
			form, text = "raw", d.Raw
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.Citation, d.Project, d.Status, form, text)
		tickets[d.TicketID] = true
	}
	w.Flush()
	fmt.Printf("board decisions: %d decision(s) from %d ticket(s)\n", len(res.Decisions), len(tickets))
}

func runBoardAudit(_ []string) int {
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	res := audit.Audit(cfg.Root(), cfg)
	for _, w := range res.Warnings {
		fmt.Printf("WARNING: %s\n", w)
	}
	for _, e := range res.Errors {
		fmt.Printf("ERROR: %s\n", e)
	}
	fmt.Printf("board audit: %d tickets, %d error(s), %d warning(s)\n",
		res.NumTickets, len(res.Errors), len(res.Warnings))
	if len(res.Errors) > 0 {
		return exitError
	}
	return exitOK
}
