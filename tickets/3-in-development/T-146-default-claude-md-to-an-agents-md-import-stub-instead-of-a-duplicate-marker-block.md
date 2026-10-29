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
./pickle-test install --path . --project demo --agent claude
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

<!-- empty until IN REVIEW -->

## History

- 2026-10-29 — created (TO DO). source: chat: user asked to replace the duplicated CLAUDE.md marker block with a `See @AGENTS.md` import; exploration found the default and symlink-fallback install paths duplicate AGENTS.md
- 2026-10-29 — refined: design narrowed to an `@AGENTS.md` body inside the existing marker pair (no sentinel, uninstall unchanged); user approved converting on refresh and keeping `--claude-symlink` as an opt-in; re-graded complexity low, cost S.
- 2026-10-29 — TO DO → READY: plan complete
- 2026-10-29 — applicability gate (fresh sub-agent): no blocking findings, 5 non-blocking. Routing approved by user: 1, 3, 4 inline; 2 inline then closed; 5 note-and-close.
- 2026-10-29 — plan amended inline: doctor drift wording depends on the file (Task 2); T-121 test correctly named (Task 3); body assertion only in the regular-file refresh case (Task 4); docs step adds the "rendered from pickle.toml" passages in cli-reference and project-structure; Finish keeps the two note prefixes the fallback tests match.
- 2026-10-29 — READY → IN DEVELOPMENT: picked up
