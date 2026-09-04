---
id: T-076
title: consume rick status --json as an artifact-state source
project: pickle
depends-on: []
spawned-by: []
family: T-075
impact: high
complexity: medium
cost: M
---

# T-076 — consume rick status --json as an artifact-state source

## Outcome

After this ships, `pickle doctor -v` on a rick-enabled child reports how many rick artifacts
exist and how many are still awaiting approval, without pickle ever re-scanning `docs/specs/**`
itself; and any Go code in this repo — starting with T-077 — can ask the same question through
one library call. Turning it on costs one config line (`rick = true`) on the child that has it.

## Description

The first member of T-075: teach pickle to learn, for a given ticket, which rick artifacts
exist and which are awaiting human approval. Everything else in the family reads this.

**Consume, do not reimplement.** `rick status --json` is a versioned public contract
(`sdlc-cli/internal/status/report.go`: `schemaVersion = 2`, documented as additive-only —
"removing or renaming a field is a breaking change that requires bumping this constant").
Its `Workflow.Tickets[].Artifacts[]` carries `Path`, `Kind`, `Status`, `Date` per artifact,
derived from disk by rick's own oracle. Crucially the **JSON is multi-ticket** even though
rick's text renderer collapses to a single one, so it is already the shape pickle needs.

Re-deriving that state by scanning `docs/specs/` ourselves would fork rick's kind-detection
and status rules and drift from them silently. Shelling out keeps rick the authority on rick's
state and pickle the presentation layer — the same judgment/mechanics split `DESIGN.md` §2
already draws.

Scope:

- **Opt-in per child-project**, config-declared, never auto-detected — matching `DESIGN.md`
  §3 decision 6 (children are registered intentionally, not discovered). Something like a
  `rick = true` / `specs_root = "docs/specs"` pair on `[[project]]`; the exact schema is a
  refinement decision.
- **Invoke and parse** `rick status --json` against the child's path, keyed by
  `schemaVersion`. An unknown version is refused with a clear message rather than
  best-effort parsed — a silently mis-parsed gate state is worse than no gate state.
- **Map artifacts to tickets via `ticket_prefix` alone.** T-058 (done) already makes a
  pickle id like `DR-142` identical to rick's `docs/specs/DR-142/` directory name, and
  `audit.go` already enforces that a ticket's id prefix matches its child's configured
  prefix. No new frontmatter field, no mapping table.
- **Fail open, always.** rick not installed, not on `PATH`, erroring, or the child not
  opted in ⇒ no artifacts, no warning, no non-zero exit. This must not become a new way for
  `board audit` to go red in projects that never heard of rick.

The deliverable is the library layer plus whatever surfacing is needed to prove it works
(a `--json` field or a `doctor` line); the visual surface is T-077.

Soft coupling: T-065 (expose board and ticket state as a versioned JSON read projection) is
the exact mirror of this ticket — pickle emitting what rick here consumes. They should agree
on versioning discipline, and whichever lands second should copy the first's conventions.

### Re-verified at refinement (2026-09-04)

`pickle` self-hosts against exactly one child — this repository, at `.` — and that child does
not run rick (`pickle.toml` has no `rick` key). So this feature is never dogfooded by the repo
that ships it; every acceptance check below has to build its own synthetic rick-enabled child
rather than pointing at a real one, the same shape `board decisions` (T-105) already used for a
feature this repo has no natural corpus for. Nothing else in the Description above has drifted:
`ticket.SplitID`, `config.Project.Prefix()` and the `audit.go` prefix↔project check it leans on
are all shipped and unchanged since T-058.

**Update, during implementation: the JSON field names are now confirmed, not guessed.** The dev
machine picking this ticket up happens to have the real `rick` CLI installed (`ig/uk/rick`
v0.10.0, the exact `github.com/ig-private/ai-sdlc/sdlc-cli` binary T-075 cites), so the wire
shape was pinned against a scratch `docs/specs/<ID>/*.md` fixture rather than left on trust. It
confirmed `Path`/`Kind`/`Status`/`Date` (lowercase on the wire, `Date` present only when the
artifact's own frontmatter carries one) and corrected the one field this paragraph flagged as
uncertain: a `workflow.tickets[]` entry's identifying field is `id`, not `key` (see History for
the full account). Decision 8's fail-open design is still what made the original guess safe to
ship against — it just turned out not to be needed here.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is the root-path child (`path = "."`), so the branch is cut in this repository:

```
git checkout main
git checkout -b feat/T-076-rick-status-json
```

Local WIP commits are encouraged. Tidy them into atomic commits before presenting (root-path
child, rules §0) and default to keeping that history rather than squashing. Do **not** push or
open a merge request without explicit user approval. Ticket and board bookkeeping is committed on
`main`, never on this branch.

### 1. Prerequisite gate (hard)

None. `depends-on:` is empty. `family: T-075` is lineage only and gates nothing. Every helper this
plan builds on is already shipped: `config.Project`/`Validate`/`Render` (`internal/config`),
`ticket.SplitID` (`internal/ticket/ticket.go:493`), the prefix↔project check in
`internal/audit/audit.go:98-106`, and the shell-out-with-timeout, fail-silent-on-any-trouble shape
`internal/vcs/vcs.go` already established for git.

### 2. Confirmed design decisions (do not deviate without asking)

1. **New package `internal/rickstatus`, not `internal/rick`.** Named for what it produces (an
   artifact-state projection), matching this codebase's convention of naming a package by its
   function (`internal/state`, `internal/decisions`, `internal/metrics`) rather than by the
   external product it happens to read.
2. **Two new per-child `pickle.toml` fields: `rick` (bool, default `false`) and `specs_root`
   (string, default `"docs/specs"`).** Added to `config.Project` (`internal/config/config.go:142`)
   as a `Rick` bool field (TOML key `rick`, `omitempty`) and a `SpecsRoot` string field (TOML key
   `specs_root`, `omitempty`). An accessor `func (p *Project) Specs() string` returns `SpecsRoot`
   or the default when unset — the same shape as `Prefix()` for `TicketPrefix`, and named
   differently from its field for the same reason `Prefix()` is not named `TicketPrefix()` (a
   method cannot share its field's name). `Validate` (`config.go:260`) adds `specs_root` to
   `invalidUTF8Field`'s checklist (`config.go:340`); no filesystem check — rick creates the
   directory lazily, so requiring it to exist would break a child adopting rick for the first
   time. `Render` (`config.go:446`) emits both lines only when `p.Rick` is true, directly after
   `wip_in_review` and before `review_addendum`, so the overwhelmingly common non-rick child's
   `pickle.toml` gains not one byte.
3. **`specs_root` is stored and validated by this ticket but not consumed by its own code.**
   `rickstatus.Query` does not need it: `rick status --json` is invoked with the child's own
   directory as `cmd.Dir` and reports on whatever `docs/specs/**` tree it finds there itself.
   Threading the field through config now — rather than as a second config-schema ticket later —
   is what T-076's own scope bullet ("config-declared… `specs_root = "docs/specs"` pair") asks
   for, and T-077 (`GET /specs/{key}/{name}`) is the first actual consumer, for path-containment
   checks on the artifact it serves.
4. **Invocation: `exec.CommandContext(ctx, "rick", "status", "--json")` with `cmd.Dir` set to the
   child's absolute path** (`filepath.Join(root, p.Path)`), a 5-second timeout (`var rickTimeout
   = 5 * time.Second`, shrinkable in tests — mirrors `vcs.probeTimeout`), and no environment
   scrubbing (rick is not git; there is no repo-pinning variable to fight). `rick` is resolved
   from `PATH`, never a configured binary path — matching `internal/hook.Probe()`'s and
   `internal/vcs`'s existing convention of trusting `PATH` for an external tool.
5. **Only `SchemaVersion == 2` is accepted.** `rickstatus.SchemaVersion = 2` is a single accepted
   value, not a floor (`>= 2`) — mirrors `internal/state.CurrentSchema`'s exact-match precedent
   rather than assuming the additive-only promise makes a higher version safe to best-effort
   parse. Raising the constant when rick ships schema 3 is this ticket's own follow-up, not a
   speculative `>=` check now.
6. **The wire types are unexported and package-private** (`wireDoc`, `wireTicket`,
   `wireArtifact` in `internal/rickstatus/wire.go`), decoded with `encoding/json` and immediately
   projected into the exported `Report`/`Artifact` shape — never returned to a caller directly,
   the same boundary `internal/state`'s package doc draws between a wire format and a package's
   own types.
7. **No filtering by `ticket_prefix` in `Query`.** T-058 already makes a pickle id and rick's id
   the same string when `ticket_prefix` is set correctly, so `Report.Tickets` is keyed verbatim on
   whatever `wireTicket.ID` rick reports *(amended during implementation — see History: the field
   is `id`, not `key`)*. An entry nobody asks about is inert — `For` simply never returns it — so
   a defensive prefix filter would add a task and a test for a problem `For`'s own shape already
   prevents.
8. **`Query` never returns an error. Every failure mode collapses to `Report{Available: false,
   Reason: "…"}`** — `rick` not on `PATH`, non-zero exit, context deadline exceeded, malformed
   JSON, and an unrecognised `SchemaVersion` all produce a distinct, human-readable `Reason` and
   an empty `Tickets` map, uniformly. This is T-075's fail-open invariant made a type-level
   guarantee rather than a convention every caller has to remember.
9. **Doctor surfacing: silent when a child has not opted in; always an `ok()` line — never a
   warning or an error — when it has.** A `rick: false` (the default) child gets no new doctor
   output at all, matching `checkClaudeView`'s silent-when-absent-and-optional shape. A `rick:
   true` child always gets exactly one passed line, whether or not the query actually succeeded:
   `"child %q: rick status ok (%d ticket(s), %d artifact(s))"` when `Available`, or `"child %q:
   rick interop enabled but unavailable (%s) — fail-open, no artifacts shown"` naming `Reason`
   otherwise. Never `r.warnf`/`r.errf` for this check, by construction — T-075's "no new errors in
   doctor… for projects that never heard of rick" extends to projects that *have* heard of it and
   hit a transient failure.

### 3. Tasks

#### Task 1 — `pickle.toml` schema: `rick` + `specs_root`

In `internal/config/config.go`: add `Rick`/`SpecsRoot` fields to `Project` (decision 2), the
`Specs()` accessor, the `invalidUTF8Field` entry, and the `Render` emission. No change to
`applyDefaults` — `Rick`'s zero value (`false`) and `SpecsRoot`'s empty string are both already
the correct "unset" state; only `Specs()` needs to know the default.

#### Task 2 — `internal/rickstatus` package

New files:

- `internal/rickstatus/rickstatus.go` — package doc explaining the fail-open contract (decision
  8) and why this package, not `internal/serve`, owns the shell-out (T-077 and any future
  consumer share it); `Report`, `Artifact`, `Report.For(id string) []Artifact`, the exported
  `Query(root string, p *config.Project) Report` entrypoint, and the exec/timeout plumbing
  (decision 4).
- `internal/rickstatus/wire.go` — the unexported wire types (decision 6) and `parse(out []byte)
  (Report, error)`, called by `Query` with the error folded into a `Reason` string, never
  propagated further.

#### Task 3 — `pickle doctor` surfacing

In `internal/doctor/doctor.go`: add `checkRickInterop(root string, cfg *config.Config, r
*Result)` (decision 9), called from `Check` alongside `checkChildren`/`checkVersion`, inside the
existing `if cfg != nil` block.

### 4. Acceptance test

All runnable via `just test` (`go test ./...`) unless noted.

1. **`internal/rickstatus` unit tests** (`rickstatus_test.go`, following the fake-binary-on-`PATH`
   pattern in `internal/hook/probe_test.go`'s `stubPickleAt`): a stub `rick` script printing a
   fixture built from the documented field shape (decision 6) with `schemaVersion: 2` → `Query`
   returns `Available: true` with the right `Tickets`/`Artifacts`; the same stub with
   `schemaVersion: 3` → `Available: false`, `Reason` names the mismatch; a stub exiting 1 →
   `Available: false`; an empty `PATH` (no `rick` at all) → `Available: false`, no panic; a stub
   that sleeps past `rickTimeout` (shrunk to a few ms for the test) → `Available: false`; a `Rick:
   false` project → `Available: false` *and* the stub is never invoked (assert this by having the
   stub write a sentinel file on any invocation and asserting its absence).
2. **`internal/config` round-trip tests** (`config_test.go`, alongside `TestRoundTrip`): a project
   with `rick = true` and an explicit `specs_root` renders and reloads byte-stable; a project
   with `rick` unset renders with neither line; `Specs()` returns the default for an unset field
   and `Validate` rejects non-UTF-8 `specs_root` (extend `TestAddProjectRejectsInvalidUTF8`'s
   table).
3. **`internal/doctor` test** (`rick_test.go`, alongside the existing `hooks_test.go` split): a
   `rick: false` child produces zero rick-related lines in any of `Errors`/`Warnings`/`Passed`; a
   `rick: true` child with a working stub produces exactly one `Passed` line with the right
   counts and zero `Errors`/`Warnings`; a `rick: true` child with no `rick` on `PATH` still
   produces zero `Errors`/`Warnings` and one `Passed` line naming the reason — the fail-open
   invariant asserted mechanically, not just read off the code.
4. `just lint` and `just docs-check` clean.

### 5. Docs update (mandatory when user-facing)

- `docs/user-manual/configuration.adoc`: add `rick` and `specs_root` to the per-child key list
  (after `wip_in_development`/`wip_in_review`, around line 71), stating the defaults (`false`,
  `"docs/specs"`) and pointing at T-077 for what turning it on actually surfaces.
- `docs/user-manual/cli-reference.adoc`: add one bullet to `pickle doctor`'s checklist (around
  line 480, alongside the payload-version bullet) describing the new rick-interop line and its
  fail-open behaviour, so `--verbose`'s output is fully documented.

### 6. Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint`/`just docs-check` clean.
2. Docs updated and registered (task 5).
3. Write a summary: files touched, the field-shape risk carried forward (re-verified note above),
   anything deferred.
4. Suggest a Conventional Commit message, e.g. `feat(config): consume rick status --json as an
   artifact-state source (T-076)`.
5. Tidy WIP commits into atomic ones (root-path child).
6. Commit locally on the ticket branch. Publish only per the project's commit policy (do not push
   or open a merge request without user approval).

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-16 — patched by T-065's review impact sweep (step 8): T-065 landed first, so this
  ticket's own instruction — "whichever lands second should copy the first's conventions" —
  now has a concrete answer instead of a deferral. The convention to copy: a top-level envelope
  with an integer `schema` (removal/retype bumps it, addition never does), `pickle_version`,
  and snake_case keys; a consumer refuses an unrecognised `schema` rather than guessing.
  Nothing re-graded
- 2026-09-04 — TO DO → READY: plan complete
- 2026-09-04 — READY → IN DEVELOPMENT: picked up
- 2026-09-04 — plan amended inline: decision 7's `wireTicket.Key` corrected to `wireTicket.ID`
  (JSON field `id`, not `key`). The dev machine happens to have the real `rick` CLI installed
  (`ig/uk/rick` v0.10.0, the exact `github.com/ig-private/ai-sdlc/sdlc-cli` binary this ticket
  cites), so the wire shape was pinned empirically against a scratch `docs/specs/<ID>/*.md`
  fixture rather than left on trust from this ticket's own prose — the one thing refinement
  flagged as unverifiable turned out to be checkable after all, and the single field it got
  wrong is now fixed before it ever shipped. `path`/`kind`/`status`/`date` (the last omitempty,
  populated only from the artifact's own frontmatter) were all confirmed correct as written.
- 2026-09-04 — IN DEVELOPMENT → IN REVIEW: acceptance green
