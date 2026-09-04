package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/rickstatus"
)

// writeExecutable and setStubPath mirror internal/rickstatus's own test
// helpers of the same shape (each package that needs a fake `rick` on PATH
// keeps its own copy, per that package's doc comment).
func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setStubPath(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// stubRick writes a fake `rick` that answers `status --json` with stdout.
func stubRick(t *testing.T, dir, stdout string) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\ncat <<'RICKEOF'\n%s\nRICKEOF\n", stdout)
	writeExecutable(t, filepath.Join(dir, "rick"), body)
}

// stubRickSentinel writes a fake `rick` that touches sentinel before
// answering, so a test can prove it was never invoked (T-076's own
// technique, reused here for resolveArtifact's whitelist guard).
func stubRickSentinel(t *testing.T, dir, sentinel string) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\ntouch %q\necho '{}'\n", sentinel)
	writeExecutable(t, filepath.Join(dir, "rick"), body)
}

func rickTestCfg() *config.Config {
	return &config.Config{Projects: []config.Project{
		{Name: "demo", Path: ".", Rick: true},
	}}
}

func fixtureRickJSON() string {
	return `{
  "schemaVersion": 2,
  "workflow": {
    "tickets": [
      {
        "id": "T-042",
        "artifacts": [
          {"path": "docs/specs/T-042/solution-design-2026-06-14-x.md", "kind": "solution-design", "status": "approved", "date": "2026-06-14"},
          {"path": "docs/specs/T-042/solution-design-2026-06-10-x.md", "kind": "solution-design", "status": "draft", "date": "2026-06-10"},
          {"path": "docs/specs/T-042/research-2026-06-01-x.md", "kind": "research", "status": "draft"}
        ]
      }
    ]
  }
}`
}

// --- buildArtifacts ----------------------------------------------------

func TestBuildArtifactsSingleInstanceNeverFlagged(t *testing.T) {
	rep := rickstatus.Report{Available: true, Tickets: map[string][]rickstatus.Artifact{
		"T-042": {{Path: "docs/specs/T-042/research-2026-06-10-x.md", Kind: "research", Status: "draft"}},
	}}
	views := buildArtifacts(rep, "T-042", "")
	if len(views) != 1 {
		t.Fatalf("len = %d, want 1", len(views))
	}
	if views[0].Duplicate || views[0].Mismatch {
		t.Errorf("single instance flagged: %+v", views[0])
	}
	if !views[0].Effective {
		t.Error("a lone instance must be Effective")
	}
}

func TestBuildArtifactsNewerFilenameDateIsEffective(t *testing.T) {
	rep := rickstatus.Report{Available: true, Tickets: map[string][]rickstatus.Artifact{
		"T-042": {
			{Path: "docs/specs/T-042/solution-design-2026-06-10-x.md", Kind: "solution-design", Status: "draft"},
			{Path: "docs/specs/T-042/solution-design-2026-06-14-x.md", Kind: "solution-design", Status: "approved"},
		},
	}}
	views := buildArtifacts(rep, "T-042", "")
	if len(views) != 2 {
		t.Fatalf("len = %d, want 2", len(views))
	}
	var effective ArtifactView
	for _, v := range views {
		if !v.Duplicate {
			t.Errorf("%+v: want Duplicate true for a two-instance kind", v)
		}
		if v.Effective {
			effective = v
		}
	}
	if effective.Name != "solution-design-2026-06-14-x.md" {
		t.Errorf("effective = %q, want the newer filename date", effective.Name)
	}
}

func TestBuildArtifactsEffectiveNotApprovedIsMismatch(t *testing.T) {
	rep := rickstatus.Report{Available: true, Tickets: map[string][]rickstatus.Artifact{
		"T-042": {
			{Path: "docs/specs/T-042/solution-design-2026-06-10-x.md", Kind: "solution-design", Status: "approved"},
			{Path: "docs/specs/T-042/solution-design-2026-06-14-x.md", Kind: "solution-design", Status: "draft"},
		},
	}}
	for _, v := range buildArtifacts(rep, "T-042", "") {
		if v.Effective && !v.Mismatch {
			t.Errorf("effective non-approved instance not flagged Mismatch: %+v", v)
		}
		if !v.Effective && v.Mismatch {
			t.Errorf("non-effective instance flagged Mismatch: %+v", v)
		}
	}
}

func TestBuildArtifactsEffectiveApprovedIsNotMismatch(t *testing.T) {
	rep := rickstatus.Report{Available: true, Tickets: map[string][]rickstatus.Artifact{
		"T-042": {
			{Path: "docs/specs/T-042/solution-design-2026-06-10-x.md", Kind: "solution-design", Status: "draft"},
			{Path: "docs/specs/T-042/solution-design-2026-06-14-x.md", Kind: "solution-design", Status: "approved"},
		},
	}}
	for _, v := range buildArtifacts(rep, "T-042", "") {
		if v.Effective && v.Mismatch {
			t.Errorf("effective approved instance flagged Mismatch: %+v", v)
		}
	}
}

// --- resolveArtifact -----------------------------------------------------

func TestResolveArtifactWhitelist(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	stubRick(t, bin, fixtureRickJSON())
	setStubPath(t, bin)

	path, ok := resolveArtifact(rickTestCfg(), root, "T-042", "solution-design-2026-06-14-x.md")
	if !ok {
		t.Fatal("resolveArtifact: want ok for a whitelisted name")
	}
	want := filepath.Join(root, "docs/specs/T-042/solution-design-2026-06-14-x.md")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestResolveArtifactRejectsNameNotInWhitelistEvenIfFileExists(t *testing.T) {
	root := t.TempDir()
	bin := t.TempDir()
	stubRick(t, bin, fixtureRickJSON())
	setStubPath(t, bin)

	specsDir := filepath.Join(root, "docs", "specs", "T-042")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "sneaky.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, ok := resolveArtifact(rickTestCfg(), root, "T-042", "sneaky.md"); ok {
		t.Error("resolveArtifact: want rejection for a name outside rick's own report, even though it exists on disk")
	}
}

func TestResolveArtifactRejectsPathShapedNameWithoutInvokingRick(t *testing.T) {
	bin := t.TempDir()
	sentinel := filepath.Join(bin, "sentinel")
	stubRickSentinel(t, bin, sentinel)
	setStubPath(t, bin)

	for _, name := range []string{"../secret", "a/b", "..", "a/../b"} {
		if _, ok := resolveArtifact(rickTestCfg(), t.TempDir(), "T-042", name); ok {
			t.Errorf("resolveArtifact(%q) = ok, want rejected", name)
		}
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("resolveArtifact invoked rick despite a path-shaped name — the whitelist guard must reject before querying")
	}
}
