# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the version is below `1.0.0`, breaking changes may land in a minor release.

## [Unreleased]

### Added

- **`pickle hooks` — a pre-commit guard for ticket bookkeeping** (T-057). New verb
  `pickle hooks install [--force] | uninstall [--dry-run] | status | run pre-commit`, plus
  `pickle install --hooks`. The installed hook refuses a commit that stages `tickets/` paths
  while a feature branch is checked out: the flow puts code on `feat/T-NNN-<slug>` and ticket +
  board bookkeeping on the base branch, and a squash-merge of a branch carrying bookkeeping
  folds or drops it, leaving `BOARD.md` disagreeing with the tickets it indexes. The hook is a
  shim calling back into the binary, so the rule reads each child's live `branch_prefix` instead
  of baking it into a script, and ownership is a `# pickle:hook v2` marker in the file — a
  `pre-commit` hook pickle did not write is never modified without `--force`. The hooks directory
  comes from git, so an existing `core.hooksPath` (Husky, Lefthook) is honoured. It **fails
  open**: no `pickle` on `PATH`, no `pickle.toml`, no git, or an older `pickle` on `PATH` all
  skip the check rather than block the commit (`hooks run` exits `1` for a violation and only
  for a violation). `pickle upgrade` refreshes a stale shim but never arms an absent one,
  `pickle uninstall` removes a pickle-owned hook, and `pickle doctor` reports the state — absent
  is not a finding, since hooks are per-clone and never cloned.
- **A PATH-capability probe for the pre-commit guard (T-068).** A `pre-commit` hook that is
  present and current on disk was still measured as silently inert: the shim resolves `pickle`
  from `PATH` at commit time, and an older release there (Homebrew lag, a second clone,
  `go install` next to a packaged copy) exits `2` on the once-unknown `hooks` verb and degrades
  correctly — but nothing said so. `pickle hooks install`, `pickle hooks status` and
  `pickle doctor` now probe the PATH `pickle` (the shim's own call, run in an empty directory) and
  warn, naming the incapable binary's path and best-effort version, when it cannot run the guard;
  a same-file or otherwise-capable PATH `pickle` stays silent.
- **The rule the guard enforces is now written into the payload** (T-057). The rules (§0), the
  review protocol, `SKILL.md` and the `AGENTS.md` marker block all state where commits land; the
  review protocol also fixes the mirror-image hazard the split creates — a reviewer on a feature
  branch must read the ticket and board from the base branch (`git show <base>:tickets/…`),
  because a branch cut before the bookkeeping landed shows a stale ticket.

### Changed

- **The pre-commit shim bumps to `# pickle:hook v2`** (T-068). Its guard-absent branch now prints
  one stderr notice (`pickle: bookkeeping guard skipped (pickle not found on PATH)`) instead of
  degrading silently — the same reasoning that already made an unexpected exit code speak — and a
  cosmetic doubled `#` in the v1 marker line is fixed. `pickle upgrade` refreshes an owned v1 shim
  in place; the fail-open contract is unchanged (exit `1` still means a violation, and only that).
- **An unrecognised top-level command answers in one line** (T-068), pointing at `pickle help`,
  instead of printing the full usage text ahead of it. `pickle` with no arguments still prints the
  whole usage summary.

## [0.2.2] - 2026-07-29

### Added

- **Board child-project filter buttons in `pickle serve`** (T-061). The dashboard's
  board page (`/`) gains a flat filter bar above the first status section: an **All**
  default plus one pill button per registered child-project, each with a count chip
  (per-child total; `All` shows the board total). Selecting a child collapses the board
  to just that child's blocks across every status section — the status headings and
  counts stay put. The bar lives outside the polled `#board`, so the active selection
  **survives the five-second htmx refresh** (a small script re-applies it on
  `htmx:afterSwap`); filtering is pure client-side show/hide, so the fragment routes and
  `buildBoard` output are byte-identical to a reload. Active-state styling derives from
  the existing `--accent` token via `color-mix`, so it tracks the light and dark palettes
  (T-054) rather than hardcoding colours. The chip counts refresh on a full reload, not on
  the five-second poll.

## [0.2.1] - 2026-07-29

### Added

- **PDF and EPUB user manual attached to every GitHub release.** The release
  workflow now builds the AsciiDoc user manual with
  [snowball](https://github.com/codcod/snowball) into `dist/docs/` before
  goreleaser runs, and a new `release.extra_files` entry in `.goreleaser.yaml`
  uploads `pickle-user-manual.pdf` and `pickle-user-manual.epub` as release
  assets. The docs build is soft-failing (`continue-on-error`): a broken manual
  does not block publishing the binaries — goreleaser simply finds no files to
  attach.

## [0.2.0] - 2026-07-28

### Added

- **Per-child `ticket_prefix` with per-child id counters** (T-058). A `[[project]]`
  may set `ticket_prefix` (shape `^[A-Z][A-Z0-9]{0,7}$`) so its tickets are
  numbered in their own namespace — `RICK-001`, `SB-001` — instead of one global
  `T-NNN` sequence. Absent ⇒ defaults to `T`, so existing installs are unchanged;
  the default `T` is exempt from the cross-child uniqueness check, and a non-`T`
  prefix must be unique across children. `pickle project add --ticket-prefix` sets
  it and `pickle project list` shows a `PREFIX` column.

  Ids are now unique only **within** a prefix — always qualify. A new audit
  invariant enforces that a ticket's id prefix matches its `project:`'s configured
  prefix, catching a mis-filed ticket. Id shape validation widened from `^T-\d+$`
  to `^[A-Z][A-Z0-9]*-\d+$`, and the board renders mixed prefixes in one table
  (sort is prefix-then-number).

- **`family:` — group tickets under an umbrella ticket for board ordering**
  (T-059). A ticket may set `family: T-NNN` pointing at an umbrella ticket (an
  ordinary same-child ticket — no new entity, board, or lifecycle). `TO DO`/`READY`
  rows then order by umbrella rank with the umbrella first and its members adjacent,
  breaking the wide `impact`-only ties that appear at scale. `pickle ticket new
  --family T-NNN` authors it (shape-checked at creation; existence deferred to the
  audit, same split as `--spawned-by`), and the board gains a `family` column. The
  audit validates a set `family` — must exist, be same-child, not self, and not nest
  — and gates nothing (lineage only). A ticket with no family scaffolds
  byte-identically to before, so the existing backlog needs no migration.

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

- **The `pickle serve` dashboard follows the system light/dark preference**
  (T-054). It honours `prefers-color-scheme` with a light palette authored for a
  light ground — not an inversion — and every text colour is verified at WCAG AA
  (≥ 4.5:1) in *both* palettes. **The dark palette is unchanged**, so a dark-mode
  reader sees exactly what the entry above describes.

  Note which way the default falls: `no-preference` was removed from
  `prefers-color-scheme` in Media Queries L5, so a browser with no expressed
  preference reports *light* and gets the light palette. Dark is the fallback only
  where the query is unsupported.

  There is deliberately **no toggle**: a read-only, stateless server has nowhere
  to persist a choice, so the OS preference is the only input.

  The page also declares `color-scheme`, so the browser paints *its own* surfaces
  — scrollbars on wide code blocks and tables, the audit banner's disclosure
  marker — in the matching scheme rather than always in light chrome.

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

[Unreleased]: https://github.com/codcod/pickle/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/codcod/pickle/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/codcod/pickle/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/codcod/pickle/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/codcod/pickle/releases/tag/v0.1.0
