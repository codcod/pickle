---
id: T-012
title: harden test coverage + TOML-safe render (config, project, board audit)
project: pickle
depends-on: [T-001, T-002, T-003]
impact: medium
complexity: low
cost: S-M
---

# T-012 — harden test coverage + TOML-safe render (config, project, board audit)

## Description

Non-blocking robustness follow-up surfaced by the T-001 review. T-001 shipped the
`internal/config` package and the `pickle project add|list|remove` commands with strong config
coverage (91.8%) but thinner command-layer coverage and a couple of edge gaps. This ticket
closes those without changing user-facing behaviour.

Scope (cohesive test-hardening + render items). Item 1's `board audit` half was added by the
T-002 review; items 4 and 5 by the T-003 review (which is why this now also depends on T-003):

1. **cli-level tests for `project add|list|remove` and `board audit`** — `internal/cli` sits at
   ~29.5% coverage; the `project` commands and `runBoardAudit` are only exercised by manual
   acceptance tests. Add table-driven tests that drive `runProject*` and `runBoardAudit` against
   a temp overarching root (temp `pickle.toml` + child dirs / tickets tree), asserting: `add`
   appends with defaults and rejects duplicate-name / missing-dir; `list` output; `remove`
   succeeds and the live-ticket remove-guard refuses when a `tickets/…/T-*.md` targets the
   child; `board audit` exits 0 on a clean tree and non-zero on a broken one.
2. **TOML-safe rendering** — `config.Render` currently formats string values with Go `%q`,
   which is not identical to TOML basic-string escaping (control characters / certain runes
   would emit `\xNN`, which is invalid TOML) and would break round-trip for exotic values.
   Escape per the TOML basic-string rules (or route values through the encoder), and add a
   round-trip test with an awkward value (e.g. a tab or non-ASCII rune in a command string).
3. **defaulting test** — the existing `config_test.go` "zero wip" case actually asserts `-1`,
   not `0`. Rename it (e.g. "negative wip") and add a case proving an **omitted / `0`** WIP
   field defaults to 1 rather than erroring.

4. **cli-level tests for `ticket new`** — `runTicketNew` is only exercised by manual acceptance
   tests. Add table-driven tests against a temp overarching root asserting: a fresh id is
   allocated (`max+1`), the scaffold file is written to `1-to-do/` and passes
   `audit.Audit` with zero errors, the board row lands under the child's sub-group in impact
   order, and the failure modes exit non-zero (unregistered `--project`, illegal grade,
   missing title).
5. **board-row title sanitization** — `ticket new` writes the raw title into both the board
   row (`| T-NNN | <title> | … |`) and the `# T-NNN — <title>` heading. A title containing a
   pipe (`|`) or newline corrupts the markdown table (extra columns) and the heading; today
   `board audit` still passes because it only parses the id. Escape or reject markdown-breaking
   characters in the title (at minimum `|` and control/newline chars) in `board.AddTODORow`
   and/or `ticket.Scaffold`, with a test proving a piped title round-trips to a well-formed
   single-cell board row.

6. **`LastHistoryStatus` transition parsing** — the parser locates the status transition with
   `LastIndex(body, "→")`, so a History **reason** clause (after the `:`) that itself contains a
   `→` is mis-parsed as the transition (surfaced while dogfooding the T-003 review: a
   `… → DONE: … 2 non-blocking → T-012` line parsed to `T-012`, failing `board audit`). Fix:
   isolate the transition by splitting on the first `:` **before** finding the arrow (the
   `OLD → NEW` transition always precedes any reason text). Add a test with an arrow in the
   reason clause.

7. **`Save` is neither atomic nor mode-preserving** (added 2026-07-25 by T-018, gate finding
   N2). `config.Save` is a bare `os.WriteFile(path, …, 0o644)` (`internal/config/config.go`), so
   an interrupted `project add|remove` can leave a truncated `pickle.toml`, and any non-default
   permission on the file is silently reset. T-018 added a sibling writer,
   `SetPayloadVersionInPlace`, which *does* write atomically (temp file + `os.Rename`) via the
   unexported `writePreservingMode` helper in the same package, and preserves the original mode.
   Route `Save` through that same helper so the two writers behave alike; T-018 left the
   asymmetry deliberately rather than widen its own blast radius. Note item 2 (TOML-safe
   escaping) also applies to `SetPayloadVersionInPlace`, which quotes with `%q` for consistency
   with `Render` — fix both together.

   **Extended 2026-07-25 by the T-018 review (non-blocking finding 10).** The in-place writer
   T-018 added needs the same hardening pass, in the same file:
   - ~~**CRLF asymmetry** — the replace path re-appends `\r`, the insert path does not, so
     inserting into a CRLF file yields mixed line endings.~~ **STRUCK 2026-07-25 by the T-018
     re-review: fixed under T-018** (`usesCRLF`, `config.go:434`), because
     `FuzzSetPayloadVersion` found it and the invariant could not be left red. A mixed-EOL file
     still gets an LF-only inserted line, which is deliberate and documented.
   - **Comment orphaning on insert** — `:419-423` walks back over blank lines but not over the
     contiguous `#` block those blanks belong to, so the inserted key steals the comment that
     documented the following table.
   - **`valueEnd` on non-scalars** — the bare-value branch (`:465-468`) stops at the first
     space/`#`, so an array or inline-table value is truncated mid-expression. Unreachable via
     `upgrade` (`Load` rejects it into a `string` field) but the function is exported.
   - **No `fsync`** — `:486-502` is rename-atomic but not crash-durable; a crash between write
     and writeback can leave a zero-length `pickle.toml`.
   - **Error messages name the temp file** — a read-only directory yields
     `open /…/.pickle.toml.2836514732: permission denied`, which tells the user nothing.
   - **A read-only (`0444`) `pickle.toml` is now silently rewritten**, because create-temp+rename
     only needs a writable directory; `os.WriteFile` used to fail. Decide and document.
   - **API shape** — a package-level `func(path, version string)` sits oddly beside `Save`'s
     method form and leaves a live `*Config`'s `PayloadVersion` stale after the call. Consider
     `func (c *Config) UpdatePayloadVersion(version string) error`. Note T-013 item 7 (drop the
     double load in `runUpgrade`) is the exact place that trap would spring.

   **Extended again 2026-07-25 by the T-018 re-review (non-blocking findings R4, R5, R7).** The
   rework closed the symlink case; these are the parts `filepath.EvalSymlinks` cannot reach:
   - **Hardlinks are severed at rc 0.** `os.Rename` unlinks the old inode, so a hardlinked
     `pickle.toml` silently diverges — verified `nlink 2 → 1`, `pickle.toml` stamped, the other
     name stranded at the old version, exit 0. The first T-018 review named this inside finding
     4's evidence ("hardlinks break the same way"); the prescribed `EvalSymlinks` fix cannot
     address it, since a hardlink has no target to resolve. The doc comment at `:473-477`
     justifies following symlinks as preserving the user's file — the same reasoning applies
     here and is not honoured. Fix by writing through the existing inode (open + truncate +
     write, or copy-on-write into the temp then `os.Rename` only when `nlink == 1`), or detect
     `nlink > 1` and refuse with an actionable message.
   - **File metadata beyond the permission bits is discarded.** `writePreservingMode` copies only
     `fi.Mode().Perm()` (0777 mask) onto a fresh temp file, so every successful write loses:
     setuid/setgid/sticky (`2644 → 0644`, `1644 → 0644`, `4644 → 0644`, all at rc 0), **all
     extended attributes** (verified: `com.apple.metadata:kMDItemComment` and a custom xattr both
     gone — Finder tags, Spotlight comments and quarantine state with them), and **group
     ownership**, because the temp file inherits the *directory's* group (verified
     `_cyberarkepm_sudoers → staff`). By the same mechanism a `sudo pickle upgrade` would leave
     the config root-owned. Decide the contract and document it; at minimum preserve the full
     `Mode()` and the owning uid/gid.
   - **The whole-tree comparison is unreachable.** `verifyOnlyPayloadVersion`'s
     `reflect.DeepEqual` refusal (`config.go:363-366`) has **0 executions** in the coverage
     profile and no input reaches it: every shape that would change another value fails the
     parse check or the version check first. Either construct the input that reaches it or accept
     it as defence-in-depth against future scanner changes — the T-018 re-review (R3) already
     required the doc comment to stop claiming more than that.
   - **The scanner can still rewrite the wrong line** when the target version already matches. A
     line inside a multi-line string that looks like the key is matched and normalised; if it
     differs only by an escape (`"\u0039.9.9"` vs `"9.9.9"`) the decoded trees compare equal, so
     the value gate passes and user content is silently rewritten at rc 0. Unreachable through
     `pickle upgrade` today only because `Upgrade` short-circuits the no-op case
     (`install.go:153`) — it is reachable by any other caller of the exported function, and it is
     the counter-example to "one line, and it is the right line".

Realistic inputs today (commands, relative paths, names, ordinary titles, History reasons
without stray arrows) are unaffected — this is hardening, hence non-blocking.

## Implementation Plan

<!-- empty until refined; must meet the READY gate (skill rules §4) before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: T-001 review (non-blocking findings N1–N3)
- 2026-07-23 — broadened by the T-002 review: added board-audit cli tests + audit path coverage (N1/N2); depends-on now [T-001, T-002]
- 2026-07-23 — broadened by the T-003 review: added ticket new cli tests + board-row title sanitization (items 4–5); depends-on now [T-001, T-002, T-003]
- 2026-07-23 — broadened again (T-003 review, dogfooding): added LastHistoryStatus transition-parsing fix (item 6)
- 2026-07-25 — broadened by the T-018 gate + review: added `Save` atomicity/mode-preservation and the in-place writer's hardening pass (item 7)
- 2026-07-25 — broadened again by the T-018 re-review (R4/R5/R7): hardlinks, file metadata beyond perm bits, the unreachable whole-tree branch, wrong-line rewrites; struck item 7's CRLF-on-insert sub-item (fixed under T-018)
- 2026-07-25 — re-anchored by the T-018 S1 re-review: item 7 line refs (:319-323 -> :419-423, :348-351 -> :465-468, :372-378 -> :486-502, :467-471 -> :473-477, config.go:428 -> :434, :357-360 -> :363-366)
