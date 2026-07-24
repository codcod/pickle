package move

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pickle/internal/audit"
	"pickle/internal/board"
	"pickle/internal/config"
	"pickle/internal/install"
	"pickle/internal/ticket"
)

// newProject lays a fresh install into a temp dir and returns (root, cfg).
func newProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	root := t.TempDir()
	payload := os.DirFS(filepath.Join("..", ".."))
	if _, err := install.Run(payload, root, "test", install.Options{
		ProjectName: "demo", ProjectPath: ".", Claude: false,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return root, cfg
}

// newTicket writes a TO DO ticket directly (bypassing the CLI) with the given
// depends-on, and returns its id.
func newTicket(t *testing.T, root, id, title string, deps ...string) {
	t.Helper()
	body := ticket.Scaffold(id, title, "demo", "medium", "medium", "M")
	if len(deps) > 0 {
		body = strings.Replace(body, "depends-on: []", "depends-on: ["+strings.Join(deps, ", ")+"]", 1)
	}
	dst := filepath.Join(root, "tickets", "1-to-do", id+"-"+ticket.Slugify(title)+".md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Register on the board so audit stays clean (same path ticket-new uses).
	bp := filepath.Join(root, "tickets", "BOARD.md")
	if err := board.AddTODORow(bp, "demo", id, title, "medium", "medium", "M"); err != nil {
		t.Fatal(err)
	}
}

func assertClean(t *testing.T, root string, cfg *config.Config) {
	t.Helper()
	if a := audit.Audit(root, cfg); len(a.Errors) > 0 {
		t.Fatalf("audit errors: %v", a.Errors)
	}
}

func mustMove(t *testing.T, root string, cfg *config.Config, id, status, reason string) Result {
	t.Helper()
	res, err := Move(root, cfg, id, status, reason)
	if err != nil {
		t.Fatalf("Move(%s -> %s): %v", id, status, err)
	}
	return res
}

func TestForwardWalkStaysClean(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, "T-001", "Alpha")
	assertClean(t, root, cfg)

	for _, step := range []struct{ status, reason string }{
		{"ready", ""},
		{"in-development", ""},
		{"in-review", ""},
		{"done", ""},
	} {
		mustMove(t, root, cfg, "T-001", step.status, step.reason)
		assertClean(t, root, cfg)
	}

	// File landed in 6-done and its History last status agrees.
	path := filepath.Join(root, "tickets", "6-done", "T-001-alpha.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected ticket in 6-done: %v", err)
	}
	if got := ticket.LastHistoryStatus(string(b)); got != "DONE" {
		t.Errorf("last History status = %q, want DONE", got)
	}
	// DONE board row carries the publish-gated default.
	board, _ := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if !strings.Contains(string(board), "publish-gated") {
		t.Error("DONE row missing publish-gated default")
	}
}

func TestIllegalTransitionRejected(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, "T-001", "Alpha")
	if _, err := Move(root, cfg, "T-001", "done", ""); err == nil {
		t.Fatal("expected illegal transition to-do -> done to be rejected")
	}
	if _, err := Move(root, cfg, "T-001", "to-do", ""); err == nil {
		t.Fatal("expected same-status move to be rejected")
	}
	assertClean(t, root, cfg)
}

func TestReasonRequired(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, "T-001", "Alpha")
	if _, err := Move(root, cfg, "T-001", "dropped", ""); err == nil {
		t.Fatal("expected drop without --reason to be rejected")
	}
	mustMove(t, root, cfg, "T-001", "dropped", "superseded")
	assertClean(t, root, cfg)
	// Reason recorded in the DROPPED row.
	board, _ := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if !strings.Contains(string(board), "superseded") {
		t.Error("DROPPED row missing reason")
	}
}

func TestWIPGate(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, "T-001", "Alpha")
	newTicket(t, root, "T-002", "Beta")
	mustMove(t, root, cfg, "T-001", "ready", "")
	mustMove(t, root, cfg, "T-001", "in-development", "")
	mustMove(t, root, cfg, "T-002", "ready", "")
	if _, err := Move(root, cfg, "T-002", "in-development", ""); err == nil {
		t.Fatal("expected WIP gate to reject a second in-development ticket")
	}
	assertClean(t, root, cfg)
}

func TestDependencyGate(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, "T-001", "Dep")
	newTicket(t, root, "T-002", "Needs", "T-001")
	mustMove(t, root, cfg, "T-002", "ready", "")

	// T-001 not done -> pickup rejected.
	if _, err := Move(root, cfg, "T-002", "in-development", ""); err == nil {
		t.Fatal("expected dependency gate to reject pickup while T-001 is not done")
	}

	// Take T-001 to done, but without a merge line -> still rejected.
	mustMove(t, root, cfg, "T-001", "ready", "")
	mustMove(t, root, cfg, "T-001", "in-development", "")
	mustMove(t, root, cfg, "T-001", "in-review", "")
	mustMove(t, root, cfg, "T-001", "done", "")
	if _, err := Move(root, cfg, "T-002", "in-development", ""); err == nil {
		t.Fatal("expected pickup rejected while dependency lacks a MERGED line")
	}

	// Record the merge, then pickup succeeds.
	donePath := filepath.Join(root, "tickets", "6-done", "T-001-dep.md")
	b, _ := os.ReadFile(donePath)
	os.WriteFile(donePath, append(b, []byte("- 2026-07-23 — merged to main (abc1234)\n")...), 0o644)
	mustMove(t, root, cfg, "T-002", "in-development", "")
	assertClean(t, root, cfg)
}

func TestSanitizeReason(t *testing.T) {
	if got := sanitizeReason("a → b\nc"); got != "a -> b c" {
		t.Errorf("sanitizeReason = %q", got)
	}
}
