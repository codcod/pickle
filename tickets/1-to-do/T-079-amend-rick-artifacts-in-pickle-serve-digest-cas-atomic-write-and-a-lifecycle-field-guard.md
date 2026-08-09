---
id: T-079
title: amend rick artifacts in pickle serve: digest-CAS, atomic write, and a lifecycle-field guard
project: pickle
depends-on: [T-077]
spawned-by: []
family: T-075
impact: medium
complexity: high
cost: L
---

# T-079 — amend rick artifacts in pickle serve: digest-CAS, atomic write, and a lifecycle-field guard

## Outcome

After this ships, a human can edit a rick artifact directly in `pickle serve` before approving it — the edit content-addressed and lifecycle-guarded, so an edited artifact can never be mistaken for one the agent itself produced and validated.

## Description

The ambitious member of T-075, and the one that has to be careful: let a human **edit** a
rick artifact in the browser before approving it in the agent TUI.

### Why this is not just "add a textarea"

rick's gate has three verbs and **none of them is "I'll fix it myself"**. `[R]evise` means
*ask the agent to redo it* — which is the wrong tool for a typo, a wrong constant, or a
sentence that needs one clause cut. Human amendment is a fourth verb rick does not have
(call it **Amend**), and introducing it from outside collides with two guarantees:

1. **Validated-before-gate.** The artifact was validated by `s-artifact-validator` *before*
   the gate was presented; an edit after that point means `[A]pprove` blesses unvalidated
   content, which the next phase then consumes as a contract.
2. **Single-writer on `docs/specs/**`.** The session holds that write scope and is parked at
   the gate. It can overwrite the path at any moment (`[R]evise`) or **delete the file**
   outright (`[D]iscard` is explicitly permitted to do so).

### Scope

- **Editor:** split-pane `<textarea>` with live preview via an htmx `POST /fragments/preview`
  against the existing goldmark renderer. **No JS editor dependency** — CodeMirror/EasyMDE
  would mean hundreds of KB plus a licence file, and `serve` has a no-build-step discipline
  with htmx as its only vendored asset (`internal/serve/static/`).
- **Digest CAS:** capture a content digest when the editor opens; refuse to save if the file
  changed underneath, and render that as a first-class "the session revised this — reload"
  state rather than a 500. Digest, not mtime. This is the same shape as rick's own best idea:
  a verification record bound to HEAD *and* a worktree digest
  (`sdlc-cli/internal/checks/verification.go`).
- **Atomic write:** temp file + `os.Rename`, so a concurrent reader never sees half a
  document.
- **Lifecycle-field guard:** parse the result back and **refuse the write if `status:` or any
  identity field changed**. This is `config.verifyOnlyPayloadVersion`
  (`internal/config/config.go:347`) inverted — pickle already has the exact pattern for
  "surgical single-field edit, reject if anything else moved". It makes T-075's *pickle never
  writes `status: approved`* invariant a testable claim rather than a convention.
- **Handoff, not silence:** the save confirmation hands the human a one-line instruction to
  paste into the session — *"I amended the artifact in place; revalidate it and re-present
  the gate"* — which restores guarantee (1) by re-running rick's validator before approval.

### Deliberately not blocked on rick

pickle **cannot** validate the edit itself today: the artifact schemas live in rick's skills
as prose for an LLM ("Required Structure and section order", "Validation Checklist" in
`framework/skills/ai-sdlc/artifacts/*/SKILL.md`), not as a machine-checkable spec. The clean
fix is an upstream ask — a deterministic `rick check artifact <path>` — recorded in
`tickets/NOTES.md`. This ticket ships the paste-to-revalidate handoff instead, so it is not
hostage to another product's roadmap; local validation becomes a follow-up if that ask ever
lands.

Soft coupling: T-056 (make the serve dashboard writable — shared write API, tree locking,
ticket field writer) solves the same class of problem for `tickets/`. This ticket writes to a
*different* tree whose other writer is a Claude Code session pickle does not control, so it
cannot simply wait for T-056's lock — but the two should share the atomic-write helper and
the CAS vocabulary rather than inventing two. Refinement should check which lands first.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
