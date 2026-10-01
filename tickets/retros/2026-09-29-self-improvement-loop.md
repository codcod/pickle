# Retro 2026-09-29 — self-improvement loop exploration, and the first baseline

First report in `tickets/retros/`. It records how this project learns from its own agent
sessions, the decisions taken, and the one target measured so far. Later reports start from the
**Open targets** section of the newest file here. Earlier session-review decisions from the same
day are in `NOTES.md` § *Session review and the 1.x release plan (2026-09-29)*. From now on,
reports live here and not in NOTES.

## Question

Can reviewing agent sessions over time, plus noticing model changes, become a repeatable loop
that improves brine? What would that loop need to be trustworthy?

## Corpus

| Weeks | Host | Sessions | Dominant models |
|---|---|---|---|
| 2026-W29..W35 (Jul–Aug) | pi | 144 | Opus 4.8, Fable 5, Opus 5, Sonnet 5, Gemini 2.5 Pro, GPT-5.4 |
| 2026-W36..W37 | — | 0 | nothing found in any host's session store |
| 2026-W38 | Claude Code | 105 | Sonnet 5 only |
| 2026-W39 | Claude Code | 184 | Opus 5.5 (125), Sonnet 5 (49) |
| 2026-W40 (2 days) | Claude Code | 28 | Opus 5.5 only |

- **Sessions sit in two hosts, in two formats.** A loop that reads one host sees part of the
  history.
- **Raw transcripts do not last.** Claude Code deletes them after `cleanupPeriodDays`, 30 by
  default. The W38 sessions start disappearing mid-October. The dated reports here are the
  durable record; raw transcripts are only the input for the current window.

## Findings

1. **The best signal is what the human types that is not a trigger phrase.** In one week that
   was 31–33 `/code-review` runs after brine validate, about 25 "pr N merged" reports, bare
   go-aheads, and rework/validate relaunches. Each recurring one is either a missing automation
   or a flaw in the flow. They led to T-137, T-138 and T-140.
2. **The model change is real and relevant.** Brine was written in W30–W33 against a
   multi-vendor mix (including Gemini 2.5 Pro and GPT-5.4). Its defensive, repetitive, emphatic
   style dates from then. Since W40 the user runs one frontier model. `/code-review` after
   validate went from 0 (W38) to 33 (W39), the week of the switch to Opus 5.5. The stronger
   model made an out-of-band review worth running, and brine's own review did not scale with
   it (T-137).
3. **Brine cannot be tuned to one model.** It ships to other projects and agents, running
   whatever model they use. The principle: write prose for a strong reader, and keep the rails
   in mechanism (hooks, `board audit`, CLI refusals) that also hold for weak models. T-135 covers
   the prose side; no new ticket is needed.
4. **Unmeasured claims were wrong more than once in a single pass.** "Correctness bugs in 29 of
   31 PRs" was really "2 of 31 runs explicitly reported none". "Nearly every gate run stopped for
   approval" turned out to be 17% (below). A loop without measured baselines files tickets for
   problems that do not exist.

## Decisions

- **`pickle retro` is filed as T-141.** It prints a versioned prompt, in the style of `snowball
  docs-prompt`, that locates the data and states the method. By the user's direction it
  overrides NOTES.md § "Rejected outright, so they are not re-proposed". It is a printed prompt,
  not a skill: nothing to install, nothing added to every session's skill list, and any agent
  can run it.
- **Reports live in `tickets/retros/`**, one dated file per run, with each target's measurement
  script saved beside it. They do not go in NOTES.md. `board audit` ignores the directory
  (checked).
- **T-139 is dropped.** Its baseline, measured before refinement, disproved its premise (below).
- **Release plan, replacing NOTES' revised plan:**
  - **1.1.0:** T-140 and T-141.
  - **1.2.0:** T-137, then T-138.
  - **Later:** T-135.

## What a target is

A target is attached to each proposal before it is built. It has:

- **Claim:** what the change is meant to do.
- **Unit:** what one observation is.
- **Metric:** a numerator and a denominator, each defined well enough for a script to count.
- **Baseline:** measured now, with the same script.
- **Threshold:** the value to reach, and the minimum sample before judging.
- **Guard:** something that must not get worse.
- **Decision rule:** what happens when the threshold is met, missed, or the result is
  inconclusive.

The next retro runs the script and applies the rule. Two rules come from how this first
measurement went:

- **Validate a metric by hand on a sample before quoting it.** The script below took three
  versions:
  - **6%:** background-task notifications were counted as the human's replies.
  - **92%:** the gate's verdict arrives as a hidden hand-back message, and the script missed it.
  - **17%:** correct, but only after one raw transcript was read by hand.
- **A target may need the flow to write a machine-readable line.** The guard below cannot be
  measured reliably from prose. A regex marked 10 runs "blocking", and a spot-check showed the
  matches were on the wrong text. An exact guard needs the gate to end with a fixed line such as
  `verdict: clean|blocking`.

### Worked example — T-139, measured and dropped

| Part | Definition |
|---|---|
| Claim | When the applicability gate finds nothing blocking, the session goes ahead without asking. |
| Unit | One gate run: an agent spawned with an applicability-gate prompt. |
| Clean | The gate's verdict reports no blocking finding. |
| Stopped | After the verdict, the human sends a real message before the agent cuts the branch or moves the ticket to development. |
| Metric | Stopped clean runs ÷ clean runs. |
| Baseline (2026-09-21..29) | 53 runs, 42 clean with a decision recorded. **7 stopped (17%); 4 of those got a bare "go".** |
| Threshold that was proposed | ≤ 5% over at least 20 clean runs. |
| Guard | Blocking runs that went ahead without the human = 0. **Not measurable yet** (see above). |
| Result | Agents already go ahead in about 83% of clean runs. The fix would save about 4 prompts a week, so the ticket was dropped. |

Script: [`2026-09-29-gate-stops.py`](2026-09-29-gate-stops.py) —
`python3 2026-09-29-gate-stops.py 2026-09-21 2026-09-29`. It reads Claude Code transcripts from
`$CLAUDE_CONFIG_DIR/projects`, or from `~/.claude/projects` when that variable is unset.

## Weekly baseline — things typed by hand

From [`2026-09-29-weekly.py`](2026-09-29-weekly.py) (Claude Code only). These are regex counts on
human messages: indicative, not validated. Validate a sample before promoting any column to a
target baseline. The `rules-co` column also matches pasted text that happens to mention
`CLAUDE.md`, so it is noisy.

```
week     sessions    brine  trigger code-rev pr-merge go-ahead model-sw rules-co  dominant model
2026-W38      105       82       42        0       24       10        0        2  sonnet-5 99
2026-W39      184      172      139       33       49       26       22       28  opus-5-5 125, sonnet-5 49
2026-W40       28       26       24        6        9        9        0        5  opus-5-5 28
```

## Open targets

None are formal yet. Each of these tickets gets its target at refinement, with the baseline
measured by a script saved here:

- **T-140:** "pr N merged" messages per recorded merge. Starting point: the `pr-merge` column
  above, which needs validating.
- **T-137:** formal since 2026-10-01, measured at pickup. Script:
  [`2026-10-01-code-review-after-validate.py`](2026-10-01-code-review-after-validate.py) —
  `python3 2026-10-01-code-review-after-validate.py SINCE UNTIL [--show SID8]`.

  | Part | Definition |
  |---|---|
  | Claim | brine's review finds the correctness bugs a separate code-review pass was finding after it |
  | Unit | one out-of-band code-review run, in a project folder that saw a validate/review trigger in the 48 hours before it |
  | Counted | the run reports ≥ 1 finding it itself labels a bug (correctness, real or possible); refactors and consistency items do not count |
  | Metric | counted runs ÷ runs |
  | Baseline (2026-09-21..29) | 34 runs. **Regex floor 26% (9/34).** The parser classified 11 runs and 10 of them reported ≥ 1 bug; 23 replies matched no pattern |
  | Validation | 3 runs read by hand: SMP-007 (4 bugs) and SMP-009 (2) matched, POR-008 missed (4 "possibly real bugs" in a nested bullet). Free-text replies vary too much for a regex |
  | Threshold | ≤ 25% over ≥ 10 runs, **classified by reading each reply** (`--show`); the regex count is only a floor |
  | Guard | median token use of validate/review sessions ≤ 2× baseline. Baseline: 73 sessions, median 4.19M tokens (cache reads included) |
  | Decision rule | met → keep; missed → re-examine the angles against the escaped bugs; < 10 runs within six weeks of release → inconclusive, re-measure at the next retro |

  Premise check at pickup (stop below 30%): the floor is under it, but every direct reading —
  10 of 11 classified runs, the one hand-sampled unclassified run, and the original session read
  (2 of 31 runs explicitly reported no correctness bug) — is far above it. Proceeded on the
  user's decision.
- **T-138:** rework → validate cycles relaunched by the human in a fresh session.
- **T-141:** its own pre-registered criterion. If three months after it ships no filing, drop
  or re-grade cites a retro report, remove the command.

## Caveats

- **pi sessions were only used for the model timeline.** Their human messages were not mined.
- **Personal settings.** The user may raise `cleanupPeriodDays` in their Claude Code settings.
  That is a personal setting, not pickle's to change.
- **Public repository.** Reports are committed to a public repository. This one names only
  public repositories and quotes no human message beyond short fragments.
