# Working in the `pickle` repository

`pickle` is a Go CLI that installs and operates a **ticket-based, board-driven feature flow**
in any project (see [`README.md`](README.md)). This repo **self-hosts that flow**: its own
work is planned and tracked through the very brine flow it ships.

Consequences of self-hosting that the generated block below cannot know, and which
therefore live out here where `pickle upgrade` will not overwrite them:

- The sole registered child-project is **this repo itself**, at the root (`.`) — see
  [`pickle.toml`](pickle.toml). Every ticket's requirements, tasks, and acceptance tests
  describe the `pickle` binary.
- `.agents/skills/brine/` is a symlink to [`skill/`](skill/), which is also the payload
  embedded in the binary. Editing the skill here changes what `pickle install` ships — and it
  is why `upgrade` leaves this repo's skill directory alone instead of replacing it.
- **Self-modify policy.** Never run `pickle install|upgrade|uninstall` against this repo from
  a feature branch: the binary is the artifact under development, and running WIP code against
  the repo's own `AGENTS.md`, `pickle.toml` and skill link is testing in production. Instead:
  - Marker-block changes are made **by hand**, mirroring `install.go`'s `markerBlock()`, inside
    the ticket's diff — so review catches them.
  - **Test installs go to a throwaway dir with the binary copied in, always renamed to
    `pickle-test`** (`D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" &&
    ./pickle-test install …`) — use this exact name every time, so repeated test binaries
    read as the same known tool instead of a new one each session — and never reference the
    repo-built binary by its in-repo path from elsewhere.
  - The one legitimate self-upgrade is a **human action**: after a merge, a clean-built
    `pickle upgrade` from `main` may re-stamp `payload_version` and verify the marker block
    round-trips, reviewing `git diff` before committing.
  - `pickle doctor` is self-host-aware (T-046): it detects the `.agents/skills/brine`
    symlink and skips the `payload_version`-vs-binary comparison entirely, reporting the skip
    as an informational passed line under `-v` instead of a warning. `pickle upgrade` still
    stamps `payload_version` in that mode — it refreshes everything else it owns — so `doctor`
    and `upgrade` agree by construction and `doctor` never sends you to the `upgrade` this
    policy forbids running from a feature branch.
- **The foreign-workspace test** (T-098). Everything under `skill/` is read by projects that are
  not pickle, in workspaces where this repo does not exist. Before shipping a sentence into the
  payload, ask: *would this help a project that is not pickle?* Four shapes fail it — a **ticket
  id the reader is told to go and look up** (`tickets/6-done/T-090` is a well-formed path that
  means something else in their repo), a **count or claim drawn from a corpus they do not have**
  ("13 header variants across the corpus" — whose?), **"this repo" meaning ours** (the
  `field-use`/`self-host` split was once defined as "another project" versus "this repo's own
  flow", which no foreign team can assign), and a **path that only resolves in pickle's own
  source tree** (`skill/resources/TEMPLATE.md` exists here; an installed workspace has
  `.agents/skills/brine/resources/TEMPLATE.md`) — the shape most easily missed, since it is
  neither a ticket id nor a first-person claim. Two uses of a ticket id stay legitimate, because
  neither asks the reader to resolve it: **syntax filler** in a grammar example
  (`board: T-084 ready → in development`) and a **provenance tag** naming which ticket introduced
  a rule ("(T-083)"). The fix is never to sterilise every `T-` in sight — it is to make the
  sentence stand on its own, or to phrase a path relative to the skill the reader is holding
  (`this skill's own resources/TEMPLATE.md`, not `skill/resources/TEMPLATE.md`). This is why the
  test lives here and not in the payload: a rule about how to write the payload, shipped inside
  the payload, is the very defect it warns against. `payload_lint_test.go` (T-099) now enforces
  the mechanical part of this paragraph: a ticket-lookup shape, a repo-only path, and the
  first-person and invisible-evidence phrasings all fail `just test` before they reach review.
  But the test cannot judge what a sentence *means* — it can only match one of its four rules — so
  this paragraph stays the authority for the judgement call the test cannot make.
- **Cite `tickets/NOTES.md` by heading, never by line number** — `NOTES.md § "Rejected outright, so
  they are not re-proposed"`, not `NOTES.md:869-874`. That file is append-only, so line-pinned refs
  happen to resolve today and would break together the moment anything is reordered or compacted.
  Headings survive all three. The reasoning, and the deliberate decision not to restructure the
  file, are in `NOTES.md` § *NOTES.md growth (2026-08-15)*.

<!-- pickle:begin -->
## Brine (start here)

**Start at [`tickets/BOARD.md`](tickets/BOARD.md)** — the generated index of every ticket by
status. No feature is built directly from a chat message or a raw idea — work enters only as a
ticket whose Implementation Plan has met the READY gate. A *review finding* is different: it
earns a **disposition** (rules §5), and most are resolved without a new ticket.

- The flow engine is the **brine skill** at `.agents/skills/brine/`. It holds
  the rules (`resources/tickets-README.md`), the ticket template
  (`resources/TEMPLATE.md`), and the review protocol
  (`resources/review-protocol.md`). Claude Code sees it via `.claude/skills/brine`.
  The directory is pickle-owned — `pickle upgrade` replaces it wholesale, so keep
  hand-written notes outside it.
- Triggers: "make it a ticket", "refine ticket T-NNN", "implement ticket T-NNN", "rework ticket
  T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board".

### Project configuration

- **Build target.** Every ticket targets one registered child-project via `project:`
  frontmatter (`pickle project list`). Registered child-projects: `pickle`.
- **Commands** (each child's, from `pickle.toml`):
  - `pickle`: build `just build` · test `just test` · lint `just lint` · docs `just docs-check`
- **Branch & commit.** Conventional Commits with the **ticket id in brackets at the end of
  the subject** (e.g. `feat(cli): add board audit (T-2)`) for child-project code. Ticket/board
  bookkeeping uses its own `board: T-NNN <verb phrase>` form instead — grammar and scope in
  the rules §0. Branch per child:
  - `pickle`: `feat/T-NNN-<slug>`
- **WIP limits** (per child):
  - `pickle`: `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1
- **Commit policy.** Child-projects are **publish-gated**: local WIP commits are encouraged;
  **no push / no merge request without explicit user approval**; after approval, finalize
  (squash or keep history) + push + open the MR — **merging is always the human's**.
  Overarching bookkeeping (tickets, board, docs) may be committed automatically,
  always with **explicit pathspecs** (`git add <paths>`, never `git add -A`/`.`).
- **Where commits land.** Code goes on the child's feature branch; **ticket and board
  bookkeeping is committed on the base branch**, never on a feature branch — a squash-merge
  folds or drops it and the board then disagrees with the tickets it indexes. This covers a
  review's own moves too, and it is why a reviewer on a feature branch reads the ticket from
  the base branch. `pickle hooks install` enforces it locally, once per clone: a `pre-commit`
  hook refuses the commit, and a `pre-push` hook refuses the push if it still slipped through
  (bypass either with `--no-verify`).

### Board rule

`tickets/BOARD.md` is **generated** — regenerated wholesale from the ticket files by
`pickle ticket new`, `pickle ticket move` and `pickle board sync`. **Never edit it by
hand**; hand-written planning notes go in `tickets/NOTES.md`. Every ticket move = move
the file + one dated `## History` line, and the board regenerates. Prefer
`pickle ticket move` — it does all of it atomically.
<!-- pickle:end -->
