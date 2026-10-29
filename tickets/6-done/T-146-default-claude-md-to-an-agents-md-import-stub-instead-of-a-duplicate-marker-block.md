---
id: T-146
title: Default CLAUDE.md to an @AGENTS.md import stub instead of a duplicate marker block
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-146 — Default CLAUDE.md to an @AGENTS.md import stub instead of a duplicate marker block

## Outcome

`pickle install --agent claude` writes a `CLAUDE.md` whose pickle block is the single line
`@AGENTS.md`, not a second copy of the brine guidance. Existing installs convert on their next
`pickle upgrade`. Claude Code still loads the full guidance, through the import, from one source
of truth, and the file works on every platform without a symlink.

## Description

Today `--agent claude` gives `CLAUDE.md` one of three shapes (`Run` in
`internal/install/install.go`, the `switch` on `claudeMD`):

- **default**: its own copy of the marker block, duplicating `AGENTS.md`'s;
- **`--claude-symlink`**: a symlink to `AGENTS.md`. Nothing is duplicated, but checkouts without
  `core.symlinks` get a placeholder file, which `doctor` warns about;
- **`--claude-symlink` where symlinks are refused**: falls back to the marker-block copy (T-143).

Two of the three duplicate content. Claude Code resolves `@path` imports in `CLAUDE.md`, so
`CLAUDE.md`'s pickle block carries the import line instead of the rendered guidance:

```
# Brine

<!-- pickle:begin -->
@AGENTS.md
<!-- pickle:end -->
```

The import sits *inside* the existing marker pair, so `injectMarker`, `stripMarker`, `markerSpan`
and `uninstallMarkerFile` keep working unchanged. `uninstall` strips the block and keeps the file,
as it does for a marker-form `CLAUDE.md` today. No sentinel and no new file form exist. An existing
user-written `CLAUDE.md` without markers gets the import block appended, the same way it gets the
full block today.

What changes is the block *content* written for `CLAUDE.md`:

- `Run` writes the import block wherever it writes `CLAUDE.md`'s marker today (default, and the
  `--claude-symlink` fallback and refresh cases);
- `RefreshMarkers` (upgrade, `project add|remove`) writes the import block into a regular-file
  `CLAUDE.md`. That **converts** existing full-block `CLAUDE.md` files. Their interior is
  pickle-owned and already overwritten on every refresh, so no user content is lost;
- `doctor` measures `CLAUDE.md`'s block against the import block, so a leftover full block is
  ordinary drift ("run `pickle upgrade`").

One side effect, which is intended: through the import, Claude Code now also sees `AGENTS.md`'s
hand-written content outside the markers. Before, it saw only the block copy.

`--claude-symlink` stays as an opt-in. Its fallback becomes the import with no code change of its
own. The symlink and placeholder handling are untouched. This repo is unaffected: its `CLAUDE.md`
is a committed symlink to `AGENTS.md`, which every path above leaves alone. The `AGENTS.md` block
is unchanged, so no hand-mirrored marker edit is needed under the self-modify policy.

Soft couplings: T-143 (the symlink fallback whose content changes), T-041 (marker refresh and
drift, which this re-targets for `CLAUDE.md`), T-121 (help text that names Claude artifacts).

## Implementation Plan

### 0. Feature branch (mandatory)

```
git checkout main
git checkout -b feat/T-146-claude-md-import-stub
```

Work on this branch, with local WIP commits. This is a root-path child (`path = "."`): tidy the
commits into atomic ones before presenting them. No push and no MR without explicit approval;
merging is the human's. This repo uses the in-tree layout: before pushing,
`git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print
nothing.

### Prerequisite gate (hard)

none. No `depends-on:`, and T-143 and T-041 are both in `6-done/` and merged.

### Confirmed design decisions (do not deviate without asking)

1. **`CLAUDE.md`'s pickle block is exactly `@AGENTS.md`, inside the existing marker pair.** It is
   one exported constant, `ClaudeImportBlock`, next to `ClaudeMDTarget`. No new sentinel and no
   new file form, so every marker helper and `uninstall` keep working unchanged.
2. **Refresh converts.** `RefreshMarkers` writes `ClaudeImportBlock` into any regular-file
   `CLAUDE.md` that is not a placeholder, replacing a full block. The interior is pickle-owned
   (user approved, 2026-10-29).
3. **`--claude-symlink` stays as an opt-in, not deprecated.** Only its fallback content changes
   (user approved, 2026-10-29).
4. **`doctor` measures `CLAUDE.md` against `ClaudeImportBlock`, `AGENTS.md` against
   `MarkerBlock(cfg)`.** A full block in `CLAUDE.md` is a drift warning, never an error.
5. **Existing `CLAUDE.md` content outside the markers is never touched.** Symlinks and
   placeholders keep today's handling exactly.

### Tasks

#### Task 1: write the import block
In `internal/install/install.go`:
- add `ClaudeImportBlock = "@" + ClaudeMDTarget` (doc comment: the `CLAUDE.md` block body; Claude
  Code resolves the import);
- in `Run`, `writeMarker` injects `ClaudeImportBlock` instead of `MarkerBlock(cfg)`;
- in `RefreshMarkers`, the regular-file `CLAUDE.md` branch injects `ClaudeImportBlock`;
- update the doc comments that say `CLAUDE.md` gets "the marker block" / "a copy" (`Options.ClaudeLink`,
  `RefreshMarkers`, the T-143 fallback comment in `Run`).

#### Task 2: doctor
In `checkMarkers` (`internal/doctor/doctor.go`), the regular-file `CLAUDE.md` branch compares its
body to `install.ClaudeImportBlock`. Factor `checkMarkerDrift` to take the wanted body (or add a
sibling) so the warning wording stays shared. Make the "differs from what pickle.toml renders"
phrase depend on the file: for `CLAUDE.md` it reads "differs from the `@AGENTS.md` import". When cfg is nil, the `CLAUDE.md` comparison still
runs, because it does not depend on cfg. Update the function's doc comment.

#### Task 3: CLI help
`internal/cli/install.go` `--claude-symlink` usage: "make CLAUDE.md a symlink to AGENTS.md instead
of an @AGENTS.md import". `internal/cli/cli.go` install summary: "(and CLAUDE.md's @AGENTS.md
import when --agent includes claude)". `TestMarkerBlockStatesNoUnconditionalAgentArtifact`
(T-121, checks `MarkerBlock` lines) is unaffected, because `MarkerBlock` does not change.

#### Task 4: tests
- `internal/install/install_test.go`: a fresh `Run` with `Agents{Claude: true}` produces
  `CLAUDE.md` == `"# Brine\n\n<!-- pickle:begin -->\n@AGENTS.md\n<!-- pickle:end -->\n"`;
  `RefreshMarkers` over a `CLAUDE.md` holding the full `MarkerBlock(cfg)`, with a user line above
  the markers, leaves the user line and an `@AGENTS.md` body; the existing regular-file refresh case
  (the `RefreshMarkers (CLAUDE.md regular file)` check) asserts the body is `ClaudeImportBlock`,
  and the symlink case keeps asserting only that the link survives; `Uninstall` on the fresh install leaves `CLAUDE.md` == `"# Brine\n"`.
- `internal/install/symlink_fallback_test.go` `TestClaudeMDSymlinkFallsBackToMarker`: assert the
  fallback body is `ClaudeImportBlock`.
- `internal/doctor/`: an import-form `CLAUDE.md` passes with no warning; a full-block `CLAUDE.md`
  warns with the drift message; the placeholder test is unchanged.
- Update any existing assertion that expected `MarkerBlock(cfg)` in `CLAUDE.md`.

### Acceptance test

```
just build && just test && just lint && just docs-check      # all green
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && git init -q
./pickle-test install --path . --project demo --agent claude --in-tree
cat CLAUDE.md          # → "# Brine", blank, begin marker, "@AGENTS.md", end marker
./pickle-test doctor   # exit 0, no CLAUDE.md warning
# simulate a pre-T-146 install: replace CLAUDE.md's body with AGENTS.md's block
python3 -c "import re;a=open('AGENTS.md').read();c=open('CLAUDE.md').read();b=re.search(r'<!-- pickle:begin -->.*?<!-- pickle:end -->',a,re.S).group();open('CLAUDE.md','w').write(re.sub(r'<!-- pickle:begin -->.*?<!-- pickle:end -->',lambda m:b,c,flags=re.S))"
./pickle-test doctor   # warns: CLAUDE.md block differs … run `pickle upgrade`
./pickle-test upgrade && cat CLAUDE.md   # body back to "@AGENTS.md"
./pickle-test uninstall && cat CLAUDE.md # → "# Brine" only
```

### Docs update (mandatory when user-facing)

- `docs/user-manual/cli-reference.adoc`: the `claude` agent-artifacts row (`CLAUDE.md` carries
  an `@AGENTS.md` import block; fallback wording), the `upgrade` refresh paragraph (a regular
  `CLAUDE.md` is re-injected with the import and converts a full block), the `doctor` markers
  bullet (`CLAUDE.md`'s block must be the import), the `uninstall` paragraph if its wording
  no longer holds, and every sentence saying the `AGENTS.md`/`CLAUDE.md` block is rendered or
  regenerated from `pickle.toml` (around the `project add|remove` re-inject paragraph, the
  marker-region ownership paragraph, and the doctor bullet): only `AGENTS.md`'s is.
- `docs/user-manual/installation.adoc`: the `--claude-symlink` fallback sentence.
- `docs/user-manual/concepts/project-structure.adoc`: the tree annotation
  (`CLAUDE.md ← --agent claude: @AGENTS.md import, or symlink to AGENTS.md`) the
  pickle-owned paragraph saying both files are replaced, and the ownership row
  (`AGENTS.md` (and `CLAUDE.md`)), which must say `CLAUDE.md` holds the import.
- `CHANGELOG.md` under `## [Unreleased]` → `### Changed`: **`CLAUDE.md` imports `AGENTS.md`
  instead of copying it** (T-146): install writes an `@AGENTS.md` block, upgrade converts an
  existing full block, doctor treats a leftover copy as drift.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated as above.
3. Summary of files touched and decisions, plus anything deferred.
4. Suggested commit (one atomic commit, or code + docs split):

   ```
   feat(install): make CLAUDE.md import AGENTS.md instead of copying its block (T-146)

   CLAUDE.md's pickle block is now the single line @AGENTS.md. install writes it,
   upgrade and project add|remove convert an existing full block, and doctor
   measures CLAUDE.md against the import. --claude-symlink stays as an opt-in.
   ```

5. Keep the note prefixes `CLAUDE.md: symlink unavailable` and
   `CLAUDE.md is already a regular file with a marker block`, because the fallback tests match them.
6. Tidy the WIP commits into atomic ones (root-path child).
7. Commit locally; do not push or open an MR. Move to `4-in-review/`.

## Review

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | spec-unclear | fixed inline | The acceptance test's install line omits `--in-tree`, so run verbatim it is refused and every later step fails | `pickle install: --path "." selects the in-tree layout; pass --in-tree explicitly` (`internal/cli/install.go:47`, already on main) | Added `--in-tree` to the plan's acceptance test |
| F2 | non-blocking | stale-xref | fixed inline | `MarkerBlock`'s doc comment says it builds the `AGENTS.md`/`CLAUDE.md` content | `internal/install/install.go` `MarkerBlock` comment | Now says `AGENTS.md`, with `CLAUDE.md` getting `ClaudeImportBlock` |
| F3 | non-blocking | stale-xref | fixed inline | `refreshMarkers`' comment says it re-injects the `AGENTS.md`/`CLAUDE.md` block from cfg | `internal/cli/project.go` `refreshMarkers` comment | Reworded: `AGENTS.md`'s from cfg, plus `CLAUDE.md`'s import |
| F4 | non-blocking | docs-gap | fixed inline | The docs don't say that through the import, Claude Code now also reads `AGENTS.md`'s text outside the markers | CHANGELOG entry and the cli-reference `claude` row described "one file" only | One sentence added to each |
| F5 | non-blocking | test-gap | fixed inline | `TestCheckClaudeMDImport`'s pass case only checked for no warnings, so it could not fail if the drift check were skipped | `internal/doctor/claude_view_test.go` | Asserts the `CLAUDE.md marker block current` pass line |
| F6 | non-blocking | correctness | noted | A user `CLAUDE.md` that already has `@AGENTS.md` outside the markers gets a second import appended; doctor reports it current | Repro: `printf '# Notes\n\n@AGENTS.md\n' > CLAUDE.md && pickle-test install --path . --project demo --agent claude --in-tree` → `grep -c '^@AGENTS.md' CLAUDE.md` = 2 | Redundant rather than wrong; revisit if users report it |

Disposition summary: 5 fixed inline (F1–F5), 1 noted (F6), 0 folded, 0 new tickets.
cost: estimated S, actual S

- [x] Reviewer independence settled (step 0): delegated. The implementer wrote the branch in this session, so steps 1–4a went to a fresh general-purpose sub-agent spawned with the step 0 prompt (`hunt: angle 7 only`). No delegated findings discarded.
- [x] In-tree stale-branch check (step 0a): `pickle doctor` warned that the branch had the ticket in `3-in-development`; rebased onto main; doctor then clean.
- [x] Implementation audit (steps 1, 2): build/test/lint/docs-check green (actionlint and shellcheck not installed locally; CI runs them). Acceptance test re-run with `--in-tree` (F1): every step as specified. Tasks 1–4 and decisions 1–5 verified.
- [x] Correctness hunt (step 3): host tool path. `/code-review medium` run by the top-level session on `main...feat/T-146-claude-md-import-stub`, no sub-agents. It reported 8 findings: 2 overlap F2/F3, 3 became F4/F5/F6, and 3 were discarded on re-verification (block interior overwritten on refresh: pickle-owned by T-146 decision 2; doctor package doc: still accurate, both files carry a marker block; `strings.Trim` in `checkMarkerDrift`: needed for the `MarkerBlock` path, cosmetic). Angle 7 ran separately in the delegated reviewer: no findings (constant content; an out-of-root `CLAUDE.md` symlink probe left the target unmodified). F6 reproduced.
- [x] Consistency audit (step 4): repo-wide grep for `CLAUDE.md` with copy/marker/block/render/regenerat; F2 and F3 only.
- [x] Documentation audit (step 4a): cli-reference, installation, project-structure and CHANGELOG cover the change (plus F4); whole-tree sweep clean; `just docs-check` clean.
- [x] Docs-readability pass (step 4b): conscious skip. Short, factual edits to existing passages.
- [x] Findings recorded (step 5): severity, class and disposition for each; summary and cost lines present.
- [x] Ticket moved to `tickets/6-done/`; `## History` appended (step 6).
- [x] Other references (step 7): no other ticket or governing document describes `CLAUDE.md`'s block content; this repo's `CLAUDE.md` is a symlink, and `AGENTS.md` is unchanged.
- [x] Remaining-tickets impact sweep (step 8): open tickets T-075, T-078 and T-079 (rick interop) don't touch `CLAUDE.md` or markers; no impact.
- [x] Summary + commit message & MR attributes presented and approved (2026-10-29); in-tree remote-base check clean after pushing main; bookkeeping committed; next-ticket suggestion given: READY is empty (step 9).

## History

- 2026-10-29 — created (TO DO). source: chat: user asked to replace the duplicated CLAUDE.md marker block with a `See @AGENTS.md` import; exploration found the default and symlink-fallback install paths duplicate AGENTS.md
- 2026-10-29 — refined: design narrowed to an `@AGENTS.md` body inside the existing marker pair (no sentinel, uninstall unchanged); user approved converting on refresh and keeping `--claude-symlink` as an opt-in; re-graded complexity low, cost S.
- 2026-10-29 — TO DO → READY: plan complete
- 2026-10-29 — applicability gate (fresh sub-agent): no blocking findings, 5 non-blocking. Routing approved by user: 1, 3, 4 inline; 2 inline then closed; 5 note-and-close.
- 2026-10-29 — plan amended inline: doctor drift wording depends on the file (Task 2); T-121 test correctly named (Task 3); body assertion only in the regular-file refresh case (Task 4); docs step adds the "rendered from pickle.toml" passages in cli-reference and project-structure; Finish keeps the two note prefixes the fallback tests match.
- 2026-10-29 — READY → IN DEVELOPMENT: picked up
- 2026-10-29 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-10-29 — plan amended inline: acceptance test install line gains `--in-tree` (review F1; `--path .` without it is refused).
- 2026-10-29 — IN REVIEW → DONE: review: no blocking findings; 5 fixed inline (F1–F5), 1 noted (F6), 0 new tickets
- 2026-10-29 — published: commit approved, pushed, PR #105 opened (https://github.com/codcod/pickle/pull/105); awaiting human merge.
- 2026-10-29 — merged to main (PR #105, 494e790, https://github.com/codcod/pickle/commit/494e790)
