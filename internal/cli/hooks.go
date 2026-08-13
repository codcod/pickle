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

// warnIfInert reports (to stderr, never affecting the caller's exit code) when
// the `pickle` the just-armed shim will actually resolve from PATH cannot run
// it (T-068) — the shim was written correctly regardless of what happens to be
// first on PATH right now, so this is advisory, at the one moment (install)
// the evidence exists and the user is looking.
//
// Probe() answers a per-binary question ("can the pickle on PATH dispatch
// `hooks run` at all"), not a per-hook one — it stays keyed to pre-commit
// (T-082 decision 6) and is called at most once per install, regardless of
// how many hooks were just armed.
func warnIfInert(prefix string) {
	p := hook.Probe().Problem()
	if p == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: warning: %s\n", prefix, p)
	if exe, err := os.Executable(); err == nil {
		fmt.Fprintf(os.Stderr, "  this binary is %s — put it first on PATH, or upgrade the pickle that is\n", exe)
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
	force := fs.Bool("force", false, "replace a hook pickle does not own")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	root, err := hookRoot()
	if err != nil {
		return errf("%v", err)
	}
	results, err := hook.InstallAll(root, *force)
	// Report every attempted hook before ever branching on err (T-082 finding
	// F4): a per-hook failure (a foreign hook without --force, say) must not
	// suppress the sibling hook's own report, or the inert-PATH check below,
	// which is unrelated to whether every hook installed cleanly.
	anyChanged := false
	for _, res := range results {
		switch {
		case res.Changed:
			fmt.Printf("  + %s\n", res.Path)
			anyChanged = true
		case res.Path != "":
			fmt.Printf("  = %s (%s)\n", res.Path, res.Skipped)
		}
	}
	if anyChanged {
		fmt.Printf("\npickle hooks install: guard(s) armed. Hooks live in .git/ and are never cloned,\n" +
			"so each clone needs this once.\n")
	}
	// The PATH-capability check runs whenever every hook installed cleanly
	// (the original, unconditional case — including a no-op re-run against
	// already-current shims, where the evidence on disk is just as real as a
	// freshly written one) *or* at least one hook still landed despite a
	// sibling failure (a foreign hook refused without --force, say): a
	// per-hook error must not suppress the report for the hook that did
	// install (T-082 finding F4).
	if err == nil || anyChanged {
		warnIfInert("pickle hooks install")
	}
	if err != nil {
		return errf("hooks install: %v", err)
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
	results, err := hook.UninstallAll(root, *dryRun)
	for _, res := range results {
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
	}
	if err != nil {
		return errf("hooks uninstall: %v", err)
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
	states, err := hook.StatusAll(root)
	if err != nil {
		return errf("hooks status: %v", err)
	}
	// hook.Probe() answers a per-binary question (can the `pickle` on PATH
	// dispatch `hooks run` at all), not a per-hook one — both hooks ship in
	// the same binary (T-082 decision 6, and its own review finding F3: this
	// used to probe, and print, once per owned hook). Probed at most once
	// here, and only when at least one hook is owned; every owned hook is
	// printed the one shared Reach.
	needsProbe := false
	for _, st := range states {
		if st.Kind == hook.KindOwned {
			needsProbe = true
			break
		}
	}
	var reach hook.Reach
	if needsProbe {
		reach = hook.Probe()
	}
	for _, st := range states {
		printHookStatus(root, st, reach)
	}
	return exitOK
}

// printHookStatus renders one hook's State the way `hooks status` reports it.
// A KindNoRepo State (StatusAll returns exactly one, whichever hook happened
// to be first) is reported once, for the repository, not per hook. reach is
// the single, at-most-once-probed hook.Reach runHooksStatus computed;
// printHookStatus never probes on its own.
func printHookStatus(root string, st hook.State, reach hook.Reach) {
	switch st.Kind {
	case hook.KindNoRepo:
		fmt.Printf("no git repository at %s — nothing to install into\n", root)
	case hook.KindAbsent:
		fmt.Printf("%s: absent (%s)\n", st.Name, st.Path)
		fmt.Printf("  install it with `pickle hooks install`\n")
	case hook.KindForeign:
		fmt.Printf("%s: present but not pickle's (%s)\n", st.Name, st.Path)
		fmt.Printf("  pickle leaves it alone; chain the guard with `pickle hooks run %s || exit 1`\n", st.Name)
	case hook.KindOwned:
		state := "current"
		if st.Stale {
			state = fmt.Sprintf("stale, v%d — run `pickle upgrade`", st.Version)
		}
		fmt.Printf("%s: installed by pickle, %s (%s)\n", st.Name, state, st.Path)
		if p := reach.Problem(); p != "" {
			fmt.Printf("  %s\n", p)
		} else {
			fmt.Printf("  PATH: %s can run the guard\n", reach.Path)
		}
	}
}

// runHooksRun is the shim's entry point, and its exit codes are a contract the
// installed shims depend on (T-057 decision 3):
//
//	1 — a real violation, and nothing else
//	0 — the commit/push is fine, *or* the guard could not run (no
//	    pickle.toml, unparseable config, no git, not a repository)
//	2 — bad invocation
//
// Reserving 1 for violations is what makes version skew harmless: an older
// pickle on PATH exits 2 on the unknown `hooks` verb, and the shim treats
// anything other than 1 as "not a violation" instead of blocking every commit
// or push. Both hooks share this contract.
func runHooksRun(args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "pickle hooks run: expected a hook name (%s or %s)\n", hook.PreCommit, hook.PrePush)
		return exitUsage
	}
	switch hook.Name(args[0]) {
	case hook.PreCommit:
		return runHooksRunPreCommit(args[1:])
	case hook.PrePush:
		return runHooksRunPrePush(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle hooks run: unknown hook %q (want %s or %s)\n", args[0], hook.PreCommit, hook.PrePush)
		return exitUsage
	}
}

// hookConfig resolves pickle.toml for the working directory a hook runs in.
// Three outcomes matter differently to callers: no cwd or no pickle.toml is
// silent (nothing to guard — a hook that nags in an unrelated repository gets
// deleted), while an unparseable pickle.toml is worth one stderr line, since
// the guard is off and the user is the only one who can fix it. Neither ever
// blocks: cfg is nil in both the silent and the error case.
func hookConfig() (cfg *config.Config, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, nil // cannot even look: never block
	}
	cfgPath, err := config.Find(wd)
	if err != nil {
		return nil, nil // not a pickle project
	}
	cfg, err = config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func runHooksRunPreCommit(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "pickle hooks run: unexpected argument %q\n", args[0])
		return exitUsage
	}
	cfg, err := hookConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	if cfg == nil {
		return exitOK
	}
	ok, err := hook.CheckPreCommit(cfg, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	if !ok {
		return exitError
	}
	return exitOK
}

// runHooksRunPrePush accepts up to two positional arguments — git's own `$1
// $2` (<remote-name> [<remote-url>]) — and reads the ref list from stdin. A
// missing remote name defaults to "origin", the same default `git push` uses.
func runHooksRunPrePush(args []string) int {
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "pickle hooks run: unexpected argument %q\n", args[2])
		return exitUsage
	}
	remote := "origin"
	if len(args) > 0 && args[0] != "" {
		remote = args[0]
	}
	cfg, err := hookConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	if cfg == nil {
		return exitOK
	}
	refs, err := hook.ParsePushRefs(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	ok, err := hook.CheckPrePush(cfg, remote, refs, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pickle: bookkeeping guard skipped (%v)\n", err)
		return exitOK
	}
	if !ok {
		return exitError
	}
	return exitOK
}
