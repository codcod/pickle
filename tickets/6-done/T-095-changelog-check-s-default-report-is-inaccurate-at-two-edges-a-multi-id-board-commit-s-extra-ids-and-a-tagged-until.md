---
id: T-095
title: changelog check's default report is inaccurate at two edges: a multi-id board: commit's extra ids and a tagged --until
project: pickle
depends-on: []
spawned-by: [T-094]
impact: low
complexity: low
cost: S
---

# T-095 — changelog check's default report is inaccurate at two edges: a multi-id board: commit's extra ids and a tagged --until

## Outcome

After this ships, `pickle changelog check`'s default one-line exclusion summary names *every*
ticket id the excluded bookkeeping commits cover, not just the first id of each; and the bare
`changelog check` invocation still answers sensibly when `HEAD` is itself tagged or has no
parent, instead of reporting the previous release against `[Unreleased]` or exiting `1`.

## Description

Two non-blocking findings from T-094's review (F1, F2), batched because they are one theme: the
two *defaults* T-094 introduced are each inaccurate at an edge. Neither is a deviation from a
T-094 confirmed decision — both are edges those decisions left — and neither affects the golden
path (`changelog check` run on an untagged `main` before cutting a release, which is what
`RELEASING.md` prescribes).

**1. The exclusion summary drops a multi-id `board:` commit's extra ids (F1).** T-094 decision 4
made the one-line summary the *only* default view of the excluded set, and specified it as "the
deduplicated ids sorted". But the id set is built from `Exclusion.ID`, and
`boardIDRE` (`^board:\s*([A-Z][A-Z0-9]*-\d+)\b`, pre-existing from T-093) captures exactly one
id — while the rules §0 grammar explicitly sanctions the multi-id form
(`board: T-057, T-072 note the shared origin-base invariant`). Measured in a scratch repo: a
single commit `board: T-010, T-011 re-aimed after review` yields
`excluded 1 board: bookkeeping commit(s) covering T-010` — `T-011` appears only under
`--show-excluded`. Before T-094 every subject printed, so both ids were visible; the summary is
where the second id became invisible. The count is always exact, so this degrades the id
inventory, not the drift alarm. Likely fix: build the summary's id set with
`FindAllStringSubmatch` over the subject's remainder after `board:` (or give `Exclusion` an
`IDs []string`), and pin it with a `Check`/report test carrying a two-id bookkeeping subject.

**2. The `<until>^` default `--since` misbehaves on a tagged or parentless `HEAD` (F2).** T-094
decision 3 resolves the default `--since` as `git describe --tags --abbrev=0 <until>^`, and
scoped its "unchanged from today" guarantee to *an untagged `HEAD`*. On a tagged `HEAD` the
behaviour therefore did change, in two ways nothing documents or tests:

- `HEAD` tagged `v0.2.0` with an earlier `v0.1.0`: bare `changelog check` now reports
  `v0.1.0..HEAD` — i.e. the whole *previous* release — against `[Unreleased]`. Pre-T-094 it
  resolved `v0.2.0..HEAD`, an empty range printing `no candidates`. Neither answer is useful, but
  the new one is loud, and it lands exactly one command after `git tag` in `RELEASING.md`.
- `HEAD` is the root commit (a fresh or shallow clone): `HEAD^` does not resolve, so a previously
  exit-`0` invocation now exits `1` with
  `no --since given and no git tag found reachable from HEAD^: … exit status 128`. The message
  blames a missing tag when the real cause is a missing parent.

Likely fix: fall back to `describe --tags --abbrev=0 <until>` when `<until>^` cannot be resolved
or described, and say in `docs/user-manual/cli-reference.adoc` what a tagged `--until`/`HEAD`
means; pin both with `internal/cli/changelog_test.go` cases.

Soft coupling: **T-093** (done) shipped the command and `boardIDRE`; **T-094** (done) shipped
both defaults, and its `## Review` carries the measured evidence for F1 and F2. Nothing here
reopens a T-093 or T-094 confirmed decision: the check stays read-only and advisory, one
directional, and free of any exemption mechanism.

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-095-changelog-summary-and-range-edges`, cut from `main` in the `pickle` child-project's
repo (path `.`) before any change. Local WIP commits encouraged; **no push and no MR without
explicit user approval** (`child_publish_gated = true`); merging is the human's. Root-path child,
so tidy WIP into atomic commits by interactive rebase and **keep history** on merge (rules §0),
not squash. Ticket/board bookkeeping stays on `main`.

### Prerequisite gate (hard)

None blocking — `depends-on:` is empty. One thing to confirm before starting, because every
measured number below was taken from the tree as it is *after* T-094's merge: `git log --oneline
-1 main` must be at or after `58dcc0d` (T-094's merge `876e63d` plus its bookkeeping). If it is
not, `printExclusions` and the `<until>^` resolution this ticket edits do not yet exist.

### Confirmed design decisions (do not deviate without asking)

1. **T-093's and T-094's decisions all stand.** The command stays read-only and advisory, always
   exiting `0` on a finding; one-directional (shipped-but-unmentioned only); free of any exemption
   mechanism; no new package, no config, no gate. This ticket changes only *which ids the summary
   names* and *how the default `--since` is resolved at two edges*.
2. **The summary's id set is every id the subject mentions, not the leading id-list.** Scan the
   whole `board:` subject with the existing `idRE` (`\b[A-Z][A-Z0-9]*-\d+\b`) rather than
   extending `boardIDRE`'s anchored leading-list match. Rationale, measured across this repo's
   entire history: of the **9** multi-id `board:` subjects, only **one** keeps its extra ids in
   the rules §0 leading comma-list — the rest carry them in the verb phrase
   (`board: T-080 review findings recorded, moved to rework; T-042, T-070, T-081 patched by the
   sweep`; `board: T-089 reviewed and done, T-090 filed, T-070 re-graded`;
   `board: T-092 and T-093 refined to READY`, which uses "and" rather than a comma). A
   grammar-strict parser would therefore still miss most real cases. It also makes both halves of
   the report recognise ids by the **same** rule, since `sectionIDs` already scans the changelog
   body with `idRE`. False-positive risk (`UTF-8` matches `idRE`) is theoretical: zero non-`T-`
   id-shaped tokens exist in the repo's entire commit-subject history.
3. **The summary's verb becomes `mentioning`, not `covering`.** "Covering" asserts the strict
   claim decision 2 declines to make; "mentioning" is exactly what a permissive scan supports, and
   a superset is the safe direction for a reader asking "is anything I care about hiding in here?"
4. **Accepted consequence: the `+N with no ticket id` clause narrows, and this repo's current
   instance of it disappears.** Today `changelog check` on `main` prints
   `… covering T-089, T-092, T-093, T-094 (+1 with no ticket id…)`, and that `+1` is
   `board: file T-092, T-093 (skill gaps found while releasing v0.5.0)` — a subject that genuinely
   violates rules §0's *id-leads-the-subject* grammar. Under decision 2 it parses two ids, so the
   clause stops firing. This is deliberate: the clause's stated job (T-094 decision 4) is to
   surface a bookkeeping commit with **no ticket id at all**, which is the rarer and more serious
   drift; detecting "id present but not in the leading position" was never what it claimed to do,
   and conflating the two made it a false alarm. Do not add a second alarm for word order — if
   that drift is worth catching it is a lint on commit subjects, not a line in this report.
5. **`Exclusion.ID string` is replaced by `IDs []string`** — not carried alongside it. There is
   exactly one consumer (`internal/cli/changelog.go`), and two fields answering "which id?" is the
   kind of ambiguity a reader has to test to resolve. Dedupe within one subject, preserve
   source order (the report sorts the union itself).
6. **`ClassifySubject`'s `(CommitKind, string)` signature does not change.** It is the classifier,
   it is table-tested across a dozen rows, and the *primary* id it returns is still what the
   `ChildProject` path needs. Building the id **list** is `Check`'s job, via one new unexported
   helper. Say so in `ClassifySubject`'s doc comment, so the now-unread-for-`Bookkeeping` second
   return does not read as dead.
7. **The `<until>^` resolution gains one fallback, not a new range mode.** Try
   `describe --tags --abbrev=0 <until>^` (T-094 decision 3, unchanged and still tried first — it
   is what makes a tag-shaped `--until` name the range *ending* there); if that fails, try
   `describe --tags --abbrev=0 <until>`; only if both fail, error, naming `<until>` rather than
   `<until>^` so the message stops blaming a missing tag for a missing parent. Verified: on a
   tagged root commit the fallback yields `v0.1.0..HEAD` and exit `0`, restoring the pre-T-094
   answer. **Explicitly rejected:** falling back to *all history up to `<until>`* (no `..`). It is
   arguably more truthful and would make a tag-less repo work, but it converts a T-093-documented
   error into a success and is a separate decision — file it if anyone wants it.
8. **A tagged `--until` earns an advisory `note:` line, printed immediately after the header.**
   When `git describe --tags --exact-match <until>` succeeds (verified: it exits non-zero when
   `<until>` is not exactly a tag, so it is a clean one-call test) **and** `--section` is still
   `Unreleased`, print one line to stdout, e.g.
   `note: v0.5.0 is a tag — this range is release 0.5.0, not [Unreleased]; try --section 0.5.0`.
   After the header, because it reframes everything below it. It is part of the report, so
   stdout, and it **never** changes the exit code. Hoist `"Unreleased"` into a
   `defaultChangelogSection` const used both by the flag default and this comparison, so the two
   cannot drift. An explicit `--section Unreleased` is indistinguishable from the default and will
   also get the note — acceptable, because the note is still true.

### Tasks

#### Task 1 — `changelog`: an id list per excluded subject

`internal/changelog/changelog.go`:

- Replace `Exclusion.ID string` with `IDs []string` (decision 5) and rewrite its field comment:
  every id the subject mentions, deduplicated, in source order; empty only when the subject names
  no id at all.
- Add an unexported `subjectIDs(subject string) []string` — `idRE.FindAllString(subject, -1)`,
  deduplicated, order preserved. Document *why* it is permissive (decision 2's measurement) and
  that it is the same rule `sectionIDs` uses, so the two stay legibly deliberate rather than
  coincidental.
- In `Check`, populate `IDs: subjectIDs(subj)` for both the `Bookkeeping` and the `Unclassified`
  cases. Leave `Shipped`, `Mentioned`, `Candidates` semantics and the `ChildProject` path
  untouched — `Shipped` still comes from `ClassifySubject`'s single returned id.
- Extend `ClassifySubject`'s doc comment per decision 6.

#### Task 2 — CLI: union the ids, and say `mentioning`

`internal/cli/changelog.go`, `printExclusions`:

- Build the id set by iterating `ex.IDs`; count an exclusion into `noID` when `len(ex.IDs) == 0`.
- Change `covering` → `mentioning` in the two branches that name ids (decision 3). Keep the
  three-way `switch` (all-no-id / none-no-id / mixed) and the exact `--show-excluded` hint text.
- Update the function's doc comment: the `+N` clause's narrowed meaning per decision 4, and the
  permissive id scan per decision 2.

#### Task 3 — CLI: the `<until>^` fallback

Still `internal/cli/changelog.go`. Extract the default-`--since` resolution out of
`runChangelogCheck` into `defaultSince(root, until string) (string, error)` — it is now two git
calls and a two-stage failure, which is more than belongs inline in a flag block:

- `describe --tags --abbrev=0 <until>^`, then `describe --tags --abbrev=0 <until>`, then an error
  reading `no --since given and no git tag found reachable from <until>: <err>` (decision 7).
- Carry T-094 decision 3's rationale comment across intact, and add why the fallback exists
  (a root commit or a shallow clone has no `<until>^`).

#### Task 4 — CLI: the tagged-`--until` note

Still `internal/cli/changelog.go`:

- Add `const defaultChangelogSection = "Unreleased"` and use it as the `--section` flag default.
- Add `tagNote(root, until, section string) string` returning the decision-8 line or `""`;
  call it from `printChangelogCheckReport` right after the header `Printf`, printing only when
  non-empty. Passing the resolved `until` in keeps the git call out of the printer's other
  responsibilities — if that reads awkwardly, resolve the note in `runChangelogCheck` and pass the
  string down instead; either is fine, but do not put a second git call inside the candidate loop.

#### Task 5 — tests

- `internal/changelog/changelog_test.go`: a `Check` case whose subjects include
  `board: T-010, T-011 re-aimed after review` asserting `Excluded[0].IDs == []string{"T-010",
  "T-011"}`; one carrying the real-world shape
  `board: T-080 review findings recorded, moved to rework; T-042, T-070, T-081 patched by the
  sweep` (4 ids, 3 of them outside the leading list — the regression that pins decision 2 against
  a well-meaning "tighten it to the §0 grammar" edit); one `board:` subject with no id at all
  asserting `IDs` is empty; and one asserting a deduplicated subject naming the same id twice
  yields it once.
- `internal/cli/changelog_test.go`: the default report names **both** ids of a two-id `board:`
  commit and uses the word `mentioning`; a tagged **root** commit resolves via the task-3 fallback
  and exits `exitOK` (`gitTag` immediately after the seed commit, no second commit); the
  tagged-`--until` note appears for `--until <tag>` with the default `--section` and is **absent**
  for the same `--until` with `--section <version>` — the pair is what stops the note becoming
  unconditional noise.

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then the live regressions — read-only, so they run against this repo's own
history (no install, no writes; the self-modify policy's throwaway-dir rule does not apply).
Every expected value below was measured on `main` at `58dcc0d`.

**1. The summary gains exactly the two ids T-094's acceptance test could only describe in prose.**

```
./pickle changelog check --since v0.4.0 --until v0.5.0 --section 0.5.0
```

Expected: unchanged payload — exactly one candidate `T-090` naming its `6-done/` file, exit `0`,
`28` exclusions, no unclassified list. The exclusion line changes from

```
  excluded 28 board: bookkeeping commit(s) covering T-080, T-081, T-089, T-090, T-091 (…)
```

to

```
  excluded 28 board: bookkeeping commit(s) mentioning T-042, T-070, T-080, T-081, T-089, T-090, T-091 (…)
```

— **`T-042` and `T-070` are the two ids T-094's own acceptance test had to assert as "absent
(`board:`-only)"**, invisible in the summary until now. Still no `+N with no ticket id` clause
(every subject in that range leads with an id).

**2. Decision 4's accepted consequence, on the default range:**

```
./pickle changelog check
```

Expected: `no candidates`, exit `0`, and the exclusion line goes from
`covering T-089, T-092, T-093, T-094 (+1 with no ticket id; …)` to
`mentioning T-089, T-092, T-093, T-094, T-095 (…)` — `T-095` gained, and the `+1` clause gone
because `board: file T-092, T-093 (skill gaps found while releasing v0.5.0)` now parses two ids.
Both changes are decisions 2 and 4 exactly; if the `+1` clause survives, decision 2 was not
applied to the whole subject.

**3. The tagged-`--until` note (decision 8):**

```
./pickle changelog check --until v0.5.0
./pickle changelog check --until v0.5.0 --section 0.5.0
```

Expected: the first prints a `note:` line naming `v0.5.0` and suggesting `--section 0.5.0`
immediately after the header; the second prints **no** note. Both exit `0`.

**4. The fallback (decision 7), in a throwaway repo — the one case this repo's history cannot
show:**

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q -b main .
# minimal pickle.toml + the seven tickets/ dirs + a CHANGELOG.md with ## [Unreleased],
# one commit, then: git tag v0.1.0     (HEAD is the ROOT commit and is tagged)
./pk changelog check
```

Expected: exit `0` with `v0.1.0..HEAD` in the header — not today's exit `1` and
`no git tag found reachable from HEAD^`. Then delete the tag and re-run: expected exit `1` with
the new message naming `HEAD`, not `HEAD^`.

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`**, `[#cmd-changelog-check]`: update the exclusion-summary
  paragraph — the example line's verb (`mentioning`) and the fact that it names **every** id a
  bookkeeping subject mentions, not just the first; state the narrowed meaning of the
  `+N with no ticket id` clause (decision 4). In the `--until` paragraph, add the `<until>^`
  fallback and the tagged-`--until` note, with the note's text quoted once so the docs and the
  binary can be diffed by eye. Keep the "mentions only", "never a gate" and "one direction only"
  paragraphs — still true.
- **`docs/user-manual/cli-reference.adoc`** command table (~line 46): the flag list is unchanged
  (no new flags), so touch it only if the one-line description drifted.
- **`internal/cli/cli.go`** help text: re-read; its current wording ("Excluded `board:`
  bookkeeping commits summarize to one line unless `--show-excluded`") stays true, so change it
  only if the note line makes it misleading.
- **`RELEASING.md`**: it says to run `changelog check` *before* retitling and tagging, which is
  still right — but decision 8's note exists precisely for the reader who runs it *after*
  `git tag`. Add one sentence: after a tag exists, audit that release with
  `--until <tag> --section <version>`. This is the doc half of F2.
- **`CHANGELOG.md`**: amend the existing `[Unreleased]` `changelog check` bullet rather than
  adding a second one (the command has never shipped; T-094 set this precedent), appending
  `T-095` to its id list.

### Finish (mandatory)

Interactive-rebase WIP into atomic commits — suggested: one for the id-list change (tasks 1–2 +
their tests), one for the range/`--since` edges (tasks 3–4 + their tests), one for docs.
Suggested Conventional Commit for the primary commit:

```
fix(changelog): name every id an excluded subject mentions, and survive a tagged or parentless --until (T-095)
```

Keep history on merge (rules §0, root-path child); do not squash. Await explicit approval before
pushing or opening the MR. **Push `origin main` first if the local base is ahead** — and verify
`git diff --name-only origin/main...HEAD` carries no `tickets/` path before pushing the branch.

## Review

Reviewed 2026-08-12 against `main` at `bf59b7a` (PR #34, merge `876e63d`'s successor — see
History). **Retroactive**: the branch was pushed, approved and merged before this review ran,
so the audit is of merged code and any fix lands on a fresh branch off `main` rather than on
`feat/T-095-changelog-summary-and-range-edges`. Both review addenda in `pickle.toml` are
commented out, so the generic protocol applied alone. Feature branch and `main` are identical
in content (`git diff main origin/feat/T-095-…` shows only `tickets/BOARD.md`, which is
base-branch bookkeeping exactly as rules §0 requires).

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped (conscious)**: the host `docs_readability` reviewer is
      pinned to `gemini-2.5-pro` via github-copilot, which returns `model_not_supported`. This
      is the same failure T-094's review recorded, and **T-096** (READY) exists to repin it.
      Step 4b is explicitly optional and never blocks.
- [x] Findings recorded with severity **and** disposition per the rules §5 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + bookkeeping committed per policy (step 9)

### Implementation audit

All five tasks **met**, in the files the plan named. All eight confirmed decisions **honoured**
as written — including decision 7, whose implementation matches its stated mechanism exactly;
finding **B1** is that the *mechanism* is broader than the *rationale* the decision gave for it.

`just build` · `just test` · `just lint` · `just docs-check` — all four green.

Acceptance test re-run verbatim:

| § | expected | result |
|---|---|---|
| 1 | `mentioning T-042, T-070, T-080, T-081, T-089, T-090, T-091`, 28 exclusions, 1 candidate `T-090`, exit `0` | **met**, byte-for-byte |
| 2 | `no candidates`, exit `0`, no `+N` clause, `T-095` gained | **met**; id list additionally carries `T-096` (22 exclusions, not the measured 21) — fully explained by two `board: T-096 …` commits a concurrent session landed after the plan's `58dcc0d` measurement, not by this change |
| 3 | `note:` line with default `--section`, absent with `--section 0.5.0`, both exit `0` | **met** |
| 4 | throwaway repo: tagged root commit → exit `0` with `v0.1.0..HEAD`; untagged root → exit `1` naming `HEAD` not `HEAD^` | **met** |

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| **B1** | **blocking** | — | `defaultSince`'s fallback fires whenever `describe <until>^` *fails for any reason*, not only when `<until>^` cannot resolve. When `<until>^` resolves but no tag is reachable from it — a project's **first release**, tagged — it falls back to describing `<until>`, which returns `<until>`'s *own* tag and yields the empty range `<tag>..<tag>`. The command then prints `no candidates — every shipped ticket is mentioned` for tickets that genuinely shipped unmentioned. This is precisely the "empty, falsely-passing `v0.5.0..v0.5.0`" failure T-094 decision 3 exists to prevent, and T-095 decision 1 locks T-094's decisions as standing. Decision 8's `note:` compounds it: it advises `try --section <version>`, which reports clean too. | Scratch repo, 3 commits, only tag `v1.0.0` on `HEAD`, `T-001`/`T-002` shipped and absent from the changelog → `changelog check: v1.0.0..HEAD` / `no candidates — every shipped ticket is mentioned`, exit `0`. The **pre-T-095 binary** (built from `c5005fa`) on the identical repo exits `1` with an honest `no git tag found reachable from HEAD^`. Found independently by the reviewer and by an independent code-audit agent. | Fall back only when `<until>^` genuinely does not resolve — test it separately (`rev-parse --verify <until>^`) instead of inferring it from `describe`'s failure — and never return a `since` equal to the resolved `<until>`. Decision 7's rejected alternative (fall back to full history) stays rejected; this is narrowing the trigger to what decision 7's own rationale describes, not widening the behaviour. |
| **B2** | **blocking** | — | The docs state the fallback's trigger as the narrow version B1 shows it isn't, in two files — and `cli-reference.adoc`'s paragraph contradicts *itself*: four sentences earlier it explains that `^` exists so the range cannot collapse to "the empty, falsely-passing `v0.5.0..v0.5.0`", then documents a fallback that produces exactly that. | `docs/user-manual/cli-reference.adoc:745` "When `<until>^` cannot resolve at all — `--until` names the repository's root commit, or a shallow clone has no parent to name —"; `CHANGELOG.md:21` "when `--until` has no parent commit to describe". Both describe only the parentless case. | Rewrite both to the trigger that ships after B1 is fixed, and state the residual empty-range case explicitly if one remains. |
| **B3** | **blocking** | — | `cli-reference.adoc` says an explicit `--section` suppresses the tagged-`--until` note. It does not: `tagNote` compares against `defaultChangelogSection`, so `--section Unreleased` still prints it — which T-095 decision 8 **explicitly accepted** ("indistinguishable from the default and will also get the note — acceptable"). The docs therefore contradict both the binary and a locked decision. | `docs/user-manual/cli-reference.adoc:759` "Passing an explicit `--section` suppresses it". Reality: `./pickle changelog check --until v0.5.0 --section Unreleased` prints `note: v0.5.0 is a tag …`. Code: `internal/cli/changelog.go:263`. | "Passing a `--section` other than the default `Unreleased` suppresses it — an explicit `--section Unreleased` is indistinguishable from the default and still gets the note." |
| N1 | non-blocking | folded → T-095 | `tagNote` prints the tag `describe --exact-match` resolved, not the ref the user passed, so it asserts that the user's ref *is* a tag when it isn't. `--until main` (a branch) prints `note: version-2 is a tag`. With two tags on one commit it names an arbitrary one. | Scratch repo: `./pk changelog check --until main` → `note: version-2 is a tag — …`. `internal/cli/changelog.go:270-271`. | Phrase as `<until> is at tag <tag>`, or only emit the note when `until` is itself the tag name. |
| N2 | non-blocking | folded → T-095 | `strings.TrimPrefix(tag, "v")` assumes a `vN` tag and mangles any tag beginning with a literal `v` word, producing visibly broken output. | `./pk changelog check --until version-2` → `note: version-2 is a tag — this range is release **ersion-2**, not [Unreleased]; try --section **ersion-2**`. `internal/cli/changelog.go:271`. | Trim only when the character after `v` is a digit. |
| N3 | non-blocking | **new ticket → T-097** | The permissive `idRE` scan does not merely add noise (which decision 2 weighed and accepted) — a false positive **silences** the `(+N with no ticket id)` alarm, because `printExclusions` counts `noID` only when `len(ex.IDs) == 0`. A bookkeeping subject with no ticket id at all is reported as though it named one. | `board: note the SHA-256 subject handling` → `excluded 1 board: bookkeeping commit(s) mentioning SHA-256`, and no `+1` clause. `cli-reference.adoc:724` promises that clause "is never the thing the summary hides". Matching tokens verified: `SHA-256`, `UTF-8`, `ISO-8601`, `AES-256`, `RFC-7231`, `CVE-2024`, `HTTP-2`. | Fixing it reopens locked decision 2 → **T-097**. |
| N4 | non-blocking | folded → T-095 | Decision 7's third leg — the error naming `<until>` rather than `<until>^` — has no test; it was verified only by hand. The B1 fix must pin this path anyway. | No test in `internal/cli/changelog_test.go` asserts the error string. | Add a case asserting the message names `HEAD`, not `HEAD^`. |
| N5 | non-blocking | noted | Decision 8 specifies the note prints "immediately after the header", but the tests assert only `strings.Contains` — nothing pins its position, so a future refactor could move it below the candidate list silently. | `TestChangelogCheckTagNoteOnDefaultSection` uses `Contains` only. | A line-order assertion, if this ever regresses. |
| N6 | non-blocking | folded → T-095 | `Exclusion.IDs` is computed for `Unclassified` commits but never consumed (`printUnclassified` prints subjects only), and via a looser rule than the kind's own `parenIDRE`. The field comment doesn't say only `Excluded` reads it today. | `internal/cli/changelog.go:238-247`; `internal/changelog/changelog.go:128-137`. The comment's claim that an `Unclassified` commit's `IDs` is never empty *was* verified true (a `parenIDRE` match implies an `idRE` match). | One clause in the field comment. |
| N7 | non-blocking | folded → T-095 | The summary's third branch — `excluded N board: bookkeeping commit(s), none with a parsable ticket id` — is undocumented; the docs describe only the `(+N …)` clause. Pre-existing from T-094, but this ticket rewrote that exact paragraph. | `internal/cli/changelog.go:220-221` vs `cli-reference.adoc:724-726`. | One clause: "when *no* excluded subject names an id, the line says so instead." |
| N8 | non-blocking | folded → T-095 | In `CHANGELOG.md` the fallback clause was appended *inside* the four-item defaults parenthetical, so it grammatically attaches to `Unreleased`, its nearest antecedent. | `CHANGELOG.md:20-21` "(the last tag before `--until`, `HEAD`, `CHANGELOG.md`, `Unreleased`, falling back to the last tag at `--until` itself …)". | Close the parenthesis after `Unreleased`, then a separate clause. |
| N9 | non-blocking | folded → T-095 | `RELEASING.md`'s new aside sits between "First, run `pickle changelog check`" and "Then update `CHANGELOG.md`", interrupting the First→Then→tag sequence with a paragraph that only applies *after* tagging — and runs six lines for the one sentence the plan asked for. | `RELEASING.md:14-20`. | Move below the `git tag` block, or mark it a `> Note:` aside. |
| N10 | non-blocking | noted | `internal/cli/cli.go`'s help text still describes the defaults as "the last git tag before `<until>`, and `HEAD`" — no fallback, no note. The plan sanctioned leaving it ("change it only if the note line makes it misleading"); recording so the omission is a decision rather than an oversight. | `internal/cli/cli.go:126-128`. | None required. |
| N11 | non-blocking | noted | `cli-reference.adoc` characterises the rules' `board:` grammar as "usually honoured in spirit rather than to the letter", while the shipped rules state it as a rule — the user manual mildly undercutting the convention the command depends on. | `cli-reference.adoc:718-720` vs `skill/resources/tickets-README.md:61`. | Soften, or cross-reference §0, if it ever grates. |
| N12 | non-blocking | noted | `boardIDRE`'s captured id is now written but never read: `Check` discards `ClassifySubject`'s second return for `Bookkeeping`, and no caller exists outside the package. Decision 6 locks the signature, and its doc comment already flags this, so no action — but it is the natural home for a strict ticket-id rule if **T-097** introduces one. | `internal/changelog/changelog.go:184`; `grep changelog.ClassifySubject` → no external callers. | None; noted for T-097. |

**Disposition summary:** 15 findings — **3 blocking** (B1 code, B2/B3 docs; not dispositioned,
fixed in rework) · 12 non-blocking: **7 folded** into this ticket's own rework scope
(N1, N2, N4, N6, N7, N8, N9 — all in the functions and paragraphs the blocking fixes must touch
anyway, so scheduling them separately would be waste), **1 new ticket** (N3 → **T-097**, the
only finding whose fix reopens a locked decision), **4 noted** (N5, N10, N11, N12).

### Verdict

**Rework.** The feature is substantially right — every task shipped, every decision was
honoured, the acceptance test reproduces byte-for-byte, and the F1 half (the id-list fix) is
clean. What sends it back is that the F2 half's fallback trades T-094's spurious *error* for a
silent false *pass* in the first-release case, and an advisory checker whose one job is to
catch "you forgot to write this down" must not answer "nothing to report" when there is. The
fix is a narrowing of the fallback's trigger, not a redesign; the two docs findings are the
same defect stated in prose, plus one sentence that contradicts decision 8.

Because the branch is already merged, the rework lands on a **new branch off `main`**
(suggest `feat/T-095-rework-fallback-trigger`) rather than on the merged feature branch.

### Rework (2026-08-12)

Landed on the **same** `feat/T-095-changelog-summary-and-range-edges` branch after all —
reset onto `main`'s current tip first (PR #34 had already merged its content, so
`git diff origin/main...HEAD` was empty immediately after the reset), then the fix commits
added on top. No new branch name was needed; a second PR from this same branch name will carry
only these three commits once the scoped re-review approves publishing.

Fixed exactly the 3 blocking findings, plus the 7 non-blocking findings this ticket's own
review dispositioned `folded → T-095` (same functions/paragraphs the blocking fixes had to
touch). The 4 `noted` findings (N5, N10, N11, N12) and **T-097** (N3) are untouched, as recorded.

| id | fix | evidence |
|---|---|---|
| **B1** | `defaultSince` now tests `<until>^`'s existence directly with `rev-parse --verify`, not by inferring it from `describe`'s failure. A parent that exists but has no reachable tag (a first release) still errors, naming `<until>^`; only a genuinely absent parent falls back to describing `<until>` itself. | New test `TestChangelogCheckNoTagBeforeFirstReleaseErrorsRatherThanFalsePasses`: 3-commit repo, only tag on `HEAD`, two shipped-unmentioned tickets — now exits `1` naming `HEAD^`, not the pre-fix `0`/`no candidates`. `TestChangelogCheckUntilFallsBackOnTaggedRootCommit` (the original decision-7 case) still passes unchanged. |
| **B2** | `cli-reference.adoc` and `CHANGELOG.md` now state the fallback's trigger as "`<until>^` does not *exist at all*", and name the still-erroring "parent exists, no tag before it" case explicitly, next to the paragraph explaining why that distinction matters. | `docs(changelog)` commit; `just docs-check` green. |
| **B3** | `cli-reference.adoc` now says a `--section` **other than the default** suppresses the note, and states `--section Unreleased` explicitly still gets it — matching decision 8 and the binary. | Same commit; live-checked `--until v0.5.0 --section Unreleased` still prints the note. |
| N1 | `tagNote` now names the ref the caller passed *and* the tag it resolved to separately: `"%s is at tag %s"`, not `"%s is a tag"` naming only the resolved tag. | New test `TestChangelogCheckTagNoteNamesTheRefNotJustTheTag` (`--until main` → `note: main is at tag v0.2.0`). |
| N2 | New `versionFromTag` helper trims a leading `v` only when a digit follows it. | New test `TestChangelogCheckTagNoteVersionTrimsOnlyANumericVPrefix` (`--until version-2` → `try --section version-2`, not `ersion-2`). |
| N4 | Pinned by the same fix that closed B1 — `TestChangelogCheckUntilFallsBackErrorsNamingUntilNotUntilCaret` asserts the genuinely-parentless error path names `HEAD`, not `HEAD^`. | Test added alongside B1's. |
| N6 | `Exclusion.IDs`'s field comment now says only `Result.Excluded`'s `IDs` is read today. | `internal/changelog/changelog.go`, doc-comment-only edit. |
| N7 | `printExclusions`'s doc comment now mentions the summary's "no excluded subject names an id at all" branch. | `internal/cli/changelog.go`, doc-comment-only edit. |
| N8 | `CHANGELOG.md`'s fallback clause now closes the defaults parenthetical before it, instead of grammatically attaching to `Unreleased`. | `CHANGELOG.md` diff. |
| N9 | `RELEASING.md`'s tagged-release aside moved below the `git tag`/`git push` block as a marked `> **Note:**`, shortened from six lines. | `RELEASING.md` diff; the First → Then → tag sequence now reads unbroken. |

`just build/test/lint/docs-check` green throughout, including a standalone-checkout build/test
of each intermediate commit. Three atomic commits:
`e5db3bb` (B1 + N4), `e03f801` (N1 + N2), `50167e5` (B2 + B3 + N6 + N7 + N8 + N9).

### Scoped re-review (2026-08-12)

Scoped to the 3 blocking and 7 folded findings above, per the rules §5 — the feature was **not**
re-audited from scratch. Read from `main` at `213e6e2`; code audited on
`feat/T-095-changelog-summary-and-range-edges` at `50167e5`. `just build` · `just test` ·
`just lint` · `just docs-check` all green.

- [x] Blocking findings B1, B2, B3 re-verified resolved (step 2, scoped)
- [x] Consistency sweep over the reworked functions and paragraphs only (step 4, scoped)
- [x] Documentation audit — the three changed docs files re-read, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped (conscious)**, third consecutive time: the host
      `docs_readability` reviewer is still pinned to `gemini-2.5-pro` via github-copilot and
      returns `model_not_supported`. **T-096** (READY) exists to repin it. Step 4b is
      explicitly optional and never blocks.
- [x] Re-review findings recorded with severity **and** disposition (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages presented for approval; bookkeeping committed per policy (step 9)

#### Blocking findings — all three resolved

| id | verdict | evidence |
|---|---|---|
| **B1** | **resolved** | `defaultSince` now gates on `rev-parse --verify <until>^` rather than inferring absence from `describe`'s failure, so "parent exists, no tag before it" and "no parent" are finally distinct facts. Reproduced the exact finding scenario live in a throwaway repo (3 commits, sole tag `v1.0.0` on `HEAD`, `T-001`/`T-002` shipped and unmentioned): now `no --since given and no git tag found reachable from HEAD^`, exit `1` — the false `no candidates`/exit `0` pass is gone. Decision 7's original case is unregressed: tagged root commit → `v0.1.0..HEAD`, exit `0`. Pinned by `TestChangelogCheckNoTagBeforeFirstReleaseErrorsRatherThanFalsePasses`. |
| **B2** | **resolved** | `cli-reference.adoc` now states the trigger as "`<until>^` does not *exist at all*" **and** spells out the still-erroring "reachable parent with no tag before it" case in the same paragraph as the empty-range rationale, so the self-contradiction is gone. `CHANGELOG.md` matches ("no parent commit at all to describe"). Both re-read in full against the shipped behaviour. |
| **B3** | **resolved** | `cli-reference.adoc` now reads "Passing a `--section` other than the default `Unreleased` suppresses it … an explicit `--section Unreleased` is indistinguishable from the default and still gets the note." Live-checked: `--until v0.1.0 --section Unreleased` prints the note; `--section <version>` does not. Matches decision 8 exactly. |

#### Folded findings — all seven resolved

N1 (`main is at tag v0.1.0`, verified live with `--until main`) · N2 (`versionFromTag`; `--until
version-2` no longer yields `ersion-2`) · N4 (`TestChangelogCheckUntilFallsBackErrorsNamingUntilNotUntilCaret`,
asserting both the presence of `HEAD` and the *absence* of `HEAD^`) · N6, N7 (doc-comment
clauses, both present and accurate) · N8 (`CHANGELOG.md` parenthetical now closes before the
fallback clause) · N9 (`RELEASING.md` aside now a `> **Note:**` below the `git tag` block; the
First → Then → tag sequence reads unbroken).

Acceptance test re-run verbatim: §1 byte-for-byte as specified; §2 as specified, with the id
list and count now at `T-089, T-092, T-093, T-094, T-095, T-096, T-097` / 23 exclusions — the
same concurrent-bookkeeping drift the first review recorded, not a behaviour change; §3 and §4
as specified.

#### Re-review findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | non-blocking | **fixed inline** | The finding-N7 edit reflowed the exclusion-summary paragraph and left one 115-character prose line — the longest in `cli-reference.adoc`, in a file whose prose wraps at ~72. Prose this branch's rework authored, no behaviour change, so it meets §5's inline bar. | `docs/user-manual/cli-reference.adoc:731` before the fix; every other >82-char line in the file is a synopsis or code line. | Fixed: rewrapped, commit `6d9e00a`. `just docs-check` green after. |
| R2 | non-blocking | noted | `defaultSince` cannot distinguish "`<until>` has no parent" from "`<until>` does not resolve at all": a typo'd ref takes the fallback branch and errors `no --since given and no git tag found reachable from <typo>`, blaming a missing tag for a missing ref. Pre-existing in shape — the pre-rework code conflated the same two cases — so per §5's causation rule it is `noted`, not `fixed inline`. | `./pk changelog check --until nosuchref` → `no git tag found reachable from nosuchref: … exit status 128`. `internal/cli/changelog.go`, the `rev-parse --verify` gate. | A `rev-parse --verify <until>` pre-check, if a typo'd `--until` ever confuses anyone. |
| R3 | non-blocking | **fixed inline** | Bookkeeping commit `c8ee4e2` duplicated the whole tail of this ticket's `## Review` (checklist → Verdict) below the Rework section, and mangled the seam into a bogus `## History` appended (step 6)` heading — a second `## History`-shaped line ahead of the real one, ~60 lines of stale duplicate. Prose this ticket's own bookkeeping authored, no behaviour change. | Ticket file on `main` at `213e6e2`, lines 427–487; `git show c8ee4e2` shows the duplication introduced whole. | Fixed: the duplicate block deleted in this review's bookkeeping commit; `## Outcome`/`## Description`/`## Implementation Plan`/`## Review`/`## History` now appear exactly once each and `pickle board audit` is clean. |

**Disposition summary:** 3 re-review findings — **0 blocking** · 3 non-blocking: **2 fixed
inline** (R1 docs rewrap, R3 duplicated Review block), **1 noted** (R2). No new tickets: neither
R2 nor anything else here passes §5's promotion test. The 4 findings the first review left
`noted` (N5, N10, N11, N12) and **T-097** (N3) remain untouched, as recorded.

#### Verdict

**Done.** Every blocking finding is fixed at the mechanism level, not papered over: B1's fix
narrows the fallback's *trigger* to what decision 7's own rationale always described, leaving
decision 7's behaviour intact and restoring the honest error in the first-release case, and it
is pinned by a regression test that fails against the pre-rework binary. B2 and B3 bring the
prose back into agreement with the binary and with decision 8. All seven folded findings landed
in the same commits. The three re-review findings are cosmetic or pre-existing.

## History

- 2026-08-12 — created (TO DO). source: T-094's review, findings F1/F2, batched by theme per the
  rules §5 (see `tickets/6-done/T-094-…md`'s `## Review`). Graded `low`/`low`/`S` against the
  backlog: both are a few lines in one young command, both are off the golden path
  (`RELEASING.md` runs the check on an untagged `main`), and both already have their measured
  reproduction recorded — lower impact than T-094 itself, which fixed the edges people actually
  hit
- 2026-08-12 — refined. Description re-verified against `main` at `58dcc0d` (T-094 merged, PR
  #33); every path, symbol and regex it names still exists and still behaves as described. Four
  design questions were settled and written into the plan as confirmed decisions: (a) the
  summary's id set comes from a **permissive `idRE` scan of the whole subject**, not from a
  stricter version of `boardIDRE`'s anchored leading-list — measured across the repo's whole
  history, 8 of the 9 multi-id `board:` subjects carry their extra ids in the *verb phrase*, so a
  grammar-strict parser would still miss most real cases, and this makes both halves of the report
  use the one id rule `sectionIDs` already uses; (b) the verb therefore becomes `mentioning`
  rather than `covering`, and the `+N with no ticket id` clause narrows to "no id anywhere" —
  accepted, with the reasoning recorded, since this repo's current `+1` instance disappears; (c)
  the `<until>^` resolution gains a single fallback to `describe <until>`, with "fall back to all
  history" explicitly rejected as a separate decision; (d) a tagged `--until` with the default
  `--section` earns an advisory `note:` line, which is a better answer than documenting the
  confusion. Not split: two findings, one command, one package, ~40 lines total — neither half
  would be picked up alone. Grades unchanged (`low`/`low`/`S`): the plan grew precise, not larger
- 2026-08-12 — TO DO → READY: plan complete; four design questions settled (permissive id scan, mentioning verb, single <until>^ fallback, tagged-until note)
- 2026-08-12 — READY → IN DEVELOPMENT: picked up
- 2026-08-12 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-12 — merged: PR #34 (`feat/T-095-changelog-summary-and-range-edges` → `main`, merge
  commit `bf59b7a`), history kept, not squashed (rules §0, root-path child). Merged on the
  human's approval **before** this ticket's review ran, so the review below is retroactive and
  its rework lands on a fresh branch off `main` rather than on the merged feature branch
- 2026-08-12 — IN REVIEW → REWORK: review: 3 blocking (B1 fallback empty-range false pass; B2/B3 docs), 7 folded, T-097 spawned, 4 noted
- 2026-08-12 — REWORK → IN REVIEW: 3 blocking + 7 folded findings fixed; ready for scoped re-review
- 2026-08-12 — IN REVIEW → DONE: scoped re-review: 3 blocking + 7 folded findings verified fixed; 2 fixed inline, 1 noted
- 2026-08-12 — published on approval: rework branch tidied to three atomic commits (`e2b1fd3`
  B1+N4, `b1ca02d` N1+N2, `7de2310` B2+B3+N6+N7+N8+N9+R1 — the review's own inline R1 rewrap
  folded into the docs commit it belonged to, and the `(T-095 rework)` subject suffixes
  normalized to the rules §0 `(T-095)` form). Fast-forward over the stale remote tip `3353b31`,
  so no history was rewritten in anger. **PR #35** open against `main`; history to be kept, not
  squashed (rules §0, root-path child). This is T-095's *second* PR — #34 merged the original
  implementation before the first review ran. **Merging is the human's**; add the `merged:` line
  here once #35 lands
- 2026-08-12 — merged to `main` (`eed6b3e`, PR #35): the rework loop closed, history kept
