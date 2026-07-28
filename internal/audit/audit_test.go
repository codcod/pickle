package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
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

// withSpawnedBy overrides the fixture's default `spawned-by: []`. Kept as a
// string rewrite rather than a ticketFile parameter so the ten existing call
// sites stay untouched (same trick as internal/move/move_test.go's depends-on).
func withSpawnedBy(body, ids string) string {
	return strings.Replace(body, "spawned-by: []", "spawned-by: "+ids, 1)
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

func writeGood(t *testing.T, root string) {
	mk(t, root, "pickle.toml", cfgBody)
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
		name    string
		mutate  func(t *testing.T, root string)
		wantErr bool
		substr  string
	}{
		{"clean", nil, false, ""},
		{"bad filename", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/NOPE.md", "x")
		}, true, "filename does not match"},
		{"missing key", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				strings.Replace(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "cost: M\n", "", 1))
		}, true, `missing "cost"`},
		{"illegal grade", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "banana", "TO DO"))
		}, true, "illegal impact"},
		{"duplicate id", func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-001-bar.md", ticketFile("T-001", "pickle", "[]", "high", "READY"))
		}, true, "duplicate id"},
		{"dangling depends-on", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[T-404]", "high", "TO DO"))
		}, true, "depends-on T-404 does not exist"},
		{"unregistered project", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "nope", "[]", "high", "TO DO"))
		}, true, `project "nope" is not a registered child`},
		// The child "pickle" has no ticket_prefix, so its effective prefix is "T";
		// a RICK-prefixed id filed against it is the half-migrated / mis-filed case
		// the new invariant (T-058) catches.
		{"prefix mismatch", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "tickets/1-to-do/T-001-foo.md")); err != nil {
				t.Fatal(err)
			}
			mk(t, root, "tickets/1-to-do/RICK-001-foo.md", ticketFile("RICK-001", "pickle", "[]", "high", "TO DO"))
			renderBoard(t, root)
		}, true, `id prefix "RICK" does not match project "pickle"`},
		// The board is a generated artifact: the one board invariant is that it
		// matches a fresh render (D6). Any hand-edit — a stray row, a moved row, a
		// deleted section — collapses into the same single staleness error.
		{"stale board (hand-edited row)", func(t *testing.T, root string) {
			b, _ := os.ReadFile(filepath.Join(root, "tickets/BOARD.md"))
			mk(t, root, "tickets/BOARD.md", strings.Replace(string(b),
				"| T-001 |", "| T-999 | ghost | low | low | S | [] |\n| T-001 |", 1))
		}, true, "stale or hand-edited"},
		{"stale board (ticket changed, board not regenerated)", func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-002-bar.md", ticketFile("T-002", "pickle", "[]", "high", "READY"))
		}, true, "stale or hand-edited"},
		{"missing board", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "tickets/BOARD.md")); err != nil {
				t.Fatal(err)
			}
		}, true, "BOARD.md"},
		{"history mismatch", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "high", "DONE"))
		}, true, "last History status is DONE"},
		{"wip breach", func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-010-a.md", ticketFile("T-010", "pickle", "[]", "high", "IN DEVELOPMENT"))
			mk(t, root, "tickets/3-in-development/T-011-b.md", ticketFile("T-011", "pickle", "[]", "high", "IN DEVELOPMENT"))
			renderBoard(t, root)
		}, true, "in 3-in-development (limit 1)"},
		{"in-dev dep not done", func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-002-bar.md", ticketFile("T-002", "pickle", "[T-001]", "high", "IN DEVELOPMENT"))
			renderBoard(t, root)
		}, true, "dependency T-001 is in 1-to-do"},

		// --- spawned-by: lineage, validated for existence but never a gate (T-024) ---
		{"valid spawned-by", func(t *testing.T, root string) {
			mk(t, root, "tickets/2-ready/T-002-bar.md",
				withSpawnedBy(ticketFile("T-002", "pickle", "[]", "high", "READY"), "[T-001]"))
			renderBoard(t, root)
		}, false, ""},
		{"dangling spawned-by", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withSpawnedBy(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "[T-404]"))
		}, true, "spawned-by T-404 does not exist"},
		{"self-referencing spawned-by", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				withSpawnedBy(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "[T-001]"))
		}, true, "spawned-by lists itself"},
		{"missing spawned-by key", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md",
				strings.Replace(ticketFile("T-001", "pickle", "[]", "high", "TO DO"), "spawned-by: []\n", "", 1))
		}, true, `missing "spawned-by"`},
		// The defining guarantee: an in-development ticket whose lineage parent is
		// neither done nor merged is still clean. The identical shape with
		// depends-on is the "in-dev dep not done" error case above.
		{"in-dev spawned-by parent not done", func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-002-bar.md",
				withSpawnedBy(ticketFile("T-002", "pickle", "[]", "high", "IN DEVELOPMENT"), "[T-001]"))
			renderBoard(t, root)
		}, false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeGood(t, root)
			if tc.mutate != nil {
				tc.mutate(t, root)
			}
			res := Audit(root, loadCfg(t, root))
			if tc.wantErr {
				if len(res.Errors) == 0 {
					t.Fatalf("expected an error, got none (warnings: %v)", res.Warnings)
				}
				if tc.substr != "" && !containsAny(res.Errors, tc.substr) {
					t.Fatalf("no error contained %q; errors: %v", tc.substr, res.Errors)
				}
			} else {
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
	mk(t, root, "tickets/1-to-do/RICK-001-foo.md", ticketFile("RICK-001", "rick", "[]", "high", "TO DO"))
	renderBoard(t, root)

	res := Audit(root, loadCfg(t, root))
	if len(res.Errors) != 0 {
		t.Fatalf("RICK-001 under a RICK-prefixed child must audit clean, got: %v", res.Errors)
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
