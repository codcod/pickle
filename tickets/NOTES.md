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
