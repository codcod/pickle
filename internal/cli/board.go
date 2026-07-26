package cli

import (
	"fmt"
	"os"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/sync"
)

// Board mechanics. `board audit` is the keystone (P1): a pure check over tickets/
// + pickle.toml. `board sync` (P3) regenerates the board from ticket state.

func runBoard(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle board <audit|sync> ...")
		return exitUsage
	}
	switch args[0] {
	case "audit":
		return runBoardAudit(args[1:])
	case "sync":
		return runBoardSync(args[1:])
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
