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
//   - scaffold/release-template/ — the two skeleton files `pickle scaffold release`
//     (T-113) writes into a target repo, entirely unrelated to brine:
//     CHANGELOG.md (Keep a Changelog shape) and RELEASING.md (section headings
//     only — no prescribed tooling).
//   - prompts/ — prompts `pickle` prints for an agent and never installs:
//     prompts/retro.md, the method `pickle retro` (T-141) prints after its
//     facts block.
//
// Embedding all four in the binary lets `pickle install`,
// `pickle scaffold release` and `pickle retro` run with no network and no
// runtime dependency.
// `all:` includes files that begin with `.` or `_`.
//
//go:embed all:skill all:agents all:scaffold all:prompts
var payloadFS embed.FS

// version is the build version. Override at build time with:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always)"
var version = "dev"
