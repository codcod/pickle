---
id: T-098
title: the shipped payload cites this repo's own ticket ids and corpus as if the reader could look them up
project: pickle
depends-on: []
spawned-by: [T-085]
impact: low
complexity: low
cost: S
---

# T-098 — the shipped payload cites this repo's own ticket ids and corpus as if the reader could look them up

## Outcome

After this ships, a reader of the installed skill in *any* project is never sent to a ticket id
or a body of evidence that exists only in pickle's own repo: the payload's examples and
justifications either stand on their own or are explicitly labelled as pickle's, so no one
wastes a lookup on `tickets/6-done/T-090` in a workspace that has no such ticket.

## Description

Two findings from T-085's review (N2, N3), batched because they are one defect in two places —
and refinement's sweep found a **third**: **the shipped payload speaks to its reader as if that
reader were this repo.**

1. **`skill/resources/review-protocol.md:170-171`** — "Worked examples from `tickets/6-done/`:
   `T-090 F1` … `T-084 F2`". This is a lookup instruction pointing into the reader's *own*
   `6-done/`, where those ids are either absent or belong to entirely different tickets. The two
   illustrations are genuinely good — they are why `correctness` and `spec-unclear` exist as
   distinct classes — so the fix is to keep the defect shapes and drop the lookup framing, or to
   attribute them explicitly to pickle's corpus.
2. **`skill/resources/review-protocol.md:143`** — the skeleton is justified by "prose-only
   drifted into 13 header variants across **the corpus** before this skeleton existed". The
   corpus is this repo's, is named nowhere, and is unavailable to the reader the sentence
   addresses. The rule is right; its stated warrant does not travel.
3. **`skill/resources/tickets-README.md:125-126`** — *found at refinement, and the most
   consequential of the three.* The provenance vocabulary T-085 shipped defines `field-use` as
   "found using pickle on **another project**" and `self-host` as "found operating **this repo's
   own flow**". That distinction only means something in a repo where the tool hosts its own
   flow. A foreign team running brine has no "self-host" case in the pickle sense, so the two
   busiest classes are un-assignable and will be filled inconsistently.

   This is worse than a dangling example. `field-use` vs `self-host` is the axis T-085 called
   *"the weight on every other signal"* — the anti-overfitting control on a self-hosted corpus.
   A closed vocabulary whose two busiest members foreign users cannot tell apart does not merely
   read oddly; it silently produces a signal nobody can trust, which is worse than having none.

   *Precision, because the neighbouring claim is easy to overstate:* this axis carries **no**
   pre-registered criterion of its own. The criterion at `NOTES.md:882-901` governs the findings
   **`class`** column, a different field. Nothing here is on a clock — the harm is an unreliable
   signal, not a countdown to a decision made on bad data.

**Why this is a real defect and not pedantry.** The payload already distinguishes the two safe
uses of a ticket id from this unsafe one: `tickets-README.md:63-64` uses ids as pure *syntax*
illustration (`board: T-084 ready → in development` — the id is arbitrary filler) and `:237`,
`:428` use them as *provenance tags* ("(T-083)" — naming which ticket introduced a rule). Neither
tells the reader to go and read them. T-085's addition is the first that does.

`tickets/NOTES.md:869-874` records this exact anti-pattern in advance — the **foreign-workspace
test** ("would this help a project that is not pickle?") kept as "an anti-overfitting guard on an
n=36 self-hosted corpus … recorded here for whoever next proposes generalising a self-host
observation into the shipped payload". This ticket is that guard firing for the first time.

### Scope — settled by refinement's sweep

The sweep the filing deferred **has been done**, across all five payload files
(`skill/SKILL.md` and the four under `skill/resources/`). It asked three questions: which
sentences cite a ticket id, which cite an unnamed body of evidence, and which say "this repo" or
"we". Result: **three real sites** — the two filed, plus the provenance definitions above — and
the payload is otherwise clean. T-085 really was the first ticket to reach for a local example.

**Five id references are false positives and must be left alone.** The payload already
distinguishes two safe uses of an id from the unsafe one, and the fix must preserve that
distinction rather than sterilise every `T-` in sight:

| site | use | verdict |
|---|---|---|
| `tickets-README.md:63-64` | syntax filler in a commit-grammar example (`board: T-084 ready → in development`) — the id is arbitrary | keep |
| `tickets-README.md:237`, `:428`, `SKILL.md:304` | provenance tags — "(T-083)" names which ticket introduced a rule | keep |
| `review-protocol.md:240` | `(T-42)` illustrating where the id sits in a commit subject | keep |

None tells the reader to go and read anything. Only T-085's `tickets/6-done/` framing does.

**Refinement's answer on the deferred question:** the foreign-workspace test does **not** become
a payload rule. Shipping every user a rule about how to write the document they are reading is
itself an instance of this ticket's defect. It belongs in this repo's `AGENTS.md`, in the
hand-written region above `<!-- pickle:begin -->` that `pickle upgrade` will not overwrite —
which is exactly where the self-modify policy and the other self-hosting consequences already
live.

**Also out of scope:** `docs/user-manual/`. The manual is documentation *about* pickle, written
for people reading pickle's own docs, so pickle ticket ids there are correct by construction.
And no mechanical guard — see the Couplings note on recurrence.

### Couplings

Soft couplings only (no `depends-on:`):

- **T-085** (`spawned-by:`) — the ticket whose review found both instances, and which authored
  the prose in question. Nothing here disturbs the `class` vocabulary or the skeleton itself;
  this is about how they are *justified and illustrated*, not what they say.
- **T-067** (no link/anchor validation in the docs pipeline) — adjacent but different. T-067 would
  mechanically catch a dead cross-reference under `docs/`; this defect is a *semantically* dead
  reference in the payload that no anchor checker could detect, because `tickets/6-done/T-090`
  is a perfectly well-formed path that simply means something else in someone else's repo.
- **T-074** (rename the installed skill directory to brine) — touches the same payload
  documents. If both are scheduled, doing T-074 first avoids a rebase; neither blocks the other.

**No guard test, and why.** A check for lookup-shaped ticket references in the payload is ~20
lines and tempting. It is not built here, on T-085's own discipline: machinery waits for
recurrence, and this theme stands at n=1. **Pre-registered trigger** (recorded in `NOTES.md` by
Task 5): file it the *first* time a second instance appears in a review — at n=2 the pattern is
real and the check pays for itself. Until then the acceptance test below is the guard, run once.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                                   # `pickle` is the root-path child
git checkout main
git checkout -b feat/T-098-payload-speaks-to-its-own-repo
```

WIP commits encouraged. **Root-path child** (`path = "."`): the Finish step tidies WIP into
atomic commits and keeps that history rather than squashing. Ticket and board bookkeeping stays
on `main`, never on this branch.

### Prerequisite gate (hard)

None. `depends-on:` is empty. **T-085 is merged** (PR #42, `89bccc5`) and its prose is on `main`,
so the three target sites exist exactly as quoted — verify the line numbers before editing, since
any later payload commit shifts them. T-074 is a soft coupling only: if it lands first this
branch takes a trivial rebase, and neither ticket blocks the other.

### Confirmed design decisions (do not deviate without asking)

1. **Payload prose only. No Go code, no `board audit` check, no new command.** Nothing under
   `internal/` or `cmd/` changes. If a task appears to need a code change, stop and ask — it
   means the scope slipped.
2. **The five provenance tokens are untouched.** Reword the *definitions* only. T-085 decision 5
   forbids adding, renaming or re-ordering the values, and the pre-registered criterion needs a
   fixed vocabulary to count against. The literal
   `<field-use|self-host|review|audit|chat>` string must survive byte-identical in all three
   places it appears.
3. **The eight `class` values are not touched at all** — not the tokens, not the definition
   table, not the one-line tests. Only the two sentences of *illustration and warrant* around
   them change.
4. **Worked examples lose their ids and the `tickets/6-done/` framing.** Keep the two defect
   shapes — they are the pedagogically valuable part — and add the *why* each maps to its class.
   Do not replace the ids with an attribution like "from pickle's corpus": that still hands the
   reader an id they cannot use.
5. **The foreign-workspace test does not enter the payload.** It goes in `AGENTS.md`, above
   `<!-- pickle:begin -->`. A rule about how to write the payload, shipped inside the payload, is
   this ticket's own defect.
6. **No guard test.** Pre-register the trigger in `NOTES.md` instead (Task 5). n=1.
7. **Do not touch the five false positives** listed in the Description's Scope table
   (`tickets-README.md:63-64,237,428`, `SKILL.md:304`, `review-protocol.md:240`). They are
   syntax filler and provenance tags, they are legitimate, and blanket-removing every `T-`
   reference would destroy information. The acceptance test asserts they are still there.
8. **`docs/user-manual/` is out of scope** — it documents pickle, so pickle ids are correct
   there. Confirm rather than assume (see Docs update).
9. **Self-modify policy** (`AGENTS.md`): never run `pickle install|upgrade|uninstall` against
   this repo from this branch. `.agents/skills/ticket-flow/` is a symlink to `skill/` — edit
   `skill/` and the installed skill follows. Any test install goes to a throwaway dir with the
   binary copied in.

### Tasks

#### Task 1 — the worked examples lose their lookup framing (N2)

`skill/resources/review-protocol.md:170-171`. Replace:

> Worked examples from `tickets/6-done/`: `T-090 F1` (a byte-widened `unicode.IsSpace` scan
> emitting invalid UTF-8) is `correctness`; `T-084 F2` (a scope rule satisfying both its own
> branches) is `spec-unclear`.

with a self-contained pair that keeps both shapes and states the reasoning, roughly:

> Two worked examples. A byte-widened `unicode.IsSpace` scan that emits invalid UTF-8 is
> `correctness` — it ships wrong output, regardless of how it reads. A scope rule that satisfies
> both of its own branches is `spec-unclear`, not `docs-gap` — the documentation exists, it just
> cannot be executed against.

The added contrast (`spec-unclear` *not* `docs-gap`) is the point: it does the discriminating
work the bare id never did.

#### Task 2 — the skeleton's warrant stops citing an invisible corpus (N3)

`skill/resources/review-protocol.md:143`. The parenthetical currently justifies the skeleton with
"prose-only drifted into 13 header variants across the corpus before this skeleton existed".
Replace the count with the mechanism, so the warrant travels: restating a table's columns in
prose lets every author reinvent the header, and a findings table that cannot be parsed the same
way twice cannot be counted at all. Do not cite a number.

#### Task 3 — the provenance definitions become assignable by a foreign reader

`skill/resources/tickets-README.md:125-126`. Today: `field-use` is "found using pickle on another
project", `self-host` is "found operating this repo's own flow". Redefine both against a test any
project can apply — the distinction that actually carries the signal is **using the flow to do
real work** versus **maintaining the flow itself**:

- `field-use` — surfaced while using the flow to ship something else; the finding is a
  by-product of real work.
- `self-host` — surfaced while working *on* the flow, its tooling or its payload.

Keep the sentence that follows about why the axis matters (a single-repo corpus is what one would
overfit to), but make it about *any* project's corpus rather than this one. **Tokens unchanged**
(decision 2).

Check whether `skill/SKILL.md:161` and `skill/resources/TEMPLATE.md:153,171` need a matching
touch: they carry the token list but, at filing, no glosses — if that holds, they need no edit,
and the acceptance test's byte-identical token check covers them.

**Also in this task (added at pickup — see History):** `skill/resources/tickets-README.md:320`
reads "(`pickle`'s own `skill/resources/TEMPLATE.md` ships exactly these seven)". `skill/` is a
path that exists only in pickle's repo; in an installed workspace the file is at
`.agents/skills/ticket-flow/resources/TEMPLATE.md`. Reword to name the file **relative to the
skill the reader is holding** — "this skill's own `resources/TEMPLATE.md`" — which is both
correct everywhere and shorter. The pickup audit found this site; it escapes all four acceptance
greps, which is why it survived the refinement sweep.

#### Task 4 — the foreign-workspace test, recorded where it binds

`AGENTS.md`, in the hand-written region **above** `<!-- pickle:begin -->` (line ~34), alongside
the existing self-hosting consequences. One bullet: payload prose is read by projects that are
not pickle, so before shipping a sentence into `skill/`, apply the **foreign-workspace test** —
*would this sentence help a project that is not pickle?* Name the failure shape concretely (a
ticket id the reader cannot resolve, a count from a corpus they do not have, "this repo" meaning
ours) and the safe alternatives (syntax filler and provenance tags, per this ticket's Scope
table). Cite T-098 so the reasoning is one hop away.

Do **not** edit anything between the marker comments — that region is generated.

#### Task 5 — pre-register the guard-test trigger

`tickets/NOTES.md`. Record: the sweep was run on this date across all five payload files and
found three sites; the payload is otherwise clean; a mechanical check was **deliberately not
built** at n=1, per T-085's discipline; and the trigger is a **second** instance found in any
review, at which point file the check. Include the two `rg` patterns from the acceptance test so
whoever files it does not have to re-derive them. Bookkeeping — commit on `main`, not on this
branch.

### Acceptance test

No Go code changes, so the build must be untouched — that is itself the check:

```sh
just build && just test && just lint && just docs-check
git diff --name-only main...HEAD | grep -E '^(internal|cmd)/|\.go$'   # expect: NO output
```

Then the payload's own consistency — note that checks 2 and 5 are **guards against
over-correction**, not just under-correction:

```sh
# 1. no lookup-shaped ticket reference survives anywhere in the payload
rg -n 'tickets/6-done/T-[0-9]|T-[0-9]{3} F[0-9]' skill/
#    expect: no output

# 2. the five legitimate id references are STILL THERE (decision 7)
rg -n 'T-[0-9]+' skill/ | grep -v 'T-NNN\|T-MMM\|T-KKK\|T-xxx'
#    expect exactly 6 lines: tickets-README.md:63 and :64 (one example spans both),
#    tickets-README.md:237, tickets-README.md:428, SKILL.md:304, review-protocol.md:240

# 3. no unattributed corpus, no first-person repo
rg -n 'the corpus|this repo' skill/                    # expect: no output

# 4. the provenance token list is byte-identical everywhere it appears (decision 2)
rg -cF '<field-use|self-host|review|audit|chat>' skill/
#    expect exactly: SKILL.md:1, tickets-README.md:1, TEMPLATE.md:2 (four hits, three files).
#    TEMPLATE.md carries it twice — the History comment block and the seed line — which is why
#    this counts hits rather than files.

# 5. the eight class values are byte-identical (decision 3)
diff <(git show main:skill/resources/review-protocol.md | sed -n '/^| class | test |/,/^$/p') \
     <(sed -n '/^| class | test |/,/^$/p' skill/resources/review-protocol.md)
#    expect: no output — the vocabulary table round-trips unchanged

# 6. the constraint landed outside the payload, not inside it
rg -n 'foreign-workspace' AGENTS.md     # expect: the new bullet (was: absent)
rg -n 'T-098' tickets/NOTES.md          # expect: the pre-registered trigger entry
rg -n 'foreign-workspace' skill/        # expect: no output
#    NB: `foreign-workspace` already occurs in NOTES.md (the T-085 exploration note at :869-874),
#    so grep for the T-098 entry there rather than the phrase, or the check passes vacuously.
```

End-to-end, in a **throwaway directory with the binary copied in** (`AGENTS.md` self-modify
policy — never the in-repo path):

```sh
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
./pk install --project demo . >/dev/null
./pk board audit                                        # expect: clean
rg -n 'tickets/6-done/T-[0-9]' .agents/skills/ticket-flow/   # expect: no output
```

The last line is the one that matters: it proves a **freshly installed foreign workspace** no
longer carries a reference to a ticket it does not have — which is the entire Outcome, tested
from the reader's position rather than from this repo's.

### Docs update (mandatory when user-facing)

The payload **is** the user-facing surface, and Tasks 1–3 are the docs change in full:
`skill/resources/review-protocol.md` and `skill/resources/tickets-README.md`. `AGENTS.md` and
`tickets/NOTES.md` (Tasks 4–5) are repo-internal and ship to nobody.

Confirm rather than assume: `rg -n 'the corpus|Worked examples' docs/ README.md` — expected empty,
since the manual paraphrases the review record without reproducing these two sentences. If it
turns out to quote either, update it too. `docs/user-manual/`'s own pickle ticket ids stay
(decision 8).

Add a `CHANGELOG.md` `[Unreleased]` → **`### Fixed`** entry (the section exists): shipped prose
assumed the reader was pickle's own repo — worked examples pointed into a `tickets/6-done/` the
reader does not have, and the `field-use`/`self-host` definitions were un-assignable outside this
repo. T-085 set the precedent that payload-prose changes earn an entry. `just docs-check` must
pass.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean; the
   `git diff` guard prints nothing; checks 2, 4 and 5 confirm nothing was over-corrected.
2. Docs updated per Tasks 1–3 plus the `CHANGELOG.md` entry.
3. Write a summary: the three sites fixed, the **six id references deliberately kept** and why,
   and confirmation that the provenance tokens and the `class` vocabulary are byte-identical.
4. Suggested Conventional Commit message:

   ```
   fix(skill): stop the payload addressing its reader as this repo (T-098)

   Worked examples pointed into `tickets/6-done/` for ticket ids that exist only
   here; the table skeleton was justified by a header-variant count from a corpus
   the reader cannot see; and the `field-use`/`self-host` provenance definitions
   were un-assignable in any project that does not host this flow itself -- the
   axis a pre-registered criterion depends on. Examples are now self-contained and
   say why each maps to its class, the warrant states the mechanism instead of a
   number, and the two classes are defined as using-the-flow versus working-on-it.

   Provenance tokens and the eight class values are byte-identical; the legitimate
   id references (syntax filler, provenance tags) are deliberately kept. The
   foreign-workspace test that catches this recurring is recorded in AGENTS.md,
   outside the payload, because a rule about writing the payload does not belong
   inside it.
   ```

5. **Tidy up before presenting** — root-path child, so interactive-rebase the WIP commits into a
   small number of atomic commits (suggested split: the two `review-protocol.md` edits; the
   provenance definitions; `AGENTS.md` + `CHANGELOG.md`) and keep that history rather than
   squashing.
6. Commit locally on the branch. Do **not** push or open an MR without explicit approval. Present
   the commit messages; after approval, verify
   `git fetch origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` prints
   nothing, then push and open the MR. Merging is the human's. Task 5's `NOTES.md` edit is
   bookkeeping and is committed separately, on `main`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-13 — created (TO DO). source: review: batched from T-085's review findings N2 and N3,
  both dispositioned *new ticket* — one theme (the payload addressing its reader as if the reader
  were this repo), two sites, filed as one ticket per rules §5's batching requirement
- 2026-08-13 — refined. Ran the sweep the filing deferred, across all five payload files, asking
  which sentences cite a ticket id, which cite unnamed evidence, and which say "this repo".
  Found a **third** site and the most consequential one: the `field-use`/`self-host` provenance
  definitions (`tickets-README.md:125-126`) are pickle-shaped and un-assignable by a foreign
  team, degrading the very axis T-085 called the weight on every other signal. Confirmed the
  payload is otherwise clean, and identified **six id references that are legitimate** (syntax
  filler and provenance tags) and must survive — the plan now guards against over-correction as
  well as under-correction. Four deferred questions settled with the user: include the third
  site but reword definitions only, leaving the five tokens byte-identical (T-085 decision 5);
  drop the worked examples' ids rather than attribute them, adding the *why* instead; keep the
  foreign-workspace test **out** of the payload and in `AGENTS.md`, since shipping a rule about
  writing the payload inside the payload is this ticket's own defect; and build no guard test at
  n=1, pre-registering a trigger at n=2 instead. Every acceptance-test command was executed
  during refinement and three were corrected — `TEMPLATE.md` carries the token list twice not
  once, `foreign-workspace` already occurs in `NOTES.md` so that check would have passed
  vacuously, and the class-vocabulary guard is now a `diff` round-trip rather than a fragile
  diff-grep. Also corrected an overstatement in this ticket's own Description: the provenance
  axis carries no pre-registered criterion — that criterion governs the findings `class` column,
  a different field. Grades unchanged at low/low/S
- 2026-08-13 — TO DO → READY: plan complete: 3 sites from the sweep, prose only, guards both ways
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
- 2026-08-13 — plan amended inline: pickup applicability audit found a **fourth** site the
  refinement sweep missed — `tickets-README.md:320` cites `skill/resources/TEMPLATE.md`, a
  pickle-repo-only path, where an installed workspace has
  `.agents/skills/ticket-flow/resources/TEMPLATE.md`. It escapes all four acceptance greps (it is
  neither a ticket id, nor "the corpus"/"this repo", nor a token-list line), which is exactly why
  the sweep did not see it. Folded into Task 3 as a one-line reword rather than a new ticket:
  same theme, same file, same commit. Three further findings noted and closed as imprecision:
  this ticket's `NOTES.md` line cites are off by ~5 (actual `862-870` and `~880-887`), the worked
  examples span `review-protocol.md:170-172` not `:170-171`, and the end-to-end block's
  `cp pickle` picks up a stale binary unless `just build` has run first (the acceptance test
  already orders it that way). Audit otherwise confirmed the plan: all quoted sites verbatim at
  the stated lines, zero board delta since READY, and every read-only check behaving as predicted
- 2026-08-13 — READY → IN DEVELOPMENT: picked up
