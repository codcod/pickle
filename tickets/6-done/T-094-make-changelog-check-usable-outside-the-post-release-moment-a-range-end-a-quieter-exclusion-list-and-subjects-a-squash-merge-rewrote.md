---
id: T-094
title: make changelog check usable outside the post-release moment: a range end, a quieter exclusion list, and subjects a squash-merge rewrote
project: pickle
depends-on: []
spawned-by: [T-093]
impact: low-medium
complexity: low
cost: S
---

# T-094 — make changelog check usable outside the post-release moment: a range end, a quieter exclusion list, and subjects a squash-merge rewrote

## Outcome

After this ships, `pickle changelog check` is readable at the moment it is actually run: its
report is candidates first and exclusions on request, it can be aimed at a closed commit range
rather than always running to `HEAD`, and it stops silently missing tickets in projects whose
merge button rewrites the commit subject.

## Description

Three findings from T-093's review (F3, F4, F5), batched because they are one theme: the command
was designed for exactly one moment — standing on `main` just before cutting a release — and is
awkward or wrong anywhere else. None of them is a defect against T-093's confirmed decisions;
all three are the edges those decisions left.

**1. No range end (`--until`), so the shipped set always runs to `HEAD` (F3).** Checking a
*past* section is therefore only meaningful in the seconds after its tag. T-093's own acceptance
test is the proof: `changelog check --since v0.4.0 --section 0.5.0` was specified to yield
exactly one candidate (`T-090`), and it yields two — `T-090` plus `T-093` itself, because the
feature branch's own four commits sit in `v0.4.0..HEAD`. The classifier is right; the range is
just open-ended. A `--until <ref>` (default `HEAD`) makes `<since>..<until>` a closed question
and makes any historical section auditable.

**2. The exclusion list is unconditional and unbounded (F4).** Decision 7 requires the excluded
`board:` commits to be visible so a convention drift shows up rather than under-reporting
silently — but it explicitly permitted "or offer behind a flag". Printing every one, always,
inverts the report: on this repo `--since v0.4.0 --section 0.5.0` prints 2 candidates and 36
exclusion lines, and a release with thirty tickets will print well over a hundred. The signal
survives with a count plus the ids (`excluded 36 board: bookkeeping commit(s) covering T-080,
T-081, …`), with the full subjects behind `--show-excluded`. Keep decision 7's intent: the reader
must still be able to see *what* was excluded without re-running anything they cannot discover.

**3. A squash-merge's rewritten subject is invisible (F5).** `ClassifySubject` requires the
ticket id in brackets at the very end of the subject, which is exactly what T-084's convention
prescribes — but GitHub's squash button appends ` (#N)`, turning
`feat(cli): add a thing (T-050)` into `feat(cli): add a thing (T-050) (#31)`. That classifies as
`Neither`: not shipped, and not listed as an exclusion either, so the ticket vanishes from the
report with nothing to notice. This repo merges rather than squashes, so it does not bite here —
but `changelog check` ships to every installed project, and the squash button is the common case
elsewhere. Tolerating a trailing PR-number token (and/or reporting a `Neither` subject that
contains a ticket id anywhere as an "unclassified" line) closes the one silent-under-report path
the design otherwise defends against everywhere else.

Soft coupling: **T-093** (done) shipped the command; its Description records why each of the
three edges was left where it is, and its `## Review` carries the evidence for all three
findings. Nothing here reopens a T-093 confirmed decision: the check stays read-only and
advisory (decision 2), one-directional (decision 4), and free of any exemption mechanism
(decision 5).

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-094-changelog-check-ergonomics`, cut from `main` in the `pickle` child-project's repo
(path `.`) before any change. Local WIP commits encouraged; **no push and no MR without explicit
user approval** (`child_publish_gated = true`); merging is the human's. Root-path child, so tidy
WIP into atomic commits by interactive rebase and **keep history** on merge (rules §0), not
squash. Ticket/board bookkeeping stays on `main`.

### Prerequisite gate (hard)

None blocking — `depends-on:` is empty. One thing to confirm before starting, because the task
list quotes the tree as it is *after* both of T-093's merges: `git log --oneline -1 main`
must be at or after `052510d` (PR #32, the review's inline fixes). If it is not, the doc
paragraphs task 4 edits will not be the ones described here.

### Confirmed design decisions (do not deviate without asking)

1. **T-093's decisions all stand.** The command stays read-only and advisory, always exiting `0`
   on a finding (T-093 decision 2); one-directional — shipped-but-unmentioned only (decision 4);
   and free of any exemption mechanism — no `changelog: none` frontmatter, no entry marker
   (decision 5). Nothing here reopens them; this ticket only changes *which commits* the shipped
   set contains and *how* the report is printed.
2. **`--until <ref>` (default `HEAD`) closes the range.** The shipped set becomes
   `<since>..<until>`, and the report header states the range rather than only its start.
3. **`--since` resolves relative to `--until`, not to `HEAD`.** Default `--since` becomes
   `git describe --tags --abbrev=0 <until>^` (today: `git describe --tags --abbrev=0`, implicitly
   from `HEAD`). Rationale: `--until v0.5.0` alone must mean "audit release 0.5.0", and with a
   `HEAD`-relative default it would instead produce `v0.5.0..v0.5.0` — an empty range that
   reports "no candidates" and looks like a pass. The `^` is what makes a tag-shaped `--until`
   name the range *ending* at that tag rather than starting from it; with the default
   `--until HEAD` on an untagged HEAD the answer is unchanged from today. When git cannot
   describe (no earlier tag), keep today's error shape: `no --since given and no git tag found`,
   naming the ref it tried.
4. **The exclusion list summarises by default; `--show-excluded` prints subjects.** Default:
   one line — count, then the deduplicated ids sorted, then how to see more, e.g.
   `excluded 28 board: bookkeeping commit(s) covering T-080, T-081, T-089, T-090, T-091
   (--show-excluded for subjects)`. Bookkeeping commits whose subject parses no id are counted
   in a trailing `(+N with no ticket id)` clause rather than dropped — they are the loudest
   possible symptom of a convention drift, so they must not be the thing the summary hides.
   This is decision 7 of T-093 exercised through the escape hatch it explicitly offered
   ("or offer behind a flag"), **not** a reversal of it: what must never happen is an exclusion
   the reader cannot discover.
5. **A trailing PR/MR-number token is stripped before classification, not matched around.**
   `feat(cli): add a thing (T-050) (#31)` classifies as `ChildProject` with id `T-050`. Strip
   exactly one trailing `(#\d+)` or `(!\d+)` group (GitHub and GitLab), once, then apply today's
   `trailingIDRE` unchanged. Do **not** widen `trailingIDRE` to "an id anywhere": a subject like
   `Merge pull request #30 from codcod/feat/T-081-gate-table` contains `T-081`, and counting
   that as shipped would resurrect the false-positive class T-093's decision 3 was written to
   kill.
6. **A fourth classification, `Unclassified`, is the safety net — narrowly defined.** A subject
   is `Unclassified` when it is not `board:`, does not classify as `ChildProject` even after the
   strip in decision 5, **and contains a parenthesised `(T-NNN)` somewhere other than the end**
   — e.g. `Revert "feat(cli): add a thing (T-050)"`. These print as their own short list, so a
   subject that mentions a ticket is never silently neither-shipped-nor-excluded. The
   parenthesised requirement is what keeps ordinary merge commits (whose branch names carry a
   bare `T-NNN`) out of the list; without it this repo alone would add four noise lines to every
   release-range run, and the noise is the thing this ticket exists to remove.
7. **No new package, no config, no gate.** Everything lands in `internal/changelog` and
   `internal/cli/changelog.go`. The flags stay flags (T-093 decision 6: no `[release]` block),
   and nothing is wired into `board audit` or CI.

### Tasks

#### Task 1 — classifier: PR-token tolerance and the `Unclassified` kind

`internal/changelog/changelog.go`:

- Add a `prTokenRE` alongside the existing package-level regexps, matching `\s*\([#!]\d+\)$`,
  and strip one match from the trimmed
  subject in `ClassifySubject` **after** the `board:` test and **before** `trailingIDRE`
  (decision 5). A `board:` subject is never stripped — bookkeeping commits are not merged
  through a PR button in this flow, and stripping there would only add a way to misparse.
- Add `Unclassified` to the `CommitKind` block (after `Bookkeeping`; `Neither` must keep its
  `iota` zero value so an unclassifiable subject stays the harmless default). Recognise it with
  a `parenIDRE` — `\(([A-Z][A-Z0-9]*-\d+)\)`, the parenthesised sibling of the existing `idRE`
  — searched against the post-strip subject, per decision 6, returning the first id it finds.
- Document each kind's *purpose* in its comment, as the existing three already are — the file's
  doc comments are load-bearing here, since the whole check rests on which subject means what.

#### Task 2 — `Check`: closed range in, unclassified out

Still `internal/changelog/changelog.go`. `Check`'s signature does not change (it takes subjects
already — the range is the CLI's problem, task 3), but:

- Add `Unclassified []Exclusion` to `Result`, populated in commit-log order from the new kind,
  reusing `Exclusion` (its `Subject` + `ID` shape fits exactly; do not mint a second struct).
- Leave `Shipped`, `Mentioned`, `Candidates` and `Excluded` semantics untouched.

#### Task 3 — CLI: `--until`, `--show-excluded`, and a quieter report

`internal/cli/changelog.go`:

- Register `--until <ref>` (default `HEAD`) and `--show-excluded` (bool, default false); update
  `changelogCheckUsage` to list all five flags.
- Resolve the default `--since` per decision 3: `vcs.Output(root, "describe", "--tags",
  "--abbrev=0", *until+"^")`.
- `commitSubjects(root, since, until)` — `git log --format=%s <since>..<until>`.
- `printChangelogCheckReport`: header states the range
  (`changelog check: v0.4.0..v0.5.0, against CHANGELOG.md's "0.5.0" section`); candidates print
  exactly as today (they are the payload); exclusions collapse to decision 4's one-line summary
  unless `--show-excluded`, in which case today's full subject list prints after it; the
  unclassified list, when non-empty, prints last with a one-line explanation of what the reader
  should do ("these mention a ticket but match neither convention — check whether they shipped").
  Sort and deduplicate the summary's ids.

#### Task 4 — tests

- `internal/changelog/changelog_test.go` — extend `TestClassifySubject`'s table: a trailing
  `(#31)` after the id classifies as `ChildProject`; a trailing `(!31)`; a bare
  `Merge pull request #30 from codcod/feat/T-081-gate-table` stays `Neither` (the regression
  guarding decision 5's "do not widen"); `Revert "feat(x): thing (T-050)"` is `Unclassified`
  with id `T-050`. Plus a `Check` case asserting an unclassified subject lands in
  `Result.Unclassified` and in neither `Shipped` nor `Excluded`.
- `internal/cli/changelog_test.go` — a `--until` case (a commit after the `--until` ref is not
  reported), a case pinning that `--until <tag>` alone resolves `--since` to the *previous* tag
  (decision 3, the footgun this closes), and one asserting the default report prints the
  one-line exclusion summary while `--show-excluded` prints the subjects.

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then the live regression — read-only, so it runs against this repo's own history
(no install, no writes; the self-modify policy's throwaway-dir rule does not apply, and the
command must stay read-only for that to hold). **This is T-093's own acceptance test, finally
runnable as it was written:**

```
./pickle changelog check --since v0.4.0 --until v0.5.0 --section 0.5.0
```

Expected: exactly **one** candidate, `T-090`, naming its file under `tickets/6-done/`; `T-093`
**absent** (it is outside the closed range — the whole point of F3); `T-042`, `T-070`, `T-091`
absent (`board:`-only); exit **0**. The exclusion summary reads `excluded 28 board: bookkeeping
commit(s) covering T-080, T-081, T-089, T-090, T-091` with no trailing `+N with no ticket id`
clause, and no unclassified list is printed (measured 2026-08-12: 28 `board:` subjects in that
range, all with a parsable id; 4 merge commits, none carrying a parenthesised id). Then:

```
./pickle changelog check --since v0.4.0 --until v0.5.0 --section 0.5.0 --show-excluded | wc -l
```

Expected: 28 more lines than the same run without the flag.

```
./pickle changelog check
```

Expected: the default path still works from `main` — header shows `<last tag>..HEAD`, exit 0.

Finally, the squash-subject case, which this repo's own history cannot demonstrate (it merges
rather than squashes) — assert it in the unit tests per task 4, and sanity-check by hand:

```
go run . changelog check --since v0.4.0 --until v0.5.0 --section 0.5.0   # unchanged output
```

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`**, `[#cmd-changelog-check]`: document `--until` (with
  decision 3's `--since`-relative-to-`--until` rule stated explicitly, since it is the one
  surprising behaviour), `--show-excluded`, the summarised exclusion line, the unclassified
  list, and the tolerated trailing `(#N)`/`(!N)`. Keep the existing "mentions only", "never a
  gate" and "one direction only" paragraphs — they remain true and are the reason the command is
  safe to read quickly.
- **`docs/user-manual/cli-reference.adoc`** command table (~line 43): update the row's flag list.
- **`internal/cli/cli.go`** help text: same flag list, same one-line description.
- **`RELEASING.md`**: unchanged unless the wording no longer matches the report's first line —
  re-read it against the new header and fix if it drifted.
- **`CHANGELOG.md`**: one `### Changed` entry under `[Unreleased]` (the command is
  unreleased, so this modifies the existing shape rather than adding a feature; if the T-093
  entry is still in `[Unreleased]`, prefer amending that bullet over adding a second one for a
  command the world has never seen).

### Finish (mandatory)

Interactive-rebase WIP into atomic commits — suggested: one for the classifier (tasks 1–2 +
their tests), one for the CLI report and flags (tasks 3–4's CLI half), one for docs. Suggested
Conventional Commit for the primary commit:

```
feat(changelog): close the range, quiet the exclusions, and see squashed subjects (T-094)
```

Keep history on merge (rules §0, root-path child); do not squash. Await explicit approval before
pushing or opening the MR. **Push `origin main` first if the local base is ahead** — and verify
`git diff --name-only origin/main...HEAD` carries no `tickets/` path before pushing the branch.

## Review

Reviewed 2026-08-12 on `feat/T-094-changelog-check-ergonomics` (three atomic commits
`74616b0`, `7489bf1`, `03d1a14`, cut from `main` at `0dd2f88`; ticket read from `main`).
**Verdict: no blocking findings — IN REVIEW → DONE.**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped**: the host's `docs_readability` reviewer is
      unavailable (`pi --provider github-copilot --model gemini-2.5-pro` →
      `model_not_supported`, exit 1). A sanctioned conscious skip (step 4b); the changed
      `.adoc`/`.md` prose was read by hand instead, which produced F3 and F6
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages & MR attributes presented for approval (step 9)

### Implementation audit — all tasks and criteria **met**

| item | result | evidence |
|---|---|---|
| Task 1 — `prTokenRE`, strip after `board:` / before `trailingIDRE`, `Unclassified` kind, `parenIDRE`, purpose-documented comments | met | `internal/changelog/changelog.go`: `prTokenRE = \s*\([#!]\d+\)$` and `parenIDRE = \(([A-Z][A-Z0-9]*-\d+)\)`; `ClassifySubject` strips *after* the `board:` branch returns; `Neither` keeps `iota` zero, `Unclassified` added after `Bookkeeping` |
| Task 2 — `Result.Unclassified []Exclusion` in commit-log order, `Exclusion` reused, other semantics untouched | met | `Result` gains one field; `Check`'s switch gains one `case`; `Shipped`/`Mentioned`/`Candidates`/`Excluded` unchanged |
| Task 3 — `--until`, `--show-excluded`, usage line, `<until>^` default `--since`, `commitSubjects(root, since, until)`, range header, summarised exclusions, unclassified list last | met | `internal/cli/changelog.go`; header renders `changelog check: v0.4.0..v0.5.0, against CHANGELOG.md's "0.5.0" section`; ids sorted + deduplicated in `printExclusions` |
| Task 4 — classifier + `Check` + CLI tests | met | 4 new `TestClassifySubject` rows (incl. the `Merge pull request #30 from codcod/feat/T-081-gate-table` → `Neither` regression guarding "do not widen"), `TestCheckUnclassifiedSubjectIsNeitherShippedNorExcluded`, and 3 new CLI tests (`…UntilExcludesLaterCommits`, `…SinceDefaultsRelativeToUntil`, `…ShowExcludedPrintsSubjects`) |
| Acceptance: `just build && just test && just lint && just docs-check` | met | all four green (re-run after this review's inline fixes) |
| Acceptance: `changelog check --since v0.4.0 --until v0.5.0 --section 0.5.0` | met | **exactly** as specified: 1 candidate `T-090` naming its `6-done/` file; `T-093` absent; `T-042`/`T-070`/`T-091` absent; exit `0`; summary reads `excluded 28 board: bookkeeping commit(s) covering T-080, T-081, T-089, T-090, T-091` with no `+N` clause; no unclassified list. **T-093's own acceptance test, now runnable as written** |
| Acceptance: `--show-excluded \| wc -l` = +28 lines | met | 4 → 32 |
| Acceptance: default `changelog check` from `main` | met | `changelog check: v0.5.0..HEAD, against CHANGELOG.md's "Unreleased" section`, exit `0` |
| Decisions 1–7 honoured | met | read-only/advisory (always exit `0`), one-directional, no exemption mechanism, no new package/config/gate, strip-not-widen, parenthesised-only `Unclassified`; `RELEASING.md` re-read and correctly left unchanged (its "since the last tag … against `[Unreleased]`" still describes the default invocation) |

`internal/cli/changelog.go` is the only consumer of the `changelog` package — no caller↔callee
drift elsewhere. Docs coverage is complete and in the right place
(`[#cmd-changelog-check]` + the command table row + `cli.go` help).

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | new ticket (T-095) | The default exclusion summary names only the **first** id of a multi-id `board:` commit, so ids the rules §0 grammar explicitly sanctions are invisible in the only default view. `boardIDRE` (pre-existing, T-093) captures one id; T-094 is what made the summary the sole default surface. | Scratch repo, one commit `board: T-010, T-011 re-aimed after review` → `excluded 1 board: bookkeeping commit(s) covering T-010`; `T-011` shows only under `--show-excluded`. Count is always exact, so the drift alarm survives — the id inventory does not. | Build the summary's id set with `FindAllStringSubmatch` over the subject after `board:` (or `Exclusion.IDs []string`); test a two-id bookkeeping subject. |
| F2 | non-blocking | new ticket (T-095) | Decision 3's `<until>^` changes the **bare** invocation on a tagged `HEAD`, and turns a previously exit-`0` invocation into exit `1` when `HEAD` has no parent. Decision 3 scoped its "unchanged from today" guarantee to an untagged `HEAD`, so this is an accepted edge, not a deviation — but nothing documents or tests it. | Scratch repo: `HEAD` tagged `v0.2.0` over `v0.1.0` → bare `changelog check` reports `v0.1.0..HEAD` (the whole previous release) against `[Unreleased]`; pre-T-094 it was the empty `v0.2.0..HEAD` → `no candidates`. Single tagged root commit → `pickle: changelog check: no --since given and no git tag found reachable from HEAD^: … exit status 128`, exit `1`. | Fall back to `describe --tags --abbrev=0 <until>` when `<until>^` won't resolve; document what a tagged `--until` means; pin both with CLI tests. |
| F3 | non-blocking | fixed inline | The short-form help, `cli-reference.adoc`'s opening paragraph and the `CHANGELOG.md` bullet all said the default `--since` is "the last git tag **reachable from** `--until`" — which mispredicts the exact case the flag exists for (`--until v0.5.0` would resolve to `v0.5.0`) and contradicted the precise `<until>^` paragraph three paragraphs later on the same page. | `internal/cli/changelog.go` flag help, `internal/cli/cli.go` help block, `docs/user-manual/cli-reference.adoc` §`pickle changelog check`, `CHANGELOG.md` `[Unreleased]`. | Say "the last git tag **before** `--until`", with `git describe --tags --abbrev=0 <until>^` spelled out once in the adoc. **Done** — commit `5ff933d`. |
| F4 | non-blocking | fixed inline | `Check`'s doc comment still described its input as `git log --format=%s <since>..HEAD` — made false by this branch's own closed range. | `internal/changelog/changelog.go`. | `<since>..<until>`. **Done** — `5ff933d`. |
| F5 | non-blocking | fixed inline | `Exclusion`'s type comment ("records one bookkeeping commit …") and its `ID` comment (`""` when no id parsed) became false for half the type's uses once `Result.Unclassified` reused it per task 2 — an `Unclassified` entry is not a bookkeeping commit and its `ID` is never `""`. | `internal/changelog/changelog.go`. | Widen the comment to both lists and say why one struct, not two. **Done** — `5ff933d`. |
| F6 | non-blocking | fixed inline | Missing space before an em dash in branch-authored prose: `This matters the moment \`--until\` is given alone—`. | `docs/user-manual/cli-reference.adoc`. | Add the space. **Done** — `5ff933d`. |

**Disposition summary:** 6 findings, 0 blocking — 4 `fixed inline` (F3, F4, F5, F6; all in
commit `5ff933d`), 2 `new ticket` batched into **T-095** (F1, F2), 0 `folded`, 0 `noted`.

### Quality, consistency and impact notes (no findings)

- The strip-not-widen choice is load-bearing and correctly guarded: `… (T-050) (#31) (#32)`
  strips one token, fails `trailingIDRE`, and surfaces as `Unclassified` rather than being
  silently lost — the safety net catching the case decision 5 deliberately did not handle.
- `parenIDRE`'s parentheses anchor does its job on this repo's real history: the 4 merge commits
  in `v0.4.0..v0.5.0` carry bare branch-name ids and produce **zero** unclassified lines.
- `printExclusions`' three-way `switch` covers all-no-id, none-no-id and mixed; the mixed branch
  fires on this repo's `main` today (`+1 with no ticket id`) against
  `board: file T-092, T-093 (skill gaps found while releasing v0.5.0)` — which genuinely does
  violate rules §0's id-leads-the-subject grammar, so the alarm is a true positive, not noise.
- Step 8 impact sweep: no ticket in `1-to-do/` or `2-ready/` references T-093, T-094 or the
  changelog at all, so nothing needed patching.

## History

- 2026-08-12 — created (TO DO). source: T-093's review, findings F3/F4/F5, batched by theme per
  the rules §5 (see `tickets/6-done/T-093-…md`'s `## Review`). Graded `low-medium`/`low`/`S`
  against the backlog: all three are small, local changes to one young command, and the
  usability half (F4) is felt at every release
- 2026-08-12 — refined. Description re-verified against `main` at `052510d` (both T-093 PRs
  merged); every path and symbol it names still exists. Three design questions were settled and
  written into the plan as confirmed decisions: (a) the default `--since` resolves relative to
  `--until` (`describe --tags --abbrev=0 <until>^`), because a `HEAD`-relative default would turn
  `--until v0.5.0` into the empty range `v0.5.0..v0.5.0` and report a false pass; (b) F5 is
  closed by *stripping* a trailing `(#N)`/`(!N)` rather than by widening the id match, since a
  merge subject like `Merge pull request #30 from codcod/feat/T-081-gate-table` carries a bare
  id that must never count as shipped; (c) an `Unclassified` list is added as the safety net,
  but restricted to subjects carrying a *parenthesised* id, which keeps ordinary merge commits
  out of it — an unrestricted rule would have added four noise lines per release range to a
  ticket whose whole point is less noise. Not split: the three findings are one command, one
  package, and roughly twenty lines each — none would be picked up alone. Grades unchanged
  (`low-medium`/`low`/`S`)
- 2026-08-12 — TO DO → READY: plan complete; three design questions settled (range-relative --since, strip-not-widen, parenthesised-only Unclassified)
- 2026-08-12 — READY → IN DEVELOPMENT: picked up
- 2026-08-12 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-12 — reviewed. Acceptance test re-run verbatim and green, including T-093's original
  one-candidate assertion that only became runnable here; `just build`/`test`/`lint`/`docs-check`
  all green. 6 findings, **0 blocking**: 4 fixed inline (stale/imprecise prose this branch
  authored or made false — commit `5ff933d`) and 2 batched into **T-095** (the exclusion
  summary's first-id-only parse of a multi-id `board:` commit; the `<until>^` default on a tagged
  or parentless `HEAD`). Docs-readability pass consciously skipped — the host reviewer is
  unavailable (`model_not_supported`)
- 2026-08-12 — IN REVIEW → DONE: review verdict: no blocking findings; 4 fixed inline, F1/F2 -> T-095
