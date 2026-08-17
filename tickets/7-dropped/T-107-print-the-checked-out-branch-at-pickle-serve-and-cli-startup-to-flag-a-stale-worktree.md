---
id: T-107
title: print the checked-out branch at pickle serve and CLI startup, to flag a stale worktree
project: pickle
depends-on: []
spawned-by: [T-065]
impact: medium
complexity: low
cost: S
---

# T-107 — print the checked-out branch at pickle serve and CLI startup, to flag a stale worktree

## Outcome

After this ships, `pickle serve`'s startup line and `pickle board state --json`'s
envelope both name the git branch pickle is reading tickets/ from — so a reader looking
at either surface from a feature-branch checkout whose bookkeeping has since moved on
`main` sees a plain fact instead of confidently-wrong ticket status with zero indication
anything is stale.

## Description

`pickle` has no git-awareness anywhere in its read path — every command, `serve` included,
resolves its root via `loadConfig()` walking up from the current working directory to find
`pickle.toml` (`internal/cli/serve.go:57,76`), then reads whatever `tickets/` looks like on
disk. `rg 'branch|checked out' internal/serve/ internal/doctor/` (excluding tests) returns
nothing — neither the dashboard nor `doctor` has ever considered which branch is checked out.

This is normally harmless, because bookkeeping only ever moves on the base branch (rules §0)
and a feature branch is usually short-lived relative to that cadence. It stops being harmless
the moment a branch outlives even one bookkeeping move on the base — which a review-and-rework
cycle routinely does.

**Demonstrated, not hypothetical.** T-065's own feature branch (`feat/T-065-json-read-projection`,
last commit `8b4caa6`) sat through a review, a rework round and a re-review while its own
`tickets/` copy stood still. Running `pickle serve` against a worktree checked out at that
commit — after `main` had already moved the ticket all the way to `6-done/` and merged it —
rendered:

```
GET /t/T-065  →  <dt>status</dt><dd>IN DEVELOPMENT</dd>
GET /         →  T-065 rendered under the "IN DEVELOPMENT 1/1" lane
```

identical markup, identical route, and nothing on the page distinguishes it from a correct
read — no branch name, no staleness flag, nothing. The same instant, `main`'s own `pickle serve`
showed `DONE`. A teammate (or a different agent, mid-session, told to review or check on a
ticket from that branch's checkout) has no signal from the tool itself that what they're
looking at might be behind.

The drift is one-directional and worth naming as such: since bookkeeping only lands on the base
branch, a stale worktree can only under-report progress (show something as still open when it
has shipped), never claim `DONE` for something still in flight. That makes it a quieter, easier
to shrug off, illusion — which is part of why it is worth flagging rather than leaving silent.

**Scope, left for refinement to pin down, is one or both of:**

1. **`pickle serve`'s startup line** — the natural first surface, since it is the long-running,
   casually-browsed one. `internal/vcs.Output(root, "rev-parse", "--abbrev-ref", "HEAD")`
   already exists and does exactly this shell-out; this ticket is almost entirely about *where*
   to print it and *how prominently* (one line at process start vs a persistent header on every
   page), not about writing new git-plumbing.
2. **`pickle board state --json`'s envelope`** — T-065 already carries `root` (absolute path)
   at the top level; a `branch` field alongside it costs one more `vcs.Output` call and is a
   compatible (additive) schema change under T-065's own versioning contract (decision 4), not
   a `schema` bump.

**Soft couplings (not `depends-on`):**

- **T-065** (done) — the read projection and its `root`/envelope shape this ticket would extend
  if scope includes the JSON surface; also where `internal/vcs` isn't yet used, so this would be
  its first consumer outside the hooks it was built for.
- **T-057, T-072, T-082, T-100** (all done) — the existing branch-awareness machinery, but all
  four are enforcement-side (stopping a *write* from landing on the wrong branch). This ticket
  is the observability-side counterpart: telling a *reader* which branch a read came from. Same
  family of problem, opposite direction, and no overlap in code touched.
- **`docs/proposals/post-merge-done-move.adoc`** — a design note (not yet a ticket) that
  independently flagged this exact gap while working through a review of T-065; this ticket is
  the one it named as the minimal, self-contained fix, filed on its own rather than bundled into
  that larger, not-yet-approved proposal.
- **`internal/doctor`** — has its own notion of hook/payload staleness already; worth checking
  during refinement whether a "which branch am I reading" line belongs there too, or whether
  `serve`'s startup line alone is enough.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-16 — created (TO DO). source: chat: discussion following T-065's review, exploring
  whether a different agent/session would notice a ticket was ready to review and what
  `pickle serve` would show from a stale checkout — surfaced by empirically running `serve`
  against T-065's own merged-and-deleted feature branch tip (`8b4caa6`) and comparing to `main`
- 2026-08-17 — TO DO → DROPPED: superseded by T-108, which derives the same warning from a recorded layout instead of the main/master guess
