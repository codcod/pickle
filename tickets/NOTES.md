# Notes

Hand-written planning notes live here — triage records, parked-ticket notes, cross-ticket
decisions, dependency rationale. `BOARD.md` is generated from the ticket files (run
`pickle board sync`), so nothing hand-written survives there.

Migrated from BOARD.md's prose sections on 2026-07-26 (T-044). The old board banner warning
against running `pickle board sync` was **deleted, not migrated**: its three confirmed
data-loss behaviours (prose deletion, pipe un-escaping, branch-cell overwrite) are gone by
construction now that the board is a pure generated artifact — see T-044.

## Parked tickets (triage 2026-07-26)

**Parked: T-009, T-010, T-016.** Real but explicitly **unscheduled** — nothing is blocked on
them and no user has asked. Do not pick one up without a demand signal. Parked status is also
recorded in each ticket's Description and History.

## Epic merge (executed 2026-07-26 — 14 tickets → 5 epics)

The sources are in `7-dropped/` with reason `absorbed into T-0NN`; each opens with an ABSORBED
banner and keeps its full analysis, measurements and line references. They are the
authoritative detail; the epic is the refinable, reviewable unit. Do not re-file a source or
implement from it.

| epic | absorbs |
|---|---|
| T-039 — BOARD.md write/validate integrity | T-014, T-023, T-034, T-037 |
| T-040 — ticket frontmatter validation | T-027, T-028, T-033 |
| T-041 — marker-block freshness | T-020, T-021 |
| T-042 — collapse duplicated internals | T-015, T-017, T-032 |
| T-043 — test harness + cli coverage | T-031, T-012 |

Deliberately **not** merged: T-013 (10 items, its own epic already), T-019 (docs-only), T-038
(input contract, successor to T-030), T-022, T-026, T-036.

**T-045 is measurement-gated, not just low priority.** It holds the two valves split out of
T-036 (backlog cap, `user-visible:` axis). Both are backstops for the leak T-036 plugs, so it
must not be refined until T-036 has landed and the spawn rate has been re-measured over at
least three reviews. Dropping it is a legitimate outcome.

## Merit challenge (2026-08-01) — T-063 filed and dropped the same day

**T-063** (derived value-per-cost board ordering) was filed from a chat exploration and dropped
hours later after an adversarial merit review run *before* refinement. The full evidence is in
`7-dropped/T-063-…` (DROPPED banner) and the verdict is folded into **T-056 work area 5**, which
had asked for exactly that hearing. **T-064** was filed for the gap the episode exposed.

**The finding worth remembering is about the board, not the ticket.** The pickup queue is
**READY** (`review-protocol.md:192`), and across all 114 revisions of `BOARD.md` READY has
**never held more than 2 rows**. TO DO's ordering — 18 rows, argued over repeatedly (T-045,
T-056·5, T-059, T-063) — is therefore mostly cosmetic: nobody picks from it. Any future ticket
proposing to reorder, cap, rank or re-axis the backlog should **re-measure READY occupancy
first**, and expect that measurement to sink it.

**Second: `family:` (T-059) has 0 adopters across 63 tickets**, four days after merging, through
exactly the 7-way `medium` tie it was built to break. That is a negative demand signal for
curated ordering generally, and it should be cited *against* the next such proposal — including
by whoever wrote the last one.

**Standing, still-unexecuted alternative: recalibrate `impact`.** `tickets-README.md:139-140`
already mandates re-grading the board on every filing, and T-045:76 already names recalibration
as the recommended starting position. `critical` and `high-critical` have **never been used** in
63 tickets — two levels of headroom above a 7-way `medium` tie. Every ordering ticket so far has
proposed new machinery instead of spending that headroom.

**Process note — superseded the same day, see below.** The challenge was requested ad hoc, not
produced by any gate. That was filed as T-064; a second adversarial pass dropped it.

## The T-064 correction (2026-08-01) — it was compliance, not a missing gate

**T-064** proposed a merit gate between filing and pickup. It was dropped hours later by the same
kind of adversarial pass it wanted to institutionalise. The findings matter more than the ticket:

- **The rule already exists and is not being followed.** `tickets-README.md:139-140` mandates
  assessing every new ticket against the backlog **"and re-grade the board."** The commits filing
  T-063 (`829a819`) and T-064 (`a3f749f`) each touched exactly **two files** — the new ticket and
  `BOARD.md`. **Zero neighbours re-graded, twice.** Diagnosing a missing gate while breaking the
  existing one in the same commit is the whole episode in one sentence.
- **§8 already contains a merit test; its heading hides it.** The mandate is *"the ticket's own
  assumptions plus the board delta"* tested for *"true, **required**, and **worth it**"*
  (`tickets-README.md:326-327`, `SKILL.md:177`) — but the section is headed *"a **freshness**
  check"* and justified purely by aging. Practice followed the heading: **0 negative verdicts in
  ~15 recorded applicability-gate runs**; T-062's History calls it "confirmed against the current
  tree". Two sentences would fix it (scope includes assumptions wrong *at filing*; DROP is a legal
  verdict, already legal per `move.go:31-38`). Pointer left in **T-022**, which edits those files.
  **Deliberately not filed as a ticket** — see the pattern below.
- **"Filed from chat" is the wrong trigger.** All three cases of real waste in the project's
  history — **T-060** (refinement session paid, then dropped), **T-062** (built → reworked →
  reviewed → reverted), **T-059** (shipped, 0 adopters) — carry `source: pickle ticket new`. Any
  future proposal to target a filing population must measure against these three first.
- **What actually works:** a human asking for an adversarial pass when they smell one. **2 for 2**
  on this backlog (T-063, T-064), costs nothing when unused, needs no schema and no payload bump.
  Automating it was the error, and the automation would have been a rubber stamp — the gate it
  proposed reusing has never once said no.

**The pattern, recorded against the next proposal — including the next one from whoever writes
here.** Three ordering/gating tickets in a row (T-063, T-064, and T-045 before them) proposed new
machinery while two standing, free, mandated actions sat unexecuted: **recalibrate `impact`**, and
**spend T-045's now-available measurement data**. Both were finally done on 2026-08-01 (below).
Before filing anything in this theme again: measure the thing, check whether an existing
instruction already covers it, and check whether it was executed.

## Spawn rate measured, T-045 dropped (2026-08-01)

The measurement T-045 was gated on, finally taken. **8 reviews since T-036 landed** (gate required
3): T-047, T-048, T-049, T-059 spawned 0 each; T-053, T-058, T-061 spawned 1 each; T-054 spawned 2.
**R = 5/8 = 0.625**, against ≈1.0 re-derived at T-036's refinement. The pre-registered condition —
*"if R has fallen well below 1, the honest outcome is to drop this ticket"* — is met, so T-045 is
dropped on evidence committed to in advance rather than on argument after the fact. The table is
in its DROPPED banner and is the baseline for any re-open; re-open only on R above ~1.5 sustained
over ≥5 reviews, or `1-to-do/` growing while completions stall.

**This is the first decision in the project made by a pre-registered criterion.** It cost one
`for` loop. Worth copying: when a ticket's real question is "is this needed?", write the
measurement and the threshold into the ticket at filing, then execute it.

## `impact` recalibration (2026-08-01) — the standing mandate, finally executed

`tickets-README.md:139-140` has mandated re-grading the board on every filing since the flow was
installed; it had never been done as a pass. Executed across all 16 TO DO tickets against the
rules' own definitions (`:129-134`). Seven changed:

| ticket | was | now | why |
|---|---|---|---|
| T-041 | medium | **high** | a stale marker block makes every agent act on wrong project config, silently — it breaks the agent-facing contract, which is the product |
| T-040 | medium | **medium-high** | duplicate frontmatter keys silently last-win, a latent data-loss path and a prerequisite for any field writer |
| T-056 | high | **medium-high** | one of its six work areas (ranking) collapsed on 2026-08-01, and demand for a *writable* dashboard is unevidenced; the concurrency foundation retains its value independently |
| T-013 | low | **low-medium** | install is the first-run experience and this bundles 10 items |
| T-019 | low | **low-medium** | the README is the adoption front door for a distributed tool |
| T-038 | low-medium | **low** | narrow input hardening on a path that already rejects the dangerous cases |
| T-055 | low-medium | **low** | cosmetic CSS specificity bug |

Distribution went from **7-way `medium` + 4-way `low-medium` + 4-way `low`** to
**high 2 · medium-high 2 · medium 5 · low-medium 3 · low 4**. Largest tie 7 → 5.

**`critical` and `high-critical` remain unused, deliberately.** The bar is "reshapes the product",
and nothing in this backlog does — pickle is shipped and working, and every open item improves or
corrects it. Recalibration means grading honestly, **not** spending headroom for its own sake; a
manufactured `critical` would be the same calibration dishonesty in the other direction. The
headroom exists for the ticket that genuinely earns it.

## `impact` recalibration (2026-08-03) — second pass, 13 TO DO tickets

Run as a pass over the whole TO DO group after T-040's review, against the rules' definitions
(`tickets-README.md:129-134`). Four changed:

| ticket | was | now | why |
|---|---|---|---|
| T-022 | medium | **medium-high** | same defect class as T-041 (graded `high` for it): the payload states commit policy, branch prefix and WIP limits unconditionally, so in any **non-default** project the shipped skill contradicts the marker block's real values and both read as authoritative — agents act on wrong project config. One notch below T-041 because it bites only non-default configurations, not every install. Complexity low / cost S. |
| T-057 | medium | **medium-high** | the only open item guarding against **silent loss**: bookkeeping committed on a `feat/` branch is eaten by the squash-merge, and it has happened three times — once *while closing the review that flagged it*. `install` defaults `--path .` (`install.go:95`), so the **default install is single-repo** and every one of them carries the hazard. |
| T-056 | medium-high | **medium** | second downgrade in three days, on new evidence only: work area 5 (ranking) closed as *"don't rank at all"*, and T-040's review (finding N9) showed its stated T-040 prerequisite was never removed — D1 kept last-wins parsing, so a field writer still needs its own guard. Demand for a *writable* dashboard remains unevidenced; the concurrency foundation keeps its value but nothing has raised it. Grading the backlog's one XL above tickets that fix measured field defects overstated it. |
| T-019 | low-medium | **low** | scope shrank to a single item — the stale `PLAN.md:227` synopsis — after T-047 deleted or fixed the other three. The ticket's own note already said "likely impact low". |

Distribution: **medium-high 2 · medium 4 · low-medium 2 · low 5** (13 tickets). Largest tie
unchanged at 5 (the `low` floor), and that is the honest answer: five genuinely narrow items.

**No `high` in this backlog, and that is the finding.** T-041 — the last `high` — is done and
merged, and nothing open is a "major capability/adoption lever": every remaining item improves
or corrects a shipped, working tool. `critical`/`high-critical` stay unused for the reason given
in the 2026-08-01 pass. Manufacturing a `high` to refill the top of the board would be the same
calibration dishonesty in the other direction.

**Effect on the queue:** the top two are now cheap (T-022 low/S, T-057 medium/M) and the XL
(T-056) sits sixth. Note the standing caveat above still applies — the pickup queue is READY, not
TO DO, so this pass changes what gets *refined next*, not what gets picked up.

**Correction to the 2026-08-01 table.** It records T-040 as `medium-high` because "duplicate
frontmatter keys silently last-win, a latent data-loss path". T-040 was re-graded to `medium` at
its own refinement (2026-08-02), and its decision D1 deliberately **kept** last-wins parsing — the
audit reports duplicates, it does not remove the hazard. That row's rationale was stale in both
halves; the shipped behaviour is recorded in T-040's Review (N9) and in T-056's soft couplings.

## Cross-epic decisions

**T-044 won the T-039-vs-T-044 design decision** (2026-07-26): the board becomes a generated
artifact; T-039 (harden the hand-maintained design) was dropped as superseded, and its
move-atomicity residue (T-014·4) is folded into T-044. Escape-vs-replace is settled by T-044's
one-way cell sanitisation — **T-043 item 5 defers to T-044**.
T-042 collides with T-044 (`internal/board`, `internal/sync`) and with T-043 (`cli_test.go`) —
sequence, do not run concurrently.

## Dependency chain (hard `depends-on:`, human-approved 2026-07-23)

- **T-001** (config/registry) → **T-002**, **T-003**, **T-004**.
- **T-002** (audit) → **T-007**, **T-008**.
- **T-003** (ticket new) → **T-012** (hardening).
- **T-004** (install) → **T-005**, **T-006**, **T-009**, **T-010**, **T-013**.

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

- **T-011** (distribution) wants the command set (P1–P3) essentially complete — narrative
  coupling only, no hard `depends-on`.

## Field-finding triage (2026-07-27) — first external workspace

Findings from operating **pickle 0.1.0** on a real migration (an 84-ticket hand-rolled flow
moved into a fresh `pickle install` workspace) plus one guardrail false positive. They were
collected in a scratch `FEEDBACK.md`, triaged here, and the file was deleted — this section is
the record. Filed: **T-049** (render-side cell cap) and **T-050** (guardrail verdict); folded:
a fifth check into **T-040** (History-line shape). Below is what was *not* filed, and why.

**Both raw findings named the wrong mechanism, and both proposed fixes followed from it.** Worth
remembering next time a field note arrives with an implementation already attached: the note's
symptom and repro were sound, its diagnosis was not, and its "possible shapes" would have
anchored refinement onto the wrong change. Field notes should carry symptom + repro + constraint.

- **"The DONE `merged` cell reproduces a whole paragraph."** It cannot. `historyRE`
  (`internal/ticket/ticket.go:104`) is line-anchored, so a multi-line note contributes only its
  first line. The 1,900-character cell was **one 1,900-character physical line** — which changes
  the fix from "handle paragraphs" to "cap width, and lint the line". See T-049 + T-040.
- **"The guardrail matches against the whole bash command string."** It does not: `segments()`
  splits on `&&`, `||`, `;`, `\n`, and the heredoc *body line* matches as its own segment. See
  T-050 for why quote-awareness was rejected outright (it opens `bash <<'EOF'` as a bypass).

**Noted, not filed — the `cd <other-workspace> && pickle upgrade` prompt.** Reported as "same
class" as the guardrail false positive; it is not. Rule 3 in
`.pi/extensions/workspace-guardrails.ts` already uses `ctx.ui.confirm`, so being *asked* to
approve a self-modifying command is the designed behaviour. `targetsTmp(seg, ctx.cwd)` is blind
across the `&&` split (the `cd` lands in a different segment), so it can never recognise a
throwaway target and always asks — a precision loss in *when you are prompted*, not a false
refusal. Fixing it needs `cd`-state tracking across segments, i.e. a resolved-working-directory
notion, which is a different change from anything in T-050. Promote only if the prompting becomes
a nuisance. Note also that the claim "scoping by resolved target directory would fix both" is
false: the documentation false positive has no target directory at all.

**Noted, not filed — the declarative mirror may be immune, which is the more interesting
finding.** `opencode.jsonc:35-38` says its patterns match "against the parsed command", and they
are prefix-anchored globs, so quoted prose inside a heredoc should not match — unlike the two TS
extensions. Untested. T-050 task 3 verifies it rather than assuming; if the declarative form is
immune, that is evidence about which guard shape to prefer, not a bug to fix.

**Noted, not filed — the repro was an anti-pattern.** The blocked command was a `python3 - <<'EOF'`
heredoc writing a markdown file, in a harness that has `write`/`edit` tools for exactly that. The
guardrail's false positive is real and T-050 fixes the verdict, but do not harden the guard to make
shell-heredoc file authoring comfortable — that optimises a workflow the toolset already replaces.

**Measured evidence kept for T-049's refinement.** Longest merge History line in `tickets/` today:
171 characters. DONE `merged` cells for T-001, T-002, T-008, T-009, T-011 and T-044 already run
90–125 characters. The defect is this repo's trend line, not migration exotica.

## Field-finding triage (2026-07-27, second wave) — first second-child onboarding

Findings from operating **pickle 0.1.0** while adding a second child-project (`snowball`
alongside `rick`) to the external `unity` workspace. Filed as **T-051** (`project add` leaves
five workspace-side consequences unstated) and **T-052** (the post-upgrade audit cannot tell a
registry-changed board from a hand-edited one). Both carry symptom + repro + constraint and
deliberately **no chosen implementation** — the lesson of the first wave, above.

**Process note.** These two were first written into a scratch `tickets/IDEAS.md`, which has
since been deleted: an idea file next to a ticket flow is a second backlog with no gate, and
the flow's own rule is that work enters as a ticket. Rejected ideas still need a home, and
that home is this file — hence the entry below. If a pre-ticket holding pen ever earns its
keep, it should be argued for as a pickle feature, not improvised as a file.

**Audited and NOT filed — "the shipped `pickle-guardrails.ts` has the same unanchored
child-path bug."** The bug is real but **workspace-local**. In `unity`'s own
`unity-guardrails.ts` the never-stage pattern was `(^|/)<child>(/|$)`, which also matched
`development/<child>/…` — the per-child development record, ordinary bookkeeping — so no pi
session rooted there could stage it; latent since `rick` was the only child, and fixed there by
anchoring at the pathspec start plus `../` climbs. Pickle's shipped guard cannot have this bug:
`agents/pi/extensions/pickle-guardrails.ts` has **no child-directory deny-list at all**, only
the explicit-pathspec rule (`-A` / `.` / `commit -a`) and the publish gate. The deny-list is a
`unity` invention. The anchoring lesson is carried in T-051's Description in case that ticket
grows such a guard.
