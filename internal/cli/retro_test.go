package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// retroSandbox builds an in-tree project with one previous report, its script,
// an older report, and session directories for both hosts: the checkout's own
// Claude Code directory, a worktree's, and the pi one. It returns the root as
// pickle itself resolves it.
func retroSandbox(t *testing.T) string {
	t.Helper()
	newProject(t)
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	write := func(p, s string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	retros := filepath.Join(root, "tickets", "retros")
	write(filepath.Join(retros, "2026-09-01-x.md"), "# x\n\n## Open targets\n- target A\n## Caveats\n- c\n")
	write(filepath.Join(retros, "2026-09-01-a.py"), "print(1)\n")
	write(filepath.Join(retros, "2026-08-01-y.md"), "## Open targets\n- old target\n")

	claude := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	write(filepath.Join(claude, "projects", claudeSlug(root), "s1.jsonl"), "{}\n")
	write(filepath.Join(claude, "projects", claudeSlug(root), "u1", "subagents", "agent-1.jsonl"), "{}\n")
	write(filepath.Join(claude, "projects", claudeSlug(root)+"-wt", "s2.jsonl"), "{}\n")
	write(filepath.Join(claude, "projects", "-elsewhere", "s3.jsonl"), "{}\n")

	home := t.TempDir()
	t.Setenv("HOME", home)
	write(filepath.Join(home, ".pi", "agent", "sessions", piSlug(root), "s.jsonl"), "{}\n")
	return root
}

func runRetroCapture(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var code int
	out := captureStdout(t, func() {
		code = Run(os.DirFS(repoRoot), "test", append([]string{"retro"}, args...))
	})
	return out, code
}

func TestRetroDefault(t *testing.T) {
	root := retroSandbox(t)
	out, code := runRetroCapture(t)
	if code != exitOK {
		t.Fatalf("retro = %d, want %d", code, exitOK)
	}
	claudeDir := filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "projects", claudeSlug(root))
	for _, want := range []string{
		"- Root: " + root,
		"- demo: path " + root + ", base branch unknown, ticket prefix T",
		"- Window: since 2026-09-01, the date of the newest report",
		"> - target A",
		"tickets/retros/2026-09-01-a.py",
		claudeDir + " — 1 sessions, ",
		claudeDir + "-wt — 1 sessions, ",
		filepath.Join(os.Getenv("HOME"), ".pi", "agent", "sessions", piSlug(root)) + " — 1 sessions, ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	for _, not := range []string{"- c\n", "old target", "-elsewhere"} {
		if strings.Contains(out, not) {
			t.Errorf("output has %q\n%s", not, out)
		}
	}
	if !strings.HasPrefix(out, "## This project") {
		t.Errorf("without a question the facts lead the output:\n%s", out)
	}
	if f, m := strings.Index(out, "## This project"), strings.Index(out, "## Method"); f < 0 || m < f {
		t.Errorf("facts (%d) must come before the method (%d)", f, m)
	}
}

func TestRetroFlags(t *testing.T) {
	retroSandbox(t)
	out, code := runRetroCapture(t, "--since", "2026-07-01")
	if code != exitOK || !strings.Contains(out, "- Window: since 2026-07-01 (given with --since)") {
		t.Errorf("--since: code %d\n%s", code, out)
	}
	for _, args := range [][]string{{"--since", "07/01"}, {"a", "b"}} {
		if _, code := runRetroCapture(t, args...); code != exitUsage {
			t.Errorf("retro %v = %d, want %d", args, code, exitUsage)
		}
	}

	out, _ = runRetroCapture(t, "why X?")
	if !strings.HasPrefix(out, "## Question\n\nwhy X?\n") {
		t.Errorf("question must lead the output:\n%s", out)
	}

	extra := t.TempDir()
	out, _ = runRetroCapture(t, "--sessions", extra)
	if !strings.Contains(out, "- given with --sessions:\n  - "+extra+" — 0 sessions") {
		t.Errorf("--sessions dir not listed:\n%s", out)
	}
}

func TestRetroNoPreviousReport(t *testing.T) {
	newProject(t)
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	out, code := runRetroCapture(t)
	if code != exitOK {
		t.Fatalf("retro = %d", code)
	}
	for _, want := range []string{"- Window: no previous report", "None recorded", "none found under "} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
}

func TestRetroSlugs(t *testing.T) {
	if got := claudeSlug("/Users/x/.config/p"); got != "-Users-x--config-p" {
		t.Errorf("claudeSlug = %q", got)
	}
	if got := piSlug("/Users/x/.config/p"); got != "--Users-x-.config-p--" {
		t.Errorf("piSlug = %q", got)
	}
}
