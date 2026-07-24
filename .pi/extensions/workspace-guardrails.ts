/**
 * Guardrails for developing `pickle` with Pi.
 *
 * Pi ships no built-in permission gates ("primitives, not features"), so the
 * non-negotiable rules from `pickle/AGENTS.md` are encoded here as a `tool_call`
 * gate. `pickle` is a single, self-hosted repo, so the rules are repo-wide.
 *
 * Rules enforced (see AGENTS.md → Commit policy / Board rule):
 *   1. Explicit pathspecs only — never `git add -A` / `git add .` / `git commit -a`.
 *   2. Publish gate — no `git push` / `glab mr create` / `gh pr create` without
 *      explicit user approval (present the commit message + MR attributes first).
 *   3. Self-modify guard — never run `pickle install|upgrade|uninstall` (or
 *      `just … install`) against this repo; it would rewrite/strip the project's
 *      own installed skill + markers. Test installs go to a throwaway dir (/tmp).
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

// Does this command clearly target a throwaway dir (allowed for test installs)?
function targetsTmp(seg: string, cwd: string): boolean {
  return /\/tmp(\/|\b)/.test(seg) || /(^|\/)tmp(\/|$)/.test(cwd);
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
              "Blocked: never `git add -A` / `git add .` in pickle. " +
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
          "pickle is publish-gated: pushing / opening a merge request requires " +
          "explicit user approval. Present the Conventional Commit message and MR " +
          "attributes first; merging is always the human's (AGENTS.md → Commit policy).";
        if (!ctx.hasUI) return { block: true, reason };
        const ok = await ctx.ui.confirm("pickle publish gate", reason + "\n\nProceed?");
        if (!ok) return { block: true, reason: "User declined the pickle publish gate." };
      }

      // Rule 3 — never self-modify this repo's own install.
      const isSelfModify =
        /\bpickle\s+(?:install|upgrade|uninstall)\b/.test(seg) ||
        /\.\/pickle\s+(?:install|upgrade|uninstall)\b/.test(seg) ||
        /\bjust\b[^\n]*\binstall\b/.test(seg);
      if (isSelfModify && !targetsTmp(seg, ctx.cwd)) {
        const reason =
          "Refusing `pickle install/upgrade/uninstall` (or `just … install`) that does " +
          "not clearly target a throwaway dir. Running it here would rewrite or strip " +
          "pickle's own installed skill + AGENTS.md/CLAUDE.md markers. Test installs go " +
          "to /tmp only (AGENTS.md).";
        if (!ctx.hasUI) return { block: true, reason };
        const ok = await ctx.ui.confirm("Self-modify guard", reason + "\n\nProceed anyway?");
        if (!ok) return { block: true, reason: "Blocked self-modify against the pickle repo." };
      }
    }

    return;
  });
}
