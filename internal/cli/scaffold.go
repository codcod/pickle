package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/codcod/pickle/internal/scaffold"
)

// Scaffold commands: pickle scaffold docs. Entirely separate from `pickle
// install` — nothing here touches brine (tickets/, BOARD.md, the skill, the
// AGENTS.md marker block), and pickle.toml is neither required nor written.

// runScaffold dispatches `pickle scaffold <subcommand>`.
func runScaffold(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "pickle scaffold: expected docs")
		return exitUsage
	}
	switch args[0] {
	case "docs":
		return runScaffoldDocs(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle scaffold: unknown subcommand %q (want docs)\n", args[0])
		return exitUsage
	}
}

func runScaffoldDocs(args []string) int {
	fs := flag.NewFlagSet("scaffold docs", flag.ContinueOnError)
	projectName := fs.String("project-name", "", "name substituted into the scaffolded docs (default: the current directory's name)")
	force := fs.Bool("force", false, "overwrite files that already exist")
	dryRun := fs.Bool("dry-run", false, "list what would be created/changed, changing nothing")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	root, err := os.Getwd()
	if err != nil {
		return errf("%v", err)
	}

	res, err := scaffold.Docs(Payload, root, scaffold.Options{
		ProjectName: *projectName,
		Force:       *force,
		DryRun:      *dryRun,
	})
	for _, c := range res.Created {
		fmt.Printf("  + %s\n", c)
	}
	for _, s := range res.Skipped {
		fmt.Printf("  = %s\n", s)
	}
	for _, n := range res.Notes {
		fmt.Printf("\n%s\n", n)
	}
	if err != nil {
		return errf("%v", err)
	}
	return exitOK
}
