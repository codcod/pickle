# Using the Pi coding agent to develop `pickle`

`pickle` **self-hosts the ticket flow it ships** — its own work is planned and tracked through
that flow. This directory configures the [Pi coding agent](https://pi.dev) to operate it.

> Read [`../AGENTS.md`](../AGENTS.md) first. Pi loads it automatically as project context; it
> is the source of truth for the flow, the commit policy, and the board rule. This file only
> covers the Pi-specific mechanics. The design rationale lives in [`../PLAN.md`](../PLAN.md);
> the live tracker is [`../tickets/BOARD.md`](../tickets/BOARD.md).

## Launch

```sh
npm install -g --ignore-scripts @earendil-works/pi-coding-agent   # once
cd pickle                                                         # the repo root
pi
```

On first run:

1. `/login` for Anthropic (or export `ANTHROPIC_API_KEY` before starting).
2. **Trust the project** when prompted — required before `.pi/` extensions and
   `.agents/skills/` load.
3. Confirm the flow engine loaded: `/skill:ticket-flow`.
4. Sanity-check the build: `just build && ./pickle board audit` (expect 0 errors).

## What Pi picks up automatically

| Item | Source | Mechanism |
|---|---|---|
| Project rules / flow / commit policy | `AGENTS.md` | Native AGENTS.md loading (cwd + ancestors to git root) |
| Ticket-flow engine | `.agents/skills/ticket-flow/` (→ `skill/`) | Native `.agents/skills` discovery + `/skill:ticket-flow` |
| Guardrails | `.pi/extensions/workspace-guardrails.ts` | Auto-loaded after trust; `/reload` to refresh |
| Docs-readability reviewer | `.pi/extensions/docs-readability.ts` | Read-only `docs_readability` tool + `/docs-readability` command (Gemini via Copilot) |
| Skill commands | `.pi/settings.json` (`enableSkillCommands`) | `/skill:name` |

`.claude/skills/ticket-flow` and the `CLAUDE.md` view are for Claude Code / Zed and are ignored
by Pi — Pi reads `AGENTS.md`.

## Resuming work

The durable memory of the flow is the board plus the done tickets (each carries its full
Review + History), **not** the chat session. To pick up:

1. `pickle board audit` (or ask *"audit the board"*) — confirm the board is clean.
2. Read `tickets/BOARD.md` — READY first, then TO DO in priority order.
3. Drive with the flow triggers: *"refine ticket T-NNN"*, *"implement ticket T-NNN"*,
   *"validate ticket T-NNN"*, *"make it a ticket"*.

## Guardrails (`workspace-guardrails.ts`)

Pi has no built-in permission gates, so the non-negotiable rules from `AGENTS.md` are enforced
by a `tool_call` gate:

- **Explicit pathspecs only** — blocks `git add -A`, `git add .`, `git commit -a`. Stage the
  paths you mean (bookkeeping commits use explicit pathspecs).
- **Publish gate** — `git push` / `glab mr create` / `gh pr create` requires an interactive
  confirmation (blocked outright in non-interactive modes). `pickle` is publish-gated: present
  the Conventional Commit message + MR attributes for approval first; merging is the human's.
- **Self-modify guard** — blocks `pickle install|upgrade|uninstall` (or `just … install`) run
  against this repo, which would rewrite/strip this project's own installed skill and markers.
  Test installs go to a throwaway dir (e.g. under `/tmp`) only.

Edit the rules in the extension and run `/reload`.

## Docs-readability reviewer (`docs-readability.ts`)

An **optional** read-only Gemini reviewer for an extra readability pass over a
ticket's changed **AsciiDoc or Markdown** docs during review. It exposes:

- **`docs_readability` tool** — LLM-callable; the flow agent calls it with the
  changed `.adoc`/`.md` files and gets back suggestions (it never edits).
- **`/docs-readability <file.adoc|file.md> [more …]`** — a thin command to run
  the pass manually.

Gemini is reached via your `pi` `/login` provider (GitHub Copilot by default, or
`export GEMINI_API_KEY=…`). The reviewer only *suggests*; you approve and apply
edits with the normal tools. Its system prompt is the shared file
[`../skill/resources/docs-readability.prompt.md`](../skill/resources/docs-readability.prompt.md)
— the same one the OpenCode `docs-readability` subagent uses (`../opencode.jsonc`).
The prompt **ships** in the skill payload, and `pickle install --agent pi`
scaffolds this same reviewer into projects (see `agents/pi/`); this file is the
dev-repo copy.

## Notes

- **No MCP.** Pi doesn't support MCP; the `pickle` flow needs none (it is local markdown +
  the CLI). If a Jira/Confluence bridge is ever wanted, it would be a `.pi/skills/` skill
  (like the workspace's atlassian skill) since Pi can't use an MCP server directly.
- **TLS / corporate proxy "Connection error"** — if Pi retries with `Error: Connection error.`
  while `curl` to the API works, it's almost always Node TLS trust behind an intercepting proxy.
  Fix by making Node use the OS trust store: `export NODE_OPTIONS="--use-system-ca"` (then
  relaunch), or point `NODE_EXTRA_CA_CERTS` at an exported keychain PEM.
