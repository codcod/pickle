# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the version is below `1.0.0`, breaking changes may land in a minor release.

## [Unreleased]

### Added

- **`pickle install --in-tree` explicitly selects the layout where the board lives inside its
  sole child's own repository**, recording the choice as `layout` (`"umbrella"` or `"in-tree"`)
  in `pickle.toml` instead of inferring it. `pickle doctor` errors when the recorded layout
  contradicts the registered children, and `pickle upgrade` back-fills `layout` for existing
  projects by the same inference `install` used to apply, so no migration command is needed.
  `pickle serve` states the resolved layout on its startup line and shows a persistent banner
  when an in-tree board is read from a branch that looks like a feature branch (or a detached
  `HEAD`) — the one situation in which the dashboard can show a ticket status that is silently
  out of date, since bookkeeping only ever lands on the base branch (T-108).
- **`pickle board decisions` queries every ticket's confirmed design decisions, already in
  citable `<ID> decision <N>` form** — filterable by registered child-project (`--project`),
  status directory (`--status`) and a topic regex over each decision's full text, statement and
  rationale alike (`--grep`), with `--json` for a machine-readable form. The same answer
  previously needed a hand-written `awk` that re-solved two parsing traps every time (frontmatter
  scoping, subsection bounding) and got the child filter wrong on any workspace whose ticket
  prefix was not `T-`. A decision with no leading bold statement is reported as unstructured,
  carrying its raw first line rather than an inferred one; an unregistered child, an unknown
  status directory, or an uncompilable `--grep` pattern are each an error, while a registered
  child or filter combination with simply nothing to report is exit `0` with an empty result
  (T-105).

### Changed

- **Breaking: `pickle install`'s `--path` no longer defaults to `"."`.** Omitting both
  `--in-tree` and `--path` now installs the umbrella layout with *no* child registered;
  `pickle project add` registers the first one afterward. The previous single-repo default is
  reached explicitly with `--in-tree` (T-108).
- **The shipped skill now states the confirmed-decision shape and its citation form**, previously
  learned only by imitation: a confirmed design decision is a numbered item whose leading bold
  run is the decision statement, numbered in one unbroken list that is never renumbered, and cited
  from another ticket as `<ID> decision <N>` (T-106).
- **The base-branch bookkeeping rule, in both the rendered `AGENTS.md`/`CLAUDE.md` marker block
  and the shipped skill, now states which repository it binds instead of naming a layout that
  never enforces it.** Previously the rule read as universal law, instructing an `umbrella`
  project (the default) to follow a base-branch discipline that only ever matters for the
  repository holding `tickets/` — the overarching project under `umbrella`, the sole child under
  `in-tree`. `install`/`upgrade`/`project add` render the marker block's "Where commits land"
  bullet accordingly, and the user manual documents what choosing `in-tree` costs beyond the
  stale-board read: one-directional staleness, a CI run per bookkeeping push, and interleaved
  release history (T-109).

## [0.9.0] - 2026-08-16

### Added

- **`pickle board state --json` prints the whole ticket tree as one versioned JSON document** — every
  registered child-project, every ticket's frontmatter, parsed `## History` and dispositioned
  `## Review` findings, per-child WIP counts and their caps, and an audit-health summary. As a
  result, a programmatic consumer (an agent-harness extension, a CI step, a git hook) no longer has
  to scrape the other commands' prose or re-implement the tree walk and grading/WIP/audit rules.
  `--json` is mandatory; a bare `pickle board state` prints usage and exits 2 rather than dumping
  the document. The findings projection is a best-effort read across the corpus's historical table
  shapes, keyed by column name, and carries only the closed-vocabulary columns (`id`, `severity`,
  `class`, `disposition`) — never the free-prose ones. The whole read runs behind the same shared
  tree lock `pickle serve` uses, and the output is a pure function of the tree (no timestamp), so
  two runs against an unchanged tree are byte-identical (T-065).

- **The `pickle serve` board page lays the active statuses out as side-by-side columns, and gains a
  search box.** READY, IN DEVELOPMENT, IN REVIEW and REWORK now render as a row of columns per
  registered child-project — the work actually in flight, visible together — while TO DO, DONE and
  DROPPED keep the stacked, full-width sections below them. The column set is derived from the
  flow definition (`Definition.ActiveStates()`: every non-terminal state except the initial one),
  so no status name is hardcoded in the dashboard and a future flow inherits the layout. A search
  field above the board filters it live by ticket id or title, composing with the existing
  child-project filter; like that filter it lives outside the polled fragment, so both the typed
  query and the selected child survive the five-second refresh. Ordering is untouched: every
  column and section is still sorted by the same code `BOARD.md` uses (T-104).

- **Every write to the ticket tree is now atomic, and concurrent `pickle` commands (including
  `pickle serve`) no longer race each other.** `BOARD.md` is written via a temp-file-and-rename
  so a concurrent reader — notably `serve`, which re-reads the tree on every request and on its
  5-second poll — can never observe a truncated or half-written board. `ticket new`, `ticket
  move` and `board sync` each take an exclusive lock on the tree spanning their full
  load-check-write, and `project add`/`project remove` take it while they re-render the board,
  so two of them running at once always serialise instead of racing; `ticket new` additionally
  allocates the new ticket's id under that same lock, so two concurrent invocations can never
  land on the same id. `pickle serve` takes only a
  shared read lock, so any number of dashboards can stay open, and is confirmed safe to leave
  running beside the CLI. A command that cannot acquire the lock within 10 seconds refuses with a
  message naming the lock file rather than hanging (T-101).

### Fixed

- **The board's at-limit WIP badge is now actually highlighted.** `.count` was declared after
  `.wip-full` at equal specificity, so a `class="count wip-full"` badge rendered muted and the
  at-limit warning never showed; `.count.wip-full` now wins by specificity rather than source
  order (T-104, absorbing T-055).

- **The `pre-push` guard decides the pushed branch from the push's destination ref, not its
  source.** It previously tried the source ref (`LocalRef`) first, so `git push origin
  main:refs/heads/feat/T-NNN-x` (a base branch carrying unpushed bookkeeping, pushed to a feature
  branch) escaped the guard entirely, and `git push origin feat/T-NNN-x:refs/heads/main` (a
  feature branch's bookkeeping pushed to the base) was wrongly refused. `RemoteRef` alone now
  decides it, with no fallback, in every case — the guard's invariant is about what a merge
  request built from the *destination* would carry, and a tag destination stays skipped either
  way. The rejection's `range:` line now names a short commit SHA instead of the destination
  branch when the two sides of the refspec disagree, since that branch need not exist locally.
  Also unifies the degraded-guard stderr line on `pickle: <hook> guard skipped (…)` (it
  previously read `pickle: bookkeeping guard skipped (…)` at the binary's own call sites while
  the installed shims already said `<hook-name>`), and gives `doctor`'s PATH-capability line
  (`hooks: the pickle on PATH can run the installed guards`) back its dropped antecedent (T-100).

## [0.8.0] - 2026-08-14

### Added

- **`pickle hooks` gains a `pre-push` guard** alongside the existing `pre-commit` one. It refuses
  a push of a feature branch whose range against the remote base (`<remote>/<base>...<local>`,
  the same three-dot form a forge diffs) still carries a `tickets/` path — the one gap the
  `pre-commit` hook and the existing `origin/<base>...HEAD` prose check both left open at publish
  time. `internal/hook` generalized from a single hook to a `Name`-keyed set (`Names()`,
  `StatusAll`/`InstallAll`/`UninstallAll`/`RefreshAll`), sharing one `ShimVersion` (2 → 3). The
  base is resolved from remote-tracking refs already on disk, with no network I/O, so a stale ref
  only ever widens the checked range. Fail-open contract, marker-prefix ownership and the
  `--no-verify` bypass are unchanged (T-082).

## [0.7.0] - 2026-08-13

### Added

- **The review findings table gains a `class` column** — one of eight closed values
  (`correctness`, `test-gap`, `docs-gap`, `stale-xref`, `plan-wrong`, `spec-unclear`, `design`,
  `other`) naming what *kind* of defect a finding was, alongside its severity and disposition.
  The table now has one canonical, pasteable skeleton in `review-protocol.md`, replacing two
  drifting prose restatements of its column list. The review's disposition-summary line now has
  a companion `cost: estimated …, actual …` line. A ticket's `created … source:` History line
  now leads with a provenance class (`field-use`/`self-host`/`review`/`audit`/`chat`). The
  previously unspecified `plan amended inline` History line is now defined and required whenever
  `## Implementation Plan` is edited after a ticket leaves `2-ready/`. Prose-only: no new `board
  audit` check, no backfill of existing findings (T-085).

- **`pickle ticket move`** and **`pickle ticket new`** now print a ready-to-paste `git add`
  line naming every path they wrote. For a move, that means both the new and the removed
  ticket path plus `tickets/BOARD.md`. Rules §0 requires bookkeeping commits to use explicit
  pathspecs, and the old path is the one most easily omitted from memory — letting a rename's
  add land without its delete, and corrupting git history in a way the worktree-based
  `board audit` cannot see (T-091).

### Changed

- **BREAKING: the installed skill directory is renamed `ticket-flow` → `brine`** (pre-1.0), so
  the tool (`pickle`), the flow (`brine`) and the on-disk skill id are not three different
  names for two things. The skill now installs at `.agents/skills/brine/`
  (`.claude/skills/brine`), and `SKILL.md`'s `name:` frontmatter — the agent-visible discovery
  name — is now `brine`, so the invocation is `/skill:brine`. **There is no migration**: an
  existing install is fixed by running `pickle upgrade`, which now *sweeps away* the old
  `.agents/skills/ticket-flow/` directory and `.claude/skills/ticket-flow` symlink before
  refreshing the new-name payload (a legacy path that was itself a self-host symlink is
  re-linked at the new name, not deleted-and-recopied); `pickle uninstall` sweeps the same
  legacy paths, so a current binary can still fully remove an old install. `pickle doctor` now
  **errors** — not warns — while either legacy path is still present, naming `pickle upgrade`
  as the fix, so a partially-upgraded project carrying both names is never reported as healthy.
  One thing the sweep does not fix: a user-edited `opencode.jsonc` that still hardcodes the old
  `.agents/skills/ticket-flow/resources/docs-readability.prompt.md` prompt path is user-owned
  after creation and must be edited by hand to the `brine` path — `doctor` performs no
  opencode checks and this change adds none (T-074).

- **`pickle doctor`** no longer warns about `payload_version` in a self-hosting checkout
  (`.agents/skills/brine` is a symlink to the payload source): the comparison is skipped
  and reported as an informational passed line under `--verbose` instead, since the payload
  is the linked source, not an installed copy. `pickle upgrade` keeps stamping
  `payload_version` in that mode — it still refreshes everything else it owns (T-046).

### Fixed

- **The installed skill no longer addresses its reader as though they were pickle's own repo.**
  The review protocol's worked examples pointed into a `tickets/6-done/` for ticket ids that
  exist only here, and justified the findings-table skeleton with a header-variant count from a
  corpus the reader cannot see; the `field-use`/`self-host` provenance classes were defined as
  "another project" versus "this repo's own flow", which is unassignable in any project that does
  not host this flow itself — leaving the two busiest classes to be filled inconsistently. The
  examples are now self-contained and say *why* each maps to its class, the skeleton's warrant
  states the mechanism rather than a number, and the two classes are defined as using the flow to
  ship something else versus working on the flow itself. Prose only: the five provenance tokens
  and the eight `class` values are byte-identical, and the legitimate uses of a ticket id (syntax
  filler, provenance tags) are deliberately kept (T-098).

- **`pickle changelog check`** no longer mistakes an id-shaped non-ticket token (`SHA-256`,
  `UTF-8`, `RFC-7231`, `CVE-2024`, ...) for a ticket id. Every id-recognition site now shares
  one predicate, restricted to the ticket-id prefixes the project actually registers in
  `pickle.toml`. As a result, a `board:` bookkeeping subject mentioning `SHA-256` no longer
  silences the `(+N with no ticket id)` drift alarm, and a child-project commit ending in
  `(RFC-7231)` no longer prints as a fabricated shipped candidate that no changelog entry
  could ever clear (T-097).

## [0.6.0] - 2026-08-12

### Added

- **`pickle changelog check`** reports tickets that shipped since the last release (by
  Conventional Commit subject, excluding `board:` bookkeeping commits) but aren't named in
  `CHANGELOG.md`'s `[Unreleased]` section, pointing at each candidate's ticket file so you
  can add an entry or confirm the ticket already records a decision to have none
  (T-093, T-094, T-095). Read-only and advisory: it always
  exits `0`, even with candidates, and is never wired into `board audit`, CI, or
  `ticket move`. `--since`, `--until`, `--changelog` and `--section` override the defaults
  (the last tag before `--until`, `HEAD`, `CHANGELOG.md`, `Unreleased`); when `--until` has
  no parent commit at all to describe, the `--since` default falls back to `--until` itself
  instead of erroring. A squash-merge's trailing `(#31)`/`(!31)` is tolerated after the
  ticket id; the excluded `board:` commits summarize to one line, naming every id any
  excluded subject mentions, unless `--show-excluded`; a tagged `--until` with the default
  `--section` gets an advisory note pointing at the section it probably means.

### Changed

- **The docs-readability reviewer's default model is now `github-copilot/gpt-5.4`**, in both
  backends `pickle install` scaffolds — the pi extension (`docs-readability.ts`) and the
  opencode subagent (`opencode.jsonc`) — replacing `github-copilot/gemini-2.5-pro` (T-096).
  The old pin was unreachable through GitHub Copilot: every recorded attempt to invoke it
  failed to reach the model (`model_not_supported` / `Model not found`), forcing a conscious
  skip of review protocol Step 4b in
  eleven reviews (T-019, T-022, T-026, T-040, T-041, T-057, T-068, T-089, T-092, T-093,
  T-094). Same provider and same shared prompt, so no new login or plumbing; the
  `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` env-var overrides are unaffected.
  The reviewer's user-facing wording is now vendor-neutral, so the next model swap is a
  one-line default change rather than a repo-wide text hunt.

- **`pickle board audit` now warns on every `6-done/` ticket with no `merged to <base>` History
  line, not only those named in another ticket's `depends-on:`.** The prior check only fired on
  the *dependent* ticket's ref; the common case — a done ticket nobody depends on — was never
  visited. It stays a warning, never an error: merging is the human's and may lag. CI now runs
  `pickle board audit` in the `build-test` job, so the board's error class (duplicate ids, a
  stale generated render, …) is checked on every push/PR instead of only when a human remembers
  to run it locally. (T-092)

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

[Unreleased]: https://github.com/codcod/pickle/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/codcod/pickle/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/codcod/pickle/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/codcod/pickle/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/codcod/pickle/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/codcod/pickle/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/codcod/pickle/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/codcod/pickle/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/codcod/pickle/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/codcod/pickle/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/codcod/pickle/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/codcod/pickle/releases/tag/v0.1.0
