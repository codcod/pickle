package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/state"
	"github.com/codcod/pickle/internal/sync"
)

// Board mechanics. `board audit` is the keystone (P1): a pure check over tickets/
// + pickle.toml. `board sync` (P3) regenerates the board from ticket state.
// `board state --json` (T-065) is the read-only, versioned JSON projection —
// no mechanics of its own, it only reads what audit/sync/board already own.

func runBoard(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle board <audit|sync|state> ...")
		return exitUsage
	}
	switch args[0] {
	case "audit":
		return runBoardAudit(args[1:])
	case "sync":
		return runBoardSync(args[1:])
	case "state":
		return runBoardState(args[1:])
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
