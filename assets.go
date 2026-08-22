package main

import "embed"

// payloadFS is the embedded install payload:
//
//   - skill/  — the canonical brine skill (skill/SKILL.md plus
//     skill/resources/: the rules, the ticket template, the review protocol,
//     and the shared docs-readability reviewer prompt) that `pickle install`
//     writes into a project's .agents/skills/brine/. The tree mirrors
//     the installed skill layout so SKILL.md's "resources/..." references
//     resolve.
//   - agents/ — the per-agent scaffolds `pickle install --agent …` lays down:
//     agents/opencode/opencode.jsonc and agents/pi/extensions/*.ts.
//   - scaffold/docs-template/ — the minimal AsciiDoc docs skeleton + GitHub Action
//     `pickle scaffold docs` (T-110) writes into a target repo, entirely unrelated
//     to brine: attributes.adoc, user-manual.adoc, user-manual/introduction.adoc,
//     and workflows/docs-release.yml.
//
// Embedding all three in the binary lets pickle scaffold either surface with no
// network and no runtime dependency. `all:` includes files that begin with `.` or `_`.
//
//go:embed all:skill all:agents all:scaffold
var payloadFS embed.FS

// version is the build version. Override at build time with:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always)"
var version = "dev"
