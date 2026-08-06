---
id: T-067
title: docs-check passes on a dead cross-reference: no link/anchor validation anywhere in the docs pipeline
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-067 — docs-check passes on a dead cross-reference: no link/anchor validation anywhere in the docs pipeline

## Description

`just docs-check` (`snowball check`, `justfile:23-24`) renders the manual and discards the
output. Rendering succeeds whether or not a cross-reference resolves, so **the docs gate is
blind to dead links** — the one class of docs defect a build gate is normally trusted to catch.

Measured 2026-08-05 while registering `docs/user-manual/concepts/agent-session-workflow.adoc`.
`<<the-flow>>` was replaced with `<<no-such-anchor-xyz>>` in a page included by
`docs/user-manual.adoc`, and:

| command | result on the injected dead xref |
|---|---|
| `just docs-check` (`snowball check`) | **passes** |
| `asciidoctor --failure-level=WARN -o /dev/null docs/user-manual.adoc` | exit 0, silent |
| `asciidoctor -b docbook -o /dev/null docs/user-manual.adoc` | exit 0, silent |

So neither the configured docs command nor the underlying renderer objects: asciidoctor emits
`[no-such-anchor-xyz]` into the output and moves on. A reviewer following the review protocol's
step 4a (*"broken links/anchors"* is named there explicitly) has no command that answers the
question, and the whole-tree sweep it mandates is currently done by eye.

### Why this is worth a ticket rather than a note

The manual is not small and it is heavily cross-linked: **22 anchors, 14 `<<…>>` targets** at
filing, plus `include::` wiring in `docs/user-manual.adoc` that a page can silently fall out of.
Two adjacent failure modes were both hit within the same hour:

1. `agent-session-workflow.adoc` was committed **unregistered** — no `include::`, so `snowball`
   never built it and its xrefs were never even parsed. `docs-check` passed the whole time.
2. Nothing linked **to** the page once registered; it was reachable only from the TOC.

Both are cheap to detect mechanically and invisible to the current gate. The repo already
guards *payload prose* this way — `TestPayloadDispositionVocabulary` and
`TestPayloadDefersToProjectConfig` (`internal/install/install_test.go`) assert on shipped text —
so a docs-tree guard is consistent with existing machinery, not a new kind of thing.

### Shape of the fix (for refinement)

A ~15-line extraction is enough for the core check, and was used to verify the manual by hand at
filing: collect every anchor definition (`[#id]`, `[[id]]`) and every `<<id>>` / `<<id,text>>`
target across `docs/user-manual.adoc` and its includes, then diff the sets. Refinement should
settle:

- **Where it lives.** A Go test under `internal/` reading `../../docs/` (like the payload guards,
  which reach the repo root via `payloadRoot()`), or a `just` recipe? The Go test runs in `just
  test` and in CI for free; a recipe keeps docs tooling together but needs wiring into
  `docs-check`.
- **How much it checks.** Unresolved xref targets are the core. Candidates to include or reject
  explicitly: orphan pages (an `.adoc` under `docs/user-manual/` with no `include::`), pages with
  no inbound xref (weaker — the TOC is a legitimate entry point), and `link:`/`http` URLs
  (out of scope: needs the network).
- **Auto-generated ids.** Asciidoctor generates section ids when an explicit `[#id]` is absent;
  the checker must not report a reference to one of those as unresolved. At filing every
  `<<…>>` target in this manual resolves to an *explicit* anchor, so the simple version is
  correct today — but that is a property to assert, not to assume.

### Also in scope: `docs-check` does not run in CI at all

Folded in from T-057's pickup gate (finding F11, 2026-08-05). Even a *perfect* anchor checker
wired into `just docs-check` would not run automatically: `.github/workflows/ci.yml` runs
`go vet` / `gofmt` / `go test` / `go build` only, so `snowball check` is a local-only gate that a
contributor (or an agent) can skip by never invoking it. This ticket's fix therefore has two
halves, and the second is what makes the first load-bearing:

1. the anchor/xref check itself (above);
2. **a CI job that runs the docs gate** — which also settles the "where it lives" question above:
   a Go test under `internal/` reaching `../../docs/` runs in CI *for free* today, whereas a
   `just docs-check` extension needs `snowball` installed in the workflow. Refinement should price
   both (a Go test needs no new CI dependency; a `snowball` job also validates the render).

### Soft couplings

- **T-066** — also docs-surface, but a *content* gap (undocumented flags, a dropped command
  cited in the payload). Different theme: that ticket fixes wrong text, this one adds the gate
  that would flag a class of it. No file overlap (T-066 owns `cli-reference.adoc`).
- **T-048** — set up the snowball render/`docs-check` pipeline this ticket extends.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-05 — created (TO DO). source: pickle ticket new
- 2026-08-05 — filed from a docs session, not a review: registering the Agent Session Workflow page turned up that `snowball check` passes on an injected dead xref, and the manual's references had to be verified by an ad-hoc script instead
- 2026-08-05 — patched by T-057's pickup gate (finding F11, disposition `folded`): scope gained the
  CI half — `docs-check` is not wired into `.github/workflows/ci.yml` at all, so the docs gate
  never runs unattended
