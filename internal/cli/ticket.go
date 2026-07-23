package cli

import (
	"fmt"
	"os"
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
		return notImplemented("P1", "ticket new",
			"allocate the next T-NNN (one global namespace), scaffold the ticket into 1-to-do/ with project: set, add the board row under that child's sub-group")
	case "move":
		return notImplemented("P3", "ticket move",
			"move a ticket (file + dated History line + board row) atomically; enforce the state machine, per-child WIP limits, and backward-move sign-off")
	default:
		fmt.Fprintf(os.Stderr, "pickle ticket: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}
