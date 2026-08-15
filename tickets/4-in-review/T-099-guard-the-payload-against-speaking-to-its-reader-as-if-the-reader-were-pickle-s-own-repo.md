---
id: T-099
title: guard the payload against speaking to its reader as if the reader were pickle's own repo
project: pickle
depends-on: []
spawned-by: [T-098]
impact: medium
complexity: low
cost: S
---

# T-099 — guard the payload against speaking to its reader as if the reader were pickle's own repo

## Outcome

After this ships, a sentence that only makes sense inside pickle's own repo cannot reach the
shipped payload unnoticed: a mechanical check fails the build the way a broken link would, instead
of depending on someone remembering to run a hand sweep and reading carefully enough to catch what
the last two sweeps missed.

## Description

T-098 removed four sites where the shipped payload (`skill/`, installed into other people's
projects as `.agents/skills/brine/`) addressed its reader as though that reader were
pickle's own repo. It deliberately built **no** mechanical guard, on T-085's discipline that
machinery waits for recurrence, and pre-registered the trigger in `tickets/NOTES.md`: *file the
check the first time a second instance is found in a review.*

**That trigger has fired, and the evidence is stronger than "n=2".** T-098's refinement ran a
deliberate sweep across all five payload files and declared the payload clean at three sites. Two
more have surfaced since, each found by a different reader after the sweep. **Both are already
fixed on T-098's branch** — they are cited here as the evidence for building the check, not as
cleanup still owed. This ticket is a *regression guard*, and the two sites below are its first
two test cases:

1. **At pickup** — `tickets-README.md:320` cited `pickle`'s own `skill/resources/TEMPLATE.md`, a
   path that only resolves in this repo (installed workspaces have
   `.agents/skills/brine/resources/TEMPLATE.md`, since T-074 renamed the installed directory).
2. **At review** — `review-protocol.md:157` froze the `class` vocabulary because "the
   **pre-registered criterion** this column exists to test needs a fixed vocabulary to count
   against". That criterion lives in *this repo's* `tickets/NOTES.md`. No other project has one.

The pattern that matters is not the count but *how* both escaped: **neither is catchable by the
four `rg` patterns T-098 left behind.** Site 1 is a path, site 2 is a definite-article appeal to
evidence the reader does not have. The existing patterns look for ticket ids and the literal
strings `the corpus` / `this repo`. So a third hand sweep is the wrong instrument — it is the
instrument that already failed twice.

### What the check must catch

Beyond the two shapes T-098's patterns already cover (lookup-shaped ticket references, and
`the corpus` / `this repo`), the two that got through:

- **Repo-only paths** — `skill/…`, `internal/…`, `cmd/…`, `docs/…`, `tickets/6-done/…` and
  friends appearing inside `skill/` as if the reader could open them. The subtlety: `tickets/`
  paths *are* legitimate in the payload (every installed project has `tickets/1-to-do/`), so this
  cannot be a blanket path ban — it is specifically paths rooted in **pickle's source tree**.
- **Definite-article appeals to invisible evidence** — "the pre-registered criterion", "the
  corpus", "the 13 variants". Hardest of the four and the one most likely to need judgement
  rather than a regex; a keyword list (`pre-registered`, `the corpus`, `our own`, bare counts
  paired with evidence nouns) is probably the honest 80%.

### Design questions — closed at refinement

All three are settled in the plan's Confirmed design decisions; recorded here so the Description
does not read as still-open:

- **Where does it live?** A **Go test over the embedded payload** (`payloadFS`), at the repo root.
  `board audit` is shipped behaviour that runs in foreign projects, where policing pickle's own
  prose is meaningless; a `just` recipe was rejected because this repo's precedent is that an
  optional lint tool degrades to a *warning* when missing, which is the one failure mode a
  regression guard cannot have.
- **What is the allowlist mechanism?** Legitimate uses are matched **by shape** — a `(T-NNN)`
  provenance tag, and a `T-NNN` inside backticks as syntax filler — with a small exact-substring
  escape hatch in the test file. No file:line list (rots immediately), no count assertion (trains
  people to bump the number), and **no in-payload allow markers** (lint machinery shipped inside
  the payload is a fresh instance of the defect being guarded).
- **Does it fail or warn?** It **fails**. Breadth is bought with precision instead: the fuzziest
  rule keeps a deliberately narrow keyword list, grown only when a real escape happens.

### Couplings

Soft couplings only:

- **T-098** (`spawned-by:`) — fixed the four sites and pre-registered this trigger. Its
  `## Review` table (N7) and the `NOTES.md` entry carry the four existing `rg` patterns, so this
  ticket starts from them rather than re-deriving them.
- **T-067** (no link/anchor validation in the docs pipeline) — plausibly the same home. If T-067
  builds a docs-linting harness, this check may be a rule inside it rather than a standalone
  thing. Worth deciding together; neither blocks the other.
- **T-074** (rename the installed skill directory to brine) — **shipped and merged**, so the
  installed path is settled at `.agents/skills/brine/` and this check can hard-code against it
  without a pending rename hanging over it. The "cheaper if T-074 lands first" note is discharged.

### Scope note added at refinement: `agents/` is payload too

The ticket was written about `skill/`, but `assets.go` embeds **`skill/` and `agents/`** — the
per-agent scaffolds (`agents/opencode/opencode.jsonc`, `agents/pi/extensions/*.ts`) also land in
foreign projects and are subject to the same test. They are clean today; including them costs
almost nothing and closes the hole before something is written into them.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-099-payload-lint
```

WIP commits encouraged. **This repo is a root-path child** (`path = "."`), so the Finish step
tidies WIP into atomic commits and keeps that history rather than squashing. Ticket and board
bookkeeping stays on `main`, never on this branch.

### Prerequisite gate (hard)

None. `depends-on:` is empty and both couplings are settled: T-098 is done and merged (its four
sites fixed, its `rg` patterns recorded in `tickets/NOTES.md`), and T-074 is done and merged, so
`.agents/skills/brine/` is the final installed path. T-067 is still in `1-to-do/` and is
deliberately not waited on (decision 9).

**Verified at refinement:** all four of T-098's patterns come back clean against today's payload,
and `agents/` is clean too — so the check must be **green on `main` the moment it is written**. If
it is not, the rule is wrong, not the payload.

### Confirmed design decisions (do not deviate without asking)

1. **It is a Go test, not a `board audit` check and not a shell recipe.** `board audit` is shipped
   behaviour that runs in *foreign* projects, where policing pickle's own prose is meaningless —
   the ticket's own argument, and it is binding. A `just` recipe was rejected because
   `lint-ci-surface` establishes the precedent that an optional tool degrades to a *warning* when
   missing, which is exactly the failure mode a regression guard must not have. A Go test is run
   by `just test` and by CI with no extra dependency.
2. **It lints `payloadFS`, not the working tree.** The test lives at the repo root in
   `package main` so it can read the embedded FS from `assets.go` directly — what actually ships,
   not a directory that happens to sit beside it. Walk it with `fs.WalkDir`; lint every text file
   under both embedded roots.
3. **Both payload trees are in scope: `skill/` and `agents/`.** `agents/` installs into foreign
   projects too.
4. **Four rules ship, all of them failing** (not warning). A warning nobody must act on is the
   third hand sweep in disguise, and this ticket exists because two hand sweeps failed.
5. **Legitimate uses are recognised by *shape*, not by a file:line allowlist.** A file:line list
   rots on the next edit. Encode the two shapes `AGENTS.md` already names as legitimate:
   a **provenance tag** — a bare `(T-NNN)` parenthetical — and **syntax filler** — a `T-NNN`
   inside a backticked span or fenced block. A new legitimate use then passes without touching the
   check. A small exact-substring escape hatch (decision 6) covers anything that fits neither.
6. **The escape hatch is an exact-substring allowlist in the test file, each entry carrying a
   one-line comment saying why.** **In-payload markers (`<!-- payload-lint: allow -->`) are
   rejected outright**: pickle-internal lint machinery embedded in prose that ships to other
   people's projects is a fresh instance of the very defect being guarded against.
7. **Do not assert a *count* of surviving references.** `NOTES.md`'s third pattern ("expect exactly
   the 6 keepers") was a human's eyeball check; as an automated assertion it fails on every
   legitimate addition and trains people to bump the number. The shape rules replace it.
8. **The fuzzy rule stays narrow on purpose.** Rule 4 (appeals to invisible evidence) is a short,
   high-precision keyword list, grown when a real escape happens — never broadened by guessing.
   A rule that cries wolf gets `t.Skip`ped within a month.
9. **T-067 is not waited on.** If it later builds a docs-lint harness, these rules are data and
   move cheaply.
10. **The check does not lint `AGENTS.md`, `docs/`, `tickets/` or `CHANGELOG.md`.** Those are
    pickle's own repo, where naming pickle's own paths is correct. Only the embedded payload is
    held to the foreign-workspace test.
11. **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against this
    repo from this branch.

### Tasks

#### Task 1 — the harness

New file `payload_lint_test.go` at the repo root, `package main`.

- `TestPayloadSpeaksToAForeignReader` walks `payloadFS` with `fs.WalkDir` over both roots
  (`skill`, `agents`), reads each file, and runs every rule over it **line by line** so a finding
  reports `path:line` plus the rule name and the offending text.
- Model a rule as a small struct — name, `*regexp.Regexp`, an explanation of *why the shape fails*
  (printed on failure, so the message teaches rather than just rejects), and an optional
  `exempt func(line string) bool` for the shape-based exemptions.
- Skip nothing by extension unless it proves necessary: `.md`, `.jsonc` and `.ts` are all prose or
  prose-carrying.
- The failure message must name the file, the line number, the rule, the matched text, and point
  at `AGENTS.md`'s foreign-workspace test for the reasoning — **that pointer is legitimate here**
  because this file is repo-local, not payload.

#### Task 2 — rule 1: lookup-shaped ticket references

- Flag a `T-\d+` reference that instructs a lookup: T-098's pattern
  `tickets/6-done/T-[0-9]|T-[0-9]{3} F[0-9]` is the seed, generalised to any `\d`-length id and any
  status directory (`tickets/[1-7]-[a-z-]+/T-\d+`).
- Exempt, per decision 5:
  - `T-NNN`, `T-MMM`, `T-KKK`, `T-xxx` and friends (metasyntactic — not `\d`, so they never match
    in the first place; assert this with a test case rather than special-casing).
  - a bare provenance tag `(T-\d+)`.
  - any `T-\d+` occurring inside a backticked span or a fenced code block — syntax filler. Track
    fenced-block state across lines in the walker; for inline spans, a per-line odd/even backtick
    scan is enough and its limits should be stated in a comment.
- **Both of `tickets-README.md:63-64`'s grammar examples and `SKILL.md:304`'s `(T-083)` tag must
  pass**; verified clean at refinement, so they are the live regression cases.

#### Task 3 — rule 2: first-person repo and unattributed corpus

- `the corpus`, `this repo`, `our own`, `in this repository` (case-insensitive).
- Note the T-098 finding this replays: the `field-use`/`self-host` glosses once read "another
  project" versus "this repo's own flow", which no foreign team can assign. Cite the *shape* in the
  comment, not the ticket id — the comment lives in a test file, so an id would be fine, but the
  shape is what the next reader needs.

#### Task 4 — rule 3: repo-only paths (the shape that escaped at pickup)

- Flag path-like tokens rooted in **pickle's source tree**: `skill/`, `internal/`, `cmd/`,
  `agents/`, `docs/`, `assets.go`, `justfile`, `.goreleaser.yaml`, `.github/`.
- **`tickets/` is explicitly *not* banned** — every installed project has `tickets/1-to-do/`, and a
  blanket path ban is the trap the Description calls out. Deep ticket paths are rule 1's job.
- **Boundary care, and it is the whole difficulty of this rule.** RE2 has no lookbehind, so match
  a leading boundary character explicitly, e.g.
  `(^|[^\w./-])(skill|internal|cmd|agents|docs|\.github)/`. Two false positives to prove absent
  with test cases:
  - `.agents/skills/brine/` — must **not** match `agents/` (the `.` must block it) and must not
    match `skill/` (the directory is `skills/`).
  - `resources/TEMPLATE.md` and `tickets/README.md`, the payload's ordinary self-references —
    unaffected, since neither root is on the list.
- The correct fix for a real hit is stated in the failure message: phrase the path relative to the
  skill the reader is holding ("this skill's own `resources/TEMPLATE.md`").

#### Task 5 — rule 4: definite-article appeals to invisible evidence

- Narrow keyword list only (decision 8): `pre-registered`, `the corpus`, `our own`, and a
  bare-count-plus-evidence-noun shape such as `\d+ (variants|instances|cases|examples) (across|in) the\b`.
- Overlap with rule 2 on `the corpus`/`our own` is fine — keep the rules separately named so the
  failure message says which *kind* of defect it is; do not merge them for tidiness.
- Comment the honest limitation: this rule is the 80%, not a proof, and the sentence-level judgement
  still belongs to the reviewer.

#### Task 6 — negative tests: prove the rules catch the two real escapes

A table test over **synthetic strings**, not files — the point is that these two would have been
caught, and they are the ticket's stated first two test cases:

1. `` see `skill/resources/TEMPLATE.md` for the shape `` → caught by rule 3.
2. `the **pre-registered criterion** this column exists to test` → caught by rule 4.

Plus a positive table asserting the legitimate shapes pass: `(T-083)`, `` `board: T-084 ready → in
development` ``, `.agents/skills/brine/resources/TEMPLATE.md`, `tickets/1-to-do/`, `T-NNN`.

#### Task 7 — retire the superseded note in `NOTES.md`

`tickets/NOTES.md` (the T-098 section, ~`:908-945`) records the pre-registered trigger, the four
`rg` patterns, and "a fifth shape escapes all four and needs the eye". Once this check exists that
passage is history, not instruction — and its `.agents/skills/ticket-flow/` reference at `:944` is
also stale after T-074.

- Keep the section as the record of what happened; append a short closing note that the trigger
  fired, T-099 built the check, and the patterns now live in `payload_lint_test.go` as executable
  rules.
- **This edit is bookkeeping, so it is committed on `main`, not on this branch** (rules §0) — do it
  as a separate step at Finish, not in a branch commit. The `pre-commit` guard will refuse it
  otherwise, correctly.

### Acceptance test

```
just build
just test          # includes the new TestPayloadSpeaksToAForeignReader
just lint
just docs-check
```

Then prove the guard actually guards, rather than merely passing:

```
# 1. green on an untouched payload
go test -run TestPayloadSpeaksToAForeignReader ./...     # PASS

# 2. re-introduce escape 1 (repo-only path) and confirm it fails
#    append to skill/resources/tickets-README.md:  see `skill/resources/TEMPLATE.md`
go test -run TestPayloadSpeaksToAForeignReader ./...     # FAIL, naming file:line + rule 3
git checkout skill/resources/tickets-README.md

# 3. re-introduce escape 2 (invisible evidence) and confirm it fails
#    append to skill/resources/review-protocol.md:  the pre-registered criterion
go test -run TestPayloadSpeaksToAForeignReader ./...     # FAIL, naming file:line + rule 4
git checkout skill/resources/review-protocol.md

# 4. the over-correction guard: the six legitimate references still pass (step 1 covers this,
#    but confirm the positive table in Task 6 is what asserts it, not luck)
```

Both injected failures must name the file, the line, and the rule — a failure that only says
"payload lint failed" does not meet this step.

### Docs update (mandatory when user-facing)

No user-facing surface: this is a repo-local test that ships in no binary and changes no command.
Three internal touches instead:

- **`AGENTS.md`** — the foreign-workspace-test paragraph (above the marker block, so hand-edited
  and outside `pickle upgrade`'s reach) currently reads as a rule a human applies. Add one sentence
  saying `payload_lint_test.go` now enforces the mechanical part, and that the paragraph remains
  the authority for the judgement the test cannot make. Keep it short; do not restate the rules.
- **`tickets/NOTES.md`** — Task 7, committed on `main`.
- **`CHANGELOG.md`** — nothing. No user-visible change; do not add an entry.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` all clean.
2. `AGENTS.md` updated as above (on the branch — it is not `tickets/` bookkeeping).
3. Write a **summary** of everything done (files touched, decisions made, anything deferred —
   in particular any rule considered and left out of the narrow keyword list).
4. Suggest a **Conventional Commit message**, e.g.:

   ```
   test(payload): fail the build on prose that only makes sense in pickle's repo (T-099)

   <body — what and why>
   ```

5. **Tidy up before presenting** — root-path child, so interactive-rebase WIP into atomic commits
   (a natural split: the harness + rules; the tests; the `AGENTS.md` note). Keep that history.
6. Commit locally on the ticket branch. Do **not** push or open a merge request without explicit
   user approval. Commit the `tickets/NOTES.md` edit **separately on `main`** with an explicit
   pathspec. After approval, verify the remote base is not behind
   (`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must
   print nothing), then push and open the merge request — merging is the human's. Then
   `pickle ticket move T-099 in-review --reason "acceptance green"` and hand back.

## Review

Reviewed 2026-08-14 on `feat/T-099-payload-lint` (3 commits: `d1738c2`, `a20d690`, `1b9c256`).
Ticket read from `main` — the branch was cut before the move to `4-in-review/` landed, so its
worktree copy still shows the ticket in `3-in-development/`.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on `AGENTS.md` (step 4b) — run; one suggestion touching this
      ticket's own added sentence, folded into N3. The rest addressed prose this ticket did not
      touch and was discarded as out of scope.
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `1-to-do/`/`2-ready/` names
      T-099 and none `depends-on:`/`spawned-by:` it. One live touch: **T-067** already reasons
      about "where it lives" from the existing payload guards and `payloadRoot()` (`:50`,
      `:61-62`), and `payload_lint_test.go` is a third, different precedent — repo root,
      `package main`, reading `payloadFS` directly. Patching T-067 is **deliberately deferred to
      the re-review**: F1 and F2 both change this file's rules, and citing it now would mint the
      stale cross-reference this very ticket exists to prevent.
- [ ] Summary + commit message presented for approval (step 9) — deferred to the re-review

**Acceptance test: green, verbatim.** `just build`, `just test`, `just lint`, `just docs-check`
all clean. Both injected escapes fail as specified and each failure names file, line, rule and
matched text: escape 1 → `skill/resources/tickets-README.md:485: [repo-only-path]`, escape 2 →
`skill/resources/review-protocol.md:285: [invisible-evidence]`. The over-correction guard
(step 4) is asserted by `TestPayloadLintRulesLeaveLegitimateShapesAlone`, not by luck.

All eleven confirmed design decisions were honoured in structure: a Go test over `payloadFS`
(1, 2), both roots linted (3), four failing rules (4), shape-based exemptions with an
exact-substring hatch and no in-payload markers (5, 6), no count assertion (7), a narrow rule 4
(8), T-067 not waited on (9), scope limited to the payload (10), no self-install (11). The two
blocking findings below are defects *inside* that structure, not departures from it — except F1,
which also widens decision 5's sanctioned exemption without asking.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | `isProvenanceTag` exempts any rule-1 match immediately followed by `)`, punching a hole through rule 1's own target shape while exempting nothing legitimate — rule 1's pattern never matches a bare `(T-083)` in the first place | `payload_lint_test.go:74-81,93`; probe below | Drop `isProvenanceTag` from `ticketRefExempt`; flip the test at `:357` to assert the line is flagged |
| F2 | blocking | correctness | — | Rule 3's `\b(assets\.go\|justfile\|\.goreleaser\.yaml)\b` alternative is dead for `.goreleaser.yaml`: the leading `\b` can only match when a word character precedes the `.`, which never happens in prose | `payload_lint_test.go:152`; probe below | Move `.goreleaser.yaml` into the boundary-prefixed alternation (`(^\|[^\w./-])\.goreleaser\.yaml\b`) or drop the leading `\b`; add a table case |
| N1 | non-blocking | design | note and close | `isEscapeHatched(line)` suppresses **all four** rules on a matching line, not the one rule the entry was added for — a future entry silently blinds the line to unrelated defects | `payload_lint_test.go:197`; probe below | Make the hatch per-rule (`rule.name + entry`) if an entry is ever added; harmless while `escapeHatch` is empty |
| N2 | non-blocking | test-gap | note and close | `escapeHatch` / `isEscapeHatched` ship with no test — being empty, `isEscapeHatched` returns `false` on every line the suite runs, so the mechanism is entirely unexercised | `payload_lint_test.go:96-110` | A two-line test setting `escapeHatch` and asserting suppression would pin N1's behaviour too |
| N3 | non-blocking | spec-unclear | fix inline | The `AGENTS.md` sentence this ticket adds says the test "cannot judge a sentence's shape, only match one of its four rules" — but the rules *are* shape-based by decision 5, so "shape" carries two contradictory meanings in one paragraph | `AGENTS.md:52-56` | "cannot judge a sentence's *meaning*"; sentence also split per the step-4b suggestion |

disposition summary: 2 blocking (F1, F2 — both in `payload_lint_test.go`, both regex-shaped); 3
non-blocking — 2 note-and-close (N1, N2), 1 fix-inline (N3), 0 new tickets, 0 absorbed.

cost: estimated S, actual S

### Probe evidence (synthetic lines through `lintFile` with the shipped rules)

```
"the finding reference (T-090 F1)"                  -> []                        # F1: escapes
"(see tickets/6-done/T-090)"                        -> []                        # F1: escapes,
                                                                                 #     and :357 asserts this pass is correct
"see the worked example (tickets/6-done/T-090 F1)"  -> [ticket-lookup]           #     caught only because ')' is not adjacent
"edit .goreleaser.yaml to add the target"           -> []                        # F2: dead alternative
"the release config is .goreleaser.yaml"            -> []                        # F2: dead alternative
"run just build then edit justfile"                 -> [repo-only-path]          #     sibling tokens work
"the payload is embedded by assets.go"              -> [repo-only-path]          #     sibling tokens work
escapeHatch=["KEEPME"], line with a repo-only path + "the corpus" -> 0 findings  # N1: 3 suppressed
rule 1 pattern vs "(T-083)", "(`## Outcome`, T-083)"             -> no match     # F1: exemption guards nothing
```

F1 matters because the shape it lets through is the one T-098 actually shipped: its review
protocol's worked examples were finding references of exactly the `T-NNN FN` form, and a worked
example is very naturally parenthesised. The plan's Task 2 and decision 5 sanctioned exempting *"a
bare provenance tag `(T-\d+)`"* — the whole parenthetical being only the id. The implementation
knowingly widened that to "immediately followed by `)`" (the comment at `:74-79` says so), which
decision 5's "do not deviate without asking" reserved for the user. The widening is also
unnecessary: the bare tags it cites as justification are unmatched by rule 1's pattern, so
removing the exemption costs no legitimate reference. The live payload has no instance today, so
this is a guard gap, not a shipped defect.

No other issues found. The consistency sweep turned up no stale cross-references: `NOTES.md`'s
T-098 passage is correctly retired to history on `main`, the four `rg` patterns survive only
inside `6-done/T-098`'s own record (where they are history, not instruction), and no doc claims
the payload is hand-swept. Docs coverage is correct as planned — no user-facing surface, no
`CHANGELOG.md` entry, `AGENTS.md` updated. The commit history is three atomic, correctly scoped
commits, as the root-path finish step requires.

### Fixed at rework (2026-08-15, `bd459ce` on `feat/T-099-payload-lint`)

- **F1 — fixed exactly as suggested.** `isProvenanceTag` removed outright; `ticketRefExempt` now
  carries only the backtick/fence exemption. The test previously named at `:357` ("a closing
  paren treats the reference as a provenance tag", asserting the escape passed) is flipped to
  assert both `"(see tickets/6-done/T-090)"` and `"the finding reference (T-090 F1)"` are now
  flagged, plus a companion case confirming a genuinely bare tag (`"(T-083)"`,
  `"(`## Outcome`, T-083)"`) still passes for the reason the finding gave — the pattern never
  touches it. Re-verified live against `payloadFS`, not just synthetic strings: appending
  `"as documented (tickets/6-done/T-090)"` to `tickets-README.md` now fails
  `TestPayloadSpeaksToAForeignReader` naming `[ticket-lookup]`, where before this fix it passed
  silently.
- **F2 — fixed exactly as suggested.** `.goreleaser.yaml` moved out of the plain-`\b`-bounded
  alternation into its own `(^|[^\w./-])\.goreleaser\.yaml\b` group, sharing the explicit
  leading-boundary approach the directory alternatives already use. New
  `TestPayloadLintRule3RepoOnlyPaths` covers the three previously-dead phrasings, confirms
  `assets.go`/`justfile` (its word-initial siblings) still fire, and confirms
  `.agents/skills/brine/resources/TEMPLATE.md` still passes (the boundary case the rule exists
  to get right).
- **N3 — applied during the review itself** (fix-inline, `665fbb0`, already on the branch before
  this rework pass): "cannot judge a sentence's shape" → "cannot judge what a sentence *means*",
  with the sentence split per the step-4b suggestion.
- **N1, N2 — left as recorded** (note and close): the escape hatch is unchanged and still
  untested; harmless while `escapeHatch` is empty, per the original disposition.

Re-ran the full acceptance test after both fixes: `just build`, `just test`, `just lint`,
`just docs-check` all clean; all three injections (escape 1, escape 2, and F1's own escape
replayed live) fail naming file, line, rule and matched text, and the untouched payload is green.
No new findings surfaced fixing these two — both changes are confined to `payload_lint_test.go`.

## History

- 2026-08-13 — created (TO DO). source: review: T-098's review, finding N7, disposition *new
  ticket*. Filed against a **pre-registered trigger** rather than a fresh judgement call: T-098
  recorded in `tickets/NOTES.md` that a mechanical guard would be built the first time a second
  instance reached a review, deliberately declining to build one at n=1. Two instances have since
  escaped T-098's own hand sweep — a repo-only path found at pickup, and an appeal to a
  pre-registered criterion the reader does not have, found at review — and neither is catchable
  by the four `rg` patterns that sweep left behind, which is the actual argument for machinery
  over a third sweep
- 2026-08-13 — patched by T-098's impact sweep (review step 8): T-098 shipped, so both cited
  instances are now fixed on its branch and the line numbers above (`tickets-README.md:320`,
  `review-protocol.md:157`) refer to the pre-fix text — read them as test cases for the check,
  not as outstanding cleanup. Scope and design questions are otherwise unchanged; nothing here
  was re-graded
- 2026-08-13 — patched by **T-074's review impact sweep**. T-074 renames the installed skill
  directory `ticket-flow` → `brine`, so two live claims in this ticket's Description are about to
  be false: the parenthetical at `:24` ("installed into other people's projects as
  `.agents/skills/ticket-flow/`") and, more importantly, test case 1 at `:38`, whose whole point
  is the contrast between pickle's own `skill/resources/TEMPLATE.md` and the path an installed
  workspace actually has — which becomes `.agents/skills/brine/resources/TEMPLATE.md`. The test
  case is *not* invalidated (the defect shape is unchanged: a path that resolves only in this
  repo); only the correct right-hand side moves. The "Cheaper if T-074 lands first" note at
  `:85-87` still holds and is now nearly settled — T-074 is in `5-rework/` as of this sweep, with
  one blocking finding that touches no path. Refinement should re-read the payload for the new
  name rather than trusting these strings
- 2026-08-14 — refined. All three design questions closed: a **Go test over `payloadFS`** (not a
  `board audit` check, which runs in foreign projects; not a `just` recipe, because this repo's own
  precedent lets an optional lint tool degrade to a warning — the one failure mode a regression
  guard cannot have); **shape-based** exemptions rather than a file:line list, a count assertion or
  in-payload allow markers; and it **fails**, with the fuzzy rule kept narrow instead. Scope grew by
  one item found at refinement: `assets.go` embeds `agents/` as well as `skill/`, so the per-agent
  scaffolds are payload too and are linted alongside. Verified the payload is clean today — all four
  T-098 patterns and `agents/` — so the check must be green on `main` when written. Both stale
  `ticket-flow` strings the T-074 sweep flagged are corrected, and that coupling is discharged
  (T-074 is done and merged). Nothing split out; T-067 deliberately not waited on. Grades held at
  medium/low/S — the extra tree and the boundary care in rule 3 are absorbed by the S the two
  escapes already justified
- 2026-08-14 — TO DO → READY: plan complete
- 2026-08-14 — READY → IN DEVELOPMENT: picked up
- 2026-08-14 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-14 — IN REVIEW → REWORK: 2 blocking: rule 1's provenance exemption opens a hole over its own target shape (F1); rule 3's .goreleaser.yaml alternative is dead (F2)
- 2026-08-15 — REWORK → IN REVIEW: findings fixed
