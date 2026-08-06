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
	path := fs.String("path", ".", "first child-project path, relative to the install root")
	build := fs.String("build", "", "child build command (optional)")
	test := fs.String("test", "", "child test command (optional)")
	lint := fs.String("lint", "", "child lint command (optional)")
	docs := fs.String("docs", "", "child docs command (optional)")
	noClaude := fs.Bool("no-claude", false, "deprecated: use --agent to choose agents (drops claude from the set)")
	claudeSymlink := fs.Bool("claude-symlink", false, "make CLAUDE.md a symlink to AGENTS.md instead of a marker block")
	agentSpec := fs.String("agent", "", `comma-separated agents to wire up: claude, opencode, pi (default "claude")`)
	hooks := fs.Bool("hooks", false, "also install the pre-commit bookkeeping guard (same as `pickle hooks install`)")
	if err := fs.Parse(args); err != nil {
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
	name := *project
	if name == "" {
		name = filepath.Base(root)
	}

	res, err := install.Run(Payload, root, Version, install.Options{
		ProjectName: name,
		ProjectPath: *path,
		Build:       *build,
		Test:        *test,
		Lint:        *lint,
		Docs:        *docs,
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

	// Post-install self-check: a correct install must be board-audit-clean.
	cfgPath, err := config.Find(root)
	if err != nil {
		return errf("%v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errf("%v", err)
	}
	a := audit.Audit(cfg.Root(), cfg)
	if len(a.Errors) > 0 {
		for _, e := range a.Errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		return errf("post-install audit found %d error(s)", len(a.Errors))
	}

	// The hook is opt-in and deliberately not part of a plain install: it writes
	// outside the install root (into .git/) and only helps in a repo that carries
	// both the code and the board. A failure here is a warning, never a failed
	// install — the scaffolding is already on disk and correct.
	if *hooks {
		if hres, herr := hook.Install(root, false); herr != nil {
			fmt.Fprintf(os.Stderr, "pickle install: hooks install skipped: %v\n", herr)
		} else {
			if hres.Changed {
				fmt.Printf("  + %s\n", hres.Path)
			} else {
				fmt.Printf("  = %s (%s)\n", hres.Path, hres.Skipped)
			}
			// Same PATH-capability check `hooks install` runs on its own (T-068):
			// this is the other moment the evidence exists and the user is looking.
			warnIfInert("pickle install")
		}
	}

	fmt.Printf("\npickle installed in %s (child %q). Next: pickle ticket new \"<title>\" --project %s\n",
		root, name, name)
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

	wd, err := os.Getwd()
	if err != nil {
		return errf("%v", err)
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		return errf("%v", err)
	}
	root := filepath.Dir(cfgPath)

	before, err := config.Load(cfgPath)
	if err != nil {
		return errf("%v", err)
	}
	prevVersion := before.PayloadVersion

	res, err := install.Upgrade(Payload, root, Version)
	for _, c := range res.Created {
		fmt.Printf("  + %s\n", c)
	}
	for _, s := range res.Skipped {
		fmt.Printf("  = %s\n", s)
	}
	if err != nil {
		return errf("%v", err)
	}

	// Post-upgrade self-check: an upgrade must leave the project board-audit-clean.
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errf("%v", err)
	}
	a := audit.Audit(cfg.Root(), cfg)
	if len(a.Errors) > 0 {
		for _, e := range a.Errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		return errf("post-upgrade audit found %d error(s)", len(a.Errors))
	}

	if prevVersion == cfg.PayloadVersion {
		fmt.Printf("\npickle upgrade: already at %s (refreshed payload + markers)\n", cfg.PayloadVersion)
	} else {
		fmt.Printf("\npickle upgrade: %s -> %s\n", prevVersion, cfg.PayloadVersion)
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

	wd, err := os.Getwd()
	if err != nil {
		return errf("%v", err)
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		return errf("%v", err)
	}
	root := filepath.Dir(cfgPath)

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

	wd, err := os.Getwd()
	if err != nil {
		return errf("%v", err)
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		return errf("%v", err)
	}
	root := filepath.Dir(cfgPath)

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
