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

The embedded payload lives in [`resources/`](resources/) (`SKILL.md`, `tickets-README.md`,
`TEMPLATE.md`, `review-protocol.md`, `BOARD.md`) and is compiled into the binary via
`//go:embed`.

## Commands

```
pickle install                          scaffold + install skill + markers + pickle.toml + first child   [P2]
pickle project add|list|remove          manage connected child-projects                                  [P2]
pickle upgrade                          refresh installed skill payload + markers                        [P2]
pickle doctor                           verify install integrity                                         [P2]
pickle uninstall                        remove skill/symlinks/markers (keep tickets/)                    [P2]
pickle ticket new "<title>" --project   allocate T-NNN, scaffold ticket, add board row                   [P1]
pickle ticket move T-NNN <status>       move file + History + board atomically                           [P3]
pickle board audit                      check every board/ticket invariant                               [P1]
pickle board sync                       repair board rows from ticket state                              [P3]
pickle version | help
```

This repository is the **initial skeleton**: the command surface, dispatch, exit codes, and
the embedded payload are in place; each command is a stub that reports its target build phase.

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
├── assets.go            //go:embed all:resources  (the payload + build version)
├── internal/cli/        command surface (dispatch + one file per command)
└── resources/           embedded skill payload (source of truth for the flow)
```

> Note: the module path is `pickle` (bare) for the skeleton; it becomes a real VCS path
> (`github.com/…`/`gitlab.com/…`) at P5 when distribution is wired up.
