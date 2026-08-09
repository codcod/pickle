---
id: T-082
title: pre-push hook: refuse a feature-branch push whose range carries tickets/ paths
project: pickle
depends-on: []
spawned-by: [T-072]
impact: medium
complexity: medium
cost: M
---

# T-082 — pre-push hook: refuse a feature-branch push whose range carries tickets/ paths

## Outcome

After this ships, a `git push` whose range still carries a `tickets/` path on a feature branch is refused before it reaches the remote, closing the one gap the existing pre-commit hook and the origin-base prose check both leave open at publish time.

## Description

Rules §0 splits every change in two — code on the child's feature branch, ticket and board
bookkeeping on the base branch — because a squash-merge of a branch carrying bookkeeping folds or
drops it and leaves `BOARD.md` disagreeing with the tickets it indexes. (T-084 note: for a
root-path child the default merge is now rebase/keep-history, under which the same bookkeeping is
*preserved* rather than folded — landing on the base as out-of-order commits instead of
vanishing. The hazard and this ticket's fix are unchanged; only the folding half of the rationale
above is specific to squash.) T-057 shipped a
`pre-commit` hook enforcing it at **commit** time. T-072 established that the same failure also
arrives at **publish** time, where that hook is structurally blind, and closed it **in prose**
(§0, review-protocol step 9, and `TEMPLATE.md`).

This ticket asks whether that prose should also be mechanical. The hazard has now appeared **four
times** — T-053/T-054 (bookkeeping committed on the branch), T-022 (branch cut before bookkeeping
landed → stale ticket), T-068 (caught during publish, pre-push), and T-073 (**not** caught: it
reached `origin/main` in squash `7b33876`, PR #18). Prose is followed by whoever reads it; the
fourth occurrence happened to an operator who had read it.

### The correction this ticket rests on

T-072's Description originally dismissed this shape: *"the failure here was the absence of a
push, not a bad one, so `pre-push` on the feature branch would not have caught it."* **That
reasoning is wrong**, and T-072 records the correction. The guard does not need to observe the
missing *base* push. It fires on the **feature-branch push** — which does happen, in every one of
these incidents — and measures the branch against `origin/<base>`. Verified against T-073's real
SHAs (`origin/main` = `152fea8`, branch head = `850ea3c`): `git diff --name-only 152fea8...850ea3c`
named **7 `tickets/` paths**, so the guard would have refused the push and printed the one-line
repair. It would equally have caught T-068's.

### Shape (for refinement — nothing is decided)

1. **The check.** Identical in substance to the one T-072 put in §0: on a push whose local ref is
   a **feature branch**, refuse if `git diff --name-only <remote-sha>...<local-sha>` names any
   `tickets/` path. `pre-push` receives `<local ref> <local sha1> <remote ref> <remote sha1>` on
   **stdin**, one line per ref, so the range is computed exactly rather than guessed — and the
   deleted-branch case (`local sha1` all-zero) must be skipped.
2. **It must not fire when the ref being pushed is the base branch.** Bookkeeping on the base is
   the *correct* destination — every `git push origin main` in this repo carries `tickets/`
   paths by design. This is the single most important way to get the guard wrong.
3. **Fail-open, exactly as `pre-commit` does.** `hook.Shim()`'s contract (T-057 decision 3) is
   that **only exit 1 means violation**; any other non-zero is reported on stderr and waved
   through, because an older `pickle` first on `PATH` exits 2 on an unknown verb and must not
   block every push in the repository. The `pre-push` shim inherits this verbatim — it must never
   grow an `exit 1` on the guard-absent branch.
4. **The structural cost, which is the real reason this is not S.** `internal/hook` is currently
   written around *exactly one* hook: `const HookName = "pre-commit"` (`hook.go:48`), and
   `Status()`, `Install()`, `Uninstall()` and the `doctor` probe each assume that single name and
   path. Supporting a second hook means generalizing that to a set — install/uninstall/status
   over N hooks, N marker lines, N staleness checks — and `pickle upgrade` must refresh both.
   The check itself is a dozen lines; this generalization is the bulk of the work.
5. **`ShimVersion` bump and ownership.** Adding a hook is a shim-text change (→ v3), which
   `pickle upgrade` already refreshes on. `.git/hooks/pre-push` may already exist and be the
   user's own: the same marker-prefix ownership rule as `pre-commit` applies (`# pickle:hook v`),
   and a foreign hook must never be clobbered.
6. **`doctor` and the docs.** `doctor` should report the new hook the way it reports `pre-commit`
   (absent / stale / foreign). `hooks install`'s output, `cli-reference.adoc`, and the
   `hooks`-related manual prose all describe "a `pre-commit` hook" in the singular and would need
   re-wording.

### Open questions for refinement

- **Is `pre-push` the right shape at all, or is `pickle publish-check` better?** A subcommand the
  protocol calls explicitly is discoverable, testable without git plumbing, and works for people
  who never ran `hooks install` — but it is opt-in, which is exactly the property that failed
  four times. A hook is automatic but per-clone and invisible. T-072 named both; neither was
  chosen.
- **Does this supersede any of T-072's prose, or sit alongside it?** Probably alongside — hooks
  are per-clone and `--no-verify` exists — but refinement should say so explicitly rather than
  leaving two half-owners of one rule.
- **Grading is provisional** (medium/medium/M) and rests on item 4. If refinement finds the
  one-hook assumption cheaper to generalize than it looks, this may be an S.

### Soft couplings

- **T-072** — lineage (`spawned-by`); shipped the prose this would mechanize, and carries the
  measured evidence plus the corrected reasoning above. Not a dependency: this ticket is coherent
  whether or not T-072's prose is ever changed again.
- **T-057** — shipped `internal/hook`, the shim contract, and the one-hook assumption in item 4.
  Its decision 2 (keep `board audit` git-free) is why a publish check is not bolted onto the
  audit, and its decision 3 is the fail-open rule in item 3.
- **T-068** — shipped the inert-guard probe; whatever this adds must be probed the same way, or
  it inherits the failure mode where a dead guard is indistinguishable from a satisfied one.
- **T-071** — hardens that probe; overlaps this ticket's item 6 if both touch `doctor`'s hook
  reporting.
- **T-066** — CLI-surface documentation gaps; item 6's `cli-reference.adoc` re-wording lands in
  the same tree.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-07 — filed from T-072's review at the user's request. T-072 closed the publish-time
  bookkeeping leak in prose and, in doing so, corrected its own earlier reasoning that a
  `pre-push` guard could not catch it — it can, verified against T-073's real SHAs (7 `tickets/`
  paths in `152fea8...850ea3c`). Filed rather than folded into T-072 because it is Go code across
  `internal/hook`, `internal/cli` and `doctor` plus a `ShimVersion` bump, against T-072's
  prose-only scope. Passes the promotion test on frequency alone: four occurrences, one of which
  shipped
- 2026-08-09 — T-084's review impact sweep: noted in the Description that the rebase/keep-history
  default for a root-path child changes the failure's *shape* (preserved, not folded) without
  changing this ticket's scope or its three-dot `origin/<base>...HEAD` check.
