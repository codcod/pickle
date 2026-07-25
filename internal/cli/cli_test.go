package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/install"
)

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, exitUsage},
		{"help", []string{"help"}, exitOK},
		{"version", []string{"version"}, exitOK},
		{"unknown command", []string{"frobnicate"}, exitUsage},
		// upgrade takes no flags, but must reject rather than ignore argv: a
		// silently-ignored -h/-n would perform a real, mutating upgrade.
		{"upgrade bad flag", []string{"upgrade", "--bogus"}, exitUsage},
		{"upgrade help flag", []string{"upgrade", "-h"}, exitUsage},
		{"upgrade stray arg", []string{"upgrade", "extra"}, exitUsage},
		{"uninstall bad flag", []string{"uninstall", "--bogus"}, exitUsage},
		{"board sync bad flag", []string{"board", "sync", "--bogus"}, exitUsage},
		{"board no subcommand", []string{"board"}, exitUsage},
		{"board unknown subcommand", []string{"board", "xyz"}, exitUsage},
		{"ticket new missing title", []string{"ticket", "new"}, exitUsage},
		{"ticket move missing args", []string{"ticket", "move"}, exitUsage},
		{"ticket move missing status", []string{"ticket", "move", "T-001"}, exitUsage},
		{"project no subcommand", []string{"project"}, exitUsage},
		{"project unknown subcommand", []string{"project", "xyz"}, exitUsage},
		{"project add missing args", []string{"project", "add"}, exitUsage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Run(nil, "test", tc.args); got != tc.want {
				t.Fatalf("Run(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}

// newProject installs a throwaway project into a temp dir and chdirs into it.
//
// The chdir is not optional: runTicketNew resolves its target through
// loadConfig() -> os.Getwd() + config.Find(wd), which walks *up* from the
// process CWD. Without it a `ticket new` test running in internal/cli/ would
// find pickle's own pickle.toml and write a real ticket into the real board,
// burning a global id (ids are never reused). Because CWD is process-global,
// no test using this helper may call t.Parallel().
func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := install.Run(os.DirFS(filepath.Join("..", "..")), root, "test", install.Options{
		ProjectName: "demo", ProjectPath: ".", Claude: false,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	t.Chdir(root)
	return root
}

// TestTicketNewSpawnedBy covers the --spawned-by flag end to end: the scaffold
// it writes and the audit's verdict on it.
func TestTicketNewSpawnedBy(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"default is empty", []string{"ticket", "new", "child", "--project", "demo"}, "spawned-by: []"},
		{"single parent", []string{"ticket", "new", "child", "--project", "demo", "--spawned-by", "T-001"}, "spawned-by: [T-001]"},
		{"two parents, brackets optional", []string{"ticket", "new", "child", "--project", "demo", "--spawned-by", "T-001,T-002"}, "spawned-by: [T-001, T-002]"},
	} {
		t.Run(tc.name, func(t *testing.T) { // no t.Parallel: newProject chdirs
			root := newProject(t)
			// the lineage parents must exist, or board audit rightly complains
			for _, id := range []string{"T-001", "T-002"} {
				if got := Run(nil, "test", []string{"ticket", "new", "parent " + id, "--project", "demo"}); got != exitOK {
					t.Fatalf("seeding %s: exit %d", id, got)
				}
			}
			if got := Run(nil, "test", tc.args); got != exitOK {
				t.Fatalf("Run(%v) = %d, want %d", tc.args, got, exitOK)
			}
			body := readTicket(t, root, "T-003")
			if !strings.Contains(body, tc.want) {
				t.Errorf("scaffold missing %q:\n%s", tc.want, body)
			}
			if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
				t.Fatalf("board audit = %d, want clean (%d)", got, exitOK)
			}
		})
	}
}

// TestTicketNewSpawnedByUnknownID pins decision 5: ids are passed through at
// creation (like depends-on) and it is board audit that rejects them.
func TestTicketNewSpawnedByUnknownID(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "orphan", "--project", "demo", "--spawned-by", "T-404"}); got != exitOK {
		t.Fatalf("ticket new = %d, want %d (creation does not validate ids)", got, exitOK)
	}
	if body := readTicket(t, root, "T-001"); !strings.Contains(body, "spawned-by: [T-404]") {
		t.Errorf("scaffold missing spawned-by: [T-404]:\n%s", body)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got == exitOK {
		t.Error("board audit = 0, want non-zero for a dangling spawned-by id")
	}
}

// readTicket returns the body of the T-NNN ticket in 1-to-do/.
func readTicket(t *testing.T, root, id string) string {
	t.Helper()
	dir := filepath.Join(root, "tickets", "1-to-do")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id+"-") {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
	}
	t.Fatalf("%s not found in %s", id, dir)
	return ""
}
