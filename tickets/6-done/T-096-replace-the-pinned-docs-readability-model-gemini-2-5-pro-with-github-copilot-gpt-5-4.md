---
id: T-096
title: replace the pinned docs-readability model gemini-2.5-pro with github-copilot/gpt-5.4
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-096 — replace the pinned docs-readability model gemini-2.5-pro with github-copilot/gpt-5.4

## Outcome

After this ships, the docs-readability reviewer (review protocol step 4b, both the pi
`docs_readability` tool/`/docs-readability` command and the opencode `docs-readability`
subagent) defaults to a model that is actually reachable through GitHub Copilot in this
environment, instead of one that has failed every recorded attempt.

## Description

The docs-readability reviewer ships one pinned default model,
`github-copilot/gemini-2.5-pro`, in two independent backends that are meant to behave
identically: the pi extension (`MODEL` constant) and the opencode subagent (`model:` field).
Every recorded attempt to actually invoke it in this environment has failed with
`model_not_supported` / `400` — `opencode models` here lists 60 models across `anthropic`,
`github-copilot`, `gitlab`, `ollama`, `opencode` and **zero** Gemini entries
(`tickets/NOTES.md:190-196`, restated `:276`, `:378`). This has forced a "sanctioned conscious
skip" of step 4b in essentially every review that reached it: T-019, T-022, T-026, T-040,
T-041, T-057, T-068, T-089, T-092, T-093, T-094 (all in `6-done/`). The feature has never
worked in practice, sample size >10, failure rate 100%.

`github-copilot/gpt-5.4` has been confirmed available through the same GitHub Copilot login in
this environment (`pi models` / `opencode models` both list it) — same provider, so no new
login/auth story, just a different model id.

Scope — swap the pinned default and its stale "Gemini" prose everywhere it appears, in both
the shipped payload and this repo's self-hosted dev copy (which cannot be re-synced from the
payload here — no `pickle upgrade` on this repo from a feature branch, per `AGENTS.md`'s
self-modify policy — so both sides need the same hand-edit):

- `agents/pi/extensions/docs-readability.ts` (shipped) and `.pi/extensions/docs-readability.ts`
  (dev copy) — the `MODEL` constant default, the file's doc comment, the tool `label`/
  `description`/`promptSnippet`, and the `/docs-readability` command `description`, all of
  which currently hard-code the word "Gemini".
- `agents/opencode/opencode.jsonc` (shipped) and `opencode.jsonc` (root, dev copy) — the
  `agent.docs-readability.model` field and the surrounding comments (including the
  `opencode models github-copilot | grep -i gemini` confirmation command).
- `.opencode/README.md` and `.pi/README.md` — prose/section headers naming "Gemini" as the
  reviewer and the same confirmation command.
- `skill/resources/docs-readability.prompt.md` (the shared system prompt both backends read)
  is already provider-neutral — no change expected there; confirm during refinement.

Open design question for refinement, given the history above (pinned vendor-specific prose has
now gone stale once already, and `tickets/NOTES.md` explicitly warns model ids rot faster than
pickle releases): should the user-facing strings (tool label, descriptions, README prose) stay
hard-coded to whatever model is currently pinned (mechanical rename, "Gemini" → "GPT-5.4"), or
should they be genericised (e.g. "AI reviewer" / drop the parenthetical vendor name) so the next
model swap is a one-line constant change instead of a repo-wide text hunt? Either is in scope;
the implementation plan should pick one and say why. `DOCS_READABILITY_PROVIDER`/
`DOCS_READABILITY_MODEL` env var overrides already exist for local retuning and are unaffected
either way — this ticket only changes the shipped/self-hosted default.

No behavioural/mechanism change: same provider (`github-copilot`), same subprocess/subagent
plumbing, same shared prompt. Purely a default-value + prose fix.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                          # pickle is the root child-project (path = ".")
git checkout main
git checkout -b feat/T-096-docs-readability-gpt-5-4
```

Commit locally as you go; this is a root-path child, so tidy WIP commits into a small number
of atomic commits before presenting (rules §0), and do not push/open an MR without explicit
user approval.

### Prerequisite gate (hard)

None. `depends-on: []`, tree is clean, no unmerged branch this ticket sits on top of.

### Confirmed design decisions (do not deviate without asking)

1. **Same provider, new model id.** Keep `PROVIDER`/`"model"` at `github-copilot` everywhere;
   change only the default model id, from `gemini-2.5-pro` to `gpt-5.4` (confirmed present via
   `pi models` and `opencode models` in this environment — see Description).
2. **Genericize the vendor-specific prose instead of mechanically renaming it.** Every
   hard-coded occurrence of the word "Gemini" in comments, labels, descriptions and README
   prose is replaced with vendor-neutral wording ("AI reviewer", "model-pinned", or simply
   dropping the parenthetical vendor name) rather than swapped for "GPT-5.4" — per the
   Description's open question, this is the option chosen, precisely so the *next* model swap
   is a one-line constant/`"model"` change, not a repo-wide text hunt like this ticket. Places
   that already interpolate the `MODEL`/`PROVIDER` variables (e.g. the `/docs-readability`
   status line `` `Reviewing … with ${MODEL}…` `` and the injected message header) are already
   correct and need no change.
3. **Drop the `GEMINI_API_KEY`/`google`-provider callout.** It documented an alternate direct
   login path to reach the *old* default model and is not an equivalent alternate for the new
   one; remove it rather than rewording it. Anyone needing a different provider/model uses the
   existing `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` env vars plus their own
   `pi`/opencode login — unaffected by this ticket.
4. **Shipped payload and this repo's self-hosted dev copy get the identical hand-edit, in the
   same commit.** `.pi/extensions/docs-readability.ts` mirrors `agents/pi/extensions/
   docs-readability.ts` byte-for-byte today (verified: `diff` is empty) — keep it that way. The
   two `opencode.jsonc` files are *not* byte-identical baseline (the root copy carries extra
   self-host-specific comments/guardrail rules) — only their shared `docs-readability` agent
   block needs the matching edit, not the whole file. No `pickle upgrade` on this repo (self-modify
   policy) — both sides are edited by hand.
5. `skill/resources/docs-readability.prompt.md` (the shared system prompt) is confirmed
   provider-neutral already (`grep -i gemini` returns nothing) — no edit.

### Tasks

#### Task 1 — pi extension: shipped payload + dev copy

Apply the identical edit to both `agents/pi/extensions/docs-readability.ts` and
`.pi/extensions/docs-readability.ts` (keep them byte-identical afterwards):

- Header doc comment: "Exposes a Gemini readability reviewer" → "Exposes an AI readability
  reviewer"; replace the "Gemini is reached through your `pi` `/login` provider — GitHub
  Copilot by default (same subscription as the opencode backend), or a direct `GEMINI_API_KEY`
  (`google` provider)." paragraph with: "The reviewer is reached through your `pi` `/login`
  provider — GitHub Copilot by default (same subscription as the opencode backend). Override
  via `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` below for any other provider/model
  your `pi` is logged into."; "spawns a Gemini-pinned `pi` subprocess" → "spawns a model-pinned
  `pi` subprocess"; closing "Requires a one-time `pi` `/login` to GitHub Copilot (or `export
  GEMINI_API_KEY=…`)." → "Requires a one-time `pi` `/login` to GitHub Copilot (or whichever
  provider `DOCS_READABILITY_PROVIDER` names)."
- `const MODEL = process.env.DOCS_READABILITY_MODEL ?? "gemini-2.5-pro";` → `?? "gpt-5.4"`.
- The two in-body "Gemini-pinned `pi` subprocess" comments (core helper + the spawn call) →
  "model-pinned `pi` subprocess" / "spawn a model-pinned `pi` in print".
- Thrown-error string: `` `docs-readability: Gemini reviewer (...)` `` → `` `docs-readability:
  reviewer (...)` ``.
- `label: "Docs readability (Gemini)"` → `label: "Docs readability"`.
- Tool `description`: "Ask a Gemini reviewer for readability-only suggestions..." → "Ask an AI
  reviewer for readability-only suggestions...".
- `promptSnippet`: "Get Gemini readability suggestions..." → "Get docs readability
  suggestions...".
- `/docs-readability` command `description`: "Readability-only Gemini review of..." →
  "Readability-only AI review of...".

#### Task 2 — opencode config: shipped payload + dev copy

In both `agents/opencode/opencode.jsonc` and root `opencode.jsonc`, inside the
`docs-readability` agent block and its leading comment only:

- "a read-only Gemini subagent" → "a read-only AI subagent".
- Confirmation-command comment `opencode models github-copilot | grep -i gemini` →
  `opencode models github-copilot` (the `-i gemini` filter no longer finds the target model).
- `"model": "github-copilot/gemini-2.5-pro"` → `"model": "github-copilot/gpt-5.4"`.

Leave every other part of both files (headers, guardrail rules) untouched — they are not part
of this ticket's scope and already differ between the two files by design.

#### Task 3 — dev docs

`.opencode/README.md`: table row "**Native** read-only Gemini subagent" → "**Native** read-only
AI subagent"; section header `## Docs-readability reviewer (Gemini)` → `## Docs-readability
reviewer`; "pinned to a Gemini model reached through" → "pinned to a model reached through";
drop `| grep -i gemini` from the example command.

`.pi/README.md`: table cell "(Gemini via Copilot)" → "(via Copilot)"; "An **optional**
read-only Gemini reviewer" → "An **optional** read-only AI reviewer"; replace "Gemini is
reached via your `pi` `/login` provider (GitHub Copilot by default, or `export
GEMINI_API_KEY=…`)." with "The reviewer is reached via your `pi` `/login` provider (GitHub
Copilot by default; override with `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` for any
other provider/model your `pi` is logged into)."

#### Task 3b — the user manual (added by the pickup applicability gate, finding F1)

`docs/user-manual/cli-reference.adoc` (~lines 123–128), the *only* **user-facing** doc that
names this reviewer — reached from `docs/user-manual.adoc:37` via
`include::user-manual/cli-reference.adoc[]`, so `just docs-check` builds it:

- "a read-only, suggestions-only Gemini reviewer" → "a read-only, suggestions-only AI
  reviewer" (decision 2's genericising, same as everywhere else).
- "Both scaffolds default to `github-copilot/gemini-2.5-pro`" → "`github-copilot/gpt-5.4`".
- Drop the ` | grep -i gemini` filter from the `opencode models github-copilot` example, exactly
  as Task 2 does for the JSONC comment.
- Leave the surrounding paragraphs (the *Step 4b* optional/never-blocking framing, the
  shell-out line for other agents, the conscious-skip sentence) untouched — all still true.

This task exists because the plan as written asserted the opposite ("No change needed to
`docs/user-manual.adoc` — it does not mention docs-readability or its model"), which was false;
the Docs update section below is corrected to match.

#### Task 4 — verify the shared prompt needs no change

`grep -in gemini skill/resources/docs-readability.prompt.md` — confirm no hits (already
verified during refinement); if any surface, generalize them the same way as Task 1/2/3.

#### Task 5 — changelog

Add an entry under `CHANGELOG.md`'s `## [Unreleased]` (new `### Changed` subsection if none
exists yet) noting: the docs-readability reviewer's default model changed from the unreachable
`github-copilot/gemini-2.5-pro` (100% `model_not_supported` failure rate, sample size >10 —
T-019/T-022/T-026/T-040/T-041/T-057/T-068/T-089/T-092/T-093/T-094) to `github-copilot/gpt-5.4`,
in both the pi extension and the opencode subagent (T-096).

### Acceptance test

```
# No stray vendor-specific wording left in the touched files:
grep -rni gemini agents/pi/extensions/docs-readability.ts .pi/extensions/docs-readability.ts \
  agents/opencode/opencode.jsonc opencode.jsonc .opencode/README.md .pi/README.md \
  docs/user-manual/cli-reference.adoc skill/resources/docs-readability.prompt.md
# → prints nothing

# Repo-wide, the ONLY remaining hits are the two that must survive (gate finding F2):
grep -rni gemini . --exclude-dir=.git --exclude-dir=tickets --exclude-dir=dist
# → CHANGELOG.md's *released* history only (an old version section, shipped and immutable —
#   never rewrite it), plus the gitignored ./pickle build artifact, which `just build`
#   regenerates from the edited payload. Nothing else.

# New default model present in all four config/extension files:
grep -n 'gpt-5.4' agents/pi/extensions/docs-readability.ts .pi/extensions/docs-readability.ts \
  agents/opencode/opencode.jsonc opencode.jsonc
# → one match per file

# The pi extension's shipped payload and self-hosted dev copy stay byte-identical:
diff agents/pi/extensions/docs-readability.ts .pi/extensions/docs-readability.ts
# → empty

# CHANGELOG.md records the change:
grep -n 'gpt-5.4' CHANGELOG.md
# → at least one match under [Unreleased]

# Build/test/lint stay green (no .go files touched, but confirm nothing else broke):
just build && just test && just lint

# Board stays clean:
./pickle board audit
```

Manual smoke test (best-effort, requires GitHub Copilot login): run
`/docs-readability <some.md>` in a `pi` session and confirm it no longer fails with
`model_not_supported`.

### Docs update (mandatory when user-facing)

User-facing: this changes the shipped default for every future `pickle install --agent
pi|opencode`. Covered by Task 5 (`CHANGELOG.md` `[Unreleased]`), Task 3 (`.opencode/README.md`,
`.pi/README.md`, the dev-facing docs for this repo's own self-hosted setup) and **Task 3b**
(`docs/user-manual/cli-reference.adoc`, the user manual).

> **Corrected by the pickup applicability gate (finding F1).** This section previously read "No
> change needed to `docs/user-manual.adoc` — it does not mention docs-readability or its
> model." That was false: `docs/user-manual/cli-reference.adoc:123-128` describes the reviewer
> and pins `github-copilot/gemini-2.5-pro` by name, and `docs/user-manual.adoc:37` includes
> that file. Missing user-manual coverage of a changed shipped default would have been a
> blocking documentation finding at review (protocol step 4a.1). `just docs-check` builds the
> manual, so it now covers Task 3b.

### Finish (mandatory)

1. Acceptance test green (above); `just build`, `just test`, `just lint` clean;
   `./pickle board audit` clean.
2. Docs updated and registered (CHANGELOG.md, `.opencode/README.md`, `.pi/README.md`).
3. Write a summary: files touched, the genericize-vs-rename decision taken and why, anything
   deferred.
4. Suggested commit message:

   ```
   fix: pin docs-readability reviewer to gpt-5.4, drop stale Gemini wiring (T-096)
   ```

5. Tidy WIP commits into atomic ones (root-path child, rules §0) before presenting.
6. Commit locally on `feat/T-096-docs-readability-gpt-5-4`. Do not push or open an MR without
   explicit user approval (commit policy: `child_publish_gated = true`). Verify
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing before pushing. Present the commit message; hand back to the user.

## Review

Reviewed 2026-08-12 against `main` at `45a9eff`; code audited on
`feat/T-096-docs-readability-gpt-5-4` at `348890f`. Both `pickle.toml` review addenda are
commented out, so the generic protocol applied alone. The implementer and the reviewer were the
same session, so the implementation/quality/consistency audit was delegated to a **fresh
sub-agent** with no prior context, briefed adversarially and told not to defer to the ticket's
own claims — the same device the pickup applicability gate uses, for the same reason.

- [x] Implementation audit — acceptance test re-run verbatim, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] **Docs-readability pass — RUN, and it worked (step 4b).** First successful Step 4b in this
      project's history; see below.
- [x] Findings recorded with severity **and** disposition per the rules §5 (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit messages presented for approval (step 9)

### Implementation audit

All six tasks (1, 2, 3, **3b**, 4, 5) **met**, in the files the plan named, each quoted "before"
string replaced with the plan's exact "after" wording. All five confirmed decisions
**honoured**: provider unchanged at `github-copilot` (1); prose genericised rather than renamed,
with the `${MODEL}`/`${PROVIDER}` interpolations correctly left alone and no half-rename
mixture (2); the `GEMINI_API_KEY`/`google` callout dropped rather than reworded, leaving no
reference to a dead env var or provider (3); the two `docs-readability.ts` copies byte-identical
(4); `skill/resources/docs-readability.prompt.md` untouched (5).

`just build` · `just test` · `just lint` · `just docs-check` · `pickle board audit` — all green.

Acceptance test re-run verbatim: every command as specified. The model-id asymmetry is right way
round in each file (bare `gpt-5.4` in both `MODEL` consts, `github-copilot/gpt-5.4` in both
`"model"` fields). Both `.jsonc` files still parse, and only the shared `docs-readability` agent
block changed — headers and `permission` rules untouched, as decision 4 requires.

**Beyond the plan's grep-level checks**, the change was verified along the whole path from source
to live answer: the exact invocation the extension spawns (`pi -p … --provider github-copilot
--model gpt-5.4 --no-builtin-tools`) returns a completion at rc 0 where `gemini-2.5-pro` returns
`model_not_supported` at rc 1; `opencode run --model github-copilot/gpt-5.4` likewise answers;
and a real `pickle install --agent pi,opencode` into a throwaway dir (self-modify policy)
scaffolds `gpt-5.4` in both backends with no "Gemini" anywhere in the installed tree.

### Docs-readability pass (step 4b) — the first one that ever ran

This ticket exists because Step 4b had never worked. It now does, which is the strongest
available evidence that the ticket achieved its `## Outcome`. Worth recording precisely, because
the mechanism has a wrinkle a future reviewer will hit:

- Invoking the **session's own** `docs_readability` tool still failed with `model_not_supported`,
  and its error text still said "Gemini reviewer" — i.e. the *pre-T-096* extension. A pi
  extension's `MODEL` const is evaluated when the extension loads, so a session that started
  before the edit keeps the old pin until `/reload` or a fresh session. **This is not a defect
  in the change**; it is the ordinary consequence of editing a loaded extension.
- So the pass was run through the mechanism the *new* extension uses: the shared prompt
  (`skill/resources/docs-readability.prompt.md`) plus the changed files, handed to a
  `gpt-5.4`-pinned `pi -p … --no-builtin-tools` subprocess. It returned nine concrete
  readability suggestions at rc 0.
- Per the protocol, 4b suggestions are polish, not findings: they never enter the table and never
  cause a rework. Of the nine, **two** fell on prose this branch authored and were applied (the
  `.pi/README.md` provider sentence, split in two — folded into R1's fix below; and a reflow in
  the manual, R3). The other seven fell on paragraphs T-096 never touched (`.opencode/README.md`'s
  introduction and prompt-sharing sentences, `cli-reference.adoc`'s `--no-claude`,
  `opencode.jsonc`-policy and per-project paragraphs) and were **discarded as out of scope** —
  §5's causation rule governs polish as much as findings: "did this branch break it?", not "is
  it small?".

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | non-blocking | **fixed inline** | Rewriting `.pi/README.md`'s reviewer paragraph dropped **"approve and"** from "you approve and apply edits" — the load-bearing half of Step 4b, which is a *suggestions-only* reviewer whose output a human approves before anything is applied. The sibling `.opencode/README.md` still said "approves and applies", so the two dev READMEs contradicted each other on the one property that makes the reviewer safe. Task 3 authorised rewriting only the *preceding* sentence, so this was collateral. | `git show main:.pi/README.md` line 77 "you **approve and** apply edits" vs the branch's "you apply edits"; `.opencode/README.md:71` "the flow agent then approves and applies". | Fixed: "approve and" restored, and the over-long sentence this branch introduced split in two (also taking 4b's suggestion on the same passage). Commit `fa47f8b`. |
| R2 | non-blocking | **fixed inline** | Removing `| grep -i gemini` from the example left the shell comment misaligned with the line below it inside the same code block. | `.opencode/README.md:74-75`: comment at col 34 vs col 52 on the next line; aligned on `main`. | Fixed: re-aligned. |
| R3 | non-blocking | **fixed inline** | Editing the manual's paragraph left a 36-character line inside a 66–80-character block. | `docs/user-manual/cli-reference.adoc:127`. | Fixed: reflowed. |
| R4 | non-blocking | **fixed inline** | Genericising the thrown error deleted the noun phrase's article: "docs-readability: reviewer (…) exited 1". | `agents/pi/extensions/docs-readability.ts:89`. | Fixed: "docs-readability: the reviewer (…)". |
| R5 | non-blocking | **fixed inline** | The `CHANGELOG.md` entry claimed every recorded attempt "failed with `model_not_supported`". The *count* (eleven) is accurate, the uniform error string is not — T-026 recorded `Model not found: github-copilot/gemini-2.5-pro` and T-092 a generic "the configured Gemini backend errored". Prose this branch authored, and inaccurate, in a user-facing file. | `tickets/6-done/T-026-*.md:249`; `CHANGELOG.md:32` as written. | Fixed: "failed to reach the model (`model_not_supported` / `Model not found`)". |
| R6 | non-blocking | folded → gate finding F3 | `dist/` build artifacts still embed the old pin. | `grep -c gemini dist/pickle_*/pickle` non-zero; `./pickle` after `just build` is clean (`grep -c gemini pickle` → 0, `strings pickle | grep -c gpt-5.4` → 2). | None — gitignored, regenerated by the release build. Already recorded as F3 at pickup. |

**Disposition summary:** 6 findings — **0 blocking** · 6 non-blocking: **5 fixed inline**
(R1–R5, all prose this branch authored or made false, no behaviour change), **1 folded** (R6 into
pickup finding F3). No new tickets and none `noted`: nothing here passes §5's promotion test, and
nothing was left merely recorded that could be corrected in the branch that broke it.

### Verdict

**Done.** The ticket set out to make a permanently-broken feature work, and the proof is that the
review's own Step 4b ran on it. The premise was checked by invocation rather than by reading a
model list — at the pickup gate, in the implementation, and again here — and the change is
verified end to end: source edit → embedded payload → real `pickle install` → live model
response. Every decision was honoured, including the two judgement calls (genericise rather than
rename; drop the `GEMINI_API_KEY` callout rather than reword it), both of which reduce the cost
of the next model swap.

The one finding worth its own sentence is **R1**: the genericising sweep quietly deleted "approve
and", turning a suggestions-only reviewer into one that reads as though it applies its own edits.
It is documentation-only and non-blocking, but it is exactly the class of error a prose-wide
find-and-replace produces, and it is why the audit was handed to a sub-agent with no stake in the
change rather than re-read by its author.

## History

- 2026-08-12 — created (TO DO). source: pickle ticket new
- 2026-08-12 — TO DO → READY: plan complete
- 2026-08-12 — plan amended at the pickup applicability gate (rules §5 inline disposition; the
  audit ran in a fresh sub-agent, per the skill's pickup procedure). The gate's critical premise
  was confirmed *empirically*, not from the ticket's prose: `pi -p … --provider github-copilot
  --model gpt-5.4` returned a completion (rc 0) while the same call with `gemini-2.5-pro`
  returned `model_not_supported` (rc 1), and `opencode run --model github-copilot/gpt-5.4`
  likewise answered — so both backends' spellings in decision 1 are right and the swap really
  does fix the failure. Three findings: **F1 (blocking)** — the plan omitted
  `docs/user-manual/cli-reference.adoc`, the only *user-facing* doc naming the reviewer, and its
  Docs update section asserted the opposite in so many words; added as **Task 3b**, the false
  claim corrected, and the file added to the acceptance grep. This is an amendment, not an
  invalidation: all five original tasks stand unchanged. **F2 (non-blocking, fixed inline)** —
  `CHANGELOG.md`'s *released* history mentions Gemini and must never be rewritten; the
  acceptance test now states that carve-out explicitly instead of leaving it to the
  implementer's judgement. **F3 (non-blocking, noted)** — the gitignored `./pickle` and `dist/`
  binaries carry the old string in their embedded payload; `just build` regenerates them, no
  action. Also verified unchanged since READY: every quoted "before" string exists verbatim, the
  two `docs-readability.ts` copies are still byte-identical, `skill/resources/docs-readability
  .prompt.md` is still vendor-neutral, no `.go`/testdata/golden file references these strings,
  and the board delta since READY is entirely unrelated T-095/T-097 work
- 2026-08-12 — READY → IN DEVELOPMENT: picked up; plan amended at the applicability gate (F1: user manual omitted)
- 2026-08-12 — IN DEVELOPMENT → IN REVIEW: acceptance green; smoke-tested against the live model on both backends
- 2026-08-12 — published on approval: one atomic commit `348890f` on
  `feat/T-096-docs-readability-gpt-5-4` (8 files; no tidying needed, the work was already a
  single coherent change). **PR #36** open against `main`; history to be kept, not squashed
  (rules §0, root-path child). Pre-push gate verified: `git diff --name-only
  origin/main...HEAD` carries no `tickets/` path, and `origin/main` was not behind local
  `main`. **Merging is the human's**; add the `merged:` line here once #36 lands. Note for the
  reviewer: PR #35 (T-095) is open concurrently and touches disjoint files
  (`internal/cli/changelog.go` and the changelog-check docs paragraphs), so the two are
  independent — but both edit `CHANGELOG.md`'s `[Unreleased]` section, #35 inside the existing
  `### Added` bullet and #36 in a new `### Changed` subsection below it, so whichever merges
  second may want a glance at that hunk
- 2026-08-12 — IN REVIEW → DONE: review: 0 blocking, 5 fixed inline, 1 folded; step 4b ran successfully for the first time
- 2026-08-12 — review's inline fixes published on approval: `fa47f8b` pushed to **PR #36**, a
  fast-forward over `348890f` (no history rewritten). The PR now carries two commits — the
  change, then the review's five `fixed inline` corrections. **Merging is the human's**; add the
  `merged:` line here once #36 lands
- 2026-08-12 — merged to `main` (`bf02f33`, PR #36): 2 commits kept, CI green
