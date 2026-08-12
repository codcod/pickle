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
  skill/resources/docs-readability.prompt.md
# → prints nothing

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
pi|opencode`. Covered by Task 5 (`CHANGELOG.md` `[Unreleased]`) and Task 3 (`.opencode/README.md`,
`.pi/README.md`, the dev-facing docs for this repo's own self-hosted setup). No change needed to
`docs/user-manual.adoc` — it does not mention docs-readability or its model (`just docs-check`
stays green because nothing there changed).

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

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: pickle ticket new
- 2026-08-12 — TO DO → READY: plan complete
