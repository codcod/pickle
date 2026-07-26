package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/install"
)

// repoRoot is the module root, resolved before TestMain moves the process CWD.
// Nothing in this package may reach for the payload with a relative path: the
// original os.DirFS(filepath.Join("..", "..")) only worked because `go test`
// starts in internal/cli/, which is exactly what TestMain changes.
var repoRoot string

// TestMain makes CWD safety the default rather than opt-in. Commands in this
// package resolve their target by walking *up* from the process CWD
// (loadConfig -> os.Getwd -> config.Find), so a test that forgot newProject's
// t.Chdir would find pickle's own pickle.toml and mutate the real board —
// burning a global ticket id, which is never reused. Running the whole package
// from a throwaway directory means such a test fails loudly instead.
//
// TestCWDIsSandboxed proves this actually holds.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: getwd: %v\n", err)
		os.Exit(1)
	}
	repoRoot, err = filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: resolve repo root: %v\n", err)
		os.Exit(1)
	}
	// Keep the path in a named variable the cleanup below references: the
	// acceptance test's mutation C deletes the Chdir call and requires a test
	// failure, not an unused-variable compile error.
	sandbox, err := os.MkdirTemp("", "pickle-cli-sandbox")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: sandbox: %v\n", err)
		os.Exit(1)
	}
	if err := os.Chdir(sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: chdir sandbox: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	// Step out before removing it: each test's t.Chdir restored the sandbox as
	// the CWD, and deleting the directory you are standing in is sloppy.
	// os.Exit runs no deferred functions, so this cannot be a defer.
	_ = os.Chdir(repoRoot)
	_ = os.RemoveAll(sandbox)
	os.Exit(code)
}

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
// The chdir puts the test *inside* its own install: runTicketNew resolves its
// target through loadConfig() -> os.Getwd() + config.Find(wd), which walks *up*
// from the process CWD, so a command only reaches this install if the CWD is in
// it. TestMain's sandbox is the second line of defence — it makes forgetting
// this call a loud failure rather than a write into the real board — but it is
// not a substitute, because a sandboxed CWD resolves no config at all.
//
// The payload comes from repoRoot, not a relative path: TestMain has already
// moved the CWD away from internal/cli/.
//
// Because both the CWD and os.Stdout (see captureStdout) are process-global, no
// test in this package may call t.Parallel().
func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := install.Run(os.DirFS(repoRoot), root, "test", install.Options{
		ProjectName: "demo", ProjectPath: ".", Agents: install.Agents{},
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	t.Chdir(root)
	return root
}

// TestCWDIsSandboxed proves TestMain's sandbox actually holds: from the default
// process CWD no pickle.toml is discoverable, so a config-resolving command run
// without newProject fails loudly instead of writing into the real board. Delete
// TestMain's os.Chdir and this test fails — that is the point of it.
func TestCWDIsSandboxed(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if path, err := config.Find(wd); err == nil {
		t.Fatalf("default test CWD %s resolves a real config at %s — the TestMain sandbox is broken", wd, path)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
// printed. The commands in this package print their diagnostics with fmt.Printf
// to the real os.Stdout (no injectable writer), so this is the only way to
// assert them. os.Stdout is process-global — like the CWD, it forbids
// t.Parallel() in this package.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	// Restore through t.Cleanup, never by an assignment after fn(): a t.Fatal or
	// panic inside fn calls runtime.Goexit, which would skip the restore and
	// leave this pipe installed as os.Stdout for every later test in the package.
	t.Cleanup(func() { os.Stdout = orig })

	// Drain concurrently: fn writing more than the pipe buffer (~64 KiB) would
	// block forever against a reader that only starts after it returns, hanging
	// the test instead of failing it.
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()

	if err := w.Close(); err != nil { // unblocks the reader
		t.Fatalf("close pipe writer: %v", err)
	}
	out := <-done
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return out
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
	// Assert the diagnostic, not just the exit code: a bare non-zero exit would
	// also be satisfied by an unrelated audit error, so it would not pin the
	// dangling-id rule at all. The substring is internal/audit's wording, the
	// same string TestAudit/dangling_spawned-by pins.
	var code int
	out := captureStdout(t, func() {
		code = Run(nil, "test", []string{"board", "audit"})
	})
	if code == exitOK {
		t.Errorf("board audit = 0, want non-zero for a dangling spawned-by id; output:\n%s", out)
	}
	if !strings.Contains(out, "spawned-by T-404 does not exist") {
		t.Errorf("board audit did not report the dangling id:\n%s", out)
	}
}

// TestTicketNewRejectsInjectionInTitle pins T-030 decision 1: an unusable title
// is rejected before anything is written.
//
// The tree-unchanged assertion is the load-bearing half. An exit-code-only test
// would also pass on a validator that rejected *after* writing the ticket file or
// the board row — precisely the weakness T-029 found next door in
// TestTicketNewSpawnedByUnknownID — and the whole point of the fix is that
// nothing malformed reaches disk.
func TestTicketNewRejectsInjectionInTitle(t *testing.T) {
	for _, tc := range []struct{ name, title string }{
		{"injects a frontmatter key", "evil\nproject: nope"},
		{"injects an id", "x\nid: T-999"},
		{"carriage return", "a\r\nb"},
		{"frontmatter terminator after a newline", "a\n---\nb"},
		// The only case that reaches the "---" branch: the newline rule catches
		// the case above first, and a bare "---" never gets past the "-" prefix
		// guard in runTicketNew.
		{"padded frontmatter terminator", "  ---  "},
		{"whitespace only", "   "},
		{"empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) { // no t.Parallel: newProject chdirs
			root := newProject(t)
			boardPath := filepath.Join(root, "tickets", "BOARD.md")
			before, err := os.ReadFile(boardPath)
			if err != nil {
				t.Fatal(err)
			}

			if got := Run(nil, "test", []string{"ticket", "new", tc.title, "--project", "demo"}); got != exitError {
				t.Errorf("Run(ticket new %q) = %d, want %d", tc.title, got, exitError)
			}
			assertNoTickets(t, root)
			after, err := os.ReadFile(boardPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("BOARD.md changed on a rejected title %q", tc.title)
			}
		})
	}
}

// TestTicketNewRejectsMalformedSpawnedBy covers the other unvalidated input. The
// exit code and the untouched tree are asserted here; the *wording* ("is not a
// ticket id") is pinned in internal/ticket's TestParseIDList, because errf writes
// to stderr and this package has no stderr capture (captureStdout would not see
// it) — see T-031.
func TestTicketNewRejectsMalformedSpawnedBy(t *testing.T) {
	for _, tc := range []struct{ name, spawnedBy string }{
		{"not an id", "banana"},
		{"injects a frontmatter key", "T-001]\nimpact: critical"},
		{"wrong case", "t-001"},
	} {
		t.Run(tc.name, func(t *testing.T) { // no t.Parallel: newProject chdirs
			root := newProject(t)
			boardPath := filepath.Join(root, "tickets", "BOARD.md")
			before, err := os.ReadFile(boardPath)
			if err != nil {
				t.Fatal(err)
			}

			args := []string{"ticket", "new", "sneaky", "--project", "demo", "--spawned-by", tc.spawnedBy}
			if got := Run(nil, "test", args); got != exitError {
				t.Errorf("Run(--spawned-by %q) = %d, want %d", tc.spawnedBy, got, exitError)
			}
			assertNoTickets(t, root)
			after, err := os.ReadFile(boardPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("BOARD.md changed on a rejected --spawned-by %q", tc.spawnedBy)
			}
		})
	}
}

// TestTicketNewDeduplicatesSpawnedBy pins T-030 decision 3: a repeated id is
// normalised away rather than rejected or written twice.
func TestTicketNewDeduplicatesSpawnedBy(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "real parent", "--project", "demo"}); got != exitOK {
		t.Fatalf("seeding T-001: exit %d", got)
	}
	if got := Run(nil, "test", []string{"ticket", "new", "dupes", "--project", "demo", "--spawned-by", "T-001,T-001"}); got != exitOK {
		t.Fatalf("ticket new = %d, want %d", got, exitOK)
	}
	if body := readTicket(t, root, "T-002"); !strings.Contains(body, "spawned-by: [T-001]") {
		t.Errorf("want spawned-by: [T-001] (de-duplicated):\n%s", body)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit = %d, want clean (%d)", got, exitOK)
	}
}

// TestTicketNewAcceptsAwkwardButLegalTitle guards against the rejection turning
// into a character whitelist: punctuation that Slugify strips, a leading "-"
// inside the title, and a '|' (whose escaping is T-014's job at the render
// boundary, not a reason to reject input) must all still be accepted.
func TestTicketNewAcceptsAwkwardButLegalTitle(t *testing.T) {
	for _, title := range []string{
		"add --flag support (finally)",
		"### hashes ###",
		"pipe | in title",
		"tabs\tand  spaces",
	} {
		t.Run(title, func(t *testing.T) { // no t.Parallel: newProject chdirs
			root := newProject(t)
			if got := Run(nil, "test", []string{"ticket", "new", title, "--project", "demo"}); got != exitOK {
				t.Fatalf("Run(ticket new %q) = %d, want %d", title, got, exitOK)
			}
			if body := readTicket(t, root, "T-001"); !strings.Contains(body, "title: "+title) {
				t.Errorf("scaffold missing the title verbatim:\n%s", body)
			}
		})
	}
}

// assertNoTickets fails if any ticket file exists in 1-to-do/. The counterpart to
// readTicket, which cannot express absence: it t.Fatalf's when the id is missing.
func assertNoTickets(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "tickets", "1-to-do")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("%s was written despite a rejected invocation", e.Name())
		}
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
