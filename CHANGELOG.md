# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the version is below `1.0.0`, breaking changes may land in a minor release.

## [Unreleased]

### Added

- **`pickle changelog check`** reports tickets that shipped since the last release (by
  Conventional Commit subject, excluding `board:` bookkeeping commits) but aren't named in
  `CHANGELOG.md`'s `[Unreleased]` section, pointing at each candidate's ticket file so you
  can add an entry or confirm the ticket already records a decision to have none (T-093).
  Read-only and advisory: it always
  exits `0`, even with candidates, and is never wired into `board audit`, CI, or
  `ticket move`. `--since`, `--changelog` and `--section` override the defaults (the last
  git tag, `CHANGELOG.md`, `Unreleased`).

## [0.5.0] - 2026-08-12

### Added

- **Each flow status now declares, as data, which ticket sections it requires — and `pickle
  ticket move`/`board audit` enforce it** (T-081). Previously the READY gate's seven
  Implementation Plan steps (rules §4) were prose an agent judged by eye; `internal/flow`'s
  `State.Requires` is a per-status gate table (`flow.Requirement`: a `##` section, an optional
  `###` sub-heading, a severity), so `2-ready/` … `5-rework/` now mechanically require the plan
  section plus its seven READY-gate headings (feature branch, prerequisite gate, confirmed
  decisions, tasks, acceptance test, docs update, finish), and `5-rework/` additionally requires
  `## Review`. A **blocking** row unmet refuses `ticket move` outright, before anything is
  written, and is an *error* from `board audit` for a ticket already sitting past the gate; an
  **advisory** row unmet only ever warns. The check is structural (a named heading present with
  a non-empty body, HTML-comment placeholders stripped) — it proves a step is present, never
  that its content is *good*. Terminal statuses (`6-done/`, `7-dropped/`) declare nothing, so an
  archived ticket is never flagged, however incomplete. Folds in T-080's review finding N6:
  `flow.Spec` gains an explicit `Pickup` state (the state the dependency+merge gate guards entry
  into) instead of `internal/move`/`internal/audit` inferring it from
  `config.WIPKeyInDevelopment`, which silently skipped the gate on a lookup miss.

- **`pickle serve` now renders a URL in a merge or History line as a clickable link** (T-089).
  A bare URL in a ticket's `## Description`/body already linked itself on the ticket page
  (goldmark's GFM `Linkify` extension), but the board's `merged` cell, the ticket page's
  `merged` summary line, and the activity timeline's per-entry text did not — those three
  render as plain, escaped strings rather than through the markdown renderer. A new `linkify`
  template helper closes that gap in all three, HTML-escaping the text and wrapping any bare
  `http(s)` URL in an anchor. The board's `merged` cell is still capped at 120 runes, so a long
  commit URL can be clipped there; the ticket page and the timeline show it in full.

### Changed

- **T-083's `## Outcome`-presence check is now one row of the new per-status gate table
  (T-081), not a hand-rolled `internal/audit` branch** — wording and severity (a warning, never
  a gate) are unchanged, but it now lives as `internal/flow` data alongside every other
  requirement instead of a `!st.Terminal` special case. `ticket.OutcomeMissing` is removed;
  `ticket.SectionMissing(text, heading)` is its generalisation, along with new
  `ticket.SubsectionMissing` and `ticket.GateViolations` for evaluating a status's whole table.
  **Upgrade note:** a ticket already sitting in `2-ready/` … `5-rework/` whose Implementation
  Plan predates the seven-heading gate now reports a **blocking** `board audit` error until the
  missing heading is written (even "none" satisfies a Prerequisite gate/Docs update row); from
  `2-ready/` specifically, moving the ticket back to `1-to-do/` also clears it, but no single
  move does from `3-in-development/`, `4-in-review/` or `5-rework/`, so writing the heading in
  place is the only way out there. Like any other `board audit` error, it also makes
  **`pickle upgrade` and `pickle install` themselves fail** (both run this same audit self-check
  and exit non-zero on it) — what is specific to this one is that clearing it means writing prose
  into each affected ticket, not re-running a command. It also makes every subsequent `ticket move` — even of an unaffected
  ticket — report a post-move audit error too, until the offending ticket is fixed. Note that
  `board sync` will usually *not* surface this on its own: it only re-runs its own audit
  self-check when it actually rewrites `BOARD.md`, and nothing in a ticket's plan body feeds any
  board column — run `board audit` directly, before upgrading, to find every affected ticket.
  There is no migration flag — see the user manual's `board audit` NOTE for detail.

- **The merge History line can now carry a commit reference** (T-089).
  `merged to <base> (<MR ref>)` stays free text — no new parser, no new ticket field — but the
  flow's rules and ticket template (the `ticket-flow` skill payload) and the user manual now
  recommend appending a commit reference alongside the MR ref: a short SHA, and a full commit
  link where the remote resolves to a known hosting URL, e.g.
  `merged to main (MR !12, a1b2c3d, https://github.com/org/repo/commit/a1b2c3d…)`. Keep the
  short SHA even when adding the link — it is what still reads in the board's capped `merged`
  cell.

- **Brine's lifecycle — its seven statuses, their transitions, and the terminal/WIP flags and
  gate targets derived from them — now comes from one declarative definition** (T-080),
  `internal/flow`, instead of being scattered as Go literals across board rendering, move
  validation, the audit checks, install/sync and the CLI. Output and behaviour are unchanged:
  the board renders byte-identical, `pickle board audit`'s checks fire on the same conditions at
  the same severities, and every move/gate refusal message reads the same. A project-authored
  flow is still unsupported — `flow = "brine"` remains the only legal value in `pickle.toml`.

## [0.4.0] - 2026-08-09

### Added

- **Every ticket now carries a `## Outcome` section, and `pickle board audit` warns when it's
  missing** (T-083). The section sits above `## Description` and states, in 1–3
  user-observable sentences, what changes when the ticket ships — descriptive, not
  evaluative; it makes no worth claim and gates nothing. `board audit` reports a *warning*
  (never an error, never a block on `ticket move`) when a non-terminal ticket's (`1-to-do/`
  through `5-rework/`) Outcome is absent, empty, or still the `TEMPLATE.md` placeholder;
  `6-done/` and `7-dropped/` are permanent archives and are never checked.

### Changed

- **Bookkeeping commits (ticket/board state changes) use their own `board: T-NNN <verb
  phrase>` form instead of Conventional Commits** (T-084), scoped to commits whose staged
  paths are limited to ticket files and `BOARD.md`. A root-path child (`path = "."`) now
  defaults to preserving commits on merge (rebase or keep-history) instead of squashing, with
  a Finish-step tidy-up (interactive rebase into atomic, correctly typed/scoped commits)
  before the sequence is presented for approval.

### Fixed

- **The release workflow's Homebrew install could fail on a stale runner image** (T-086).
  `ubuntu-latest`'s preinstalled Homebrew is a point-in-time snapshot; when homebrew-core ships a
  bottle using a newer post-install-step DSL than that snapshot understands (observed: `node`'s
  `"remove"` step), the stale `brew` aborts with `unknown install step: remove` — a
  version-mismatch failure the existing 3-attempt retry loop (written for transient broken-pipe
  downloads) can never clear, since all three attempts fail identically. The "Build user manual"
  step now runs `brew update --quiet` once before that retry loop.
- **The release workflow's user-manual build (PDF/EPUB) had been failing invisibly since
  `v0.2.2`** (T-087): `v0.2.2` and `v0.3.0` both shipped with no manual attached, each for a
  different toolchain reason. The build step's deliberate soft-fail (a broken manual must not
  block publishing the binaries) exited silently on a miss, so nobody noticed. A missing manual
  is now annotated on the release run, and the toolchain (`.github/scripts/build-manual.sh`) can
  be exercised from a new `manual-smoke` workflow without cutting a release.

## [0.3.0] - 2026-08-07

### Added

- **`pickle board audit` now validates ticket frontmatter, status directories and
  History-line shape** (T-040). Five checks land in `internal/audit`, the one component that
  sees every ticket however it was authored: a duplicate frontmatter key (e.g. two `impact:`
  lines) is now an error instead of silently last-wins; a ticket listing itself in `depends-on`
  is now an error, mirroring the existing `spawned-by`/`family` self-reference guards; the
  audit's required/optional key set is checked against `TEMPLATE.md` by a new test so the two
  can no longer drift apart silently; all seven status directories are validated directly — a
  missing one is an error, an empty one with no tracked `.gitkeep` is a warning; and a
  status-transition or merge `## History` line over 400 runes now warns (free-form dated notes
  and `created` lines are exempt, and the threshold was picked by measuring this repo's own 303
  History entries).
- **`pickle doctor` and `pickle project add` now warn when a registered child is
  stageable** (T-051). Registering a second child-project at a nested path
  (`pickle project add <name> <path>`, `path` other than `.`) leaves it as an
  ordinary, untracked directory of the overarching repo until a `.gitignore`
  entry (or an index entry, for a deliberate submodule/gitlink) says
  otherwise — a staging accident waiting for whoever forgets that hand edit.
  `project add` now names the missing entry right after registering, and
  `pickle doctor` repeats the same check on every run, so a child that drifts
  into this state later (an entry deleted, or a child registered before the
  check existed) is still caught, as a warning (never an error — a deliberate
  submodule/gitlink is reported as fine). pickle asks git directly
  (`git check-ignore`, `git ls-files`) rather than parsing `.gitignore`, and,
  as with every other file it does not own, never writes to it itself.
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
- **A named flow: `flow` key and `pickle flow show|list`** (T-073). `pickle.toml` gains an
  optional `flow` key (shape-validated, defaults to `"brine"`) naming the ticket-based flow
  pickle installs — cheap to add now, and the seam a second flow would later select. `pickle
  flow show` prints the configured flow (or the default when the key is absent), and `pickle
  flow list` currently prints that one entry. Elsewhere the rename is prose-only: the flow is
  now called **brine** throughout `skill/SKILL.md`, the review protocol, the
  `AGENTS.md`/`CLAUDE.md` marker block, this repo's own self-hosted files, and the docs
  (a new `:flow:` attribute). **Nothing on disk moves** — the installed skill directory, the
  `.claude/skills/` symlink and `SKILL.md`'s own `name:` frontmatter still say `ticket-flow`,
  so no existing install needs migrating.

### Changed

- **`pickle upgrade` no longer refuses legal `pickle.toml` files it used to misdiagnose**
  (T-026). The `payload_version` line scanner tracked no notion of being inside a multi-line
  string or array, so a continuation line that merely looked like a table header (or began with
  `[`) was read as ending the top-level scope — silently and permanently refusing to stamp four
  legal shapes: a multi-line string containing a `[`-leading line, a `nan` value anywhere in the
  file, the quoted spelling `"payload_version"`, and a multi-line array on the insert path. The
  scanner now carries state (multi-line delimiter, bracket depth) across lines, and the
  parse-back safety gate treats two `NaN`s as equal instead of reporting them as changed. Every
  refusal that remains by design (the key's own value being multi-line or an array — pickle
  never writes either) now names the line and the cause instead of a generic message, and
  `pickle doctor` probes before recommending `pickle upgrade`, so it no longer sends the user to
  a command that is going to fail.
- **The `AGENTS.md`/`CLAUDE.md` marker block stays in step with `pickle.toml`, and drift from it
  is now detected** (T-041). `pickle project add`/`pickle project remove` now re-inject the
  marker block through one shared refresh path immediately after they mutate the config, so it
  no longer keeps describing yesterday's set of children — previously invisible, and capable of
  making an agent refuse legitimate work on a project it was never told about, or build with a
  stale WIP limit or command. `pickle doctor` gained the other half: it now warns (never errors)
  when the installed block differs from what today's `pickle.toml` would render, including a
  block that predates a payload change, and the branch-name bullet it renders now honours each
  child's configured `ticket_prefix`.
- **The pre-commit shim bumps to `# pickle:hook v2`** (T-068). Its guard-absent branch now prints
  one stderr notice (`pickle: bookkeeping guard skipped (pickle not found on PATH)`) instead of
  degrading silently — the same reasoning that already made an unexpected exit code speak — and a
  cosmetic doubled `#` in the v1 marker line is fixed. `pickle upgrade` refreshes an owned v1 shim
  in place; the fail-open contract is unchanged (exit `1` still means a violation, and only that).
- **An unrecognised top-level command answers in one line** (T-068), pointing at `pickle help`,
  instead of printing the full usage text ahead of it. `pickle` with no arguments still prints the
  whole usage summary.
- **`pickle board audit` (and hence `pickle upgrade`'s post-check) no longer errors on a board
  that is merely out of date in its generated layout** (T-052). The documented onboarding
  sequence — `pickle project add <name> <path>` then `pickle upgrade` — used to end in
  `ERROR: BOARD.md is stale or hand-edited` and a non-zero exit for a workspace where nothing
  was wrong: registering a child changes the board's *generated shape* (a new per-child section
  under every status heading, a new WIP line), so a board that was in sync a second earlier no
  longer matches a fresh render. The check is now two-tiered: if every ticket row still matches
  (same status section, same child sub-group, same cell text) and only the generated scaffolding
  around them is stale, that is a *warning* — `BOARD.md is out of date in its generated layout
  only (every ticket row matches) — run pickle board sync` — and does not fail `board audit` or
  `upgrade`. If any row itself disagrees with the tickets, it is still an *error* —
  `BOARD.md does not match the ticket files (rows differ) — run pickle board sync`. Nothing can
  tell a hand-edit apart from a routine registry or renderer change, so the split is on harm, not
  cause. `pickle project add`/`pickle project remove` now also regenerate `BOARD.md` themselves
  (printing `+ tickets/BOARD.md`) right after they refresh the `AGENTS.md`/`CLAUDE.md` marker
  block, since the registered-child list feeds both — the reported sequence is now silent and
  clean end to end. `pickle upgrade` still never reads or writes anything under `tickets/`, and
  `pickle board sync --dry-run` is unaffected: it still reports layout-only drift as a pending
  change and exits non-zero.
- **`pickle.toml`'s writers are now TOML-correct and crash-durable** (T-069). `Config.Render`
  quoted every string field with Go's `%q`, which can emit escapes TOML has no equivalent for
  (`\a`, `\v`) or a byte-for-byte encoding of invalid UTF-8 that TOML reads back as a
  *different*, valid string. This was reachable end to end through `pickle project add`, whose
  values pass straight from argv: one input could brick `pickle.toml` for every later command,
  another could silently rename a registered child at exit `0`. Fixed with a `tomlQuote` helper
  (TOML's own short escapes plus `\uXXXX`) at every call site, plus a validation gate rejecting
  invalid UTF-8 before it ever reaches the file, with rollback. `Save` now fsyncs before its
  rename (crash-durable, not merely atomic), and two narrower `pickle upgrade` line-editor bugs
  — a multi-line string's escaped quote misread as ending the string, and a stray trailing `\r`
  left when nothing followed an insertion point — are also fixed.

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

[Unreleased]: https://github.com/codcod/pickle/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/codcod/pickle/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/codcod/pickle/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/codcod/pickle/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/codcod/pickle/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/codcod/pickle/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/codcod/pickle/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/codcod/pickle/releases/tag/v0.1.0
