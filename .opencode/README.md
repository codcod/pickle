# Using OpenCode to develop `pickle`

This directory plus [`../opencode.jsonc`](../opencode.jsonc) configures
[OpenCode](https://opencode.ai) to operate `pickle`'s **self-hosted brine
flow**. It is dev tooling for building pickle — nothing here is part of the flow
pickle ships (the embedded `skill/` payload).

> Read [`../AGENTS.md`](../AGENTS.md) first. OpenCode loads it automatically as
> project context; it is the source of truth for the flow, the commit policy,
> and the board rule. This file only covers the OpenCode-specific mechanics.

## Launch

```sh
curl -fsSL https://opencode.ai/install | bash   # once (or npm/brew — see docs)
cd pickle                                        # the repo root
opencode
```

**Do not run `/init`.** It scans the repo and (re)generates `AGENTS.md`; this
repo already has a maintained one and `/init` risks rewriting it.

On first run, authenticate with a provider:

- `/connect` for OpenCode Zen (curated models), or log in with **GitHub** to use
  your **Copilot** subscription (also what the docs-readability reviewer uses),
  or with **OpenAI** for **ChatGPT Plus/Pro**;
- or set `ANTHROPIC_API_KEY` for direct Claude access (the tiers brine
  is written against — haiku/sonnet/opus).

## What OpenCode picks up automatically

| Item | Source | Mechanism |
|---|---|---|
| Project rules / flow | `AGENTS.md` | Native `AGENTS.md` discovery (cwd → git root) |
| Brine engine | `.agents/skills/brine/` (→ `skill/`) | **Native** `.agents/skills/<name>/SKILL.md` discovery |
| Guardrails | `opencode.jsonc` (`permission.bash`) | Declarative config, loaded at startup — no plugin code |
| Docs-readability reviewer | `opencode.jsonc` (`agent.docs-readability`) | **Native** read-only AI subagent |

`.claude/skills/brine` is also read (Claude Code compatibility) — same
skill, same name, so this is harmless double discovery, not a conflict.

## Guardrails (`opencode.jsonc` → `permission.bash`)

The non-negotiable rules from `AGENTS.md` (they mirror
`.pi/extensions/workspace-guardrails.ts`) are expressed as declarative glob
patterns:

- **Explicit pathspecs only** — denies `git add -A`, `git add .`, `git commit -a`.
- **Publish gate** — `git push` / `glab mr create` / `gh pr create` always prompt
  for approval (`ask`); pickle is publish-gated.
- **Self-modify guard** — `pickle install|upgrade|uninstall` / `just … install`
  always prompt (`ask`); those would rewrite this repo's own installed skill +
  markers. Test installs go to a throwaway dir (e.g. `/tmp`).

Patterns match the parsed bash command; the **last matching rule wins**, so
`"*": "allow"` must stay first. Edit `opencode.jsonc` and restart OpenCode to
change the rules.

## Docs-readability reviewer

The `docs-readability` subagent (defined in `../opencode.jsonc` under
`agent.docs-readability`) is read-only (`edit`/`bash` denied — it only
*suggests*), pinned to a model reached through your **GitHub Copilot**
login. It reviews **AsciiDoc or Markdown** prose. Its system prompt is the shared file
[`../skill/resources/docs-readability.prompt.md`](../skill/resources/docs-readability.prompt.md)
(shipped in the skill payload; `pickle install --agent opencode` scaffolds this
same reviewer into projects) — the same one the Pi backend
(`.pi/extensions/docs-readability.ts`) uses. Point
it at a ticket's changed `.adoc`/`.md` files for an optional readability pass;
the flow agent then approves and applies the suggestions inline.

```sh
opencode models github-copilot                     # confirm/adjust the model id
opencode agent list                                # confirm it registered
opencode run --agent docs-readability --file <changed.adoc> "…"
```

Because the reviewer is reachable via `opencode run`, the **same reviewer is
usable from Pi, Claude Code, and Zed sessions too** — they just shell out to
`opencode run` (OpenCode needs the one-time Copilot login). Reviewing without it
is a sanctioned conscious skip, recorded in the ticket's `## Review`.

## Custom plugins (optional)

This directory is otherwise empty by design — the guardrails and the reviewer
live in `opencode.jsonc`, so no plugin code is needed. If you ever want to write
a TypeScript OpenCode plugin, `npm install` here (it pulls `@opencode-ai/plugin`)
and add it under `agents/`. `node_modules/`, `package.json`, and lockfiles are
gitignored.

## Known gotcha: possible TLS "Connection error"

OpenCode is Node/Bun-based, so it can hit the same TLS-trust failure as Pi
behind an intercepting proxy (`UNABLE_TO_GET_ISSUER_CERT_LOCALLY`). If a
connection error appears while `curl` to the API works, make the runtime use the
OS trust store (Node generally honours `NODE_EXTRA_CA_CERTS`; confirm the exact
option for Bun via `opencode --help` / the OpenCode docs).

## Isolation reminder

Operate the flow only through tickets and the brine skill. Feature work
happens on `feat/T-NNN-<slug>` branches; the child-project (`pickle`) is
publish-gated (present the commit message + MR attributes for approval before
pushing), while board/ticket bookkeeping may be committed automatically with
explicit pathspecs (`git add <paths>`, never `-A`/`.`).
