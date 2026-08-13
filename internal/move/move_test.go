package move

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/audit"
	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/ticket"
)

// newProject lays a fresh install into a temp dir and returns (root, cfg).
func newProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	root := t.TempDir()
	payload := os.DirFS(filepath.Join("..", ".."))
	if _, err := install.Run(payload, root, "test", install.Options{
		ProjectName: "demo", ProjectPath: ".", Agents: install.Agents{},
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return root, cfg
}

// readyGateBody is a minimal ## Implementation Plan body satisfying every
// Blocking Requirement in brine's readyGate (T-081: the plan section itself,
// plus its seven READY-gate ### stems) — injected by newTicketFull in place
// of Scaffold's own placeholder, so every existing test that walks a ticket
// past READY keeps working under the new gate without individually pasting a
// skeleton into each one. TestReadyGateRefusesIncompletePlan below
// deliberately does NOT go through newTicketFull, so it still exercises a raw
// Scaffold's empty placeholder — the exact shape of the defect this ticket
// closes.
const readyGateBody = `### 0. Feature branch (mandatory)

feat/T-NNN-demo

### Prerequisite gate (hard)

none

### Confirmed design decisions

d1

### Tasks

t1

### Acceptance test

just test

### Docs update

no user-facing surface

### Finish (mandatory)

summary + suggested commit message`

// newTicketFull writes a TO DO ticket directly (bypassing the CLI) with the given
// depends-on and spawned-by, then regenerates the board (the path ticket-new uses).
// Its Implementation Plan is pre-filled with readyGateBody so a caller can walk
// the ticket straight through READY without the gate table (T-081) refusing it.
//
// Lineage goes in typed, because ticket.Scaffold takes a spawnedBy parameter;
// only depends-on needs the string rewrite, since Scaffold hardcodes
// `depends-on: []`. (internal/audit's fixtures rewrite both — but they are raw
// string literals, not Scaffold output.)
func newTicketFull(t *testing.T, root string, cfg *config.Config, id, title string, deps, spawnedBy []string) {
	t.Helper()
	body := ticket.Scaffold(id, title, "demo", "medium", "medium", "M", spawnedBy, "")
	body = strings.Replace(body,
		"<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->",
		readyGateBody, 1)
	if len(deps) > 0 {
		body = strings.Replace(body, "depends-on: []", "depends-on: ["+strings.Join(deps, ", ")+"]", 1)
	}
	dst := filepath.Join(root, "tickets", "1-to-do", id+"-"+ticket.Slugify(title)+".md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := board.Regenerate(flow.ForName(cfg.FlowName()), root, cfg); err != nil {
		t.Fatal(err)
	}
}

// newTicket writes a TO DO ticket with the given depends-on and no lineage.
func newTicket(t *testing.T, root string, cfg *config.Config, id, title string, deps ...string) {
	t.Helper()
	newTicketFull(t, root, cfg, id, title, deps, nil)
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
	newTicket(t, root, cfg, "T-001", "Alpha")
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
	if got := ticket.LastHistoryStatus(flow.ForName(cfg.FlowName()), string(b)); got != "DONE" {
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
	newTicket(t, root, cfg, "T-001", "Alpha")
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
	newTicket(t, root, cfg, "T-001", "Alpha")
	if _, err := Move(root, cfg, "T-001", "dropped", ""); err == nil {
		t.Fatal("expected drop without --reason to be rejected")
	}
	mustMove(t, root, cfg, "T-001", "dropped", "superseded")
	assertClean(t, root, cfg)
	// Reason lands in the DROPPED row's reason cell, derived from the History
	// line the move wrote (D3) — not carried over from any previous board.
	board, _ := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if !strings.Contains(string(board), "| T-001 | Alpha | superseded |") {
		t.Errorf("DROPPED row missing derived reason:\n%s", board)
	}
}

func TestWIPGate(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	newTicket(t, root, cfg, "T-002", "Beta")
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
	newTicket(t, root, cfg, "T-001", "Dep")
	newTicket(t, root, cfg, "T-002", "Needs", "T-001")
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

// TestSpawnedByDoesNotGatePickup is the move-side twin of the audit's
// "in-dev spawned-by parent not done" case (internal/audit/audit_test.go):
// lineage never blocks a pickup, so a ticket whose spawned-by parent is still in
// 1-to-do/ picks up cleanly. The identical tree shape with depends-on instead is
// TestDependencyGate's first rejection above — same fixture, opposite verdict.
//
// Load-bearing check: adding t.SpawnedBy to Move's pickup gate loop must make
// this test fail. Before T-029 that mutation left the whole suite green, because
// only the audit half of the guarantee was guarded.
func TestSpawnedByDoesNotGatePickup(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Parent") // stays in 1-to-do: neither done nor merged
	newTicketFull(t, root, cfg, "T-002", "Child", nil, []string{"T-001"})
	mustMove(t, root, cfg, "T-002", "ready", "")
	mustMove(t, root, cfg, "T-002", "in-development", "") // must NOT be gated
	assertClean(t, root, cfg)
}

// TestMoveIsWriteNewThenRemoveOld pins D7 (T-014·4): the updated text lands at
// the NEW path before the old file goes away, so a ticket can never be left
// recording a transition that did not happen.
func TestMoveIsWriteNewThenRemoveOld(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	mustMove(t, root, cfg, "T-001", "ready", "")

	oldPath := filepath.Join(root, "tickets", "1-to-do", "T-001-alpha.md")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old path still exists after move: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, "tickets", "2-ready", "T-001-alpha.md"))
	if err != nil {
		t.Fatalf("new path missing: %v", err)
	}
	if !strings.Contains(string(b), "TO DO → READY") {
		t.Errorf("new path missing the History transition:\n%s", b)
	}
}

// TestMoveOldPathIsSourceStatusPath is the T-091 regression: a move's Result
// carries the removed file's own path, not just the new one, so a caller
// naming both paths from the Result can never stage the add without the
// delete — the pairing is the point.
func TestMoveOldPathIsSourceStatusPath(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	res := mustMove(t, root, cfg, "T-001", "ready", "")

	wantOld := filepath.Join("tickets", "1-to-do", "T-001-alpha.md")
	if res.OldPath != wantOld {
		t.Errorf("OldPath = %q, want %q", res.OldPath, wantOld)
	}
	if _, err := os.Stat(filepath.Join(root, res.OldPath)); !os.IsNotExist(err) {
		t.Errorf("OldPath %q still exists after move: %v", res.OldPath, err)
	}
}

// TestMoveRegeneratesWIPCounts is the T-014·1 regression: a move refreshes the
// whole board, so the (n/limit) counts can never go stale.
func TestMoveRegeneratesWIPCounts(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	mustMove(t, root, cfg, "T-001", "ready", "")
	mustMove(t, root, cfg, "T-001", "in-development", "")

	b, _ := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if !strings.Contains(string(b), "### demo (1/1)") {
		t.Errorf("IN DEVELOPMENT count not refreshed to (1/1):\n%s", b)
	}

	mustMove(t, root, cfg, "T-001", "in-review", "")
	b, _ = os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	s := string(b)
	dev := s[strings.Index(s, "## IN DEVELOPMENT"):strings.Index(s, "## IN REVIEW")]
	if !strings.Contains(dev, "### demo (0/1)") {
		t.Errorf("IN DEVELOPMENT count not back to (0/1) after moving on:\n%s", dev)
	}
	if !strings.Contains(s[strings.Index(s, "## IN REVIEW"):], "### demo (1/1)") {
		t.Errorf("IN REVIEW count not refreshed to (1/1):\n%s", s)
	}
}

func TestSanitizeReason(t *testing.T) {
	if got := sanitizeReason("a → b\nc"); got != "a -> b c" {
		t.Errorf("sanitizeReason = %q", got)
	}
}

// TestReadyGateRefusesIncompletePlan is the regression for T-081's whole
// reason to exist: measured on main at refinement, a ticket scaffolded by
// `pickle ticket new` — empty Implementation Plan — moved to READY cleanly,
// printing only the ## Outcome warning. It must now be refused, leaving the
// ticket exactly where it was.
func TestReadyGateRefusesIncompletePlan(t *testing.T) {
	root, cfg := newProject(t)
	// Deliberately not newTicketFull: a raw Scaffold is exactly what
	// `pickle ticket new` produces, with the placeholder plan intact.
	body := ticket.Scaffold("T-001", "Alpha", "demo", "medium", "medium", "M", nil, "")
	dst := filepath.Join(root, "tickets", "1-to-do", "T-001-alpha.md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := board.Regenerate(flow.ForName(cfg.FlowName()), root, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := Move(root, cfg, "T-001", "ready", "plan complete"); err == nil {
		t.Fatal("expected move to READY to be refused: Implementation Plan is empty")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("ticket no longer at its original path after a refused move: %v", err)
	}
	// 1-to-do only requires ## Outcome (Advisory) — a warning, never an error.
	assertClean(t, root, cfg)
}

// TestReadyGateAcceptsCompletePlan is the positive twin: the same shape of
// ticket, this time with the seven READY-gate headings filled in
// (newTicketFull's readyGateBody), moves cleanly.
func TestReadyGateAcceptsCompletePlan(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	mustMove(t, root, cfg, "T-001", "ready", "plan complete")
	assertClean(t, root, cfg)
}

// TestReworkRequiresReviewRecorded pins the one Requirement that is Blocking
// only on 5-rework: a rework's whole scope is its recorded findings (rules
// §5), so ## Review must be substantive before the ticket may sit there.
func TestReworkRequiresReviewRecorded(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	mustMove(t, root, cfg, "T-001", "ready", "plan complete")
	mustMove(t, root, cfg, "T-001", "in-development", "")
	mustMove(t, root, cfg, "T-001", "in-review", "")

	if _, err := Move(root, cfg, "T-001", "rework", "blocking finding"); err == nil {
		t.Fatal("expected rework to be refused: ## Review is still the placeholder")
	}

	path := filepath.Join(root, "tickets", "4-in-review", "T-001-alpha.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(b), "<!-- empty until IN REVIEW -->", "N1 | blocking | fixed the thing", 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}

	mustMove(t, root, cfg, "T-001", "rework", "blocking finding")
	assertClean(t, root, cfg)
}

// TestBackToToDoNeverRefused pins the documented escape hatch (rules §2,
// T-081 decision 6): 1-to-do requires only ## Outcome (Advisory), so a
// ticket may always retreat there regardless of how incomplete its plan is
// — including one whose plan the gate had just refused to let past READY.
func TestBackToToDoNeverRefused(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	mustMove(t, root, cfg, "T-001", "ready", "plan complete")
	mustMove(t, root, cfg, "T-001", "to-do", "gate no longer holds")
	assertClean(t, root, cfg)
}
