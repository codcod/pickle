package cli

import (
	"fmt"
	"os"
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
		return notImplemented("P1", "board audit",
			"check every invariant: one board row per ticket in the right section, unique ids matching filenames, complete frontmatter with legal grades, project: is a registered child, depends-on targets exist, per-child WIP limits, last History matches directory, in-dev deps done+merged (in their own child repo)")
	case "sync":
		return notImplemented("P3", "board sync",
			"regenerate/repair board rows from ticket frontmatter + locations (escape hatch when hand-edits drift)")
	default:
		fmt.Fprintf(os.Stderr, "pickle board: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}
