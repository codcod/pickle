---
id: T-011
title: distribution (goreleaser + Homebrew tap + releases + docs)
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M-L
---

# T-011 — distribution (goreleaser + Homebrew tap + releases + docs)

## Description

Make `pickle` installable. Switch the module path from the bare `pickle` to a real VCS path
(`github.com/…`/`gitlab.com/…`); wire goreleaser for cross-compiled static binaries and
GitHub/GitLab releases; publish a **Homebrew tap in a separate tap repo**; stamp the build
version via `-ldflags -X main.version=…` and pin a payload version; write user-facing docs
(install, `install`/`project`/`ticket`/`board` usage). Enables `brew install` and
`go install`. Wants the command set (P1–P3) essentially complete first. Phase P5.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P5)
