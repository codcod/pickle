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
	case "project":
		return runProject(args[1:])
	case "ticket":
		return runTicket(args[1:])
	case "board":
		return runBoard(args[1:])
	case "version", "--version", "-v":
		return runVersion(args[1:])
	case "help", "--help", "-h":
		usage(os.Stdout)
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "pickle: unknown command %q\n\n", args[0])
		usage(os.Stderr)
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
  upgrade                 Refresh the installed skill payload + marker block to this
                          binary's version (never touches tickets).
  doctor                  Verify install integrity (skill, symlinks, markers, child paths).
  uninstall [--dry-run]   Remove skill/symlinks/markers; leave tickets/ and pickle.toml
                          intact. --dry-run (-n) lists what would go, changing nothing.

Flow commands:
  ticket new "<title>" --project <name> [--impact .. --complexity .. --cost ..]
                       [--spawned-by "T-NNN[,T-MMM]"]
                          Allocate the next T-NNN, scaffold the ticket, add the board row.
                          --spawned-by records lineage (never gates pickup).
  ticket move T-NNN <status> --reason "<why>"
                          Move a ticket (file + History + board) atomically.
  board audit             Check the board/ticket invariants (exit non-zero on any error).
  board sync              Repair board rows from ticket frontmatter + locations.

Other:
  version                 Print the pickle version.
  help                    Show this help.
`)
}
