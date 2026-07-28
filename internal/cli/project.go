package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// Child-project registry commands. The [[project]] array in pickle.toml is the
// source of truth for the connected child-projects.

func runProject(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle project <add|list|remove> ...")
		return exitUsage
	}
	switch args[0] {
	case "add":
		return runProjectAdd(args[1:])
	case "list":
		return runProjectList(args[1:])
	case "remove":
		return runProjectRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle project: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

// loadConfig finds and loads pickle.toml from the cwd upward.
func loadConfig() (*config.Config, int) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errf("%v", err)
	}
	path, err := config.Find(wd)
	if err != nil {
		return nil, errf("%v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, errf("%v", err)
	}
	return cfg, exitOK
}

func runProjectAdd(args []string) int {
	// name and path are the two leading positionals; flags follow them (the stdlib
	// flag package stops at the first positional, so we split explicitly).
	if len(args) < 2 || strings.HasPrefix(args[0], "-") || strings.HasPrefix(args[1], "-") {
		fmt.Fprintln(os.Stderr, "usage: pickle project add <name> <path> [flags]")
		return exitUsage
	}
	name, path := args[0], args[1]

	fs := flag.NewFlagSet("project add", flag.ContinueOnError)
	build := fs.String("build", "", "build command")
	test := fs.String("test", "", "test command")
	lint := fs.String("lint", "", "lint command")
	docs := fs.String("docs", "", "docs command")
	branch := fs.String("branch-prefix", config.DefaultBranchPrefix, "feature branch prefix")
	ticketPrefix := fs.String("ticket-prefix", config.DefaultTicketPrefix, "per-child ticket id prefix (e.g. RICK → RICK-001)")
	wipDev := fs.Int("wip-dev", config.DefaultWIPInDevelopment, "WIP limit for in-development")
	wipRev := fs.Int("wip-review", config.DefaultWIPInReview, "WIP limit for in-review")
	if err := fs.Parse(args[2:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle project add <name> <path> [flags]")
		return exitUsage
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	abs := filepath.Join(cfg.Root(), path)
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return errf("path %q does not resolve to a directory under %s", path, cfg.Root())
	}
	p := config.Project{
		Name: name, Path: path,
		Build: *build, Test: *test, Lint: *lint, Docs: *docs,
		TicketPrefix: *ticketPrefix, BranchPrefix: *branch,
		WIPInDevelopment: *wipDev, WIPInReview: *wipRev,
	}
	if err := cfg.AddProject(p); err != nil {
		return errf("%v", err)
	}
	if err := cfg.Save(""); err != nil {
		return errf("%v", err)
	}
	fmt.Printf("registered child-project %q at %s\n", name, path)
	return exitOK
}

func runProjectList(_ []string) int {
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tPREFIX\tBUILD\tTEST\tLINT\tWIP(dev/review)")
	for _, p := range cfg.Projects {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d/%d\n",
			p.Name, p.Path, p.Prefix(), dash(p.Build), dash(p.Test), dash(p.Lint), p.WIPInDevelopment, p.WIPInReview)
	}
	w.Flush()
	return exitOK
}

func runProjectRemove(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: pickle project remove <name>")
		return exitUsage
	}
	name := args[0]
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	if _, ok := cfg.Project(name); !ok {
		return errf("project %q is not registered", name)
	}
	if live := liveTicketsTargeting(cfg.Root(), name); len(live) > 0 {
		return errf("refusing to remove %q: %d live ticket(s) target it: %s",
			name, len(live), strings.Join(live, ", "))
	}
	if err := cfg.RemoveProject(name); err != nil {
		return errf("%v", err)
	}
	if err := cfg.Save(""); err != nil {
		return errf("%v", err)
	}
	fmt.Printf("removed child-project %q\n", name)
	return exitOK
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// liveTicketsTargeting returns the ids of tickets in a non-terminal status dir
// whose project: frontmatter is name. It reuses the shared loader so the
// frontmatter scan lives in exactly one place (internal/ticket).
func liveTicketsTargeting(root, name string) []string {
	tickets, _ := ticket.LoadAll(root)
	var hits []string
	for _, t := range tickets {
		if st, ok := ticket.StatusByDir(t.Dir); ok && st.Terminal {
			continue
		}
		if t.Project() == name {
			hits = append(hits, t.Base())
		}
	}
	sort.Strings(hits)
	return hits
}
