package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

const cfgBody = `payload_version = "1"
[commit]
overarching_auto = true
child_publish_gated = true
[[project]]
name = "pickle"
path = "."
wip_in_development = 1
wip_in_review = 1
`

func ticketFile(id, project, depends, impact, hist string) string {
	return fmt.Sprintf(`---
id: %s
title: %s title
project: %s
depends-on: %s
spawned-by: []
impact: %s
complexity: medium
cost: M
---

# %s

## History
- 2026-07-23 — created (%s). source: test
`, id, id, project, depends, impact, id, hist)
}

// historyTemplate is ticketFile's sibling for cases that need a hand-built
// `## History` body (multiple entries, or an entry longer than ticketFile's
// single created line) instead of one created-line. Args, by explicit index so
// id can be reused: %[1]s id, %[2]s project, %[3]s depends-on, %[4]s the raw
// (already dated-bullet-formatted) History body.
const historyTemplate = `---
id: %[1]s
title: %[1]s title
project: %[2]s
depends-on: %[3]s
spawned-by: []
impact: high
complexity: medium
cost: M
---

# %[1]s

## History
%[4]s
`

// withSpawnedBy overrides the fixture's default `spawned-by: []`. Kept as a
// string rewrite rather than a ticketFile parameter so the ten existing call
// sites stay untouched (same trick as internal/move/move_test.go's depends-on).
func withSpawnedBy(body, ids string) string {
	return strings.Replace(body, "spawned-by: []", "spawned-by: "+ids, 1)
}

// withFamily inserts a `family:` line after the fixture's `spawned-by: []`, the
// same string-rewrite trick as withSpawnedBy (family is an optional field, so the
// clean fixture has no such line).
func withFamily(body, id string) string {
	return strings.Replace(body, "spawned-by: []", "spawned-by: []\nfamily: "+id, 1)
}

// writeGood lays down a clean, audit-passing tree at root.
func mk(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mkStatusDirs creates all seven status directories, each carrying a .gitkeep —
// the fixture-side mirror of what `pickle install` scaffolds, and a prerequisite
// for a clean audit now that auditStatusDirs (T-040) checks for them directly.
func mkStatusDirs(t *testing.T, root string) {
	t.Helper()
	for _, s := range ticket.Statuses {
		mk(t, root, filepath.Join("tickets", s.Dir, ".gitkeep"), "")
	}
}

func writeGood(t *testing.T, root string) {
	mk(t, root, "pickle.toml", cfgBody)
	mkStatusDirs(t, root)
	mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "high", "TO DO"))
	renderBoard(t, root)
}

// renderBoard regenerates BOARD.md from the tree — the board is a generated
// artifact (T-044), so a "good" board is by definition a fresh render.
func renderBoard(t *testing.T, root string) {
	t.Helper()
	if err := board.Regenerate(root, loadCfg(t, root)); err != nil {
		t.Fatalf("regenerate board: %v", err)
	}
}

func loadCfg(t *testing.T, root string) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatalf("load cfg: %v", err)
	}
	return cfg
}

func TestAudit(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(t *testing.T, root string)
		wantErr    bool
		substr     string
		wantWarn   bool // T-040: some new checks are warnings, not errors
		warnSubstr string
	}{
		{name: "clean"},
		{name: "bad filename", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/NOPE.md", "x")
		}, wantErr: true, substr: "filename does not match"},
		{name: "missing key", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				strings.Replace(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "cost: M\n", "", 1))
		}, wantErr: true, substr: `missing "cost"`},
		{name: "illegal grade", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "banana", "TO DO"))
		}, wantErr: true, substr: "illegal impact"},
		{name: "duplicate id", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-001-bar.md", ticketFile("T-001", "pickle", "[]", "high", "READY"))
		}, wantErr: true, substr: "duplicate id"},
		// A key that appears twice silently overwrites in the parse (last wins,
		// T-033) — malformed however it arrived.
		{name: "duplicate frontmatter key", mutate: func(t *testing.T, root string) {
			body := strings.Replace(ticketFile("T-001", "pickle", "[]", "high", "TO DO"),
				"impact: high\n", "impact: high\nimpact: high\n", 1)
			mk(t, root, "tickets/1-to-do/T-001-foo.md", body)
		}, wantErr: true, substr: `duplicate key "impact"`},
		{name: "dangling depends-on", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[T-404]", "high", "TO DO"))
		}, wantErr: true, substr: "depends-on T-404 does not exist"},
		// T-027: a ticket citing itself in depends-on audits clean today, then
		// silently self-blocks at pickup (the dependency can never reach 6-done/
		// while this very ticket is in development).
		{name: "self-referencing depends-on", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[T-001]", "high", "TO DO"))
		}, wantErr: true, substr: "depends-on lists itself"},
		{name: "unregistered project", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "nope", "[]", "high", "TO DO"))
		}, wantErr: true, substr: `project "nope" is not a registered child`},
		// The child "pickle" has no ticket_prefix, so its effective prefix is "T";
		// a RICK-prefixed id filed against it is the half-migrated / mis-filed case
		// the new invariant (T-058) catches.
		{name: "prefix mismatch", mutate: func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "tickets/1-to-do/T-001-foo.md")); err != nil {
				t.Fatal(err)
			}
			mk(t, root, "tickets/1-to-do/RICK-001-foo.md", ticketFile("RICK-001", "pickle", "[]", "high", "TO DO"))
			renderBoard(t, root)
		}, wantErr: true, substr: `id prefix "RICK" does not match project "pickle"`},
		// The board is a generated artifact (T-044 D6). Two tiers (T-052): a row
		// disagreeing with the ticket files — a stray row, a moved row, a deleted
		// section — is an error; a fresh render's *rows* all matching, with only
		// generated scaffolding stale, is a warning.
		{name: "stale board (hand-edited row)", mutate: func(t *testing.T, root string) {
			b, _ := os.ReadFile(filepath.Join(root, "tickets/BOARD.md"))
			mk(t, root, "tickets/BOARD.md", strings.Replace(string(b),
				"| T-001 |", "| T-999 | ghost | low | low | S | [] |\n| T-001 |", 1))
		}, wantErr: true, substr: "BOARD.md does not match the ticket files"},
		{name: "stale board (ticket changed, board not regenerated)", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-002-bar.md", ticketFile("T-002", "pickle", "[]", "high", "READY"))
		}, wantErr: true, substr: "BOARD.md does not match the ticket files"},
		// Registering a second child changes the board's generated shape (a new
		// `### <child>` section + WIP line per status) without any existing row
		// disagreeing — exactly the T-052 field report (project add, then upgrade).
		{name: "stale board (layout only — new registered child, no tickets touched)", mutate: func(t *testing.T, root string) {
			cfg := loadCfg(t, root)
			if err := cfg.AddProject(config.Project{Name: "web", Path: ".", WIPInDevelopment: 1, WIPInReview: 1}); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Save(""); err != nil {
				t.Fatal(err)
			}
			// BOARD.md is deliberately left as writeGood rendered it (single child) —
			// the new project's own section is what's missing, not any ticket row.
		}, wantWarn: true, warnSubstr: "BOARD.md is out of date in its generated layout only"},
		{name: "missing board", mutate: func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "tickets/BOARD.md")); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true, substr: "BOARD.md"},
		{name: "history mismatch", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "high", "DONE"))
		}, wantErr: true, substr: "last History status is DONE"},
		{name: "wip breach", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-010-a.md", ticketFile("T-010", "pickle", "[]", "high", "IN DEVELOPMENT"))
			mk(t, root, "tickets/3-in-development/T-011-b.md", ticketFile("T-011", "pickle", "[]", "high", "IN DEVELOPMENT"))
			renderBoard(t, root)
		}, wantErr: true, substr: "in 3-in-development (limit 1)"},
		{name: "in-dev dep not done", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-002-bar.md", ticketFile("T-002", "pickle", "[T-001]", "high", "IN DEVELOPMENT"))
			renderBoard(t, root)
		}, wantErr: true, substr: "dependency T-001 is in 1-to-do"},

		// --- status directories (T-040 D3): the seven-directory tree invariant ---
		{name: "missing status directory", mutate: func(t *testing.T, root string) {
			if err := os.RemoveAll(filepath.Join(root, "tickets/7-dropped")); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true, substr: "tickets/7-dropped/: status directory is missing"},
		{name: "empty status dir without .gitkeep", mutate: func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "tickets/5-rework/.gitkeep")); err != nil {
				t.Fatal(err)
			}
		}, wantWarn: true, warnSubstr: "tickets/5-rework/: empty and not kept by a .gitkeep"},

		// --- History-entry length (T-040 D4/D5): a warning, and kind-aware ---
		{name: "over-long transition warns", mutate: func(t *testing.T, root string) {
			reason := strings.Repeat("x", 450)
			body := fmt.Sprintf(historyTemplate, "T-005", "pickle", "[]",
				"- 2026-07-23 — created (TO DO). source: test\n"+
					"- 2026-07-24 — TO DO → READY: "+reason)
			mk(t, root, "tickets/2-ready/T-005-foo.md", body)
			renderBoard(t, root)
		}, wantWarn: true, warnSubstr: "History entry 2026-07-24 is"},
		// A free-form dated note of the same length is exempt: it carries no
		// TEMPLATE-prescribed shape to violate (measured up to 2199 runes in this
		// repo's own History, all legitimate).
		{name: "over-long free-form note stays clean", mutate: func(t *testing.T, root string) {
			note := strings.Repeat("x", 450)
			body := fmt.Sprintf(historyTemplate, "T-005", "pickle", "[]",
				"- 2026-07-23 — created (TO DO). source: test\n"+
					"- 2026-07-24 — a free-form analysis note with no arrow at all: "+note)
			mk(t, root, "tickets/1-to-do/T-005-foo.md", body)
			renderBoard(t, root)
		}},
		// A created line of the same length is exempt too: it carries provenance
		// prose (measured up to 331 runes in this repo's own History).
		{name: "over-long created line stays clean", mutate: func(t *testing.T, root string) {
			source := strings.Repeat("x", 450)
			body := fmt.Sprintf(historyTemplate, "T-005", "pickle", "[]",
				"- 2026-07-23 — created (TO DO). source: "+source)
			mk(t, root, "tickets/1-to-do/T-005-foo.md", body)
			renderBoard(t, root)
		}},

		// --- spawned-by: lineage, validated for existence but never a gate (T-024) ---
		{name: "valid spawned-by", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-002-bar.md",
				withSpawnedBy(ticketFile("T-002", "pickle", "[]", "high", "READY"), "[T-001]"))
			renderBoard(t, root)
		}},
		{name: "dangling spawned-by", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withSpawnedBy(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "[T-404]"))
		}, wantErr: true, substr: "spawned-by T-404 does not exist"},
		{name: "self-referencing spawned-by", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withSpawnedBy(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "[T-001]"))
		}, wantErr: true, substr: "spawned-by lists itself"},
		{name: "missing spawned-by key", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				strings.Replace(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "spawned-by: []\n", "", 1))
		}, wantErr: true, substr: `missing "spawned-by"`},
		// The defining guarantee: an in-development ticket whose lineage parent is
		// neither done nor merged is still clean. The identical shape with
		// depends-on is the "in-dev dep not done" error case above.
		{name: "in-dev spawned-by parent not done", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-002-bar.md",
				withSpawnedBy(ticketFile("T-002", "pickle", "[]", "high", "IN DEVELOPMENT"), "[T-001]"))
			renderBoard(t, root)
		}},

		// --- family: same-child umbrella grouping, validated but never a gate (T-059) ---
		{name: "valid family", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-002-bar.md",
				withFamily(ticketFile("T-002", "pickle", "[]", "high", "READY"), "T-001"))
			renderBoard(t, root)
		}},
		{name: "dangling family", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withFamily(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "T-404"))
		}, wantErr: true, substr: "family T-404 does not exist"},
		{name: "self-referencing family", mutate: func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withFamily(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "T-001"))
		}, wantErr: true, substr: "family lists itself"},
		{name: "nested family", mutate: func(t *testing.T, root string) {
			// T-002 is a member of T-001; T-003 pointing at T-002 is the no-nesting case.
			mk(t, root, "tickets/2-ready/T-002-bar.md",
				withFamily(ticketFile("T-002", "pickle", "[]", "high", "READY"), "T-001"))
			mk(t, root, "tickets/2-ready/T-003-baz.md",
				withFamily(ticketFile("T-003", "pickle", "[]", "high", "READY"), "T-002"))
			renderBoard(t, root)
		}, wantErr: true, substr: "families do not nest"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeGood(t, root)
			if tc.mutate != nil {
				tc.mutate(t, root)
			}
			res := Audit(root, loadCfg(t, root))
			switch {
			case tc.wantErr:
				if len(res.Errors) == 0 {
					t.Fatalf("expected an error, got none (warnings: %v)", res.Warnings)
				}
				if tc.substr != "" && !containsAny(res.Errors, tc.substr) {
					t.Fatalf("no error contained %q; errors: %v", tc.substr, res.Errors)
				}
			case tc.wantWarn:
				if len(res.Errors) != 0 {
					t.Fatalf("expected no errors (warning-only case), got: %v", res.Errors)
				}
				if len(res.Warnings) == 0 {
					t.Fatalf("expected a warning, got none")
				}
				if tc.warnSubstr != "" && !containsAny(res.Warnings, tc.warnSubstr) {
					t.Fatalf("no warning contained %q; warnings: %v", tc.warnSubstr, res.Warnings)
				}
			default:
				if len(res.Errors) != 0 {
					t.Fatalf("expected clean audit, got errors: %v", res.Errors)
				}
				if len(res.Warnings) != 0 {
					t.Fatalf("expected no warnings, got: %v", res.Warnings)
				}
			}
		})
	}
}

// TestAuditPrefixMatch confirms the positive side of the T-058 invariant: a
// child with an explicit ticket_prefix and a matching id audits clean. (The
// mismatch and legacy-default cases are covered in the TestAudit table.)
func TestAuditPrefixMatch(t *testing.T) {
	const cfgRick = `payload_version = "1"
[commit]
overarching_auto = true
child_publish_gated = true
[[project]]
name = "rick"
path = "."
ticket_prefix = "RICK"
wip_in_development = 1
wip_in_review = 1
`
	root := t.TempDir()
	mk(t, root, "pickle.toml", cfgRick)
	mkStatusDirs(t, root)
	mk(t, root, "tickets/1-to-do/RICK-001-foo.md", ticketFile("RICK-001", "rick", "[]", "high", "TO DO"))
	renderBoard(t, root)

	res := Audit(root, loadCfg(t, root))
	if len(res.Errors) != 0 {
		t.Fatalf("RICK-001 under a RICK-prefixed child must audit clean, got: %v", res.Errors)
	}
}

// TestAuditFamilyCrossChild: a family must stay within one child. Two registered
// children, an umbrella in one and a member in the other, must fail (the board
// groups per child, so a cross-child family could not render as one group).
func TestAuditFamilyCrossChild(t *testing.T) {
	const cfgTwo = `payload_version = "1"
[commit]
overarching_auto = true
child_publish_gated = true
[[project]]
name = "pickle"
path = "."
wip_in_development = 1
wip_in_review = 1
[[project]]
name = "rick"
path = "rick"
ticket_prefix = "RICK"
wip_in_development = 1
wip_in_review = 1
`
	root := t.TempDir()
	mk(t, root, "pickle.toml", cfgTwo)
	mkStatusDirs(t, root)
	if err := os.Mkdir(filepath.Join(root, "rick"), 0o755); err != nil {
		t.Fatal(err)
	}
	mk(t, root, "tickets/1-to-do/T-001-umbrella.md", ticketFile("T-001", "pickle", "[]", "high", "TO DO"))
	mk(t, root, "tickets/1-to-do/RICK-001-member.md",
		withFamily(ticketFile("RICK-001", "rick", "[]", "high", "TO DO"), "T-001"))
	renderBoard(t, root)

	res := Audit(root, loadCfg(t, root))
	if !containsAny(res.Errors, "family T-001 is in a different child-project") {
		t.Fatalf("expected a cross-child family error; errors: %v", res.Errors)
	}
}

func containsAny(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

// TestFrontmatterKeysMatchTemplate is the TEMPLATE-drift guard (T-040 D2):
// requiredKeys/optionalKeys, skill/resources/TEMPLATE.md, and ticket.Scaffold
// must all agree on the frontmatter key vocabulary. It lives here rather than in
// internal/ticket (where TestScaffoldSectionsMatchTemplate already guards the
// ## section set) because only internal/audit can see requiredKeys/optionalKeys
// alongside ticket.Scaffold without an import cycle.
func TestFrontmatterKeysMatchTemplate(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "skill", "resources", "TEMPLATE.md")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Skipf("TEMPLATE.md not found: %v", err)
	}
	text := string(data)

	// (a) TEMPLATE's *active* frontmatter keys must equal requiredKeys.
	fm, ok := ticket.ParseFrontmatter(text)
	if !ok {
		t.Fatal("TEMPLATE.md has no frontmatter block")
	}
	required := sliceSet(requiredKeys)
	if active := keySet(fm); !setsEqual(active, required) {
		t.Errorf("TEMPLATE.md's active frontmatter keys = %v, requiredKeys = %v — "+
			"the audit and the authoring guide have drifted", active, required)
	}

	// (b) TEMPLATE's *commented-out* frontmatter keys must equal optionalKeys, and
	// the two lists (required vs optional) must stay disjoint — an ambiguous key
	// would mean the audit and TEMPLATE disagree on whether it's mandatory.
	commented := commentedFrontmatterKeys(t, text)
	optional := sliceSet(optionalKeys)
	if !setsEqual(commented, optional) {
		t.Errorf("TEMPLATE.md's commented-out (optional) frontmatter keys = %v, optionalKeys = %v",
			commented, optional)
	}
	for k := range commented {
		if required[k] {
			t.Errorf("TEMPLATE.md's commented key %q also appears in requiredKeys — ambiguous", k)
		}
	}

	// (c) ticket.Scaffold must render the same vocabulary: requiredKeys with no
	// family, requiredKeys+family with one (T-059's contract: the family line is
	// omitted entirely, not emitted empty).
	none, ok := ticket.ParseFrontmatter(ticket.Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil, ""))
	if !ok {
		t.Fatal("Scaffold produced no frontmatter block")
	}
	if got := keySet(none); !setsEqual(got, required) {
		t.Errorf("Scaffold (no family) keys = %v, want requiredKeys %v", got, required)
	}
	withFam, ok := ticket.ParseFrontmatter(ticket.Scaffold("T-001", "x", "pickle", "high", "medium", "M", nil, "T-002"))
	if !ok {
		t.Fatal("Scaffold produced no frontmatter block")
	}
	wantWithFam := sliceSet(requiredKeys)
	wantWithFam["family"] = true
	if got := keySet(withFam); !setsEqual(got, wantWithFam) {
		t.Errorf("Scaffold (with family) keys = %v, want requiredKeys + family = %v", got, wantWithFam)
	}
}

var commentedKeyRE = regexp.MustCompile(`^#\s*([a-z-]+):`)

// commentedFrontmatterKeys scans only the leading `---`-delimited frontmatter
// block (same bounds as ticket.ParseFrontmatter) for commented-out `# key: ...`
// lines, so a commented example elsewhere in the authoring guide can never be
// mistaken for an optional frontmatter key.
func commentedFrontmatterKeys(t *testing.T, text string) map[string]bool {
	t.Helper()
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatal("TEMPLATE.md does not start with a frontmatter block")
	}
	out := map[string]bool{}
	for i := 1; i < len(lines) && strings.TrimSpace(lines[i]) != "---"; i++ {
		if m := commentedKeyRE.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil {
			out[m[1]] = true
		}
	}
	return out
}

func keySet(fm map[string]string) map[string]bool {
	out := make(map[string]bool, len(fm))
	for k := range fm {
		out[k] = true
	}
	return out
}

func sliceSet(keys []string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}

func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
