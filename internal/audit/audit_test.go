package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
impact: %s
complexity: medium
cost: M
---

# %s

## History
- 2026-07-23 — created (%s). source: test
`, id, id, project, depends, impact, id, hist)
}

const goodBoard = `# Board
## TO DO
### pickle
| id | title |
|---|---|
| T-001 | foo |
`

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
	mk(t, root, "tickets/BOARD.md", goodBoard)
	mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "high", "TO DO"))
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
		{"board status mismatch", func(t *testing.T, root string) {
			mk(t, root, "tickets/BOARD.md", strings.Replace(goodBoard, "## TO DO", "## READY", 1))
		}, true, "listed under READY"},
		{"missing from board", func(t *testing.T, root string) {
			mk(t, root, "tickets/BOARD.md", "# Board\n## TO DO\n### pickle\n")
		}, true, "missing from the board"},
		{"history mismatch", func(t *testing.T, root string) {
			mk(t, root, "tickets/1-to-do/T-001-foo.md", ticketFile("T-001", "pickle", "[]", "high", "DONE"))
		}, true, "last History status is DONE"},
		{"wip breach", func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-010-a.md", ticketFile("T-010", "pickle", "[]", "high", "IN DEVELOPMENT"))
			mk(t, root, "tickets/3-in-development/T-011-b.md", ticketFile("T-011", "pickle", "[]", "high", "IN DEVELOPMENT"))
			mk(t, root, "tickets/BOARD.md", goodBoard+
				"## IN DEVELOPMENT\n### pickle\n| id | t |\n|---|---|\n| T-010 | a |\n| T-011 | b |\n")
		}, true, "in 3-in-development (limit 1)"},
		{"in-dev dep not done", func(t *testing.T, root string) {
			mk(t, root, "tickets/3-in-development/T-002-bar.md", ticketFile("T-002", "pickle", "[T-001]", "high", "IN DEVELOPMENT"))
			mk(t, root, "tickets/BOARD.md", goodBoard+
				"## IN DEVELOPMENT\n### pickle\n| id | t |\n|---|---|\n| T-002 | bar |\n")
		}, true, "dependency T-001 is in 1-to-do"},
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

func containsAny(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
