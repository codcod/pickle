---
id: T-016
title: ship docs-readability as an optional review step (Step 4b)
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: medium
cost: M
---

# T-016 — ship docs-readability as an optional review step (Step 4b)

## Description

> **PARKED (triage 2026-07-26).** Real, but explicitly unscheduled: nothing is blocked on
> this and no user has asked for it. Do not pick it up without a demand signal. Unparking is a
> user decision — note it in History.

Make the **docs-readability reviewer part of the flow `pickle` ships**, not just a workspace
dev tool. Today a docs-readability capability exists only as *development* tooling for building
pickle (a Pi extension + an opencode subagent + a shared prompt, all kept **outside** the
embedded `skill/` payload so they never ride into `pickle install`); the shipped review
protocol (`skill/resources/review-protocol.md`) stops at **Step 4a** (documentation audit) and
has no readability pass. This ticket adds an **optional Step 4b: a read-only, second-opinion
readability pass over the docs a ticket changed** — suggestions only; the flow agent approves
and applies edits — to the flow that any pickle-driven project inherits.

The reviewer must be **format-agnostic across AsciiDoc and Markdown** (pickle's own `docs/`
will be AsciiDoc; child projects in a pickle-driven workspace may use either), and the step
must be genuinely **optional** — reviewing without it is a sanctioned, recorded skip, so the
flow never blocks on it.

**This is a product decision that respects pickle's "split judgment from mechanics" principle,
and refinement must resolve how far the split goes:**

- **Judgment (definitely in scope).** Add the optional **Step 4b** to the shipped
  `skill/resources/review-protocol.md` (and its `## Review` checklist): *when the host
  environment provides a docs-readability reviewer, get a second-opinion readability pass on the
  ticket's changed `.adoc`/`.md` files; otherwise record a conscious skip.* This is
  agent-agnostic prose and installs with the skill.
- **Mechanics (open — needs a decision).** `pickle install` lays down a **skill + markers**, not
  agent extensions. So does pickle *ship the reviewer executor* too, and if so how?
  Options to weigh at refinement:
  1. **Prose-only** — ship just the Step 4b guidance; each project wires its own reviewer (as
     this workspace already does). Smallest, cleanest, honours "mechanics are the host's."
  2. **Scaffold on install** — teach `install`/`upgrade`/`uninstall` to optionally lay down the
     reviewer wiring per `--agent`: the Pi extension for `--agent pi`, the opencode subagent for
     `--agent opencode`, plus the shared prompt. This **overlaps T-010** (`.pi/` scaffold) and
     **T-009** (opencode wiring) and would extend their payloads.
  3. **A `pickle` subcommand** (e.g. `pickle docs readability <files…>`) that shells out to a
     configured reviewer — makes it reachable from Claude Code / Zed too, not just Pi/opencode.

**Open questions for refinement**

- Which option (1/2/3, or a mix)? If (2/3), does the reviewer's Gemini/Copilot dependency belong
  in a dependency-free CLI's story at all, or stay a pure prose + bring-your-own-reviewer
  contract?
- If scaffolding (option 2), reconcile with T-009/T-010 so the reviewer wiring isn't authored
  twice — likely **fold the scaffolding into T-009/T-010** and keep T-016 as the review-protocol
  Step 4b + the format-agnostic prompt shipped in `skill/resources/`.
- Provider neutrality: the shipped prose must not hard-code Gemini/Copilot (those are host
  choices); keep it "a docs-readability reviewer, if available."
- Docs: pickle's README/`docs/` should document the optional Step 4b and how to enable a reviewer.

**Soft couplings** (no hard `depends-on:` without sign-off): **T-010** (Pi `.pi/` scaffold) and
**T-009** (opencode wiring) — the natural homes for any install-time reviewer scaffolding;
**T-006** (`upgrade`/`uninstall`) — **shipped**; its artifact list is hardcoded in
`install.Upgrade` (`internal/install/install.go:124-148`) and `install.Uninstall` (`:176-212`)
with no agent registry, and its rework scope is closed, so any wiring scaffolded by option 2 must
carry its own symmetric refresh/removal there — the obligation sits with this ticket (or
T-009/T-010), not with T-006. The
reference implementation to adapt is this workspace's dev tooling
(`.pi/extensions/docs-readability.ts`, `opencode.jsonc` `agent.docs-readability`,
`.agents/docs-readability.prompt.md`) — adapt, don't assume it ships verbatim.

**Scope boundary.** This is additive and optional; it must not change the existing Steps 1–4a
or make any review block on the reviewer's availability.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-26 — parked (stays in TO DO, unscheduled). source: board triage — backlog growth analysis
