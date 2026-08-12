---
id: T-093
title: reconcile merged tickets against the changelog's Unreleased section
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low-medium
cost: S-M
---

# T-093 — reconcile merged tickets against the changelog's Unreleased section

## Outcome

After this ships, preparing a release starts from a **mechanical reconciliation** instead of a
from-scratch reading of the log: every ticket merged since the last tag either has a
`CHANGELOG.md` entry naming it, or a recorded decision in its own file saying it deliberately has
none — and anything else is reported. Choosing the version number, tagging, and verifying release
artifacts stay exactly where they are today: a human, `RELEASING.md`, and CI.

## Description

**This ticket was filed much larger and has been cut down.** The original filing proposed a whole
release procedure for the skill — changelog sweep, SemVer proposal, approval-gated tag, artifact
verification, and a new optional `[release]` block in `pickle.toml` — generalised from a single
session that cut `v0.5.0` by hand. The filing session then challenged it, and three of those four
parts did not survive (reasons recorded below, so refinement does not re-add them). What remains
is the one part **pickle is uniquely positioned to do**, because it is the only thing in the room
that knows what merged: reconciling merged tickets against the changelog.

**The check.** For every ticket whose History records a merge dated after the last release tag,
assert that `CHANGELOG.md` accounts for it — either an entry naming it, or a recorded, confirmed
decision in the ticket itself that it deliberately gets no entry. Report anything else.

**The join-key problem was overstated at filing, and refinement measured it away.** The filing
claimed the ticket↔entry join "is not currently reliable in either direction", citing 28 changelog
bullets carrying a `(T-NNN)` reference against 20 carrying none. That count was taken with a
first-line-only grep and is wrong: bullets in this changelog are multi-line, and the reference
often sits on a continuation line. Counting whole bullets per release section:

| release | bullets | without a `T-NNN` reference |
|---|---|---|
| 0.5.0 | 5 | 0 |
| 0.4.0 | 4 | 0 |
| 0.3.0 | 12 | 0 |
| 0.2.2 | 1 | 0 |
| 0.2.1 | 1 | 1 |
| 0.2.0 | 5 | 0 |
| 0.1.0 | 6 | 6 (predates the ticket flow entirely) |

So from `v0.2.0` on — every release written under this flow — the convention holds for 21 of 22
bullets. **The join key is already a de-facto standard; it does not need inventing, and no new
frontmatter field or entry marker is required.**

**What refinement did find is a real false-positive source, with a clean fix.** Taking "tickets
shipped since the last tag" from commit subjects, the naive query over `v0.4.0..v0.5.0` yields
T-042, T-070, T-080, T-081, T-089, T-090, T-091 — but four of those never shipped a line of code
in that range. T-042, T-070 and T-091 appear **only** in `board:` bookkeeping commits (filed,
re-graded, or patched by an impact sweep), and the changelog rightly ignores them. Filtering to
child-project commits only — the Conventional Commit form with the ticket id in brackets at the
end of the subject, excluding the `board:` form — leaves T-080, T-081, T-089, T-090, and diffing
against the four ids the `[0.5.0]` section names (T-080, T-081, T-083, T-089) leaves exactly
**one** candidate: T-090. Which is the one true positive, and the one with a recorded decision
explaining itself.

That precision is only available because **T-084 gave bookkeeping commits their own convention**,
deliberately distinct from child-project Conventional Commits. This ticket is the first consumer
to depend on that distinction *mechanically* rather than as prose, which is worth stating plainly:
if bookkeeping commits stop following the `board:` form, this check silently mis-reports. It stays
advisory (a report a human reads) rather than a gate, and it lists what it excluded, so the
failure is visible rather than silent.

**The direction matters, and only one direction is checkable.** The check is *shipped-but-
unmentioned*, never *mentioned-but-unshipped*: the `[0.5.0]` section legitimately names T-083 while
T-083 shipped in an earlier release, because an entry may reference an older ticket for context
(T-081 folded T-083's check into the gate table). Flagging that would be noise.

**The remaining judgement is the exemption, and it does not need mechanising.** T-090 has no entry
by a recorded, confirmed decision in its own Implementation Plan (it folded into T-089's
not-yet-released entry, since the code it hardened had never shipped). With the candidate list at
roughly one item per release, a human reading the report can settle that in seconds; inventing a
`changelog: none` frontmatter field to automate a once-per-release judgement call would cost more
than it saves. Decision recorded in the plan.

**REFINEMENT GATE — ANSWERED 2026-08-12: build, small.** `release-please`,
`git-cliff`, `changesets` and `semantic-release` all generate changelogs and versions from history,
and none of them were considered at filing. Refinement must state why pickle should grow its own
rather than adopt one, or **drop this ticket**. The honest argument for building: those tools
reconcile against *commits*, and none of them know what a ticket is, so none can check "this
merged ticket has no entry, and no recorded decision saying it shouldn't." The honest argument
against: that is a narrow benefit for a project whose value proposition is ticket flow, not
release automation. Dropping is a legitimate outcome of refining this ticket.

**Dropped from the original filing, with reasons:**

- *A SemVer bump proposed from changelog categories* (`Added` → minor, `Fixed`/`Changed` → patch,
  incompatible → major, with a pre-1.0 exception read from the changelog header). **The rule is
  unsound, and the release that inspired it proves it.** T-081's entries sit under `Added` and
  `Changed`, so the rule says *minor* — but T-081 introduced a real migration hazard (a
  pre-existing ticket missing a now-required section makes `pickle upgrade`/`install` fail), which
  post-`1.0.0` is a **major** by any honest reading. `v0.5.0` was correct only because the pre-1.0
  exception masked the error. Category-counting cannot see breakingness; encoding it would produce
  a confident wrong answer. If a version proposal is ever wanted, it should surface evidence and
  refuse to name a number.
- *Release-artifact verification* (all archives, checksums, the manual, the Homebrew tap commit).
  If a release missing the user manual is invalid — and T-086/T-087 establish that it is — then
  `release.yml` should **fail** on it. That is a small CI ticket, and once it exists an agent-side
  re-check is dead weight. Belongs in CI, not in the skill.
- *A new optional `[release]` block in `pickle.toml`* (`changelog`, `tag_prefix`, `release_doc`).
  Generalised from N=1: it encodes pickle's own setup (Keep a Changelog, SemVer, tag-triggered CI)
  as though it were the shape of every child-project. Premature; add config when a second
  project's needs actually disagree.
- *Bundling a release procedure into the shipped skill payload.* The original filing rejected a
  standalone `release-flow` skill on install-plumbing cost — an argument about implementation
  convenience, not conceptual fit — and never asked whether a release procedure should ship to
  every installed project at all. With the scope cut to a reconciliation check, the question is
  moot for now.

Soft coupling: **T-092** (detect an unfinalized merge) was first noted here as this check's ground
truth. Refinement removed that: **decision 3 takes "what shipped" from commit subjects, not merge
History lines**, so T-092 neither feeds nor gates this ticket. What survives is a shared
motivation — both exist because bookkeeping drifts silently from what git actually records, and
both were found the same way, by measuring the repo instead of trusting the prose. T-092's own
note has been corrected to match.

Soft coupling: **T-065** (JSON read projection) would give this a structured source for "tickets
merged since the last tag" instead of parsing `## History` text across `6-done/` — though decision 3
reduces the need for it, since the shipped-set now comes from commit subjects rather than ticket
files. Nice-to-have either way.

Soft coupling: **T-086**/**T-087** (release-CI hardening, both done) are where the dropped
artifact-verification work belongs if anyone ever wants it.

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-093-changelog-reconcile`, created in the `pickle` child-project's repo (path `.`) before
any change. Slug shortened from the filename's, per `feat/T-081-gate-table`. Local WIP commits
fine; **no push and no MR without explicit user approval** (`child_publish_gated = true`); merging
is the human's. Root-path child, so tidy WIP into atomic commits by interactive rebase and **keep
history** on merge (rules §0), not squash. Bookkeeping stays on `main`.

### Prerequisite gate (hard)

None. This reads git and `CHANGELOG.md` only; it does not depend on T-092 (see decision 3 — the
shipped-set comes from commit subjects, not merge History lines).

### Confirmed design decisions (do not deviate without asking)

1. **Build, not buy** — settled at refinement; the reasoning is in the Description's gate
   paragraph. Do not reopen it by pulling in `git-cliff`/`release-please` as a dependency.
2. **Read-only report; never a gate.** The command reports and exits **0** even when it finds
   candidates. It is not wired into `board audit`, CI, or `ticket move`. Reason: a merged ticket
   with no entry is not an error — an entry may legitimately be written any time before the
   release, so a gate would fire constantly and be ignored.
3. **"What shipped" comes from commit subjects, not ticket History lines.** Take ids from
   `git log --format=%s <since>..HEAD`, keeping only child-project Conventional Commits (id in
   brackets at the end of the subject) and **excluding** the `board:` bookkeeping form (T-084).
   Measured: this is what takes the `v0.4.0..v0.5.0` candidate list from seven ids to one.
   History-line parsing is *not* used — it answers "was it merged", not "did it ship in this
   range".
4. **One direction only: shipped-but-unmentioned.** Never report mentioned-but-unshipped — the
   `[0.5.0]` section legitimately references T-083 for context though T-083 shipped earlier.
5. **No exemption mechanism.** Do not add a `changelog: none` frontmatter field or entry marker.
   The candidate list is ~1 per release and the judgement (e.g. T-090's recorded fold-in decision)
   is a human's. The report should *point at* each candidate ticket file so that judgement is one
   click away, and say plainly that a recorded decision may explain it.
6. **No `[release]` config block, no skill-payload prose.** Both were dropped when this ticket was
   cut down; the changelog path and the since-ref are flags with defaults, not config.
7. **The exclusion list is part of the output.** Because the check now depends mechanically on
   T-084's `board:` convention, print (or offer behind a flag) the commits it excluded, so a
   convention drift shows up as a visibly odd exclusion rather than a silent under-report.

### Tasks

1. **`internal/vcs/vcs.go`** — add an output-capturing helper alongside the existing `exitCode`
   (which only returns a status). It must reuse `withoutRepoEnv(os.Environ())` and the `-C root`
   pinning for the same reason `exitCode` does, and apply a timeout like `probeTimeout`. Two
   callers needed: `git describe --tags --abbrev=0` (default since-ref) and
   `git log --format=%s <since>..HEAD`.
2. **New `internal/changelog/` package** — pure, dependency-free (mirroring `internal/audit`'s
   shape) with the logic split from the I/O so it is table-testable:
   - parse ids from commit subjects, classifying each subject as child-project / `board:`
     bookkeeping / neither;
   - parse the ids referenced in a named `## [<section>]` block of `CHANGELOG.md` (default
     `[Unreleased]`), counting whole multi-line bullets — **not** first lines only, the mistake
     this ticket's own filing made;
   - diff, returning candidates plus the exclusions from decision 7.
3. **`internal/cli/`** — wire `pickle changelog check` with flags `--since <ref>` (default: last
   tag via task 1), `--changelog <path>` (default `CHANGELOG.md`), `--section <name>` (default
   `Unreleased`). Register it in the command table and `pickle help` alongside `board`.
4. **Tests** — `internal/changelog/*_test.go` table cases: a `board:`-only ticket is excluded; a
   child-project commit with a trailing `(T-NNN)` is included; a multi-line bullet whose reference
   sits on a continuation line counts as mentioned (regression for the filing's own miscount); a
   ticket both shipped and mentioned yields no candidate; the mentioned-but-unshipped case yields
   nothing (decision 4). Plus a CLI test for flag defaults and exit code 0 with candidates present
   (decision 2).

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then the **live regression that pins this ticket's whole design** — run it against
this repo's own real history, which is safe because the command is read-only (no install, no
writes, so the self-modify policy's throwaway-dir rule does not apply; it must stay read-only for
that to hold):

```
./pickle changelog check --since v0.4.0 --section 0.5.0
```

Expected: exactly **one** candidate, `T-090`, naming its ticket file under `tickets/6-done/`; and
T-042, T-070 and T-091 must **not** appear (they occur only in `board:` commits). Exit code **0**.
That is the measured ground truth from refinement — if the output differs, the classifier is wrong.

Then confirm the default path works on a clean tree:

```
./pickle changelog check
```

Expected: `[Unreleased]` is empty and no child-project commits exist since `v0.5.0`, so it reports
no candidates and exits 0.

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`** — a new `[#cmd-changelog-check]` section documenting
  the command, its three flags, and (importantly) its **advisory, exit-0 contract**, so nobody
  wires it into CI expecting a gate. State the `board:`-exclusion rule explicitly, since a reader
  whose project does not follow T-084's convention needs to know the check depends on it.
- **`docs/user-manual/cli-reference.adoc`** command table near line 40 — add the row.
- **`RELEASING.md`** — add the command as the first step of the release checklist, before
  retitling `[Unreleased]`; that is the moment it is meant to be run.
- **`CHANGELOG.md`** — one `### Added` entry under `[Unreleased]`.

### Finish (mandatory)

Interactive-rebase WIP into atomic commits — suggested: one for the `internal/vcs` helper, one for
`internal/changelog` + tests, one for the CLI wiring, one for docs. Suggested Conventional Commit
for the primary commit:

```
feat(changelog): report merged tickets missing a changelog entry (T-093)
```

Keep history on merge (rules §0, root-path child); do not squash. Await explicit approval before
pushing or opening the MR.

## Review

Reviewed 2026-08-12 on `feat/T-093-changelog-reconcile` (4 commits, `ed6b1f9..1c5bdc5`), ticket
read from `main` per the review protocol.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped: the configured reviewer is unavailable** (the
  `docs_readability` backend errored: `gemini-2.5-pro` is not supported by the logged-in
  provider). Sanctioned conscious skip (step 4b); the changed prose was read manually instead,
  which is where F2 came from
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

**Verification.** `just build`, `just test`, `just lint`, `just docs-check` all green.
All four plan tasks are present in the files they name: `vcs.Output` (`internal/vcs/vcs.go`,
reusing `withoutRepoEnv` + `probeTimeout` + `-C root` as required), the pure
`internal/changelog` package (no I/O, `internal/audit`'s shape), the CLI wiring with all three
flags registered in the command table and `pickle help`, and both test files. Every confirmed
decision holds: no changelog generator was pulled in (1); the command exits 0 with candidates
present and is wired into nothing (2, pinned by `TestChangelogCheckFlagDefaults`); the shipped
set comes from `git log --format=%s` with `board:` excluded (3); only shipped-but-unmentioned is
reported (4, pinned by `TestCheckMentionedButUnshippedYieldsNothing`); no exemption field was
added and the report points at each candidate's ticket file (5); no `[release]` block and no
skill-payload prose (6); exclusions are printed (7).

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | noted | The plan's live regression yields **two** candidates (`T-093`, `T-090`), not the one it specifies — but the classifier is correct and the expectation was unwritable at refinement time. `<since>..HEAD` on the feature branch necessarily includes T-093's own four commits, which did not exist when the plan named its expected output. | `./pickle changelog check --since v0.4.0 --section 0.5.0` → `T-093`, `T-090`, exit 0. Ground truth re-derived by hand: child-project subjects in `v0.4.0..v0.5.0` name T-080/T-081/T-089/T-090; `[0.5.0]` names T-080/T-081/T-083/T-089; so `T-090` is the only true candidate in that closed range, and the plan's three required absentees (T-042, T-070, T-091 — `board:`-only) are absent. | Nothing to fix in the classifier. The open-ended range that makes the discrepancy possible is F3. |
| F2 | non-blocking | fixed inline | The manual, the CHANGELOG entry and both doc comments claimed the check accounts for a shipped ticket *either* by an entry naming it *or* by "a recorded, confirmed decision inside the ticket's own file". It does not: decision 5 declined an exemption mechanism, so a ticket with a recorded decision is printed like every other candidate — T-090, the one true positive, is exactly that case. A reader could reasonably conclude a candidate means "no decision recorded either". | `docs/user-manual/cli-reference.adoc` `[#cmd-changelog-check]` ¶1; `CHANGELOG.md` `[Unreleased]` entry; `internal/changelog/changelog.go` package doc; `internal/cli/changelog.go` file comment. | Fixed in `5412023`: say plainly that mentions are all that is checked and the recorded-decision judgement is the reader's (which is why the report points at the ticket file), and note that genuine errors still exit non-zero so "always exits 0" is not read too literally. |
| F3 | non-blocking | new ticket (T-094) | No range end: the shipped set is always `<since>..HEAD`, so aiming `--section` at a past release is only meaningful in the seconds after its tag. This is what makes F1 possible, and it would mislead anyone auditing a historical section. | Same run as F1: `--since v0.4.0 --section 0.5.0` silently includes every commit made after `v0.5.0` was cut. | Add `--until <ref>` (default `HEAD`) so `<since>..<until>` is a closed question. |
| F4 | non-blocking | new ticket (T-094) | The exclusion list is unconditional and unbounded, so it dominates the report at exactly the moment `RELEASING.md` makes it step 1. Decision 7 required the exclusions to be *visible*, and explicitly allowed "behind a flag". | `--since v0.4.0 --section 0.5.0` prints 2 candidate lines and **36** exclusion lines; a thirty-ticket release would print well over a hundred. | Print a count plus the ids by default, full subjects behind `--show-excluded` — decision 7's drift-detection intent survives, the noise does not. |
| F5 | non-blocking | new ticket (T-094) | A squash-merge's rewritten subject is a silent false negative. `ClassifySubject` demands the id at the very end, but GitHub's squash button appends ` (#N)`: `feat(cli): add a thing (T-050) (#31)` classifies as `Neither` — not shipped, and not listed as an exclusion either, so the ticket vanishes with nothing to notice. Harmless in this repo (it merges rather than squashes), but the command ships to every installed project. | `internal/changelog/changelog.go` `trailingIDRE` = `\(([A-Z][A-Z0-9]*-\d+)\)\s*$`; no test covers a trailing `(#N)`. | Tolerate a trailing PR-number token, and/or report a `Neither` subject that contains a ticket id anywhere as an explicit "unclassified" line. |
| F6 | non-blocking | noted | Classification is looser than its own documentation: `ChildProject` requires only a trailing `(T-NNN)`, never the Conventional Commit prefix, while the manual describes the kept set as "a Conventional Commit for the target child-project's own code with a ticket id in brackets at the end". The looseness is the safe direction (it over-includes rather than under-includes) and every subject in this repo's history classifies correctly, so this is a wording-vs-code gap, not a defect. | `ClassifySubject` (`internal/changelog/changelog.go`) tests `board:` prefix, then `trailingIDRE`, and nothing else. | Leave the permissive behaviour; if it is ever tightened, tighten the prose with it. Recorded so a later reader does not "fix" the code to match the docs and start dropping commits. |
| F7 | non-blocking | noted | `vcs.Output` discards git's stderr, so a user error surfaces as a bare exit status. `--since bogus` reports `git log --format=%s bogus..HEAD: exit status 128` instead of git's own `fatal: ambiguous argument …`, which is the sentence that would tell the user what to type instead. | `./pickle changelog check --since bogus` → the message above, exit 1. | Attach `(*exec.ExitError).Stderr` to the wrapped error. Small, and it would improve `vcs`'s only other caller too. |
| F8 | non-blocking | fixed inline | `runChangelogCheck` re-derives the project root with a second `cfg.Root()` call in the print statement, though the resolved `root` is already in scope a few lines above. Pure idiom, this branch's own code, no behaviour change. | `internal/cli/changelog.go`, `printChangelogCheckReport(cfg.Root(), …)`. | Fixed in `5412023`: pass `root`. |
| F9 | non-blocking | noted | `Result.Mentioned` is computed and returned but never rendered; only tests read it. Not dead code (it is a reasonable API surface for a pure package, and it makes the `Check` result self-describing), but worth recording so it is not mistaken for an omission in the report. | `internal/changelog/changelog.go` `Result.Mentioned`; no reference in `printChangelogCheckReport`. | Keep. If the report ever grows a verbose mode, this is what it should print. |

**Disposition summary:** 0 blocking, 9 non-blocking — 2 fixed inline (F2, F8, in commit
`5412023` on the ticket branch), 3 to one new ticket batched by theme (F3, F4, F5 → **T-094**),
4 noted and closed (F1, F6, F7, F9). No finding was folded.

**Impact sweep (step 8).** `T-092` is the only other ticket whose text mentions T-093, and it is
in `6-done/`; its soft-coupling note ("neither blocks nor materially helps the other", corrected
during T-093's refinement) is still accurate against what shipped, so nothing needed patching.
No ticket in `1-to-do/` or `2-ready/` depends on T-093 or assumes anything this change
falsifies. `pickle board audit`: 93 tickets, 0 errors, 0 warnings (before T-094 was filed).

## History

- 2026-08-12 — created (TO DO). source: chat design discussion following the session that cut
  `v0.5.0` by hand (see `tickets/6-done/T-081-…md`'s own History and this repo's `CHANGELOG.md`);
  that session's exact steps were this ticket's starting spec. Graded `medium`/`medium`/`M-L`
  against the backlog at filing
- 2026-08-12 — **cut down and retitled** (was: *a release procedure for the ticket-flow skill:
  changelog completeness, SemVer proposal, approval-gated tag*) after the filing session
  challenged its own two new tickets. Three of the four proposed parts were dropped with reasons
  recorded in the Description — the SemVer heuristic (demonstrably wrong on T-081, the very change
  that inspired it), artifact verification (belongs in `release.yml`, not an agent), and the
  `[release]` config block (generalised from N=1) — leaving only the merged-ticket ↔ changelog
  reconciliation, the one part that needs to know what a ticket is. A build-vs-buy refinement gate
  was added, with dropping named an explicitly legitimate outcome. Regraded
  `medium`/`medium`/`M-L` → `low-medium`/`medium`/`S-M` on much smaller scope
- 2026-08-12 — refined. Two of the filing's claims were measured and both moved. (1) The
  ticket↔entry join key was called unreliable on the strength of a naive first-line grep; counting
  whole multi-line bullets, the `(T-NNN)` convention holds for 21 of 22 entries from `v0.2.0` on,
  so it needs no new frontmatter field or marker. (2) The real precision problem is `board:`
  bookkeeping commits polluting "what shipped" — filtering to child-project Conventional Commits
  (T-084's convention, of which this is the first mechanical consumer) takes the `v0.4.0..v0.5.0`
  candidate list from seven ids to one, T-090, the single true positive. Build-vs-buy gate
  answered **build, small**: release-please/git-cliff/changesets *generate* changelogs from
  commits, whereas this project writes prose entries by hand and wants them *verified*, and none
  of them can see a ticket or tell a `board:` commit from a feature commit. Complexity `medium` →
  `low-medium` now the design is concrete; impact/cost unchanged
- 2026-08-12 — TO DO → READY: plan complete; build-vs-buy gate answered (build, small)
- 2026-08-12 — READY → IN DEVELOPMENT: picked up
- 2026-08-12 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-12 — IN REVIEW → DONE: review: 0 blocking, 9 non-blocking (2 fixed inline, 3 -> T-094, 4 noted)
