package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/decisions"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/lock"
	"github.com/codcod/pickle/internal/metrics"
	"github.com/codcod/pickle/internal/state"
	"github.com/codcod/pickle/internal/sync"
)

// Board mechanics. `board audit` is the keystone (P1): a pure check over tickets/
// + pickle.toml. `board sync` (P3) regenerates the board from ticket state.
// `board state --json` (T-065) is the read-only, versioned JSON projection —
// no mechanics of its own, it only reads what audit/sync/board already own.
// `board decisions` (T-105) is the same shape one level down: a read-only
// query over every ticket's `### Confirmed design decisions` subsection.
// `board metrics` (T-126) is the same shape again, over interval durations
// mined from `## History` instead — backlog dwell, lead time and open-ticket
// age.

func runBoard(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle board <audit|sync|state|decisions|metrics> ...")
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
	case "metrics":
		return runBoardMetrics(args[1:])
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

const boardMetricsUsage = "usage: pickle board metrics [--project <child>] [--as-of YYYY-MM-DD] [--json]"

// metricsNow is time.Now, indirected so a test can pin the clock the default
// (no --as-of) path reads.
//
// This exists because a plain end-to-end test of that path provably cannot
// guard it (T-126 review, R1). The F1 defect — a raw local instant used where
// a midnight-UTC date belongs — only changes the output when the local date
// and the instant's UTC date differ, i.e. for local times before the zone's
// UTC offset (east of UTC) or after 24h minus it (west). In a UTC test
// environment the two agree at every hour, so a test that merely ran the
// command with no flag would pass against the defect and "cover" nothing.
// Pinning the clock to an instant chosen to straddle that boundary is what
// makes the guard deterministic on any machine, in any zone.
//
// Kept unexported and restored via defer by its one test; nothing else in the
// package reads it.
var metricsNow = time.Now

// resolveMetricsAsOf turns the --as-of flag value and a wall-clock instant
// into the report's reference date: the flag when given, otherwise today.
//
// Split out of runBoardMetrics deliberately. As three lines inline it was
// unreachable by any test — the default branch depends on the process clock
// and on time.Local, which Go caches at first use, so no test could pin it —
// and that is precisely where F1 hid (T-126 review): `time.Now()` went in raw,
// so the day arithmetic against parsed midnight-UTC endpoints moved with the
// hour and the zone. As a pure function of (flag, instant) both branches are
// directly testable, including from zones the test machine is not in.
func resolveMetricsAsOf(flagValue string, now time.Time) (time.Time, error) {
	if flagValue == "" {
		return metrics.DateOf(now), nil
	}
	d, err := time.Parse("2006-01-02", flagValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --as-of %q: must be YYYY-MM-DD", flagValue)
	}
	return d, nil
}

// runBoardMetrics implements `pickle board metrics` (T-126): backlog dwell,
// lead time and open-ticket age mined from every ticket's `## History`,
// aggregated per child-project. Modeled directly on runBoardDecisions
// (T-105): a leaf query package wrapped in the shared tree lock, a short
// human table by default, `--json` for the versioned envelope.
//
// --as-of defaults to today, but is the one flag that makes a run's own
// output depend on the clock rather than only the tree — pinning it, as the
// acceptance test does, is what keeps two runs byte-identical (the property
// `board state --json` established, T-065 decision 3, and the reason
// internal/metrics.Compute takes a caller-supplied time rather than reading
// one itself).
func runBoardMetrics(args []string) int {
	fs := flag.NewFlagSet("board metrics", flag.ContinueOnError)
	project := fs.String("project", "", "filter by registered child-project name")
	asOfFlag := fs.String("as-of", "", "the report's reference date, YYYY-MM-DD (default: today)")
	jsonFlag := fs.Bool("json", false, "print the machine-readable JSON form")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, boardMetricsUsage)
		return exitUsage
	}

	asOf, err := resolveMetricsAsOf(*asOfFlag, metricsNow())
	if err != nil {
		return errf("board metrics: %v", err)
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	def := flow.ForName(cfg.FlowName())

	var res metrics.Result
	// Same lock discipline as runBoardState/runBoardDecisions: a concurrent
	// writer's atomic rename is either fully visible or not visible yet, never
	// half-written.
	if err := lock.WithShared(cfg.Root(), func() error {
		var merr error
		res, merr = metrics.Compute(def, cfg.Root(), cfg, metrics.Options{Project: *project, AsOf: asOf})
		return merr
	}); err != nil {
		return errf("board metrics: %v", err)
	}

	if *jsonFlag {
		doc := metrics.Document{
			Schema:        metrics.CurrentSchema,
			PickleVersion: Version,
			AsOf:          res.AsOf,
			Resolution:    "days",
			Filters:       metrics.Filters{Project: *project, AsOf: res.AsOf},
			Aggregates:    metrics.NonNil(res.Aggregates),
			Intervals:     metrics.NonNil(res.Intervals),
			Issues:        metrics.NonNil(res.Issues),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		// Same reasoning as runBoardState: ticket prose is full of "&", "<" and
		// "→", which HTML-escaping would help no consumer of this CLI-only,
		// non-HTML wire format.
		enc.SetEscapeHTML(false)
		if err := enc.Encode(doc); err != nil {
			return errf("board metrics: %v", err)
		}
		return exitOK
	}

	renderMetricsText(res)
	return exitOK
}

// renderMetricsText prints res as two pasteable tables — the aggregates,
// then every still-open interval (an open_age row, or a lead_time row on a
// done-but-unmerged ticket), oldest first — followed by any data-quality
// issues and a summary line mirroring `board audit`'s own, so the zero case
// prints a clear 0 rather than silence.
func renderMetricsText(res metrics.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tPROJECT\tN\tMIN\tP50\tP90\tMAX")
	for _, a := range res.Aggregates {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\n", a.Metric, a.Project, a.N, a.Min, a.P50, a.P90, a.Max)
	}
	w.Flush()

	// Every still-open interval, nothing truncated (decision 12): an open_age
	// row, or a lead_time row on a done-but-unmerged ticket — one row per open
	// ticket, since a non-terminal ticket no longer also yields an open
	// lead_time row identical to its open_age (T-126 review, F2). Age
	// descending, then ticket id, so the oldest — what is most worth acting
	// on — sorts first.
	open := make([]metrics.Interval, 0, len(res.Intervals))
	for _, iv := range res.Intervals {
		if iv.Open {
			open = append(open, iv)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		if open[i].Days != open[j].Days {
			return open[i].Days > open[j].Days
		}
		return open[i].TicketID < open[j].TicketID
	})

	fmt.Println()
	w = tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "TICKET\tPROJECT\tSTATUS\tMETRIC\tAGE")
	for _, iv := range open {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", iv.TicketID, iv.Project, iv.Status, iv.Metric, iv.Days)
	}
	w.Flush()

	if len(res.Issues) > 0 {
		fmt.Println()
	}
	for _, is := range res.Issues {
		fmt.Printf("ISSUE: %s %s: %s\n", is.Kind, is.TicketID, is.Detail)
	}

	tickets := map[string]bool{}
	for _, iv := range res.Intervals {
		tickets[iv.TicketID] = true
	}
	for _, is := range res.Issues {
		tickets[is.TicketID] = true
	}
	openTickets := map[string]bool{}
	for _, iv := range open {
		openTickets[iv.TicketID] = true
	}
	fmt.Println()
	fmt.Printf("board metrics: %d ticket(s), %d open, as of %s; intervals are whole days\n",
		len(tickets), len(openTickets), res.AsOf)
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
