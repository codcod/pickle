# `pickle` — externalizing the ticket flow as a CLI

> **Origin design doc** (formerly `PLAN.md`, renamed by T-019 — old `PLAN.md:NNN` citations in
> `tickets/6-done/` and `tickets/NOTES.md` refer to this file). This is the plan `pickle` was
> built from — distilled from a hand-wired ticket-flow in a separate workspace. It is kept for
> the *why* (rationale and locked decisions); it is **not** the live tracker. The live tracker
> is [`tickets/BOARD.md`](tickets/BOARD.md) — this repo self-hosts the flow it ships, so all
> work flows through tickets (`T-NNN`), not this doc.
>
> **Progress is not tracked here.** `tickets/BOARD.md` is the only place that knows what is
> delivered; a hand-maintained summary in this file duplicated it and drifted (it claimed P4 was
> outstanding for eleven days after T-009 merged it), so it was removed rather than repaired —
> repairing it only restarts the drift. **Everything this doc planned has shipped**; read the
> board's DONE section for anything finer-grained. Where this doc and a ticket disagree, **the
> ticket wins**; treat §-level decisions here as the standing rationale behind them.
>
> Historical references below to the workspace this was distilled from are context, not paths
> in this repo.

## 1. Goal

Ship a CLI-installable tool, `pickle`, that drops the exact ticket-based flow currently
hand-wired into `translator/` into **any** project with one command:

```sh
cd my-project
pickle install
```

After `pickle install`, a coding agent (Claude Code, opencode, Pi) working in that project
understands requests like *"create a feature to review the Jira board"* and responds by
**authoring a correctly-formatted ticket** in `tickets/1-to-do/T-NNN-<slug>.md` and **updating
`BOARD.md`** — because `install` wrote the skill, the generated board, and the
agent-instruction markers that teach it the flow.

The worked example: a project building a Zig-based Jira-board reviewer. `pickle install` there,
then "create a feature to review the Jira board" → a ticket describing that capability, filed
and boarded.

`pickle` also supports an **overarching project that contains several connected
child-projects** — e.g. the frontend and backend layers of one application, or two
cooperating tools. One board spans all of them; each ticket names its target child in
frontmatter and gets a `<PREFIX>-NNN` id from that child's own counter (§3). `translator` is
itself a one-child instance of this (board in `translator/`, build target the nested
`ai-sdlc/` repo). See §3.

## 2. Design principle — split judgment from mechanics

The flow already embodies this split; `pickle` should preserve it hard:

- **Judgment (the agent, driven by the skill + `AGENTS.md`)** — deciding what a ticket says,
  grading impact/complexity/cost against the backlog, refining a plan to the READY gate,
  reviewing and classifying findings blocking vs non-blocking. This is *prompt/skill*
  territory. `pickle` ships it as content, it never tries to automate it.
- **Mechanics (the CLI, deterministic)** — scaffolding directories, rendering the board fresh
  from ticket state, allocating the next id (per-child counter, `max()+1` within that child's
  prefix), moving a ticket file + regenerating the board + appending a History line atomically,
  and running the invariant checker (`check-board.py`). Today these are done by the agent "by
  hand"; every one of them is arithmetic or file-shuffling that a CLI does more reliably.

Net: `pickle` = **installer of the judgment layer** + **executor of the mechanical layer**.
The agent calls `pickle` for the mechanical steps (or a human does), and the skill tells the
agent *when* and *why*.

## 3. Multi-project model (connected child-projects)

The overarching project (where `pickle` is installed and the board lives) can hold several
**connected** child-projects — the design assumption is that they cooperate (frontend +
backend of one app, two interacting tools), so a single board and one backlog are the right
unit of planning. The six locked decisions:

1. **One shared board, sub-grouped by child.** A single `tickets/BOARD.md` indexes every
   ticket across all child-projects. Within each status section the tickets are
   **sub-grouped by child-project** (a `### <child>` heading under the status heading), with
   the impact ordering applied inside each child's group for TO DO / READY. There is exactly
   one board — no per-child boards.
2. **Per-child id counters** *(superseded — originally locked as a single sequence shared by
   every child; see below)*. Each child configures its own `ticket_prefix` in `pickle.toml`
   (default `T`), and an id is `<PREFIX>-NNN` with `max()+1` **within that prefix**, across all
   status dirs. Numbers are unique only within a prefix, so an id is always written qualified
   (`RICK-137`, never "137"). **A single global namespace is now the degenerate case**:
   children that all leave `ticket_prefix` unset share the one legacy `T` counter, which is
   exactly what a single-child project sees. The original lock — one global `T-NNN` sequence
   for every child, chosen to keep allocation trivial and let a ticket move between children
   without renumbering — was unlocked once multi-child workspaces needed to tell a frontend's
   ids apart from a backend's at a glance; re-homing a ticket to a differently-prefixed child
   is now a deliberate renumber, not a free relabel — and a manual one: T-060 proposed a
   `pickle ticket renumber` command and was dropped (`7-dropped/`), the migration being a rare
   one-off not worth automating.
3. **Target named in frontmatter.** Each ticket declares its child-project in a new
   `project: <name>` frontmatter field (the registered short name from `pickle project add`).
   The `tickets/` directory keeps its flat seven-status layout — tickets are **not** split
   into per-child subdirectories; the `project` field + the board's per-child sub-groups do the grouping.
   `board audit` validates every `project` value against the registry.
4. **Each child is its own git repo.** A child-project is a separate repository nested under
   the overarching project (exactly as `ai-sdlc/` sits inside `translator/`). A ticket's
   `feat/T-NNN-<slug>` branch is cut **inside that child's repo**, and the publish-gated
   commit policy (§6) applies there. The board's `branch` columns therefore implicitly
   reference the target child's repo; the ticket's `project` field disambiguates where the
   branch lives.

   **Cross-child dependencies are allowed and gated.** A ticket may `depends-on:` a ticket
   targeting a *different* child (a frontend feature blocked on a backend endpoint). The
   merged-branch pickup gate then checks the **dependency's own child repo** — the dependency
   is satisfied only once its `feat/` branch is merged to the base of *that* child, not the
   consumer's. `board audit` / `ticket move` resolve each dependency's `project` to know which
   repo's merge state to check.
5. **WIP limits per child.** `in-development ≤ 1` / `in-review ≤ 1` (or whatever the project
   sets) apply **independently per child-project**, so the frontend and backend children can
   each carry one ticket in development simultaneously. `board audit` counts WIP per
   `project`, and `ticket move` enforces the limit against the target child's count only.
6. **Explicit registration.** Child-projects are declared with
   `pickle project add <name> <path>` (recorded in `pickle.toml`), not auto-discovered — so
   the set of children is intentional and each carries its own config (build/validate
   commands, branch/commit + publish policy, per-child WIP). `pickle project list` / `pickle
   project remove` round it out.

**Single-child is just the degenerate case.** A project with one registered child (the
`translator` → `ai-sdlc` shape, or the Zig example) works identically: one entry in the
registry, one child sub-group on the board. Nothing about the single-project experience gets
heavier — `pickle install` registers the initial child, and everything above collapses to the
simple case.

## 4. What gets installed (the payload)

The skill payload (`SKILL.md` + `resources/`: the rules, the ticket template, the review
protocol) is **embedded in the binary** and installed into the agent skill dir(s) —
**referenced, never copied**, so `pickle upgrade` can refresh it without touching a project's
tickets. `tickets/BOARD.md` is the one thing that is *not* embedded payload: it is **generated**
fresh from the ticket tree by `install` (only when the file is absent), and regenerated
wholesale by `ticket new` / `ticket move` / `board sync` — never a copied skeleton, never
hand-edited (the original design — a copied skeleton meant to be maintained by hand — was
overturned by T-044).
`check-board.py`'s invariant-checking role is **superseded** by `pickle board audit` (native, no
Python dependency); the script may still ship for editor-standalone use.

## 5. The hard part — multi-agent installation

Each agent discovers "skills" + project instructions differently. `pickle install` must fan out
to whichever agents the project uses (auto-detect + flags):

| Agent | Instruction file | Skill discovery | `pickle` action |
|---|---|---|---|
| **Zed / Pi** | `AGENTS.md` | `.agents/skills/<name>/` (cwd + ancestors to git root) | write skill to `.agents/skills/ticket-flow/`; inject `AGENTS.md` markers |
| **Claude Code** | `CLAUDE.md` | `.claude/skills/<name>/` | symlink `.claude/skills/ticket-flow → ../../.agents/skills/ticket-flow`; inject `CLAUDE.md` markers (or symlink `CLAUDE.md → AGENTS.md`) |
| **opencode** | `AGENTS.md` | opencode's own skill/instruction mechanism (confirm current: AGENTS.md + `.opencode/`?) | inject `AGENTS.md`; add opencode skill hook if it has one |

Strategy:
1. **One canonical skill copy** at `.agents/skills/ticket-flow/`; every other agent points at
   it by symlink (mirrors what `translator` does for Claude Code today). No duplicated payload
   to drift.
2. **Marker-block injection** into `AGENTS.md` / `CLAUDE.md` — idempotent, fenced with
   `<!-- pickle:begin -->` / `<!-- pickle:end -->` so `pickle upgrade` can rewrite just that
   block and `pickle uninstall` can strip it cleanly. The block states: *start at
   `tickets/BOARD.md`*, points to the skill, and holds the **project configuration** (§6).
3. **Symlink `CLAUDE.md → AGENTS.md`** where the project wants a single source (translator's
   pattern), else inject both.
4. **Pi extras (optional)** — if `--agent pi`, also scaffold `.pi/` with the guardrails
   extension pattern (git-staging deny-list, publish gate, self-install guard) adapted to the
   project's config. This is the one place the payload is genuinely agent-specific.

`pickle install` takes **flags** for which agents to wire (`--agent claude,pi,opencode`) or
auto-detects from existing `.claude/`, `.pi/`, `AGENTS.md` — there is no interactive prompt
(§13.4).

**Install scope is per-project, not global.** `pickle` installs the skill, board, markers, and
config into the overarching project's own tree (`.agents/skills/`, `tickets/`, `pickle.toml`,
`AGENTS.md`) — it does *not* write to `~/.agents/` or `~/.claude/`. Each project carries its
own pinned payload version, so different projects can sit on different `pickle` releases and
`pickle upgrade` is an explicit, per-project act. (This deliberately diverges from the skill's
own "promote to global" note: `pickle`'s unit is the project.)

## 6. Project configuration (captured at `install`, consumed by the skill)

The skill is project-agnostic; the project plugs specifics into its `AGENTS.md` marker block.
`pickle install` gathers these from `pickle.toml` / flags — non-interactively (§13.4) — and
writes them.

**Overarching-project config** (one set):
- **Commit policy** (mirrors `translator`'s, uniform across all children):
  - **Overarching project** (where the board/tickets live): bookkeeping commits (tickets,
    board, docs) may be made **automatically**, always with **explicit pathspecs**
    (`git add <paths>`) — never `git add -A`/`.`.
  - **Child-projects** (each build target): **publish-gated.** Do the work on the ticket's
    `feat/T-NNN-<slug>` branch, committing locally at will as you go; then **suggest a
    Conventional Commit message** — `<type>[(<scope>)]: <description>` with the **ticket id in
    brackets at the end of the title**, e.g. `feat(review): add board scanner (T-42)`. Only
    **after the user approves** do you finalize + push the branch and **open the MR**;
    **merging the MR is always the human's**.
- **Extra review rules — layered addenda (shared + per-child, both applied).** The skill ships
  the generic `review-protocol.md`; projects layer extra checks on top, keyed to its step
  numbers (translator's `review-protocol.md` is the template). Two optional layers:
  - an **overarching** addendum (`review_addendum` at the top of `pickle.toml`) for rules true
    of the *whole* project (cross-child consistency, commit-title convention, …);
  - an **optional per-child** addendum (`review_addendum` inside each `[[project]]`) for that
    child's tech-specific checks.
  Reviewing a ticket runs **generic protocol → overarching addendum → the ticket's own child
  addendum** (whichever exist). Common rules live once (overarching); child differences (a
  React frontend vs. a Zig backend) live in their own child addendum — no duplication.
  `pickle install` / `pickle project add` can generate stub addendum files keyed to the
  generic protocol's step numbers.
- **Ticket id prefix / vocabulary** — default `T-`; configurable. Board section names fixed.

**Per child-project config** (one block per registered child — see §3 / `pickle project add`):
- **Name + path** — the registered short name (used in the ticket `project:` field and the
  board column) and the child repo's path relative to the overarching root.
- **Build / validate commands** — build, test, lint, docs (used by the READY gate's
  acceptance test and the review protocol). For the Zig example: `zig build`, `zig build test`.
- **Branch & commit conventions** — default `feat/T-NNN-<slug>` (cut *inside that child's
  repo*) + Conventional Commits with the ticket id appended to the subject.
- **WIP limits** — default `in-development ≤ 1`, `in-review ≤ 1`, enforced **per child**.

Persist everything in a `pickle.toml` at the overarching-project root (a `[[project]]` array
of the registered children plus the shared block) so `upgrade`/`audit` and re-runs are
reproducible.

## 7. CLI surface

The live command surface — every command, every flag, its full contract — is
`docs/user-manual/cli-reference.adoc`; nothing here should describe it a second time, or the
two will drift (as they did: this section's `ticket new` synopsis fell behind `--spawned-by`
and `--family`; its `board sync` prose still described the hand-maintained board T-044
overturned; and its `install` prose described an interactive registration wizard that was never
built — §13.4). What survives is what the manual doesn't cover:

- **Wire `board audit` into CI + a pre-commit hint** — still a live recommendation (see T-057),
  not a shipped feature; the manual documents the command as it exists today, not the
  aspiration.
- **Design tension, unresolved by design:** how much the agent uses the CLI vs. edits files
  directly. The skill instructs the agent to *prefer* the CLI for id allocation and moves
  (mechanical, error-prone by hand) while authoring prose directly in the ticket file. Board
  audit is the safety net that catches either path drifting.

## 8. How the agent actually understands the request

Unchanged from today — this is the whole point of shipping the skill as-is:
1. The `AGENTS.md`/`CLAUDE.md` marker block says *start at `tickets/BOARD.md`* and names the
   skill.
2. The skill's `description` frontmatter triggers on "make it a ticket / create a feature /
   file a ticket" phrasings; its *make it a ticket* procedure (dedupe → author from template
   → grade → board row → History line) is what runs.
3. `pickle` adds one improvement: the skill's procedure can now say "run `pickle ticket new`
   to allocate the id and scaffold" instead of "compute `max+1` yourself", removing the most
   common manual-error class.

So the agent behavior in the Zig example is: hears "create a feature to review the Jira
board" → skill's *make it a ticket* → `pickle ticket new "Review Jira boards" --project
<child>` → fills the Description with the capability spec → board updated. When several
children are registered and the request doesn't name one, the skill has the agent **ask which
child-project the feature targets** rather than guessing.

## 9. Packaging & distribution

- **Language**: strong candidates are Go (single static binary, trivial `embed` of the
  payload, matches the ai-sdlc toolchain habits) or Rust/Zig. Go is the safe default:
  `go:embed` the entire `resources/` tree, cross-compile, distribute via a Homebrew tap +
  `go install` + GitHub/GitLab releases. (Avoid a runtime dep like Python so `board audit`
  needs nothing installed — hence reimplementing `check-board.py` natively.)
- **Payload versioning**: embed a payload version; `pickle upgrade` compares it to the
  installed skill's version and rewrites in place. `pickle.toml` records the version a project
  was initialized/upgraded to.
- **The skill source of truth**: keep the canonical `resources/` in the `pickle` repo; the
  `translator` copy becomes a *consumer* of `pickle`, not the master. (The dogfooding path this
  once pointed at is complete — this repo now self-hosts the flow it ships; see
  `tickets/BOARD.md`.)

## 10. Naming / vocabulary

- CLI: `pickle`; config: `pickle.toml`; marker fences: `<!-- pickle:begin/end -->`.
- Keep the flow's own vocabulary intact (`tickets/`, `T-NNN`, `BOARD.md`, the seven status
  dirs, impact/complexity/cost) — it's proven and the skill is written against it. Make the
  id prefix configurable but default `T-`.
- Skill name stays `ticket-flow` (the discovery name agents match on).
- New multi-project vocabulary: **child-project** (a registered, connected build target with
  its own repo), the ticket `project:` frontmatter field, the board's per-child `### <child>`
  sub-groups, and `pickle project add/list/remove`.

## 13. Resolved questions

All four of this section's original open questions are settled, each as the lean predicted.
Kept as locked decisions with their rationale — that is what this doc is for — rather than
deleted. **They are closed: do not re-litigate them from this file.** Resolutions confirmed
against the code on 2026-08-04.

1. **Agent-call ergonomics** — *offer the CLI, don't mandate it; `board audit` is the backstop.*
   Resolved as leaned. The skill's procedures say to run `pickle ticket new` and to **prefer**
   `pickle ticket move` ("it does all of it atomically and enforces the state machine + per-child
   WIP"), but hand-authoring stays legal and the audit catches the drift.
2. **opencode's exact skill/instruction mechanism** — resolved by **T-009** (`6-done/`).
   opencode gets `opencode.jsonc` (`install.go:40`) alongside the shared
   `.agents/skills/ticket-flow/`.
3. **Marker vs symlink for `CLAUDE.md`** — resolved as *"probably a flag"*, and it is one:
   `install --claude-symlink` (`internal/cli/install.go:26`) makes `CLAUDE.md` a symlink to
   `AGENTS.md`; the default is a dual marker block. **This repo uses the symlink**
   (`CLAUDE.md -> AGENTS.md`).
4. **How opinionated `install` is** — resolved as **flags only**, always writing `pickle.toml`.
   The "optional wizard" half of the lean was never built: there is no interactive prompt
   anywhere in `internal/install` or `internal/cli/install.go`. Non-interactive install is
   what makes `install` scriptable and testable, so the absence is a feature, not a gap.

