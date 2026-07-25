---
id: T-018
title: upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body)
project: pickle
depends-on: []
impact: high
complexity: medium
cost: M
---

# T-018 — upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body)

## Description

Non-blocking follow-up from the T-006 review. `pickle upgrade` shipped as specified (decision D5
deliberately accepted `pickle.toml` normalisation), but in practice it destroys two kinds of
hand-written content without warning. Both are pre-existing behaviours of the install/marker
machinery; T-006 makes them far more reachable by shipping a command whose *entire job* is
refreshing them.

1. **`pickle.toml` comments are silently dropped.** On a version change `Upgrade` calls
   `cfg.Save("")` (`internal/install/install.go:154-158`), which re-renders the file through
   `config.Render` — a canonical writer that emits no comments. This repo's own `pickle.toml`
   is 15 lines of hand-written rationale (`pickle.toml:1-13`) explaining the bootstrap
   chicken-and-egg; a single `pickle upgrade` here erases all of it. Decide and implement:
   preserve comments (round-trip-preserving edit of just the `payload_version` line), or warn
   before rewriting, or at minimum have `Render` re-emit a managed header. **The docs half of
   this is being fixed as a blocking T-006 rework finding** (`README.md:94-95` wrongly enumerates
   only `project add|remove` as normalising) — this ticket is about the *behaviour*.

   **Added 2026-07-24 (T-006 scoped re-review): the same error of fact ships inside the generated
   file itself.** `config.Render` hardcodes the header
   `# Managed by pickle. Hand-edits are preserved on load but normalised to this layout on the
   next pickle project add|remove.` (`internal/config/config.go:196-198`, from T-001 `3dc0c26`).
   That is the *identical* wrong enumeration T-006's rework just corrected in the README — but in
   a surface that ships into **every** installed project, is the first thing a user reads in their
   own `pickle.toml`, and appears in the very file whose comments were just erased, while saying
   nothing about comment loss. Proven by a mutation run of this repo's own tree: one `upgrade`
   replaced 16 lines of hand-written rationale with that header. Kept non-blocking for T-006
   because the README (the surface a user consults) is now correct, the string predates T-006 and
   is in a package T-006 never touched, and whichever remedy this ticket picks *changes this header
   anyway* — fixing it in T-006 would mean writing it twice. Whatever is chosen, the header must
   end up enumerating every writer (`install`, `upgrade`, `project add|remove`) and stating the
   comment contract truthfully.

2. **Hand-maintained content inside the `AGENTS.md` marker block is silently replaced.**
   `install.go:22-23` declares everything between `<!-- pickle:begin -->`/`<!-- pickle:end -->`
   pickle-managed and replaced on re-run, but this repo's `AGENTS.md` keeps genuinely
   repo-specific bullets *inside* that region that `markerBlock` (`install.go:515-544`) does not
   emit: the child's **Commands** (`just build`/`just test`/`just lint`), the **WIP limits**, the
   `skill/` self-host symlink note, and the publish-gate wording. Verified during review:
   grepping `markerBlock` for `just build`/`wip_in_development` returns **0** matches, so
   `pickle upgrade` in this repo would delete them, and `doctor.checkMarkers`
   (`doctor.go:111-137`) only checks marker *presence*, so the loss is invisible. Fix by either
   teaching `markerBlock` to render the child's commands + WIP limits from `pickle.toml` (making
   the block genuinely regenerable), or moving the repo-specific bullets outside the markers, or
   both. Mitigation already in place for the dev workspace only:
   `opencode.jsonc:63` gates `pickle upgrade*` behind `ask`, and `.pi/extensions/workspace-guardrails.ts:99`
   has a self-modify guard — neither protects an end user's project.

3. **While in `markerBlock`: one stale conditional the refresh now propagates.**
   `install.go:544` still reads "Prefer `pickle ticket move` **once available**; otherwise do the
   three edits by hand" — `ticket move` shipped in T-007, and the skill payload states it
   unconditionally (`skill/SKILL.md:93`). Because `upgrade` rewrites this block in every
   installed project, the stale sentence is now actively propagated. Re-word it.

Soft coupling: T-013 (install polish) touches the same `injectMarker`/summary surface; keep the
changes disjoint or sequence them. **T-020** (doctor marker-drift detection) was split out of
this ticket during refinement and should be refined only after this one lands.

**Re-graded 2026-07-25: impact `medium` → `high`.** Every other open ticket is polish,
consolidation, or new surface area; this is the only one where a **shipped** command destroys
user-authored content by default, in every installed project, with no warning before and no
report after. `pickle upgrade` is also the command a user is most likely to run repeatedly and
trust blindly. Complexity `medium` and cost `M` stand: three bounded edits (a line-level config
writer, a `markerBlock` that reads config, a docs pass) with the design questions now closed.

### Refinement note — 2026-07-25 (re-verified against `main` @ `fedfcc8`, post-T-006 merge)

All refs above still resolve. Two corrections and one addition:

- `markerBlock` is at `internal/install/install.go:506+` (not `:515-544`); the stale board-rule
  sentence of item 3 is its **last line**, now reading "Prefer `pickle ticket move` once
  available; otherwise do the three edits by hand".
- **Item 1's error of fact has a *third* instance**, in the same file: the `config` package doc
  (`internal/config/config.go:4-8`) also says the file is "normalised to the canonical layout on
  the next mutation (project add/remove)". `rg 'project add/remove|project add\|remove'` returns
  exactly three hits — `README.md:96` (correct, fixed by T-006), `config.go:8` and `config.go:198`
  (both wrong). All three must agree after this ticket, which *changes the truth again*: once
  `upgrade` stops re-rendering, the honest enumeration is `install` and `project add|remove`.
- **Exact loss measured**, by running the merged binary against a throwaway clone of this repo
  (`pickle upgrade`, then `git diff`): `AGENTS.md` −37/+32 lines. Destroyed: the whole
  **Commands** bullet (`just build`/`just test`/`just lint`), the whole **WIP limits** bullet,
  the self-host `skill/`-symlink note, the concrete `feat/T-NNN-<slug>` prefix (→ literal
  `<branch_prefix>`), the "This repo has a single child … at the repo root" prose, and the
  publish-gate's "after approval, finalize (squash or keep history) + push + open the MR" clause.
  `pickle.toml`: 16 comment lines → the 3-line generated header. Commands, WIP limits and branch
  prefix are all **in `pickle.toml`** and therefore renderable; only the self-host note is
  genuinely un-derivable.

## Implementation Plan

### 0. Feature branch (mandatory)

`pickle`'s sole child is the repo root (`.`), so the branch is cut in this repo:

```
git checkout main
git checkout -b feat/T-018-upgrade-preserve-user-content
```

Local WIP commits encouraged; **no push / MR without explicit user approval** (child is
publish-gated — see `pickle.toml` / `AGENTS.md`).

### Prerequisite gate (hard)

- **T-006 must be merged** — it is: `4bcfc00` on `main` (board `merged` column ticked). This
  ticket edits `Upgrade`, which T-006 introduced.
- No `depends-on:`. Soft couplings, all currently in `1-to-do/` and none required first:
  **T-012** (config hardening — owns `Render`'s TOML escaping; this ticket adds a *second* writer
  path to the same file, so whoever lands second re-reads the other's changes), **T-013** (install
  polish — same `injectMarker`/`Result` surface), **T-017** (unify marker-pair detection — if it
  lands first, reuse its `markerSpan` helper instead of scanning by hand), **T-020** (doctor drift
  detection — split out of this ticket; refine it *after* this lands).
- Clean working tree on the new branch.

### Confirmed design decisions (user sign-off 2026-07-25 — do not deviate without asking)

1. **`pickle.toml`: surgical `payload_version` edit, not a re-render (D1).** `upgrade` must
   rewrite **only** the `payload_version = "…"` line, leaving every other byte — comments,
   blank lines, key order, alignment — untouched. `project add|remove` keep the canonical
   `Render` path, because they change structure. **Rejected alternatives:** warn-before-rewrite
   (content still dies by default), a full comment-preserving TOML round-trip (new dependency /
   hand-rolled parser, overlaps T-012), docs-only (leaves the destruction in place).
2. **No new dependency.** BurntSushi/toml decodes only; the surgical edit is line-based text
   manipulation in `internal/config`. Do **not** swap the TOML library.
3. **Fallback is explicit, never silent (D1a).** If the file has no top-level `payload_version`
   line to replace (hand-mangled, or key absent), fall back to the canonical `Save` **and record
   that the file was normalised** in the `Result`, so the user is told comments were lost. Never
   fall back silently.
4. **`markerBlock`: render everything derivable from `pickle.toml` (D2).** Commands
   (`build`/`test`/`lint`), WIP limits (`wip_in_development`/`wip_in_review`) and `branch_prefix`
   are all in the config and must be emitted, so the block is genuinely regenerable. Multi-child
   projects must render correctly (per-child lines) — this repo's single child must not become a
   hidden assumption.
5. **Relocate what is not derivable (D2a).** The self-host `skill/`-symlink note and the
   "this repo's sole child is the repo root, tickets describe the `pickle` binary itself" prose
   move **outside** the marker region in this repo's `AGENTS.md`. Nothing repo-specific stays
   inside a block that upgrade regenerates.
6. **Commit policy is rendered from `[commit]`, not hardcoded.** `markerBlock` currently asserts
   "Child-projects are **publish-gated**" unconditionally; it must read
   `cfg.Commit.ChildPublishGated`/`OverarchingAuto` so it cannot state a policy the project does
   not have. Restore the lost "after approval, finalize (squash or keep history) + push + open
   the MR" clause while there — it is generic flow prose, not repo-specific.
7. **Fix the stale sentence (item 3).** `markerBlock`'s last line must state `pickle ticket move`
   unconditionally (it shipped in T-007), matching `skill/SKILL.md:93`.
8. **All three normalisation statements must end up telling the same, new truth.**
   `config.go:4-8` (package doc), `config.go:196-198` (generated header) and `README.md:94-97`.
   After this ticket the truth is: **`install` and `project add|remove` re-render canonically and
   drop comments; `upgrade` preserves them.** The generated header must also stop implying
   `upgrade` rewrites the layout.
9. **Drift *detection* is out of scope** — split to **T-020** by user decision. This ticket only
   stops the destruction.

### Tasks

#### Task 1 — surgical version writer (`internal/config/config.go`)
Add, next to `Save`:

```go
// SetPayloadVersionInPlace rewrites only the payload_version line in the file at
// path, preserving every other byte (comments, spacing, key order). Reports
// ErrNoPayloadVersionLine if no top-level payload_version key is present.
func SetPayloadVersionInPlace(path, version string) error
```

- Scan lines; match the **first** top-level `payload_version` assignment: a line whose trimmed
  form starts `payload_version` followed by optional spaces and `=`. **Skip commented lines**
  (`#` first non-space) and **stop at the first table header** (a trimmed line starting `[`) so a
  `payload_version` inside `[[project]]` can never be hit.
- Replace with `payload_version = "<version>"`, preserving the original leading whitespace and
  any trailing inline comment on that line.
- Escape `version` the same way `Render` does (see T-012 item 2 — reuse whatever escaping
  helper exists; if none, quote via `strconv.Quote`).
- Write back **atomically** (temp file + `os.Rename`) and **preserve the original file mode**
  (`os.Stat` first) — mirror `Save`'s existing write strategy.
- Return `ErrNoPayloadVersionLine` (an exported sentinel, `errors.New`) when no match is found.

#### Task 2 — use it from `Upgrade` (`internal/install/install.go:150-160`)
Replace the `cfg.PayloadVersion = payloadVersion; cfg.Save("")` pair with:

```go
err := config.SetPayloadVersionInPlace(cfg.Path(), payloadVersion)
if errors.Is(err, config.ErrNoPayloadVersionLine) {
    cfg.PayloadVersion = payloadVersion
    if err := cfg.Save(""); err != nil { return res, err }
    res.created(config.FileName + " (payload_version -> " + payloadVersion + "; file normalised, comments not preserved)")
} else if err != nil {
    return res, err
} else {
    res.created(config.FileName + " (payload_version -> " + payloadVersion + ")")
}
```

Keep the existing early return for the already-at-version case **unchanged** (it must still not
touch the file at all).

#### Task 3 — `markerBlock` renders the config (`internal/install/install.go:506+`)
Extend the `fmt.Sprintf` block, keeping it a pure function of `*config.Config`:

- **Commands bullet** — for a single child: ``- **Commands** (the child's, from `pickle.toml`): `build`, `test`, `lint`.`` using the actual values; for multiple children, one line per child prefixed with its name. Omit a command that is empty; omit the bullet entirely if no child defines any.
- **WIP limits bullet** — ``- **WIP limits (per child):** `3-in-development/` ≤ N · `4-in-review/` ≤ M.`` from the child's values; per-child lines when children differ.
- **Branch & commit bullet** — render the real prefix (`feat/T-NNN-<slug>`), not the literal
  `<branch_prefix>`; per-child when prefixes differ.
- **Commit-policy bullet** — branch on `cfg.Commit.ChildPublishGated` and
  `cfg.Commit.OverarchingAuto` (decision 6), including the restored finalize/push/MR clause.
- **Board rule** — drop "once available; otherwise" (decision 7).

#### Task 4 — relocate this repo's non-derivable prose (`AGENTS.md`)
Move outside the `<!-- pickle:begin -->`/`<!-- pickle:end -->` region (into the existing intro
prose above it): the self-host `skill/`-symlink note and the "single child at the repo root /
tickets describe the `pickle` binary itself" sentences. After Task 3 the remaining in-block
content must be **exactly** what `markerBlock` renders.

#### Task 5 — make all three normalisation statements true (decision 8)
- `internal/config/config.go:4-8` — package doc.
- `internal/config/config.go:196-198` — the generated header written into **every** installed
  project. Must enumerate every re-rendering writer (`install`, `project add|remove`), state that
  those re-renders drop comments, and state that `upgrade` preserves them.
- `README.md:94-97` — the Configuration paragraph T-006 just corrected; correct it again for the
  new behaviour.

#### Task 6 — tests
`internal/config/config_test.go`:
- Round-trip: a file with comments, blank lines, inline comments and a `[[project]]` block →
  `SetPayloadVersionInPlace` → assert the output is **byte-identical except the one line**, and
  that `Load` still parses it with the new version.
- `payload_version` appearing **commented out** and **inside `[[project]]`** → both ignored;
  a genuine top-level key later in the file is still found.
- No top-level key → `ErrNoPayloadVersionLine`.
- Version strings needing escaping (`"`, `\`) survive a `Load` round-trip.
- File mode preserved.

`internal/install/install_test.go`:
- `Upgrade` on a config **with comments** → comments still present afterwards, `payload_version`
  updated (this is the regression test for the whole ticket).
- `Upgrade` on a config with no `payload_version` line → falls back, and the `Result` records
  the normalisation.
- `markerBlock` renders commands/WIP/branch prefix for a single child **and** for two children
  with differing values; a child with no commands omits the bullet; `child_publish_gated = false`
  renders the non-gated wording.

#### Task 7 — self-host no-op check
After Tasks 3–4, `pickle upgrade` run in a clone of this repo must leave `AGENTS.md`
**byte-identical**. This is the acceptance test's centrepiece, so verify it during development,
not only at the end.

### Acceptance test

Run from the repo root. The centrepiece is that `upgrade` becomes a genuine no-op on this
repo's own tree apart from the single version line:

```
just build
CLONE=$(mktemp -d)/clone
git clone -q --local . "$CLONE"
cp pickle "$CLONE"/
( cd "$CLONE" && ./pickle upgrade >/dev/null )

# 1. AGENTS.md must be untouched (nothing hand-written left inside the markers):
( cd "$CLONE" && git diff --quiet AGENTS.md ) && echo "AGENTS.md: byte-identical OK"

# 2. pickle.toml: exactly one changed line, and it is payload_version; all comments survive:
( cd "$CLONE" && git diff --numstat pickle.toml )            # expect: 1  1  pickle.toml
( cd "$CLONE" && git diff -U0 pickle.toml | grep '^[+-]payload_version' )
( cd "$CLONE" && grep -c '^#' pickle.toml )                  # expect: same as the pre-upgrade count

# 3. Idempotent: a second upgrade changes nothing further:
( cd "$CLONE" && ./pickle upgrade >/dev/null && git diff --numstat )   # expect: still only pickle.toml 1 1

# 4. The install path still renders canonically (fallback intact):
TMP=$(mktemp -d)
( cd "$TMP" && git init -q && "$OLDPWD/pickle" install --project demo >/dev/null && ./pickle upgrade >/dev/null 2>&1 || true )
rm -rf "$TMP" "$CLONE"

just test    # incl. the new config + install tests
just lint
./pickle doctor
```

Expected: `AGENTS.md` byte-identical after upgrade; `pickle.toml` differs by exactly the
`payload_version` line with its comment count unchanged; the second upgrade adds no further
diff; `just test`/`just lint`/`doctor` clean.

### Docs update (mandatory when user-facing)

- `README.md:94-97` — Configuration paragraph: `install` and `project add|remove` re-render
  canonically and drop comments; **`upgrade` preserves them** (it edits only the version line).
- `internal/config/config.go:4-8` + `:196-198` — package doc and the generated header, per
  decision 8. The header ships into every installed project, so its wording is user-facing docs.
- `README.md` `## pickle upgrade` section (`:185-207`) — it currently warns that stamping a new
  version "re-renders `pickle.toml` and **drops its comments**" and that marker-block hand-edits
  are replaced. Both statements change: comments now survive, and the marker block no longer
  swallows commands/WIP/branch-prefix. Re-word rather than delete the caution (content outside
  the markers is still the safe place for hand-written notes).

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. Docs updated (all three normalisation statements agree; README upgrade section re-worded).
3. Write a summary (files touched, decisions honoured, anything deferred — expect T-020 to be
   the named follow-up).
4. Suggested Conventional Commit:

   ```
   fix(install): stop upgrade discarding user content (T-018)

   `pickle upgrade` rewrote pickle.toml through the canonical renderer,
   erasing every comment, and regenerated the AGENTS.md marker block from a
   template that omitted the child's commands, WIP limits and branch prefix —
   so a refresh silently deleted hand-written content.

   Rewrite only the payload_version line in place (comments, spacing and key
   order preserved; explicit fallback to the canonical render when the key is
   absent), and render commands, WIP limits, branch prefix and the commit
   policy into the marker block from pickle.toml so it is genuinely
   regenerable. Correct the normalisation contract in the package doc, the
   generated pickle.toml header and the README.
   ```

5. Commit locally on `feat/T-018-upgrade-preserve-user-content`; **do not push or open an MR
   without user approval**. Present the commit message, then move the ticket to IN REVIEW.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-25 — TO DO → READY: plan complete; D1-D9 confirmed with user
