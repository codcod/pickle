# `tickets/` — pickle's self-hosted feature flow

This is the live board for building **`pickle`** itself. Every change flows through one
artifact per feature: a **ticket** — a markdown file whose status is the directory it lives in,
targeting a registered child-project via `project:` frontmatter, with an append-only History.

- **Start at [`BOARD.md`](BOARD.md)** — the maintained index of every ticket by status.
- **The rules, ticket template, and review protocol live in the ticket-flow skill**, not here:
  - rules: `.agents/skills/ticket-flow/resources/tickets-README.md` (so `§N` references resolve
    there)
  - template: `.agents/skills/ticket-flow/resources/TEMPLATE.md`
  - review protocol: `.agents/skills/ticket-flow/resources/review-protocol.md`
  (`.agents/skills/ticket-flow` is a symlink to the repo's `skill/`, the embedded payload.)
- **Build target:** every ticket targets the sole registered child-project **`pickle`** (the
  repo root; see `../pickle.toml`). Tickets describe the `pickle` binary.

This directory holds **instance data only** (the board + the tickets). See `../AGENTS.md` for
the project configuration and commit policy.
