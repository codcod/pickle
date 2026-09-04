package rickstatus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codcod/pickle/internal/config"
)

// writeExecutable writes a shell script to path and marks it executable — a
// local copy of internal/hook's helper of the same shape (unexported there,
// so each package that needs a fake external binary on PATH keeps its own).
func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// stubRick writes a fake `rick` into dir that prints stdout and exits rc when
// invoked as `rick status --json`, and — for the "never invoked" assertion in
// TestQueryDisabledNeverInvokesRick — always touches a sentinel file first, so
// a test can prove the stub ran at all regardless of what it then printed.
func stubRick(t *testing.T, dir, sentinel, stdout string, rc int) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\ntouch %q\ncat <<'RICKEOF'\n%s\nRICKEOF\nexit %d\n", sentinel, stdout, rc)
	writeExecutable(t, filepath.Join(dir, "rick"), body)
}

// stubRickSleeping writes a fake `rick` that sleeps before answering, for the
// timeout test.
func stubRickSleeping(t *testing.T, dir string, sleep time.Duration) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\nsleep %f\necho '{}'\n", sleep.Seconds())
	writeExecutable(t, filepath.Join(dir, "rick"), body)
}

// setStubPath puts dir first on PATH so the stub `rick` resolves, while
// keeping the rest of the real PATH behind it — the stub script's own body
// uses ordinary shell builtins/utilities (touch, cat) that must still
// resolve. A bare t.Setenv("PATH", dir) would strip those out from under it
// and the script would silently no-op (mirrors internal/hook/probe_test.go's
// same fix for the identical trap).
func setStubPath(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// fixtureJSON mirrors the real shape confirmed against an actually installed
// `rick` v0.10.0 binary during this ticket's implementation (see wire.go): a
// ticket entry's identifying field is `id`, and `date` is present only when
// the artifact's own frontmatter carries one — the second artifact here has
// none, matching a real `research-*.md` fixture observed with no `date` key
// at all (not an empty string).
func fixtureJSON(schemaVersion int) string {
	return fmt.Sprintf(`{
  "schemaVersion": %d,
  "workflow": {
    "tickets": [
      {
        "id": "DR-142",
        "artifacts": [
          {"path": "docs/specs/DR-142/solution-design-2026-06-14-x.md", "kind": "solution-design", "status": "approved", "date": "2026-06-14"},
          {"path": "docs/specs/DR-142/research-2026-06-10-x.md", "kind": "other", "status": "draft"}
        ]
      }
    ]
  }
}`, schemaVersion)
}

func rickProject(t *testing.T) *config.Project {
	t.Helper()
	return &config.Project{Name: "demo", Path: ".", Rick: true}
}

func TestQuerySchemaVersion2Available(t *testing.T) {
	bin := t.TempDir()
	stubRick(t, bin, filepath.Join(bin, "sentinel"), fixtureJSON(2), 0)
	setStubPath(t, bin)

	rep := Query(t.TempDir(), rickProject(t))
	if !rep.Available {
		t.Fatalf("Query() = %+v, want Available", rep)
	}
	artifacts := rep.For("DR-142")
	if len(artifacts) != 2 {
		t.Fatalf("For(%q) = %v, want 2 artifacts", "DR-142", artifacts)
	}
	if artifacts[0].Kind != "solution-design" || artifacts[0].Status != "approved" {
		t.Errorf("artifacts[0] = %+v, want kind solution-design, status approved", artifacts[0])
	}
	if artifacts[1].Kind != "other" || artifacts[1].Status != "draft" {
		t.Errorf("artifacts[1] = %+v, want kind other, status draft", artifacts[1])
	}
	if artifacts[1].Date != "" {
		t.Errorf("artifacts[1].Date = %q, want empty (no date key in the fixture, matching a real dateless artifact)", artifacts[1].Date)
	}
}

func TestQueryUnrecognisedSchemaVersionFailsOpen(t *testing.T) {
	bin := t.TempDir()
	stubRick(t, bin, filepath.Join(bin, "sentinel"), fixtureJSON(3), 0)
	setStubPath(t, bin)

	rep := Query(t.TempDir(), rickProject(t))
	if rep.Available {
		t.Fatalf("Query() = %+v, want not Available for schemaVersion 3", rep)
	}
	if !strings.Contains(rep.Reason, "schema 3") {
		t.Errorf("Reason = %q, want it to name the mismatched schema", rep.Reason)
	}
}

func TestQueryNonZeroExitFailsOpen(t *testing.T) {
	bin := t.TempDir()
	stubRick(t, bin, filepath.Join(bin, "sentinel"), "not json", 1)
	setStubPath(t, bin)

	rep := Query(t.TempDir(), rickProject(t))
	if rep.Available {
		t.Fatalf("Query() = %+v, want not Available for a non-zero exit", rep)
	}
	if rep.Reason == "" {
		t.Error("Reason is empty, want it to name the failure")
	}
}

func TestQueryMissingBinaryFailsOpenWithoutPanic(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty — nothing resolves, not even rick

	rep := Query(t.TempDir(), rickProject(t))
	if rep.Available {
		t.Fatalf("Query() = %+v, want not Available with no rick on PATH", rep)
	}
	if rep.Reason == "" {
		t.Error("Reason is empty, want it to say rick was not found")
	}
}

func TestQueryTimesOut(t *testing.T) {
	orig := rickTimeout
	rickTimeout = 20 * time.Millisecond
	t.Cleanup(func() { rickTimeout = orig })

	bin := t.TempDir()
	stubRickSleeping(t, bin, 2*time.Second)
	setStubPath(t, bin)

	rep := Query(t.TempDir(), rickProject(t))
	if rep.Available {
		t.Fatalf("Query() = %+v, want not Available when rick times out", rep)
	}
}

func TestQueryDisabledNeverInvokesRick(t *testing.T) {
	bin := t.TempDir()
	sentinel := filepath.Join(bin, "sentinel")
	stubRick(t, bin, sentinel, fixtureJSON(2), 0)
	setStubPath(t, bin)

	p := &config.Project{Name: "demo", Path: ".", Rick: false}
	rep := Query(t.TempDir(), p)
	if rep.Available {
		t.Fatalf("Query() = %+v, want not Available for a project with Rick: false", rep)
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("the rick stub was invoked despite Rick: false — Query must not shell out at all")
	}
}

func TestQueryNilProject(t *testing.T) {
	rep := Query(t.TempDir(), nil)
	if rep.Available {
		t.Fatalf("Query(nil) = %+v, want not Available", rep)
	}
}

func TestReportForUnknownTicketIsNil(t *testing.T) {
	var rep Report
	if got := rep.For("DR-999"); got != nil {
		t.Errorf("For() on a zero Report = %v, want nil", got)
	}
}

// TestQueryAgainstRealRickBinary is a bonus regression guard, not part of
// this ticket's required acceptance test (which is deliberately
// fixture-only — this workspace has no checkout of sdlc-cli to depend on):
// skipped whenever a real `rick` is not on PATH, and exercised against one
// when it happens to be (as it was during this ticket's own implementation —
// `ig/uk/rick` v0.10.0, `github.com/ig-private/ai-sdlc/sdlc-cli`). That
// binary reports schemaVersion 1, so this pins the fail-open behaviour T-076
// decision 5 exists for: an older real rick must degrade to Available: false
// naming the mismatch, never a crash or a best-effort parse.
func TestQueryAgainstRealRickBinary(t *testing.T) {
	if _, err := exec.LookPath("rick"); err != nil {
		t.Skip("rick not on PATH")
	}

	child := t.TempDir()
	specs := filepath.Join(child, "docs", "specs", "DR-1")
	if err := os.MkdirAll(specs, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := "---\nstatus: approved\ndate: 2026-06-14\n---\n# Solution design\n"
	if err := os.WriteFile(filepath.Join(specs, "solution-design-2026-06-14-x.md"), []byte(artifact), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := Query(child, &config.Project{Name: "demo", Path: ".", Rick: true})
	if rep.Available {
		t.Fatalf("Query() against a real rick = %+v, want not Available (this build only accepts schemaVersion %d)", rep, SchemaVersion)
	}
	if !strings.Contains(rep.Reason, "schema") {
		t.Errorf("Reason = %q, want it to name the schema mismatch", rep.Reason)
	}
}
