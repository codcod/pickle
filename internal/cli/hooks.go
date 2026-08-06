package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/hook"
)

// Hook commands: pickle hooks install | uninstall | status | run <hook>.

// runHooks dispatches `pickle hooks <subcommand>`.
func runHooks(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "pickle hooks: expected install, uninstall, status or run")
		return exitUsage
	}
	switch args[0] {
	case "install":
		return runHooksInstall(args[1:])
	case "uninstall":
		return runHooksUninstall(args[1:])
	case "status":
		return runHooksStatus(args[1:])
	case "run":
		return runHooksRun(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle hooks: unknown subcommand %q (want install, uninstall, status or run)\n", args[0])
		return exitUsage
	}
}

// hookRoot locates the install root (the directory holding pickle.toml).
func hookRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		return "", err
	}
	return filepath.Dir(cfgPath), nil
}

func runHooksInstall(args []string) int {
	fs := flag.NewFlagSet("hooks install", flag.ContinueOnError)
	force := fs.Bool("force", false, "replace a pre-commit hook pickle does not own")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	root, err := hookRoot()
	if err != nil {
		return errf("%v", err)
	}
	res, err := hook.Install(root, *force)
	if err != nil {
		return errf("hooks install: %v", err)
	}
	if res.Changed {
		fmt.Printf("  + %s\n", res.Path)
		fmt.Printf("\npickle hooks install: %s guard armed. Hooks live in .git/ and are never cloned,\n"+
			"so each clone needs this once.\n", hook.HookName)
	} else {
		fmt.Printf("  = %s (%s)\n", res.Path, res.Skipped)
	}
	return exitOK
}

func runHooksUninstall(args []string) int {
	fs := flag.NewFlagSet("hooks uninstall", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "list what would be removed without changing anything")
	fs.BoolVar(dryRun, "n", false, "list what would be removed without changing anything (shorthand)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	root, err := hookRoot()
	if err != nil {
		return errf("%v", err)
	}
	res, err := hook.Uninstall(root, *dryRun)
	if err != nil {
		return errf("hooks uninstall: %v", err)
	}
	switch {
	case res.Would:
		fmt.Printf("  - %s (dry-run)\n", res.Path)
	case res.Changed:
		fmt.Printf("  - %s\n", res.Path)
	case res.Path != "":
		fmt.Printf("  = %s (%s)\n", res.Path, res.Skipped)
	default:
		fmt.Printf("  = %s\n", res.Skipped)
	}
	return exitOK
}

func runHooksStatus(args []string) int {
	fs := flag.NewFlagSet("hooks status", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	root, err := hookRoot()
	if err != nil {
		return errf("%v", err)
	}
	st, err := hook.Status(root)
	if err != nil {
		return errf("hooks status: %v", err)
	}
	switch st.Kind {
	case hook.KindNoRepo:
		fmt.Printf("%s: no git repository at %s — nothing to install into\n", hook.HookName, root)
	case hook.KindAbsent:
		fmt.Printf("%s: absent (%s)\n", hook.HookName, st.Path)
		fmt.Printf("  install it with `pickle hooks install`\n")
	case hook.KindForeign:
		fmt.Printf("%s: present but not pickle's (%s)\n", hook.HookName, st.Path)
		fmt.Printf("  pickle leaves it alone; chain the guard with `pickle hooks run %s || exit 1`\n", hook.HookName)
	case hook.KindOwned:
		state := "current"
		if st.Stale {
			state = fmt.Sprintf("stale, v%d — run `pickle upgrade`", st.Version)
		}
		fmt.Printf("%s: installed by pickle, %s (%s)\n", hook.HookName, state, st.Path)
	}
	return exitOK
}

// runHooksRun is the shim's entry point, and its exit codes are a contract the
// installed shim depends on (T-057 decision 3):
//
//	1 — a real violation, and nothing else
//	0 — the commit is fine, *or* the guard could not run (no pickle.toml,
//	    unparseable config, no git, not a repository)
//	2 — bad invocation
//
// Reserving 1 for violations is what makes version skew harmless: an older
// pickle on PATH exits 2 on the unknown `hooks` verb, and the shim treats
// anything other than 1 as "not a violation" instead of blocking every commit.
func runHooksRun(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "pickle hooks run: expected a hook name (%s)\n", hook.HookName)
		return exitUsage
	}
	if args[0] != hook.HookName {
		fmt.Fprintf(os.Stderr, "pickle hooks run: unknown hook %q (only %s is supported)\n", args[0], hook.HookName)
		return exitUsage
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "pickle hooks run: unexpected argument %q\n", args[1])
		return exitUsage
	}

	wd, err := os.Getwd()
	if err != nil {
		return exitOK // cannot even look: never block the commit
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		// Not a pickle project. Silent: a hook that nags on every commit in an
		// unrelated repository gets deleted, and there is nothing to guard here.
		return exitOK
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// A broken pickle.toml *is* worth one line — the guard is off and the
		// user is the only one who can fix it — but it still must not block.
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	ok, err := hook.PreCommit(cfg, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	if !ok {
		return exitError
	}
	return exitOK
}
