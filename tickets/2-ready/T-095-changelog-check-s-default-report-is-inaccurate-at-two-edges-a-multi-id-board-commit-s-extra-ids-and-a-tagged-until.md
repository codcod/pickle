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

<!-- empty until IN REVIEW -->

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
