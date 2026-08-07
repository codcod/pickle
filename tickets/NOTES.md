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

## Model-tier exploration (2026-08-03) — explored, not filed

Idea from chat: run each flow step on a different model tier — *"filing a ticket would be
Sonnet's work, refining and reviewing Opus"*. Explored against the tree; recorded here rather
than filed, per the standing lesson that this theme attracts machinery. **Tier 0 (below) was
folded into T-022; Tier 1 is parked with a pre-registered kill criterion; Tier 2 is argued
against.**

**The precedent already exists at N=1.** Review protocol step 4b (`docs-readability`) is a
per-step model assignment shipping in the payload today: a pinned model
(`agents/opencode/opencode.jsonc:30`, `agents/pi/extensions/docs-readability.ts:44-46`), one
shared prompt (`skill/resources/docs-readability.prompt.md`), least privilege (opencode denies
edit/bash/webfetch; pi passes `--no-builtin-tools`), and — decisively — *"genuinely optional and
never blocks a review"* (`review-protocol.md:98-113`). So the proposal is **generalising an
existing pattern from one step to seven**, and the honest way to price it is to look at what
that one step actually cost.

**Measured cost of the one tiered step we ship: it does not work here.** T-040's review recorded
step 4b as a conscious skip — the reviewer returned `model_not_supported`. Diagnosed 2026-08-03:
`opencode models` lists **60 models across `anthropic`, `github-copilot`, `gitlab`, `ollama`,
`opencode` and contains zero Gemini entries**, so `github-copilot/gemini-2.5-pro` is unavailable
to *both* bindings. This is not a login failure — it is a payload default asserting a provider
catalog the environment does not have, i.e. **the same defect shape as T-022**: the payload
stating one configuration as universal. Sample size 1, failure rate 100 %.

**Also: the default agent cannot reach the tiered step at all.** `install --agent` defaults to
`claude` (`install.go:72-86`), which gets the skill symlink and the `CLAUDE.md` marker — pickle
ships **no `.claude/agents/` definitions**, although Claude Code subagents support `model:`
frontmatter. Any tiering scheme must answer for the default installation first.

**What pickle can control — three levers, and no more.** Pickle is a scaffolder with **no runtime
role during a flow step**; it never sees the model.

| lever | cost | enforceable |
|---|---|---|
| A — payload prose (a suggested-tier table in `SKILL.md`) | ~free, one edit, propagates by `upgrade` | no, advisory |
| B — harness scaffolds (`.claude/agents/`, `opencode.jsonc`, pi extensions) | real pinning, × 3 harnesses × N steps | per harness |
| C — `[tier]` in `pickle.toml` generating B | schema + generator + drift checks + docs | transitively |

A rule pickle cannot check is what T-064 was dropped for: the rule existed, compliance did not.
Lever A inherits that weakness by construction.

**Two corrections to the proposed mapping**, from reading what the steps actually demand:

- **It leaves the largest saving untouched.** Filing is a handful of tickets a week; **implement**
  is the token mass, and the flow is built so the plan is *"the executable prompt"* — the single
  strongest tier-down candidate, and it is not in the proposed split. Meanwhile *audit the board*
  needs **no model at all** (`pickle board audit` is pure mechanics), which is the reminder that
  the cheapest tier is sometimes zero.
- **The real axis is independence, not cost.** The steps that would benefit — review, and the
  pickup applicability gate, whose text already mandates *"a fresh sub-agent — free of the
  implementer's sunk-cost bias"* (`SKILL.md:173`) — want a **different model family**, not merely
  a better one. The evidence is already in this file: human-requested adversarial passes are **2
  for 2** (T-063, T-064 both dropped), while the same-agent applicability gate has **0 negative
  verdicts in ~15 runs**. That is an argument for *difference*, and it is the only part of the
  idea with measured support.

**The case against, which this file requires:** no demand signal (nobody has produced a bad flow
outcome attributable to model tier); a portability tax of per-step × per-harness (one step cost
two implementations plus a shared prompt); model ids that rot faster than pickle releases
(demonstrated above); and `doctor` already treating the pinned model as pickle-owned — it warns
when `.pi/extensions/docs-readability.ts` differs from the shipped copy, so a user who retunes
the model reads as drift.

**Three increments, ascending:**

- **Tier 0 — advisory table in the payload. Folded into T-022 (2026-08-03), then dropped back out
  at T-022's refinement (2026-08-05) — it stays here, unfiled.** The fold-in reasoning was that
  T-022 already rewrites `SKILL.md`, `tickets-README.md` and `review-protocol.md` to state
  defaults and defer, so a suggested-tier table is the same move; free, harness-agnostic, and
  actionable by hand (`/model` before saying "refine T-NNN"). Refinement rejected it on two
  grounds. First, it is a **different theme** — "the payload says nothing about X", not "the
  payload states per-project config as universal" — and carrying two themes costs T-022 its `S`.
  Second, and decisively, **T-022's own item A is the counter-evidence**: the pickup gate's prose
  already asks for exactly the behaviour it never got, measured at **0 negative verdicts in ~15
  runs**. Another advisory, unenforceable prose block is the same bet that just lost. Per the
  standing lesson, it is recorded here rather than filed. Re-open only with a demand signal.
- **Tier 1 — pin only the applicability gate**, the one step whose text already asks for a
  separate agent, mirroring step 4b's shape exactly (shared prompt, optional, never blocking,
  least privilege) and pinning a **different family** rather than a higher tier. **File only on a
  demand signal**, and only with this **pre-registered kill criterion**, in the T-045 style:
  *if after 5 gate runs the tiered gate has produced 0 findings the flow agent would not have
  produced, drop it.* Note the criterion has a live baseline to beat: 0 negative verdicts in ~15
  same-agent runs.
- **Tier 2 — `[tier]` config generating per-harness agent definitions.** Argued against until
  Tier 1 has data: it turns every model-id rotation into a `pickle upgrade` for every installed
  project, which is exactly the failure measured above, multiplied by the number of tiered steps.

**Open, not decided: what to do about step 4b's broken default.** It is a shipped default that
fails on a legitimate Copilot login. Candidates: fold a "do not pin a model the provider may not
offer — degrade to a named default plus an env override, and say so" item into T-022 (same defect
shape, different files), or file it small on its own. Do not simply bump the model id: the lesson
is the pinning, not the id.

> **T-022 refinement (2026-08-05) declined the first candidate, leaving this open.** The defect
> class matches exactly, but the fix is a *code* change — `agents/opencode/opencode.jsonc:30` and
> `agents/pi/extensions/docs-readability.ts:44-46`, plus an env override and its docs — while
> T-022 is prose-only and graded `S`. Folding it in would have changed the ticket's shape and its
> cost. So the choice is now between filing it small on its own and continuing to carry a shipped
> default with a measured 100 % failure rate; nobody has been blocked by it, which is why it is
> still here.

> **The T-022 review (2026-08-05) hit it again, n=2.** Step 4b's `docs_readability` tool exited 1
> with `400 model_not_supported` for `github-copilot/gemini-2.5-pro`, so the review's own
> readability pass was a recorded conscious skip. Also logged there as finding **F5**: the same
> defect shape in the *other* direction — `agents/pi/extensions/pickle-guardrails.ts:5-11,84-88`
> and `agents/opencode/opencode.jsonc:59-67` assert the publish gate unconditionally without
> reading `child_publish_gated`, so a project that turns publish-gating off still gets the prompt
> and the comment that says it cannot be off. Same two files, same class; if the pin is ever filed
> small on its own, this belongs in the same ticket.

## Pi-as-best-tier exploration (2026-08-04) — three roads, and a process failure

Idea from chat: make Pi the **"best experience tier"** — hard gates instead of prose rules, agents
spawned on different model tiers, one install command, and a board UI with a path to editable.
Explored against pi.dev's extension/package/skills references and this tree. Recorded here rather
than filed as a programme. **Two of the five stated requirements are already answered on the board
— one of them against, with measurement. One item survives independent of Pi. The Pi-facing work
is filable but contingent, and the enforcement premise it rests on has a poor field record.**

**Process failure first, because this file exists to catch it.** The roadmap was produced across a
long chat *without reading this file or the affected tickets*. `:100-101` mandates the opposite:
*"measure the thing, check whether an existing instruction already covers it, and check whether it
was executed."* Cost of skipping it: three of fourteen proposed tickets were already filed as
**T-056**, and a fourth was already argued against here on 2026-08-03 with a 100 %-failure
measurement. Cite this paragraph against the next chat-produced roadmap — **including the next one
from whoever writes here.** A multi-phase plan is the highest-risk artifact this backlog accepts,
because its size hides the collisions.

### What was asked for

| requirement | verdict |
|---|---|
| 1 — one install command | **New, filable.** Reachable two ways; see *the premise is satisfiable twice* below |
| 2 — Pi as best tier (gates, enforcement) | **New, filable, contingent** — but see the guardrail firing record |
| 3 — close to ideal UX | not independently actionable; a property of 1/2/5 |
| 4 — agents on model tiers, sensible defaults | **Already decided, mostly against** — `:173-258` |
| 5 — board UI, path to editable | **Already filed as T-056**, XL, twice downgraded for unevidenced demand |

### The three roads, and the axis that separates them

Not Go-vs-TypeScript. The axis is **where the Pi-facing surface lives and who versions it**.

| road | shape | ongoing cost |
|---|---|---|
| **A** — generated payload | binary writes `.pi/extensions/*.ts` + agents into the repo | **high** — ~1–2 k lines of generated TS to drift-manage, and an unenforced version contract between repo-resident TS and `PATH`-resident binary |
| **B** — package-owned | one npm package ships extension + skill + **the Go binary** (platform `optionalDependencies`, the esbuild pattern); `bin` field keeps `PATH` use | low — an npm publish step over existing goreleaser artifacts. **Zero Go rewritten** |
| **C** — API-first | `serve` becomes the single write path; extension, web UI and any future client are thin clients | medium — daemon lifecycle; **and this is T-056** |

**The premise is satisfiable twice, and the chosen half was the expensive one.** "Avoid two installs"
was given as the reason to keep generating config from the binary (road A). But npm can *carry* the
binary, so `pi install npm:pickle` is also one command — and it deletes both the repo payload and
the version contract. If road A is chosen it must be for a stated reason other than install count
(e.g. refusing a second registry, or keeping `go install` canonical). Note **T-056's Scope boundary
already rules "the sidecar-vs-single-binary packaging question" out of scope by user decision** —
so packaging is a separate ticket either way, not a T-056 work area.

### Collisions with the board

| proposed | already exists |
|---|---|
| `internal/api` read+write layer, typed errors, JSON-serializable types | **T-056 work area 1**, verbatim the same package name, with a fuller audit: TOCTOU id allocation (`cli/ticket.go:124`), zero `flock` hits repo-wide, no frontmatter serializer, eight `move.Move` errors collapsed to exit 1 |
| write endpoints + tree locking | **T-056 work areas 1–2**; area 2 is flagged *"worth doing on its own merits regardless of the dashboard"* |
| editable board UI | **T-056**, literally its title |
| per-step model-tiered agent files | **`:250-252`** — *"Tier 2 … **argued against** until Tier 1 has data: it turns every model-id rotation into a `pickle upgrade` for every installed project"* |

T-056 also documents its own split seam (*"a plausible sequence: work area 2 → 1 → 3 → 4 → 5 → 6"*)
and was **filed as one unit at the user's direction**. Carving work area 1 out reverses that
direction and is the user's call, not a refinement decision. T-043 overlaps area 1 directly — *"doing
both separately means writing the same tests twice."*

**T-010 (`pi-guardrail-scaffold`) was dropped as absorbed into T-009, not on merit** — *"agent
enablement owns the pi scaffold **+ symmetry obligation**"*. That obligation is the standing answer
to harness asymmetry: whatever Pi gains, the other harnesses owe an equivalent. Any gate-only-on-Pi
proposal must answer to it.

### The correction that de-risks B: it does not depend on T-056

Worked out after reading T-056, and it reverses the ordering proposed in chat. The extension's
**tools can shell out to the existing CLI**, which already works, is already tested, and already
enforces every gate in `move.go:31-115`. `internal/api` was only ever needed for **C** (an in-process
HTTP surface). So B's only Go prerequisite is a **JSON read surface** — not the write extraction.
Sequence B → C, and C inherits T-056 rather than duplicating it.

**Partly retracted on 2026-08-04, at filing time (see the postscript below).** "B does not touch
T-056" holds for the *transport* claim — shelling out to the CLI is real, and no write API is
needed. It fails for the read surface: the projection an extension consumes has to be lifted out
of `internal/serve/view.go`, and that lift **is** T-056 work area 1's extraction seam. So B and C
share one piece of work. The revised claim: **B needs no *write* API, and one shared read
projection that T-056 area 1 would otherwise build twice.**

### The enforcement premise has a poor field record — this is the finding

The strongest argument for a Pi extension was that Pi's `tool_call` hook converts pickle's prose
rules into enforcement. But **pickle already ships exactly one such guard**, and its recorded field
history is: **one false positive (T-050, a `python3 - <<'EOF'` heredoc misread as a staging
violation), one prompting nuisance (`:302-311`), and zero recorded true positives.** Sample size is
small, but it is the only evidence there is, and it runs the wrong way. Any ticket proposing more
gates must state what it expects them to catch and how that will be measured — *before* filing, in
the T-045 style. "Enforcement beats convention" is a claim about this codebase, not an axiom, and
the one instance we have has cost more than it saved.

### Measured evidence against shipping model pins (unchanged from 2026-08-03)

Restated because a road-B "agent tier" ticket would re-propose it: the one pinned model pickle ships
(`github-copilot/gemini-2.5-pro`) is **unavailable in the observed environment — sample size 1,
failure rate 100 %** (`:190-196`), model ids rot faster than pickle releases, and `doctor` reads a
user's retune as drift. **Do not file per-role agent files with pinned models.** The live path
remains **Tier 1** — pin *only* the applicability gate, a different model *family* rather than a
higher tier, with the pre-registered kill criterion already written at `:243-249`.

### The case against, which this file requires

- **No demand signal beyond the requester's own ask.** Same provenance as T-060, T-062 and T-059 —
  the three recorded wastes (`:87-90`).
- **Harness asymmetry on a shared artifact.** Gates on Pi and prose elsewhere means the invariants
  hold on one harness and are aspirational on the others, while `tickets/` and `BOARD.md` are shared.
  Mitigated only by decision 3 below.
- **Pi is pre-1.0** and churns; its flagship package sits at 0.40.x. Coupling a payload to it is one
  thing; coupling the flow engine to it is another.
- **A second toolchain.** This repo is two Go dependencies and a 1.14:1 test-to-source ratio.
  TypeScript lands at 0:1 unless tsc + a test runner ship in the *same* ticket as the first line.
- **`serve` is 20 % of the codebase and already the board UI.** Requirement 5 is a T-056 question,
  not a Pi question.

### Decisions constraining any future ticket in this theme

1. **Never ship a pinned model id.** Measured, above. Degrade to a named default plus an env
   override, and say so in the payload.
2. **Pin exact versions in `.pi/settings.json`; never float a tool that owns a data schema.** Verify
   whether Pi's updater treats `^` ranges as pinned before relying on a range.
3. **Every Pi gate ships with a matching `board audit` invariant.** Pi fails fast; the binary catches
   the same thing everywhere else. This is what keeps T-010's symmetry obligation satisfied and stops
   a rule existing only in TypeScript.
4. **One editable board front-end, ever.** Two is the `board.Render` mistake in a new costume.
5. **Nothing enters `.pi/` except the version pin and `.gitignore` entries.** No generated TS in a
   user's repo.
6. **`upgrade` moves the tool; a migration moves the data. Never fuse them.** A release must not
   silently rewrite sixty tickets on someone's laptop.
7. **Three version axes stay separate**: tool version (npm/`pickle version`), data schema, config
   schema. `payload_version` today conflates the first with nothing else, and that is fine — the
   error would be overloading it.
8. **The skill has exactly one source per session.** Pi natively discovers `.agents/skills/`, so a
   package-shipped copy collides by `name:` and Pi *"warns and keeps the first skill found"* — a
   stale repo copy would shadow a fresh packaged one. The extension must arbitrate via
   `resources_discover`, not ship a second copy blindly.

### If you proceed: the first batch

Four candidates. Grades are against the 2026-08-03 recalibration (`medium-high 2 · medium 4 ·
low-medium 2 · low 5`, no `high`), and the honest reading is that **none of these is a `high`** —
each improves a shipped, working tool.

| # | title | impact | cplx | cost | notes |
|---|---|---|---|---|---|
| ~~1~~ | ~~`schema_version` + fail-closed version check~~ | — | — | — | **not filed** — no reachable hazard; pre-registered against T-056 area 4. See postscript |
| 2 | ~~`--json` on the read commands~~ → a versioned JSON read projection | low-medium | medium | M | **filed as T-065**, re-scoped: two of the five commands did not exist |
| 3 | distribute pickle as an npm package carrying the binary (one-command install for Pi) | medium | medium | M | requirement 1 |
| 4 | Pi extension: version handshake, skill arbitration, gates, tools | medium | medium | L | soft-couples 2 and 3; must answer the guardrail firing record |

**Sequence:** 1 and 2 are independent; 3 soft-couples nothing; 4 wants 2 and 3 landed. Propose all
couplings as **soft** (Description cross-references) — a hard `depends-on:` needs explicit sign-off
per §3 and none of these gates the others hard.

**Grade re-trigger on #1.** It guards silent loss that has occurred **zero** times, which is why it
sits below T-057 (`medium-high`, hazard *"has happened three times"*). The reason it has not bitten:
every frontmatter change so far — `spawned-by` (T-024), `family` (T-059) — was **additive and
optional**, so old binaries tolerate new trees. The first *required* or *renamed* key breaks every
older binary silently, and shipping a second distribution channel (#3) raises the odds of mixed
versions. **If #3 lands, re-grade #1 to `medium`.**

**Deliberately not in the batch, with reasons:**

- **A migration framework.** Speculative machinery, and this file's whole lesson is that this theme
  attracts it. **Pre-registered:** file `pickle migrate` only when a genuinely non-additive schema
  change is proposed. Until then #1's fail-closed guard *is* the deliverable — it converts silent
  corruption into a refusal, which is the whole safety value; the migration path can be written when
  there is something to migrate.
- **Per-role model-tiered agent files** — measured against; see above. Tier 1 only, on a demand
  signal, with the existing kill criterion.
- **`internal/api`, write endpoints, editable UI** — T-056. Note area 1 carries its own
  self-nullifying condition: *"If that is not the goal, the same three pieces could land in the
  existing packages instead, and the extraction should be dropped."*
- **A Pi TUI board** — decision 4. `serve` is the board UI.

**What would falsify this whole programme, stated in advance:** if #4 ships and, across 10 recorded
flow steps under Pi, the gates block zero actions an agent would otherwise have taken, the
enforcement premise is unsupported and the extension should be reduced to skill + tools. The
baseline it must beat is the shipped guardrail's record above: 1 false positive, 0 true positives.

### Postscript (2026-08-04, same day) — filing the batch falsified two of its four rows

The batch above was written from the tree's *documentation*. Reading the **code** to ground rows 1
and 2 killed one and re-scoped the other. Recorded because the batch table would otherwise read as
authoritative, which is the T-041/T-022 defect class applied to this file.

**Row 1 does not describe a reachable hazard.** The claim was that a newer field meets an older
binary and is silently dropped. No pickle command can do that:

- `internal/move/move.go:123` — `newText := appendHistory(t.Text, …)`. Moves **append**; frontmatter
  is never re-rendered.
- `internal/ticket/ticket.go:529` `Scaffold()` is the only frontmatter *renderer*, and its sole
  caller is `internal/cli/ticket.go:141` — brand-new files only.
- `parseFrontmatter` reads into `Front map[string]string`, so unknown keys survive a round trip.
- `pickle ticket renumber` — which `tickets-README.md:122-123` describes as though it ships — **does
  not exist**; `audit.go:101` marks it as T-060, unbuilt. (Minor doc-overstates-code finding, noted
  not filed.)

So the hazard is not merely zero-occurrence, it is **structurally absent**: it requires a
frontmatter re-render path, and the only proposed one is T-056 work area 4's field writer. The
guard is now **pre-registered on that**, in T-065's Description, alongside the migration framework
under the same discipline. Its one useful part — a version handshake — moved into T-065's JSON
envelope, where a wire format belongs rather than in 64 ticket files.

**Row 2 was mis-scoped in two directions at once.** It read "`--json` on the read commands
(`board state`, `ticket show`, …)" — but `board state` and `ticket show` **are not commands**
(`internal/cli/cli.go:48-66`), so it was not a flag addition. Meanwhile the projection those
commands would emit **already exists** in `internal/serve/view.go` (`buildBoard:77`,
`buildTicket:168`, `ChildWIP:280`, `buildHealth:298`), so the hard part is lifting template-shaped
structs into a wire type, not designing one. Filed as **T-065** (`low-medium` / medium / M) for the
projection; `--json` on the four reads that *do* exist (`doctor`, `board audit`, `project list`,
`version`) fell out of the critical path entirely — different audience, separable, unfiled.

**Re-grade pass: nothing moved**, and that is the honest result. T-065 joins T-013 and T-050 at
`low-medium`; distribution is now **medium-high 2 · medium 4 · low-medium 3 · low 5** (14 tickets),
largest tie still the 5-way `low` floor. Two candidates were considered and declined: **T-043**
(T-065 would be a fourth `captureStdout` consumer, but it is prospective, and the 2026-08-03 pass
declined to credit prospective demand when it downgraded T-056 — the same rule cuts both ways) and
**T-056** (a second customer for area 1's extraction is mild evidence for it, but T-056's weight is
the writable dashboard, still unevidenced).

**Phase B is now three tickets, not four.** The general lesson, which is the same one this file
already records twice: *grade and scope against the code, not against the docs describing it.*

### Second postscript (2026-08-04) — rows 3 and 4 die too; the phase does not survive

Rows 1 and 2 were corrected above. Applying the **same evidentiary standard** to rows 3 and 4 —
*what recorded evidence shows this solves a problem we actually have?* — ends the phase.

**Row 3, the npm package: not filed.** Six arguments were tried and all six fail.

1. **"One install command" is already true, twice.** `README.md:21-30` ships
   `brew install codcod/tap/pickle` and `go install github.com/codcod/pickle@latest`. npm would be
   a *third* channel. The stated requirement was satisfied before it was raised.
2. **No field evidence of install friction.** The only two recorded real-world sessions — the
   `unity` 84-ticket migration and the `snowball` second-child onboarding (both triaged in this
   file) — produced **five filed tickets** (T-049, T-050, T-051, T-052, plus a fold into T-040).
   **Not one concerns obtaining the binary.** All five are post-install semantics.
3. **T-013 says the same from the other side.** Its ten "install polish" items are marker spacing,
   summary labels, CLI coverage, root resolution, a double config load, file modes — **all
   internals of the `install` command, none about the channel.**
4. **Extension delivery is already solved.** `internal/install/install.go:58-59` writes
   `.pi/extensions/docs-readability.ts` and `.pi/extensions/pickle-guardrails.ts` from embedded
   assets. Road A is not a rejected alternative, it is the **shipped mechanism**; npm would replace
   working code, so the burden is on npm to beat it.
5. **The version-contract argument is answered by T-065.** npm+pin's one real edge over road A is
   enforceable binary↔extension coupling. T-065's envelope carries the emitting binary's version so
   a consumer can refuse a dialect it does not understand — drift is handled at runtime regardless
   of delivery.
6. **Per-project pinning has no team to serve, and is circular.** It helps the *second* contributor
   onward; the record shows one external workspace and notes no second contributor. And the pin
   only exists after someone ran `pickle install`, which needs the binary — so it never helps the
   first user.

**Pre-registered triggers — file it when any one fires:** (a) a version-drift incident actually
reported in a pickle-using repo; (b) the extension needs a third-party npm dependency Pi's
built-in imports do not provide; (c) a second contributor hits a mismatch the runtime handshake
cannot resolve.

**Row 4, the Pi extension: not filed — T-057 already owns its one evidenced rule, and already
decided against the extension as the primary mechanism.** Gate by gate:

- **Skill arbitration** — the hazard was *created by row 3* (a packaged copy shadowing the repo
  copy). With no package there is one copy, `.agents/skills/ticket-flow/`. **Moot.**
- **Version handshake** — mechanism now exists in T-065; needs a consumer, and row 4 was to be it.
  Circular.
- **WIP / board-hand-edit gates** — no recorded occurrence. T-052 is the audit *misclassifying*
  staleness as a hand-edit; the feared event has not been observed.
- **Bookkeeping on a `feat/` branch** — the one gate with real evidence: **three occurrences**, one
  *"immediately after the same mistake was written up as a finding"*. It is already **T-057**
  (`medium-high`, the backlog's joint-top grade).

And T-057 has already settled the mechanism against the extension: *"**a pi extension only guards
a pi session.** All three violations above were made by an agent shelling out to `git` outside any
such hook."* Its likely answer is a `pre-commit` hook as the real enforcement — harness-agnostic,
guarding humans and scripts too — with a guardrail rule as the fast in-session explanation. So the
one evidenced gate wants **a fourth rule in the existing 96-line `pickle-guardrails.ts`**, junior
to a git hook, not a new extension architecture.

The shipped extension's own record remains the baseline any expansion must beat: **1 false
positive (T-050), 0 recorded true positives.**

**The general rule this yields, worth keeping:** *anything enforceable in the binary belongs in the
binary* — it works in every harness and is covered by Go tests. An agent-side gate is justified
only for actions that **bypass pickle entirely** (raw `git` in a bash call), which is exactly the
two rules already shipped.

### Where that leaves the programme

**Phase B is not a phase.** Of four proposed tickets: one filed and re-scoped (T-065), one
pre-registered (`schema_version`, on T-056 area 4), two killed with triggers recorded. The
"Pi as best-experience tier" idea reduces, on the evidence, to **T-057** — already filed, already
top-graded, already analysed, and not Pi-specific at all.

**Consequence for T-065, recorded honestly:** its motivating consumer was row 4, which is now not
being filed. Its Description says as much. It keeps standalone value (there is no machine-readable
output at all, and T-056 area 1 would build the projection anyway), but its consumer story is
weaker than at filing, and **refinement should decide whether it stands alone or folds into T-056
work area 1** rather than defend it because it is already on the board.

**Requirement 3 — "close-to-ideal UX" — is the one with field evidence, and it is already on the
board:** T-013, T-051, T-052, T-057. Five findings from real use, none needing a new distribution
channel or a harness tier.

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

## Rick interop — the asks that live upstream (2026-08-07)

Filed T-075 (umbrella) + T-076/T-077/T-078/T-079 for pickle↔`rick` interoperability. Three
things the design wants are **rick's to build, not pickle's**, and they cannot be tickets on
this board: every ticket must target a registered child-project (`audit.go` validates
`project:` against the registry), and `rick` — a separate product in a separate GitLab repo —
should not be registered as one. Registering it would put another team's roadmap under this
board's WIP limits and audit invariants, which is exactly the coupling the interop design
avoids. So they are recorded here:

1. **`rick check artifact <path>`** — a deterministic, exit-coded artifact validator. The
   highest-value ask: it turns T-079's "reimplement rick's schemas in pickle" into "shell
   out". rick already has the muscle — `tools/framework-lint` does frontmatter and structure
   linting over the framework tree; the same idea pointed at `docs/specs/**` artifacts would
   do it. Today those schemas exist only as prose for an LLM in
   `framework/skills/ai-sdlc/artifacts/*/SKILL.md` ("Required Structure and section order",
   "Validation Checklist"), so no non-Claude tool can check them.
2. **An `Amend` verb in the shared approval gate.** `_shared/approval-gate.response.md`
   offers `[A]pprove` / `[R]evise` / `[D]iscard`, and instructs that any other response is
   answered and then the same three options re-presented. A human who edited the artifact
   out-of-band has no path through that gate — the flow cannot acknowledge an amendment it
   did not make. T-079 works around this with a paste-to-revalidate handoff.
3. **Actually write `.ai-sdlc/vmodelsessions/current-phase.json`.** The marker file is
   declared at `sdlc-cli/internal/status/workflow.go` with exactly the schema an external
   tool needs (`Phase`, `Awaiting`, `Step`, `TotalSteps`) — and *nothing in rick writes it*;
   only a test does. Its own comments call it "the normal case until Step 17 ships". If rick
   wrote it, pickle could distinguish "session parked at a gate, safe to edit" from "session
   actively writing", which is the missing arbitration in T-079. Costs rick almost nothing.

**Not asked for: a shared state file, a shared lock, or a shared config.** The interop seam
is deliberately two read-only contracts — `rick status --json` (versioned, additive-only,
`schemaVersion = 2`) and the filesystem path `docs/specs/<KEY>/`, which lines up with pickle
ids for free via `ticket_prefix` (T-058). Anything that requires both tools to agree on a
mutable file is a coupling neither project should accept.

**Also not filed: `brine-v`.** The three enablers (T-073 `flow` key, T-080 lifecycle as data,
T-081 gate table as data) are filed; the second flow itself is not, and should not be until
they land *and* someone asks for it. Filing it now would recreate the checkbox-pluralism
failure the review of this idea warned about — options that cannot be maintained are worse
than options never shipped, and this repo self-hosts brine, so any sibling flow is
undogfooded by construction.

**Adjacent, deliberately out of scope: `pickle verify`.** The single best mechanic in rick is
its verification record bound to *both* HEAD and a worktree digest
(`sdlc-cli/internal/checks/verification.go`), which makes a green build record go stale the
moment the tree moves. brine's "acceptance tests pass" is an honour system by comparison.
This is pure mechanics — pickle's half of the `DESIGN.md` §2 split — and independent of the
naming, interop and brine-v tracks. Worth a ticket if someone wants it; not filed with them.

## T-083 filed (2026-08-07) — the backlog assessment §3 mandates, actually run

**T-083** (`## Outcome` section + a `board audit` warning) was filed from chat after a proposed
"Decision Inputs" ticket section — three tables per ticket, covering cost-of-inaction, RICE-ish
scoring and a scope matrix — was challenged and reduced to its one surviving finding.

**Why the original was rejected, recorded so it is not re-proposed:** it was an
incident-postmortem template aimed at a general-purpose feature flow; its Section 2 introduced a
second prioritisation vocabulary (Reach/Impact/Confidence/Effort) alongside the existing
`impact`/`complexity`/`cost` with no reconciliation rule; its Section 1d verdict line is the
"is this worth it? yes" checkbox **T-064** was dropped for; and it was anchored to a
`## What to add (for refinement)` section that does not exist in this repo's `TEMPLATE.md`.

**The surviving finding is about ticket legibility, not ordering or gating** — which is what
keeps it out of the T-045/T-063/T-064 graveyard. Measured across `1-to-do/` (23 tickets): **9
open with mechanism or provenance, 8 with an outcome, 6 mixed**, and the split correlates with
provenance (review-spawned tickets lead with lineage; field-spawned tickets lead with the
symptom). Second measurement, and the one that makes the fix cheap: the two `impact`
recalibration passes wrote **11 one-line justifications into the tables in this file and into
zero tickets** — `T-055` and `T-038` contain no impact rationale at all, while excellent ones
("cosmetic CSS specificity bug"; "narrow input hardening on a path that already rejects the
dangerous cases") sit here.

**Re-grade pass: nothing moved, and that is the honest result** — the same outcome as the
2026-08-04 pass. T-083 is `medium` ("meaningful quality/consistency win"), joining the 10 other
`medium` rows; distribution is now **high 3 · medium 11 · low-medium 5 · low 5** (24 tickets).
One candidate was considered and declined: **T-081** gains a second prospective customer for its
gate table, since T-083's Item 2 checks *"a `##` section (and its non-emptiness)"* — its exact
unit. Declined on the precedent set on 2026-08-04, which refused to credit prospective demand
when downgrading T-056: **the same rule cuts both ways.** Recorded as a soft coupling in T-083's
Q2 instead, with the sequencing question (T-081 is `depends-on: [T-080]`, an `L` refactor, so
folding a ~25-line warning into it may cost more than it saves) left to refinement.

**Standing caveat that applies to T-083 and was checked:** the pickup queue is READY, not TO DO.
It survives because the 2026-08-03 pass scoped TO DO ordering to *"what gets **refined** next"*,
and refinement-triage is precisely the activity T-083 serves — it changes what a ticket *says*,
not how the board sorts.

## Field finding (2026-08-07) — the bookkeeping guard was inert, and doctor's accepted noise hid it

Observed while committing T-083's bookkeeping on `main`. The commit itself was correct (three
paths, all under `tickets/`, on the base branch), so nothing was mis-committed — but the guard
that was supposed to check it never ran.

**The chain, all verified:**

1. `pickle.toml:9` stamps `payload_version = "v0.2.2-54-g92154e5"` — the workspace was last
   upgraded by a **local build**, 54 commits past the release.
2. That upgrade (`fda096b`, *"refresh hook shim v1->v2"*) installed **shim v2**, whose line 9 is
   `pickle hooks run pre-commit`.
3. The `PATH` binary is Homebrew **0.2.2**, which has no `hooks` subcommand — it exists only in
   the source tree (`internal/cli/cli.go:56`, `internal/cli/hooks.go`).
4. The shim degraded exactly as designed — `pickle: unknown command "hooks"`, then
   `bookkeeping guard skipped (hooks run exited 2)`, `exit 0`. **Inert on every commit in this
   clone**, silently, since `fda096b`.

**The part that is new, and the reason this is written down.** T-046's History already recorded
the same 0.2.2 skew on 2026-08-06 and predicted T-068's new inert-hook warning would "fire for
real" here. It does not fire at all. `pickle doctor` reported `0 error(s), 1 warning(s)`, and the
single warning was the `payload version … differs` line. **T-068's probe and its `checkHooks`
inert branch are shipped in the binary the user does not have** — so the diagnostic for "your
`PATH` pickle is too old" is itself gated behind having a new enough `PATH` pickle. That is a
bootstrap floor on the whole probe design, not a gap in its logic, and no change to `Probe()`
can reach a binary that lacks `Probe()`. The remedies are shim-side (the shim *is* refreshed by
`upgrade`, so it could version-check) or release-side. Noted on **T-071** with an explicit
instruction not to absorb it silently.

**Second-order finding, and the one with a grade attached.** `AGENTS.md` designates the
`payload version … differs` warning as *accepted self-host noise*. On this incident it was the
**only** thing `doctor` said, while a guard sat inert behind it. Accepted noise is not free: it
is the channel real diagnostics arrive on. **T-046 re-graded `low` → `low-medium`** on that
basis — an *observed* incident, so the 2026-08-04 precedent against crediting prospective demand
does not apply (it was applied, correctly, to decline a T-081 re-grade the same day). Only one
notch: the blast radius is this repo alone, since an ordinary workspace has an installed skill
copy and a matching stamp and never sees the noise. Distribution is now
**high 3 · medium 11 · low-medium 6 · low 4** (24 tickets).

**Operational fix, for the record:** `just build && cp pickle "$(command -v pickle)"`, or wait
for Homebrew to catch up. No ticket — this is a stale binary, not a defect.

**The standing lesson this reinforces:** a guard that fails open is the right design, and it is
also the design whose failures you will not notice. When one is installed, something other than
the guard has to assert it is alive — which is precisely what T-068 built, and precisely what
could not run here.

## Self-improvement exploration (2026-08-07) — T-085 filed, two halves noted not filed

Question from chat: *"how can pickle learn from itself — each implemented ticket is an
opportunity to improve the process if the correct data is captured along the way."* The premise
needed one correction, and the correction is the finding: **capture is not the gap.** brine
already records dated transitions, a findings table with severity + disposition + evidence, and
a merge line. What it cannot do is **aggregate or retrieve** any of it.

**The measurements that establish that** (taken over `6-done/`, 36 tickets, all with a
`## Review`):

| signal | state |
|---|---|
| ≈**165** dispositioned findings | groupable by severity and disposition, and by nothing else |
| rework rate | **9 of 36** took a blocking finding — never aggregated |
| review yield | `Disposition summary` present in only **23 of 36** |
| plan-defect rate | `plan amended inline` exists in **exactly one ticket** (T-049) |
| drop corpus | 24 dropped, but ~19 are `absorbed into X` from the one 2026-07-26 epic merge — only ~5 are evidence-based drops |

**The proof that retrieval, not capture, is the failure:** T-045, T-063 and T-064 each proposed
new machinery while the data that would have settled them sat unread in this file (`:95-101`).

**Filed: T-085** (`medium` / low / S-M) — four capture items batched as one theme per §5, each
turning a prose field into a groupable one: a `class` column on the findings table, T-049's
`plan amended inline` line promoted to a rule, `cost` actual-vs-estimate at review, and a
provenance class on the `created … source:` line. It carries a **pre-registered criterion** in
the T-045 style: after 8 further reviews, a class at ≥25% of non-blocking findings is promoted
to a lint; a flat distribution (nothing above 15%) deletes the column and drops the direction.

**Patched: T-065** — an explicit refinement question on whether the JSON projection includes the
`## Review` findings table (History is already in its scope; the findings table is not), with a
middle option to project only the closed-vocabulary columns. T-085 recorded as a second
prospective consumer, and **deliberately not a re-grade**: the 2026-08-04 precedent refuses
prospective demand, it was applied to decline a T-081 bump the same day it was set, and it cuts
both ways.

### Not filed, deliberately — record before re-proposing

Both were in the original five-part proposal and both were cut on the same reasoning: the
backlog holds 24 TO DO tickets with `2-ready/` empty, and filing work whose value T-085's
measurement is supposed to establish is the inversion T-045 was dropped for.

1. **Prior-art surfacing at filing** — `ticket new` printing keyword matches from `7-dropped/`,
   non-gating, print-only. This is the *strongest* of the two: it is the only item on the list
   with a **measured recurrence** (three instances — T-045, T-063, T-064) rather than a
   hypothesis, and it is ~30 lines. It is held only because it is retrieval polish on a corpus
   whose aggregation is not yet built. **File it on the third recurrence after T-085 lands, or
   immediately if a fourth instance appears first.**
2. **A disposition ladder for lessons** — `codified` / `ruled` / `noted` (default) / `dropped`,
   mirroring §5's four finding dispositions, with the promotion target being *"a lesson that can
   be checked mechanically becomes an audit check, not a paragraph"* and a **foreign-workspace
   test** guarding `ruled` (would this help a project that is not pickle?). Held because it is a
   rules paragraph with no mechanism, and this file's standing lesson is that this theme attracts
   exactly that. Its one genuinely load-bearing idea — the foreign-workspace test as an
   anti-overfitting guard on an n=36 self-hosted corpus — is worth more than the ladder around
   it, and is recorded here for whoever next proposes generalising a self-host observation into
   the shipped payload.

### Rejected outright, so they are not re-proposed

- **A metrics command, a retro command, or a dashboard.** T-045's measurement cost one `for`
  loop. Build T-065 and let the queries be ad-hoc.
- **Backfilling classes onto the existing 165 findings.** T-025 precedent — archaeology with no
  consumer. All new capture is prospective only.
- **A scheduled retro ceremony.** Run it as a pass, like the two `impact` recalibrations, not as
  a feature.
- **Anything touching ordering, ranking, scoring or gating.** T-045 / T-063 / T-064.
