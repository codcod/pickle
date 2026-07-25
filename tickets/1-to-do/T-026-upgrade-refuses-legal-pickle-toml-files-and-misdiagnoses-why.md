---
id: T-026
title: upgrade refuses legal pickle.toml files and misdiagnoses why
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
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

**1. Legal configs become permanently un-upgradeable.** The line scanner stops at the first
line that looks like a table header (`internal/config/config.go:388`), so a multi-line string
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

**Soft couplings.** **T-012** item 7 owns the in-place writer's other hardening (hardlinks, file
metadata, the unreachable whole-tree branch, wrong-line rewrites) — same function, so whoever
lands second re-reads the other's changes; the error-message wording there and here must agree.
**T-020** (doctor marker-block drift) touches the same `doctor` output. **T-018** must be merged
first, since this ticket only exists to refine the writer it introduced.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-25 — broadened by the T-018 S1 re-review: fourth wedging shape (multi-line array, insert path); re-anchored config.go:382 -> :388
