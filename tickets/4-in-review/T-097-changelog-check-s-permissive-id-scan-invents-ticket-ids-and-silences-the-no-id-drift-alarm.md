---
id: T-097
title: changelog check's permissive id scan invents ticket ids and silences the no-id drift alarm
project: pickle
depends-on: []
spawned-by: [T-095]
impact: medium
complexity: low
cost: M
---

# T-097 — changelog check's permissive id scan invents ticket ids and silences the no-id drift alarm

## Outcome

After this ships, `pickle changelog check` never presents a non-ticket token (`SHA-256`,
`UTF-8`, `RFC-7231`, `CVE-2024`) as a ticket id — not in its exclusion summary and not as a
candidate — because it recognises only the ticket-id prefixes the project actually registers;
and its `(+N with no ticket id)` clause fires for every bookkeeping subject that names no
*ticket*, rather than being silenced by an id-shaped token that happens to appear in the prose.

## Description

Non-blocking finding N3 from T-095's review. T-095 decision 2 deliberately made the exclusion
summary's id set a **permissive** scan of the whole `board:` subject with the pre-existing
`idRE` (`\b[A-Z][A-Z0-9]*-\d+\b`), rather than a grammar-strict parse anchored to the rules §0
leading-id form. That decision was right about the case it weighed — measured across this
repo's history, 8 of 9 multi-id `board:` subjects carry their extra ids in the verb phrase, so
a strict parser drops most of them — and it explicitly accepted false-positive *noise* as the
price.

What decision 2 did **not** weigh is that a false positive does not merely add noise: it
**silences an alarm**. `printExclusions` counts a subject into `noID` only when
`len(ex.IDs) == 0`, so any id-shaped token anywhere in the subject suppresses that subject's
contribution to the `(+N with no ticket id)` clause — the clause T-094 decision 4 introduced as
"the loudest possible symptom of a convention drift", and which
`docs/user-manual/cli-reference.adoc` currently promises "is never the thing the summary
hides".

Measured (scratch repo, `pickle` at `bf59b7a`), a bookkeeping commit carrying **no ticket id at
all**:

```
$ git commit -m "board: note the SHA-256 subject handling"
$ ./pk changelog check
  excluded 1 board: bookkeeping commit(s) mentioning SHA-256 (--show-excluded for subjects)
```

Both failures at once: `SHA-256` is reported as though it were a ticket id, and the `+1 with no
ticket id` clause that should have fired does not. Tokens verified to match `idRE`: `SHA-256`,
`UTF-8`, `ISO-8601`, `AES-256`, `RFC-7231`, `CVE-2024`, `HTTP-2`, `PR-42`, `MR-7`.

The exposure is currently zero *in this repo* — T-095 measured that no non-`T-` id-shaped token
exists anywhere in its commit-subject history — but `pickle` is installed into other projects,
whose bookkeeping prose this project has never seen, and where a `board:` commit mentioning
`UTF-8` or `CVE-2024` is entirely ordinary.

Fixing this reopens T-095 decision 2, which is why it is a ticket rather than a review fix.
Candidate approaches, cheapest first: (a) recognise only the id prefixes the flow actually
uses, reading the configured ticket-id prefix rather than accepting any `[A-Z][A-Z0-9]*`
family; (b) keep the permissive scan for *display* but compute the `noID` count from a strict
ticket-id rule, so the alarm and the inventory stop sharing one predicate; (c) a small
stop-list.

**Chosen at refinement (2026-08-13): (a), generalised to every id-recognition site in the
package.** (b) was the front-runner at filing but is the wrong trade once (a) is on the table:
splitting the display set from the alarm's count leaves the *display* still fabricating ids,
which the Outcome forbids; and once the scan cannot match `SHA-256` at all, the two can keep
sharing one predicate, which is simpler than maintaining two. (c) stays rejected — the space of
id-shaped non-ids is unbounded.

**Found at refinement, and it widens the fix: the exclusion summary is not the worst instance.**
`trailingIDRE` (`internal/changelog/changelog.go:69`) is equally permissive and feeds the
*shipped* set. A perfectly ordinary child-project commit — `fix(http): handle chunked encoding
(RFC-7231)` — classifies as `ChildProject` with id `RFC-7231`, so the report prints
`RFC-7231  (no ticket file found — check for a recorded decision before adding an entry)` in
its **candidate** list: a fabricated ticket id in the loudest part of the output, and a
permanent phantom candidate that no changelog entry can ever clear. `parenIDRE` (the
Unclassified safety net) and `sectionIDs` (the changelog-side scan) share the same defect. All
six id patterns therefore move to one prefix-aware predicate; fixing only `subjectIDs` would
leave the louder bug in place. This is why the cost is regraded `S` → `M`: the change itself is
still small, but it touches every id pattern in the package and every `Check`/`ClassifySubject`
call site in its tests.

Whatever ships must keep T-095 decision 2's measured property: extra ids carried in a `board:`
subject's *verb phrase* must still be named (`board: T-089 reviewed and done, T-090 filed,
T-070 re-graded` → all three). Pin it with the regression tests T-095 already added in
`internal/changelog/changelog_test.go`.

Soft couplings: **T-093** shipped the command and `idRE`; **T-094** decision 4 introduced the
`+N` clause this finding shows can be silenced; **T-095** decision 2 chose the permissive scan
and is the decision this ticket reopens. Related, no action required: `boardIDRE`'s captured id
is now written but never read outside package tests (T-095 review finding N12) — if this ticket
introduces a strict ticket-id rule, that regex is the natural place for it to live.

## Implementation Plan

### 0. Feature branch (mandatory)

`feat/T-097-prefix-aware-id-scan`, created in the `pickle` child-project's repo (path `.`)
before any change. Local WIP commits are fine; **no push and no MR without explicit user
approval**, and merging is always the human's. Root-path child, so tidy the WIP commits by
interactive rebase into a small number of atomic commits and **keep that history** on merge
(rules §0) rather than squashing.

Bookkeeping (this ticket file + `BOARD.md`) is committed on `main`, never on this branch.

### Prerequisite gate (hard)

None. T-093/T-094/T-095 all shipped and merged; `config.Project.Prefix()` and
`config.ticketPrefixRE` already exist. No ticket needs to land first.

### Confirmed design decisions (do not deviate without asking)

1. **One prefix-aware predicate, applied at every id-recognition site.** "Is this token a
   ticket id?" gets exactly one definition in `internal/changelog`: the token's prefix is one
   the project registers. All six patterns are rebuilt from it — `boardIDRE`, `trailingIDRE`,
   `parenIDRE`, `idRE`, and through them `subjectIDs` and `sectionIDs`. Do **not** fix only
   `subjectIDs`/the exclusion summary: `trailingIDRE` fabricates candidates, which is worse
   (see the Description).
2. **The display set and the `noID` count keep sharing one predicate.** Approach (b) — a
   permissive scan for display, a strict one for the count — is explicitly **not** what ships.
   With decision 1 in place there is nothing left for a second, looser predicate to buy.
3. **T-095 decision 2's measured property survives, and is the regression to protect.** The
   scan stays a *whole-subject* scan, not a grammar-strict parse of the leading
   `board: T-NNN[, T-MMM …]` list. `board: T-089 reviewed and done, T-090 filed, T-070
   re-graded` must still yield all three ids. This ticket narrows *which tokens* count, never
   *where* they may appear. The regression tests T-095 added must pass unchanged in substance.
4. **The prefix set comes from `pickle.toml`'s registered children** — the union of
   `cp.Prefix()` over `cfg.Projects`, deduplicated (`Prefix()` already applies the `T`
   default). Not from a tree walk of `tickets/`, not from a new config key.
5. **Two consequences of decision 4 are accepted and documented, not engineered around.** An id
   whose prefix belonged to a child that has since been *unregistered*, or that was *renamed*
   via `ticket_prefix`, becomes invisible to this check. That is the correct trade for a
   read-only advisory command: the alternative (remembering every prefix ever used) needs
   persistent state the flow does not have. Say so in the docs so a reader who renames a prefix
   is not surprised.
6. **`internal/changelog` stays config-free.** It must not import `internal/config` — the
   package doc promises "pure text-in, text-out logic" and it is a leaf today. The prefix set
   arrives as a `[]string` parameter, resolved by `internal/cli/changelog.go` from `cfg`.
7. **Empty prefix slice falls back to `["T"]`, never to today's permissive behaviour.**
   `config.Validate` guarantees at least one project, so the CLI cannot pass empty — but a
   future caller must not be able to silently reinstate the bug. `T` is duplicated as an
   unexported literal in `internal/changelog` (with a comment naming `config.DefaultTicketPrefix`
   as the source of truth) rather than imported, per decision 6.
8. **Prefixes are `regexp.QuoteMeta`'d before interpolation.** `config.ticketPrefixRE`
   (`^[A-Z][A-Z0-9]{0,7}$`) already makes them metacharacter-free, so this is belt-and-braces
   against a future loosening of that rule — keep it, and say why in a comment.
9. **No behaviour change to exit codes, gating, or the three-branch summary.** The command stays
   advisory and always exits `0` (T-093 decision 2); the summary keeps its three branches
   (named ids / named ids + `(+N with no ticket id)` / "none with a parsable ticket id") —
   T-095 review finding N7's third branch must stay coherent, per this ticket's second History
   line.

### Tasks

#### Task 1 — `idPatterns`: the prefix-aware predicate

In `internal/changelog/changelog.go`, replace the four id regex package vars
(`boardIDRE`, `trailingIDRE`, `parenIDRE`, `idRE` — lines 63–86) with a struct built per call:

```go
// defaultPrefix mirrors config.DefaultTicketPrefix. Duplicated rather than
// imported: this package is a leaf and its doc promises pure text-in,
// text-out logic (T-097 decision 6).
const defaultPrefix = "T"

type idPatterns struct{ board, trailing, paren, any *regexp.Regexp }

func newIDPatterns(prefixes []string) *idPatterns
```

`newIDPatterns` builds one alternation group from the (deduplicated, `regexp.QuoteMeta`'d)
prefixes — `(?:T|OPS)` — and interpolates it into the four existing shapes, keeping every
other byte of each pattern identical:

- `board`: `^board:\s*(<alt>-\d+)\b`
- `trailing`: `\((<alt>-\d+)\)\s*$`
- `paren`: `\((<alt>-\d+)\)`
- `any`: `\b<alt>-\d+\b`

An empty or all-blank `prefixes` yields `["T"]` (decision 7). **Keep every existing comment**
on those four patterns — they record why each is anchored the way it is — and add to each a
sentence that the prefix group is now closed. `prTokenRE` and `sectionHeadingRE` are untouched:
neither matches a ticket id.

#### Task 2 — thread the patterns through the package

Still in `internal/changelog/changelog.go`:

- `ClassifySubject(subject string) (CommitKind, string)` →
  `ClassifySubject(subject string, prefixes []string) (CommitKind, string)`, a thin wrapper that
  builds `newIDPatterns(prefixes)` and delegates to an unexported
  `classifySubject(subject string, p *idPatterns)`. `Check` calls the unexported form so it
  compiles the patterns **once**, not once per subject.
- `Check(subjects []string, changelogText, section string)` →
  `Check(subjects []string, changelogText, section string, prefixes []string)`. Build
  `p := newIDPatterns(prefixes)` first and pass it to `classifySubject`, `subjectIDs` and
  `sectionIDs`.
- `subjectIDs(subject string)` → `subjectIDs(subject string, p *idPatterns)`, using `p.any`.
  Rewrite its doc comment: it is no longer "deliberately the same permissive rule", it is the
  same *whole-subject* rule with a closed prefix set — keep the T-095 measurement it cites
  (that is decision 3's evidence) and add why the prefix set closed (T-097).
- `sectionIDs(changelogText, section string)` → `sectionIDs(changelogText, section string, p
  *idPatterns)`, using `p.any`. Note in its comment that a changelog body legitimately contains
  `SHA-256`/`UTF-8` prose, so closing the prefix set matters on this side too.
- Update the package doc (`internal/changelog/changelog.go:1–15`) to state the recognition rule
  and that the prefix set is supplied by the caller.

#### Task 3 — the CLI resolves the prefix set

In `internal/cli/changelog.go`:

- Add an unexported helper `ticketPrefixes(cfg *config.Config) []string` returning the
  deduplicated union of `cfg.Projects[i].Prefix()`, in registration order (stable output for
  tests).
- Pass it at the `changelog.Check` call (line 98).
- Extend `printExclusions`' doc comment (line 201–216): the summary's id set is a whole-subject
  scan **restricted to registered ticket-id prefixes** (T-097), which is what makes the
  `(+N with no ticket id)` clause mean what it says.

#### Task 4 — tests

In `internal/changelog/changelog_test.go` (mechanical arg addition at ~13 `Check` call sites
and the `ClassifySubject` table; pass `[]string{"T"}` unless the case is about prefixes):

- `TestClassifySubject`: add cases proving `fix(http): handle chunked encoding (RFC-7231)` is
  **`Neither`** (was `ChildProject`/`RFC-7231`), and `Revert "x (CVE-2024)"` is `Neither` (was
  `Unclassified`).
- New `TestNonTicketTokensAreNotIDs`: a `board:` subject naming no ticket but containing
  `SHA-256` — assert `Exclusion.IDs` is **empty**, which is what restores the `+N` clause. Use
  the Description's exact reproduction subject (`board: note the SHA-256 subject handling`).
- New `TestMultiPrefixRecognition`: with `prefixes = []string{"T", "OPS"}`, a subject naming
  both `T-001` and `OPS-7` yields both; with `prefixes = []string{"T"}`, the same subject
  yields only `T-001` (decision 5's accepted blind spot, pinned so it is a choice and not a
  surprise).
- New `TestEmptyPrefixesFallBackToT`: `Check(..., nil)` recognises `T-001` and not `SHA-256`
  (decision 7).
- Keep T-095's verb-phrase regression (`board: T-089 reviewed and done, T-090 filed, T-070
  re-graded` → three ids) passing unchanged — decision 3.
- Assert `sectionIDs` no longer harvests `SHA-256` from a changelog body: a fixture section
  mentioning `SHA-256` plus a shipped `T-001` still reports `T-001` as a candidate.

In `internal/cli/changelog_test.go`: a test for `ticketPrefixes` (dedup + order) with a
two-child config where one child sets `ticket_prefix = "OPS"` and the other leaves it unset.

### Acceptance test

From the repo root on the feature branch:

```
just build && just test && just lint && just docs-check
```

All four green. Then reproduce the Description's measurement, in a **throwaway directory — never
against this repo** (self-modify policy, `AGENTS.md`):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q .
./pk install --project demo
git add -A && git commit -qm "chore: scaffold" && git tag v0.0.1
git commit -q --allow-empty -m "board: note the SHA-256 subject handling"
git commit -q --allow-empty -m "fix(http): handle chunked encoding (RFC-7231)"
git commit -q --allow-empty -m "board: T-001 reviewed and done, T-002 filed"
./pk changelog check
```

Expected — all three of:

1. the exclusion line names `T-001, T-002` and **not** `SHA-256`;
2. it carries `(+1 with no ticket id)` for the `SHA-256` subject — the alarm this ticket
   restores;
3. the candidate list does **not** contain `RFC-7231` (before this ticket it did, as
   `RFC-7231  (no ticket file found …)`).

Exit code is `0` throughout (decision 9): `./pk changelog check; echo $?` prints `0`.

Then prove decision 5's blind spot is a choice: in the same throwaway repo, register a second
child with an `OPS` prefix (`./pk project add ops ./ops` after `mkdir ops`, then set
`ticket_prefix = "OPS"` in `pickle.toml`), commit `feat(ops): do a thing (OPS-1)`, and confirm
`OPS-1` is now recognised as a candidate. Remove the child again and confirm it is not.

Finally, confirm this repo's own report is unchanged in substance: `go run . changelog check`
on the feature branch names the same ids it does on `main` (T-095 measured that no non-`T-`
id-shaped token exists in this history, so the only legal difference is none).

### Docs update (mandatory when user-facing)

- **`docs/user-manual/cli-reference.adoc`**, the `pickle changelog check` section — the
  paragraph beginning *"By default the exclusion list is a single summary line…"* (~line 735;
  **search the text, not the line number**, per this ticket's second History line). Amend the
  "permissive scan of the whole subject" phrasing: the scan is still whole-subject, but it
  recognises only the ticket-id prefixes registered in `pickle.toml`, so an ordinary
  `SHA-256`/`UTF-8`/`CVE-2024` in bookkeeping prose is not mistaken for a ticket — which is what
  makes the `(+N with no ticket id)` clause trustworthy. Keep the T-095 verb-phrase
  justification intact (decision 3).
- **Same section**, the *"Shipped is computed from `git log`…"* paragraph (~line 723): state
  that the trailing-bracket id must carry a registered prefix, so a subject ending in
  `(RFC-7231)` is not read as a shipped ticket.
- **Same section**, a short new note for decision 5: an id whose prefix is no longer registered
  (a removed child, or a renamed `ticket_prefix`) is invisible to this check — stated as a
  documented limit, not a bug.
- **`CHANGELOG.md`** — one entry under `[Unreleased]` → `### Fixed` (create the heading) naming
  T-097: both the fabricated-id and the silenced-alarm halves.
- **No skill-payload change.** This is CLI behaviour, not a flow rule.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (`cli-reference.adoc`, `CHANGELOG.md`).
3. Write a summary of everything done — files touched, and explicitly whether T-095 decision 2's
   verb-phrase property still holds (decision 3), since that is the property this ticket most
   risks breaking.
4. Suggest a Conventional Commit message, e.g.:

   ```
   fix(changelog): recognise only registered ticket-id prefixes (T-097)

   The id scan accepted any [A-Z][A-Z0-9]*-\d+ token, so SHA-256 was
   reported as a ticket id and silenced the "(+N with no ticket id)"
   drift alarm, and a subject ending in (RFC-7231) was reported as a
   shipped candidate no changelog entry could ever clear. All id
   patterns now share one prefix-aware predicate, fed from pickle.toml's
   registered children.
   ```

5. **Tidy up before presenting** — root-path child: interactive-rebase the WIP commits into a
   small number of atomic, correctly typed commits and keep that history.
6. Commit locally on the ticket branch. Do **not** push or open an MR without user approval.
   Present the commit message; after approval, verify the remote base is not behind
   (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing), push, and open the merge request — merging is always the human's. Hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: T-095's review, non-blocking finding N3, dispositioned
  `new ticket` because fixing it reopens T-095's locked decision 2 (the permissive `idRE` scan)
  rather than correcting an error in its implementation. Graded `medium`/`low`/`S`: impact
  raised above T-095's own `low` because this one can make the report state something false
  (a fabricated id) and suppress the drift alarm the command's docs promise, in *installed*
  projects rather than only at an edge of this one; complexity `low` and cost `S` because the
  likely fix is splitting one predicate into two in `printExclusions`/`Check`, with the
  regression tests already in place
- 2026-08-12 — patched by T-095's scoped re-review impact sweep. T-095's rework does **not**
  invalidate this ticket: the `noID` predicate it targets (`len(ex.IDs) == 0` in
  `printExclusions`) is untouched, and the measured reproduction still holds. Two notes for
  whoever picks this up: (a) the `cli-reference.adoc` paragraph this ticket quotes was reflowed
  and gained a clause about the summary's *third* branch ("no excluded subject names an id at
  all"), so the quoted promise survives but its line numbers have shifted — search the text,
  not the line; (b) the branch this ticket must not break is that same third branch, which is
  now documented, so a strict-count fix has to keep both it and the `(+N …)` clause coherent
- 2026-08-13 — TO DO → READY: plan complete: prefix-aware id predicate at every recognition site; cost S -> M
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
