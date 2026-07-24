// Command pickle is a CLI that installs and operates a ticket-based, board-driven
// feature flow (the "ticket flow") into any project, with support for several
// connected child-projects under one overarching project.
//
// The command surface is implemented across the phased build plan (P1–P5); see
// README.md for the per-command status.
package main

import (
	"os"
	"runtime/debug"

	"github.com/codcod/pickle/internal/cli"
)

func main() {
	os.Exit(cli.Run(payloadFS, resolveVersion(), os.Args[1:]))
}

// resolveVersion picks the most authoritative build version available. A release
// build (goreleaser or `just`) stamps it via -ldflags -X main.version. When that
// did not happen — notably `go install github.com/codcod/pickle@vX.Y.Z`, where no
// ldflags run — fall back to the module version the Go toolchain records in the
// build info (a tag version, or a commit pseudo-version for a plain `go build`
// from a checkout). Only the sentinel "(devel)" / empty is discarded in favour of
// the ldflags default ("dev").
func resolveVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}
