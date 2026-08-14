package main

// payload_lint_test.go is the mechanical half of AGENTS.md's foreign-workspace
// test (see the paragraph above the pickle:begin marker). It walks payloadFS —
// what actually ships into another project's tree, not the working copy that
// happens to sit beside it — and fails the build the way a broken link would
// when a sentence in it only makes sense to a reader standing inside pickle's
// own repo. It replaces three hand sweeps (T-098's original, and the two
// escapes that reached review afterwards) with something that cannot forget
// to run.
//
// This file is itself repo-local, not payload: `agents/` and `skill/` are the
// two embedded roots under lint; AGENTS.md, docs/, tickets/ and this file are
// not, because naming pickle's own paths here is correct — the whole point is
// the payload cannot do the same.

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// payloadLintRule is one mechanical check standing in for a piece of
// AGENTS.md's foreign-workspace-test judgement. Four ship, all of them
// failing (never warning) — a regression guard that degrades to advisory is
// the very hand sweep this file exists to replace.
type payloadLintRule struct {
	name    string
	pattern *regexp.Regexp
	why     string // printed on failure: teaches the fix, not just the defect
	// exempt reports whether the match at line[start:end] is a recognised
	// legitimate shape (decision 5) and should not be flagged. ctx carries
	// fenced-code-block state tracked by the walker across lines; nil means
	// the rule has no shape exemption.
	exempt func(ctx lineCtx, start, end int) bool
}

type payloadLintFinding struct {
	path string
	line int
	rule string
	text string
	why  string
}

func (f payloadLintFinding) String() string {
	return fmt.Sprintf(
		"%s:%d: [%s] %q\n    %s\n    see AGENTS.md's foreign-workspace test for the reasoning",
		f.path, f.line, f.rule, f.text, f.why)
}

// lineCtx is what a rule's exempt func sees about the line a candidate match
// sits on: the raw text (for a per-line backtick scan) and whether the walker
// currently considers the line inside a fenced code block.
type lineCtx struct {
	text    string
	inFence bool
}

// insideBackticks reports whether byte offset start in ctx.text sits inside an
// inline code span, by counting backticks strictly before it: an odd count
// means an unmatched opening backtick precedes the match, i.e. we are inside
// a span. This is a per-line approximation (decision: "a per-line odd/even
// backtick scan is enough") — it misreads an inline span that itself wraps
// onto a second physical line (CommonMark permits this inside one paragraph),
// and an escaped backtick. Neither shape appears in the payload today; if one
// ever does, this comment is where to widen the check.
func insideBackticks(ctx lineCtx, start, _ int) bool {
	return strings.Count(ctx.text[:start], "`")%2 == 1
}

// isProvenanceTag reports whether the matched id closes a parenthetical on
// the same line — "(T-42)", or just as legitimately "(`## Outcome`, T-083)"
// where the parens hold other text too. Either way the id names which ticket
// introduced a rule rather than asking the reader to go look one up, so the
// test is deliberately loose: "immediately followed by ')'", not "the whole
// parenthetical is only the id".
func isProvenanceTag(ctx lineCtx, _, end int) bool {
	return end < len(ctx.text) && ctx.text[end] == ')'
}

// ticketRefExempt is the shape exemption for rule 1 (decision 5): a
// backtick/fenced syntax-filler example, or a provenance tag. A metasyntactic
// id (T-NNN, T-MMM, ...) never reaches this func at all — rule 1's pattern
// requires digits, so those never match in the first place (see
// TestPayloadLintRule1LookupShapedReferences/metasyntactic-id-never-matches).
func ticketRefExempt(ctx lineCtx, start, end int) bool {
	if ctx.inFence {
		return true
	}
	return insideBackticks(ctx, start, end) || isProvenanceTag(ctx, start, end)
}

// escapeHatch is the one allowlist decision 6 permits: an exact substring, in
// this file, each entry carrying a one-line comment saying why it is
// legitimate despite matching a rule. No file:line list (rots on the next
// edit) and no in-payload marker (lint machinery shipped inside the payload
// is a fresh instance of the defect being guarded against) — only this. Empty
// today: the payload needs no excuse as of this ticket, and an entry should
// be earned one at a time, not grown by habit.
var escapeHatch []string

func isEscapeHatched(line string) bool {
	for _, entry := range escapeHatch {
		if strings.Contains(line, entry) {
			return true
		}
	}
	return false
}

// payloadLintRules returns the four checks, freshly built per call so tests
// that only want a subset (or want to add a scratch escapeHatch entry) never
// share mutable state with each other.
func payloadLintRules() []payloadLintRule {
	return []payloadLintRule{
		{
			name: "ticket-lookup",
			// T-098's own seed pattern (tickets/6-done/T-[0-9]|T-[0-9]{3}
			// F[0-9]), generalised to any digit-length id and any status
			// directory. This deliberately does NOT flag every bare T-\d+ —
			// only the shape that instructs a reader to go resolve a specific
			// id, which a foreign workspace cannot do (their tickets/ has
			// different numbers under the same status directories).
			pattern: regexp.MustCompile(`tickets/[1-7]-[a-z-]+/T-\d+|T-\d+\s*F\d+`),
			why: "a ticket id the reader is told to go look up: a well-formed path or finding " +
				"reference that means something else in their own tickets/ tree. Phrase the " +
				"point without sending the reader to resolve a specific id.",
			exempt: ticketRefExempt,
		},
		{
			name:    "first-person-repo",
			pattern: regexp.MustCompile(`(?i)in this repository|this repo\b|the corpus\b|our own\b`),
			why: `"this repo" meaning ours, or a body of evidence ("the corpus") the reader does ` +
				"not have: no foreign team can assign either. Name the thing directly instead of " +
				"pointing at an implicit shared context.",
		},
		{
			name: "repo-only-path",
			// Rooted in pickle's own source tree: skill/, internal/, cmd/,
			// agents/, docs/, .github/, and the bare top-level files. A
			// leading boundary is matched explicitly (RE2 has no lookbehind)
			// so ".agents/skills/brine/" does not trip on "agents/" (blocked
			// by the "." immediately before it) or "skill/" (the directory
			// there is "skills/", one character short of the literal).
			// tickets/ is deliberately absent from this list — every
			// installed project has tickets/1-to-do/, so a blanket path ban
			// would be the trap this rule exists to avoid; a ticket-shaped
			// path is rule 1's job, not this one's.
			pattern: regexp.MustCompile(`(^|[^\w./-])(skill|internal|cmd|agents|docs|\.github)/|\b(assets\.go|justfile|\.goreleaser\.yaml)\b`),
			why: "a path that only resolves in pickle's own source tree; an installed workspace " +
				"has no skill/, internal/, cmd/, docs/ or .github/ at its root. Phrase the path " +
				"relative to the skill the reader is holding (\"this skill's own " +
				"resources/TEMPLATE.md\"), not to pickle's checkout.",
		},
		{
			name: "invisible-evidence",
			// The fuzziest of the four, kept deliberately narrow (decision
			// 8): a short keyword list plus one shape (a bare count next to
			// an evidence noun), grown only when a real escape happens, never
			// broadened by guessing. This is the honest 80%, not a proof —
			// the sentence-level judgement AGENTS.md describes still belongs
			// to whoever writes or reviews the sentence.
			pattern: regexp.MustCompile(`(?i)pre-registered|the corpus\b|our own\b|\d+ (variants|instances|cases|examples) (across|in) the\b`),
			why: "a definite-article appeal to evidence the reader does not have (\"the " +
				"pre-registered criterion\", \"the corpus\", \"the 13 variants\"): whose? State " +
				"the claim so it stands on its own, or drop it.",
		},
	}
}

// lintFile runs every rule over content line by line, tracking fenced-code-
// block state across lines so a multi-line ``` block is exempt for rule 1 in
// its entirety (an inline backtick span, by contrast, cannot cross a line —
// see insideBackticks). content is linted as-is; path is carried through only
// to label findings.
func lintFile(path, content string, rules []payloadLintRule) []payloadLintFinding {
	var findings []payloadLintFinding
	inFence := false
	for i, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		isFenceDelim := strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
		ctx := lineCtx{text: line, inFence: inFence}
		if isFenceDelim {
			// Lint the delimiter line itself at its pre-toggle state (it is
			// fence syntax, not fenced content), then flip for what follows.
			inFence = !inFence
		}
		for _, r := range rules {
			for _, loc := range r.pattern.FindAllStringIndex(line, -1) {
				start, end := loc[0], loc[1]
				if r.exempt != nil && r.exempt(ctx, start, end) {
					continue
				}
				if isEscapeHatched(line) {
					continue
				}
				findings = append(findings, payloadLintFinding{
					path: path, line: i + 1, rule: r.name, text: line[start:end], why: r.why,
				})
			}
		}
	}
	return findings
}

// lintPayload walks both embedded payload roots (skill/, agents/ — assets.go
// documents why both ship) and lints every file it finds, extension-agnostic:
// .md, .jsonc and .ts are all prose or prose-carrying, and skipping by
// extension is exactly the kind of exception that lets the next one through.
func lintPayload(t *testing.T) []payloadLintFinding {
	t.Helper()
	rules := payloadLintRules()
	var findings []payloadLintFinding
	err := fs.WalkDir(payloadFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := payloadFS.ReadFile(path)
		if err != nil {
			return err
		}
		findings = append(findings, lintFile(path, string(data), rules)...)
		return nil
	})
	if err != nil {
		t.Fatalf("walking payloadFS: %v", err)
	}
	return findings
}

// TestPayloadSpeaksToAForeignReader is the regression guard itself: it fails
// the build the moment a sentence reaches skill/ or agents/ that only makes
// sense to a reader standing inside pickle's own repo, instead of depending
// on a hand sweep catching what the last two missed.
func TestPayloadSpeaksToAForeignReader(t *testing.T) {
	findings := lintPayload(t)
	if len(findings) == 0 {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "payload lint found %d line(s) that speak to the reader as if the reader "+
		"were standing in pickle's own repo:\n\n", len(findings))
	for _, f := range findings {
		b.WriteString(f.String())
		b.WriteString("\n\n")
	}
	t.Fatal(b.String())
}

// TestPayloadLintRulesCatchTheEscapesTheySawInReview replays the ticket's
// stated first two test cases: the two shapes that reached T-098's own review
// after its hand sweep declared the payload clean, and that none of T-098's
// four rg patterns could have caught. These are synthetic strings, not files
// — the point is that these two specific sentences would have been caught.
func TestPayloadLintRulesCatchTheEscapesTheySawInReview(t *testing.T) {
	rules := payloadLintRules()
	cases := []struct {
		name     string
		line     string
		wantRule string
	}{
		{
			name:     "escape 1 (found at pickup): a repo-only path",
			line:     "see `skill/resources/TEMPLATE.md` for the shape",
			wantRule: "repo-only-path",
		},
		{
			name:     "escape 2 (found at review): an invisible-evidence appeal",
			line:     "the **pre-registered criterion** this column exists to test",
			wantRule: "invisible-evidence",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings := lintFile("synthetic.md", c.line, rules)
			for _, f := range findings {
				if f.rule == c.wantRule {
					return
				}
			}
			t.Fatalf("expected rule %q to flag %q; findings: %+v", c.wantRule, c.line, findings)
		})
	}
}

// TestPayloadLintRulesLeaveLegitimateShapesAlone is the over-correction
// guard: a check that deletes the payload's legitimate references is worse
// than no check. Every line here is a real shape drawn from the live payload
// today (SKILL.md's (T-083) tag, tickets-README.md's grammar examples, the
// installed TEMPLATE.md path, an ordinary tickets/ path, and the metasyntactic
// placeholder id) and must produce zero findings across all four rules.
func TestPayloadLintRulesLeaveLegitimateShapesAlone(t *testing.T) {
	rules := payloadLintRules()
	lines := []string{
		"(T-083)",
		"`board: T-084 ready → in development`",
		".agents/skills/brine/resources/TEMPLATE.md",
		"tickets/1-to-do/",
		"T-NNN",
		"resources/TEMPLATE.md",
		"tickets/README.md",
	}
	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			if findings := lintFile("synthetic.md", line, rules); len(findings) != 0 {
				t.Fatalf("expected %q to pass, got findings: %+v", line, findings)
			}
		})
	}
}

// TestPayloadLintRule1LookupShapedReferences exercises rule 1 directly: the
// shape it targets, the two exemption paths (backtick/fenced syntax filler,
// provenance-tag closing paren) that let a legitimate use through unflagged,
// and the metasyntactic ids that never reach the exemption logic at all
// because the pattern requires digits.
func TestPayloadLintRule1LookupShapedReferences(t *testing.T) {
	rules := payloadLintRules()
	one := func(name string) payloadLintRule {
		for _, r := range rules {
			if r.name == name {
				return r
			}
		}
		t.Fatalf("no rule named %q", name)
		return payloadLintRule{}
	}
	rule1 := one("ticket-lookup")

	t.Run("flags an unwrapped lookup-shaped reference", func(t *testing.T) {
		line := "see tickets/6-done/T-090 F1 for the pattern"
		if findings := lintFile("s.md", line, []payloadLintRule{rule1}); len(findings) == 0 {
			t.Fatalf("expected %q to be flagged", line)
		}
	})

	t.Run("backtick span exempts a syntax-filler example of the bad shape", func(t *testing.T) {
		line := "the shape to avoid: `tickets/6-done/T-090 F1`"
		if findings := lintFile("s.md", line, []payloadLintRule{rule1}); len(findings) != 0 {
			t.Fatalf("expected backtick-wrapped %q to pass, got %+v", line, findings)
		}
	})

	t.Run("fenced code block exempts a multi-line example", func(t *testing.T) {
		content := "```\nsee tickets/6-done/T-090 F1\n```\n"
		if findings := lintFile("s.md", content, []payloadLintRule{rule1}); len(findings) != 0 {
			t.Fatalf("expected fenced block to pass, got %+v", findings)
		}
	})

	t.Run("a closing paren treats the reference as a provenance tag", func(t *testing.T) {
		line := "(see tickets/6-done/T-090)"
		if findings := lintFile("s.md", line, []payloadLintRule{rule1}); len(findings) != 0 {
			t.Fatalf("expected %q to pass, got %+v", line, findings)
		}
	})

	t.Run("metasyntactic id never matches: rule requires digits", func(t *testing.T) {
		line := "tickets/6-done/T-NNN"
		if findings := lintFile("s.md", line, []payloadLintRule{rule1}); len(findings) != 0 {
			t.Fatalf("expected metasyntactic %q to pass, got %+v", line, findings)
		}
	})
}
