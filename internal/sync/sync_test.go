package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pickle/internal/audit"
	"pickle/internal/board"
	"pickle/internal/config"
	"pickle/internal/install"
	"pickle/internal/move"
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

// addTODO writes a TO DO ticket + its board row (the path ticket-new uses).
func addTODO(t *testing.T, root, id, title, impact string) {
	t.Helper()
	body := ticket.Scaffold(id, title, "demo", impact, "medium", "M")
	dst := filepath.Join(root, "tickets", "1-to-do", id+"-"+ticket.Slugify(title)+".md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bp := filepath.Join(root, "tickets", "BOARD.md")
	if err := board.AddTODORow(bp, "demo", id, title, impact, "medium", "M"); err != nil {
		t.Fatal(err)
	}
}

func boardPath(root string) string { return filepath.Join(root, "tickets", "BOARD.md") }
func readBoard(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(boardPath(root))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func writeBoardFile(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(boardPath(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
func mustSync(t *testing.T, root string, cfg *config.Config, dryRun bool) Result {
	t.Helper()
	res, err := Sync(root, cfg, dryRun)
	if err != nil {
		t.Fatalf("Sync(dryRun=%v): %v", dryRun, err)
	}
	return res
}
func assertClean(t *testing.T, root string, cfg *config.Config) {
	t.Helper()
	if a := audit.Audit(root, cfg); len(a.Errors) > 0 {
		t.Fatalf("audit errors: %v", a.Errors)
	}
}

func TestSyncIsIdempotent(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")
	addTODO(t, root, "T-002", "Beta", "medium")

	mustSync(t, root, cfg, false) // canonicalise
	first := readBoard(t, root)

	res := mustSync(t, root, cfg, false)
	if res.Changed {
		t.Errorf("second sync reported Changed; summary=%v", res.Summary)
	}
	if readBoard(t, root) != first {
		t.Error("second sync produced different bytes (not idempotent)")
	}
	assertClean(t, root, cfg)
}

func TestSyncRepairsDrift(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")
	addTODO(t, root, "T-002", "Beta", "medium")
	mustSync(t, root, cfg, false)

	// Corrupt: drop T-002's row, mangle a WIP count, add an orphan row.
	b := readBoard(t, root)
	var kept []string
	for _, ln := range strings.Split(b, "\n") {
		if strings.Contains(ln, "| T-002 |") {
			continue
		}
		kept = append(kept, ln)
	}
	b = strings.ReplaceAll(strings.Join(kept, "\n"), "(0/1)", "(9/1)")
	// Inject an orphan row *inside* the TO DO section (before T-001's row).
	b = strings.Replace(b, "| T-001 |", "| T-999 | ghost | low | low | S | [] |\n| T-001 |", 1)
	writeBoardFile(t, root, b)

	if a := audit.Audit(root, cfg); len(a.Errors) == 0 {
		t.Fatal("expected the corrupted board to fail audit")
	}

	res := mustSync(t, root, cfg, false)
	if !res.Changed {
		t.Error("expected Changed on a corrupted board")
	}
	assertClean(t, root, cfg)

	got := readBoard(t, root)
	if !strings.Contains(got, "| T-002 |") {
		t.Error("dropped T-002 row not restored")
	}
	if strings.Contains(got, "T-999") {
		t.Error("orphan T-999 row not removed")
	}
	if strings.Contains(got, "(9/1)") {
		t.Error("bad WIP count not refreshed")
	}
}

func TestSyncPreservesHumanCells(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")
	addTODO(t, root, "T-002", "Beta", "medium")
	// Walk T-001 to DONE (no deps).
	for _, s := range []string{"ready", "in-development", "in-review", "done"} {
		if _, err := move.Move(root, cfg, "T-001", s, ""); err != nil {
			t.Fatalf("move T-001 -> %s: %v", s, err)
		}
	}
	// Hand-author the merged cell, and separately corrupt a WIP count.
	b := readBoard(t, root)
	b = strings.ReplaceAll(b, "no — publish-gated", "yes — merged (deadbee)")
	b = strings.ReplaceAll(b, "(0/1)", "(7/1)")
	writeBoardFile(t, root, b)

	mustSync(t, root, cfg, false)
	assertClean(t, root, cfg)

	got := readBoard(t, root)
	if !strings.Contains(got, "yes — merged (deadbee)") {
		t.Error("human-authored merged cell was clobbered (D1 violation)")
	}
	if strings.Contains(got, "(7/1)") {
		t.Error("WIP count not refreshed alongside cell preservation")
	}
}

func TestSyncPreservesPreambleAndAppendix(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")

	// Add an appendix section after the status region + a marker in the preamble.
	b := readBoard(t, root)
	b += "\n---\n\n## Dependency chain\n\n- hand-written note that must survive\n"
	b = strings.Replace(b, "# Board", "# Board\n\nCUSTOM-PREAMBLE-MARKER", 1)
	writeBoardFile(t, root, b)

	mustSync(t, root, cfg, false)
	assertClean(t, root, cfg)

	got := readBoard(t, root)
	if !strings.Contains(got, "CUSTOM-PREAMBLE-MARKER") {
		t.Error("preamble marker lost")
	}
	if !strings.Contains(got, "## Dependency chain") || !strings.Contains(got, "hand-written note that must survive") {
		t.Error("appendix lost")
	}
	if !strings.Contains(got, "Last updated: ") || !strings.Contains(got, "(board sync)") {
		t.Error("Last updated line not refreshed")
	}
}

func TestSyncTerminalMembership(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")
	for _, s := range []string{"ready", "in-development", "in-review", "done"} {
		if _, err := move.Move(root, cfg, "T-001", s, ""); err != nil {
			t.Fatalf("move: %v", err)
		}
	}
	// Simulate the DONE row ageing off the board (terminal rows may).
	b := readBoard(t, root)
	var kept []string
	for _, ln := range strings.Split(b, "\n") {
		if strings.Contains(ln, "| T-001 |") {
			continue
		}
		kept = append(kept, ln)
	}
	writeBoardFile(t, root, strings.Join(kept, "\n"))
	assertClean(t, root, cfg) // terminal-absent is allowed

	mustSync(t, root, cfg, false)
	assertClean(t, root, cfg)
	if strings.Contains(readBoard(t, root), "| T-001 |") {
		t.Error("D3: an aged-off DONE ticket must not be re-added by sync")
	}
}

func TestSyncDryRunReportsWithoutWriting(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, "T-001", "Alpha", "high")
	mustSync(t, root, cfg, false)

	// Corrupt, then dry-run.
	b := readBoard(t, root)
	writeBoardFile(t, root, strings.ReplaceAll(b, "(0/1)", "(9/1)"))
	before := readBoard(t, root)

	res := mustSync(t, root, cfg, true)
	if !res.Changed {
		t.Error("dry-run should report Changed on a drifted board")
	}
	if readBoard(t, root) != before {
		t.Error("dry-run must not write")
	}
}
