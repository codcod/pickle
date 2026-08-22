---
id: T-112
title: review protocol has no bias-mitigation step for a same-session implementer reviewing their own ticket
project: pickle
depends-on: []
spawned-by: [T-050]
impact: medium
complexity: low
cost: S
---

# T-112 — review protocol has no bias-mitigation step for a same-session implementer reviewing their own ticket

## Outcome

After this ships, the review protocol ("validate ticket"/"rework ticket") states when a
self-review carries meaningful bias risk — the same agent session that just implemented or
reworked the code is also the one auditing it — and what to do about it, instead of silently
relying on whoever is running the review to notice the conflict on their own.

## Description

The flow already recognises implementer bias once: rules §8's pickup applicability gate requires
the plan's assumptions to be "re-verified against the current target child-project by a spawned,
**unbiased** sub-agent... free of the implementer's sunk-cost bias", run on every pickup, before a
READY ticket enters `3-in-development/`. That clause exists precisely because the agent about to
implement a ticket is the worst-positioned party to judge whether its own plan still holds.

The review protocol (`resources/review-protocol.md`, driving "validate ticket T-NNN" and "rework
ticket T-NNN") has no equivalent. Nothing in its steps 1-9, nor in the rules' §5 findings
machinery, asks whether the agent about to audit implementation/quality/consistency/docs is the
same agent — in the same session, with full memory of every design choice it just made — that
authored the code under review. The bias this omits is at least as strong as the one §8 already
names: a reviewer who just wrote the code is primed to read its own prose as self-evidently
correct, exactly the sunk-cost condition §8 exists to guard against at pickup.

Concretely, in T-050: one continuous session implemented the guardrail fix, then immediately ran
"review ticket T-050" over its own work, found one real blocking finding (a shipped comment that
failed the project's own foreign-workspace test), reworked it, and then re-reviewed its own
rework and closed the ticket — all without the flow ever prompting a consideration of whether that
self-review setup was appropriate. It happened to catch a genuine defect this time; that is one
data point, not evidence the setup is sound. Nothing stops the same session from missing a defect
in its own reasoning precisely because it is the same reasoning.

Open for refinement: what the mitigation should look like — mirroring §8 exactly (delegate the
audit portion of "validate ticket"/"rework ticket" to a fresh sub-agent whenever the reviewing
agent is the same session that authored the branch under review) is the closest precedent, but a
lighter-weight alternative (an explicit self-disclosure line recorded in the ticket's `## Review`
when reviewer and implementer are the same session, so a human reading it can weigh it
accordingly, without mandating a sub-agent spawn for every trivial fix) may be more proportionate.
This ticket does not prescribe which; it is a design decision for refinement, not something to
decide by filing.

This is a **skill-payload** change (`skill/resources/review-protocol.md`, and possibly
`skill/resources/tickets-README.md` alongside §8's own clause), embedded via `assets.go`'s
`all:skill` and shipped to every project that installs brine — it must be phrased so it stands on
its own for a foreign reader (no ticket ids the reader is told to go look up, no "this repo",
per `AGENTS.md`'s foreign-workspace test, `payload_lint_test.go`), a lesson this same ticket's own
provenance (T-050's rework, F1) just supplied first-hand.

### Couplings

Soft coupling only (no `depends-on:`): rules §8's existing "spawned, unbiased sub-agent" clause
(`resources/tickets-README.md`) is the direct precedent and likely gets a cross-reference from
whatever this ticket adds to the review protocol, so both should read as one coherent policy
rather than two independently-invented ones.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: review: T-050's implement/review/rework/re-review cycle
  ran in one continuous session with no consideration of self-review bias; the human asked why
  the flow didn't offer a fresh reviewer, which surfaced the gap between rules §8's pickup gate
  (explicitly unbiased) and the review protocol (silent on it)
