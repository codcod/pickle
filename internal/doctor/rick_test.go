package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

// rickFixture is installFixture with rick enabled on the sole child ("demo",
// at ".") — the config.Save round-trip a real `pickle project add`-adjacent
// flow would produce, not a struct built by hand.
func rickFixture(t *testing.T) string {
	t.Helper()
	root := installFixture(t)
	cfgPath := filepath.Join(root, config.FileName)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.Projects[0].Rick = true
	if err := cfg.Save(""); err != nil {
		t.Fatalf("cfg.Save: %v", err)
	}
	return root
}

// stubRick writes a fake `rick` into dir answering `status --json` with
// stdout and rc, and puts dir first on PATH ahead of the real PATH — a bare
// t.Setenv("PATH", dir) would strip out the shell builtins/utilities (cat)
// the script body itself needs (internal/rickstatus's own test hit this
// exact trap first).
func stubRick(t *testing.T, dir, stdout string, rc int) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\ncat <<'RICKEOF'\n%s\nRICKEOF\nexit %d\n", stdout, rc)
	if err := os.WriteFile(filepath.Join(dir, "rick"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const rickFixtureJSON = `{
  "schemaVersion": 2,
  "workflow": {
    "tickets": [
      {"id": "T-001", "artifacts": [
        {"path": "docs/specs/T-001/solution-design-2026-06-14-x.md", "kind": "solution-design", "status": "approved", "date": "2026-06-14"},
        {"path": "docs/specs/T-001/task-2026-06-10-x.md", "kind": "task", "status": "draft"}
      ]}
    ]
  }
}`

// TestCheckRickDisabledIsSilent: a child that has not opted in (Rick: false,
// installFixture's default) gets no rick-related line at all, in any of
// Errors/Warnings/Passed — matching checkClaudeView's silent-when-absent
// shape for the overwhelmingly common case.
func TestCheckRickDisabledIsSilent(t *testing.T) {
	root := installFixture(t)
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	for _, lines := range [][]string{res.Errors, res.Warnings, res.Passed} {
		if hasWarnContaining(lines, "rick") {
			t.Fatalf("expected no rick-related line for a child with Rick: false, got: %v", lines)
		}
	}
}

// TestCheckRickEnabledWorkingStubIsOnePassedLine: a rick-enabled child with a
// working stub gets exactly one Passed line naming the right counts, and no
// Errors/Warnings.
func TestCheckRickEnabledWorkingStubIsOnePassedLine(t *testing.T) {
	root := rickFixture(t)
	bin := t.TempDir()
	stubRick(t, bin, rickFixtureJSON, 0)

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors, got: %v", res.Errors)
	}
	if hasWarnContaining(res.Warnings, "rick") {
		t.Fatalf("expected no rick-related warning, got: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, `child "demo": rick status ok (1 ticket(s), 2 artifact(s))`) {
		t.Fatalf("expected a passed line with the right counts, got: %v", res.Passed)
	}
}

// TestCheckRickEnabledUnavailableStaysFailOpen (T-075's central invariant,
// made a mechanical assertion rather than read off the code): a rick-enabled
// child with no `rick` on PATH still produces zero Errors/Warnings, and one
// Passed line naming the reason.
func TestCheckRickEnabledUnavailableStaysFailOpen(t *testing.T) {
	root := rickFixture(t)
	// No stub written; PATH is whatever the test process already has, which
	// installFixture/rickFixture never touch — if a real `rick` happens to be
	// on the developer's PATH already, force a guaranteed miss by pointing
	// PATH at an empty directory instead, same as
	// internal/rickstatus.TestQueryMissingBinaryFailsOpenWithoutPanic.
	t.Setenv("PATH", t.TempDir())

	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("expected no errors when rick is unavailable, got: %v", res.Errors)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("expected no warnings when rick is unavailable (fail-open), got: %v", res.Warnings)
	}
	if !hasPassedContaining(res.Passed, `child "demo": rick interop enabled but unavailable`) {
		t.Fatalf("expected a passed line naming the child and the fail-open reason, got: %v", res.Passed)
	}
}
