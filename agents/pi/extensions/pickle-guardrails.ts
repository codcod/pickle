/**
 * pickle-guardrails — Pi extension scaffolded by `pickle install --agent pi`.
 *
 * Pi ships no built-in permission gates ("primitives, not features"), so
 * brine's non-negotiable git rules (see the AGENTS.md marker block →
 * Commit policy) are encoded here as a `tool_call` gate:
 *
 *   1. Explicit pathspecs only — never `git add -A` / `git add .` / `git commit -a`.
 *   2. Publish gate — no `git push` / `glab mr create` / `gh pr create` without
 *      explicit user approval (present the commit message + MR attributes first);
 *      merging is always the human's.
 *
 * The marker block's remaining git rule — *where commits land*: code on the
 * feature branch, ticket/board bookkeeping on the base branch of whichever
 * repository holds `tickets/` (mandatory in practice under `layout = "in-tree"`;
 * under the default `umbrella` layout it binds the overarching project's own
 * repository, not any child's) — is deliberately NOT mirrored here. It is
 * enforced by `pickle hooks install`, a `pre-commit` hook that guards every
 * committer (agent, human or script) rather than only a pi session, and that
 * reads each child's `branch_prefix` from `pickle.toml`.
 *
 * This file is pickle-owned: `pickle upgrade` refreshes it in place and
 * `pickle uninstall` removes it. Put project-specific rules in a SEPARATE
 * extension file next to this one, not in here.
 *
 * Install: auto-discovered from `.pi/extensions/` once the project is trusted.
 * Hot-reload after edits with `/reload`.
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { isToolCallEventType } from "@earendil-works/pi-coding-agent";

// Split a compound shell command into inspectable segments.
function segments(command: string): string[] {
  return command
    .split(/&&|\|\||;|\n/)
    .map((s) => s.trim())
    .filter(Boolean);
}

// Extract the pathspec/flag arguments after `git add`/`git stage`.
function gitStageArgs(seg: string): string[] {
  const m = seg.match(/\bgit\s+(?:add|stage)\b(.*)$/s);
  if (!m) return [];
  return m[1]
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((a) => a.replace(/^\.\//, "")); // normalize leading ./
}

type Block = { block: true; reason: string };

export default function (pi: ExtensionAPI) {
  pi.on("session_start", (_event, ctx) => {
    if (ctx.hasUI) {
      ctx.ui.notify("pickle guardrails active", "info");
    }
  });

  pi.on("tool_call", async (event, ctx): Promise<Block | undefined> => {
    if (!isToolCallEventType("bash", event)) return;
    const raw = event.input.command ?? "";
    if (!raw.trim()) return;

    for (const seg of segments(raw)) {
      // Rule 1 — git staging discipline.
      if (/\bgit\s+(?:add|stage)\b/.test(seg)) {
        const args = gitStageArgs(seg);
        if (args.some((a) => a === "-A" || a === "--all" || a === ".")) {
          return {
            block: true,
            reason:
              "Blocked: never `git add -A` / `git add .` in this project. " +
              "Stage with explicit pathspecs (`git add <paths>`).",
          };
        }
      }
      if (/\bgit\s+commit\b/.test(seg) && /\s(-a|--all)\b/.test(seg)) {
        return {
          block: true,
          reason:
            "Blocked: `git commit -a/--all` bypasses explicit pathspecs. " +
            "Stage explicit paths first, then commit.",
        };
      }

      // Rule 2 — publish gate (push / merge request).
      const isPublish =
        /\bgit\s+push\b/.test(seg) ||
        /\bglab\s+mr\s+create\b/.test(seg) ||
        /\bgh\s+pr\s+create\b/.test(seg);
      if (isPublish) {
        const reason =
          "Child-projects are publish-gated: pushing / opening a merge request requires " +
          "explicit user approval. Present the Conventional Commit message and MR " +
          "attributes first; merging is always the human's (AGENTS.md → Commit policy).";
        if (!ctx.hasUI) return { block: true, reason };
        const ok = await ctx.ui.confirm("Publish gate", reason + "\n\nProceed?");
        if (!ok) return { block: true, reason: "User declined the publish gate." };
      }
    }

    return;
  });
}
