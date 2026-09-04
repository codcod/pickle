package ticketset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/testutil"
	"github.com/codcod/pickle/internal/ticket"
)

// newProject lays a fresh install into a temp dir and returns (root, cfg) —
// same shape as internal/move's own helper of the same name (each package
// keeps its own copy rather than sharing test-only code across packages).
func newProject(t *testing.T) (string, *config.Config) {
	t.Helper()
	root := t.TempDir()
	payload := os.DirFS(testutil.RepoRoot())
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

// newTicket writes a well-formed TO DO ticket via ticket.Scaffold (the same
// writer `pickle ticket new` uses) and regenerates the board.
func newTicket(t *testing.T, root string, cfg *config.Config, id, title string) {
	t.Helper()
	body := ticket.Scaffold(id, title, "demo", "medium", "medium", "M", nil, "")
	dst := filepath.Join(root, "tickets", "1-to-do", id+"-"+ticket.Slugify(title)+".md")
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := board.Regenerate(flow.ForName(cfg.FlowName()), root, cfg); err != nil {
		t.Fatal(err)
	}
}

func readTicket(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSetGradeHappyPath(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")

	res, err := Set(root, cfg, "T-001", "impact", "high")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if res.Field != "impact" || res.Old != "medium" || res.New != "high" {
		t.Errorf("Result = %+v, want {impact medium high ...}", res)
	}
	if res.Path != filepath.Join("tickets", "1-to-do", "T-001-alpha.md") {
		t.Errorf("Result.Path = %q", res.Path)
	}

	body := readTicket(t, root, res.Path)
	if !strings.Contains(body, "impact: high") {
		t.Errorf("file not updated:\n%s", body)
	}
	boardBody := readTicket(t, root, filepath.Join("tickets", "BOARD.md"))
	if !strings.Contains(boardBody, "| T-001 | Alpha | high |") {
		t.Errorf("BOARD.md not regenerated with the new grade:\n%s", boardBody)
	}
}

func TestSetTitleHappyPath(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")

	res, err := Set(root, cfg, "T-001", "title", "Renamed")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if res.Old != "Alpha" || res.New != "Renamed" {
		t.Errorf("Result = %+v, want Old=Alpha New=Renamed", res)
	}
	body := readTicket(t, root, res.Path)
	if !strings.Contains(body, "title: Renamed") || !strings.Contains(body, "# T-001 — Renamed") {
		t.Errorf("title/H1 not both updated:\n%s", body)
	}
	// The filename is untouched (T-102 decision 4: no rename on a title edit).
	if _, err := os.Stat(filepath.Join(root, "tickets", "1-to-do", "T-001-alpha.md")); err != nil {
		t.Errorf("original filename gone: %v", err)
	}
}

func TestSetFamilyHappyPath(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	newTicket(t, root, cfg, "T-002", "Umbrella")

	res, err := Set(root, cfg, "T-001", "family", "T-002")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if res.Old != "" || res.New != "T-002" {
		t.Errorf("Result = %+v, want Old=\"\" New=T-002", res)
	}
	body := readTicket(t, root, res.Path)
	if !strings.Contains(body, "family: T-002") {
		t.Errorf("family: not inserted:\n%s", body)
	}
}

func TestSetNoOpDoesNotRegenerateBoard(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	boardPath := filepath.Join(root, "tickets", "BOARD.md")
	before, err := os.ReadFile(boardPath)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Set(root, cfg, "T-001", "impact", "medium") // already medium
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if res.Old != res.New {
		t.Errorf("Result = %+v, want a no-op (Old == New)", res)
	}
	after, err := os.ReadFile(boardPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("BOARD.md changed on a no-op set")
	}
}

func TestSetRefusesOnDuplicateKey(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")
	path := filepath.Join(root, "tickets", "1-to-do", "T-001-alpha.md")
	body := readTicket(t, root, filepath.Join("tickets", "1-to-do", "T-001-alpha.md"))
	body = strings.Replace(body, "impact: medium", "impact: medium\nimpact: medium", 1)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Set(root, cfg, "T-001", "cost", "L")
	if err == nil {
		t.Fatal("expected a refusal for a ticket with a duplicate frontmatter key")
	}
	if !strings.Contains(err.Error(), "impact") {
		t.Errorf("error = %q, want it to name the duplicated key", err)
	}
}

func TestSetRefusesOnUnknownField(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")

	if _, err := Set(root, cfg, "T-001", "depends-on", "[T-002]"); err == nil {
		t.Fatal("expected a refusal for a field outside SettableFields")
	}
}

func TestSetRefusesOnUnknownID(t *testing.T) {
	root, cfg := newProject(t)
	newTicket(t, root, cfg, "T-001", "Alpha")

	if _, err := Set(root, cfg, "T-999", "impact", "high"); err == nil {
		t.Fatal("expected a refusal for a ticket id that does not exist")
	}
}
