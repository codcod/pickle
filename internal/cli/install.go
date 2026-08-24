package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/doctor"
	"github.com/codcod/pickle/internal/hook"
	"github.com/codcod/pickle/internal/install"
)

// Setup commands: install, upgrade, doctor, uninstall.

func runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	project := fs.String("project", "", "first child-project name (default: the root dir name)")
	path := fs.String("path", "", "first child-project path, relative to the install root (registers no child if omitted)")
	inTree := fs.Bool("in-tree", false, "select the in-tree layout: the board lives inside its sole child, registered at \".\"")
	build := fs.String("build", "", "child build command (optional)")
	test := fs.String("test", "", "child test command (optional)")
	lint := fs.String("lint", "", "child lint command (optional)")
	docs := fs.String("docs", "", "child docs command (optional)")
	noClaude := fs.Bool("no-claude", false, "deprecated: use --agent to choose agents (drops claude from the set)")
	claudeSymlink := fs.Bool("claude-symlink", false, "make CLAUDE.md a symlink to AGENTS.md instead of a marker block")
	agentSpec := fs.String("agent", "", `comma-separated agents to wire up: claude, opencode, pi (default "claude")`)
	hooks := fs.Bool("hooks", false, "also install the bookkeeping guard hooks (same as `pickle hooks install`)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	// The layout is never inferred (T-108 decision 1): --in-tree is the only
	// way to select the in-tree layout, and it implies the sole child at ".".
	// A bare --path "." without --in-tree is refused rather than silently
	// treated as in-tree, so the choice is always the one the caller stated.
	childFlagsGiven := *project != "" || *build != "" || *test != "" || *lint != "" || *docs != ""
	switch {
	case *inTree && *path != "" && *path != ".":
		fmt.Fprintf(os.Stderr, "pickle install: --in-tree registers the child at \".\"; --path %q conflicts with it\n", *path)
		return exitUsage
	case *inTree:
		*path = "."
	case *path == ".":
		fmt.Fprintln(os.Stderr, "pickle install: --path \".\" selects the in-tree layout; pass --in-tree explicitly")
		return exitUsage
	case *path == "" && childFlagsGiven:
		// Silently discarding an explicit --project/--build/--test/--lint/--docs
		// is exactly the kind of guess this ticket exists to remove (T-108
		// decision 1's spirit, extended): say so instead.
		fmt.Fprintln(os.Stderr, "pickle install: --project/--build/--test/--lint/--docs need --path (or --in-tree) to name a child; omit them to install the umbrella layout with no child yet")
		return exitUsage
	}

	spec := *agentSpec
	if spec == "" {
		spec = "claude"
	}
	agents, err := install.ParseAgents(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle install: %v\n", err)
		return exitUsage
	}
	if *noClaude {
		fmt.Fprintln(os.Stderr, "pickle install: --no-claude is deprecated; use --agent to choose agents (e.g. --agent opencode)")
		agents.Claude = false
	}

	root, err := os.Getwd()
	if err != nil {
		return errf("%v", err)
	}
	// A name is only meaningful when a child is actually being registered
	// (T-108 decision 2): *path == "" is the umbrella layout's fresh-install
	// state, with no child yet.
	var name string
	if *path != "" {
		name = *project
		if name == "" {
			name = filepath.Base(root)
		}
	}

	res, err := install.Run(Payload, root, Version, install.Options{
		ProjectName: name,
		ProjectPath: *path,
		Build:       *build,
		Test:        *test,
		Lint:        *lint,
		Docs:        *docs,
		InTree:      *inTree,
		Agents:      agents,
		ClaudeLink:  *claudeSymlink,
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

	// Post-install self-check: a correct install must be board-audit-clean
	// (zero errors; a warning — e.g. layout-only board staleness, T-052 — is
	// printed but does not fail the install).
	cfgPath, err := config.Find(root)
	if err != nil {
		return errf("%v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errf("%v", err)
	}
	a := audit.Audit(cfg.Root(), cfg)
	for _, w := range a.Warnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}
	if len(a.Errors) > 0 {
		for _, e := range a.Errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		return errf("post-install audit found %d error(s)", len(a.Errors))
	}

	// The child registered at *path ("." by default) just appeared on disk;
	// if it is not "." and this repository would still stage it whole, say so
	// now — the same sentence `pickle doctor` would warn with later, and the
	// same helper `project add` uses (T-051).
	if p, ok := cfg.Project(name); ok {
		noteIfStageable(cfg.Root(), *p)
	}

	// The hooks are opt-in and deliberately not part of a plain install: they
	// write outside the install root (into .git/) and only help in a repo that
	// carries both the code and the board. A failure here is a warning, never a
	// failed install — the scaffolding is already on disk and correct.
	if *hooks {
		// Report every attempted hook regardless of herr (T-082 finding F4): a
		// per-hook failure (a foreign hook without --force) must not swallow the
		// report of a sibling hook InstallAll already wrote — only ErrNoRepo
		// (ErrNoRepo) leaves results empty, the same shape a lone hook.Install
		// call would report as a bare error.
		results, herr := hook.InstallAll(root, false)
		anyArmed := false
		for _, hres := range results {
			if hres.Changed {
				fmt.Printf("  + %s\n", hres.Path)
				anyArmed = true
			} else if hres.Path != "" {
				fmt.Printf("  = %s (%s)\n", hres.Path, hres.Skipped)
			}
		}
		if herr != nil {
			fmt.Fprintf(os.Stderr, "pickle install: hooks install skipped: %v\n", herr)
		}
		// Same PATH-capability check `hooks install` runs on its own (T-068):
		// this is the other moment the evidence exists and the user is looking.
		// Runs whenever every hook installed cleanly, or at least one still
		// landed despite a sibling failure (a foreign hook refused without
		// --force) — a per-hook error must not suppress the sibling's report
		// (T-082 finding F4). Neither holds when the repository did not exist
		// at all (ErrNoRepo): nothing was ever armed to have an opinion about.
		if herr == nil || anyArmed {
			warnIfInert("pickle install")
		}
	}

	if *path != "" {
		fmt.Printf("\npickle installed in %s (child %q). Next: pickle ticket new \"<title>\" --project %s\n",
			root, name, name)
	} else {
		fmt.Printf("\npickle installed in %s (umbrella layout, no child registered yet). Next: pickle project add\n", root)
	}
	return exitOK
}

func runUpgrade(args []string) int {
	// Takes no flags, but must still parse: silently ignoring argv would make
	// `pickle upgrade -h` (or a typo, or a guessed `-n`) perform a real upgrade.
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "pickle upgrade: unexpected argument %q\n", fs.Arg(0))
		return exitUsage
	}

	root, code := projectRoot()
	if code != exitOK {
		return code
	}
	cfgPath := filepath.Join(root, config.FileName)

	res, err := install.Upgrade(Payload, root, Version)
	// Removed first: a pre-brine legacy path swept away (T-074) is the one
	// thing here that is not a routine refresh, so it reads before the
	// created/skipped lines rather than being buried under them.
	for _, r := range res.Removed {
		fmt.Printf("  - %s\n", r)
	}
	for _, c := range res.Created {
		fmt.Printf("  + %s\n", c)
	}
	for _, s := range res.Skipped {
		fmt.Printf("  = %s\n", s)
	}
	if err != nil {
		return errf("%v", err)
	}

	// Post-upgrade self-check: an upgrade must leave the project board-audit-clean
	// (zero errors; a warning — e.g. layout-only board staleness, T-052 — is
	// printed but does not fail the upgrade).
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errf("%v", err)
	}
	a := audit.Audit(cfg.Root(), cfg)
	for _, w := range a.Warnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}
	if len(a.Errors) > 0 {
		for _, e := range a.Errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		return errf("post-upgrade audit found %d error(s)", len(a.Errors))
	}

	if res.PrevVersion == cfg.PayloadVersion {
		fmt.Printf("\npickle upgrade: already at %s (refreshed payload + markers)\n", cfg.PayloadVersion)
	} else {
		fmt.Printf("\npickle upgrade: %s -> %s\n", res.PrevVersion, cfg.PayloadVersion)
	}
	return exitOK
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	verbose := fs.Bool("verbose", false, "also list the checks that passed")
	fs.BoolVar(verbose, "v", false, "also list the checks that passed (shorthand)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	root, code := projectRoot()
	if code != exitOK {
		return code
	}

	res := doctor.Check(root, Version, Payload)
	if *verbose {
		for _, p := range res.Passed {
			fmt.Printf("ok: %s\n", p)
		}
	}
	for _, w := range res.Warnings {
		fmt.Printf("WARNING: %s\n", w)
	}
	for _, e := range res.Errors {
		fmt.Printf("ERROR: %s\n", e)
	}
	fmt.Printf("pickle doctor: %d error(s), %d warning(s)\n", len(res.Errors), len(res.Warnings))
	if len(res.Errors) > 0 {
		return exitError
	}
	return exitOK
}

func runUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "list what would be removed without changing anything")
	fs.BoolVar(dryRun, "n", false, "list what would be removed without changing anything (shorthand)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "pickle uninstall: unexpected argument %q\n", fs.Arg(0))
		return exitUsage
	}

	root, code := projectRoot()
	if code != exitOK {
		return code
	}

	res, err := install.Uninstall(Payload, root, install.UninstallOptions{DryRun: *dryRun})
	for _, r := range res.Removed {
		fmt.Printf("  - %s\n", r)
	}
	for _, s := range res.Skipped {
		fmt.Printf("  = %s\n", s)
	}
	if err != nil {
		return errf("%v", err)
	}

	if *dryRun {
		fmt.Printf("\npickle uninstall --dry-run: %d item(s) would be removed; nothing changed\n", len(res.Removed))
	} else {
		fmt.Printf("\npickle uninstall: removed %d item(s); tickets/ and pickle.toml left intact\n", len(res.Removed))
	}
	return exitOK
}
