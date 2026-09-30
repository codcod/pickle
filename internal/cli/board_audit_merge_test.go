package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/vcs"
)

// doneTicket is a DONE ticket that passes every gate but carries no merge line.
func doneTicket(id string) string {
	plan := ""
	for _, h := range []string{"0. Feature branch (mandatory)", "Prerequisite gate (hard)", "Confirmed design decisions",
		"Tasks", "Acceptance test", "Docs update", "Finish (mandatory)"} {
		plan += "### " + h + "\n\nx\n\n"
	}
	return "---\nid: " + id + "\ntitle: t\nproject: demo\ndepends-on: []\nspawned-by: []\nimpact: low\ncomplexity: low\ncost: S\n---\n\n" +
		"# " + id + " — t\n\n## Outcome\n\nSomething observable changes.\n\n## Description\n\nd\n\n## Implementation Plan\n\n" + plan +
		"## Review\n\nno findings\n\n## History\n\n- 2026-09-01 — created (TO DO). source: chat: fixture\n" +
		"- 2026-09-02 — IN REVIEW → DONE: fixture\n"
}

// TestBoardAuditFindsMergeInGit is T-140's CLI acceptance case: a DONE ticket
// whose commit is on local main (no remote) gets the line to record printed
// under its warning; one git knows nothing about gets none; the warning count
// and exit code are unchanged.
func TestBoardAuditFindsMergeInGit(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	for _, id := range []string{"T-001", "T-002"} {
		p := filepath.Join(root, "tickets", "6-done", id+"-fixture.md")
		if err := os.WriteFile(p, []byte(doneTicket(id)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitInit(t, root, "main")
	writeAndCommit(t, root, "x.txt", "x", "feat: x (T-001)")
	if got := Run(nil, "test", []string{"board", "sync"}); got != exitOK {
		t.Fatalf("board sync = %d", got)
	}
	sha, err1 := vcs.Output(root, "rev-parse", "--short=7", "HEAD")
	date, err2 := vcs.Output(root, "log", "-1", "--format=%cs")
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}

	var code int
	out := captureStdout(t, func() { code = Run(nil, "test", []string{"board", "audit"}) })
	if code != exitOK {
		t.Fatalf("board audit = %d, want %d:\n%s", code, exitOK, out)
	}
	want := "6-done/T-001-fixture.md: DONE but has no 'MERGED'"
	i := strings.Index(out, want)
	if i < 0 {
		t.Fatalf("no T-001 warning:\n%s", out)
	}
	next := strings.SplitN(out[i:], "\n", 3)[1]
	if line := "  → found in git, record: " + date + " — merged to main (" + sha + ")"; next != line {
		t.Errorf("line under T-001's warning = %q, want %q\n%s", next, line, out)
	}
	if strings.Count(out, "→ found in git") != 1 {
		t.Errorf("want exactly one suggestion (none for T-002):\n%s", out)
	}
	if !strings.Contains(out, "0 error(s), 2 warning(s)") {
		t.Errorf("want 2 warnings:\n%s", out)
	}
}
