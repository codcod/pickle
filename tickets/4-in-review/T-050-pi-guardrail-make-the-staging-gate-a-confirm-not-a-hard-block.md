---
id: T-050
title: pi guardrail: make the staging gate a confirm, not a hard block
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-050 — pi guardrail: make the staging gate a confirm, not a hard block

## Outcome

After this ships, the pi guardrail's staging-discipline rule prompts for confirmation instead of hard-blocking, so a command that merely mentions `git add -A` in its own text (e.g. inside a documentation heredoc) is no longer refused outright.

## Description

The shipped pi guardrail (`agents/pi/extensions/pickle-guardrails.ts`, embedded payload per
`assets.go:19`, installed to `.pi/extensions/`) enforces three rules with **two different
verdicts, for no stated reason**:

| rule | verdict today |
|---|---|
| 1 — staging discipline (`git add -A` / `git add .` / `git commit -a`) | **hard `block`**, no prompt |
| 2 — publish gate (`git push`, `glab mr create`, `gh pr create`) | `ctx.ui.confirm`, blocks only without a UI |
| 3 — self-modify (repo-local `.pi/extensions/workspace-guardrails.ts` only) | `ctx.ui.confirm`, blocks only without a UI |

Rule 1 is the only unconditional refusal, and it is the one that misfires. Hit in the field
(2026-07-27): a `python3 - <<'EOF' … EOF` heredoc **documenting the guardrails** — containing the
sentence "blocks `git add -A`, `git add .`, `git commit -a`" — was refused with *"Blocked:
`git commit -a/--all` bypasses explicit pathspecs."* Nothing was being staged. The same block
fires on `echo 'never use git commit -a here'`. It makes *documenting the rule* harder than
breaking it, and the workaround it teaches (splitting the literal, `"git " + "commit"`) is a
trick a guard should not train into an agent.

### Why the fix is the verdict, not the matcher

The obvious reading — "the matcher is too loose, make it quote/heredoc-aware" — is wrong twice.

**First, the stated mechanism is wrong.** The guard does *not* match against the whole command
string: `segments()` (`:32-38`) splits on `&&`, `||`, `;` and `\n`, so the heredoc's body line
becomes its own segment and matches *there*. The defect is that the segmenter is a text splitter
with no notion of shell quoting or heredocs, and so inspects prose as if it were argv.

**Second, hardening it removes protection.** Measured by porting `segments()`/`gitStageArgs()`
verbatim into a harness (2026-07-27):

```
bash <<'EOF' \n git add -A \n EOF        -> BLOCK   (caught ONLY because heredoc bodies are inspected)
if true; then git add -A; fi             -> BLOCK   (caught ONLY because the match is unanchored)
```

So "skip heredoc payloads" turns `bash <<'EOF'` into a one-line bypass of rules 1 *and* 2, and
"require the match at a segment start" loses the `then`-branch (and every `env` / `sudo` / `time`
prefix). Both proposals trade a cosmetic false positive for a real hole.

**And the guard cannot be sound anyway**, so there is nothing to protect by refusing hard. Same
harness, all currently allowed:

```
g=git; $g add -A            -> allow
echo -A | xargs git add     -> allow
git commit "-""a" -m x      -> allow
```

A gate a cooperative agent evades by accident is a **reminder, not a sandbox**. For a reminder,
the cost of a false positive should be one keypress — which is precisely what rules 2 and 3
already charge.

### Scope

1. **Make rule 1 symmetric with rules 2 and 3** in `agents/pi/extensions/pickle-guardrails.ts`:
   present the reason via `ctx.ui.confirm`, hard-block only when `!ctx.hasUI` (unattended runs keep
   today's strictness). Mirror the change in this repo's own
   `.pi/extensions/workspace-guardrails.ts`, whose rule 1 is the same code.
2. **State the guard's nature in its header comment** — a reminder encoding the AGENTS.md commit
   policy, not a security boundary; trivially bypassable by design; do not file matcher-hardening
   tickets against it. This is the durable half: without it, every future false positive re-opens
   the same wrong investigation.
3. **Verify — do not assume — whether `opencode.jsonc` shares the defect.** Its own comment
   (`opencode.jsonc:35-38`) says patterns match "against the parsed command" and its globs are
   prefix-anchored (`"git commit -a*"`), which suggests quoted prose in a heredoc does **not**
   match. If the declarative mirror is immune, that is a finding about which shape is better, and
   it should be recorded rather than "fixed".

### Explicitly out of scope

- **Making the matcher quote- or heredoc-aware.** Rejected above, with evidence. If a future need
  arises it is a shell-parser dependency, not a regex tweak.
- **The `cd <other-workspace> && pickle upgrade` prompt.** Reported alongside the finding above as
  "same class"; it is not. Rule 3 already uses `confirm`, so being *asked* to approve a
  self-modifying command is the designed behaviour, and the human says yes. `targetsTmp(seg,
  ctx.cwd)` is blind across the `&&` split (the `cd` lands in a different segment), so it cannot
  recognise the throwaway target and always asks — a precision loss in *when you are prompted*, not
  a false refusal. Recorded in `tickets/NOTES.md`; promote only if the prompting becomes a nuisance.
- Rule 2's and rule 3's own logic — unchanged.

### Couplings

Soft couplings (no `depends-on:`, no ordering enforced):

- **T-009** (DONE) authored this extension and absorbed the dropped T-010 scaffold ticket — read
  both for the guard's original intent before changing its contract.
- **T-046** — the other "self-host friction" ticket, but on the Go side (`doctor`/`upgrade`
  payload-version noise). No file overlap; listed so the two are not mistaken for duplicates.
- Changing shipped payload means an installed project only sees it after `pickle upgrade`; this
  repo's own copy is hand-maintained (AGENTS.md self-modify policy), so **both files must be edited
  in the same commit** or they drift.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-050-pi-guardrail-confirm
```

Root-path child (`project: pickle`). Tidy WIP commits into atomic ones before presenting (Finish,
below).

### Prerequisite gate (hard)

None. No `depends-on:`.

### Confirmed design decisions (do not deviate without asking)

1. **Rule 1's new shape mirrors rule 2's existing idiom exactly**, in both TS files: `if
   (!ctx.hasUI) return { block: true, reason }; const ok = await ctx.ui.confirm(<title>, reason +
   "\n\nProceed?"); if (!ok) return { block: true, reason: "User declined ..." };` — no new
   helper, no new pattern; rule 1 becomes structurally identical to rule 2's already-shipped
   confirm/decline handling.
2. **Both staging sub-checks (the `git add`/`stage` block and the separate `git commit -a` block)
   get the same treatment**, as two independent confirms — they are already two independent
   `return` sites today and stay that way; this ticket changes their verdict, not their control
   flow shape.
3. **`opencode.jsonc`'s rule 1 (`"git add -A*": "deny"` etc.) is left as `deny`, not changed to
   `"ask"`, regardless of what Task 3's investigation finds.** The Description's scope is
   explicit: verify whether the *matcher* shares the false-positive defect and *record* that
   finding — a verdict change there is a separate decision this ticket does not make. If Task 3
   finds `opencode.jsonc` **is** vulnerable to the same class of false positive, that goes into
   `tickets/NOTES.md` as a candidate follow-up, not an inline fix here (asking the user first, per
   the rules' promotion test, is out of this ticket's scope by design).
4. **The header-comment addition (Description item 2) goes into both TS files**, immediately
   above the `tool_call` handler (`agents/pi/extensions/pickle-guardrails.ts:59-60` and
   `.pi/extensions/workspace-guardrails.ts` at the equivalent line), stating: this gate is a
   reminder encoding `AGENTS.md`'s commit policy, not a security boundary; it is trivially
   bypassable by a cooperative agent (see the harness evidence recorded in this ticket's
   Description); do not file matcher-hardening tickets against it.

### Tasks

#### Task 1 — flip rule 1's verdict in the shipped payload
In `agents/pi/extensions/pickle-guardrails.ts`:
- The `git add`/`stage` block (`:69-76`): replace the unconditional `return { block: true, reason
  }` with the confirm idiom (decision 1), title e.g. `"Staging discipline"`.
- The `git commit -a` block (`:78-85`): same treatment, title e.g. `"Staging discipline"`.
- Add the header-comment paragraph (decision 4) above the `tool_call` handler.

#### Task 2 — mirror the same change in this repo's own self-host copy
Apply the identical edits to `.pi/extensions/workspace-guardrails.ts` (`git add`/`stage` block at
`:64-71`, `git commit -a` block at `:73-79`, header comment above the handler) — **in the same
commit** as Task 1 (Description's coupling note: the two files are independent copies and must
not drift). Leave Rule 3 (self-modify) in this file untouched — it already uses the confirm idiom
and is out of scope.

#### Task 3 — verify (do not assume) whether `opencode.jsonc`'s rule 1 shares the matcher defect
`agents/opencode/opencode.jsonc` (mirrored at `opencode.jsonc`, repo root) matches `"git add
-A*"`/`"git commit -a*"` as **prefix**-anchored globs against, per its own comment, "the parsed
command" (`:35-38`) — not the naively `&&`/`;`/newline-split segments the TS matcher inspects.
Reproduce this ticket's own field PoC as closely as opencode's tooling allows:
1. Read whatever opencode ships on how `permission.bash` patterns are matched (its published
   docs/schema, or `opencode --help`/`opencode debug` surfaces, if any expose this) to confirm
   prefix-anchoring is against the *whole top-level command string opencode parses*, not a
   per-line split.
2. If practical without new infrastructure (no committed test harness exists for either TS file
   today — the Description's own harness was a one-off script, not checked in), smoke-test with
   a scratch project and opencode's non-interactive runner: attempt the field PoC's shape (a
   command that does not itself start with `git add`/`git commit` but contains the phrase in a
   heredoc body) and observe whether `bash` is denied.
3. Record the outcome — which method was used, and the conclusion (immune / vulnerable) — as a
   dated entry in `tickets/NOTES.md`. If vulnerable, note it as a candidate follow-up per the
   rules §5 promotion test; do not fix it here (decision 3).

### Acceptance test

```
just build
just test
just lint
just docs-check
```
All clean (this ticket does not add Go test coverage — neither TS file has any today, and this
change is TS-only + a NOTES.md entry; the Go suite must simply stay green because nothing it
checks changed).

Manual verification (the same class of harness the Description used to find the bug, now proving
the fix):
1. In a pi session with the guardrail loaded (or a throwaway dir per `AGENTS.md`'s self-modify
   policy: `D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install
   --in-tree --project demo --agent pi`), run the field PoC:
   `python3 - <<'EOF'\nprint("blocks git add -A, git add ., git commit -a")\nEOF` — expect a
   **confirm prompt**, not an unconditional block; approving it must proceed, declining it must
   report `"User declined ..."` (not silently succeed).
2. Run a literal `git add -A` in that same session — expect the same confirm prompt (not a
   silent allow): the fix must not have accidentally turned rule 1 into a no-op.
3. Confirm Task 3's investigation is recorded in `tickets/NOTES.md` regardless of outcome.

### Docs update (mandatory when user-facing)

No user-facing docs change: `docs/user-manual/cli-reference.adoc`'s agent-artifact table
(`:164-183`) describes the guardrails generically ("bash guardrails", "the same rules as a Pi
`tool_call` gate") without naming rule 1's verdict, so it stays accurate under either a hard
block or a confirm. `tickets/NOTES.md` gets Task 3's finding (that is bookkeeping, not a docs
surface).

### Finish (mandatory)

1. Acceptance test green; both manual verification steps done and their outcomes noted in the
   ticket summary.
2. `tickets/NOTES.md` carries Task 3's dated finding.
3. Write a summary: both TS files touched identically, the NOTES.md entry, and Task 3's
   conclusion (immune/vulnerable) for `opencode.jsonc`.
4. Suggested commit message:
   ```
   fix(pi): make the staging guardrail a confirm, not a hard block (T-050)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-050 in-review --reason "acceptance green"`.

## Review

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — run via the `docs_readability` tool on `CHANGELOG.md` (the one
      `.md` file this ticket changed that carries shipped, user-facing prose; `tickets/NOTES.md`
      and the ticket file itself are internal bookkeeping, not in scope). All suggestions
      returned were on pre-existing entries this ticket did not touch (T-110, T-108, etc.); the
      new T-050 entry drew no suggestion. Nothing to apply.
- [x] Findings recorded with severity, class, disposition; disposition summary + cost line (step 5)
- [x] Ticket moved; `## History` appended (step 6)
- [x] Other references checked (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message/MR attributes presented for approval (step 9) — pending: this
      ticket returns to `5-rework/` first (F1 is blocking)

**Implementation audit (step 2).** All three tasks done, on `feat/T-050-pi-guardrail-confirm`
(commit `16c99e8`, plus a review-added `977fe2c`). Task 1/2: rule 1's `git add`/`stage` block and
`git commit -a` block both flipped from unconditional `return { block: true, reason }` to the
confirm idiom, identically in `agents/pi/extensions/pickle-guardrails.ts` and
`.pi/extensions/workspace-guardrails.ts` — verified line-by-line, `diff`-clean between the two
files apart from the two project-name/title strings the original files already varied by. Task 3:
`opencode.jsonc` (both copies) verified unchanged, per decision 3. Acceptance test re-run on this
branch: `just build`, `just test`, `just lint`, `just docs-check` all clean. Manual verification
re-run independently (a fresh throwaway Node harness driving the actual extension modules through
mocked `ExtensionAPI` contexts, per the ticket's own note that neither file has a committed one):
both files, with a UI, route the heredoc field PoC and a literal `git add -A`/`git commit -a`
through `ctx.ui.confirm` (approve → proceeds, decline → `"User declined the staging discipline
gate."`); without a UI, both still hard-block — matches decision 1 and the plan's manual-
verification steps 1–2 exactly. Task 3's `tickets/NOTES.md` entry (step 3 of the plan's manual
verification) present and dated 2026-08-22, citing both the opencode docs quote and the smoke-test
JSON output as its two methods.

**Quality/consistency audit (steps 3–4).** Control flow after an approved confirm falls through to
the remaining checks in the same segment exactly as rule 2 already does — no early-return bug
introduced. Wording (`"Staging discipline"` title, `"User declined the staging discipline
gate."`) matches rule 2's established style. Whole-tree search for other `"hard block"` /
`"Rule 1"` references found nothing else stale. `docs/user-manual/cli-reference.adoc`'s
agent-artifact table (`:164-183`, re-checked at its current line numbers) stays generic under
either verdict, confirming the plan's own "no docs change" call.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | docs-gap | — | **Fixed.** The new header comment in the **shipped payload** file `agents/pi/extensions/pickle-guardrails.ts` told the reader to "see T-050's ticket" / "see T-050's Description" — a ticket-lookup reference that resolves to nothing (or something unrelated) in any project other than pickle itself once `pickle install --agent pi` embeds this file elsewhere. This was exactly the first failing shape in AGENTS.md's foreign-workspace test ("a ticket id the reader is told to go and look up"). `payload_lint_test.go`'s rule 1 did not catch this phrasing (its pattern only matches a `tickets/<dir>/T-\d+` path or a `T-\d+\s*F\d+` finding shape, not "see T-NNN's ticket/Description" prose). `.pi/extensions/workspace-guardrails.ts` carried the identical sentence but is **not** shipped payload (never embedded, self-host-only), so it needed no change and was left as-is (legitimately self-referential). | Rework commit `3fbef32` on `feat/T-050-pi-guardrail-confirm`: rewrote the comment to state the evasions (`g=git; $g add -A`, `echo -A \| xargs git add`, `git commit "-""a"`) and the hardening trade-off (quote/heredoc-aware skip opens a `bash <<'EOF'` bypass; segment-start anchoring loses every `then`/`env`/`sudo`/`time` prefix) directly, with no ticket reference. Re-verified: `grep -n "T-[0-9]" agents/pi/extensions/pickle-guardrails.ts` now matches nothing; `TestPayloadSpeaksToAForeignReader` passes; a fresh harness run (same shape as the original review) reproduces identical confirm/decline/no-UI behaviour, confirming only the comment changed. | — (resolved) |
| F2 | non-blocking | stale-xref | noted | The Description's own "why the fix is the verdict, not the matcher" section still cites `assets.go:19` for the `go:embed` directive; the directive is at `assets.go:23` today (drifted by the unrelated T-110, which added header-comment lines to `assets.go`, not by this branch). The bar for `fixed inline` is "did this branch break it", and this branch did not — T-110 already had. | `assets.go:23` (current), ticket Description (`assets.go:19` citation) | Leave as noted; a future editor of that paragraph can correct it in passing. |
| F3 | non-blocking | docs-gap | fixed inline | `CHANGELOG.md`'s `[Unreleased]` → `Fixed` section had no entry for this ticket's shipped, user-facing behaviour change (rule 1 confirm instead of hard block), unlike sibling entries for T-110/T-066 already in the same section. This branch is what created the gap (a new shipped behaviour with no announcement), so the inline-fix bar ("made false by this branch") is met, and the fix is pure prose addition with no behaviour change. | `CHANGELOG.md` (`[Unreleased]` → `Fixed`, before the existing T-066 entry) | Fixed inline this review, commit `977fe2c` on `feat/T-050-pi-guardrail-confirm`; passed a `docs_readability` pass with no suggestions on the new entry. |

Disposition summary: 1 blocking (F1, routes to rework), 1 `noted` (F2), 1 `fixed inline` (F3).

cost: estimated S, actual S

## History

- 2026-07-27 — created (TO DO). source: field finding, pickle 0.1.0 guardrail false positive;
  re-scoped at triage from hardening the matcher to changing the verdict
- 2026-08-06 — patched by the T-057 review's impact sweep (finding N5, disposition `fixed inline`
  there): T-057 deliberately added **no** fourth guardrail rule (its decision 1 — a pi extension
  guards only a pi session, and the `pre-commit` hook guards every committer), but it did add a
  header paragraph to `agents/pi/extensions/pickle-guardrails.ts` saying so, which shifts this
  ticket's line references (`segments()` `:24-29` and below) by 5. Behaviour is untouched, so the
  verdict-semantics question here is unchanged; re-verify the line numbers at refinement
- 2026-08-20 — refined: line numbers re-verified in both TS files and in `opencode.jsonc`;
  confirmed rule 1 is still the sole hard block in both TS files and `opencode.jsonc` still
  denies (not asks) on the same rule, matching the Description's table. Grading unchanged (still
  a small, mechanical, two-file fix). Implementation Plan written; the `opencode.jsonc`
  matcher-immunity question (item 3) is left as a plan task, not resolved at refinement, since its
  answer is only decidable by reading or exercising opencode's own matching code.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — READY → IN DEVELOPMENT: picked up
- 2026-08-22 — implemented: applicability-gate audit (fresh sub-agent) found zero blocking
  findings and one non-blocking cosmetic finding (a stale `segments()` line citation, `:24-29`,
  in the Description's explanatory prose, unconnected to any Task step) — fixed inline, corrected
  to `:32-38`. Task 1 + Task 2: flipped rule 1's staging-discipline verdict (both the `git
  add`/`stage` block and the `git commit -a` block) from an unconditional hard `block` to the
  confirm idiom mirrored verbatim from rule 2, identically in `agents/pi/extensions/pickle-
  guardrails.ts` and `.pi/extensions/workspace-guardrails.ts`; added the header-comment paragraph
  stating the gate's reminder-not-boundary nature to both files, immediately above the `tool_call`
  handler. Verified with a throwaway Node harness (not committed, per the ticket's own note that
  neither file has one) driving the actual extension modules through mocked `ExtensionAPI`
  contexts: with a UI, both the field-PoC heredoc and a literal `git add -A`/`git commit -a` now
  route through `ctx.ui.confirm` (proceeding on approve, `"User declined ..."` on decline);
  without a UI (`!ctx.hasUI`), both still hard-block, matching decision 1's unattended-stays-
  strict requirement — identical results from both TS files. Task 3: verified (not assumed)
  that `opencode.jsonc`'s rule 1 is immune to the same class of false positive, by two methods —
  opencode's published docs (`https://opencode.ai/docs/permissions`) state `bash` permission rules
  match "parsed commands", i.e. against the command opencode's own parser resolves per tool call,
  not a naive text split; and a live smoke test (`opencode run --format json` against a scratch
  project carrying the actual rule-1 patterns) showed the exact field PoC executing clean while a
  literal `git add -A` in the same session was denied. Recorded as a dated entry in
  `tickets/NOTES.md` (appended after the pre-existing "Noted, not filed" paragraph); no change
  to `opencode.jsonc` itself (decision 3). Acceptance test green: `just build/test/lint/docs-
  check` all clean; both manual-verification cases (heredoc PoC and literal `git add -A`) now
  produce a confirm prompt rather than a silent allow or an unconditional block.
- 2026-08-22 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-22 — IN REVIEW → REWORK: F1 blocking: shipped header comment fails the foreign-workspace test (ticket-lookup phrasing in payload)
- 2026-08-22 — REWORK → IN REVIEW: findings fixed
