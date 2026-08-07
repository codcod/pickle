---
id: T-078
title: draft rick Revise feedback in the browser for pasting into the agent TUI
project: pickle
depends-on: [T-077]
spawned-by: []
family: T-075
impact: medium
complexity: low
cost: S
---

# T-078 — draft rick Revise feedback in the browser for pasting into the agent TUI

## Description

A small ticket that completes the read-side of T-075 without taking on any of its risk.

rick's approval gate offers three verbs — `[A]pprove`, `[R]evise`, `[D]iscard` — and
`[R]evise` is agent-mediated: the human describes what they want changed, and the subflow
re-runs its generation path with that feedback. So the human's job at the gate is to *write
good revision feedback*, having just read a long document. If they read it in pickle (T-077),
they should be able to draft the feedback there too, in a real text box, beside the artifact
they are critiquing — rather than composing it blind in a terminal prompt.

Scope: a textarea on the artifact view plus a copy-to-clipboard control. The human pastes
the result into the agent TUI as the `[R]evise` input. Optionally the copied text is prefixed
with the artifact path, since the gate is answered in a session that may be several screens
from where the path was printed.

**Amendment stays agent-mediated. pickle writes nothing.** That is what makes this ticket
small and safe: every invariant in T-075 holds trivially, there is no second writer on
`docs/specs/**`, rick's validated-before-gate promise is untouched, and it does not depend on
T-056 (writable serve dashboard) in any way. It is the cheapest available proof that the
queue tool and the execution tool are better together.

T-079 is the version of this that lets the human edit the document directly; the two are
alternatives in ambition, not in sequence — shipping this one first is deliberate, and if
T-079 is later dropped, this still stands on its own.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
