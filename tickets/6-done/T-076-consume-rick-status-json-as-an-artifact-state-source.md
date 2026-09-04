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

### Checklist

- [x] Reviewer independence settled (step 0): **delegated** — the orchestrating reviewer authored
  this branch in this session, so steps 2–4a were run by a fresh, independent sub-agent, briefed
  adversarially with no memory of writing the code. Every delegated finding was re-verified by
  hand before being recorded below (two reproduced directly: the timeout/`Reason` bug with a
  standalone repro of `errors.Is(err, context.DeadlineExceeded)` against a real killed process,
  and the `specs_root`-discarded-when-`rick`-false case with a throwaway test against `Render()`).
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (steps 1, 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on `docs/user-manual/configuration.adoc` and `cli-reference.adoc` —
  every suggestion's quoted text verified against the file; none fabricated; 3 applied (all to text
  this ticket added), the rest correctly left alone as pre-existing, out-of-scope prose (step 4b)
- [x] Findings recorded with severity, class and disposition; disposition summary + cost line
  present (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6a)
- [x] Other references updated if needed; governing documents reconciled (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message + next-ticket suggestion presented for approval (step 9) — pending
  the rework round; nothing to publish until the blocking finding is fixed

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | A real `rick status --json` timeout does **not** produce the "distinct, human-readable Reason" decision 8 promises for it — `exec.CommandContext` kills the process with SIGKILL, so `cmd.Output()`'s error is a plain `*exec.ExitError` (`"signal: killed"`, `ExitCode() == -1`), which never satisfies `errors.Is(err, context.DeadlineExceeded)`. The timeout case in `execFailureReason` is dead code; a real timeout instead falls into the generic exit-code branch and reads as `"rick status --json exited -1"`, indistinguishable from an ordinary crash. | `internal/rickstatus/rickstatus.go:129-142`; reproduced standalone (`errors.Is` false, `errors.As(*exec.ExitError)` true, `ExitCode()==-1`) and end-to-end via `Query` with a 20ms `rickTimeout` against a 2s-sleeping stub, giving `Reason = "rick status --json exited -1"`; `go tool cover -func` confirms the `context.DeadlineExceeded` line in `execFailureReason` is never covered | Detect the timeout via `ctx.Err() != nil` (checked after the command fails, not via `errors.Is` on the returned error) rather than trying to match it out of the wrapped exec error; update `TestQueryTimesOut` to assert on `Reason` so this class of regression cannot ship silently again (folds in F2) |
| F2 | non-blocking | test-gap | noted | `TestQueryTimesOut` only asserts `Available`, never `Reason` — the one test written for this exact scenario could not have caught F1. | `internal/rickstatus/rickstatus_test.go`, `TestQueryTimesOut` | Addressed as part of F1's fix (see F1's suggestion) rather than separately — recorded here so the gap itself is on the record independent of whether F1's fix remembers it |
| F3 | non-blocking | docs-gap | fixed inline | Package doc omitted the "why `rickstatus`, not `internal/serve`" rationale Task 2 explicitly asked for. | `internal/rickstatus/rickstatus.go` (package doc, pre-fix) | Added a paragraph naming the two-consumers-neither-owns-the-other rationale |
| F4 | non-blocking | docs-gap | fixed inline | `configuration.adoc` listed `rick`/`specs_root` after `review_addendum`, the opposite of `Render()`'s actual emission order (which the ticket's own decision 2 specifies as directly after `wip_in_review`, before `review_addendum`). | `docs/user-manual/configuration.adoc` (pre-fix) vs `internal/config/config.go:503-511` | Reordered the doc bullets to match |
| F5 | non-blocking | docs-gap | fixed inline | `specs_root`'s doc bullet didn't say the field is inert until T-077 lands, as Task 5 asked ("pointing at T-077 for what turning it on actually surfaces"). | `docs/user-manual/configuration.adoc` (pre-fix); Task 5 text | Added a sentence stating no code reads it yet and naming T-077 as the first consumer |
| F6 | non-blocking | design | noted | Setting `specs_root` without `rick = true` is silently discarded the next time `pickle project add\|remove` fully re-renders `pickle.toml` (`Render()` gates both lines on `p.Rick`) — a real, reproduced edge case, but exactly what decision 2 specifies, not a deviation from it. | Reproduced directly: `Project{SpecsRoot: "custom/specs", Rick: false}` renders with no `specs_root` line at all | No change now — changing this means revisiting decision 2, not a review-time call. Now documented explicitly in `configuration.adoc` (see F5's fix) so it isn't a silent trap |
| F7 | non-blocking | docs-gap | fixed inline | The `rick` bullet linked `https://gitlab.com[rick/ai-sdlc]` — the bare GitLab homepage, not a page for the (private) project, providing no navigation value. | `docs/user-manual/configuration.adoc` (pre-fix), line ~74 | Dropped the link, kept the plain-text mention |
| F8 | non-blocking | other | noted | Acceptance test item 2 says "extend `TestAddProjectRejectsInvalidUTF8`'s table", but that test is not table-driven and never was — not something this branch made false (rules out `stale-xref`), and no confirmed design decision is at issue (rules out `plan-wrong`). | `internal/config/config_test.go:714-734` (single-case, not a table); the plan's acceptance-test wording | None needed — the implementation reasonably added a sibling test (`TestAddProjectRejectsInvalidUTF8SpecsRoot`) with equivalent coverage instead; harmless plan-wording slip |
| F9 | non-blocking | test-gap | fixed inline (partial) | Two structural branches were untested: `parse`'s empty-`id` skip, and `execFailureReason`'s `default` case. | `go tool cover -func`: `internal/rickstatus/wire.go:51` (`parse`) at 85.7%, `execFailureReason`'s default branch uncovered | Added `TestParseSkipsTicketsWithEmptyID` (now covered). The `execFailureReason` default-branch test was attempted but found to be **environment-dependent and misleading**: a non-executable stub earlier on `PATH` doesn't produce a permission error, it makes `PATH` resolution fall through to the next entry — on this reviewer's machine, straight to the real installed `rick` binary, which happens to answer with an unrelated schema mismatch that only accidentally satisfies the assertion. Discarded rather than shipped as a passing-but-wrong test; left as `noted` for the default branch specifically (defensive completeness, genuinely hard to trigger deterministically without an injectable command runner, not worth the redesign for one leaf-package branch) |

Disposition summary: 1 blocking (F1, → rework); 8 non-blocking — 4 fixed inline (F3, F4, F5, F7,
plus F9 partially), 4 noted (F2, F6, F8, and F9's remaining half).

cost: estimated M, actual M — the blocking finding is a small, well-isolated fix (one
function, one test); it does not change the estimate.

### Rework fix record — round 1 (commit 4b3a406)

Fixed F1: `execFailureReason` (`internal/rickstatus/rickstatus.go`) now detects a real timeout by
checking `ctx.Err() != nil` after the command fails — `cancel` is deferred until `Query` returns,
so this is non-nil here iff the deadline actually fired — checked ahead of the `*exec.ExitError`
case a timed-out process is also (a killed process's error never satisfied
`errors.Is(err, context.DeadlineExceeded)`, which is why the original branch was dead code).
`TestQueryTimesOut` now asserts on `Reason` (both that it names a timeout and that it does **not**
read as a generic `"exited N"`), closing F2 in the same commit — the gap that let F1 ship
undetected the first time. Full suite, `go vet`, `gofmt`, `just lint`/`docs-check`, and the bonus
real-`rick`-binary test (`TestQueryAgainstRealRickBinary`) all re-run clean; coverage on this
package rose from 90.5% to 95.2% (`execFailureReason` 83.3%, the remaining gap being F9's
noted-not-fixed default-branch case).

### Scoped re-review — round 1

Again delegated (step 0: the orchestrating reviewer authored the fix commit in this session) to a
fresh, independent sub-agent, scoped exactly to F1/F2 plus a fresh read of the fix diff for new
defects — not a re-audit of the whole feature. Re-verified by hand afterward (`go test
./internal/rickstatus/... -v -race`, `go build`/`go vet`/`gofmt -l .`, full `just test`,
`just docs-check`, `pickle board audit`, all clean).

**F1: confirmed closed.** The reviewer reproduced the underlying Go semantics independently (not
just re-running the shipped test): `exec.CommandContext` killing a timed-out process yields a
plain `*exec.ExitError` (`ExitCode() == -1`) that never satisfies
`errors.Is(err, context.DeadlineExceeded)`, while `ctx.Err()` correctly reports the deadline at
that point — confirming the fix's checked-`ctx.Err()` approach is the right fix, not merely one
that happens to pass. Case ordering in the new switch (`ctx.Err()` checked ahead of
`*exec.ExitError`) is correct given a timed-out process is also an `*exec.ExitError`.

**F2: confirmed closed, and confirmed non-vacuous.** The reviewer reverted just the
`execFailureReason` logic to the pre-fix check in a scratch copy, kept the new test, and showed it
fails exactly as F1 originally shipped (`Reason = "rick status --json exited -1"`) — proof the
test would have caught the original bug, not a tautology.

**New-defect sweep of the diff:** none found. The other `execFailureReason` branches, `wire.go`
(untouched by this commit), `Query`'s error-free contract (decision 8), and `doctor`'s
rick-never-warns contract (decision 9, also untouched by this commit) were all spot-checked and
intact.

**One informational observation, explicitly not raised as a finding:** `ctx.Err() != nil` is
checked after `cmd.Output()` returns rather than atomically with the kill, so a process that
exits with a genuine non-zero code in the exact same instant the deadline independently elapses
could theoretically be misreported as "timed out" rather than "exited N". This only affects
diagnostic wording (still `Available: false`, still fail-open), requires sub-millisecond
coincidence, and is inherent to polling a context flag rather than a defect in this fix — recorded
here for the record, not dispositioned, since a non-finding has nothing to disposition.

**Verdict: no blocking findings. Ticket proceeds to `tickets/6-done/`.** No new findings from this
round (F3–F9's round-1 dispositions stand unchanged; none were in this round's scope).

cost: estimated M, actual M — unchanged from round 1's assessment; the fix stayed exactly as
small as decision 8's fail-open design predicted it would.

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
- 2026-09-04 — IN REVIEW → REWORK: 1 blocking finding (F1): timeout does not produce a distinct Reason
- 2026-09-04 — REWORK → IN REVIEW: F1 fixed (commit 4b3a406); findings fixed
- 2026-09-04 — IN REVIEW → DONE: review clean; 8 non-blocking dispositioned (4 fixed inline, 4 noted), 1 blocking (F1) fixed in rework round 1, scoped re-review confirmed closed with no new findings
- 2026-09-04 — branch tidied at publish time (rules §0, root-path child: three atomic commits kept
  rather than squashed). This rewrote the SHAs, so the Review section's rework fix record now
  cites a commit that no longer resolves — anticipated by the rules' own fallback, and mapped here
  rather than left to reconstruction: `0743b1d` → `a2e4542` (feat), `138089f` → `e086a4a` (review
  fixups, also retyped from the invalid `docs+test:` to `chore:`), `4b3a406` → `8c85c18` (the F1
  fix the rework record describes). Verified that the tidy changed no code: the branch's own diff
  excluding `tickets/` is byte-identical before and after (896 lines either side). The only
  substantive change is the middle commit's type prefix; the branch is also now based on the
  later `main`, so its `tickets/` ancestry differs, which is why that path is excluded from the
  comparison rather than included in it.
- 2026-09-04 — merged to main (PR #84, 8147a63)
