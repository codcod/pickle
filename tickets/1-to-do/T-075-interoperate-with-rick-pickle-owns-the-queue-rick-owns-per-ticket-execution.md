---
id: T-075
title: interoperate with rick: pickle owns the queue, rick owns per-ticket execution
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: high
cost: XL
---

# T-075 — interoperate with rick: pickle owns the queue, rick owns per-ticket execution

## Description

**Umbrella ticket.** Members: T-076, T-077, T-078, T-079 (via `family:`). It carries the
rationale and the invariants they all inherit; it ships nothing itself.

`rick` (the `ai-sdlc` project, a separate GitLab repo — not a registered child-project here)
is a V-Model workflow layer for Claude Code. A ticket at `docs/tickets/<JIRA-KEY>.md` is
driven through Research → Plan → Implement → Validate, each phase emitting an artifact under
`docs/specs/<KEY>/` that a human approves at an `[A]pprove / [R]evise / [D]iscard` gate.

The reason to interoperate rather than absorb is that **rick and brine are not two answers
to the same question**:

- **brine is a queue discipline.** Many tickets, one board, WIP limits, impact ordering,
  dependency gating, invariant audit. It answers *what next, and is the backlog healthy?*
  State lives **between** tickets — as the directory a file sits in.
- **rick's V-Model is an execution discipline.** One ticket, four phases, an approved
  artifact per phase, a human gate at each boundary. It answers *how does this one ticket get
  from words to merged code without skipping verification?* State lives **within** a ticket —
  as the set of artifacts on disk whose frontmatter says `status: approved`. rick's own docs
  make this argument about Scrum (`docs/shared/foundations/v-model-in-scrum.adoc`: Scrum
  answers *when*, the V-Model answers *how*, and is "a quality engine that runs inside your
  existing process"); substitute brine for Scrum and it is this design.

Each lacks exactly what the other has. rick has no board, no WIP limits, no id allocation, no
dependency gating and no invariant audit. brine has one undifferentiated review and an
honour-system "acceptance tests pass". So the split is: **pickle owns the queue and the
reading surface; rick owns per-ticket execution and every approval.**

### Invariants every member inherits

1. **pickle never writes `status: approved`.** That single-field frontmatter edit is the
   approval gate's, and the gate is a human act in the agent's TUI. rick's shared gate is
   explicit that this is "the one place the status becomes `approved`". If pickle ever writes
   it, rick's phase gates are satisfied by a document no agent presented and no validator
   checked.
2. **Validated-before-gate must survive.** rick promises you are "never asked to approve a
   malformed or incomplete document" because `s-artifact-validator` runs first. Any
   human amendment lands *after* that validation, so it must re-enter validation before the
   gate is answered (see T-079).
3. **`docs/specs/**` has one writer today.** rick grants that scope to the Claude Code
   session; the discipline is *subagent renders → main session writes verbatim*. A browser
   POST is a second, unsynchronised writer — against a session that is parked at a gate and
   may at any moment overwrite the path (`[R]evise`) or delete the file (`[D]iscard`).
4. **Fail open.** rick absent, not installed, or a different version ⇒ the feature is
   silently off. No new errors in `doctor` or `board audit` for projects that never heard of
   rick.

### The two seams that make it cheap

- **State:** `rick status --json` is already a versioned public contract
  (`sdlc-cli/internal/status/report.go`: `schemaVersion = 2`, evolution additive-only,
  "removing or renaming a field is a breaking change that requires bumping this constant").
  pickle consumes it and never reimplements the scan.
- **Identity:** rick keys artifacts by Jira key and pattern-matches `[A-Z][A-Z0-9]*-[0-9]+`.
  T-058 (done) gave each child a `ticket_prefix`, and `audit.go` already enforces
  prefix↔project consistency. Set `ticket_prefix = "DR"` and a pickle ticket **is** `DR-142`
  — the same string as `docs/specs/DR-142/`. No new frontmatter field; the seam is a path
  join.

### Not in scope

The rick-side asks (a deterministic `rick check artifact <path>`; an `Amend` verb in the
shared approval gate; actually *writing* `.ai-sdlc/vmodelsessions/current-phase.json`, which
is declared at `status/workflow.go:17` but written by nothing) cannot be tickets on this
board — every ticket must target a registered child-project, and rick should not be
registered as one. They are recorded in `tickets/NOTES.md`.

Soft couplings: T-065 (versioned JSON read projection) is the mirror image of T-076 — two
tools exchanging JSON projections in opposite directions, and they should agree on
versioning style; T-056 (make the serve dashboard writable) shares T-079's concurrency
problem but operates on `tickets/`, a different tree with a different other-writer.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
