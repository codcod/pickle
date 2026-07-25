---
id: T-018
title: upgrade must not silently discard user content (pickle.toml comments, AGENTS.md marker body)
project: pickle
depends-on: []
impact: medium
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
changes disjoint or sequence them.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
