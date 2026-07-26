# pickle

`pickle` installs and operates a **ticket-based, board-driven feature flow** in any project —
the same flow prototyped by hand in an earlier workspace, now a single-binary CLI.

Run `pickle install` in a project and a coding agent (Claude Code, opencode, Pi) understands
requests like *"create a feature to review the Jira board"*: it authors a correctly-formatted
ticket in `tickets/1-to-do/T-NNN-<slug>.md` and regenerates `BOARD.md`, because `install` laid
down the flow skill, the board, and the agent-instruction markers.

## Install

> **Two different “installs.”** *This* section installs the **`pickle` binary** on your machine.
> The [`pickle install`](#commands) *command* is a separate step you run **inside a project** to
> lay down the flow (skill, board, markers).

**Homebrew** (via the tap):

```sh
brew install codcod/taps/pickle
```

**`go install`** (Go 1.26+, any platform):

```sh
go install github.com/codcod/pickle@latest
```

**From source** — see [Build](#build).

Then, inside any project:

```sh
pickle install        # lay down the ticket flow
pickle version
```

> `brew install` / `go install` work once the GitHub repo + tap exist and a release tag is cut
> — see [`RELEASING.md`](RELEASING.md).

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
- **Mechanics** (scaffold dirs, allocate the next `T-NNN`, move a ticket file + append History +
  regenerate the board atomically, audit the invariants) are deterministic **CLI commands**.

The embedded payload lives in [`skill/`](skill/) — laid out exactly as the installed skill
(`skill/SKILL.md` + `skill/resources/{tickets-README,TEMPLATE,review-protocol}.md`) — and
is compiled into the binary via `//go:embed`. `pickle` **self-hosts** this flow: its own board
lives in [`tickets/`](tickets/), the skill is discoverable via `.agents/skills/ticket-flow`
(a symlink to `skill/`), and [`pickle.toml`](pickle.toml) registers the sole child-project
(`pickle`, at the repo root).

## Commands

```
pickle install [--agent …]              scaffold + install skill + markers + pickle.toml + first child   [done: T-004]
pickle project add|list|remove          manage connected child-projects                                  [done: T-001]
pickle upgrade                          refresh installed skill payload + markers                        [done: T-006]
pickle doctor                           verify install integrity                                         [done: T-005]
pickle uninstall [--dry-run]            remove skill/symlinks/markers (keep tickets/ + pickle.toml)      [done: T-006]
pickle ticket new "<title>" --project   allocate T-NNN, scaffold ticket, regenerate board                [done: T-003]
pickle ticket move T-NNN <status>       move file + History + regenerate board atomically                [done: T-007]
pickle board audit                      check every ticket invariant + board freshness                   [done: T-002]
pickle board sync                       regenerate the board from ticket state                           [done: T-008]
pickle version | help
```

This repository is **early**, but the full command surface is implemented: `install`
(T-004), `project add|list|remove` (T-001), `ticket new` (T-003), `ticket move` (T-007),
`board audit` (T-002), `board sync` (T-008), `doctor` (T-005), and `upgrade`/`uninstall`
(T-006).

## Configuration — `pickle.toml`

`pickle.toml` lives at the overarching-project root and is **tool-managed**, but only one
command rewrites a file you already own:

- **`project add|remove`** re-render the whole file to a canonical layout. That re-render
  **does not preserve comments** — this is the one case where hand-written notes are lost.
- **`upgrade`** rewrites only the `payload_version` line. Comments, blank lines, key order and
  every other line's spacing survive; that one line comes back normalised to
  `payload_version = "value"`, with any inline comment after the value kept — so that one line's
  alignment and quoting style are not preserved. Because the edit is line-based, some shapes are
  beyond it: a quoted `"payload_version"` key, a `payload_version` whose own value is a
  multi-line string, or a multi-line string holding a line that looks like a `[table]` header or
  like the key itself, sitting **above** the key (above the first `[table]` header, in a file
  that has no `payload_version` yet). The result is parsed and compared against the original
  before anything is written, so those shapes make `upgrade` **refuse and change nothing**, with
  an error telling you to set the version by hand. Position is what decides it, not the mere
  presence of a multi-line string: the scan stops at the key, so a multi-line string *below* the
  key is never read and cannot trigger a refusal, and one above it is read correctly as long as
  it holds neither of those two lines.
- **`install`** writes `pickle.toml` only when it does not yet exist; on an existing file it
  reports `= pickle.toml (exists)` and leaves it alone.

Hand-edits are always preserved on load. The keys:

- **overarching:** `payload_version`, optional `review_addendum`, and a `[commit]` table
  (`overarching_auto`, `child_publish_gated`). Both default to **`true`** when the key or the
  whole table is absent — the cautious reading, and what `install` writes: a child is not pushed
  without your approval. Set either to `false` explicitly to opt out; the wording rendered into
  `AGENTS.md` follows whichever way they are set (see
  [`pickle upgrade`](#pickle-upgrade));
- **`[[project]]` array** (one per connected child): `name`, `path` (relative to the root),
  `build`/`test`/`lint`/optional `docs`, `branch_prefix` (default `feat/`),
  `wip_in_development` / `wip_in_review` (default 1), optional per-child `review_addendum`.

TOML is decoded with [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) — a
**build-time** dependency compiled into the static binary; nothing is fetched at runtime.

## `pickle board audit`

A pure, dependency-free check of the flow invariants over `tickets/` + `pickle.toml`. It exits
non-zero (and prints `ERROR:`/`WARNING:` lines plus a summary) when any invariant fails:

- each ticket file sits in a known status dir and is named `T-NNN-<slug>.md`;
- frontmatter is complete (`id`, `title`, `project`, `depends-on`, `spawned-by`, `impact`,
  `complexity`, `cost`), grades are legal, the id matches the filename, and ids are unique
  globally;
- `project:` names a **registered child**, every `depends-on:` target exists, and every
  `spawned-by:` target exists without a ticket citing itself — lineage is checked for
  existence but **never gates** anything;
- `BOARD.md` matches a fresh render of the ticket files — the board is a **generated
  artifact** (T-044), so any hand-edit or staleness is exactly one error:
  `BOARD.md is stale or hand-edited — run pickle board sync`;
- **per-child** WIP limits (`wip_in_development`/`wip_in_review`) hold;
- each ticket's last History transition matches its directory;
- nothing is in `3-in-development/` with a dependency not yet in `6-done/` (warning if a done
  dependency has no `MERGED` History line) — `spawned-by:` parents are deliberately exempt.

> **Upgrading an existing project:** `spawned-by:` is a **required** key, and `pickle upgrade`
> never touches tickets. Tickets written before this key existed therefore report
> `frontmatter missing "spawned-by"` until you backfill them — add `spawned-by: []` under each
> ticket's `depends-on:` line (a one-line mechanical edit per file).

Missing empty status directories are treated as empty, not errors (git does not track empty
dirs).

## `pickle board sync`

```
pickle board sync [--dry-run]
```

**Regenerates `BOARD.md` wholesale from ground truth** (ticket files + frontmatter +
`pickle.toml`). The board is a pure generated artifact (T-044): the whole file — banner,
preamble, WIP-limit lines, and the seven status sections — is a render, and **nothing in it is
preserved**; hand-written planning notes belong in `tickets/NOTES.md`, which sync never
touches. **"In sync" is defined as `board audit` reporting zero errors**, so a successful sync
always leaves the board audit-clean.

Everything on the board is derived:

- section per status directory, one `### <child>` sub-group per registered child, `(n/limit)`
  WIP counts computed at render time;
- deterministic ordering — TO DO/READY by descending impact (ties by id), everything else by
  id; every DONE/DROPPED ticket is always listed (nothing "ages off");
- DONE `merged` from the ticket's merge History line (`yes — merged to …` / `no —
  publish-gated`), DROPPED `reason` and REWORK `open findings` from the last transition's
  `--reason`;
- every cell passes one-way sanitisation (`|` → `¦`, newlines → space) — cells are never
  parsed back, so a pipe in a title cannot corrupt the table.

A board differing from a fresh render only in the `Last updated:` date is in sync — no write.
`--dry-run` reports the drift, writes nothing, and **exits non-zero if the board would change**
(wire it into CI). After a write, sync runs `board audit` as a self-check.

## `pickle install`

```
pickle install [--project <name>] [--path <path>] [--build/--test/--lint/--docs <cmd>]
               [--agent claude,opencode,pi] [--claude-symlink]
```

Run once in a project to lay down the flow. It:

- copies the embedded skill payload into `.agents/skills/ticket-flow/` (real files) —
  including `resources/docs-readability.prompt.md`, the shared system prompt for the optional
  docs-readability reviewer (review protocol Step 4b);
- scaffolds `tickets/` (the seven ordered status dirs, each with a `.gitkeep`), renders a
  fresh `tickets/BOARD.md` (a generated file — see `board sync`), scaffolds `tickets/NOTES.md`
  for hand-written planning notes, and writes a short `tickets/README.md` pointer;
- injects an idempotent `<!-- pickle:begin -->`…`<!-- pickle:end -->` marker block into
  `AGENTS.md`;
- writes `pickle.toml` registering the first child-project (`--project`, default: the root dir
  name; `--path`, default `.`);
- wires up each agent named in `--agent` (see below).

### `--agent` — which coding agents to wire up

`--agent` takes a comma-separated set of `claude`, `opencode`, `pi` (default: `claude`).
There is **no autodetection** — the set is exactly what you name. Every agent reads the same
`AGENTS.md` + `.agents/skills/ticket-flow/` core; the per-agent artifacts are:

| Agent | Artifacts | Ownership |
|---|---|---|
| `claude` | `.claude/skills/ticket-flow` symlink + `CLAUDE.md` marker block (`--claude-symlink` makes `CLAUDE.md` a symlink to `AGENTS.md` instead) | pickle-owned |
| `opencode` | `opencode.jsonc` — the `docs-readability` reviewer subagent + declarative bash guardrails (explicit-pathspec staging, publish gate). OpenCode picks up `AGENTS.md` and the skill natively; this config is the only opencode-specific piece | **user-owned once written** (see below) |
| `pi` | `.pi/extensions/pickle-guardrails.ts` (the same rules as a Pi `tool_call` gate) + `.pi/extensions/docs-readability.ts` (a `docs_readability` tool + `/docs-readability` command) | pickle-owned |

`--no-claude` is **deprecated**: it still works (drops `claude` from the set, with a warning)
— use `--agent` instead.

**`opencode.jsonc` policy.** pickle writes it whole **only if absent** and never parses or
merges JSONC: if the file already exists it is left byte-for-byte untouched and the template is
printed for you to merge by hand. From then on the file is yours — `upgrade` never touches it,
and `uninstall` removes it only while it is still byte-identical to the shipped template.

**The docs-readability reviewer** is a read-only, suggestions-only Gemini reviewer for the
`.adoc`/`.md` files a ticket changed (review protocol **Step 4b** — optional, never blocking).
Both scaffolds default to `github-copilot/gemini-2.5-pro` (one-time GitHub Copilot login;
adjust with `opencode models github-copilot | grep -i gemini`, or
`DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` for the pi extension). Sessions in other
agents (zed, Claude Code, …) can reach the opencode reviewer by shelling out:
`opencode run --agent docs-readability --file <changed.md> "…"` — or record a conscious skip.

**Per-project** (never writes to `~/` or outside the root), **idempotent, and safe to re-run**:
the payload and marker block are refreshed in place while instance data (`BOARD.md`,
`NOTES.md`, tickets, `pickle.toml`) is preserved once present. After installing, a post-install `board audit`
self-check must pass. An existing `.agents/skills/ticket-flow` **symlink** (a dev/self-host
link) is left untouched.

## `pickle upgrade`

```
pickle upgrade
```

Refreshes an existing install to the running binary's payload. It re-copies the embedded skill
payload into `.agents/skills/ticket-flow/` (removing files that the new payload dropped),
re-injects the `AGENTS.md` marker block, and stamps `payload_version` in `pickle.toml`. Takes no
flags and no arguments.

**It never reads or writes anything under `tickets/`** — your board and tickets are instance data
and are not upgrade's business. Only artifacts that already exist are refreshed: `CLAUDE.md` is
re-injected only if it is a regular file (a `CLAUDE.md -> AGENTS.md` symlink is left alone), and
the `.claude/skills/ticket-flow` view is re-linked only if it is already there. A self-host
**symlinked** `.agents/skills/ticket-flow` is left untouched, never followed and overwritten.

Idempotent: re-running at the current version still refreshes the payload and marker block (so
drift is corrected) and reports `already at <version>` instead of failing. A post-upgrade
`board audit` self-check must pass.

Upgrade will not cost you hand-written content in `pickle.toml`: stamping a new version rewrites
**only** the `payload_version` line, leaves every other line alone, and refuses outright if it
cannot do so safely (see [Configuration](#configuration--pickletoml)). Two things it *does*
replace wholesale, by design:

- the `<!-- pickle:begin -->`…`<!-- pickle:end -->` region of `AGENTS.md`/`CLAUDE.md`. It is
  regenerated from `pickle.toml`, so everything it states about your project comes back — the
  registered child names, each child's `build`/`test`/`lint`/`docs` commands, each child's
  `branch_prefix`, each child's WIP limits, and the `[commit]` policy wording — but anything you
  wrote in there yourself does not. Keep notes **outside** the markers, where pickle never looks.
  Because that block is what an agent reads first, the `[commit]` keys decide whether it is told
  to hold a child-project back pending your approval or to push as the work needs.
- the skill directory `.agents/skills/ticket-flow/`, which is pickle-owned and re-copied in
  full. Do not keep hand-written files there (a self-host *symlink* is left untouched).

Agent scaffolds follow the same probe-what-exists rule: the pickle-owned `.pi/extensions/`
files are refreshed **if present** (put customizations in sibling extension files — `doctor`
warns when a pickle-owned one drifts), while `opencode.jsonc` is user-owned after creation and
is **never** touched by upgrade.

## `pickle uninstall`

```
pickle uninstall [--dry-run|-n]
```

Detaches the flow from the project, leaving your work behind. It removes
`.agents/skills/ticket-flow/`, the `.claude/skills/ticket-flow` symlink, the
`<!-- pickle:begin -->`…`<!-- pickle:end -->` marker block from `AGENTS.md`/`CLAUDE.md` (a
`CLAUDE.md` that is a symlink to `AGENTS.md` is removed outright; a regular file keeps everything
outside the markers, and the file itself is never deleted), and the agent scaffolds: the
pickle-owned `.pi/extensions/` files go (pruning `.pi/` only when left empty — your own
extensions keep it alive), and `opencode.jsonc` goes **only** while still byte-identical to the
shipped template; one you have edited is yours and stays.

**`tickets/` and `pickle.toml` are both left intact** — the board, the tickets, and the
child-project registry are instance data, so a later `pickle install` or `pickle upgrade`
re-attaches cleanly to the same configuration.

`--dry-run` (`-n`) lists what would be removed and changes nothing. Idempotent: on an
already-clean tree it reports nothing removed rather than failing. A self-host **symlinked**
skill directory has only the link removed — the tree it points at is never deleted.

## `pickle ticket new`

```
pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V] [--spawned-by "T-NNN[,T-MMM]"]
```

Allocates the next `T-NNN` (one global namespace = `max(existing) + 1`), writes a clean,
audit-passing scaffold to `tickets/1-to-do/T-NNN-<slug>.md` (filled frontmatter + the standard
section skeleton), and regenerates `BOARD.md` (the row lands under the child's `### <child>`
sub-group in the TO DO section, in impact order). `--project` must name a registered child;
grades default to `medium`/`medium`/`M` and must be legal. The `<title>` may not be empty or
contain newlines — it is written into the frontmatter and the heading, so a newline would
corrupt both; a rejected invocation writes nothing at all. (A `|` in a title is fine: board
cells are sanitised one-way at render time.) `--spawned-by` records **lineage** — the
ticket(s) this one was born from (a review finding, an audit, a split) — comma-separated, brackets
optional, defaulting to `[]`. It is provenance, not a dependency: it never gates pickup. Each id is
checked for **shape** (`T-NNN`) at creation, while whether it **exists** is checked by
`board audit` — a forward reference to a ticket not yet filed is legal input. Repeated ids are
de-duplicated. The agent fills the Description prose afterwards; the full `TEMPLATE.md` (installed
with the skill) is the authoring guide.

## `pickle ticket move`

```
pickle ticket move <T-NNN> <status> [--reason "<why>"]
```

Moves a ticket through the flow as **one operation**: write the updated ticket (with its dated
`## History` transition line) to the target `tickets/<status>/` dir, remove the old file, and
regenerate `BOARD.md`. The new file is written **before** the old one is removed, so a crash
can never leave a ticket whose History records a transition that did not happen. `<status>` accepts the dir name (`3-in-development`), the short form
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
  base"). This is intentionally stricter than `board audit`, which only warns. **`spawned-by:`
  takes no part in this gate** — lineage is provenance, so a ticket is pickable however its
  parents stand.

After applying, `ticket move` runs `board audit` as a self-check and fails loudly if the tree
is no longer clean.

## Build

Requires Go 1.26+ and (optionally) [`just`](https://github.com/casey/just). Releasing also needs
[`goreleaser`](https://goreleaser.com) (`brew install goreleaser`).

```sh
just build         # → ./pickle   (or: go build -o pickle .)
just test          # go test ./...
just lint          # go vet + gofmt check
just dist-check    # validate .goreleaser.yaml
just dist-snapshot # local cross-compiled build into ./dist (no publish)
./pickle version
```

Cutting an actual release is tag-driven and documented in [`RELEASING.md`](RELEASING.md).

## Phased build plan

- **P1 — payload + audit engine.** Embed resources; `pickle board audit` (invariants incl. the
  multi-project ones); `pickle ticket new --project` (id allocation + template + board row).
- **P2 — install + project registry.** `install`, `pickle.toml` `[[project]]` registry,
  `project add/list/remove`, marker injection, skill install (Claude Code + Zed/Pi), `doctor`,
  `upgrade`, `uninstall`.
- **P3 — moves + state machine.** `ticket move` (state machine, per-child WIP, cross-child merge
  gate); `board sync`.
- **P4 — multi-agent breadth.** `--agent claude,opencode,pi`: opencode wiring, Pi guardrail
  scaffold, and the shipped docs-readability reviewer (review Step 4b). **[done: T-009]**
- **P5 — distribution.** goreleaser cross-compiled release build, Homebrew tap, tag-driven
  release CI, install docs. **[done: T-011]**

## Layout

```
.
├── main.go              entrypoint
├── assets.go            //go:embed all:skill all:agents  (the payload + build version)
├── internal/cli/        command surface (dispatch + one file per command)
├── skill/               embedded skill payload (source of truth for the flow)
│   ├── SKILL.md
│   └── resources/       tickets-README.md, TEMPLATE.md, review-protocol.md, docs-readability.prompt.md
├── agents/              embedded per-agent scaffolds (opencode/opencode.jsonc, pi/extensions/*.ts)
├── tickets/             pickle's OWN board (self-hosted flow) + seeded backlog
├── pickle.toml          overarching config + [[project]] registry (child: pickle)
├── AGENTS.md            marker block: start at tickets/BOARD.md + project config
└── .agents/skills/ticket-flow -> ../../skill   (skill discovery; .claude mirror)
```

> Note: the module path is `github.com/codcod/pickle`; releases are cut with goreleaser and the
> Homebrew cask is published to the `codcod/homebrew-taps` tap (see [`RELEASING.md`](RELEASING.md)).
