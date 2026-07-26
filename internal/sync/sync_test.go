package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/ticket"
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

// addTODO writes a TO DO ticket and regenerates the board (the path ticket-new uses).
func addTODO(t *testing.T, root string, cfg *config.Config, id, title, impact string) {
	t.Helper()
	body := ticket.Scaffold(id, title, "demo", impact, "medium", "M", nil)
	dst := filepath.Join(root, "tickets", "1-to-do", id+"-"+ticket.Slugify(title)+".md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := board.Regenerate(root, cfg); err != nil {
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
	addTODO(t, root, cfg, "T-001", "Alpha", "high")
	addTODO(t, root, cfg, "T-002", "Beta", "medium")

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
	addTODO(t, root, cfg, "T-001", "Alpha", "high")
	addTODO(t, root, cfg, "T-002", "Beta", "medium")
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

// TestSyncReplacesHandProse is the T-037 reproduction, flipped: the board is a
// generated artifact (T-044), so hand-written prose inside it is *replaced* by
// regeneration — that is now the contract — while tickets/NOTES.md is never
// touched.
func TestSyncReplacesHandProse(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, cfg, "T-001", "Alpha", "high")

	notesPath := filepath.Join(root, "tickets", "NOTES.md")
	if err := os.WriteFile(notesPath, []byte("# Notes\n\nHAND-WRITTEN-NOTE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := readBoard(t, root)
	b = strings.Replace(b, "# Board", "# Board\n\nHAND-WRITTEN-PROSE", 1)
	b += "\n## Dependency chain\n\n- hand-written appendix\n"
	writeBoardFile(t, root, b)

	mustSync(t, root, cfg, false)
	assertClean(t, root, cfg)

	got := readBoard(t, root)
	if strings.Contains(got, "HAND-WRITTEN-PROSE") || strings.Contains(got, "hand-written appendix") {
		t.Error("hand-written prose survived regeneration — the board must be a pure render")
	}
	notes, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notes), "HAND-WRITTEN-NOTE") {
		t.Error("sync touched tickets/NOTES.md")
	}
}

// TestSyncDateOnlyDifferenceIsNotDrift: a board differing only in the
// `Last updated:` date is in sync — no write, no Changed.
func TestSyncDateOnlyDifferenceIsNotDrift(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, cfg, "T-001", "Alpha", "high")
	mustSync(t, root, cfg, false)

	b := readBoard(t, root)
	re := strings.SplitN(b, "Last updated: ", 2)
	if len(re) != 2 {
		t.Fatalf("no Last updated line:\n%s", b)
	}
	writeBoardFile(t, root, strings.Replace(b, "Last updated: ", "Last updated: 1999-01-01 was ", 1))

	res := mustSync(t, root, cfg, false)
	if res.Changed {
		t.Errorf("date-only difference reported as drift; summary=%v", res.Summary)
	}
}

func TestSyncDryRunReportsWithoutWriting(t *testing.T) {
	root, cfg := newProject(t)
	addTODO(t, root, cfg, "T-001", "Alpha", "high")
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
