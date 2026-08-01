---
id: T-026
title: upgrade refuses legal pickle.toml files and misdiagnoses why
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M-L
---

# T-026 — upgrade refuses legal pickle.toml files and misdiagnoses why

## Description

Spawned by the **T-018 re-review** (non-blocking finding R6), 2026-07-25.

T-018 replaced `upgrade`'s destructive whole-file re-render with a surgical one-line edit,
guarded by a parse-back gate: the new text is decoded and compared against the original, and
unless `payload_version` came out exactly right with every other value identical, the write is
**refused** and the file left untouched. That is the intended design — the first T-018 review
asked for exactly this, and it is verifiably safe: no input found (60 hand-crafted, 18,636
parseable fuzz cases) corrupts or mis-stamps a `pickle.toml`.

The problem is what the refusal *costs*, and what it *says*.

> **Re-verified end-to-end 2026-08-01 at refinement** (throwaway dir, binary copied out per the
> self-modify policy). All four shapes still reproduce on `v0.2.2-2-ge39e26c`, each from a fixture
> confirmed legal by `tomllib`: every one exits `upgrade` with rc 1 and is then told by `doctor` to
> run the command that just failed. Two corrections to the text below, folded in: the scanner
> anchor **`config.go:388` is stale — the `[`-prefix heuristic is now `config.go:429`**
> (`:388` is today `verifyOnlyPayloadVersion`), and the **quoted-key case now refuses with
> *"would leave the file unparseable (Key 'payload_version' has already been defined)"***, because
> the scanner does not recognise the quoted spelling and inserts a *second* key. True, but it
> names neither the cause nor the line — the same defect as the rest of §2.

**1. Legal configs become permanently un-upgradeable.** The line scanner stops at the first
line that looks like a table header (`internal/config/config.go:429`), so a multi-line string
containing a `[`-leading line hides the real key below it. A review checklist in a
`review_addendum` is enough:

```toml
review_addendum = """
Checklist:
[ ] security
[ ] performance
"""
payload_version = "0.0.1"
```
```
$ pickle upgrade
pickle: …/pickle.toml: could not set payload_version (it would end up "0.0.1",
        not "928164f"); set it by hand                                    rc 1
$ pickle doctor
WARNING: payload version "0.0.1" differs from binary "928164f" — run `pickle upgrade`
```

The file is correctly untouched, but the project can never be upgraded by the tool again, and
`doctor` instructs the user to run the command that just failed. 32% of parseable fuzz inputs
land in this state. Two more legal shapes do the same: a `nan` anywhere in the file (because
`reflect.DeepEqual(NaN, NaN)` is false, so the gate can *never* pass), and the quoted-key
spelling `"payload_version" = "…"`.

**A fourth shape, added 2026-07-25 by the T-018 S1 re-review.** The wedge is not specific to
multi-line *strings*: any construct whose continuation line starts with `[` is taken for a table
header. A multi-line **array** does it, and on the *insert* path — so it bites a legal file whose
`payload_version` was removed by hand:

```toml
matrix = [
[1, 2],
]
```
```
$ pickle upgrade
pickle: …/pickle.toml: setting payload_version would leave the file unparseable
        (…); set it by hand                                               rc 1
```

The key is inserted inside the array literal, the parse-back gate catches it, and the file is
correctly left alone — but the message names neither the array nor the line, and the project is
wedged exactly as in case 1. Worth a fixture whichever remedy is chosen, because it shows the
cause is the `[`-prefix heuristic itself rather than multi-line strings.

**2. The messages misdiagnose.** The `nan` case is refused with *"setting payload_version would
change other values in the file"* — nothing changed and nothing would; the diagnosis is simply
wrong. The table-looking-line case names neither the cause nor the offending line, so the user
cannot act on it. Three failure paths surface raw errnos naming a temp file the user never
created (`rename /…/.pickle.toml.398433048 …: permission denied` under an ACL deny-delete or
`chflags uchg`; `open /…/.pickle.toml.3494394215: permission denied` under a read-only parent).

The two halves are one feature: a refusal is only acceptable if it tells the user which line
defeated the scanner and what to do, and if `doctor` stops recommending a command that cannot
succeed. Options to weigh at refine time — teach the scanner minimal multi-line-string state
(narrows the wedge but never eliminates it); name the offending line in the message; special-case
the NaN comparison; have `doctor` detect the un-upgradeable state and say *"edit `payload_version`
by hand"* instead; offer `upgrade --force-render` as a documented, comment-losing escape hatch.

**Soft couplings — re-anchored 2026-08-01; both original partners were dropped and absorbed.**

- **T-043** (not T-012, which was dropped into it by the 2026-07-26 board triage) now owns the
  in-place writer's other hardening — hardlinks, file metadata beyond perm bits, the unreachable
  whole-tree branch, wrong-line rewrites. Same function, so whoever lands second re-reads the
  other's changes, and the error-message wording there and here must agree. **Boundary confirmed
  at refinement:** the three raw-errno paths in `writePreservingMode` are wrapped *here* (they are
  diagnosis, this ticket's theme); everything else on that list stays T-043's.
- **T-041** (not T-020, dropped into it as the "detect" half) touches the same `doctor` output.
  This ticket adds one warning path to `checkVersion`; T-041 adds marker-drift detection to
  `checkMarkers`. Different functions, same report — no ordering constraint.
- **T-018 is DONE and merged to `main` (`1485242`, 2026-07-25)**, so the prerequisite this ticket
  was filed behind is satisfied. It exists only to refine the writer T-018 introduced.

## Implementation Plan

**Branch.** `feat/T-026-upgrade-refuses-legal-pickle-toml` in the `pickle` child (path `.`, so
this repo), cut from `main`. Code on the branch; `tickets/` bookkeeping on `main` — the split
T-057 exists to guard, and which has been violated three times here.

**Prerequisites.** T-018 done **and merged** (`1485242`) — satisfied. No `depends-on:`.

### Confirmed decisions (user, 2026-08-01)

| # | decision |
|---|---|
| **D1** | Fix the root cause: a **state-aware line scanner** tracking multi-line-string state, array bracket depth and quoted keys. A bounded lexer, *not* a TOML parser. **The parse-back gate stays** — it is T-018's proven safety property and the new scanner's backstop. |
| **D2** | **No `upgrade --force-render`.** A whole-file re-render destroys comments, which is precisely what T-018 was built to prevent. With a message naming the line, the hand fix is one line and lossless. |
| **D3** | When `payload_version`'s **own value** is a multi-line string or an array, keep **refusing** — but name the line and the cause. Those shapes cannot occur in a file pickle wrote; span-replacement logic would be complexity for zero real users. Cases 1–4 and `nan` all become **accepted**. |
| **D4** | `doctor` must stop recommending a command that cannot succeed: probe first, and on a refusal say *edit `payload_version` by hand (line N)*. |
| **D5** | The three raw-errno paths are wrapped **here**; T-043 keeps the rest of the writer hardening. |

### Tasks

1. **Replace the `[`-prefix heuristic** — `internal/config/config.go:419-471`
   (`replacePayloadVersionLine`). Add an unexported line classifier that walks the file once,
   carrying two pieces of state: `inMultiline` (opened by `"""` or `'''`, closed by the matching
   delimiter) and `arrayDepth` (`[`/`]` counted **outside** strings and comments). Per line it must
   skip `#` comments, basic strings (honouring `\` escapes) and literal `'…'` strings before
   counting brackets, so `note = "a [ bracket"` changes nothing.
   - A line counts as a **table header** only when `arrayDepth == 0 && !inMultiline`. This alone
     fixes cases 1 and 4.
   - Recognise the key in three spellings — bare `payload_version`, `"payload_version"`,
     `'payload_version'` — fixing case 3. Keep the existing guard that
     `payload_version_note` is *not* a match (pinned by
     `TestSetPayloadVersionInPlaceKeyPrefixIsNotAMatch`, `config_test.go:401`).
   - Per D3, when the located key's value opens `"""`/`'''` or `[`, refuse — carrying the
     **1-based line number** for task 3.
2. **Make the safety gate NaN-tolerant** — `internal/config/config.go:404`. `reflect.DeepEqual`
   reports `NaN != NaN`, so a legal `nan` anywhere makes the gate **unpassable by construction**
   and produces the flatly wrong *"would change other values"*. Replace with an unexported
   `treeEqual(a, b any) bool` recursing through `map[string]any` and `[]any` and treating two NaN
   float64s as equal; every other comparison keeps `reflect.DeepEqual` semantics.
3. **Make every refusal name the cause and the line** — `verifyOnlyPayloadVersion`
   (`config.go:388-409`) and the new scanner. Each refusal states the 1-based line, what was found,
   and the remedy: *"pickle.toml:12: payload_version's value is a multi-line string, which the
   line editor cannot rewrite safely — set it by hand"*. Keep the `; set it by hand` tail: the
   invariant fixtures match on it.
4. **Stop `doctor` recommending a doomed command** — export a dry-run
   `config.PayloadVersionStampable(path, version) error` (calls `setPayloadVersion` and discards
   the text; writes nothing), then use it in `checkVersion`
   (`internal/doctor/doctor.go:200-207`). Version differs **and** stampable → today's warning,
   unchanged. Version differs **and** refused → warn with the refusal's line and reason plus
   *"set `payload_version` by hand"*, and **never** `run pickle upgrade`. Still a warning, never an
   error (`checkVersion`'s existing contract).
5. **Wrap the three raw-errno paths** (D5) — `writePreservingMode`, `config.go:519-544`:
   `os.CreateTemp`, `os.Chmod` and `os.Rename` currently surface a temp path the user never
   created (`rename /…/.pickle.toml.398433048…: permission denied`). Wrap each with the real
   file's path and the actionable cause (unwritable parent directory, immutable/ACL-locked file),
   keeping the underlying error with `%w`.
6. **Tests** — `internal/config/config_test.go`:
   - Flip the three fixtures the current behaviour pins as refusals (`config_test.go:560-575`) to
     `ok: true`: `quoted key`, `multi-line string with a table-looking line`, `multi-line string
     containing the key`. **`multi-line string containing the key` is the sharp one** — the writer
     must update the real key and leave the identical-looking line *inside* the string untouched;
     assert both.
   - Keep `ok: false`, with the new precise message asserted via `wantErr`: `multi-line value on
     the key itself`, `array value with a space` (D3), `already duplicated key` (input does not
     parse — out of scope).
   - Add fixtures for the two shapes with no coverage: **`nan`** (`threshold = nan` beside a
     normal key — must now be accepted) and the **multi-line array on the insert path**
     (`matrix = [\n[1, 2],\n]` with no `payload_version` — must insert at top level, not inside
     the array).
   - Add a table-header-lookalike *inside a single-line string* (`note = "[not a header]"`) to pin
     that the bracket counter ignores string content.
   - `checkPayloadVersionInvariant` (`config_test.go:443`) and `FuzzSetPayloadVersion`
     (`:618`) need **no change** — that is the point of the invariant, and any regression in the
     new scanner surfaces there rather than in a hand-written expectation.
   - `internal/doctor/`: one test per branch of task 4 — stampable → *run `pickle upgrade`*;
     refused → hand-edit instruction, and assert the string `run \`pickle upgrade\`` is **absent**.
7. **Docs** — `docs/user-manual/configuration.adoc:10-25` is a precise refusal contract that this
   ticket makes **largely false**: it names the quoted key, the table-looking line and the
   key-looking line as beyond the editor, and explains the rule as positional ("the scan stops at
   the key"). Rewrite it to the new contract — the editor now reads multi-line strings and arrays,
   so position no longer decides; what remains beyond it is only a `payload_version` whose *own*
   value is multi-line or an array, and a file that does not parse. Re-check
   `docs/user-manual/cli-reference.adoc:183-186` ("refuses outright if it cannot do so safely")
   — still true, and it defers to `<<configuration>>`, so it likely needs no edit. Update the
   generated header in `Config.Render()` (`config.go:266-271`) only if its wording became untrue.

### Acceptance test

Run from the repo root; every step must pass.

```sh
just build && just test && just lint && just docs-check

# The invariant battery, the fuzz target, and doctor's two branches.
go test ./internal/config/... -run 'PayloadVersion' -count=1
go test ./internal/doctor/... -count=1
go test ./internal/config/... -run FuzzSetPayloadVersion -fuzz FuzzSetPayloadVersion -fuzztime 60s

# End-to-end, in a throwaway dir with the binary copied out (self-modify policy).
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install >/dev/null
sed -i '' 's/^payload_version = .*/payload_version = "0.0.1"/' pickle.toml && cp pickle.toml base.toml
for shape in \
  'review_addendum = """\nChecklist:\n[ ] security\n"""' \
  'threshold = nan' \
  '__QUOTED__' \
  '__ARRAY__'
do :; done   # four fixtures built as in the Description; each must now:
#   1. be legal TOML   (python3 -c "import tomllib;tomllib.load(open('pickle.toml','rb'))")
#   2. `./pk upgrade`  -> rc 0, and pickle.toml carry the binary's version
#   3. `./pk doctor`   -> no payload-version warning
#   4. every other line byte-identical to the fixture (diff -u, one line differing)
```

**Pass condition.** All four Description shapes upgrade cleanly and leave every other byte
untouched; the two D3 shapes still refuse but name their line and cause; `doctor` never prints
`run pickle upgrade` for a file that cannot be stamped; fuzz finds no new failure.

### Finish

`just test && just lint && just docs-check` green; write the ticket summary; prepare the
Conventional Commit (`fix(config): read multi-line TOML when stamping payload_version (T-026)`)
and commit **on the branch, code only**. **Do not push or open an MR without approval**
(publish-gated). Then `pickle ticket move T-026 in-review --reason "acceptance green"`, committing
that bookkeeping on `main`.

## Review

**2026-08-01 — verdict: pass, no blocking findings.** Branch
`feat/T-026-upgrade-refuses-legal-pickle-toml` (`e7d494f` + the review's `4a5d4a5`), reviewed
against `main`.

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — **skipped**: the `docs-readability` subagent is misconfigured in
      this session (`Model not found: github-copilot/gemini-2.5-pro`). Sanctioned conscious skip
      (step 4b); the branch's `.adoc` prose was read by hand instead.
- [x] Findings recorded with severity **and** disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated (step 7) — T-046 patched, T-043 gained one item
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message & MR attributes presented for approval (step 9)

### Implementation audit (step 2)

| plan item | verdict | evidence |
|---|---|---|
| T1 state-aware scanner replaces the `[`-prefix heuristic | **met** | `scanState`/`advance`/`matchKey` in `internal/config/config.go`; all three key spellings; `payload_version_note` guard intact (`TestSetPayloadVersionInPlaceKeyPrefixIsNotAMatch` still green) |
| T2 NaN-tolerant gate | **met** | `treeEqual` replaces `reflect.DeepEqual`; fixture `nan elsewhere in the file` accepted; `a = [nan, 1.0]` and a `nan` nested in `[commit]` both accepted in an ad-hoc probe |
| T3 every refusal names cause + line | **partially met** | the scanner's two refusals do (`line 8: payload_version's value is a multi-line string; set it by hand`, verified end-to-end). `verifyOnlyPayloadVersion`'s two non-wrapping refusals still name no line → finding **R3** |
| T4 `doctor` stops recommending a doomed command | **met** | `PayloadVersionStampable` probe in `checkVersion`; end-to-end the wedged file warns *"…cannot fix it automatically (…line 8…) — edit payload_version by hand"* and never prints ``run `pickle upgrade` ``; `TestCheckVersionDriftUnstampableSuggestsHandEdit` asserts the absence |
| T5 three raw-errno paths wrapped | **met** | `writePreservingMode` wraps CreateTemp/Write/Close/Chmod/Rename against the real path; `TestSetPayloadVersionInPlaceUnwritableParentNamesTheRealFile` pins the lead-with-real-path shape |
| T6 tests | **met** | 3 fixtures flipped to `ok: true`, 5 added; invariant checker and `FuzzSetPayloadVersion` unmodified as intended |
| T7 docs | **met, with corrections** | `configuration.adoc` contract rewritten; `Config.Render()`'s header re-checked and still true. Two prose defects found and fixed inline → **R1**, **R2** |

**Acceptance test, re-run verbatim.** `just build` / `just test` / `just lint` / `just docs-check`
all green. `go test ./internal/config/... -run 'PayloadVersion'` and `go test ./internal/doctor/...`
green. `FuzzSetPayloadVersion -fuzztime 90s` — 9,958,835 execs, no failure. End-to-end in a
throwaway dir with the binary copied out (self-modify policy honoured): all four Description
shapes are legal `tomllib` input, all four now exit `upgrade` rc 0, `doctor` prints no
payload-version warning, and `diff -u` against the fixture shows **exactly one changed line** in
each. The multi-line-array case was additionally re-run on the true *insert* path (key deleted):
the key lands at top level, `matrix` decodes unchanged. The two D3 shapes still refuse, untouched,
naming their line. **Pass condition met.**

**Independent measurement.** Replaying the 1,406-entry fuzz corpus through `setPayloadVersion`:
of 716 parseable inputs, **13 (1.8%) still refuse** — 6 are the D3 array-value shape (by design),
6 are R5, 1 is a dotted `payload_version.0` key. Against the Description's *"32% of parseable fuzz
inputs land in this state"*, the wedge class is closed in substance.

### Findings

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| R1 | non-blocking | fixed inline | `configuration.adoc`'s new contract overstated the fix twice: table-/key-looking lines inside a multi-line string are read correctly *"regardless of where they sit"*, and refusals come *"with an error naming the line"*. R4 is a counter-example to the first; `verifyOnlyPayloadVersion`'s refusals are to the second. | `docs/user-manual/configuration.adoc:14-24` on `e7d494f` | Reworded to promise what the code delivers — bounded scanner, exotic shapes can still defeat it, line named *when known*. Fixed in `4a5d4a5`. |
| R2 | non-blocking | fixed inline | `cli-reference.adoc`'s `doctor` bullet still said a `payload_version` mismatch is *"a warning pointing at `pickle upgrade`"* — prose D4 made false. Task 7 re-checked `cli-reference.adoc:183-186` but not the `doctor` section. | `docs/user-manual/cli-reference.adoc:226-227` on `e7d494f` | Documented the stampability probe and the hand-edit branch. Fixed in `4a5d4a5`. |
| R3 | non-blocking | noted | Task 3 said *"every refusal names the cause and the line — `verifyOnlyPayloadVersion` **and** the new scanner"*. Only the scanner's did. Two gate refusals still name no line: `could not set payload_version (it would end up …)` and `setting payload_version would change other values in the file`. The other two wrap the decoder's error, which carries a line. | `internal/config/config.go:413-419`; reproduced live by the R4 shape | The scanner knows which line it rewrote or inserted; it could hand that number to the gate. Near-unreachable today (1/716 parseable corpus inputs, a dotted `payload_version.0` key) — recorded, not scheduled. |
| R4 | non-blocking | folded → T-043 | Residual wedge: `advance` honours `\` escapes inside a *single-line* basic string but not inside a *multi-line* one, so a `"""…"""` value containing an escaped `\"""` closes the string early. The scanner then reads a later `[table]` line as top level and wedges a legal file — exactly the class T-026 set out to close. Pre-existing (this shape refused on `main` too), so not inline-fixable per rules §5. | Probe: `note = """\na \""" b\n[x]\n"""\npayload_version = "v1"` → `could not set payload_version (it would end up "v1", not "9.9.9")` | ~4 lines: in `advance`'s multi-line branch, skip `\`-escaped bytes when the delimiter is `"""` (not `'''`, which has no escapes). Add a fixture. |
| R5 | non-blocking | folded → T-043 | Residual wedge on the **insert** path: `usesCRLF` inspects `lines[:len-1]`, so a consistently-CRLF file whose last line is unterminated is judged CRLF, and the appended entry gets a trailing lone `\r` with no `\n` — unparseable, so the gate refuses a legal file. 6 of 716 corpus inputs (e.g. `"\r\n#"`). Pre-existing (`insertPayloadVersion`/`usesCRLF` unchanged by this branch). | corpus replay, above | Only append `\r` when the insert point is followed by another line, or normalise the file's final terminator. |
| R6 | non-blocking | noted | `insertPayloadVersion` returns `(string, error)` but can never fail; the `error` is dead in every caller. Cosmetic, and a plausible future refusal point. | `internal/config/config.go:645-655` | Leave as-is unless the signature stops earning its keep. |

**Disposition summary — 6 findings, 0 blocking:** 2 *fixed inline* (R1, R2 — both docs, in
`4a5d4a5`), 2 *folded* into **T-043** (R4, R5 — one added item, "residual `payload_version`
line-editor wedges"), 2 *noted* (R3, R6). **No new ticket minted.**

### Quality / consistency notes (no finding)

- The scanner was probed with 29 hand-built shapes beyond the fixtures — `'''` literal
  multi-line strings, `[[project]]` inside a multi-line string, nested multi-line arrays,
  brackets and apostrophes inside single-line strings, escaped quotes, inline tables, datetime
  and `inf` values, CRLF files, indented and both quoted key spellings, a `"""x""""` close. All
  28 legal ones round-trip correctly; only R4 fails.
- The parse-back gate is unchanged in role and remains the backstop for exactly this class —
  every residual defect above surfaces as a *safe refusal*, never as a corrupted file. That is
  T-018's property held intact.
- Branch discipline (T-057) held: the feature branch carries code and docs only; all `tickets/`
  bookkeeping is on `main`.

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-25 — broadened by the T-018 S1 re-review: fourth wedging shape (multi-line array, insert path); re-anchored config.go:382 -> :388
- 2026-08-01 — refined: all four wedge shapes re-reproduced end-to-end on v0.2.2-2-ge39e26c; re-anchored config.go:388 -> :429 (scanner) and corrected the quoted-key refusal message; soft couplings re-pointed T-012 -> T-043 and T-020 -> T-041 (both dropped/absorbed); T-018 confirmed merged (1485242); D1-D5 confirmed with user; cost M -> M-L (7 tasks, adds a bounded TOML lexer, a doctor probe and a docs-contract rewrite)
- 2026-08-01 — TO DO → READY: plan complete; D1-D5 confirmed with user; all four shapes re-reproduced at refinement
- 2026-08-01 — READY → IN DEVELOPMENT: picked up
- 2026-08-01 — IN DEVELOPMENT → IN REVIEW: acceptance green: all four wedge shapes upgrade cleanly end-to-end; go test ./... and 3M+ fuzz execs pass; just build/test/lint/docs-check green
- 2026-08-01 — IN REVIEW → DONE: review passed: 6 findings, 0 blocking — 2 fixed inline (docs, 4a5d4a5), 2 folded into T-043 (item 8), 2 noted; no new ticket
