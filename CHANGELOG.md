# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the version is below `1.0.0`, breaking changes may land in a minor release.

## [Unreleased]

### Added

- **`pickle serve` — a local, read-only web view of the board** (T-053). Renders the
  board (`/`), one page per ticket (`/t/T-NNN`) with its Markdown body and *both*
  directions of its dependency/lineage edges, and an activity timeline (`/activity`)
  merging every ticket's `## History` newest-first. That timeline is something
  `BOARD.md` cannot give you: the board shows current state, while the timeline shows
  movement over time. A banner reports what `pickle board audit` would say, plus
  per-child WIP counts.

  Default address is `127.0.0.1:8745`; `--addr host:port` overrides it, and a
  non-loopback address prints a warning — **there is no authentication**.

  **It writes nothing**: no route creates, moves or regenerates anything, not even a
  stale board, so it is safe to leave running while tickets move. Pages re-read the
  ticket files per request (and refresh themselves every five seconds), so they are
  never stale.

  Two new dependencies come with it: `github.com/yuin/goldmark` v1.8.2 (ticket-body
  Markdown; raw HTML stays escaped) and a vendored, embedded copy of htmx 2.0.4
  (0BSD, licence included) for the refresh. There is no CDN reference, no npm and no
  build step — templates and assets are compiled into the binary.

  **Size note:** serving HTTP pulls `net/http` and `html/template` into what was a
  pure file-manipulating CLI. Consequently, the released binary grows from ~2.8 MB to
  ~9.5 MB. goldmark accounts for a small fraction (~0.1 MB); the stdlib server and
  template engine are the bulk.

### Changed

- **Board cells are capped at 120 runes** with a trailing `…` (T-049). One
  over-long value — typically a paragraph-long `merged to …` History line on a
  ticket imported from another flow — could previously render a single table cell
  thousands of characters wide and make a whole status section unreadable. The cap
  applies to every column at the renderer's single sanitisation choke point, counts
  runes (never bytes, so multi-byte text is never cut mid-character), and preserves
  the head of the value so a `merged` cell keeps its commit/MR ref.

  It is a **rendering** bound only: the ticket file keeps the full text and remains
  the single source of truth. **Migration:** if an existing board holds a cell longer
  than 120 runes, `pickle board audit` will report `BOARD.md is stale or hand-edited`
  until you run `pickle board sync` once — expected, since the board is a generated
  artifact whose only invariant is that it matches a fresh render.

## [0.1.0] - 2026-07-27

Initial feature set. A single Go binary that installs and operates a
ticket-based, board-driven feature flow in any project, developed by
self-hosting that very flow (see `tickets/`).

### Added

- **Setup commands.**
  - `pickle install` scaffolds `tickets/` (status directories, `BOARD.md`,
    `NOTES.md`), installs the embedded ticket-flow skill into
    `.agents/skills/ticket-flow/`, wires it up for detected coding agents
    (Claude Code, opencode, Pi) via `--agent`-selectable symlinks and configs,
    injects marker blocks into `AGENTS.md`/`CLAUDE.md`, writes `pickle.toml`,
    and registers the first child-project.
  - `pickle upgrade` refreshes the installed skill payload and marker block to
    the binary's version — never touching tickets or hand-written content.
  - `pickle uninstall [--dry-run]` removes the skill, symlinks and markers,
    leaving `tickets/` and `pickle.toml` intact.
  - `pickle doctor` verifies install integrity: skill payload, agent symlinks,
    marker blocks, and child-project paths.
  - `pickle project add|list|remove` manages the registry of connected
    child-projects (removal is refused while a child has live tickets).
- **Flow commands.**
  - `pickle ticket new` allocates the next `T-NNN` id, scaffolds the ticket
    from the template, and regenerates the board. `--spawned-by` records
    review-finding lineage without ever gating pickup.
  - `pickle ticket move` transitions a ticket atomically: file move, dated
    `## History` line, and board regeneration — enforcing the status state
    machine, per-child WIP limits, and the dependency gate.
  - `pickle board audit` checks the ticket invariants and board freshness,
    exiting non-zero on any error.
  - `pickle board sync` regenerates `BOARD.md` wholesale from ticket
    frontmatter and file locations — the board is a generated artifact, never
    hand-edited.
- **Embedded ticket-flow skill** (`skill/`, shipped inside the binary): the
  flow rules, ticket template, and review protocol that teach a coding agent
  to author, refine, implement and review tickets. Review findings are
  classified by severity and dispositioned (note-and-close by default) instead
  of spawning tickets unboundedly.
- **docs-readability reviewer**: an optional, suggestions-only Gemini subagent
  for AsciiDoc/Markdown prose during ticket review, scaffolded for both
  opencode and Pi from a single shared prompt.
- **Distribution**: goreleaser config and a tag-driven GitHub release workflow
  producing `darwin`/`linux` × `amd64`/`arm64` archives with checksums, plus a
  Homebrew formula published to `codcod/homebrew-tap`
  (`brew install codcod/tap/pickle`). `go install github.com/codcod/pickle`
  works too, with the version read from build info.
- **Documentation**: an AsciiDoc user manual (`docs/user-manual.adoc`) in
  three parts — Start Here, Concepts, CLI Reference — validated with
  `just docs-check` and rendered to PDF/EPUB with `just docs-build` (both via
  [snowball](https://github.com/codcod/snowball)).

[Unreleased]: https://github.com/codcod/pickle/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/codcod/pickle/releases/tag/v0.1.0
