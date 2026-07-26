---
id: T-009
title: agent enablement — --agent contract, opencode + pi wiring, shipped docs-readability reviewer
project: pickle
depends-on: [T-004]
spawned-by: []
impact: high
complexity: high
cost: L
---

# T-009 — agent enablement: `--agent` contract, opencode + pi wiring, shipped docs-readability reviewer

## Description

Make `pickle install --agent claude,opencode,pi` install support for every named agent, and
ship the **docs-readability reviewer** as part of the install — the same mechanism proven in
the translator workspace (and mirrored by this repo's own dev tooling), so it can be re-used
by any pickle-driven project.

**Absorbed tickets** (both dropped in favour of this one, per user decision 2026-07-26):

- **T-010 — Pi guardrail scaffold**: the `.pi/` scaffold (`--agent pi`) now lands here,
  including T-010's **symmetry obligation** from the T-006 review — anything install lays down
  must be refreshed by `upgrade`, removed by `uninstall`, and checked by `doctor`, following
  T-006's decision-D6 pattern of *probing the filesystem* (no agent registry is persisted to
  `pickle.toml`).
- **T-016 — ship docs-readability as an optional review step (Step 4b)**: both halves land
  here — the optional **Step 4b prose** in the shipped review protocol *and* the reviewer
  mechanics (T-016's "option 2": scaffold on install).

**Confirmed scope (user decisions, 2026-07-26):**

- `--agent` takes a comma-separated set (`claude`, `opencode`, `pi`); **no autodetection**;
  `--no-claude` is **deprecated in favour of `--agent`**.
- The reviewer works **the way it works in the translator workspace**: one shared prompt, an
  opencode subagent (`agent.docs-readability` in `opencode.jsonc`) and a pi extension exposing
  a `docs_readability` tool. It must be reachable from **pi, zed, and opencode sessions** (zed
  and other agents shell out to `opencode run --agent docs-readability …`); Claude sessions are
  **not** required to reach it.
- Ship `github-copilot/gemini-2.5-pro` as the working default model, with a comment/README note
  to adjust via `opencode models`.
- `opencode.jsonc` is written whole **only if absent**; if one exists, print the blocks to merge
  manually — never parse/merge user JSONC.
- Guardrails ship too: the generic flow rules (explicit pathspecs, publish gate) as
  `permission.bash` globs for opencode and as a pi extension.

**Key facts the plan builds on:** opencode picks up `AGENTS.md` and
`.agents/skills/<name>/SKILL.md` **natively** — both are already laid down by `pickle install`,
so the only genuinely opencode-specific artifacts are the `opencode.jsonc` guardrail +
subagent config. Pi has no permission system or subagent config, so its slice is the two
extensions. The reference implementations are this repo's own dev tooling (`opencode.jsonc`,
`.pi/extensions/docs-readability.ts`, `.pi/extensions/workspace-guardrails.ts`,
`.agents/docs-readability.prompt.md`) — adapted, not shipped verbatim (the dev-repo
self-modify guard does not apply to installed projects).

Soft couplings: **T-013** (install polish) touches the same CLI surface; **T-041** (marker
freshness) is unaffected — agent choices are not rendered into the marker block.

## Implementation Plan

### 0. Feature branch (mandatory)

Before any change, create a feature branch inside the target child-project's repo (the
`pickle` repo root — the child is the repo itself):

```
git checkout main
git checkout -b feat/T-009-agent-enablement
```

Do all work on this branch, committing locally as you go (WIP commits are encouraged).
**Never push or open a merge request without explicit user approval** (commit policy): end
with a summary and a suggested commit message (see Finish); after approval, finalize the
branch, push, and open the MR — merging is always the human's.

### Prerequisite gate (hard)

- T-004 (install) is in `6-done/` and merged to main (33f05e3) — satisfied.
- `3-in-development/` is empty for `pickle` (WIP ≤ 1) — verify at pickup.
- **Self-modify policy** (AGENTS.md): never run the built binary's
  `install`/`upgrade`/`uninstall` against this repo; all acceptance runs use a throwaway dir
  with the binary copied in.

### Confirmed design decisions (do not deviate without asking)

1. **`--agent` contract.** Comma-separated set; legal values `claude`, `opencode`, `pi`;
   default when omitted: `claude` (today's behaviour). Unknown value → usage error listing the
   legal values. **No autodetection.** `--no-claude` is deprecated: still honoured (removes
   `claude` from the set) but prints a deprecation warning pointing at `--agent`.
   `--claude-symlink` is unchanged and meaningful only when `claude` is in the set.
2. **Reviewer mechanism = translator's.** One shared prompt; two backends: the opencode
   subagent (`agent.docs-readability`) and the pi extension (a `docs_readability` tool +
   `/docs-readability` command). Other sessions (zed, etc.) reach the same reviewer via
   `opencode run --agent docs-readability …`; Claude sessions may consciously skip it.
3. **The prompt ships in the payload** at `skill/resources/docs-readability.prompt.md`; both
   backends reference it as `.agents/skills/ticket-flow/resources/docs-readability.prompt.md`.
   This deliberately reverses the earlier "keep the prompt out of `skill/`" decision; pickle's
   own dev tooling is repointed at the shipped copy (one canonical prompt, resolved through the
   self-host symlink).
4. **Model default:** `github-copilot/gemini-2.5-pro`, temperature 0.2, read-only permissions
   (`edit`/`bash`/`webfetch`/`websearch` denied). A comment in the scaffold + the README explain
   how to adjust (`opencode models github-copilot | grep -i gemini`). The *shipped protocol
   prose* stays provider-neutral.
5. **`opencode.jsonc` write policy.** Created whole only if absent (with explanatory comments);
   if present, install **skips it** and prints the `agent` + `permission` blocks for manual
   merge — never parse or merge JSONC. Once created it is **user-owned**: `upgrade` never
   touches it; `uninstall` removes it **iff byte-identical to the shipped template**, otherwise
   leaves it in place with a note; `doctor` performs **no** opencode checks.
6. **Pi scaffold files are pickle-owned.** `.pi/extensions/docs-readability.ts` and
   `.pi/extensions/pickle-guardrails.ts`: install writes them; `upgrade` refreshes them **if
   present** (probe the filesystem — T-006 D6 pattern; nothing is persisted to `pickle.toml`);
   `uninstall` removes them and prunes `.pi/extensions/`/`.pi/` if left empty; `doctor` warns
   when a present file drifts from the embedded asset ("run `pickle upgrade`"). File headers
   say so; user customizations belong in sibling extension files.
7. **Shipped guardrails are the generic flow rules only**: (a) explicit-pathspec staging
   discipline — deny `git add -A/--all/.`, `git commit -a/--all`; (b) child publish gate — ask
   on `git push`, `glab mr create`, `gh pr create`. The dev repo's self-modify guard does
   **not** ship (running `pickle upgrade` inside an installed project is legitimate).
8. **Step 4b** is added to the shipped `skill/resources/review-protocol.md` after §4a:
   optional, suggestions-only, never blocking; skipping (e.g. Claude-only review) is a
   sanctioned, recorded skip in `## Review`. Provider-neutral wording; names the pi tool, the
   opencode subagent, and the `opencode run` shell-out as example hosts.
9. **Agent assets live in a new embedded tree `agents/`** at the repo root
   (`//go:embed all:agents` in `assets.go`), keeping the `skill/` payload purely the skill —
   the prompt being the one deliberate exception (decision 3).

### Tasks

#### Task 1 — ship the prompt in the payload; repoint the dev tooling
- `git mv .agents/docs-readability.prompt.md skill/resources/docs-readability.prompt.md`.
- Repoint the dev tooling at the shipped copy (resolved via the self-host symlink):
  `opencode.jsonc` → `"prompt": "{file:./.agents/skills/ticket-flow/resources/docs-readability.prompt.md}"`;
  `.pi/extensions/docs-readability.ts` → `PROMPT_PATH`.
- Update the now-obsolete "kept OUT of `skill/` so it never ships" comments/links in
  `opencode.jsonc`, `.pi/extensions/docs-readability.ts`, `.pi/README.md`,
  `.opencode/README.md` — the prompt now ships by design.

#### Task 2 — embedded agent assets (`agents/` tree)
- `agents/opencode/opencode.jsonc` — the scaffold template: `$schema`;
  `agent.docs-readability` per decision 4 (prompt path per decision 3);
  `permission.bash` guardrails per decision 7 (`"*": "allow"` first; header comment explains
  last-match-wins and how to change the model).
- `agents/pi/extensions/docs-readability.ts` — adapted from `.pi/extensions/docs-readability.ts`
  (generic header per decision 6; prompt path per decision 3; keep the
  `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` env overrides).
- `agents/pi/extensions/pickle-guardrails.ts` — adapted from
  `.pi/extensions/workspace-guardrails.ts`, rules 1+2 only (decision 7).
- `assets.go`: add `//go:embed all:agents` (new var); plumb it into `internal/cli` alongside
  `Payload` so `internal/install` receives both FS trees.

#### Task 3 — `--agent` flag (`internal/cli/install.go`)
- Replace the reserved `--agent` string with real parsing: split on `,`, trim, validate against
  the legal set, default `claude`; usage error on unknown values.
- `--no-claude`: keep working (drop `claude` from the set) + stderr deprecation warning.
- Pass the set via `install.Options`.

#### Task 4 — install (`internal/install/install.go`)
- Replace `Options.Claude bool` with the agent set (keep `ClaudeLink`); add path constants
  (e.g. `PiExtensionsDir`, `OpencodeConfigFile`, and the two pi filenames).
- opencode slice: absent `opencode.jsonc` → write the template (`res.created`); present →
  `res.skipped` + print the manual-merge guidance (decision 5).
- pi slice: `MkdirAll(.pi/extensions)`, write both files (create or refresh in place).

#### Task 5 — upgrade/uninstall symmetry (`internal/install/install.go`)
- `Upgrade`: probe for the two pi files; refresh each that is present from the embedded assets.
  Never touch `opencode.jsonc`.
- `Uninstall` (incl. `--dry-run`): remove present pi files; prune `.pi/extensions/` and `.pi/`
  when left empty; remove `opencode.jsonc` iff byte-identical to the shipped template,
  otherwise `res.skipped("opencode.jsonc (user-modified, left in place)")`.

#### Task 6 — doctor (`internal/doctor/doctor.go`)
- New check: for each pi extension file that exists, compare against the embedded asset; on
  drift emit a **warning** naming the file and suggesting `pickle upgrade`. Absent files are
  not findings. No opencode checks (decision 5).

#### Task 7 — Step 4b in the shipped review protocol (`skill/resources/review-protocol.md`)
- Insert `## 4b. Docs-readability pass (optional)` after §4a, per decision 8, and add a
  checklist line to the paste-in checklist ("step 4b, optional — record a conscious skip if no
  reviewer is available").
- Sweep `skill/SKILL.md` and `skill/resources/tickets-README.md` for step-number references
  that should now mention 4b; update pickle's own review addendum in `pickle.toml` only if it
  keys on affected step numbers.

#### Task 8 — tests
- `internal/install` tests: full-set install lays down every artifact; default install lays
  down none of the new ones; pre-existing `opencode.jsonc` is untouched; `Upgrade` refreshes a
  drifted pi file and skips absent ones; `Uninstall` removes a pristine `opencode.jsonc` but
  preserves a modified one; dry-run lists without mutating; empty-dir pruning.
- `internal/cli/cli_test.go`: `--agent` parsing (valid combos, unknown value → usage exit,
  default = claude-only), `--no-claude` deprecation warning.
- `internal/doctor` test: drift warning fires on a modified pi file, silent when absent/current.

#### Task 9 — README
- Document the `--agent` contract (values, default, `--no-claude` deprecation), a per-agent
  artifact table (claude: symlink+CLAUDE.md; opencode: `opencode.jsonc`; pi:
  `.pi/extensions/*`), the reviewer (what it does, default model + how to change it, how each
  session type reaches it incl. `opencode run` for zed), and the `opencode.jsonc`
  absent/present policy.

### Acceptance test

```sh
set -euo pipefail
REPO=$(pwd)
just build && just test && just lint

# 1. Full-set install into a throwaway dir (self-modify policy: copied binary).
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
./pk install --project demo --agent claude,opencode,pi
test -f .agents/skills/ticket-flow/resources/docs-readability.prompt.md
grep -q '"docs-readability"' opencode.jsonc && grep -q 'gemini-2.5-pro' opencode.jsonc
grep -q 'git push' opencode.jsonc                      # guardrails present
test -f .pi/extensions/docs-readability.ts
test -f .pi/extensions/pickle-guardrails.ts
test -L .claude/skills/ticket-flow
grep -q '4b' .agents/skills/ticket-flow/resources/review-protocol.md
./pk doctor

# 2. A pre-existing opencode.jsonc is never touched; guidance is printed.
D2=$(mktemp -d) && cp "$REPO/pickle" "$D2/pk" && cd "$D2"
printf '// mine\n{}\n' > opencode.jsonc
./pk install --project demo --agent opencode | grep -qi 'opencode.jsonc'
grep -q '// mine' opencode.jsonc

# 3. Uninstall symmetry: pristine scaffold fully removed.
cd "$D" && ./pk uninstall
test ! -e opencode.jsonc
test ! -e .pi/extensions/docs-readability.ts && test ! -e .pi/extensions/pickle-guardrails.ts

# 4. Modified opencode.jsonc survives uninstall.
cd "$D2" && ./pk uninstall && grep -q '// mine' opencode.jsonc

# 5. Default stays claude-only.
D3=$(mktemp -d) && cp "$REPO/pickle" "$D3/pk" && cd "$D3" && ./pk install --project demo
test ! -e opencode.jsonc && test ! -e .pi
echo ACCEPTANCE OK
```

Manual smoke (needs the one-time Copilot login, not CI-able): from an installed project,
`opencode run --agent docs-readability --file <some .md> "review"` returns suggestions; in a
pi session the `docs_readability` tool is discovered from `.pi/extensions/`.

### Docs update (mandatory when user-facing)

`README.md` per Task 9; the dev-tooling docs (`.pi/README.md`, `.opencode/README.md`) per
Task 1; the shipped protocol change is itself docs (Task 7). No marker-block change (agent
choices are not derivable from `pickle.toml`, so they don't belong between the markers).

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint` clean.
2. Docs updated (Tasks 1, 7, 9).
3. Write a summary of everything done (files touched, decisions honoured, anything deferred).
4. Suggested Conventional Commit message:

   ```
   feat(install): multi-agent --agent contract, opencode + pi wiring, shipped docs-readability reviewer (T-009)

   <body — what and why>
   ```

5. Commit locally on `feat/T-009-agent-enablement`; do **not** push or open an MR without
   user approval. Present the commit message; after approval finalize, push, open the MR;
   merging is the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P4)
- 2026-07-26 — parked (stays in TO DO, unscheduled). source: board triage — backlog growth analysis
- 2026-07-26 — unparked (user decision): demand signal — the docs-readability reviewer is in active use and wanted from pi/zed/opencode sessions in installed projects
- 2026-07-26 — refined: scope widened to the full `--agent claude,opencode,pi` contract; absorbed T-010 (pi scaffold + T-006 symmetry obligation) and T-016 (Step 4b + reviewer shipping); title updated (filename kept — id is the stable key); re-graded medium/medium/M → high/high/L
- 2026-07-26 — TO DO → READY: plan complete (READY gate met); scope: --agent contract + opencode/pi wiring + shipped docs-readability reviewer
