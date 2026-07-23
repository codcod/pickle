package cli

import (
	"fmt"
	"os"

	"pickle/internal/audit"
)

// Board mechanics. `board audit` is the keystone (P1): a pure check over tickets/
// + pickle.toml. `board sync` (P3) repairs the hand-maintained board from state.

func runBoard(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle board <audit|sync> ...")
		return exitUsage
	}
	switch args[0] {
	case "audit":
		return runBoardAudit(args[1:])
	case "sync":
		return notImplemented("P3", "board sync",
			"regenerate/repair board rows from ticket frontmatter + locations (escape hatch when hand-edits drift)")
	default:
		fmt.Fprintf(os.Stderr, "pickle board: unknown subcommand %q\n", args[0])
		return exitUsage
	}
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
