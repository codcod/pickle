---
id: T-083
title: a ticket does not say what you get from it: add an Outcome section and warn when it is missing
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: M
---

# T-083 — a ticket does not say what you get from it: add an Outcome section and warn when it is missing

## Outcome

After this ships, you can decide whether a TO DO ticket is worth refining by reading its first
two lines, instead of reconstructing the payoff from a mechanism narrative written for whoever
files it. Every ticket states, up front and in user-observable terms, what changes when it
lands — and `board audit` tells you which tickets still do not.

## Description

*(This Description is evidence and options, per the house style — the Outcome above is the
feature itself.)*

### The problem, measured

`TEMPLATE.md:20` gives the Description exactly one instruction — *"The current spec of the
feature, in prose"* — and says nothing about stating a benefit. Nothing downstream asks either.
The result, counted across all 23 tickets in `1-to-do/` by whether sentence 1–2 tells a reader
what they *get*:

| opening shape | count | tickets |
|---|---|---|
| outcome / symptom first | 8 | T-051, T-052, T-055, T-065, T-067, T-077, T-078, T-079 |
| mechanism or provenance first | 9 | T-013, T-038, T-042, T-046, T-070, T-071, T-074, T-076, T-080 |
| mixed / roadmap | 6 | T-050, T-056, T-066, T-075, T-081, T-082 |

**The split correlates with provenance**, which is the actionable part: tickets born from
reviews and audits open with lineage and mechanism (*"Filed by the T-043 review (finding R2)…
It left the fourth reader behind"* — T-070), while tickets born from field use open with the
symptom (*"the documented onboarding sequence ends in an error and a non-zero exit, for a
workspace where nothing is wrong"* — T-052). T-070 is the clearest cost: its actual payoff — a
wrapped merge line is silently truncated in the board's DONE `merged` cell and in `serve`'s
ticket view — sits roughly 20 lines below the fold. T-077 is the best counter-example in the
tree and shows the fix is free: *"the reason the whole family is worth building: reviewing a
400-line solution design in a terminal, before approving it, is miserable."*

### The finding that makes this cheap: the sentence already gets written, into the wrong file

The two `impact` recalibration passes (`NOTES.md`, 2026-08-01 and 2026-08-03) produced **11
one-line justifications** that are exactly the missing text — and left every one of them in a
`NOTES.md` table instead of in the ticket:

| ticket | the sentence that answers "what do I get" | where it lives today |
|---|---|---|
| T-055 | "cosmetic CSS specificity bug" | `NOTES.md` only |
| T-038 | "narrow input hardening on a path that already rejects the dangerous cases" | `NOTES.md` only |
| T-057 | "the only open item guarding against **silent loss** … it has happened three times" | `NOTES.md` only |

Verified: neither `T-055` nor `T-038` mentions its impact rationale anywhere in the ticket file.
A reader opens the ticket, sees `impact: low`, and gets no reason — while a well-written one
sits one directory away. T-019 is the counter-example proving it is writable inline: *"which is
why the impact moves `low` → `low-medium`"*.

So the raw material exists and the authoring habit exists. What is missing is a **home for it in
the ticket** and anything that notices its absence.

### Item 1 — the Outcome lead (payload prose)

Give the ticket a place to say what it buys, and make grading write its reason there:

- `TEMPLATE.md` — an `## Outcome` section (1–3 sentences, user-observable terms, "after this
  ships, X"), placed **above** `## Description`, with the placeholder wording and a worked
  example.
- `tickets-README.md` §1/§3 — define the section, and extend §3's standing *"assess every new
  ticket against the existing backlog … and re-grade the board"* mandate so a grade's **reason
  lands in the ticket**, not only in a triage table.
- `SKILL.md` *make it a ticket* / *refine a ticket* — one clause each.

This is payload text: it ships to every installed workspace and bumps `payload_version`.

### Item 2 — audit the section's presence, as a warning only (Go)

A rule nothing checks is the bet this repo has already lost twice (`NOTES.md:100-101`; T-064's
verdict was *"the rule already exists and is not being followed"*). So Item 1 ships with a check:
`board audit` reports a ticket whose `## Outcome` is absent, empty, or still the template
placeholder.

**It must be `warnf`, never `errf`, and this is a hard constraint rather than a preference.**
`audit.Audit` runs as a post-op self-check at **six** call sites — `move.go:142`,
`cli/board.go:76`, `cli/install.go:89`, `cli/install.go:165`, `sync.go:73` and
`serve/view.go:299`. An error would make `ticket move`, `board sync`, `install`, `upgrade` and
the dashboard's health panel all fail for the 81 tickets that predate the section — including
the moves that would drain the backlog. For the same reason `Outcome` must **not** join
`requiredKeys` (`audit.go:27`); that list is frontmatter-only, and T-045 documented the same
migration break (`upgrade` never touches tickets, so every installed workspace would light up).

**Check structure, not prose.** A heuristic "does this sentence sound like a benefit" check
would be the T-050 defect shape, and this repo's one shipped guardrail has a recorded field
record of **1 false positive, 0 true positives**. Presence-and-not-placeholder is mechanical and
has no judgement in it.

### Why this is not T-045, T-063 or T-064

The theme has a graveyard and `NOTES.md:100-101` requires any new entrant to answer for it:

- **T-063** (value-per-cost ordering) and **T-045 valve 3** (backlog cap) changed how the board
  **sorts**. This changes what a ticket **says**; it adds no ordering, no score and no config.
  The standing "the pickup queue is READY, not TO DO" objection does not bite — the 2026-08-03
  pass explicitly scopes TO DO to *"what gets **refined** next"*, which is the exact activity
  this ticket serves.
- **T-064** proposed a **merit gate** that judges whether a ticket deserves to exist, and died
  because *"a checklist item ('is this worth it? yes') … launders the omission"*. `## Outcome`
  is **descriptive, not evaluative**: it states what changes, and makes no worth claim. Nothing
  is gated, nothing is blocked, no verdict is recorded.
- **T-045 valve 4** was a `user-visible:` **frontmatter axis**, dropped for *"recalibrate
  `impact` instead of adding an axis"*. This adds no key and no axis — it gives the
  recalibration's own prose a place to live, which is the cheaper thing that pass recommended.

The honest residual risk: Item 1 alone is unenforceable prose, which is what failed before.
Item 2 exists because of that, and the two should be weighed as a pair.

### Decisions confirmed at refinement (2026-08-07)

1. **Section, not a lead sentence.** `## Outcome` is a real, checkable section — the only form
   Item 2 can audit without a prose heuristic.
2. **Does not wait for T-081.** Ship the ~25-line warning now; the coupling to T-081's gate
   table stays **soft** (recorded below, in T-081's own Description). Re-verified against the
   current tree at refinement: T-081 is still `depends-on: [T-080]`, itself un-started, so gating
   this ticket's cheap warning behind that programme would cost more than it saves.
3. **Warning scope is non-terminal tickets only** (`1-to-do/` … `5-rework/`). `6-done/` and
   `7-dropped/` are permanent, immutable archives (rules §3) — auditing them would add ~62
   permanent warnings here and a comparable pile in every installed workspace, for records
   nobody is about to act on.
4. **Backfill the 25 open tickets** (re-verified count at refinement — the Description's "23"
   is now stale; T-084…T-087 were filed after this ticket). The 11 `NOTES.md` sentences make a
   visible chunk of it near-free. `6-done/`/`7-dropped/` are left alone. Backfilling is a
   **bookkeeping edit to ticket files** — per rules §0 it is committed on the base branch, not
   on this ticket's `feat/` branch, and is Task 5 below, done after the code lands.
5. **"Missing" is structural, not judged.** The section is missing when it is **absent**, or
   its body — after stripping HTML comments — is empty/whitespace-only. No prose heuristic (the
   Description's own case against one, T-050's shape, stands). No `requiredKeys` entry (T-045's
   migration-break precedent) — this stays a body-section check, exactly like the existing
   `## History` presence check it sits beside in `audit.go`.
6. **Does not reach the board or `serve`.** No board cell, so no cell-length cap (T-049) is
   needed. `serve`'s ticket view already renders the ticket's full markdown body, so the section
   shows up there for free with zero new code.
7. **Never blocking.** `warnf` only, at every one of the six `audit.Audit` call sites
   (`move.go:142`, `cli/board.go:76`, `cli/install.go:91`, `cli/install.go:180`, `sync.go:73`,
   `serve/view.go:299` — re-verified line numbers at refinement; two moved since filing) — none
   of them may fail for the 25 tickets that predate the section.

### Soft couplings

- **T-081** — gate table as data; overlaps Item 2's mechanism. See Q2.
- **T-074** — renames the installed skill directory and rewrites payload text; both touch
  `TEMPLATE.md`/`tickets-README.md`. Sequence, do not run concurrently.
- **T-022** (done) — the precedent for payload-prose-only tickets, and the source of the
  "another advisory prose block is the bet that just lost" objection Item 2 answers.
- **T-066** — CLI-surface documentation drift; a new `board audit` warning needs a line in
  `docs/user-manual/cli-reference.adoc`, which is that ticket's territory.
- **T-045**, **T-063**, **T-064** (all dropped) — the theme's prior art, addressed above.

### The case against, and what would falsify it

Demand is **n = 1** — the requester's own friction, the same provenance as T-059, T-060 and
T-062, the three recorded wastes (`NOTES.md:87-90`). No ticket has been picked up wrongly, or
skipped, for lack of an Outcome line; the cost is reading time, which nobody has measured. What
is stronger here than in those three is that the defect is **counted** (9 of 23) and the fix's
raw material is **already written** (11 sentences in `NOTES.md`).

**Pre-registered kill criterion**, in the T-045 style: if `## Outcome` ships and the next
refinement pass over `1-to-do/` is not measurably faster to triage — or if the sections are
being filled with restated titles — **drop the audit check and revert to a lead-sentence
convention**. A section that gets padded is worse than no section, and the padding will be
visible in the same scan that produced the table above.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle` is a single-repo child (`project: pickle`, path `.`) — the branch is cut in this same
repo:

```
git checkout main
git checkout -b feat/T-083-outcome-section
```

WIP commits on this branch as work proceeds; never push or open an MR without explicit user
approval (rules §0/§4). Task 5 (backfilling the 25 open tickets' `## Outcome` sections) is
bookkeeping, not code — it is committed on `main`, never on this branch (rules §0); do it after
the code half is reviewed-ready, as its own commit(s) with explicit pathspecs.

### Prerequisite gate

None. `depends-on: []`; T-081's overlap is a soft coupling only (see Description Q2 /
Decision 2).

### Confirmed design decisions (do not deviate without asking)

1. `## Outcome` is a new top-level body section, placed **above** `## Description` (so the
   section order becomes Outcome, Description, Implementation Plan, Review, History).
2. "Missing" = the section is absent from the ticket, **or** its body is empty/whitespace-only
   once HTML comments (`<!-- ... -->`) are stripped. No prose-quality heuristic.
3. The check is a `board audit` **warning**, never an error, and never gates `ticket move`
   (mirrors the existing `no parseable status line in ## History` warning in `audit.go`, not
   the History-status **error** below it).
4. Scope is the five **non-terminal** status directories (`1-to-do/`, `2-ready/`,
   `3-in-development/`, `4-in-review/`, `5-rework/`). `6-done/`/`7-dropped/` are never checked.
5. `Outcome` is **not** added to `requiredKeys`/`optionalKeys` in `internal/audit/audit.go` —
   those are frontmatter-only; this is a body-section check, same family as the existing
   `## History` presence/shape checks.
6. No board or `serve`-specific rendering change: `serve`'s ticket view already renders the
   full ticket markdown, so `## Outcome` shows up there automatically.

### Tasks

#### Task 1 — `internal/ticket/ticket.go`: a reusable section-body reader

Add, near `SectionHeadings` (`ticket.go:543`):

```go
// SectionBody returns the trimmed body text of the top-level ("## <heading>")
// section named heading, and whether that section exists at all. Matching is
// exact and case-sensitive, mirroring SectionHeadings. Unlike SectionHeadings
// (which only lists names) this reads content, for callers that must judge
// whether a section is *empty* (T-083's Outcome-presence check) rather than
// merely present.
func SectionBody(text, heading string) (body string, found bool)

// OutcomeMissing reports whether text's "## Outcome" section is absent, or its
// body is empty once HTML comments (the TEMPLATE.md placeholder form) are
// stripped (T-083). Used by board audit as a warning, never a gate.
func OutcomeMissing(text string) bool
```

`SectionBody` walks lines like `SectionHeadings`, capturing everything between the matching
`## <heading>` line and the next `## ` line (or EOF). `OutcomeMissing` calls it with
`"Outcome"`, strips `<!--...-->` (a package-level `regexp.MustCompile("(?s)<!--.*?-->")`,
named e.g. `htmlCommentRE`), trims whitespace, and reports whether the result is empty.

Unit tests in `internal/ticket/ticket_test.go`: `SectionBody` on a heading present with a
body / present-and-empty / absent; `OutcomeMissing` true for absent, true for
comment-only-body, true for whitespace-only, false for real prose, false when other sections'
content could be mistaken for Outcome's (a body containing the literal text `## Outcome`
inside e.g. a fenced code block — confirm the walk is line-prefix-based like
`SectionHeadings` already is, so this is a non-issue, but assert it).

#### Task 2 — `internal/audit/audit.go`: the warning

In the per-ticket loop (the same one that runs the History checks, `audit.go:~172`), add —
scoped to non-terminal tickets only:

```go
if t.Dir != "6-done" && t.Dir != "7-dropped" {
    if ticket.OutcomeMissing(t.Text) {
        r.warnf("%s: ## Outcome is missing, empty, or still a placeholder — say what changes "+
            "when this ships, in user-observable terms", ref)
    }
}
```

Place it in the existing per-ticket loop right after the frontmatter/grade checks (the same
loop already guarded per-ticket by `ref`), not the separate History loop — it needs no History
parsing. Add a doc comment above it explaining the terminal-directory exclusion (rules §3's
archive-immutability point) and that it is a warning by hard constraint (mirroring the
Description's "must be `warnf`, never `errf`" note), so a future editor does not "fix" it into
an error.

Tests in `internal/audit/audit_test.go`: a fixture ticket in `1-to-do/` with no `## Outcome` →
warning present; one with `## Outcome\n\n<!-- TODO -->` → warning present; one with a real
sentence → no warning; one in `6-done/` with no `## Outcome` → no warning (archive exclusion);
one in `5-rework/` missing it → warning present (non-terminal, not just `1-to-do/`).

#### Task 3 — `skill/resources/TEMPLATE.md`: the section itself

Add a new `## Outcome` section directly above `## Description`, with placeholder wording and a
worked example (paraphrased on T-083's own Description lead), e.g.:

```markdown
## Outcome

<!-- 1–3 sentences, in user-observable terms: "After this ships, <who> can <do/see what>."
Descriptive, not evaluative — state what changes, not whether it was worth it. Example:
"After this ships, you can decide whether a TO DO ticket is worth refining by reading its
first two lines, instead of reconstructing the payoff from a mechanism narrative." -->

## Description
```

#### Task 4 — `internal/ticket/ticket.go`: `Scaffold` + template-drift tests

`Scaffold` (`ticket.go:577`) must render the same section skeleton, so a freshly-`ticket new`'d
ticket audits with the same (expected) Outcome warning as any other unrefined ticket rather
than silently disagreeing with `TEMPLATE.md`. Add, right after the `# %s — %s` heading and
before `## Description`:

```
## Outcome

<!-- TODO: 1-3 sentences, in user-observable terms: what changes when this ships. -->

```

`TestScaffoldSectionsMatchTemplate` (`internal/ticket/ticket_test.go:511`) already asserts
`Scaffold`'s `## ` heading set equals `TEMPLATE.md`'s — it fails until both are updated
together, and passing it is this task's acceptance signal (no new test needed here beyond the
existing one going green).

#### Task 5 — Backfill `## Outcome` into the 25 open tickets (bookkeeping, on `main`)

Not on this feature branch (Decision 4 / Task 0). After Tasks 1–4 are code-complete on the
branch: on `main`, add a real `## Outcome` section (above `## Description`) to every ticket
currently in `1-to-do/`. Reuse the 11 stranded `NOTES.md` sentences verbatim/lightly-edited
where they exist (at least T-055, T-038, T-057 per the Description's table — re-check
`NOTES.md`'s two recalibration tables for the full set of 11 at write time, since this ticket's
Description only quotes three); write a fresh 1–3 sentence Outcome for the rest, in
user-observable terms, from each ticket's own Description. Do **not** touch `6-done/` or
`7-dropped/` (Decision 3). Commit with explicit pathspecs (`git add tickets/1-to-do/T-*.md`),
never `-A`/`.`.

### Acceptance test

1. `just build` — compiles clean.
2. `just test` (`go test ./...`) — all green, including the new `SectionBody`/`OutcomeMissing`
   unit tests, the new `audit_test.go` cases, and the existing `TestScaffoldSectionsMatchTemplate`
   / `TestFrontmatterKeysMatchTemplate` (still green — Outcome is a body section, not
   frontmatter, so the latter is untouched).
3. `just lint` — `go vet ./...` clean, `gofmt -l .` empty.
4. `just docs-check` — `snowball check` clean after the docs update (Docs update, below).
5. Manual: `./pickle board audit` on this repo, on the feature branch, **before** Task 5's
   backfill — confirm every one of the 24 tickets in `1-to-do/`, plus T-083 itself (now in
   `2-ready/`, pending its own Outcome), produces exactly one new `Outcome is missing` warning
   and zero new errors; confirm no `6-done/`/`7-dropped/` ticket produces one.
6. Manual, after Task 5 lands on `main` and is merged into the branch's view (or re-run on
   `main` directly): `./pickle board audit` reports **zero** `Outcome is missing` warnings
   across `1-to-do/` and `2-ready/` (`3-in-development/`–`5-rework/` are empty on this repo
   today, so nothing to check there, but the code path is exercised by the audit_test.go
   fixtures from Task 2).

### Docs update (mandatory — user-facing)

- `skill/resources/tickets-README.md` — §3's grading mandate: extend *"Assess every new
  ticket against the existing backlog… and re-grade the board"* so a grade's one-line reason is
  expected to land in the ticket's own `## Outcome` (or Description), not only in a
  `NOTES.md` triage table. §7 ("Ticket structure"): add `## Outcome` as the new first body
  section in the list, one sentence describing it (descriptive, not evaluative; checked by
  `board audit` as a warning only).
- `skill/SKILL.md` — one clause each in *Procedure: make it a ticket* (step 3, after scaffolding:
  fill in `## Outcome` alongside the Description) and *Procedure: refine a ticket* (step 2: also
  confirm/tighten the Outcome section against the current target child-project).
- `docs/user-manual/concepts/tickets.adoc` — "Body" section (`:80`): the section list is
  currently numbered 1–4 (Description, Implementation Plan, Review, History); add Outcome as
  the new item 1, renumbering the rest.
- `docs/user-manual/cli-reference.adoc` — `pickle board audit` (`#cmd-board-audit`, `:497`):
  add one bullet to the invariant list for the new warning, matching the existing bullets'
  style (e.g. next to the History-line-length warning bullet).
- `CHANGELOG.md` — `## [Unreleased]` — `### Added`: one entry, ticket id in parens, describing
  the `## Outcome` section and the new `board audit` warning.

### Finish (mandatory)

1. Acceptance test green (above); `just build`/`test`/`lint`/`docs-check` clean.
2. Docs updated and registered (above).
3. Summary: files touched, the Item-1/Item-2 split, the T-081 soft coupling left as-is, the
   backfill done separately on `main` (Task 5), and the pre-registered kill criterion from the
   Description (drop the audit check if the next `1-to-do/` triage pass is not measurably
   faster, or if sections get padded with restated titles) carried into T-083's History for a
   future reviewer to check against.
4. Suggested commit message:

   ```
   feat(tickets): add an Outcome section and a board-audit presence warning (T-083)

   Every ticket gets a ## Outcome section (above ## Description) stating, in
   user-observable terms, what changes when it ships. `board audit` warns
   (never errors) when a non-terminal ticket's Outcome is absent, empty, or
   still the template placeholder.
   ```

5. Commit locally on `feat/T-083-outcome-section`. Do not push / open an MR without explicit
   user approval (rules §0/§4). Before pushing: `git fetch origin main && git diff --name-only
   origin/main...HEAD | grep '^tickets/'` must print nothing (Task 5's backfill living on
   `main`, not this branch, is what keeps that true). Hand back to the user.

## Review

### Review 1 (2026-08-09) — verdict: REWORK (1 blocking finding)

Checklist:

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — run; every
      suggestion landed on prose this ticket did not touch, none on the new Outcome text,
      so all were discarded as out of the ticket's diff (they are not findings)
- [x] Findings recorded with severity **and** disposition; disposition summary present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Commit message / MR attributes — deferred: publishing waits for the rework pass (step 9)

Commands re-run verbatim on `feat/T-083-outcome-section`: `just build` ✅ · `just test` ✅ (all
packages) · `just lint` ✅ (`go vet` + `gofmt -l` empty) · `just docs-check` ✅ (`snowball check`)
· `./pickle board audit` → `87 tickets, 0 error(s), 0 warning(s)` ✅ (Task 5's backfill landed
first, so acceptance step 6 is what is observable now; step 5's pre-backfill state was verified
indirectly — see N-A). End-to-end in a throwaway install (`pk install` → `pk ticket new` →
`pk board audit`): exactly one `## Outcome is missing …` warning, exit 0, and `pk ticket move`
succeeds while printing the warning — Decision 7 ("never blocking") holds in the shipped binary.

Backfill quality (Task 5, 25 tickets): sampled all 24 in `1-to-do/` — every Outcome is a real,
user-observable sentence, none is a restated title, so the ticket's own pre-registered kill
criterion ("sections padded with restated titles") is not tripped at landing time.

#### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| B1 | **blocking** | — | `TEMPLATE.md`'s `## Outcome` placeholder ships in the angle-bracket form (`<1–3 sentences…>`), not the HTML-comment form Task 3 specified. `OutcomeMissing` strips only `<!-- … -->`, so a ticket authored from `TEMPLATE.md` with the placeholder untouched is **not** flagged — while four shipped statements say it is. Rules §7 tells authors to write tickets from `TEMPLATE.md`, so this is a golden path, and the deviation from a "do not deviate without asking" task snippet was never surfaced. | `OutcomeMissing(TEMPLATE.md)` returns `false` (scratch test; section body = the literal `<1–3 sentences…>` placeholder). False claims: `skill/resources/TEMPLATE.md:23-24` ("or still **this** placeholder"), `skill/resources/tickets-README.md:344` , `docs/user-manual/cli-reference.adoc:545-546` , `CHANGELOG.md:17-18`. Contrast `Scaffold` (`ticket.go:648`), whose `<!-- TODO … -->` form *is* flagged. | Restore Task 3's HTML-comment form in `TEMPLATE.md` (makes all four statements true and matches `Scaffold`), **or** teach `OutcomeMissing` to treat a body that is a single `<…>` angle placeholder as empty. Either way add a regression test asserting `TEMPLATE.md`'s own Outcome placeholder is flagged — a sibling of `TestScaffoldSectionsMatchTemplate`, so the two forms cannot drift again. |
| N1 | non-blocking | noted | Task 1's required test — "a body containing the literal text `## Outcome` inside e.g. a fenced code block … assert it" — was not written, and the premise it rested on ("so this is a non-issue") is false: the line-prefix walk *does* match a fenced heading. The implementation documents the limitation honestly in `SectionBody`'s doc comment, which is the right call, but the resulting plan amendment was never recorded in `## History` as the plan's own amendment rule requires. | Scratch test: for a ticket with no real Outcome and a ```` ```markdown ```` block containing `## Outcome`, `SectionBody` returns `"example text\n```"` and `OutcomeMissing` returns `false`. Shared with the pre-existing `SectionHeadings` walk (`ticket.go:543`). | Leave as is: the effect is a warning-only false negative, on tickets that document brine itself. If ever promoted, it belongs with **T-070**/**T-080**'s "one shared section walk" theme, not to a ticket of its own. |
| N2 | non-blocking | fixed inline | This branch falsified two "the scaffold is audit-clean" claims: a freshly `ticket new`'d ticket now trips the Outcome warning **by design** (Task 4). | `internal/ticket/ticket.go:621` ("the minimal, audit-clean starting point"); `docs/user-manual/cli-reference.adoc:435` ("writes a clean, audit-passing scaffold"). Reproduced in a throwaway install. | Fixed on the branch (commit `686777e`): both now say *error-free*, and note that the `## Outcome` placeholder is the one thing `board audit` warns about until it is filled in. |
| N3 | non-blocking | fixed inline | Decision 2 states the T-081 soft coupling is "recorded below, **in T-081's own Description**". It is not — T-081's Description records only a T-064 coupling. Doubles as the step-8 impact sweep's one action. | `tickets/1-to-do/T-081-…md:52-54` mentions T-064 only; `rg T-083 tickets/1-to-do/T-081-*` → no match. | Fixed inline: the soft-coupling line naming T-083 (and what already shipped) added to T-081's Description. |
| N4 | non-blocking | fixed inline | Finish step 3 required the pre-registered kill criterion to be "carried into T-083's History for a future reviewer to check against". `## History` carries the four transition lines only. | `## History` below, before this review. | Fixed inline: a dated History note records the kill criterion verbatim. |
| N5 | non-blocking | fixed inline | Task 5's backfill gave T-085 a `## Outcome` that restates, near-verbatim, the `**Outcome.**` paragraph its Description already opened with — the same text twice in one ticket. | `tickets/1-to-do/T-085-…md:16` vs `:20-24`. | Fixed inline: the Description's now-redundant `**Outcome.**` lead trimmed to its non-duplicated remainder. |
| N6 | non-blocking | folded (T-080) | The new check re-types the terminal-status literals (`t.Dir != "6-done" && t.Dir != "7-dropped"`) instead of using the existing predicate `ticket.StatusByDir(dir).Terminal`. Behaviourally identical; it is one more copy of the fact T-080 exists to centralise. | `internal/audit/audit.go:158`; the idiom in use at `internal/cli/project.go:237`; precedent copy at `audit.go:226`. T-080's title is literally "states, transitions and **terminal**/WIP flags into a flow definition". | No change here — T-080 already owns the ground; it inherits one more call site. |

**Disposition summary:** 7 findings — 1 blocking (B1, → `5-rework/`); 6 non-blocking: 4 *fixed
inline* (N2, N3, N4, N5), 1 *folded* into T-080 (N6), 1 *noted* (N1). No new tickets spawned.

**Rework scope (B1 only):** make the shipped placeholder and the shipped claims agree, and add
the regression test that pins them together. Nothing else on this branch is in scope.

**N-A (audit note, not a finding).** Acceptance step 5 ("before Task 5's backfill, every open
ticket produces exactly one new warning") could not be re-run verbatim, because the backfill is
already on `main`. It was verified equivalently: the unit fixtures in `audit_test.go` cover
absent / placeholder / non-terminal / terminal cases, and a throwaway install reproduces the
one-warning-per-unfilled-ticket behaviour end to end.

### Pickup applicability gate

**Pickup applicability gate (2026-08-07).** Fresh sub-agent audit, no blocking findings. Four
non-blocking, all cosmetic line/count drift since filing — dispositioned here per rules §5
(this is a pickup gate, not a review, but the same table/vocabulary applies) rather than in a
separate Review table, since the ticket has not shipped yet:

| id | finding | disposition |
|---|---|---|
| F1 | plan said "25 open tickets"; `1-to-do/` holds 24 today, T-083 itself (the 25th) sits in `2-ready/` | fixed inline — acceptance test wording corrected above |
| F2 | Task 2's `audit.go:~172` citation sits at the boundary of two per-ticket loops (frontmatter/grade at :72, History at :175); intent (end of the first loop) is recoverable from context | noted |
| F3 | `TestScaffoldSectionsMatchTemplate` cited at `ticket_test.go:511` is its doc comment; the func itself is :512 | noted |
| F4 | `tickets.adoc`'s `== Body` heading cited at `:80` is actually `:78` | noted |

## History

- 2026-08-07 — created (TO DO). source: chat — a proposed "Decision Inputs" ticket section was
  challenged and reduced; the surviving finding was that 9 of 23 TO DO tickets open with
  mechanism rather than outcome, and that the two impact recalibrations wrote 11 usable
  one-line justifications into NOTES.md tables instead of into the tickets
- 2026-08-09 — TO DO → READY: plan complete
- 2026-08-09 — READY → IN DEVELOPMENT: picked up
- 2026-08-09 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-09 — kill criterion recorded (Finish step 3, added by the review as finding N4): if
  `## Outcome` ships and the next refinement pass over `1-to-do/` is not measurably faster to
  triage — or if the sections are being filled with restated titles — drop the audit check and
  revert to a lead-sentence convention. A future reviewer checks the feature against this.
- 2026-08-09 — IN REVIEW → REWORK: review 1: B1 blocking — TEMPLATE.md's Outcome placeholder is not the form the check detects, while four shipped statements say it is
