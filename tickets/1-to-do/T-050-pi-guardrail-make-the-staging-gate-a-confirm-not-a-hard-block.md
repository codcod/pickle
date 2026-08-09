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
string: `segments()` (`:24-29`) splits on `&&`, `||`, `;` and `\n`, so the heredoc's body line
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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: field finding, pickle 0.1.0 guardrail false positive;
  re-scoped at triage from hardening the matcher to changing the verdict
- 2026-08-06 — patched by the T-057 review's impact sweep (finding N5, disposition `fixed inline`
  there): T-057 deliberately added **no** fourth guardrail rule (its decision 1 — a pi extension
  guards only a pi session, and the `pre-commit` hook guards every committer), but it did add a
  header paragraph to `agents/pi/extensions/pickle-guardrails.ts` saying so, which shifts this
  ticket's line references (`segments()` `:24-29` and below) by 5. Behaviour is untouched, so the
  verdict-semantics question here is unchanged; re-verify the line numbers at refinement
