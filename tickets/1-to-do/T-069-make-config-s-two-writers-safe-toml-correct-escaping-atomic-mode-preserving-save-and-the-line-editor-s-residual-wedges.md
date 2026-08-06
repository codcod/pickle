---
id: T-069
title: make config's two writers safe: TOML-correct escaping, atomic mode-preserving Save, and the line editor's residual wedges
project: pickle
depends-on: []
spawned-by: [T-043]
impact: medium-high
complexity: medium
cost: M
---

# T-069 — make config's two writers safe: TOML-correct escaping, atomic mode-preserving Save, and the line editor's residual wedges

## Description

**Split out of T-043 at refinement (2026-08-06, D1 confirmed by the user).** T-043 was an epic
merged from T-031 (test-harness defects) and T-012 (coverage + render). Its *test-surface* half
stays in T-043; this ticket takes the half that is **production correctness in
`internal/config`** — three defects in the package's two writers, one of which is reachable from
a documented command and bricks `pickle.toml`.

All line references were re-verified against `main` at `545a4c5` (2026-08-06). `internal/config`
is at **85.6%** coverage.

### 1. `Render`'s `%q` is not TOML — and `project add` can brick the config

`Render` quotes every string field with Go's `%q` (`internal/config/config.go:273,275,282,283,288,291,292,296`
— `payload_version`, both `review_addendum`s, and each child's `name`, `path`,
`build`/`test`/`lint`/`docs`, `ticket_prefix`, `branch_prefix`). `Save` (`:315`) writes that text.

T-012 filed this as "control characters emit `\xNN`, which is invalid TOML". **That headline is
now false** and must not be carried into the fix: the decoder is pinned at
`BurntSushi/toml v1.6.0` (`go.mod:5`), which accepts TOML 1.1's `\xNN`. Measured round-trips of
`name = %q`:

| input | rendered | result |
|---|---|---|
| `a\x01b`, tab, `\n`, `héllo→`, U+0085, `\x7f` | `\x01` / `\t` / `\n` / literal / `\u0085` / `\x7f` | round-trips |
| `a\x07b` (BEL) | `"a\ab"` | **parse error** — `invalid escape '\a'` |
| `a\x0bb` (VT) | `"a\vb"` | **parse error** — `invalid escape '\v'` |
| `a\xffb` (invalid UTF-8) | `"a\xffb"` | **silent mutation** → `aÿb` |

`\a` and `\v` are Go escapes with **no TOML equivalent at any version**; invalid UTF-8 is lossy
because `%q` emits `\xff` and the decoder reads it back as U+00FF.

**It is reachable, not theoretical.** `Validate` only requires a non-empty unique `name`
(`config.go:178-186`) and a path that `Stat`s to a directory; `internal/cli/project.go:62,90`
passes `argv` straight through, as do `--build/--test/--lint/--docs`. Measured on a throwaway
install:

```
$ pk project add $'be\all' sub      # writes  name = "be\all"
$ pk project list
pickle: parse …/pickle.toml: toml: line 23 … invalid escape in string '\a'
$ pk project add $'u\xffb' sub      # writes  name = "u\xffb"  ->  reloads as "uÿb"
```

The first bricks every subsequent command in the project; the second renames a child-project at
exit 0. **Fix both ends (D2):** emit real TOML escapes in `Render` (`\u0007`, `\u000b`, and the
TOML-defined short forms where they exist), *and* reject invalid UTF-8 in `Validate` so a value
that cannot round-trip never reaches the file. Render-only would still accept `uÿb`; validate-only
would leave the exported `Render` wrong for every other caller. `SetPayloadVersionInPlace` quotes
with `%q` for consistency with `Render` — fix both together (T-012's original instruction).

### 2. `Save` is neither atomic nor mode-preserving — and the correct writer already exists

```go
func (c *Config) Save(path string) error {   // config.go:308
	…
	return os.WriteFile(path, []byte(c.Render()), 0o644)   // :315
}
```

Truncate-then-write, so an interrupted `project add|remove` leaves a truncated `pickle.toml`; and
`0o644` is hard-coded, so a `0600` config is silently widened. T-018 added a sibling writer that
does it properly — `writePreservingMode` (`config.go:709-745`): `EvalSymlinks` → `Stat` the mode →
`CreateTemp(dir, "."+base+".*")` → write → `Chmod` → `os.Rename`, with the path-naming error
messages T-026 decision D5 required. T-018 left the asymmetry deliberately rather than widen its
own blast radius.

**Fix:** `return writePreservingMode(path, []byte(c.Render()))` — one line. Reuse
`config_test.go:663` (symlink) and `:697` (unwritable parent) as the test shapes.

### 3. Two residual line-editor wedges — legal files that refuse to upgrade

Both are **safe refusals**, verified: the parse-back gate catches each, so neither corrupts a
file. Each is a legal `pickle.toml` that `pickle upgrade` cannot stamp.

**(a) `advance` does not honour escapes inside a multi-line basic string.** The multi-line branch
(`config.go:505`) is a bare `strings.Index(line[i:], st.multilineDelim)`; only the single-line
`"` branch (`:521-527`) skips `\`-escaped bytes. So a `"""…"""` value containing an escaped
`\"""` closes early and a later `[table]` line is misread as top level. Repro (legal TOML,
decoding to `note = "\"\"\" b\n[x]\n"`):

```toml
note = """
a \""" b
[x]
"""
payload_version = "v1"
```

Traced: line 1 clears `multilineDelim`, line 2 `[x]` is taken as a table header, the key is
inserted **inside the string**, and the gate refuses with
`could not set payload_version (it would end up "v1", not "9.9.9")`.
**Fix:** scan the multi-line branch byte-wise, skipping `\`+next before testing for the
delimiter, **only when the delimiter is a basic-string one** (`"""`) — a literal `'''` string has
no escapes. An escaped backslash before the delimiter must still close the string.

**(b) `usesCRLF` misjudges a file whose last line is unterminated.** It inspects `lines[:len-1]`
(`config.go:659-668`), so `"\r\n#"` (lines `["\r","#"]`) is judged CRLF; `insertPayloadVersion`
then appends the key last with a trailing `\r` and **no `\n`** — raw output
`"\r\n#\npayload_version = \"9.9.9\"\r"` — and the gate refuses with
`would leave the file unparseable (… control characters: '0x0d')`. Same for `"a = 1\r\n# tail"`.
**Fix:** only append `\r` when another line follows the insert point.

`FuzzSetPayloadVersion` (`config_test.go:639`) is seeded from `payloadVersionFixtures` (`:527`,
30 cases); the checked-in corpus at `internal/config/testdata/fuzz/FuzzSetPayloadVersion/` has
**one** entry. Add a fixture per shape above.

### The contract, written down instead of re-found (D3)

T-012's item 7 accumulated ~10 further sub-items across the T-018 review and re-review that
T-043's summary never enumerated. Scoping them all in would be a second epic; leaving them
implicit means every future reviewer re-finds them. So this ticket **states the contract in the
doc comment** of `writePreservingMode` and declines the rest explicitly:

- **In scope:** `fsync` before rename (three lines, and the difference between rename-atomic and
  crash-durable — a crash between write and writeback can leave a zero-length `pickle.toml`).
- **Documented and declined** (each was verified real by the T-018 re-review; none is reachable
  through normal use, and the fix cost is disproportionate): **hardlinks are severed** by
  `os.Rename` (`nlink 2 → 1`, the other name stranded at the old version); **metadata beyond the
  permission bits is discarded** — setuid/setgid/sticky (`2644 → 0644`), **all extended
  attributes** (Finder tags, Spotlight comments, quarantine state), and group ownership, because
  the temp file inherits the *directory's* group; a **read-only `0444` config is silently
  rewritten**, because create-temp+rename needs only a writable directory where `os.WriteFile`
  used to fail. The doc comment must say: *preserves the permission bits and follows symlinks;
  does not preserve hardlink identity, ownership, extended attributes, or mode bits outside
  `Perm()`.* A test asserting the declined behaviour is what stops it being re-litigated.
- **Also declined, with a one-line reason each:** comment orphaning on insert (`:419-423` walks
  back over blank lines but not the `#` block they belong to) — cosmetic; `valueEnd` truncating
  a non-scalar (`:465-468`) — unreachable via `upgrade` because `Load` rejects an array into a
  `string` field, though the function is exported; the unreachable `reflect.DeepEqual` refusal in
  `verifyOnlyPayloadVersion` (`:363-366`, 0 executions in the profile) — accepted as
  defence-in-depth, which T-026's re-review already required the doc comment to say; the
  **wrong-line rewrite** when the target version already matches (a line inside a multi-line
  string that differs only by an escape, `"\u0039.9.9"` vs `"9.9.9"`, passes the value gate) —
  unreachable through `upgrade` only because `Upgrade` short-circuits the no-op case
  (`install.go:153`), so it stays a documented caveat of the exported function.
- **API shape** (a package-level `func(path, version string)` beside `Save`'s method form, leaving
  a live `*Config`'s `PayloadVersion` stale) — **out of scope**, it is T-013 item 7's ground
  (dropping the double load in `runUpgrade`) and changing the signature here would collide.

### Soft couplings

- **T-043** (lineage, `spawned-by`) — the other half of the old epic: the `internal/cli` test
  harness and cli-level coverage. Disjoint files; no ordering required. The cli-level
  `project add` tests live there; the escaping fix and its round-trip test live here.
- **T-013** item 7 — the `runUpgrade` double load; owns the API-shape question declined above.
- **T-026** (landed) — set the parse-back gate, `writePreservingMode`'s error-message rule (D5)
  and the `PayloadVersionStampable` probe this ticket must not regress.
- **T-046** / **T-068** both edit `internal/doctor`, not this package — no collision.

### Why medium-high

Graded above T-043 deliberately (D7): a documented command (`pickle project add`) can render a
project's `pickle.toml` unparseable, and a second input silently renames a registered child. That
outranks test coverage, and `medium-high` is the only way to sort it above T-043, since equal
impact breaks ties by id and this id is higher.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-06 — created (TO DO). source: pickle ticket new
- 2026-08-06 — filed by T-043's refinement split (rules §3/§5 spawn gate, user-confirmed D1): the
  production-correctness half of the old T-031+T-012 epic. Every claim re-verified against
  `545a4c5` first — T-012's "invalid `\xNN`" headline was **disproved** (BurntSushi v1.6.0 accepts
  TOML 1.1 `\xNN`) and replaced with the measured defects (`\a`/`\v`, lossy invalid UTF-8), which
  turned out **reachable end-to-end** via `pickle project add`. D2 (fix render *and* validate),
  D3 (fsync in; hardlinks/xattrs/ownership/read-only documented and declined) and D7
  (medium-high, above T-043) are user-confirmed
