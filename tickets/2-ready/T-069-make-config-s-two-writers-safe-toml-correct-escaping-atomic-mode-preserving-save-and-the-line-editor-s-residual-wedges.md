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
`config_test.go:691` (symlink) and `:727` (unwritable parent) as the test shapes.

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

`FuzzSetPayloadVersion` (`config_test.go:664`) is seeded from `payloadVersionFixtures` (`:550`,
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

All line references re-verified against `main` at `022b857` (2026-08-07); nothing in
`internal/config/config.go` moved since the ticket was filed at `545a4c5`.

**Branch:** `feat/T-069-config-writers-safe` in the `pickle` repo (this repo, `.`), off `main`.

**Prerequisites:** none — `depends-on: []`, no other in-flight ticket touches
`internal/config/`.

**Confirmed decisions** (user sign-off above):

- **D-A.** `Config.Validate()` gains the invalid-UTF-8 reject; `AddProject` calls
  `c.Validate()` after appending the new project (with defaults applied) and rolls the
  append back on failure, so an invalid value never reaches `Save` from *any* caller
  (`project add`, `install.go`'s `writeConfig`) — no `internal/cli` change needed. This also
  makes `AddProject` reject an illegal `ticket_prefix`/WIP the same way, closing the same
  reachability gap for those fields as a side effect.
- **D-B.** The CRLF-insert bug is fixed with a one-line guard in `insertPayloadVersion`
  (`usesCRLF(lines) && insertAt < len(lines)`), not a rewrite of `usesCRLF`.
- **D-C.** One `tomlQuote` helper replaces all 10 `%q` call sites (8 in `Render`, 2 in the
  payload-version rewrite/insert paths). It assumes valid UTF-8 (guaranteed upstream by D-A).
- **D-D.** `writePreservingMode` gets an `fsync` before rename, an updated doc comment
  stating the preserve/decline contract verbatim, and two portable regression tests
  (hardlink severance, read-only-file rewrite) — no root- or xattr-dependent test.

### Tasks

1. **`tomlQuote` helper** (`internal/config/config.go`, new function placed just above
   `Render`, ~line 264): a TOML basic-string quoter over `s`'s runes — `\` and `"` doubled,
   `\b \t \n \f \r` for their control chars, `\u%04x` for every other rune `< 0x20` or
   `== 0x7f`, every other rune written verbatim. Replace every `fmt.Fprintf(&b, "...= %q\n", …)`
   call in `Render` (lines 273, 275, 282, 283, 288, 291, 292, 296) with `tomlQuote(...)`, and
   replace `fmt.Sprintf("%q", version)` in `rewriteFoundKey` (line 635) and
   `insertPayloadVersion` (line 649) the same way.
2. **Reject invalid UTF-8 in `Validate`** (`internal/config/config.go:177`): import
   `unicode/utf8`; inside the per-project loop add a case (alongside the existing empty-name /
   duplicate / path / ticket-prefix / WIP cases) rejecting a project whose `Name`, `Path`,
   `Build`, `Test`, `Lint`, `Docs`, `TicketPrefix`, `BranchPrefix` or `ReviewAddendum` is not
   `utf8.ValidString`; add the same check for the top-level `PayloadVersion` and
   `ReviewAddendum` before the loop. Error text names the project and field (Go's own `%q` on
   an invalid-UTF-8 Go string is fine here — it never reaches the file, only stderr).
3. **Wire the reject into `AddProject`** (`internal/config/config.go:230`): after
   `c.Projects = append(c.Projects, p)`, call `c.Validate()`; on error, pop the just-appended
   project back off (`c.Projects = c.Projects[:len(c.Projects)-1]`) and return the error
   unchanged.
4. **Atomic, mode-preserving `Save`** (`internal/config/config.go:308-315`): replace
   `return os.WriteFile(path, []byte(c.Render()), 0o644)` with
   `return writePreservingMode(path, []byte(c.Render()))`.
5. **fsync before rename** (`internal/config/config.go`, inside `writePreservingMode`,
   ~line 733): after the successful `tmp.Write(data)` and before `tmp.Close()`, call
   `tmp.Sync()`; on error, close the temp file and return
   `fmt.Errorf("%s: writing the update: %w", path, err)` (same shape as the existing write-error
   return).
6. **Rewrite `writePreservingMode`'s doc comment** to state the contract verbatim: *preserves
   the permission bits and follows symlinks; does not preserve hardlink identity, ownership,
   extended attributes, or mode bits outside `Perm()`* — plus a line noting the added `fsync`
   closes the crash-durability gap (rename-atomic vs. crash-durable) while the declined items
   stay declined (hardlinks, xattrs, ownership, setuid/setgid/sticky, silently rewriting a
   `0444` file), each with the one-line reason already drafted in the ticket's Description
   under "The contract, written down instead of re-found (D3)".
7. **Escape-aware multi-line scan in `advance`** (`internal/config/config.go:502-527`): in the
   `st.multilineDelim != ""` branch, keep the current `strings.Index` fast path only for the
   literal delimiter (`'''`, no escapes in TOML); for the basic delimiter (`"""`) scan
   byte-wise from `i`, treating `\` followed by another byte as an escaped pair to skip (a
   trailing `\` at end-of-line is a line-continuation, not an escape, so just advance past it),
   and testing `strings.HasPrefix` at each unescaped position for `"""`. Preserve the existing
   behaviour for an *unescaped* `\` immediately before the delimiter (`\\"""` — an escaped
   backslash — must still close the string; the byte-wise scan handles this because the two
   backslashes are consumed as one escaped pair before the delimiter check runs).
8. **Fix the CRLF-insert guard** (`internal/config/config.go:649`, `insertPayloadVersion`):
   change `if usesCRLF(lines) {` to `if usesCRLF(lines) && insertAt < len(lines) {`. Leave
   `usesCRLF` itself untouched.
9. **Tests — escaping / round-trip** (`internal/config/config_test.go`): extend or add a
   `TestRenderEscaping`-style table covering the Description's round-trip table verbatim —
   `a\x01b`, tab, `\n`, `héllo→`, U+0085, `\x7f`, `a\x07b` (BEL), `a\x0bb` (VT) — asserting
   `Render` output parses back via `toml.Decode` to the original string for each, including
   through a full `Save`+`Load` round-trip for at least one case.
10. **Tests — UTF-8 reject** (`internal/config/config_test.go`): a test constructing a project
    `Name` with an invalid UTF-8 byte (e.g. `"u\xffb"` as a raw byte sequence, matching the
    ticket's repro) and asserting `AddProject` returns an error *and* `c.Projects` is unchanged
    in length (the rollback held) — this is the regression test for the "second input silently
    renames a registered child" defect.
11. **Tests — `Save` atomicity** (`internal/config/config_test.go`): add
    `TestSaveFollowsSymlink` and `TestSaveUnwritableParentNamesTheRealFile`, mirroring
    `TestSetPayloadVersionInPlaceFollowsSymlink` (`:691`) and
    `TestSetPayloadVersionInPlaceUnwritableParentNamesTheRealFile` (`:727`) but calling
    `(*Config).Save` instead.
12. **Tests — `writePreservingMode` declined contract** (`internal/config/config_test.go`):
    `TestWritePreservingModeSeversHardlink` — `os.Link` a second name to the target, call
    `writePreservingMode`, assert the second name's `Stat().Sys()` `Nlink` dropped to 1 (skip
    if the platform can't report `Nlink`); `TestWritePreservingModeRewritesReadOnlyFile` —
    `os.Chmod(0o444)` the target, call `writePreservingMode`, assert it **succeeds** (documenting
    the decline, not regressing it).
13. **Tests — line-editor wedges**: add the two repros from the Description as new entries in
    `payloadVersionFixtures` (`config_test.go:552`) — (a) the escaped-`\"""`-inside-a-multi-line-
    string TOML from item 3(a), expecting `ok: true` and the payload_version line rewritten
    without disturbing `[x]`; (b) `"\r\n#"` (and/or `"a = 1\r\n# tail"`) from item 3(b),
    expecting `ok: true` with no dangling bare `\r`. Both are picked up automatically by
    `FuzzSetPayloadVersion`'s seed corpus (`:664`) and the table-driven test that walks
    `payloadVersionFixtures`; additionally copy the regenerated fuzz corpus entries `go test
    -run FuzzSetPayloadVersion -fuzz FuzzSetPayloadVersion -fuzztime 30s ./internal/config/`
    (or a shorter local run) may add under
    `internal/config/testdata/fuzz/FuzzSetPayloadVersion/` into git.

### Acceptance test

```
cd pickle-repo-root
just build
just test        # go test ./... — must include every test added above, green
just lint
```

Plus the two manual reachability repros from the Description, run against the newly built
binary in a throwaway dir (never the in-repo binary path):

```
D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D" && ./pk install --project-name demo --project-path . -y
./pk project add $'be\all' sub   # was: bricks pickle.toml — must now be rejected with an error, file untouched
./pk project add $'u\xffb' sub   # was: silently renamed at exit 0 — must now be rejected with an error, file untouched
./pk project list                 # must still work: pickle.toml was never corrupted
```

### Docs step

No user-facing behaviour changes what a documented command is *supposed* to do (`project add`
still adds a project; it now also refuses a value that could never round-trip) — no
`docs/user-manual.adoc` update is required. `just docs-check` still runs as part of the
acceptance test to confirm that holds.

### Finish step

Commit locally on `feat/T-069-config-writers-safe` (Conventional Commit, ticket id in brackets,
e.g. `fix(config): TOML-correct escaping, atomic Save, line-editor escape/CRLF fixes (T-069)`).
Do not push or open an MR without explicit user approval (child-project commit policy is
publish-gated). Move the ticket with
`pickle ticket move T-069 in-review --reason "acceptance green"` and hand back for review.

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
- 2026-08-06 — patched by T-043's review impact sweep: T-043 landed its item 3 (the only edit it
  made to `internal/config/config_test.go`), inserting `TestLoadDefaultsZeroWIP` and renaming
  `TestLoadErrors`' “zero wip” case to “negative wip” — what it actually asserts. Every line
  reference in this ticket shifted **+25** and is refreshed above (`:663`→`:691` symlink,
  `:697`→`:727` unwritable parent, `:639`→`:664` `FuzzSetPayloadVersion`, `:527`→`:550`
  `payloadVersionFixtures`). No production code moved, so the plan's substance is unchanged; the
  “do not touch `internal/config` production code” split (D1) held in both directions
- 2026-08-07 — TO DO → READY: plan complete
