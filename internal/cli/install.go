package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/doctor"
	"github.com/codcod/pickle/internal/install"
)

// Setup commands. install is implemented (P2); upgrade/doctor/uninstall follow.

func runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	project := fs.String("project", "", "first child-project name (default: the root dir name)")
	path := fs.String("path", ".", "first child-project path, relative to the install root")
	build := fs.String("build", "", "child build command (optional)")
	test := fs.String("test", "", "child test command (optional)")
	lint := fs.String("lint", "", "child lint command (optional)")
	docs := fs.String("docs", "", "child docs command (optional)")
	noClaude := fs.Bool("no-claude", false, "skip Claude Code artifacts (.claude symlink + CLAUDE.md)")
	claudeSymlink := fs.Bool("claude-symlink", false, "make CLAUDE.md a symlink to AGENTS.md instead of a marker block")
	_ = fs.String("agent", "", "reserved (pi/opencode wiring lands later); Claude is on by default")
	if err := fs.Parse(args); err != nil {
		return exitUsage
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
		Claude:      !*noClaude,
		ClaudeLink:  *claudeSymlink,
	})
	for _, c := range res.Created {
		fmt.Printf("  + %s\n", c)
	}
	for _, s := range res.Skipped {
		fmt.Printf("  = %s\n", s)
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

	fmt.Printf("\npickle installed in %s (child %q). Next: pickle ticket new \"<title>\" --project %s\n",
		root, name, name)
	return exitOK
}

func runUpgrade(_ []string) int {
	return notImplemented("P2", "upgrade",
		"refresh the installed skill payload + marker block to this binary's version")
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

	res := doctor.Check(root, Version)
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

func runUninstall(_ []string) int {
	return notImplemented("P2", "uninstall",
		"remove skill/symlinks/markers; leave tickets/ intact")
}
