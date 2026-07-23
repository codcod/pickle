package cli

import (
	"fmt"
	"os"
)

// Child-project registry commands. Implemented in P2 — the [[project]] array in
// pickle.toml is the source of truth for the connected child-projects.

func runProject(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle project <add|list|remove> ...")
		return exitUsage
	}
	switch args[0] {
	case "add":
		return notImplemented("P2", "project add",
			"register a connected child-project (name + path + build/validate + branch policy + per-child WIP) in pickle.toml")
	case "list":
		return notImplemented("P2", "project list",
			"list registered child-projects")
	case "remove":
		return notImplemented("P2", "project remove",
			"unregister a child-project (refused while any live ticket targets it)")
	default:
		fmt.Fprintf(os.Stderr, "pickle project: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}
