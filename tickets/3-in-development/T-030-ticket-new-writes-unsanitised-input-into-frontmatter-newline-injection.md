---
id: T-030
title: ticket new writes unsanitised input into frontmatter (newline injection)
project: pickle
depends-on: []
spawned-by: [T-024]
impact: medium
complexity: low
cost: S
---

# T-030 — ticket new writes unsanitised input into frontmatter (newline injection)

## Description

`pickle ticket new` interpolates user input straight into the generated frontmatter
(`internal/ticket/ticket.go:293-310` via `internal/cli/ticket.go:119-122`), so a newline in an
argument **injects arbitrary frontmatter keys**. Reproduced against the built binary during the
T-024 review:

```
pickle ticket new "sneaky" --project demo --spawned-by "$(printf 'T-001]\nimpact: critical')"
→ created T-002
---
spawned-by: [T-001]
impact: critical]        ← injected line
impact: medium
---
board audit: 2 tickets, 0 error(s), 0 warning(s)   ← silently clean
```

The result is a corrupted ticket the audit calls clean. The same hole **pre-dates T-024** on the
`title` positional: `ticket new "$(printf 'evil\nproject: nope')"` injects `project: nope`, also
audit-clean. `--spawned-by` merely inherits it.

This is a correctness bug rather than a security hole — the input comes from the operator or
their agent, not an untrusted party — but it silently produces malformed tickets that the
board's own validator endorses, which is exactly what the audit exists to prevent.

### Correction at refinement (2026-07-25): the mechanism, and the real blast radius

Everything above was re-verified against the built binary in a throwaway install, **and one claim is
wrong.** The original text said `ParseFrontmatter` "takes the *first* occurrence of a key, so the
injected `impact` wins over the real one". It does the opposite: `ParseFrontmatter`
(`internal/ticket/ticket.go:105-123`) assigns `fm[key] = …` inside its loop, so **the last
occurrence wins**. Measured:

```
impact: critical]        ← injected
impact: medium           ← Scaffold's real value, later, therefore the one that parses
→ board audit: 2 tickets, 0 error(s), 0 warning(s)
```

That matters for what this ticket is actually fixing. `Scaffold` emits keys in a fixed order (`id`,
`title`, `project`, `depends-on`, `spawned-by`, `impact`, `complexity`, `cost`), and an injected line
always lands immediately *after* the field it was injected through — i.e. **before** the real keys it
could shadow. So injected metadata does **not** take effect, and the grades/project of a
newline-injected ticket are the ones the operator passed. The exception is `id:`, which `Scaffold`
emits first: a title-injected `id:` *does* win, and is caught today —
`ticket new "$(printf 'x\nid: T-999')"` → `ERROR: … frontmatter id T-999 != filename id T-001`.

What is genuinely broken, all four re-measured:

1. **Duplicate keys in frontmatter, audit-clean.** The file is malformed and `pickle board audit`
   endorses it. Cosmetic *today* only because of the last-wins accident above — which is an emergent
   property of a map assignment, not a decision anything anchors. Reorder `Scaffold` or make the
   parse first-wins and this silently becomes wrong metadata. → the audit-side half is now
   **T-033**.
2. **The injected line leaks into the body.** `ticket new "$(printf 'evil\nproject: nope')"` produces
   `# T-003 — evil` followed by a bare `project: nope` line in the document body, below the H1.
3. **The board table row is split across two lines**, which is worse than a wrong cell:
   ```
   | T-003 | evil
   project: nope | medium | medium | M | [] |
   ```
   `pickle board audit` reports **0 errors** on that — the row-presence check still finds the id on
   the first physical line. So a corrupt board passes its own validator.
4. **`--spawned-by "banana"`** → `ERROR: … spawned-by banana does not exist`, framing a malformed
   token as a missing ticket (review finding N3). Confirmed still current.

Also confirmed **not** a vector, so the plan does not touch them: `--project` is safe (the value is
looked up in the registry with `cfg.Project()`, so an injected string fails the lookup and the
command exits before writing), and `--impact/--complexity/--cost` are safe (`ticket.ValidGrade`
rejects anything not in the legal sets). The two unvalidated inputs are exactly the `title`
positional and `--spawned-by`.

Duplicate ids are also still accepted verbatim: `--spawned-by "T-001,T-001"` →
`spawned-by: [T-001, T-001]`, audit-clean.

### Why the fix is obvious

The repo already has the convention. `move.sanitizeReason` (`internal/move/move.go:234-240`)
strips newlines and arrows from a `--reason` "so a reason can never be mis-read by
`ticket.LastHistoryStatus`". Ticket creation needs the same discipline.

### Scope

Input validation at the `pickle ticket new` boundary, in `internal/cli/ticket.go` plus one new
exported helper in `internal/ticket`. Nothing else changes behaviour.

- **Reject** a `title` that is empty/whitespace-only, or contains a newline, a carriage return, or a
  line that trims to exactly `---`, before anything is written (decision 1 below).
- **Validate `--spawned-by` token shape** (`^T-\d+$`), rejecting malformed tokens with a message
  that says they are malformed — which closes the route by which `ticket new` can produce finding
  N3's misleading "does not exist". Note the audit emits that message itself
  (`internal/audit/audit.go:80`), so a malformed id arriving by hand-edit or bad merge still gets
  the misleading wording until **T-027** applies `ticket.ValidID` in the audit's own loops. This
  ticket does not fix N3 outright, and the Description above should not be read as claiming it does.
- **De-duplicate** repeated `--spawned-by` ids, preserving first-seen order (decision 3).
- Tests for each: rejection exits non-zero and writes **nothing** (no ticket file, no board row),
  and `board audit` is clean afterwards.

Explicitly **out of scope**, with owners:

- **Escaping board cells** (a `|` in a title yields `| T-005 | pipe | in title | medium | … |`, an
  8-cell row in a 6-column table, audit-clean — re-measured). That is **T-014**, which already owns
  "escape/replace `|` and collapse newlines in **every** cell". This ticket stops the *newline* at
  the input boundary; T-014 escapes at the *render* boundary. Both are wanted and neither is
  redundant, but do not do T-014's job here — and note that after this ticket a newline can no
  longer reach a cell via `ticket new`, so T-014's newline case must be reproduced through
  `ticket move --reason` or a hand-edit instead.
- **Detecting duplicate frontmatter keys in the audit** → **T-033**, split out at refinement with
  user sign-off.
- **TOML-safe rendering of `project add` input** → **T-012** (its title already says "TOML-safe
  render"). Note T-012 has a *second* overlap the refinement missed: its **item 5
  ("board-row title sanitization")** proposes "escape or reject markdown-breaking characters in the
  title (at minimum `|` and control/newline chars)" — i.e. this ticket lands the reject-newline half
  of it. Add a note there when this lands (mirroring the one already in T-014), and be aware that
  `|`-escaping currently has **two** live owners, T-012 item 5 and T-014 item 2. T-012's **item 4**
  ("cli-level tests for `ticket new`") also partly overlaps Task 3 — say in the finish summary which
  of its cases are now covered so T-012 is not re-refined against a stale gap list.
- **`--reason`** is already sanitised (`move.sanitizeReason`, `internal/move/move.go:234-240`).

### Couplings

`spawned-by: [T-024]` — surfaced by T-024's review (findings N2, N3).

Soft couplings (no `depends-on`, no ordering enforced):

- **T-027** ("audit: flag depends-on self-references") has an explicit "Also worth folding in: id
  **shape** validation" section proposing the same `^T-\d+$` check in the *audit's* loops, and it
  already says "Coordinate with T-030 … the two should share one helper rather than diverge". This
  ticket lands that helper as exported `internal/ticket` API (decision 2), so T-027 imports it. If
  T-027 lands first, the helper must be lifted out of `internal/audit` rather than duplicated here.
- **T-033** ("board audit: flag duplicate frontmatter keys") is this ticket's other half, spawned
  from it. T-030 removes the only known *producer* of duplicate keys; T-033 detects them whatever
  the source. T-030 should land first, but neither blocks the other.
- **T-014** (board cell escaping) — see the out-of-scope note above.
- **T-031** ("harden the internal/cli test harness") — added at the pickup gate, which caught that
  T-031 exists precisely to close harness gaps *"before a second consumer arrives and inherits
  them"*, and **Task 3 below is that second consumer**. Not a blocker; it dictates one habit in
  Task 3 (see its preamble): `captureStdout` leaves `os.Stdout` on a **closed pipe** after it
  returns, so any call to it must be the last statement in its test.
- **T-034** / **T-035** — filed from this ticket's pickup gate (2026-07-25). T-034 is the audit-side
  detection of a malformed board *row* (wrong cell count / split row) plus the insert-point scan it
  breaks; T-035 repairs the one such row already in this repo's board. Both are downstream of the
  same "the validator endorses a malformed artifact" theme and neither blocks this ticket.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child-project `pickle` is this repo itself (`pickle.toml`, path `.`):

```
git checkout main
git checkout -b feat/T-030-validate-ticket-new-input
```

Local WIP commits encouraged. **Never push or open a merge request without explicit user
approval** (child-projects are publish-gated); merging is always the human's.

### Prerequisite gate (hard)

None. `depends-on: []`. T-024's `spawned-by` field is in `6-done/` **and merged** to `main`
(`3c4c131`), and T-029's test harness in `internal/cli/cli_test.go` is merged too (`0b7cd91`) — both
are prerequisites in the "the code you are editing exists" sense only, and both are satisfied on
`main` today. Verify with `git log --oneline -3 main` before branching.

### Confirmed design decisions (do not deviate without asking)

1. **A bad title is rejected, not sanitised.** `ticket new` exits non-zero and writes **nothing** —
   no ticket file, no board row — when the title is empty/whitespace-only, or contains `\n`, `\r`, or
   a line that is exactly `---` after trimming. Rationale (user decision, 2026-07-25): a title is a user-authored label
   that becomes the filename, the H1, and a board cell; silently rewriting it produces a ticket the
   operator did not ask for. This deliberately **diverges from `move.sanitizeReason`**, which
   sanitises — that is the right call for a free-text `--reason` and the wrong one for an identifier.
   Say so in a comment at the new validator, or the next reader will "fix" the inconsistency.

   **Amended at the pickup gate (2026-07-25) — the `---` rule is real but its stated rationale was
   false.** This plan asserted three times that a bare `---` "would truncate the frontmatter". It
   cannot: `Scaffold` always interpolates the title after the literal `title: ` prefix
   (`internal/ticket/ticket.go:297`), and `ParseFrontmatter`'s terminator (`:111`) only matches a
   line trimming to *exactly* `---`. Measured: `ticket new "  ---"` yields `title:   ---`, an ordinary
   key line, and `board audit` is clean. The bare form is unreachable anyway — `ticket new "---"`
   exits 2 at `internal/cli/ticket.go:72` on the `-` prefix guard. **Keep the check** (user decision:
   it rejects a degenerate, unusable title) but comment it as exactly that, **not** as a truncation
   defence — enshrining a threat that does not exist is how this plan got the claim in the first
   place, and the whole point of this decision's comment mandate is to stop a future reader acting on
   a wrong premise. Also: Task 3's `"a\n---\nb"` case is caught by the **newline** rule and never
   reaches the `---` branch, so add a padded, newline-free case (`"  ---  "`) or that branch ships
   untested.
2. **The id-shape validator is exported from `internal/ticket`**, not private to `internal/cli`
   (user decision). It sits beside the existing ticket-shape knowledge (`ParseDepends`, `ValidGrade`,
   `Slugify`, `filenameRE`) so **T-027** can import it for the audit's `depends-on`/`spawned-by`
   loops instead of duplicating the regex.
3. **Duplicate `--spawned-by` ids are de-duplicated silently**, first-seen order preserved (user
   decision). A repeated id carries no extra meaning, so this is normalisation, not policy — and it
   cannot break a legitimate use.
4. **Validation happens before any write.** All checks run before `os.WriteFile` and before
   `board.AddTODORow`, so a rejected invocation leaves the tree byte-identical. This matters because
   `ticket new` is **not atomic** across its two writes (file then board row) — T-014 owns that
   separately; do not fix it here, but do not make it worse by validating between the two.

   **Amended at the pickup gate:** "byte-identical" is not literally true if the `--spawned-by` parse
   stays where it is. The measured order in `runTicketNew` is usage guard `:72-75` → flag parse `:84`
   → `--project` empty `:91` → grade loop `:94-98` → `loadConfig()` `:100` → registry `:104` →
   `os.Stat` `:113` → **`os.MkdirAll` `:116`** → spawned-by parse `:121` → `os.WriteFile` `:122` →
   `board.AddTODORow` `:125`. A rejection at `:121` therefore happens *after* a directory may have
   been created. It is a no-op in any installed tree (`internal/install/install.go:314-316`
   pre-creates every status dir), so the planned tests pass either way — but move the
   `ticket.ParseIDList` call **up beside the grade loop, before `loadConfig()`** (it needs no config)
   so the decision is true as written rather than true by accident.
5. **Grade and project inputs are left alone.** Re-measured at refinement: `--project` is safe (the
   value must match a registered child via `cfg.Project()`) and the three grades are safe
   (`ticket.ValidGrade`). Adding redundant checks there is scope creep.
6. **No change to `Scaffold`, `ParseFrontmatter`, or the audit.** The fix is at the input boundary.
   The parser-side defence is T-033; board-cell escaping is T-014.
7. **Exit code is `exitError` (1), not `exitUsage` (2).** These are rejected *values*, not a
   malformed invocation — consistent with the existing `errf("illegal %s value %q …")` grade check
   in the same function (`internal/cli/ticket.go:94-98`), which returns `exitError`.

### Tasks

#### Task 1 — the exported id-shape helper in `internal/ticket/ticket.go`

1. Add the regex beside the existing package-level ones (`filenameRE`, `fmKeyRE`, `slugStripRE`,
   around `internal/ticket/ticket.go:95-97` and `:249`):

   ```go
   // idRE is the canonical ticket-id shape, the same `T-\d+` filenameRE anchors.
   // Exported through ValidID so internal/cli (creation time) and internal/audit
   // (T-027) share one definition instead of two drifting regexes.
   var idRE = regexp.MustCompile(`^T-\d+$`)
   ```

2. Add the two functions, documented in the style of the neighbouring `ParseDepends`/`ValidGrade`:

   ```go
   // ValidID reports whether s has the canonical ticket-id shape (T-<digits>).
   // It is a *shape* check only — existence is the audit's job (rules §3).
   func ValidID(s string) bool { return idRE.MatchString(s) }

   // ParseIDList parses a comma-separated (or bracketed) id list like ParseDepends,
   // but validates each token's shape and de-duplicates, preserving first-seen order.
   // The error names the offending token so a malformed id is not mis-reported as a
   // missing ticket.
   func ParseIDList(raw string) ([]string, error)
   ```

   Implement `ParseIDList` on top of `ParseDepends` (do not re-tokenise: `ParseDepends` already
   handles the optional brackets, the comma split, the trimming, and empty segments). Then, in one
   pass: reject the first token failing `ValidID` with e.g.
   `fmt.Errorf("%q is not a ticket id (expected T-NNN)", tok)`, and skip tokens already seen.
   Leave `ParseDepends` itself **untouched** — it is used for parsing already-written frontmatter,
   where rejecting is not an option.

3. Tests in `internal/ticket/ticket_test.go` (table-driven, matching the file's existing style):
   `ValidID` accepts `T-1`, `T-001`, `T-9999`; rejects `""`, `banana`, `t-001`, `T-`, `T-001x`,
   `T-001]`, `"T-001\nimpact: critical"`, ` T-001 ` (leading/trailing space is the caller's problem —
   `ParseDepends` trims, `ValidID` does not). `ParseIDList`: `""` → `(nil, nil)`;
   `"T-001,T-002"` → both; `"T-001,T-001"` → `["T-001"]`; `"[T-002, T-001]"` → order preserved as
   `["T-002","T-001"]` (**not** sorted); `"T-001,banana"` → error naming `banana`;
   `"T-001]\nimpact: critical"` → error.

   **Amended at the pickup gate — state the unbalanced-bracket answer explicitly.** Layering on
   `ParseDepends` inherits its bracket tolerance: `ParseDepends("T-001]")` strips the `]` via
   `TrimSuffix` (`internal/ticket/ticket.go:132`), so `ParseIDList("T-001]")` **accepts and
   normalises** it to `["T-001"]` — likewise `"[T-001"`. That is intended (the brackets are the
   frontmatter's own syntax, not part of the id), but the table above only pins
   `ValidID("T-001]") == false`, a *different* function, so an implementer could reasonably write the
   `ParseIDList` case as expecting an error and then "fix" the layering to make it pass. Assert
   `ParseIDList("T-001]") == ["T-001"]` explicitly, with a comment saying why it is not an error.
   The `"T-001]\nimpact: critical"` case above still errors — the newline is what makes it malformed,
   not the bracket.

#### Task 2 — reject an unusable title in `internal/cli/ticket.go`

1. Add a small validator in `internal/cli/ticket.go` (private — it is a CLI input policy, not ticket
   knowledge, so unlike Task 1 it does not belong in `internal/ticket`):

   ```go
   // validateTitle rejects titles that cannot be rendered safely. A newline would
   // inject frontmatter keys (Scaffold interpolates the title into the block) and
   // split the BOARD.md table row; a bare "---" would truncate the frontmatter.
   //
   // Deliberately a rejection, not a sanitisation like move.sanitizeReason: a
   // --reason is free text, but a title becomes the filename, the H1 and a board
   // cell, so quietly rewriting it hands back a ticket nobody asked for (T-030
   // decision 1).
   func validateTitle(title string) error
   ```

   Checks: non-empty after `strings.TrimSpace`; no `\n`; no `\r`; no line equal to `---` after
   trimming (see decision 1's amendment for what that check is and is not for). One error per
   condition, each naming the condition — not a generic "invalid title".

   **Amended at the pickup gate — narrow the "untitled" rationale.** This step justified the
   non-empty check as guarding the `Slugify` → `"untitled"` fallback from producing
   `T-00N-untitled.md`. It only half-does: `""` and `"   "` are indeed caught (measured:
   `ticket new "" --project demo` currently yields `T-002-untitled.md`), but any non-whitespace
   unsluggable title still reaches it — `Slugify("###") == "untitled"`
   (`internal/ticket/ticket_test.go:103`) and `###` passes every check proposed here. State the
   rationale as **whitespace-only**, and accept that `untitled` stays reachable. Do **not** widen into
   a character whitelist — Task 3 item 4 forbids exactly that, correctly.

2. Wire both validators into `runTicketNew`, **before** `loadConfig()` so a bad invocation fails
   without touching the filesystem, and beside the existing grade loop at
   `internal/cli/ticket.go:94-98`:

   ```go
   if err := validateTitle(title); err != nil {
   	return errf("%v", err)
   }
   ```

3. Replace the `ParseDepends` call at `internal/cli/ticket.go:119-121` — **note the corrected range:
   the original plan said `:119-122`, but `:122` is the `os.WriteFile`, and an implementer replacing
   the range literally would delete the write.** Per decision 4's amendment, do not replace it in
   place: move the call **up beside the grade loop, before `loadConfig()`**. The comment there
   ("spawned-by ids are passed through unvalidated … `pickle board audit` is what checks they
   exist") is **the bug's own documentation** — rewrite it, do not leave it:

   ```go
   // Shape-checked and de-duplicated at creation time; whether the ids *exist* is
   // still the audit's job (a forward reference to a not-yet-filed ticket is legal).
   lineage, err := ticket.ParseIDList(*spawnedBy)
   if err != nil {
   	return errf("--spawned-by: %v", err)
   }
   ```

   Note the distinction the new comment must preserve: this validates **shape**, not **existence**.
   `--spawned-by T-999` for a not-yet-created ticket stays accepted-then-flagged-by-audit; that is
   deliberate and unchanged.

4. Update `ticketNewUsage` (`internal/cli/ticket.go:35`) only if the flag's contract is now clearer
   to state; it already reads `[--spawned-by "T-NNN[,T-MMM]"]`, which is accurate, so **prefer no
   change**. Do check `pickle ticket new --help`/`pickle help` output still matches — T-024's review
   found a blocking finding of exactly that kind (help omitting a flag).

#### Task 3 — tests in `internal/cli/cli_test.go`

Use the harness merged with T-029: `newProject(t) string` for an isolated install,
`readTicket(t, root, id) string`, and `captureStdout(t, fn) string` if an assertion needs printed
output. **Do not add a second `TestMain`** (Go permits one per package) and **do not call
`t.Parallel()`** (CWD and `os.Stdout` are process-global). Both constraints re-verified at the pickup
gate, along with `errf` → **stderr** (`internal/cli/cli.go:30-31`) and the absence of any
`captureStderr`.

Two harness facts the gate added, both of which change how these tests must be written:

- **`readTicket` cannot express "nothing was written."** It `t.Fatalf`s when the id is absent
  (`internal/cli/cli_test.go:254`) and only ever looks in `tickets/1-to-do` (`:240`). The
  tree-unchanged assertions below therefore need a direct `os.ReadDir` (or a small `assertNoTicket`
  helper) — not `readTicket`.
- **`captureStdout` leaves `os.Stdout` on a closed pipe after it returns** (measured by T-031:
  `n=0 err=write |1: file already closed`), so anything printed later in the same test silently
  vanishes. Make any `captureStdout` call the **last** statement of its test, or assert exit code +
  tree only. T-031 fixes the helper; this ticket must not inherit the trap.

1. `TestTicketNewRejectsInjectionInTitle` — table of hostile titles:
   `"evil\nproject: nope"`, `"x\nid: T-999"`, `"a\r\nb"`, `"a\n---\nb"`, `"   "`, plus (added at the
   pickup gate) `""` and a padded newline-free `"  ---  "` — the latter is the only case that reaches
   the `---` branch at all, since `"a\n---\nb"` is caught by the newline rule first. For each: `Run`
   returns non-zero, **and the tree is unchanged** — assert `tickets/1-to-do/` contains no new
   `.md` file and `tickets/BOARD.md` is byte-identical to before the call. The tree-unchanged half
   is the load-bearing part: an early `return` that still wrote the board row would pass a
   naive exit-code-only test (this is precisely the weakness T-029 fixed in the neighbouring
   `TestTicketNewSpawnedByUnknownID`).
2. `TestTicketNewRejectsMalformedSpawnedBy` — `--spawned-by "banana"` and
   `--spawned-by "T-001]\nimpact: critical"` → non-zero, tree unchanged, and the message names the
   token (capture stdout/stderr as appropriate; `errf` writes to **stderr** —
   `internal/cli/cli.go:30-31`, confirmed at the pickup gate — and `captureStdout` will **not** see
   it, so either add the stderr twin or assert only the exit code plus the tree, and say which in the
   summary).
3. `TestTicketNewDeduplicatesSpawnedBy` — `--spawned-by "T-001,T-001"` on a tree where `T-001`
   exists → exit 0, the written ticket contains `spawned-by: [T-001]`, and `board audit` is clean.
4. Keep a **positive** case green: a normal title with punctuation that is legal but slug-stripped
   (`"add --flag support (finally)"`) must still succeed — the rejection must not become a
   character whitelist.

### Acceptance test

All commands from the repo root. Steps 4–6 are mutation checks proving the new tests are
load-bearing; **commit the real work first** so `git checkout` restores only the mutation, and revert
each before moving on.

```
# 1. baseline
just test
just lint

# 2. the new tests pass by name
go test ./internal/ticket/ -run 'TestValidID|TestParseIDList' -v
go test ./internal/cli/    -run 'TestTicketNewRejects|TestTicketNewDeduplicates' -v

# 3. end-to-end against the built binary, in a throwaway install (never this repo)
just build
S=$(mktemp -d) && cd "$S" && "$OLDPWD/pickle" install --project demo --path . >/dev/null
"$OLDPWD/pickle" ticket new "real parent" --project demo
"$OLDPWD/pickle" ticket new "$(printf 'evil\nproject: nope')" --project demo   # MUST exit != 0
"$OLDPWD/pickle" ticket new "sneaky" --project demo --spawned-by "$(printf 'T-001]\nimpact: critical')"  # MUST exit != 0
"$OLDPWD/pickle" ticket new "banana" --project demo --spawned-by "banana"      # MUST exit != 0, message says "is not a ticket id"
"$OLDPWD/pickle" ticket new "dupes"  --project demo --spawned-by "T-001,T-001" # MUST exit 0
grep 'spawned-by' tickets/1-to-do/T-002-dupes.md    # → spawned-by: [T-001]
ls tickets/1-to-do/                                  # → exactly T-001 and T-002, nothing else
"$OLDPWD/pickle" board audit                         # → 0 errors, 0 warnings
cd "$OLDPWD" && rm -rf "$S"

# 4. mutation A — title check removed: delete the validateTitle call from runTicketNew
go test ./internal/cli/ -run TestTicketNewRejectsInjectionInTitle   # MUST FAIL
git checkout internal/cli/ticket.go

# 5. mutation B — shape check neutered: make ValidID `return true`
go test ./internal/ticket/ ./internal/cli/ -run 'TestValidID|TestParseIDList|TestTicketNewRejectsMalformedSpawnedBy'  # MUST FAIL
git checkout internal/ticket/ticket.go

# 6. mutation C — de-dup removed: drop the seen-set from ParseIDList
go test ./internal/ticket/ ./internal/cli/ -run 'TestParseIDList|TestTicketNewDeduplicatesSpawnedBy'  # MUST FAIL
git checkout internal/ticket/ticket.go

# 7. the real board is untouched by a full test run (the T-024/T-029 invariant)
git status --short tickets/ > /tmp/t030-before
just test
git status --short tickets/ > /tmp/t030-after
diff /tmp/t030-before /tmp/t030-after    # MUST print nothing

# 8. final state clean, and this repo's own board still validates
just lint
just build && ./pickle board audit       # 0 errors, 0 warnings
```

Expected: steps 1–3, 7, 8 green; steps 4, 5, 6 each fail on exactly the named tests and nothing
else; step 7's `diff` silent.

### Docs update

`ticket new`'s input contract becomes stricter, which **is** user-facing, so this step is not a
no-op:

1. **`README.md`** — the flags are documented at `README.md:271-284`, and the pickup gate found a
   line there that goes **half-stale**: `README.md:282-283` currently reads that the ids are
   "**checked for existence by `board audit`, not at creation**". After this ticket their *shape* is
   checked at creation and their *existence* still is not — amend it to say exactly that, since the
   shape/existence split is the subtlety most likely to be misread. Also state that titles may not
   contain newlines. Do not duplicate the rule in two places; cross-reference (T-019 exists because
   prose already duplicates the command table).
2. **`skill/SKILL.md`** — the *make it a ticket* procedure tells an agent to run
   `pickle ticket new "<title>" --project <name>`. If a rejected title is now possible, one clause
   is warranted; keep it short, and remember this file is the **embedded payload** — editing it
   changes what `pickle install` ships (see `AGENTS.md`).
3. **`skill/resources/tickets-README.md`** — rules **§3** defines the id namespace *and* the
   `spawned-by` field (`skill/resources/tickets-README.md:140-149`, inside
   `## 3. IDs, priority, dependencies, and lineage` at `:103`). **The original plan said §5; that is
   wrong — §5 (`:176`) is "The review loop", and editing it would be an unrelated change.** If §3
   implies ids are free-form, tighten it.
4. No docs *build* is configured for this child (`pickle.toml` has `docs` commented out), so there is
   nothing to compile — say so explicitly in the summary rather than leaving step 4a ambiguous.
5. `pickle help` / `pickle ticket new` usage output must still list every flag (T-024's blocking
   finding was exactly this). Verified already true at the pickup gate — both list `--spawned-by` —
   so this is a don't-regress check, and Task 2 step 4's "prefer no change" stands.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint` clean; all three mutations reverted
   (`git diff` shows only the intended files).
2. State which docs changed (or that none needed to, with the reason) and that no docs build exists.
3. Write a summary: files touched, the mutation results proving each test load-bearing, the exact
   rejection messages a user now sees, and anything deferred to T-033/T-014/T-027.
4. Suggested Conventional Commit message:

   ```
   fix(ticket): reject newline-injecting titles and malformed spawned-by ids (T-030)

   ticket new interpolated the title positional and --spawned-by straight into
   the generated frontmatter, so a newline injected duplicate keys, leaked into
   the document body, and split the BOARD.md table row — all of it audit-clean.
   Titles containing newlines or a bare --- are now rejected before anything is
   written; --spawned-by tokens are shape-checked against T-NNN (fixing a
   malformed id being reported as a missing ticket) and de-duplicated. The
   shape check is exported as ticket.ValidID/ParseIDList so the audit can reuse
   it in T-027.
   ```

   Scope `ticket` is the dominant package; the `internal/cli` edit is the call site.

5. Commit locally on the branch. **Do not push or open a merge request without user approval.**
   Present the message; after approval, finalize (squash or keep history — user's choice), push, open
   the MR. Merging is the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-25 — refined: Description re-verified against the built binary and **corrected** (the
  original "first occurrence wins" claim is wrong — `ParseFrontmatter` is last-wins, so injected
  metadata does not take effect; the real damage is duplicate keys, body leakage, a split BOARD.md
  row, and a misleading error, all audit-clean). Grades re-checked and held at medium/low/S. Four
  decisions taken with the user: reject (not sanitise) bad titles; export the id-shape helper from
  `internal/ticket` for T-027 to reuse; de-duplicate `--spawned-by` silently; split the audit-side
  duplicate-key check out as **T-033**. Board-cell escaping confirmed as T-014's, noted there.
- 2026-07-25 — TO DO → READY: plan complete: 4 decisions confirmed, Description corrected, T-033 split out
- 2026-07-25 — pickup applicability gate (independent sub-agent): **0 blocking**, 7 non-blocking
  amendments, 2 adjacent findings; routing **PROCEED WITH AMENDMENTS**, approved by the user. Bug
  re-confirmed reproducible byte-for-byte against a freshly built `be0ccb5`; both claimed non-vectors
  (`--project`, grades) re-confirmed safe; no existing test breaks. Amendments applied to this plan:
  corrected `:119-122`→`:119-121` (`:122` is the `os.WriteFile`); moved the `ParseIDList` call before
  `loadConfig()` so decision 4's "byte-identical" is literally true rather than true only because
  `os.MkdirAll` is a no-op; **dropped the false `---`-truncates-frontmatter rationale** (measured:
  it cannot) while keeping the check as a degenerate-title guard, plus a case that actually reaches
  that branch; `readTicket` cannot assert "nothing written" (it `t.Fatalf`s on absence) so those
  assertions use `os.ReadDir`; added **T-031** as a soft coupling and the `captureStdout`-last rule;
  pinned `ParseIDList("T-001]")` as accept-and-normalise; fixed docs §5→§3, the `README.md:282-283`
  staleness, the duplicated `sanitizeReason` range, the N3 over-claim, and T-012's item-5 overlap.
  Adjacent findings filed as **T-034** (audit does not check row cell count; the insert-point scan
  breaks on a malformed row) and **T-035** (this repo's own T-021 board row is corrupt) — diagnosed
  while filing: `insertIntoBoard` reads impact as `cells[3]` of a plain `|` split, so the corrupt row
  has rank 0 and every new row piles up in front of it.
- 2026-07-25 — READY → IN DEVELOPMENT: picked up; pickup gate clean (0 blocking, 7 amendments applied)
