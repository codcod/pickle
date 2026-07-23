# Working in the `pickle` repository

`pickle` is a Go CLI that installs and operates a **ticket-based, board-driven feature flow**
in any project (see [`README.md`](README.md)). This repo **self-hosts that flow**: its own
work is planned and tracked through the very ticket flow it ships.

<!-- pickle:begin -->
## Ticket flow (start here)

**Start at [`tickets/BOARD.md`](tickets/BOARD.md)** — the live index of every ticket by status.
Nothing is built directly from a chat message, an idea, or a review finding — only from a
ticket whose Implementation Plan has met the READY gate.

- The flow engine is the **ticket-flow skill** at `.agents/skills/ticket-flow/` (a symlink to
  this repo's [`skill/`](skill/), which is also the payload embedded in the binary). It holds
  the rules (`resources/tickets-README.md`), the ticket template (`resources/TEMPLATE.md`), and
  the review protocol (`resources/review-protocol.md`). Claude Code sees it via
  `.claude/skills/ticket-flow`.
- Triggers: "make it a ticket", "refine ticket T-NNN", "implement ticket T-NNN", "rework ticket
  T-NNN", "validate ticket T-NNN" (or "review ticket T-NNN"), "audit the board".

### Project configuration

- **Build target.** Every ticket targets one registered child-project via `project:`
  frontmatter. This repo has a single child, **`pickle`**, at the repo root (`.`) — see
  [`pickle.toml`](pickle.toml). Each ticket's requirements, tasks, and acceptance tests describe
  the `pickle` binary itself.
- **Commands** (the child's, from `pickle.toml`): `just build`, `just test`, `just lint`.
- **Branch & commit.** `feat/T-NNN-<slug>` cut in this repo; Conventional Commits with the
  **ticket id in brackets at the end of the subject** (e.g. `feat(cli): add board audit (T-2)`).
- **Commit policy.** The child-project (`pickle`) is **publish-gated**: local WIP commits are
  encouraged; **no push / no merge request without explicit user approval**; after approval,
  finalize (squash or keep history) + push + open the MR — **merging is always the human's**.
  Overarching bookkeeping (tickets, board, docs) may be committed automatically, always with
  **explicit pathspecs** (`git add <paths>`, never `git add -A`/`.`).
- **WIP limits (per child):** `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1.

### Board rule

Every ticket move = move the file + one dated `## History` line + one `tickets/BOARD.md` edit,
in the same change. A move that doesn't touch the board is a bug. Prefer `pickle ticket move`
once it exists; until then, do the three edits by hand.
<!-- pickle:end -->
