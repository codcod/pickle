package main

import "embed"

// payloadFS is the embedded skill payload — the canonical ticket-flow resources
// (SKILL.md, the rules, the ticket template, the review protocol, and the board
// skeleton) that `pickle install` writes into a project. Embedding it in the
// binary is what lets pickle install the flow with no network and no runtime
// dependency. `all:` includes files that begin with `.` or `_`.
//
//go:embed all:resources
var payloadFS embed.FS

// version is the build version. Override at build time with:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always)"
var version = "dev"
