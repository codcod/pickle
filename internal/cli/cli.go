// Package cli implements pickle's command surface: the top-level dispatcher and
// one handler per command. Every command in the dispatch table is implemented
// (see README.md); the dispatch table, usage text, exit codes, and command
// grouping are stable.
package cli

import (
	"fmt"
	"io/fs"
	"os"
)

// Payload is the embedded skill payload (the resources/ tree), injected by main.
// install and upgrade both read from it to (re-)write the on-disk skill copy;
// doctor and uninstall only ever inspect/modify the on-disk install, never the
// embedded payload.
var Payload fs.FS

// Version is the build version, injected by main.
var Version = "dev"

// Exit codes.
const (
	exitOK    = 0 // success
	exitError = 1 // runtime error (bad config, I/O, rejected operation)
	exitUsage = 2 // bad invocation / unknown command
)

// errf prints a pickle error to stderr and returns the error exit code.
func errf(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "pickle: "+format+"\n", a...)
	return exitError
}

// Run dispatches args (os.Args[1:]) to a command and returns a process exit code.
func Run(payload fs.FS, version string, args []string) int {
	Payload = payload
	if version != "" {
		Version = version
	}

	if len(args) == 0 {
		usage(os.Stderr)
		return exitUsage
	}

	switch args[0] {
	case "install":
		return runInstall(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "hooks":
		return runHooks(args[1:])
	case "project":
		return runProject(args[1:])
	case "flow":
		return runFlow(args[1:])
	case "ticket":
		return runTicket(args[1:])
	case "board":
		return runBoard(args[1:])
	case "changelog":
		return runChangelog(args[1:])
	case "serve":
		return runServe(args[1:])
	case "scaffold":
		return runScaffold(args[1:])
	case "version", "--version", "-v":
		return runVersion(args[1:])
	case "help", "--help", "-h":
		usage(os.Stdout)
		return exitOK
	default:
		// Terse, deliberately (T-068): a hooks-aware shim degrades on an unknown
		// verb from an *older* pickle first on PATH (T-057 decision 3), and that
		// binary dumping its full usage() text ahead of the shim's one real
		// notice buries the message that matters under noise on every commit.
		// The no-argument path above keeps the full usage — this is only for a
		// command that was typed and not recognised.
		fmt.Fprintf(os.Stderr, "pickle: unknown command %q — run `pickle help`\n", args[0])
		return exitUsage
	}
}

func usage(w *os.File) {
	fmt.Fprint(w, `pickle — install and operate a ticket-based feature flow in any project.

Usage:
  pickle <command> [arguments]

Setup commands:
  install                 Scaffold tickets/, install the skill for detected agents,
                          inject AGENTS.md/CLAUDE.md markers, write pickle.toml, and
                          register the first child-project.
  project add <name> <path>   Register another connected child-project.
  project list                List registered child-projects.
  project remove <name>       Unregister a child-project (refused if it has live tickets).
  flow show                   Print the configured flow name.
  flow list                   List available flow names (exactly one, today).
  upgrade                 Refresh the installed skill payload + marker block to this
                          binary's version (never touches tickets).
  doctor                  Verify install integrity (skill, symlinks, markers, child paths).
  uninstall [--dry-run]   Remove skill/symlinks/markers; leave tickets/ and pickle.toml
                          intact. --dry-run (-n) lists what would go, changing nothing.
  hooks install [--force]     Install the pre-commit and pre-push guards that refuse
                              ticket bookkeeping staged (or pushed) on a feature branch.
                              Once per clone.
  hooks uninstall [--dry-run] Remove pickle's hooks (a foreign hook is left alone).
  hooks status                Report each hook's state and resolved path.
  hooks run pre-commit        Run the pre-commit guard (the installed shim's entry point).
  hooks run pre-push          Run the pre-push guard (the installed shim's entry point).
                              Both exit 1 only for a violation.

Flow commands:
  ticket new "<title>" --project <name> [--impact .. --complexity .. --cost ..]
                       [--spawned-by "T-NNN[,T-MMM]"] [--family T-NNN]
                          Allocate the next T-NNN, scaffold the ticket, regenerate the board.
                          --spawned-by records lineage (never gates pickup).
                          --family groups the ticket under an umbrella id for board
                          ordering (single id, same child; never gates pickup).
  ticket move T-NNN <status> --reason "<why>"
                          Move a ticket (file + History + board regeneration) atomically.
  board audit             Check the ticket invariants + board freshness (exit non-zero on any error).
  board sync              Regenerate BOARD.md from ticket frontmatter + locations.
  board state --json      Print the whole ticket tree as one versioned JSON document
                          (schema, children/WIP, tickets incl. parsed History and
                          review findings, audit health). Read-only. --json is
                          mandatory; omitting it prints usage and exits 2.
  board decisions [--project <name>] [--status <dir>] [--grep <regex>] [--json]
                          Query every ticket's confirmed design decisions, in citable
                          "<ID> decision <N>" form. Read-only. --json is optional here
                          (default output is a short table, not a document dump).
  changelog check [--since <ref>] [--until <ref>] [--changelog <path>] [--section <name>] [--show-excluded]
                          Report tickets that shipped in <since>..<until> (defaults: the
                          last git tag before <until>, and HEAD) but aren't named
                          in the changelog's named section (default "Unreleased").
                          Excluded board: bookkeeping commits summarize to one line unless
                          --show-excluded. Read-only and advisory — always exits 0.

Other scaffolding (unrelated to brine):
  scaffold docs [--project-name <name>] [--force] [--dry-run]
                          Write a minimal AsciiDoc docs skeleton, best-effort snowball init,
                          additive justfile docs-check/docs-build recipes (only if a justfile
                          already exists), and a standalone GitHub Action that attaches the
                          built manual to a release. Entirely separate from pickle install.

Visualize:
  serve [--addr host:port]
                          Serve a read-only web view of the board, each ticket, and a
                          merged History timeline. --addr (-a) sets the listen address
                          (default 127.0.0.1:8745). Writes nothing.

Other:
  version                 Print the pickle version.
  help                    Show this help.
`)
}
