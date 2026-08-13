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

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.md` files (step 4b) — run on
  `skill/resources/review-protocol.md` and `skill/resources/tickets-README.md`; two of its
  twelve suggestions landed on lines this branch authored and corroborate N1 and N2 below, which
  is why they are findings rather than discarded polish. The other ten target prose the ticket
  did not touch and were discarded as out of scope
- [x] Findings recorded with severity, class and disposition; disposition + cost lines present (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated if needed; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval (step 9)

**Acceptance test: re-run verbatim, all green.** `just build`, `just test`, `just lint`,
`just docs-check` clean. The no-Go-code guard prints nothing. Checks 1–6 all pass, including the
two over-correction guards: the six legitimate id references survive, the provenance tokens count
1/1/2, and the eight-value `class` table `diff`s identical against `main`. The end-to-end install
into a throwaway dir gives a workspace matching neither `tickets/6-done/T-[0-9]` nor
`the corpus|this repo`, with `board audit` clean. `pickle doctor`: 0 errors, 0 warnings.

**Tasks 1–5: all met.** The one deviation from the plan's literal wording is deliberate and
correct — Task 3 prescribed `self-host` as "working *on* the flow, its tooling or its **payload**",
but "payload" is pickle-internal vocabulary that appears nowhere in the shipped skill except
`SKILL.md:312`'s "payload version". Shipping it in a gloss would have been a fresh instance of
this ticket's own defect. "its documentation" was used instead.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | spec-unclear | — | The vocabulary's freeze is justified by "the **pre-registered criterion** this column exists to test" — a definite-article reference to pickle's own research design, which lives in *this repo's* `tickets/NOTES.md` and does not exist in any other project. Same defect family as the two sites this ticket fixed, seven lines above one of them. While it stands, the ticket's Outcome ("never sent to … a body of evidence that exists only in pickle's own repo") is false as written | `skill/resources/review-protocol.md:156-157`; sole hit of `rg -n 'pre-registered' skill/`; escapes all four acceptance patterns | Replace the warrant with one that travels: the vocabulary is frozen because its value is comparability across reviews, which a vocabulary that shifts under you destroys. Do not mention a criterion the reader does not have. Check 5's `diff` guard is unaffected — it starts at the `\| class \| test \|` header, below this line |
| N1 | non-blocking | design | fixed inline | The new parenthetical echoes the clause five words before it: "…rather than restating the column list in prose (a column list restated in prose lets every author…)". Also "cannot be counted at all" quietly presumes a counting party the reader does not have | `skill/resources/review-protocol.md:142-144`; independently flagged by the step-4b readability pass | "…rather than restating the column list in prose (a prose list lets every author reinvent the header, and reviews whose tables are not shaped the same way twice cannot be compared)" |
| N2 | non-blocking | spec-unclear | fixed inline | "a backlog is easiest to overfit to whatever its authors were doing at the time" — superlative with no comparison class, and the direction is inverted: one overfits *to* a backlog; the backlog *drifts toward* its authors' activity | `skill/resources/tickets-README.md:129-131`; flagged independently by the readability pass and the adversarial read | "a backlog drifts toward whatever its authors happened to be doing when they filed, so knowing which findings came from real use, which from working on the flow itself, and which from idle chat matters more than any other axis" |
| N3 | non-blocking | spec-unclear | fixed inline | `self-host`'s gloss — "working *on* the flow itself, its tooling or its documentation" — can be read as *subject matter* rather than *activity*, under which any flow-related finding surfaced during real work satisfies both classes. The "surfaced while" prefix does disambiguate on a careful reading, but a careful reader still misread it, which is evidence enough for a vocabulary whose whole purpose is consistent assignment | `skill/resources/tickets-README.md:125-128`; adversarial reviewer walked three concrete foreign-project findings and could assign only one | Add an explicit tie-break: `self-host` is when the ticket's *product* is a change to the flow, its tooling or its own docs; `field-use` is everything else surfaced while shipping; when both fit, `self-host` wins |
| N4 | non-blocking | design | fixed inline | Worked example 1 justifies itself with "it ships wrong output", which restates the `correctness` row verbatim, so it adds none of the discriminating power example 2 gets from its "not `docs-gap`" contrast. "regardless of how it reads" also has no clear antecedent | `skill/resources/review-protocol.md:171-172` vs the table row at `:161` | "…is `correctness`, even if the surrounding code and its comments read as correct" — giving example 1 a contrast of its own |
| N5 | non-blocking | spec-unclear | fixed inline | The guard says "**Three** shapes fail it" and lists three — but the repo-only **path** shape, the one that escaped the refinement sweep *and* all four greps and that this branch fixed at `tickets-README.md:320`, appears only obliquely in the later "the fix is…" sentence. `NOTES.md` calls it "a fifth shape". The guard under-lists precisely the shape that actually evaded detection | `AGENTS.md:35-38` vs `tickets/NOTES.md`'s closing paragraph | Promote the path shape to a fourth enumerated item and say "Four shapes" |
| N6 | non-blocking | other | noted | The adversarial reviewer argued the four `(T-083)`-style provenance tags are the same defect as the deleted worked-example ids and should also go. **Decision upheld, not adopted:** plan decision 7 locks them as legitimate and acceptance check 2 asserts their presence. A provenance tag is opaque-but-harmless to a foreign reader; "Worked examples from `tickets/6-done/`" instructed a lookup. Recorded so a later reviewer can promote it by citing this row rather than re-deriving the argument | `skill/SKILL.md:304`, `skill/resources/tickets-README.md:239`, `:430`, `review-protocol.md:242` | none — keep as is |
| N7 | non-blocking | other | new ticket | **The pre-registered guard-test trigger has fired.** `NOTES.md` (written by this ticket's own Task 5) says to file the mechanical check the first time a second instance is found in a review. Two further instances have now surfaced *after* the refinement sweep declared the payload clean: the repo-only path caught at pickup, and F1 caught here. The sweep is demonstrably not self-checking, and both misses escaped all four `rg` patterns | this review (F1) + the pickup audit's fourth site; trigger recorded in `tickets/NOTES.md` | Filed as **T-099** (`--spawned-by T-098`). Must catch the two shapes greps currently miss: repo-only paths, and definite-article appeals to evidence the reader lacks |

### Rework round 1 — fixes applied, verbatim per each suggestion

All on the same branch, one commit (`eed05ea`):

- **F1** — `review-protocol.md:156-157`. Replaced the pre-registered-criterion warrant with
  "its value is comparability across reviews, which a vocabulary that shifts under you
  destroys." `pre-registered` no longer appears anywhere under `skill/`.
- **N1** — `review-protocol.md:142-144`. Parenthetical replaced verbatim per the suggestion:
  "a prose list lets every author reinvent the header, and reviews whose tables are not shaped
  the same way twice cannot be compared."
- **N2** — `tickets-README.md:129-131`. Sentence replaced verbatim per the suggestion: "a
  backlog drifts toward whatever its authors happened to be doing when they filed…"
- **N3** — `tickets-README.md:125-128`. Added the explicit tie-break: `self-host` is when the
  ticket that would carry the finding changes the flow, its tooling or its own docs;
  `field-use` is everything else surfaced while shipping.
- **N4** — `review-protocol.md:171-172`. Example 1 now reads "is `correctness`, even if the
  surrounding code and its comments read as correct" — its own contrast, not the table row
  restated.
- **N5** — `AGENTS.md:33-48`. "Three shapes" → "Four shapes", the repo-only path promoted to
  its own enumerated item and flagged as "the shape most easily missed."

**Re-verified: the full acceptance test, re-run verbatim.** `just build`/`test`/`lint`/
`docs-check` clean; the no-Go-code guard prints nothing; checks 1–6 all pass including both
over-correction guards (six id keepers intact, tokens 1/1/2, `class` table `diff`-identical
against `main`); a fresh e2e install into a throwaway dir matches neither
`tickets/6-done/T-[0-9]` nor `the corpus|this repo`, `board audit` clean. Scope held at exactly
F1 + N1–N5 — no new finding surfaced during the fix pass.

### Scoped re-review of round 1 — 5 of 6 verified, 1 not resolved

Re-verified only the six rows the rework was scoped to, per the protocol's scoped-re-review
rule. Mechanical state is green throughout: `just build`/`test`/`lint`/`docs-check` clean, the
no-Go-code guard silent, checks 1–6 pass (six id keepers, tokens 1/1/2, `class` table
`diff`-identical against `main`).

| finding | verdict |
|---|---|
| F1 | **fixed.** `rg -n 'pre-registered' skill/` now returns nothing; the warrant reads "its value is comparability across reviews, which a vocabulary that shifts under you destroys" — true in any project, cites nothing the reader lacks |
| N1 | **fixed.** Applied verbatim; the echo is gone and the sentence is shorter |
| N2 | **fixed.** Applied verbatim; direction corrected, superlative gone |
| N3 | **not resolved** — see F2 |
| N4 | **fixed.** Example 1 now carries its own contrast rather than restating the `correctness` row |
| N5 | **fixed.** "Four shapes", path shape enumerated and flagged as the most easily missed |

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F2 | blocking | spec-unclear | — | **N3's fix replaced an ambiguity with a contradiction.** The parenthetical glosses still define `field-use`/`self-host` by **activity** ("surfaced while…"), but the tie-break added beneath them says "the test is the ticket's own *product*, **not the activity it was noticed during**" — which repudiates the criterion the glosses are written in. The two halves give different answers, and the glosses are what a reader actually applies, since they sit attached to the tokens | `skill/resources/tickets-README.md:125-133`. Blind test: a reader given only this paragraph and four scenarios reported it "decidable, but not consistent as written", found the glosses "mis-sort three of four cases" read alone, and isolated a **clean unresolved divergence** the worked example does not cover — *"the READY gate blocked a legitimate hotfix, file a ticket proposing an exception path"*: the `field-use` gloss ("by-product of real work") says `field-use`, the tie-break says `self-host` | Phrase **both glosses by product** so gloss and tie-break state one criterion, then delete the tie-break sentence as redundant: `field-use` (the ticket's product changes what you ship, not the flow — the finding is a by-product of real work), `self-host` (the ticket's product changes the flow itself, its tooling or its own docs). Keep one short example *agreeing* with the glosses: "a slow `board audit` spotted mid-feature is `self-host` — fixing it changes the tooling." This is net **shorter** than what is there now |

**Disposition summary:** 1 finding — 1 blocking (F2 → rework round 2); 0 non-blocking. F1 and
N1–N5 all verified fixed; no finding outside the round-1 scope was raised, and the review did
not re-audit the feature.

`cost: estimated S, actual M — three rounds on one S-sized prose ticket; the sites were cheap,
but the definitional passage needed more care than "reword two glosses" implied`

**Why F2 blocks rather than being fixed inline.** Rework round 1 authored the contradiction, so
by rules §5 it *is* eligible for *fixed inline* — the bar is causation, and this branch caused it.
It is nonetheless blocking, on this ticket's own stated standard: the Description calls the
`field-use`/`self-host` axis "the most consequential of the three" sites and argues that an
unassignable vocabulary "silently produces a signal nobody can trust, which is worse than having
none". Glosses that mis-sort three of four cases when read alone fail that bar. Nor should the
ticket's headline passage be rewritten by the reviewer's own hand and shipped unre-verified —
that is precisely the content that earns a second pair of eyes.

**Round 2 scope is F2 alone**, and the suggestion above is the whole fix. The two prior rounds
found sites; this one is a single contained passage with the replacement text already written.

### Rework round 2 — F2 fixed

One commit (`9c5539f`), `tickets-README.md:125-133` only. Both glosses rephrased to the product
test the tie-break used ("the ticket's product changes what you ship, not the flow…" /
"…changes the flow itself, its tooling or its own docs"); the tie-break sentence deleted as
redundant and replaced by a worked example that **agrees** with the glosses instead of
overriding them ("A slow `board audit` spotted mid-feature is `self-host`, not `field-use` —
fixing it changes the tooling, even though it was noticed while shipping something else").

**Re-verified with a second blind read** (same method as the finding, fresh scenarios plus the
original four): given only the new paragraph and five cases, the reader classified all five
consistently, reported the worked example's "even though" clause "doing the same work as the
glosses' dash-clauses", and concluded "no tension… the paragraph never claims context determines
class". The specific divergence F2 found — the hotfix-exception ticket — now resolves to
`self-host` by the gloss alone, no tie-break needed.

**Full acceptance test re-run and green:** `build`/`test`/`lint`/`docs-check` clean, no-Go-code
guard silent, checks 1–6 pass including both over-correction guards, fresh e2e install into a
throwaway workspace clean, `board audit` 0/0/0 in the install target.

### Scoped re-review of round 2 — F2 resolved, verdict DONE

**F2: fixed.** Glosses and worked example now state one criterion (the ticket's product), and
`rg -n 'surfaced while|noticed during|operating this|found using' skill/` returns nothing — no
activity framing survives anywhere in the payload. `SKILL.md:161` and `TEMPLATE.md:153,171`
carry the token list with pointer-only parentheticals deferring to rules §1, so nothing
elsewhere contradicts the new definitions. Acceptance test re-run verbatim and green: checks
1–6 pass, six id keepers intact, tokens 1/1/2, `class` table `diff`-identical, no Go code.

One observation surfaced while verifying F2, recorded rather than actioned:

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| N8 | non-blocking | spec-unclear | noted | The five provenance classes now sit on **two different axes**. On `main` all five were source-phrased ("found…", "found…", "spawned from…", "from…", "from…"); F2's fix made `field-use`/`self-host` **product**-phrased while `review`/`audit`/`chat` stayed **source**-phrased. Where both axes fire, no precedence is stated | **Self-evidencing:** T-099, filed by this ticket's own review, has a product that changes the flow's tooling (`self-host` by the new test) *and* was spawned from a review finding (`review` by source). It was classed `review` — correctly, but by instinct, not by a stated rule | Add one clause giving the specific-origin classes precedence: `review`, `audit` and `chat` name a concrete triggering event and win when they apply; `field-use` vs `self-host` decides the rest |

**Why N8 is `noted` and not another round.** The overlap is **not introduced** by this branch —
under the old gloss `self-host` ("found operating this repo's own flow") a review-spawned flow
ticket fitted both classes just as it does now; F2's fix sharpened the axis mismatch without
creating the ambiguity. The ticket's Outcome is about `field-use`-vs-`self-host` assignability by
a foreign reader, and that is met and blind-verified. A reader also has a natural resolution and
reaches it unaided — as this review did with T-099. Per rules §5 it stays in this table with its
evidence, promotable by a later reviewer citing this row.

**Verdict: no blocking findings → `6-done/`.** Three rounds: 4 sites fixed, 1 blocking finding
from round 1 (F1), 1 from round 2 (F2), 8 non-blocking findings dispositioned across the three
passes (6 fixed inline, 2 noted, 1 new ticket → T-099).

**Final disposition summary:** 9 findings across all passes — 2 blocking (F1, F2, both fixed and
verified); 7 non-blocking: 5 *fixed inline* (N1–N5), 2 *noted* (N6, N8), 1 *new ticket* (N7 →
T-099).

`cost: estimated S, actual M — three rounds on an S-sized prose ticket; the four sites were
cheap, but the definitional passage took two attempts to make self-consistent`

**Disposition summary:** 7 findings — 1 blocking (F1 → rework); 6 non-blocking: 5 *fixed inline*
(N1, N2, N3, N4, N5 — all branch-authored prose, applied in the rework pass), 1 *noted* (N6),
1 *new ticket* (N7 → T-099, batched as the single guard-check theme).

`cost: estimated S, actual S`

**Note on F1's severity, since it is the one call that sends this back.** F1 is pre-existing
prose (T-085 authored it), which by rules §5 makes it ineligible for *fixed inline* — the inline
bar is causation, not size. That leaves blocking, `folded` or `noted`. It is blocking because
severity is judged against **this ticket's Outcome**, which names "a body of evidence that exists
only in pickle's own repo" as the thing no reader may be sent to. Shipping T-098 with a live
instance of exactly that, in the same file and section as a site it did fix, makes the
implementation audit *partially met* rather than met. The precedent is this ticket's own: the
fourth site was folded into the branch at pickup on identical reasoning.

**Rework scope is closed at F1 + N1–N5.** Any *further* instance found after this belongs to
T-099, not to another round here — two sweeps have now each missed one, and a third hand sweep is
the wrong instrument. That is what T-099 exists to replace.

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
- 2026-08-13 — implemented. **Four sites fixed, prose only**, in three atomic commits (root-path
  child, history kept): `f03e6e9` the two `review-protocol.md` edits — worked examples now
  self-contained and carrying the `spec-unclear`-not-`docs-gap` contrast that does the
  discriminating work the bare ids never did, and the skeleton's warrant restated as the
  mechanism (a header reinvented per author cannot be counted) with no number; `dafaf0c` the
  `field-use`/`self-host` glosses redefined as *using the flow to ship something else* versus
  *working on the flow itself*, plus the pickup audit's fourth site (the repo-only path
  `skill/resources/TEMPLATE.md` → "this skill's own `resources/TEMPLATE.md`"); `f644f52`
  `AGENTS.md` + `CHANGELOG.md`. **Nothing over-corrected:** the five provenance tokens are
  byte-identical (1/1/2 hits), the eight `class` values `diff` clean against `main`, and all
  **six legitimate id references survive** — four provenance tags and two lines of syntax filler,
  none of which asks the reader to resolve anything. No Go code: the
  `git diff --name-only main...HEAD | grep -E '^(internal|cmd)/|\.go$'` guard prints nothing.
  `just build`/`test`/`lint`/`docs-check` all green. The Outcome was verified from the reader's
  position rather than this repo's — a fresh `./pk install` into a throwaway dir gives a
  workspace whose `.agents/skills/ticket-flow/` matches neither `tickets/6-done/T-[0-9]` nor
  `the corpus|this repo`, with `board audit` clean. `NOTES.md` (Task 5) records the sweep, the
  six deliberate keepers, the n=2 pre-registered trigger and the four `rg` patterns — plus the
  fifth shape that escapes all of them, the repo-only path, which is exactly how site four
  survived refinement
- 2026-08-13 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-13 — IN REVIEW → REWORK: review: 1 blocking (F1 pre-registered-criterion warrant still repo-only); 6 non-blocking dispositioned (5 fixed inline, 1 noted, 1 -> T-099)
- 2026-08-13 — rework round 1: F1 fixed (the class vocabulary's warrant no longer cites
  pickle's own pre-registered criterion); N1–N5 fixed inline verbatim per their suggestions.
  One commit (`eed05ea`), prose only — the no-Go-code guard still prints nothing. Full
  acceptance test re-run and green, including both over-correction guards. Scope held exactly
  at F1 + N1–N5
- 2026-08-13 — REWORK → IN REVIEW: findings fixed
- 2026-08-13 — IN REVIEW → REWORK: scoped re-review: F1 + N1,N2,N4,N5 verified fixed; F2 blocking (N3's fix left the provenance glosses contradicting their own tie-break)
- 2026-08-13 — REWORK → IN REVIEW: F2 fixed
- 2026-08-13 — IN REVIEW → DONE: scoped re-review clean: F2 verified fixed; 9 findings total (2 blocking fixed, 5 fixed inline, 2 noted, 1 -> T-099)
- 2026-08-13 — published with user approval: bookkeeping pushed to `main` first — the §0 pre-push
  gate fired exactly as designed, `origin/main...HEAD` carrying `tickets/` paths because
  `origin/main` was 8 bookkeeping commits behind; pushing the base cleared it and the re-check
  came back empty. Branch then pushed and MR
  [#43](https://github.com/codcod/pickle/pull/43) opened against `main` at `9c5539f`
  (5 commits kept rather than squashed — the root-path default, so both rework rounds stay
  visible; 4 files, +51/-15). CI green: `build-test`, `ci-surface`, `goreleaser-check` all pass.
  `manual-smoke` correctly did **not** run — it is path-filtered to `docs/**`, `snowball.yaml`
  and the build script, none of which this branch touches. PR is MERGEABLE / CLEAN. Awaiting the
  human merge — append the `merged to main (#43, <sha>)` line and run `pickle board sync` once it
  lands, which also clears `board audit`'s standing DONE-without-MERGED warning
