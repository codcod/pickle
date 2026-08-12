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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: pickle ticket new
