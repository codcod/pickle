# pickle

`pickle` installs and operates a **ticket-based, board-driven feature flow** in any project —
the same flow prototyped by hand in an earlier workspace, now a single-binary CLI.

Run `pickle install` in a project and a coding agent (Claude Code, opencode, Pi) understands
requests like *"create a feature to review the Jira board"*: it authors a correctly-formatted
ticket in `tickets/1-to-do/T-NNN-<slug>.md` and updates `BOARD.md`, because `install` laid down
the flow skill, the board, and the agent-instruction markers.

## Multi-project

The overarching project (where `pickle` is installed and the board lives) can hold several
**connected child-projects** — cooperating build targets such as the frontend and backend of
one application, each its own git repo. Locked model:

- **one shared board**, sub-grouped by child-project;
- **one global ticket-id namespace** (`T-NNN`, `max()+1` across all children);
- each ticket names its target child in a **`project:` frontmatter field**;
- each child is **its own git repo**; `feat/T-NNN-<slug>` branches are cut inside it;
- **WIP limits are per child**;
- children are registered explicitly with `pickle project add <name> <path>` (recorded in
  `pickle.toml`).

A single-child project is just the degenerate case.

## Design principle — split judgment from mechanics

- **Judgment** (what a ticket says, grading, refining, reviewing) lives in the embedded
  ticket-flow **skill** + the project's `AGENTS.md`; `pickle` ships it as content and never
  automates it.
- **Mechanics** (scaffold dirs, allocate the next `T-NNN`, move a ticket file + sync the board +
  append History atomically, audit the invariants) are deterministic **CLI commands**.

The embedded payload lives in [`skill/`](skill/) — laid out exactly as the installed skill
(`skill/SKILL.md` + `skill/resources/{tickets-README,TEMPLATE,review-protocol,BOARD}.md`) — and
is compiled into the binary via `//go:embed`. `pickle` **self-hosts** this flow: its own board
lives in [`tickets/`](tickets/), the skill is discoverable via `.agents/skills/ticket-flow`
(a symlink to `skill/`), and [`pickle.toml`](pickle.toml) registers the sole child-project
(`pickle`, at the repo root).

## Commands

```
pickle install                          scaffold + install skill + markers + pickle.toml + first child   [done: T-004]
pickle project add|list|remove          manage connected child-projects                                  [done: T-001]
pickle upgrade                          refresh installed skill payload + markers                        [P2]
pickle doctor                           verify install integrity                                         [P2]
pickle uninstall                        remove skill/symlinks/markers (keep tickets/)                    [P2]
pickle ticket new "<title>" --project   allocate T-NNN, scaffold ticket, add board row                   [done: T-003]
pickle ticket move T-NNN <status>       move file + History + board atomically                           [done: T-007]
pickle board audit                      check every board/ticket invariant                               [done: T-002]
pickle board sync                       repair board rows from ticket state                              [P3]
pickle version | help
```

This repository is **early**, but the core loop is live: `install` (T-004), `project
add|list|remove` (T-001), `ticket new` (T-003), and `board audit` (T-002) are implemented.
The remaining commands (`upgrade`/`doctor`/`uninstall`, `board sync`) are stubs that report
their target build phase.

## Configuration — `pickle.toml`

`pickle.toml` lives at the overarching-project root and is **tool-managed** (hand-edits are
preserved on load but normalised to a canonical layout on the next `project add|remove`):

- **overarching:** `payload_version`, optional `review_addendum`, and a `[commit]` table
  (`overarching_auto`, `child_publish_gated`);
- **`[[project]]` array** (one per connected child): `name`, `path` (relative to the root),
  `build`/`test`/`lint`/optional `docs`, `branch_prefix` (default `feat/`),
  `wip_in_development` / `wip_in_review` (default 1), optional per-child `review_addendum`.

TOML is decoded with [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) — a
**build-time** dependency compiled into the static binary; nothing is fetched at runtime.

## `pickle board audit`

A pure, dependency-free check of the flow invariants over `tickets/` + `pickle.toml`. It exits
non-zero (and prints `ERROR:`/`WARNING:` lines plus a summary) when any invariant fails:

- each ticket file sits in a known status dir and is named `T-NNN-<slug>.md`;
- frontmatter is complete (`id`, `title`, `project`, `depends-on`, `impact`, `complexity`,
  `cost`), grades are legal, the id matches the filename, and ids are unique globally;
- `project:` names a **registered child**, and every `depends-on:` target exists;
- every ticket appears exactly once on `BOARD.md`, under the section **and** child sub-group
  matching its directory (terminal tickets may age off the board); every board row has a file;
- **per-child** WIP limits (`wip_in_development`/`wip_in_review`) hold;
- each ticket's last History transition matches its directory;
- nothing is in `3-in-development/` with a dependency not yet in `6-done/` (warning if a done
  dependency has no `MERGED` History line).

Missing empty status directories are treated as empty, not errors (git does not track empty
dirs).

## `pickle install`

```
pickle install [--project <name>] [--path <path>] [--build/--test/--lint/--docs <cmd>]
               [--no-claude] [--claude-symlink]
```

Run once in a project to lay down the flow. It:

- copies the embedded skill payload into `.agents/skills/ticket-flow/` (real files);
- symlinks the Claude Code view `.claude/skills/ticket-flow -> ../../.agents/skills/ticket-flow`;
- scaffolds `tickets/` (the seven ordered status dirs, each with a `.gitkeep`), seeds
  `tickets/BOARD.md` from the skeleton (child name + date substituted) and a short
  `tickets/README.md` pointer;
- injects an idempotent `<!-- pickle:begin -->`…`<!-- pickle:end -->` marker block into
  `AGENTS.md` and `CLAUDE.md` (`--claude-symlink` makes `CLAUDE.md` a symlink to `AGENTS.md`;
  `--no-claude` skips all Claude artifacts);
- writes `pickle.toml` registering the first child-project (`--project`, default: the root dir
  name; `--path`, default `.`).

**Per-project** (never writes to `~/` or outside the root), **idempotent, and safe to re-run**:
the payload and marker block are refreshed in place while instance data (`BOARD.md`, tickets,
`pickle.toml`) is preserved once present. After installing, a post-install `board audit`
self-check must pass. An existing `.agents/skills/ticket-flow` **symlink** (a dev/self-host
link) is left untouched.

The pi/opencode agent wiring (`.pi/`, opencode) lands later (their own tickets); today
`install` covers the agent-agnostic `.agents/` skill + `AGENTS.md` and the Claude Code view.

## `pickle ticket new`

```
pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V]
```

Allocates the next `T-NNN` (one global namespace = `max(existing) + 1`), writes a clean,
audit-passing scaffold to `tickets/1-to-do/T-NNN-<slug>.md` (filled frontmatter + the standard
section skeleton), and inserts the board row under the child's `### <child>` sub-group in the
TO DO section, in impact order. `--project` must name a registered child; grades default to
`medium`/`medium`/`M` and must be legal. The agent fills the Description prose afterwards; the
full `TEMPLATE.md` (installed with the skill) is the authoring guide.

## `pickle ticket move`

```
pickle ticket move <T-NNN> <status> [--reason "<why>"]
```

Moves a ticket through the flow as **one operation** — the three edits the board rule demands,
together: relocate the file between `tickets/<status>/` dirs, append a dated `## History`
transition line, and rewrite the board row into the target section (correct child sub-group
*and* column shape). `<status>` accepts the dir name (`3-in-development`), the short form
(`in-development`), or the display name (`"in review"`).

**State machine** (anything else is rejected):

```
to-do → ready → in-development → in-review → done
                                     ↘ rework → in-review
backward/abort:  in-development → ready,  ready → to-do,  (any non-terminal) → dropped
```

`done` and `dropped` are terminal. Gates enforced before anything is written:

- **Sign-off** — `--reason` is required for every backward move, every `→ rework`, and every
  `→ dropped`; the human running/approving the command is the sign-off, and the reason is
  recorded in History (and the DROPPED/REWORK board cell). Forward moves may omit it.
- **Per-child WIP** — moving into `in-development`/`in-review` cannot exceed that child's limit
  (from `pickle.toml`); counts only the ticket's own `project`.
- **Dependency + cross-child merge gate** (pickup only) — every `depends-on:` target must be
  `done` **and** carry a `MERGED` History line (the cross-child proxy for "merged to its own
  base"). This is intentionally stricter than `board audit`, which only warns.

After applying, `ticket move` runs `board audit` as a self-check and fails loudly if the tree
is no longer clean.

## Build

Requires Go 1.26+ and (optionally) [`just`](https://github.com/casey/just).

```sh
just build        # → ./pickle   (or: go build -o pickle .)
just test         # go test ./...
just lint         # go vet + gofmt check
./pickle version
```

## Phased build plan

- **P1 — payload + audit engine.** Embed resources; `pickle board audit` (invariants incl. the
  multi-project ones); `pickle ticket new --project` (id allocation + template + board row).
- **P2 — install + project registry.** `install`, `pickle.toml` `[[project]]` registry,
  `project add/list/remove`, marker injection, skill install (Claude Code + Zed/Pi), `doctor`,
  `upgrade`, `uninstall`.
- **P3 — moves + state machine.** `ticket move` (state machine, per-child WIP, cross-child merge
  gate); `board sync`.
- **P4 — multi-agent breadth.** opencode wiring; Pi guardrail scaffold.
- **P5 — distribution.** `go:embed` release build, Homebrew tap, releases, docs.

## Layout

```
.
├── main.go              entrypoint
├── assets.go            //go:embed all:skill  (the payload + build version)
├── internal/cli/        command surface (dispatch + one file per command)
├── skill/               embedded skill payload (source of truth for the flow)
│   ├── SKILL.md
│   └── resources/       tickets-README.md, TEMPLATE.md, review-protocol.md, BOARD.md
├── tickets/             pickle's OWN board (self-hosted flow) + seeded backlog
├── pickle.toml          overarching config + [[project]] registry (child: pickle)
├── AGENTS.md            marker block: start at tickets/BOARD.md + project config
└── .agents/skills/ticket-flow -> ../../skill   (skill discovery; .claude mirror)
```

> Note: the module path is `pickle` (bare) for the skeleton; it becomes a real VCS path
> (`github.com/…`/`gitlab.com/…`) at P5 when distribution is wired up.
