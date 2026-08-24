package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/install"
	"github.com/codcod/pickle/internal/testutil"
)

// repoRoot is the module root, resolved before TestMain moves the process CWD.
// Nothing in this package may reach for the payload with a relative path: the
// original os.DirFS(filepath.Join("..", "..")) only worked because `go test`
// starts in internal/cli/, which is exactly what TestMain changes.
var repoRoot string

// testSandbox identifies TestMain's sandbox directory by inode (via os.Stat),
// not by path string: os.Getwd() and the path MkdirTemp returned are not
// guaranteed to be textually identical (e.g. a symlinked TMPDIR), so
// TestCWDIsSandboxed compares with os.SameFile instead of string equality.
var testSandbox os.FileInfo

// TestMain makes CWD safety the default rather than opt-in. Commands in this
// package resolve their target by walking *up* from the process CWD
// (loadConfig -> os.Getwd -> config.Find), so a test that forgot newProject's
// t.Chdir would find pickle's own pickle.toml and mutate the real board —
// burning a global ticket id, which is never reused. Running the whole package
// from a throwaway directory means such a test fails loudly instead.
//
// TestCWDIsSandboxed proves this actually holds. Go permits exactly one
// TestMain per package: a second consumer of this harness (T-012, T-057) must
// extend this function, never add a second one.
func TestMain(m *testing.M) {
	// testutil.RepoRoot() is computed from its own source file's compile-time
	// path, not a guess against the process CWD (T-042 item 3), so it cannot
	// resolve a wrong root the way the old wd/../.. computation could — the
	// defensive os.Stat(skill/SKILL.md) validation that used to guard that
	// guess is therefore dead weight, deleted with it rather than ported.
	repoRoot = testutil.RepoRoot()

	sandbox, err := os.MkdirTemp("", "pickle-cli-sandbox")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: sandbox: %v\n", err)
		os.Exit(1)
	}
	if testSandbox, err = os.Stat(sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: stat sandbox: %v\n", err)
		os.Exit(1)
	}

	// Run inside an inner function so a defer is legal: os.Exit runs no
	// deferred functions in TestMain's own body, but a defer inside an inner
	// function still runs while that function's stack unwinds — on a normal
	// return, on a failed Chdir below, and during a panic inside m.Run — which
	// is what stops the sandbox leaking on exactly those two paths (T-031 N3).
	// Keep `sandbox` a named variable the deferred cleanup closes over: the
	// acceptance test's mutation A deletes the os.Chdir call below and must
	// produce a test failure, not an unused-variable compile error.
	code := func() int {
		defer func() {
			_ = os.Chdir(repoRoot)
			_ = os.RemoveAll(sandbox)
		}()
		if err := os.Chdir(sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "TestMain: chdir sandbox: %v\n", err)
			return 1
		}
		return m.Run()
	}()
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
		// runInstall's own four bespoke validation branches (internal/cli/install.go:38-51),
		// none of which were exercised by TestRunExitCodes before T-013 even though every
		// other command's usage errors already were.
		{"install --in-tree and --path conflict", []string{"install", "--in-tree", "--path", "foo"}, exitUsage},
		{"install --path \".\" without --in-tree", []string{"install", "--path", "."}, exitUsage},
		{"install child flags without --path/--in-tree", []string{"install", "--project", "foo"}, exitUsage},
		{"install bad flag", []string{"install", "--bogus"}, exitUsage},
		// upgrade takes no flags, but must reject rather than ignore argv: a
		// silently-ignored -h/-n would perform a real, mutating upgrade.
		{"upgrade bad flag", []string{"upgrade", "--bogus"}, exitUsage},
		{"upgrade help flag", []string{"upgrade", "-h"}, exitUsage},
		{"upgrade stray arg", []string{"upgrade", "extra"}, exitUsage},
		{"uninstall bad flag", []string{"uninstall", "--bogus"}, exitUsage},
		// runUninstall ignores stray positionals today (T-013 item 9): a typo'd
		// `pickle uninstall foo` performs a real uninstall instead of exitUsage.
		{"uninstall stray arg", []string{"uninstall", "extra"}, exitUsage},
		{"board sync bad flag", []string{"board", "sync", "--bogus"}, exitUsage},
		{"board decisions bad flag", []string{"board", "decisions", "--bogus"}, exitUsage},
		{"board no subcommand", []string{"board"}, exitUsage},
		{"board unknown subcommand", []string{"board", "xyz"}, exitUsage},
		{"ticket new missing title", []string{"ticket", "new"}, exitUsage},
		{"ticket move missing args", []string{"ticket", "move"}, exitUsage},
		{"ticket move missing status", []string{"ticket", "move", "T-001"}, exitUsage},
		{"project no subcommand", []string{"project"}, exitUsage},
		{"project unknown subcommand", []string{"project", "xyz"}, exitUsage},
		{"project add missing args", []string{"project", "add"}, exitUsage},
		// serve is long-running, so only its *rejections* are exercised here: each
		// of these must fail during argument parsing, before a listener is opened.
		{"serve bad flag", []string{"serve", "--bogus"}, exitUsage},
		{"serve stray arg", []string{"serve", "extra"}, exitUsage},
		{"serve missing addr value", []string{"serve", "--addr"}, exitUsage},
		{"serve empty addr", []string{"serve", "--addr="}, exitUsage},
		{"scaffold no subcommand", []string{"scaffold"}, exitUsage},
		{"scaffold unknown subcommand", []string{"scaffold", "xyz"}, exitUsage},
		{"scaffold release bad flag", []string{"scaffold", "release", "--bogus"}, exitUsage},
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
		// InTree: true — T-108: a root-path child must declare the in-tree layout,
		// or doctor's layout invariant (checkLayoutInvariant) refuses it.
		ProjectName: "demo", ProjectPath: ".", InTree: true, Agents: install.Agents{},
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
//
// It also asserts the CWD *is* testSandbox (by inode, not by path string), not
// merely "config-free": a leaked os.Chdir to some other config-free directory
// (e.g. a t.TempDir() from an earlier test that forgot t.Chdir's auto-restore)
// would satisfy the first check but not the second.
func TestCWDIsSandboxed(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if path, err := config.Find(wd); err == nil {
		t.Fatalf("default test CWD %s resolves a real config at %s — the TestMain sandbox is broken", wd, path)
	}
	wdInfo, err := os.Stat(wd)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(wdInfo, testSandbox) {
		t.Fatalf("CWD %s is not TestMain's sandbox — some earlier test left the CWD somewhere else", wd)
	}
}

// capture redirects *target (os.Stdout or os.Stderr — both process-global, like
// the sandboxed CWD, so no test in this package may call t.Parallel()) through
// a pipe for the duration of fn and returns what was written. The commands in
// this package print with fmt.Printf/fmt.Fprintln to the real stream (no
// injectable writer), so this is the only way to assert them; that refactor is
// deliberately declined here (T-043 D4) rather than done package-wide inside a
// harness-hardening change.
//
// *target is restored as soon as fn returns — *before* the pipe is closed — not
// only via t.Cleanup. Restoring solely in t.Cleanup, which runs at the *test's*
// end, left *target pointing at a closed pipe for everything between a call
// returning and the test finishing: every later print in the package silently
// discarded its output ("file already closed"), and two real call sites hit
// this before the early restore was added (T-031 N1, T-043). t.Cleanup stays as
// the Goexit/panic backstop for when fn calls t.Fatal and skips the assignment
// below entirely — this is an *additional* release, not a reversal of T-029's
// original decision to restore only through t.Cleanup.
//
// Closing the write end and joining the reader goroutine happen exactly once,
// from whichever path runs first, via closeAndJoin wrapped in sync.OnceFunc:
// nine of this package's fifteen capture call sites invoke t.Fatalf inside fn,
// which triggers runtime.Goexit and skips everything after fn() in the body
// below, so the t.Cleanup path must be able to do the whole teardown alone or
// leak the pipe and its reader goroutine for the rest of the process (T-031 N2).
func capture(t *testing.T, target **os.File, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := *target
	*target = w

	// Drain concurrently: fn writing more than the pipe buffer (16–64 KiB,
	// OS-dependent — darwin starts at 16 KiB) would block forever against a
	// reader that only starts after fn returns, hanging the test instead of
	// failing it.
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	var out string
	closeAndJoin := sync.OnceFunc(func() {
		_ = w.Close() // unblocks the reader
		out = <-done
		_ = r.Close()
	})
	t.Cleanup(func() {
		*target = orig
		closeAndJoin()
	})

	fn()

	*target = orig // early release — see the doc comment above
	closeAndJoin()
	return out
}

// captureStdout captures fn's writes to os.Stdout. See capture.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stdout, fn)
}

// captureStderr captures fn's writes to os.Stderr. See capture. Historically a
// verbatim copy of captureStdout living in agents_test.go, carrying the same two
// defects; unified here so there is exactly one implementation to keep correct
// (T-043).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return capture(t, &os.Stderr, fn)
}

// TestCaptureStdoutRestoresBeforeCleanup pins T-031 N1 / T-043: os.Stdout must
// be usable immediately after captureStdout returns, not only once the *test*
// ends and t.Cleanup finally runs. Before the early restore this failed with
// "file already closed", because captureStdout's own (already-closed) pipe
// writer was still installed as os.Stdout at this point — exactly the failure
// two real call sites hit (cli_test.go's project-remove line, hooks_test.go's
// hooks-uninstall line) before this fix.
func TestCaptureStdoutRestoresBeforeCleanup(t *testing.T) {
	captureStdout(t, func() { fmt.Println("inside the capture") })
	n, err := fmt.Fprintln(os.Stdout, "after capture returns, before the test ends")
	if err != nil {
		t.Fatalf("os.Stdout is unusable immediately after captureStdout returns: %v", err)
	}
	if n == 0 {
		t.Fatal("wrote 0 bytes to os.Stdout after captureStdout returned")
	}
}

// goexitChildEnv gates TestCaptureGoexitChild: it must never run as part of a
// normal `go test ./internal/cli/` (which would either fail the suite on its
// intentional t.Fatal, or contaminate results depending on run order), only as
// the subprocess TestCaptureGoexitDoesNotLeak launches.
const goexitChildEnv = "PICKLE_TEST_CAPTURE_GOEXIT_CHILD"

// TestCaptureGoexitChild intentionally fails inside a capture call. A t.Run
// subtest cannot exercise this in-process: a subtest's failure bubbles up and
// fails the parent regardless of any check performed afterwards, so the child
// registers its own leak check as a t.Cleanup *before* calling capture. Cleanups
// run LIFO, so capture's own cleanup (registered second, inside capture) runs
// first — by the time this one runs, capture's close+join has already
// happened if the fix holds. It prints a marker rather than asserting: this
// process's own exit code is always non-zero (the test really did fail) and
// carries no information about whether the leak check passed.
func TestCaptureGoexitChild(t *testing.T) {
	if os.Getenv(goexitChildEnv) != "1" {
		t.Skip("only runs as TestCaptureGoexitDoesNotLeak's subprocess")
	}
	before := runtime.NumGoroutine()
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if runtime.NumGoroutine() <= before {
				fmt.Println("LEAK-CHECK-OK")
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Println("LEAK-CHECK-FAIL")
	})
	capture(t, &os.Stdout, func() {
		t.Fatal("intentional failure inside fn — capture must still close its pipe and join its goroutine")
	})
}

// TestCaptureGoexitDoesNotLeak pins T-031 N2 / T-043: nine of this package's
// fifteen capture call sites call t.Fatalf inside fn, which invokes
// runtime.Goexit and skips every statement after fn() in capture's body —
// including the close+join that unblocks and retires the reader goroutine. If
// t.Cleanup cannot perform that teardown on its own, the pipe and its goroutine
// leak for the rest of the process, once per such failure.
func TestCaptureGoexitDoesNotLeak(t *testing.T) {
	if os.Getenv(goexitChildEnv) == "1" {
		t.Skip("this process is the child")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCaptureGoexitChild$")
	cmd.Env = append(os.Environ(), goexitChildEnv+"=1")
	// The parent's own CWD is already TestMain's sandbox, not go test's usual
	// internal/cli/ — the child's own TestMain computes repoRoot as wd/../..,
	// so it must start from the real internal/cli/ (via the already-validated
	// repoRoot) or its own repo-root validation fails for an unrelated reason.
	cmd.Dir = filepath.Join(repoRoot, "internal", "cli")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected the child subprocess to fail (t.Fatal inside fn); output:\n%s", out)
	}
	switch {
	case bytes.Contains(out, []byte("LEAK-CHECK-OK")):
		// pass
	case bytes.Contains(out, []byte("LEAK-CHECK-FAIL")):
		t.Errorf("capture's reader goroutine leaked when fn called t.Fatal inside it; child output:\n%s", out)
	default:
		t.Fatalf("child subprocess produced neither leak-check marker; output:\n%s", out)
	}
}

// TestProjectAddRefreshesMarkerBlock is T-041's write-half regression: a new
// child, its own WIP limits and its own branch/ticket prefixes must appear in
// AGENTS.md immediately, not only after a later `pickle upgrade`.
func TestProjectAddRefreshesMarkerBlock(t *testing.T) {
	root := newProject(t) // installs child "demo" at "."
	// demo's repo root and web's sub-repo both need a .git entry for `doctor`'s
	// checkChildren to pass — not something project add itself cares about.
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{
			"project", "add", "web", "sub",
			"--ticket-prefix", "WEB", "--branch-prefix", "ticket/",
			"--wip-dev", "5", "--wip-review", "5",
		}); got != exitOK {
			t.Fatalf("project add = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "registered child-project \"web\"") {
		t.Errorf("missing the registration confirmation line, got:\n%s", out)
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	block := string(agents)
	for _, want := range []string{
		"`web`",
		"- `web`: `ticket/WEB-NNN-<slug>`",
		"- `web`: `3-in-development/` ≤ 5 · `4-in-review/` ≤ 5",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("AGENTS.md missing %q after project add, got:\n%s", want, block)
		}
	}

	// doctor must see the refreshed block as current, not as drift.
	doctorOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"doctor"}); got != exitOK {
			t.Fatalf("doctor = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(doctorOut, "markers:") {
		t.Errorf("doctor reported marker drift right after project add refreshed the block:\n%s", doctorOut)
	}

	// project remove drops web from the block again.
	if got := Run(nil, "test", []string{"project", "remove", "web"}); got != exitOK {
		t.Fatalf("project remove = %d, want %d", got, exitOK)
	}
	agents, err = os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agents), "`web`") {
		t.Errorf("AGENTS.md still names web after project remove:\n%s", agents)
	}
}

// TestProjectAddRegeneratesBoard (T-052): the registered-child list is a
// board.Render input (a new child gains its own `### <child>` section + WIP
// line under every status heading), so `project add` must regenerate
// tickets/BOARD.md the same moment it refreshes the marker block — otherwise
// the very next `pickle upgrade` reports the board stale, for a workspace
// where nothing is wrong (the ticket's own field report).
func TestProjectAddRegeneratesBoard(t *testing.T) {
	root := newProject(t) // installs child "demo" at "."
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "add", "web", "sub"}); got != exitOK {
			t.Fatalf("project add = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "+ tickets/BOARD.md") {
		t.Errorf("project add did not report regenerating the board, got:\n%s", out)
	}

	auditOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
			t.Fatalf("board audit = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(auditOut, "0 error(s), 0 warning(s)") {
		t.Errorf("board audit not clean right after project add, got:\n%s", auditOut)
	}

	// project remove regenerates too — web's section must disappear, not just
	// its frontmatter registration.
	removeOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "remove", "web"}); got != exitOK {
			t.Fatalf("project remove = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(removeOut, "+ tickets/BOARD.md") {
		t.Errorf("project remove did not report regenerating the board, got:\n%s", removeOut)
	}
	board, err := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(board), "### web") {
		t.Errorf("BOARD.md still carries a web section after project remove:\n%s", board)
	}
}

// TestUpgradeWarnsOnLayoutOnlyBoardDrift (T-052): a board that is stale only in
// its generated layout (here, a hand-edited WIP limit in pickle.toml, the same
// class of divergence a registry change produces) must not fail `pickle
// upgrade` — it prints the warning and still exits 0. Row divergence stays an
// error either way (unchanged by this ticket) and is covered at the audit
// level by internal/audit's own test table, not duplicated here.
func TestUpgradeWarnsOnLayoutOnlyBoardDrift(t *testing.T) {
	root := newProject(t) // installs child "demo" at ".", board already in sync

	cfgPath := filepath.Join(root, "pickle.toml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Projects {
		cfg.Projects[i].WIPInReview = cfg.Projects[i].WIPInReview + 1
	}
	if err := cfg.Save(""); err != nil {
		t.Fatal(err)
	}

	auditOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
			t.Fatalf("board audit = %d, want %d (layout-only drift must not error)", got, exitOK)
		}
	})
	if !strings.Contains(auditOut, "WARNING: BOARD.md is out of date in its generated layout only") {
		t.Errorf("board audit did not warn about layout-only drift, got:\n%s", auditOut)
	}

	// runUpgrade reads Payload (set by Run's first argument, distinct from the
	// install.Run(os.DirFS(repoRoot), ...) newProject used directly) to refresh
	// the skill copy — nil would panic inside install.copyPayload, so it must be
	// passed here even though this test cares only about the board warning.
	// Prints WARNING/ERROR to stderr, alongside its own +/= progress lines on
	// stdout (mirroring runInstall) — captured separately.
	var code int
	upgradeErr := captureStderr(t, func() {
		code = Run(os.DirFS(repoRoot), "test", []string{"upgrade"})
	})
	if code != exitOK {
		t.Fatalf("upgrade = %d, want %d (a warning must not fail the upgrade); stderr:\n%s", code, exitOK, upgradeErr)
	}
	if !strings.Contains(upgradeErr, "WARNING: BOARD.md is out of date in its generated layout only") {
		t.Errorf("upgrade did not print the layout-drift warning, got:\n%s", upgradeErr)
	}
}

// TestCLIUninstallDryRunLeavesTreeUnchanged is the T-013 item 9 CLI-level
// regression: install.Uninstall's own dry-run behaviour is already covered by
// TestUninstallDryRun in internal/install, but runUninstall's CLI wiring
// (flag parsing, cwd-based project resolution) had no coverage at all before
// this. It runs against newProject's temp root, never pickle's own
// (mandatory per this package's cwd hazard, see TestMain/newProject), and
// asserts --dry-run reports what it would remove without touching anything.
func TestCLIUninstallDryRunLeavesTreeUnchanged(t *testing.T) {
	root := newProject(t)

	skillDir := filepath.Join(root, filepath.FromSlash(install.SkillDir))
	if _, err := os.Stat(skillDir); err != nil {
		t.Fatalf("skill dir missing before dry-run: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if got := Run(os.DirFS(repoRoot), "test", []string{"uninstall", "--dry-run"}); got != exitOK {
			t.Fatalf("uninstall --dry-run = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, install.SkillDir) {
		t.Errorf("dry-run output does not name the skill dir it would remove, got:\n%s", out)
	}

	after, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("uninstall --dry-run changed AGENTS.md")
	}
	if _, err := os.Stat(skillDir); err != nil {
		t.Errorf("uninstall --dry-run removed the skill dir: %v", err)
	}
}

// TestCLIUninstallHappyPath is the T-013 item 9 CLI-level happy-path
// regression, on the same temp root as the dry-run test above: a real
// `pickle uninstall` exits exitOK and actually removes the skill dir and the
// AGENTS.md marker block.
func TestCLIUninstallHappyPath(t *testing.T) {
	root := newProject(t)
	skillDir := filepath.Join(root, filepath.FromSlash(install.SkillDir))

	if got := Run(os.DirFS(repoRoot), "test", []string{"uninstall"}); got != exitOK {
		t.Fatalf("uninstall = %d, want %d", got, exitOK)
	}

	if _, err := os.Lstat(skillDir); !os.IsNotExist(err) {
		t.Errorf("skill dir still present after uninstall: err=%v", err)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(agents), install.MarkerBegin) {
		t.Error("AGENTS.md still has a marker block after uninstall")
	}
}

// TestProjectAddNotesStageableChild (T-051): registering a child at a nested
// path that is a real, unignored git repository prints a note naming it —
// the same sentence `pickle doctor` warns with, and neither changes the exit
// code. Once `.gitignore` covers the child, both the note and the doctor
// warning disappear.
func TestProjectAddNotesStageableChild(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t) // installs child "demo" at "."
	gitInit(t, root, "main")

	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInit(t, sub, "main")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "add", "web", "sub"}); got != exitOK {
			t.Fatalf("project add = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(out, "registered child-project \"web\"") {
		t.Errorf("missing the registration confirmation line, got:\n%s", out)
	}
	if !strings.Contains(out, "note:") || !strings.Contains(out, "/sub/") {
		t.Errorf("expected a note: line naming /sub/, got:\n%s", out)
	}

	doctorOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"doctor"}); got != exitOK {
			t.Fatalf("doctor = %d, want %d (a stageable child is a warning, not an error)", got, exitOK)
		}
	})
	if !strings.Contains(doctorOut, "WARNING:") || !strings.Contains(doctorOut, "/sub/") {
		t.Errorf("expected doctor to warn about /sub/, got:\n%s", doctorOut)
	}

	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/sub/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doctorOut = captureStdout(t, func() {
		if got := Run(nil, "test", []string{"doctor"}); got != exitOK {
			t.Fatalf("doctor = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(doctorOut, "WARNING:") {
		t.Errorf("expected no warning once .gitignore covers /sub/, got:\n%s", doctorOut)
	}
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

// TestTicketMovePrintsStageLine is the T-091 regression: a move is a rename, so
// the printed summary must name both the new and the removed path plus the
// board, as a literal ready-to-paste `git add` — a bookkeeping commit made
// with rules §0's explicit pathspecs can otherwise stage the add without the
// delete. The first line must stay byte-identical (decision 2).
func TestTicketMovePrintsStageLine(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "fixture", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"ticket", "move", "T-001", "dropped", "--reason", "test fixture"}); got != exitOK {
			t.Fatalf("ticket move = %d", got)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d:\n%s", len(lines), out)
	}
	wantFirst := "moved T-001: TO DO → DROPPED  (tickets/7-dropped/T-001-fixture.md)"
	if lines[0] != wantFirst {
		t.Errorf("first line = %q, want %q", lines[0], wantFirst)
	}
	wantRemoved := "  removed  tickets/1-to-do/T-001-fixture.md"
	if lines[1] != wantRemoved {
		t.Errorf("removed line = %q, want %q", lines[1], wantRemoved)
	}
	wantStage := "  stage:   git add tickets/7-dropped/T-001-fixture.md tickets/1-to-do/T-001-fixture.md tickets/BOARD.md"
	if lines[2] != wantStage {
		t.Errorf("stage line = %q, want %q", lines[2], wantStage)
	}
}

// TestTicketNewPrintsStageLine is T-091's other half: `ticket new` also writes
// two paths (the ticket and the regenerated board), so its stage line must name
// both and must not print a `removed` line — it removes nothing. The first
// line must stay byte-identical (decision 2).
func TestTicketNewPrintsStageLine(t *testing.T) {
	newProject(t)
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"ticket", "new", "fixture", "--project", "demo"}); got != exitOK {
			t.Fatalf("ticket new = %d", got)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected exactly 2 lines (no removed line), got %d:\n%s", len(lines), out)
	}
	wantFirst := "created T-001  (tickets/1-to-do/T-001-fixture.md)"
	if lines[0] != wantFirst {
		t.Errorf("first line = %q, want %q", lines[0], wantFirst)
	}
	wantStage := "  stage:   git add tickets/1-to-do/T-001-fixture.md tickets/BOARD.md"
	if lines[1] != wantStage {
		t.Errorf("stage line = %q, want %q", lines[1], wantStage)
	}
	for _, ln := range lines {
		if strings.Contains(ln, "removed") {
			t.Errorf("ticket new printed a removed line, want none:\n%s", out)
		}
	}
}

// TestTicketNewUsesChildPrefix pins T-058 end to end: a child registered with a
// ticket_prefix gets ids under that prefix, numbered independently of the default
// "T" child, and the resulting tree audits clean.
func TestTicketNewUsesChildPrefix(t *testing.T) {
	root := newProject(t) // installs child "demo" at "." with the default "T" prefix
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Run(nil, "test", []string{"project", "add", "lib", "lib", "--ticket-prefix", "PK"}); got != exitOK {
		t.Fatalf("project add = %d, want %d", got, exitOK)
	}
	// Seed a T- ticket on demo so the two prefixes are proven to count separately.
	if got := Run(nil, "test", []string{"ticket", "new", "demo feature", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new (demo) = %d", got)
	}
	if got := Run(nil, "test", []string{"ticket", "new", "first lib feature", "--project", "lib"}); got != exitOK {
		t.Fatalf("ticket new (lib) = %d", got)
	}
	body := readTicket(t, root, "PK-001")
	if !strings.Contains(body, "id: PK-001") || !strings.Contains(body, "project: lib") {
		t.Errorf("expected PK-001 targeting lib:\n%s", body)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit = %d, want clean (%d)", got, exitOK)
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

// TestTicketNewFamily covers --family end to end (T-059): a valid same-child
// umbrella writes a family: line and audits clean; a malformed id is rejected at
// creation (shape check), while an unknown-but-well-shaped id is written and left
// for the audit — the same creation/existence split as --spawned-by.
func TestTicketNewFamily(t *testing.T) {
	root := newProject(t)
	// Umbrella must exist for a clean audit.
	if got := Run(nil, "test", []string{"ticket", "new", "umbrella", "--project", "demo"}); got != exitOK {
		t.Fatalf("seed umbrella: exit %d", got)
	}
	if got := Run(nil, "test", []string{"ticket", "new", "member", "--project", "demo", "--family", "T-001"}); got != exitOK {
		t.Fatalf("ticket new --family = %d, want %d", got, exitOK)
	}
	if body := readTicket(t, root, "T-002"); !strings.Contains(body, "family: T-001") {
		t.Errorf("scaffold missing family: T-001:\n%s", body)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit = %d, want clean (%d)", got, exitOK)
	}

	// Malformed id rejected at creation — nothing written.
	if got := Run(nil, "test", []string{"ticket", "new", "bad", "--project", "demo", "--family", "nope"}); got == exitOK {
		t.Errorf("--family nope should be rejected (bad id shape)")
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
		// The three Unicode line terminators (T-038 finding N1). None of them
		// changes pickle's own behaviour — ParseFrontmatter splits on \n alone —
		// but YAML 1.1 readers (Ruby Psych, PyYAML) treat all three as line
		// breaks, so each one injects the same phantom key \n does. The titles
		// below are the measured repros from the ticket.
		{"NEL line terminator", "a\u0085project: nope"},
		{"line separator", "a\u2028project: nope"},
		{"paragraph separator", "a\u2029project: nope"},
		// One rune past maxTitleRuneLen (T-038 finding N2).
		{"over-length title", strings.Repeat("a", 121)},
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

// TestTicketNewAcceptsTitleAtTheCap pins the two halves of T-038 decision 2 that
// the rejection table cannot reach, because a table of *rejected* titles can only
// ever prove the cap fires — never that it fires in the right place. Both cases
// below were confirmed to fail under a deliberate mutation of validateTitle, so
// neither is decorative:
//
//   - Exactly maxTitleRuneLen runes must be ACCEPTED. Catches an off-by-one:
//     flipping `>` to `>=` rejects this title while every other test stays green.
//   - maxTitleRuneLen *multi-byte* runes must be ACCEPTED. Catches the rune/byte
//     confusion: 120 two-byte runes are 240 bytes, so a len()-based cap rejects
//     this title — again with every other test still green. This is the case that
//     makes utf8.RuneCountInString load-bearing rather than stylistic.
//
// The slug for the multi-byte case collapses to "untitled" (every rune is
// non-[a-z0-9]), which is correct and is why the assertion reads the frontmatter
// rather than the filename.
func TestTicketNewAcceptsTitleAtTheCap(t *testing.T) {
	for _, tc := range []struct {
		name, title string
	}{
		{"exactly at the cap", strings.Repeat("a", maxTitleRuneLen)},
		{"at the cap in multi-byte runes", strings.Repeat("é", maxTitleRuneLen)},
	} {
		t.Run(tc.name, func(t *testing.T) { // no t.Parallel: newProject chdirs
			if n := utf8.RuneCountInString(tc.title); n != maxTitleRuneLen {
				t.Fatalf("test fixture is %d runes, want exactly %d", n, maxTitleRuneLen)
			}
			root := newProject(t)
			if got := Run(nil, "test", []string{"ticket", "new", tc.title, "--project", "demo"}); got != exitOK {
				t.Fatalf("Run(ticket new <%s>) = %d, want %d — a title at the cap must be accepted", tc.name, got, exitOK)
			}
			if body := readTicket(t, root, "T-001"); !strings.Contains(body, "title: "+tc.title) {
				t.Errorf("scaffold missing the title verbatim:\n%s", body)
			}
		})
	}
}

// TestTicketNewOverlongTitleErrorNamesTheContract is the regression proof for
// T-038 finding N2. Before the cap, an over-long title was stopped only by the
// OS: os.WriteFile failed with `open /abs/path/…: file name too long`, which
// leaked an absolute path and blamed the filesystem for what is really a
// contract violation of the input. The tree stayed consistent either way — the
// write precedes board.AddTODORow — so this was always a message-quality bug,
// which is exactly why only an assertion on the *wording* can catch a
// regression. Asserting the exit code alone would still pass if the raw syscall
// error came back.
func TestTicketNewOverlongTitleErrorNamesTheContract(t *testing.T) {
	root := newProject(t) // chdirs; also gives us a path that must not appear
	title := strings.Repeat("a", maxTitleRuneLen+1)

	var code int
	stderr := captureStderr(t, func() {
		code = Run(nil, "test", []string{"ticket", "new", title, "--project", "demo"})
	})

	if code != exitError {
		t.Errorf("Run(ticket new <121 runes>) = %d, want %d", code, exitError)
	}
	for _, want := range []string{"exceeds", "120"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to contain %q (the contract, per T-038 decision 3)", stderr, want)
		}
	}
	// The two halves of the leak: the syscall's own phrasing, and the absolute
	// path it embedded. Both must be gone, not merely reworded around.
	if strings.Contains(stderr, "file name too long") {
		t.Errorf("stderr = %q, want the contract error, not the raw syscall error", stderr)
	}
	if strings.Contains(stderr, root) {
		t.Errorf("stderr = %q, want no absolute path (leaked project root %q)", stderr, root)
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

// TestParseServeArgs covers the serve flag contract without binding a port — no
// test may occupy the documented default 127.0.0.1:8745.
func TestParseServeArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantAddr string
		wantCode int
	}{
		{"default", nil, defaultAddr, exitOK},
		{"addr separate value", []string{"--addr", "127.0.0.1:9999"}, "127.0.0.1:9999", exitOK},
		{"addr equals form", []string{"--addr=0.0.0.0:1234"}, "0.0.0.0:1234", exitOK},
		{"short flag", []string{"-a", "127.0.0.1:1"}, "127.0.0.1:1", exitOK},
		{"unknown flag", []string{"--bogus"}, "", exitUsage},
		{"stray positional", []string{"8080"}, "", exitUsage},
		{"missing value", []string{"--addr"}, "", exitUsage},
		{"empty value", []string{"--addr="}, "", exitUsage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, code := parseServeArgs(tc.args)
			if code != tc.wantCode || addr != tc.wantAddr {
				t.Fatalf("parseServeArgs(%v) = (%q, %d), want (%q, %d)",
					tc.args, addr, code, tc.wantAddr, tc.wantCode)
			}
		})
	}
	if defaultAddr != "127.0.0.1:8745" {
		t.Errorf("defaultAddr = %q; the documented default is 127.0.0.1:8745", defaultAddr)
	}
}

// TestIsLoopback pins which addresses trigger the "no authentication" warning.
func TestIsLoopback(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8745", true},
		{"[::1]:8745", true},
		{"localhost:8745", true},
		{"0.0.0.0:8745", false}, // all interfaces
		{":8745", false},        // empty host is all interfaces
		{"192.168.1.10:8745", false},
		{"example.invalid:8745", false}, // unresolvable → warn rather than stay quiet
	}
	for _, tc := range cases {
		if got := isLoopback(tc.addr); got != tc.want {
			t.Errorf("isLoopback(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

// TestServeHelpIsAdvertised keeps the dispatch table and the usage text in step:
// a command that exists but is undocumented is invisible.
func TestServeHelpIsAdvertised(t *testing.T) {
	out := captureStdout(t, func() { Run(nil, "test", []string{"help"}) })
	for _, want := range []string{"serve", "127.0.0.1:8745", "read-only"} {
		if !strings.Contains(out, want) {
			t.Errorf("pickle help does not mention %q", want)
		}
	}
}

// TestScaffoldReleaseIsUnrelatedToBrine (T-113): pickle scaffold release
// writes its files into a plain directory with no pickle.toml / brine
// install at all — unlike every newProject-based test in this file, on
// purpose, since the whole point of the command is that it does not require
// (or touch) brine.
func TestScaffoldReleaseIsUnrelatedToBrine(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	if got := Run(os.DirFS(repoRoot), "test", []string{"scaffold", "release", "--project-name", "demo"}); got != exitOK {
		t.Fatalf("scaffold release = %d, want exitOK", got)
	}
	for _, f := range []string{"CHANGELOG.md", "RELEASING.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
	}
	// Nothing brine-owned was written.
	for _, f := range []string{"pickle.toml", "tickets", "AGENTS.md", ".agents"} {
		if _, err := os.Stat(filepath.Join(root, f)); !os.IsNotExist(err) {
			t.Errorf("%s exists (err=%v) — scaffold release must never touch brine", f, err)
		}
	}
}

// TestProjectAddRejectsDuplicateAndMissingDir (T-043 item 1): the two argument
// validations runProjectAdd performs that no test exercised — a name already
// registered, and a path that does not resolve to a directory under the
// config root — and that neither rejection mutates the registry.
func TestProjectAddRejectsDuplicateAndMissingDir(t *testing.T) {
	root := newProject(t) // registers "demo" at "."
	if got := Run(nil, "test", []string{"project", "add", "demo", "."}); got == exitOK {
		t.Error("project add demo (duplicate name) = exitOK, want an error")
	}
	if got := Run(nil, "test", []string{"project", "add", "other", "does-not-exist"}); got == exitOK {
		t.Error("project add other does-not-exist = exitOK, want an error")
	}
	cfg, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 1 {
		t.Errorf("Projects = %v, want exactly one entry (demo) — neither rejection should register anything", cfg.Projects)
	}
}

// TestProjectListOutput (item 1): runProjectList's tabwriter output, end to end.
func TestProjectListOutput(t *testing.T) {
	root := newProject(t)
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Run(nil, "test", []string{"project", "add", "lib", "lib", "--build", "make build"}); got != exitOK {
		t.Fatalf("project add = %d, want %d", got, exitOK)
	}
	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "list"}); got != exitOK {
			t.Fatalf("project list = %d, want %d", got, exitOK)
		}
	})
	for _, want := range []string{"NAME", "demo", "lib", "make build", "1/1"} {
		if !strings.Contains(out, want) {
			t.Errorf("project list missing %q, got:\n%s", want, out)
		}
	}
}

// TestProjectRemoveRefusesLiveTicket (item 1): the guard that stops removing a
// child while a non-terminal ticket still targets it, and confirms removal
// succeeds once that ticket reaches a terminal status.
func TestProjectRemoveRefusesLiveTicket(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "still open", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	if got := Run(nil, "test", []string{"project", "remove", "demo"}); got == exitOK {
		t.Error("project remove demo = exitOK, want a refusal (T-001 targets it)")
	}
	cfg, err := config.Load(filepath.Join(root, "pickle.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Project("demo"); !ok {
		t.Error("demo was removed despite the live-ticket guard")
	}
	if got := Run(nil, "test", []string{"ticket", "move", "T-001", "dropped", "--reason", "test cleanup"}); got != exitOK {
		t.Fatalf("ticket move = %d", got)
	}
	if got := Run(nil, "test", []string{"project", "remove", "demo"}); got != exitOK {
		t.Fatalf("project remove demo = %d once the blocking ticket was dropped, want %d", got, exitOK)
	}
}

// TestBoardAuditCLIExitCodes (item 1): runBoardAudit's exit code on a clean vs
// a broken tree, at the cli boundary (internal/audit has its own unit tests).
func TestBoardAuditCLIExitCodes(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit (clean tree) = %d, want %d", got, exitOK)
	}
	if got := Run(nil, "test", []string{"ticket", "new", "broken", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	dir := filepath.Join(root, "tickets", "1-to-do")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "T-001-") {
			path = filepath.Join(dir, e.Name())
		}
	}
	if path == "" {
		t.Fatal("T-001 scaffold not found")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A dangling depends-on: the target does not exist anywhere in the tree.
	broken := strings.Replace(string(body), "depends-on: []", "depends-on: [T-404]", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got == exitOK {
		t.Error("board audit (dangling depends-on) = exitOK, want non-zero")
	}
}

// TestTicketNewAllocatesMaxPlusOne (item 4): a fresh id is max(existing)+1 and
// the scaffold audits clean immediately, not only after a later board sync.
func TestTicketNewAllocatesMaxPlusOne(t *testing.T) {
	root := newProject(t)
	for _, title := range []string{"first", "second"} {
		if got := Run(nil, "test", []string{"ticket", "new", title, "--project", "demo"}); got != exitOK {
			t.Fatalf("ticket new %q = %d", title, got)
		}
	}
	if body := readTicket(t, root, "T-002"); !strings.Contains(body, "title: second") {
		t.Errorf("expected T-002 to be the second ticket:\n%s", body)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit = %d, want clean", got)
	}
}

// TestTicketNewRejectsUnregisteredProject and TestTicketNewRejectsIllegalGrade
// (item 4): the two argument-validation failure modes, both asserting the
// tree is untouched on rejection (the same load-bearing shape as
// TestTicketNewRejectsInjectionInTitle next door).
func TestTicketNewRejectsUnregisteredProject(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "x", "--project", "nope"}); got != exitError {
		t.Errorf("ticket new --project nope = %d, want %d", got, exitError)
	}
	assertNoTickets(t, root)
}

func TestTicketNewRejectsIllegalGrade(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "x", "--project", "demo", "--impact", "banana"}); got != exitError {
		t.Errorf("ticket new --impact banana = %d, want %d", got, exitError)
	}
	assertNoTickets(t, root)
}

// TestTicketNewBoardRowOrderedByImpact (item 4): a fresh ticket's board row
// lands in its child's impact-ordered sub-group immediately — runTicketNew
// calls board.Regenerate itself, not only a later `board sync`.
func TestTicketNewBoardRowOrderedByImpact(t *testing.T) {
	root := newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "the low one", "--project", "demo", "--impact", "low"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	if got := Run(nil, "test", []string{"ticket", "new", "the critical one", "--project", "demo", "--impact", "critical"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	board, err := os.ReadFile(filepath.Join(root, "tickets", "BOARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	lowIdx := strings.Index(string(board), "T-001")
	criticalIdx := strings.Index(string(board), "T-002")
	if lowIdx < 0 || criticalIdx < 0 {
		t.Fatalf("expected both T-001 and T-002 in BOARD.md:\n%s", board)
	}
	if criticalIdx > lowIdx {
		t.Errorf("T-002 (critical) sorted after T-001 (low) in BOARD.md — impact ordering not applied on ticket new")
	}
}

// TestTicketNewPipeTitleYieldsAuditCleanBoard is the residue of item 5
// (T-012's original title-sanitization item) once T-044/T-049 landed one-way
// board-cell sanitisation at the render boundary: a title containing '|' —
// already proven to round-trip verbatim into the ticket file by
// TestTicketNewAcceptsAwkwardButLegalTitle next door — must also still
// produce an audit-clean board, the one assertion this item shrank to.
func TestTicketNewPipeTitleYieldsAuditCleanBoard(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"ticket", "new", "pipe | in title", "--project", "demo"}); got != exitOK {
		t.Fatalf("ticket new = %d", got)
	}
	if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
		t.Fatalf("board audit = %d, want clean after a piped title", got)
	}
}

// TestInstallHooksFlag (item 9): `pickle install --hooks` had no coverage but
// T-057's manual acceptance transcript, which just(test)/CI never runs. Two
// branches: the success path (a real git repository) and the deliberate "a
// hook failure is a warning, the install still succeeds" branch (no git
// repository at all, install.go:101-103).
func TestInstallHooksFlag(t *testing.T) {
	payload := os.DirFS(repoRoot)

	t.Run("git repository: hook installed, exit OK", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		gitInit(t, root, "main")
		out := captureStdout(t, func() {
			if code := Run(payload, "test", []string{"install", "--in-tree", "--project", "demo", "--hooks"}); code != exitOK {
				t.Fatalf("install --hooks = %d, want %d", code, exitOK)
			}
		})
		if !strings.Contains(out, "pre-commit") {
			t.Errorf("install --hooks did not report the hook path:\n%s", out)
		}
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
			t.Errorf("hook was not written: %v", err)
		}
	})

	t.Run("not a git repository: warning, install still succeeds", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		var stderr string
		_ = captureStdout(t, func() {
			stderr = captureStderr(t, func() {
				if code := Run(payload, "test", []string{"install", "--in-tree", "--project", "demo", "--hooks"}); code != exitOK {
					t.Fatalf("install --hooks (no git) = %d, want %d (a hook failure is a warning, not a failed install)", code, exitOK)
				}
			})
		})
		if !strings.Contains(stderr, "hooks install skipped") {
			t.Errorf("expected the skip warning on stderr, got: %q", stderr)
		}
		if _, err := os.Stat(filepath.Join(root, "pickle.toml")); err != nil {
			t.Errorf("install itself should still have succeeded: %v", err)
		}
	})

	t.Run("a second install hits the Skipped branch", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		gitInit(t, root, "main")
		if code := Run(payload, "test", []string{"install", "--in-tree", "--project", "demo", "--hooks"}); code != exitOK {
			t.Fatalf("first install --hooks = %d", code)
		}
		// A second install re-applies the whole scaffold; hook.Install must see
		// its own shim already current and report "skipped" rather than an error.
		out := captureStdout(t, func() {
			if code := Run(payload, "test", []string{"install", "--in-tree", "--project", "demo2", "--hooks"}); code != exitOK {
				t.Fatalf("second install --hooks = %d, want %d", code, exitOK)
			}
		})
		if !strings.Contains(out, "current") {
			t.Errorf("expected the second install to report the hook already current, got:\n%s", out)
		}
	})
}

// TestProjectAddSilentOnNonRepoChild (T-051 review F3): registering a plain
// directory that is not a git repository must not call it a "nested git
// repository". That state is doctor's error to report, and the two must not
// contradict each other about the same child — the note exists precisely so
// registration and the next doctor run speak with one voice.
func TestProjectAddSilentOnNonRepoChild(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	// A plain directory: no .git inside it.
	if err := os.MkdirAll(filepath.Join(root, "plain"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "add", "plain", "plain"}); got != exitOK {
			t.Fatalf("project add = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(out, "note:") {
		t.Errorf("a non-repo child must not get a nested-git-repository note, got:\n%s", out)
	}

	// ...and doctor is the one that speaks up, as an error.
	doctorOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"doctor"}); got == exitOK {
			t.Error("doctor = 0, want non-zero for a child that is not a git repository")
		}
	})
	if !strings.Contains(doctorOut, "is not a git repository") {
		t.Errorf("expected doctor to report the non-repo child, got:\n%s", doctorOut)
	}
	if strings.Contains(doctorOut, "nested git repository") {
		t.Errorf("doctor must not also call it a nested git repository, got:\n%s", doctorOut)
	}
}

// TestProjectAddSilentOnRootSpelledDotSlash (T-051 review R1): a child
// registered as "./" is the overarching repository itself, however it was
// spelled. It must never be described as a nested git repository, and doctor
// must not warn about it — the advised entry ("/./") is one git does not
// honour, so such a warning could never be silenced by obeying it.
func TestProjectAddSilentOnRootSpelledDotSlash(t *testing.T) {
	gitOrSkip(t)
	root := newProject(t)
	gitInit(t, root, "main")

	out := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"project", "add", "same", "./"}); got != exitOK {
			t.Fatalf("project add = %d, want %d", got, exitOK)
		}
	})
	if strings.Contains(out, "note:") {
		t.Errorf("the repository root must not be called a nested git repository, got:\n%s", out)
	}

	// T-108: registering a second child at "./" — also the repository root —
	// now trips the layout invariant (exactly one child may sit at "."), a
	// real and separate finding from the nested-git-repository warning T-051
	// guards against. doctor is expected to error for that reason; the point
	// pinned here is narrower and unaffected: it must never *also* describe
	// the "./"-spelled root as a nested git repository.
	doctorOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"doctor"}); got != exitError {
			t.Fatalf("doctor = %d, want %d (two children now share the repo root)", got, exitError)
		}
	})
	if strings.Contains(doctorOut, "WARNING:") {
		t.Errorf("doctor must not warn about the repo root registered as %q, got:\n%s", "./", doctorOut)
	}
	if !strings.Contains(doctorOut, "layout") {
		t.Errorf("expected the layout invariant to fire for two children sharing the repo root, got:\n%s", doctorOut)
	}
	if strings.Contains(doctorOut, "nested git repository") {
		t.Errorf("doctor must not describe the \"./\"-spelled root as a nested git repository, got:\n%s", doctorOut)
	}
}

// TestCreateExclusiveRefusesWhenTargetExists is T-101 Task 5's direct,
// deterministic regression for the os.Stat-then-os.WriteFile pair that
// createExclusive replaced: given a path that already exists, it must
// refuse with an os.IsExist error rather than silently overwriting, and
// must never touch the existing file's content.
func TestCreateExclusiveRefusesWhenTargetExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T-001-fixture.md")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := createExclusive(path, "clobbered\n")
	if err == nil {
		t.Fatal("createExclusive succeeded against an existing file, want a refusal")
	}
	if !os.IsExist(err) {
		t.Errorf("error = %v, want an os.IsExist error", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "original\n" {
		t.Errorf("existing file was modified: got %q, want unchanged %q", got, "original\n")
	}
}

// TestConcurrentTicketNewGetsDistinctIDs is T-101's in-process regression for
// the id-allocation TOCTOU: before the tree lock, two concurrent `ticket
// new` calls could both read the same NextNum and each write a file, one of
// them silently colliding on (or racing to create) the same id. Racing
// through the public Run entry point — the same one a real second `pickle`
// process would use — this asserts both calls succeed and land on distinct,
// sequential ids, and that the resulting tree is audit-clean. This is the
// in-process complement to the ticket's acceptance test 1, which proves the
// same property across real OS processes.
func TestConcurrentTicketNewGetsDistinctIDs(t *testing.T) {
	root := newProject(t)

	const n = 5
	var wg sync.WaitGroup
	results := make([]int, n)
	captureStdout(t, func() {
		for i := 0; i < n; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = Run(nil, "test", []string{"ticket", "new", fmt.Sprintf("concurrent %d", i), "--project", "demo"})
			}()
		}
		wg.Wait()
	})

	for i, code := range results {
		if code != exitOK {
			t.Errorf("goroutine %d: ticket new exit = %d, want %d", i, code, exitOK)
		}
	}

	entries, err := os.ReadDir(filepath.Join(root, "tickets", "1-to-do"))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "T-") || !strings.HasSuffix(e.Name(), ".md") {
			continue // e.g. .gitkeep
		}
		ids[e.Name()[:5]] = true // "T-NNN" prefix of "T-NNN-slug.md"
	}
	if len(ids) != n {
		t.Errorf("got %d distinct ids, want %d (a duplicate means the lock did not serialise): %v", len(ids), n, ids)
	}

	auditOut := captureStdout(t, func() {
		if got := Run(nil, "test", []string{"board", "audit"}); got != exitOK {
			t.Errorf("board audit = %d, want %d", got, exitOK)
		}
	})
	if !strings.Contains(auditOut, "0 error(s)") {
		t.Errorf("board audit reported errors after concurrent creates:\n%s", auditOut)
	}
}
