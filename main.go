// Command pickle is a CLI that installs and operates a ticket-based, board-driven
// feature flow (the "ticket flow") into any project, with support for several
// connected child-projects under one overarching project.
//
// This is the initial skeleton: the command surface exists and compiles; the
// behaviour behind each command is filled in over the phased build plan (P1–P5).
package main

import (
	"os"

	"pickle/internal/cli"
)

func main() {
	os.Exit(cli.Run(payloadFS, version, os.Args[1:]))
}
