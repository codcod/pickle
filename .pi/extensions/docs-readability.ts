/**
 * docs-readability — Pi extension scaffolded by `pickle install --agent pi`:
 * an optional AsciiDoc/Markdown readability reviewer for ticket-flow reviews
 * (review protocol Step 4b).
 *
 * Exposes an AI readability reviewer two ways, sharing one core helper:
 *   - `docs_readability` tool  — LLM-callable, used by the flow agent mid-review.
 *   - `/docs-readability`      — thin command to run the pass manually.
 *
 * The reviewer is reached through your `pi` `/login` provider — GitHub Copilot
 * by default (same subscription as the opencode backend). Override via
 * `DOCS_READABILITY_PROVIDER`/`DOCS_READABILITY_MODEL` below for any other
 * provider/model your `pi` is logged into.
 *
 * It is a SUGGESTION SERVICE only: it reads the given AsciiDoc/Markdown files
 * and returns a list of suggestions. It never edits files — the flow agent
 * applies approved edits with its own tools. This mirrors the opencode
 * `docs-readability` subagent (opencode.jsonc); both share ONE prompt, shipped
 * with the skill at `.agents/skills/ticket-flow/resources/docs-readability.prompt.md`.
 *
 * Mechanism: Pi has no in-process "call model X" primitive, so — like Pi's own
 * examples/extensions/subagent — it spawns a model-pinned `pi` subprocess in
 * print mode with `--no-builtin-tools`, so the child can neither read nor write
 * files; it only transforms the prompt text into suggestions.
 *
 * This file is pickle-owned: `pickle upgrade` refreshes it in place and
 * `pickle uninstall` removes it. Put customizations in a SEPARATE extension
 * file next to this one, not in here.
 *
 * Install: auto-discovered from `.pi/extensions/` once the project is trusted.
 * Hot-reload after edits with `/reload`. Requires a one-time `pi` `/login` to
 * GitHub Copilot (or whichever provider `DOCS_READABILITY_PROVIDER` names).
 */
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import {
  truncateHead,
  formatSize,
  DEFAULT_MAX_BYTES,
  DEFAULT_MAX_LINES,
} from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import { readFile } from "node:fs/promises";
import { join, resolve } from "node:path";

// Reviewer model, reached via your `pi` /login provider. Override with env vars.
const PROVIDER = process.env.DOCS_READABILITY_PROVIDER ?? "github-copilot";
const MODEL = process.env.DOCS_READABILITY_MODEL ?? "gpt-5.4";

// Canonical reviewer prompt — SHARED with the opencode docs-readability agent;
// shipped with the installed ticket-flow skill.
const PROMPT_PATH = ".agents/skills/ticket-flow/resources/docs-readability.prompt.md";

// Guard the single-arg prompt size passed to the subprocess.
const MAX_INPUT_BYTES = 200_000;

export default function (pi: ExtensionAPI) {
  // Core: read the shared prompt + the changed doc files (.adoc/.md), ask a
  // model-pinned `pi` subprocess (read-only) for readability suggestions, return them.
  async function review(files: string[], cwd: string, signal?: AbortSignal): Promise<string> {
    const clean = files.filter((f) => f.trim());
    if (clean.length === 0) {
      throw new Error("docs-readability: no files given to review.");
    }

    const promptHeader = await readFile(join(cwd, PROMPT_PATH), "utf8");

    let corpus = "";
    for (const rel of clean) {
      const abs = resolve(cwd, rel.replace(/^@/, "")); // some models prefix @
      const body = await readFile(abs, "utf8");
      corpus += `\n\n===== FILE: ${rel} =====\n${body}\n`;
    }

    let fullPrompt = `${promptHeader}\n\nReview these changed documentation files:\n${corpus}`;
    if (Buffer.byteLength(fullPrompt, "utf8") > MAX_INPUT_BYTES) {
      const trimmed = truncateHead(fullPrompt, { maxBytes: MAX_INPUT_BYTES, maxLines: 100_000 });
      fullPrompt = `${trimmed.content}\n\n[input truncated for size]`;
    }

    // Pi has no in-process inference API; spawn a model-pinned `pi` in print
    // mode. `--no-builtin-tools` guarantees the child cannot read/write files.
    // NOTE: verify the print-mode invocation for your Pi version (`pi --help`);
    // adjust the flags below if the prompt is not passed as a positional arg.
    const args = ["-p", fullPrompt, "--provider", PROVIDER, "--model", MODEL, "--no-builtin-tools"];

    const result = await pi.exec("pi", args, { signal, timeout: 180_000 });
    if (result.code !== 0) {
      throw new Error(
        `docs-readability: reviewer (\`pi --provider ${PROVIDER} --model ${MODEL}\`) ` +
          `exited ${result.code}. Is that provider logged in (\`pi\` \`/login\`)? ` +
          `stderr:\n${result.stderr?.slice(0, 2000) ?? ""}`,
      );
    }

    const out = truncateHead(result.stdout ?? "", {
      maxBytes: DEFAULT_MAX_BYTES,
      maxLines: DEFAULT_MAX_LINES,
    });
    let text = out.content.trim() || "(the reviewer returned no suggestions)";
    if (out.truncated) {
      text +=
        `\n\n[reviewer output truncated: ${out.outputLines} of ${out.totalLines} lines, ` +
        `${formatSize(out.outputBytes)} of ${formatSize(out.totalBytes)}]`;
    }
    return text;
  }

  // LLM-callable tool — used by the flow agent during a ticket review.
  pi.registerTool({
    name: "docs_readability",
    label: "Docs readability",
    description:
      "Ask an AI reviewer for readability-only suggestions on the given AsciiDoc " +
      "or Markdown files. Suggestions only — never edits files. Use during a " +
      "ticket-flow review on the .adoc/.md files the ticket changed.",
    promptSnippet:
      "Get docs readability suggestions for changed AsciiDoc/Markdown files (ticket review)",
    promptGuidelines: [
      "Use docs_readability during a ticket review to get a second-opinion readability " +
        "pass on the .adoc/.md files the ticket changed, then present each suggestion for " +
        "approval and apply the approved ones yourself.",
    ],
    parameters: Type.Object({
      files: Type.Array(Type.String(), {
        description:
          "Paths (relative to the project root) of the changed .adoc/.md files to review",
      }),
    }),
    async execute(_toolCallId, params, signal, _onUpdate, ctx) {
      const text = await review(params.files, ctx.cwd, signal);
      return {
        content: [{ type: "text", text }],
        details: { provider: PROVIDER, model: MODEL, files: params.files },
      };
    },
  });

  // Thin manual wrapper — `/docs-readability <file> [more …]` (.adoc or .md).
  pi.registerCommand("docs-readability", {
    description:
      "Readability-only AI review of the given .adoc/.md files (suggestions only). " +
      "Usage: /docs-readability <file.adoc|file.md> [more …]",
    handler: async (args, ctx) => {
      const files = args.split(/\s+/).filter(Boolean);
      if (files.length === 0) {
        ctx.ui.notify("Usage: /docs-readability <file.adoc|file.md> [more …]", "warning");
        return;
      }

      ctx.ui.setStatus("docs-readability", `Reviewing ${files.length} file(s) with ${MODEL}…`);
      let text: string;
      try {
        text = await review(files, ctx.cwd, ctx.signal);
      } catch (e) {
        ctx.ui.notify(`docs-readability failed: ${(e as Error).message}`, "error");
        return;
      } finally {
        ctx.ui.setStatus("docs-readability", undefined);
      }

      // Inject the suggestions into the conversation (visible + available to the
      // agent for approve/apply). No turn is triggered; the flow agent picks
      // them up on the next prompt.
      pi.sendMessage({
        customType: "docs-readability",
        content: `Docs readability suggestions (${MODEL}) for: ${files.join(", ")}\n\n${text}`,
        display: true,
      });
      ctx.ui.notify("Readability suggestions added to the conversation.", "info");
    },
  });
}
