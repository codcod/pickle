---
id: T-022
title: payload states commit policy, branch/ticket prefixes and WIP limits unconditionally; §8 pickup gate reads as freshness, not merit
project: pickle
depends-on: []
spawned-by: []
impact: medium-high
complexity: low
cost: S
---

# T-022 — payload states commit policy, branch/ticket prefixes and WIP limits unconditionally; §8 pickup gate reads as freshness, not merit

## Description

Non-blocking finding 8 from the T-018 review. **Line references re-verified against the tree at
refinement (2026-08-05)** — the filed ones had all drifted; the defect had not.

The embedded skill — which ships into **every** installed project — states as absolutes several
things that are in fact per-project configuration, and it now sits directly beside an `AGENTS.md`
marker block that renders the project's *real* values (`internal/install/install.go:796-877`).
When a project is configured non-default the two surfaces contradict each other, and both read as
authoritative.

1. **Commit policy**, stated as an absolute in **nine** places: `skill/SKILL.md:3` (the
   frontmatter description every agent loader reads — "pushing a child-project requires explicit
   user approval"), `skill/SKILL.md:190-191`, `:223-225`;
   `skill/resources/tickets-README.md:199-202`; `skill/resources/review-protocol.md:33-35`,
   `:183-188`, `:211`; `skill/resources/TEMPLATE.md:46-49`, `:102-104`. A project with
   `child_publish_gated = false` gets a marker block saying commit and push as the work needs
   (`install.go:833-836`) and a skill saying never push.
2. **Branch prefix** — `feat/` hardcoded in the *rules* prose: `skill/SKILL.md:81`, `:185`,
   `:201`; `skill/resources/tickets-README.md:47`, `:68`, `:99`, `:149`, `:198`, `:309`;
   `skill/resources/review-protocol.md:31`; `skill/resources/TEMPLATE.md:42`, `:121`.
   (`skill/SKILL.md:69` sits under a `Defaults:` heading and is fine as-is.)
3. **WIP limits** — `skill/SKILL.md:108` states `≤ 1` as a rule, with no "tune per project"
   escape. (`:74` is under the same `Defaults:` carve-out.)
4. **Ticket-id prefix — new since filing, same defect.** T-058 shipped per-child `ticket_prefix`,
   and the marker block renders a branch as `<branch_prefix><ticket_prefix>-NNN-<slug>`
   (`install.go:821`). Every literal `T-NNN` in the payload's rules prose is therefore wrong in a
   child that sets `ticket_prefix`, for exactly the reason `feat/` is. The ticket predates T-058
   and did not carry this.

All four are genuinely configurable — `internal/config/config.go:41-46` (`DefaultBranchPrefix`,
`DefaultTicketPrefix`, `DefaultWIPInDevelopment`, `DefaultWIPInReview`) and `:68-84`
(`child_publish_gated`, `overarching_auto`, `branch_prefix`, `wip_*`, `ticket_prefix`).

`skill/resources/tickets-README.md:202` already models the correct hedge — *"The project's
`AGENTS.md` / `pickle.toml` states specifics."* — and `:290` hedges WIP with *"(tune per
project)"*. The work is to apply that deferral consistently to the rules/procedure occurrences
**without** turning every sentence into a caveat: the skill should state the flow's defaults once
per file and point at the marker block for the project's actual values.

Note this is a payload change: `pickle upgrade` propagates it into every installed project, and in
this repo `.agents/skills/ticket-flow/` is a symlink to `skill/`, so editing it here changes what
the binary ships **and this repo's own operating rules, mid-ticket**.

Soft couplings: **T-018** (made the marker block render real values, creating the contradiction);
**T-058** (added `ticket_prefix`, i.e. item 4); **T-066** (edits
`skill/resources/tickets-README.md:122-123` for a different defect — same file, disjoint lines).
~~T-016~~ was dropped (absorbed into T-009); the coupling filed here is dead.

### Adjacent item A — the §8 "freshness" heading (**in scope**, decided at refinement)

A separate §8 wording defect was measured on 2026-08-01 (T-064, dropped; full evidence in
`NOTES.md`). It is **not** an instance of this ticket's "unconditional statement that should defer
to config" problem, but it edits the same two files and costs minutes alongside the work above, so
it is taken here and the title is widened for it:

- `tickets-README.md:327-336` heads the pickup applicability gate **"Pickup is gated by a
  freshness check"** and justifies it purely by aging, while its actual mandate is *"the ticket's
  own assumptions plus the board delta"* tested for *"true, **required**, and **worth it**"*
  (`SKILL.md:177-178`). Practice has followed the heading, not the mandate: **0 negative verdicts
  in ~15 recorded runs**. Two sentences fix it — say the scope includes assumptions **that were
  wrong at filing**, and that **DROP is a legal verdict**. Verified at refinement: `2-ready →
  7-dropped` is already legal (`internal/move/move.go:33`), so this adds **no mechanism**.

### Adjacent item B — suggested model tier per flow step (**out of scope**, decided at refinement)

Tier 0 of the 2026-08-03 model-tier exploration was folded in here on 2026-08-03. At refinement it
is **judged out of scope and left in `NOTES.md`**, per this ticket's own instruction not to file
it:

- it is a **different theme** — "the payload says nothing about X" rather than "the payload states
  per-project config as universal" — and widening the ticket to two themes costs its `S`;
- the evidence in this very ticket argues against it: item A measures **0 negative verdicts in
  ~15 runs** of a gate whose prose already asks for exactly the behaviour it did not get. Adding
  another advisory, unenforceable prose block is the same bet that just lost.

Still open in `NOTES.md` and **not** taken here either: step 4b's pinned
`github-copilot/gemini-2.5-pro` default, which resolves to nothing in a legitimate Copilot login.
It is genuinely this ticket's defect class (payload asserting a configuration the environment may
not offer), but the fix is a **code** change in `agents/opencode/opencode.jsonc:30` and
`agents/pi/extensions/docs-readability.ts:44-46` plus an env override — it breaks this ticket's
prose-only shape and its `S` cost. It stays parked, undecided, in `NOTES.md`.

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git pull --ff-only
git checkout -b feat/T-022-payload-defers-to-project-config
```

The target child-project is `pickle`, registered at the repo root (`.`), so the branch is cut in
this repo. Local WIP commits encouraged; **no push and no merge request without explicit user
approval** (this project's commit policy, `pickle.toml` → `child_publish_gated = true`).

### Prerequisite gate (hard)

- Clean tree on `main`, `just build`/`just test`/`just lint`/`just docs-check` green **before**
  the first edit — this ticket rewrites the prose those tests read, so a pre-existing failure must
  not be mistaken for one of ours.
- `depends-on:` is empty; nothing to await.

### Confirmed design decisions (do not deviate without asking)

**D1 — one precedence rule per file, hedges only at the high-risk sites.** Do **not** caveat every
sentence. Add the precedence note once to each of the four payload files, and add an inline hedge
only where the text reads as a *rule or instruction* (the sites listed in Tasks 2–4). Everywhere
else the concrete default text stays exactly as it is.

The precedence note, adapted in wording per file but identical in content, and containing the
literal anchor string **`Project configuration wins`** (Task 6 asserts on it):

> **Project configuration wins.** Where this document names a branch prefix, a ticket-id prefix, a
> WIP limit or a commit policy, it states the flow's **default**. The project's `AGENTS.md` marker
> block renders what is actually configured in `pickle.toml` for each child — it wins on any
> disagreement.

**D2 — the frontmatter description (`skill/SKILL.md:3`).** Replace the trailing clause *"pushing a
child-project requires explicit user approval"* with *"publishing a child-project follows the
project's configured commit policy (default: explicit user approval)"*. Change nothing else on
that line: it is the one line every agent loader reads, and its length is a cost.

**D3 — keep the concrete branch literal.** `feat/T-NNN-<slug>` stays as the running example
throughout; do **not** substitute a `<branch_prefix><PREFIX>-NNN-<slug>` placeholder — it is
unreadable and the payload's examples depend on the concrete form. Only sites phrased as a *rule*
are reworded (Task 3). The precedence note (D1) explicitly names **both** the branch prefix and
the ticket-id prefix, which is what covers the remaining literals.

**D4 — scope.** Adjacent item **A** is in (Task 5); adjacent item **B** and the step-4b pinned
model are **out**, for the reasons recorded in the Description. Do not file either as a ticket:
both are already recorded in `tickets/NOTES.md`.

**D5 — a narrow guard test.** The new test asserts (i) the presence of the anchor string in all
four payload files and (ii) a **short** blocklist of exact retired phrasings. Do **not** grow it
into a regex sweep for every "never push": the payload legitimately states defaults in that
language, and a broad matcher would fail on correct prose. Model it on the existing
`TestPayloadDispositionVocabulary` (`internal/install/install_test.go:652-694`), including its
doc-comment style — say *why* each assertion exists.

**D6 — self-modify policy (from `AGENTS.md`).** `.agents/skills/ticket-flow/` is a symlink to
`skill/`, so these edits change this repo's own operating rules while the ticket is in flight.
**Never run `pickle install|upgrade|uninstall` against this repo.** The one install performed is
Task 7's throwaway-directory check. No marker block changes: `MarkerBlock()` is not touched.

**D7 — no behaviour change.** This ticket changes payload prose, one docs file, and adds one test.
No change to any non-test `.go` file. `payload_version` is the build version stamped by
`-ldflags` (`main.go:26-36`); nothing to bump by hand.

### Tasks

#### Task 1 — add the precedence note to each payload file (D1)

One short block per file, placed where a reader meets the defaults:

- `skill/SKILL.md` — at the end of the `## Project configuration (in pickle.toml + the AGENTS.md
  marker block)` section, directly after the four `Defaults:` bullets (after line 74).
- `skill/resources/tickets-README.md` — at the end of `## 0. Child-projects (the multi-project
  model)` (after line 49), whose bullets already name the per-child config.
- `skill/resources/review-protocol.md` — inside the opening scope blockquote, adjacent to the
  "Review on the ticket's feature branch" paragraph (lines 30-35).
- `skill/resources/TEMPLATE.md` — in the `### 0. Feature branch (mandatory)` block, after the
  commit-policy sentence (lines 45-51).

#### Task 2 — de-absolutise the commit policy at the nine sites (D1, D2)

- `skill/SKILL.md:3` — apply D2 verbatim.
- `skill/SKILL.md:190-191` (implement, step 8) and `:223-225` (validate, step 4) — qualify with
  *per the project's commit policy (default: …)*; keep the default text.
- `skill/resources/tickets-README.md:199-202` (READY gate item 1) — it already ends with the
  correct hedge; make the **preceding** clause read as the default rather than as an absolute.
- `skill/resources/review-protocol.md:33-35` (scope blockquote), `:183-188` (step 9), `:211`
  (checklist line) — same treatment; the checklist line stays one line.
- `skill/resources/TEMPLATE.md:46-49` and `:102-104` — same treatment. TEMPLATE prose is copied
  into every new ticket, so keep it short.

#### Task 3 — reword the rule-shaped branch-prefix sites only (D3)

- `skill/resources/tickets-README.md:198` — "**Feature branch** named (`feat/T-NNN-<slug>`)" →
  named per the child's configured branch and ticket prefixes, with `feat/T-NNN-<slug>` kept as
  the stated default/example.
- `skill/SKILL.md:81` — "its `feat/` branch is cut in that child's repo" → "its feature branch".
- Leave `SKILL.md:185`, `:201`; `tickets-README.md:47`, `:68`, `:99`, `:149`, `:309`;
  `review-protocol.md:31`; `TEMPLATE.md:42`, `:121` **as they are** — they are examples and
  narrative, and D1's note now covers them.

#### Task 4 — WIP limits (D1)

- `skill/SKILL.md:108` — mark the `≤ 1` figures as defaults read per child from `pickle.toml`,
  matching the hedge `tickets-README.md:290` already carries. Leave `:290` alone.

#### Task 5 — adjacent item A: the §8 gate heading (D4)

- `skill/resources/tickets-README.md:327` — retitle **"Pickup is gated by a freshness check"** to
  name the real test (applicability/merit), and add the two sentences: the scope includes
  assumptions that were **wrong at filing**, not only ones that **aged**; and **DROP is a legal
  verdict** (`2-ready/ → 7-dropped/`, already permitted — `internal/move/move.go:33`), alongside
  proceed and route-back.
- `skill/SKILL.md:175-183` (implement, step 3) — add DROP to the routing options presented for
  approval, in one clause. Add **no** mechanism, no new command, no new frontmatter.

#### Task 6 — the guard test (D5)

Add `TestPayloadDefersToProjectConfig` to `internal/install/install_test.go`, beside
`TestPayloadDispositionVocabulary`, reusing its `payloadRoot()` helper and its four-file list:

1. **Anchor present.** Each of `skill/SKILL.md`, `skill/resources/tickets-README.md`,
   `skill/resources/review-protocol.md`, `skill/resources/TEMPLATE.md` contains
   `Project configuration wins`.
2. **Retired phrasings gone** (exact strings, kept short and few):
   - `pushing a child-project requires explicit user approval` — anywhere in the four files (the
     D2 clause; it is the agent-loader-visible lie);
   - `Pickup is gated by a freshness check` — in the rules (item A).
3. Doc-comment explains that the four fields are per-child config
   (`internal/config/config.go:41-46`) rendered by `MarkerBlock` (`internal/install/install.go:821-836`),
   so payload prose stating them as absolutes contradicts the marker block in any non-default
   project — and that the blocklist is deliberately short (D5).

#### Task 7 — verify the payload as installed, not just as edited (D6)

The edits are to `skill/`, but what matters is what a *fresh install* lays down. Verify in a
throwaway directory with a copied binary, never by running the repo binary against this repo.

### Acceptance test

Run from the repo root, in order; every step must pass.

```
just build
just test
just lint
just docs-check
```

```
# 1. The new guard test passes, and by name.
go test ./internal/install/ -run TestPayloadDefersToProjectConfig -v

# 2. The guard test actually guards: reintroduce a retired phrase and watch it fail.
#    (Restore the file afterwards — `git checkout` the path.)
#    Expect FAIL for each injected string, then PASS again after restore.

# 3. The precedence note ships in a real install (Task 7).
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && git init -q . \
  && ./pk install --project demo --path . \
  && grep -rl "Project configuration wins" .agents/skills/ticket-flow/ \
  && grep -c "Project configuration wins" .agents/skills/ticket-flow/SKILL.md
# Expect: all four payload files listed by the first grep; count 1 in SKILL.md.

# 4. A non-default child no longer contradicts itself. In the same throwaway dir:
#    set child_publish_gated = false in pickle.toml, re-render the marker block via
#    `./pk project add second ./second` (or `pickle doctor`), and confirm AGENTS.md says
#    "not publish-gated" while the installed SKILL.md frontmatter no longer asserts that
#    pushing requires approval:
grep -n "not publish-gated" AGENTS.md
grep -c "pushing a child-project requires explicit user approval" \
  .agents/skills/ticket-flow/SKILL.md   # expect 0

# 5. This repo's own board is untouched by the edit.
cd - && ./pickle board audit          # expect no errors
```

Manual check, because no test can express it: read `git diff skill/` end to end and confirm the
prose still reads as instructions rather than as a list of caveats (D1). If any file has gained
more than the one precedence note plus its listed inline hedges, it has overshot.

### Docs update (mandatory when user-facing)

`docs/user-manual/configuration.adoc` — in the per-child key list (lines 48-57):

1. Add the **one sentence** the user asked for: the `AGENTS.md` marker block is the authoritative
   *render* of these keys for agents, and the shipped skill states only the defaults — so after
   changing `branch_prefix`, `ticket_prefix`, `wip_*` or `[commit]`, the marker block is what the
   agent reads.
2. Add the missing **`ticket_prefix`** bullet (default `T`). It is absent from the list although
   T-058 shipped it, and the sentence above references it, so it is fixed here rather than left
   dangling. **Note for T-066's refinement:** that ticket owns `cli-reference.adoc`; this bullet is
   in `configuration.adoc` and is taken here — do not double-fix.

No other manual page changes: `configuration.adoc:55` already documents `branch_prefix (default
feat/)` correctly, and the manual is not the surface carrying the defect. `README.md` and
`DESIGN.md` are unaffected.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (`docs/user-manual/configuration.adoc`) and `just docs-check` green.
3. Write the summary: files touched, the four-field deferral applied, item A taken, item B and the
   step-4b model pin explicitly **not** taken (and why), and the `ticket_prefix` docs bullet noted
   for T-066.
4. Suggested Conventional Commit message:

   ```
   docs(skill): defer per-project config to the marker block; name the pickup gate's real test (T-022)

   The payload stated commit policy, branch prefix, ticket prefix and WIP limits
   as absolutes while the AGENTS.md marker block renders each project's real
   values, so any non-default project shipped two authoritative, contradicting
   surfaces. State the defaults once per file, defer to the marker block, and
   hedge only the rule-shaped sites. Also retitle §8's pickup gate from a
   "freshness check" to the merit test its own mandate describes, and record that
   DROP is a legal verdict there (already permitted by move.go; no mechanism
   added). Guarded by TestPayloadDefersToProjectConfig.
   ```

5. Commit locally on the ticket branch; **do not push or open a merge request without user
   approval**. Present the commit message; only after approval finalize (squash or keep history),
   push, and open the MR — merging is the human's. Then
   `pickle ticket move T-022 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-08-03 — re-graded medium → **medium-high** by the second impact recalibration pass: same
  defect class as T-041 (which was graded `high` for it) — the payload contradicts the marker
  block's real values, so agents act on wrong project config — one notch down because it bites
  only non-default configurations. Now the top of the TO DO group at complexity low / cost S.
- 2026-08-03 — adjacent item B added (Tier 0 of the model-tier exploration, `NOTES.md`): an
  advisory suggested-tier-per-step block in the payload, constrained to prose — no schema, no
  per-agent scaffolds, no enforcement. Filed here rather than as its own ticket because it fails
  the promotion test alone and edits files this ticket already opens.
- 2026-08-05 — refined. Description re-verified against the tree: every filed line reference had
  drifted and was corrected; the defect count rose from 5 to 9 commit-policy sites; a fourth field
  (`ticket_prefix`, shipped by T-058 after filing) was added; the dead T-016 coupling was struck
  and T-058/T-066 couplings added. Decisions D1–D7 confirmed with the user. **Item A taken**
  (title widened for it); **item B judged out of scope** and left in `NOTES.md` per this ticket's
  own instruction, on the evidence that item A measures an unenforceable prose gate at 0/15. The
  step-4b pinned-model item stays parked: same defect class, but a code change, not prose.
  Grades unchanged (medium-high / low / S).
- 2026-08-05 — TO DO → READY: plan complete
- 2026-08-05 — READY → IN DEVELOPMENT: picked up
