package cli

// T-009: the install --agent contract at the CLI boundary — value parsing,
// the claude-only default, and the --no-claude deprecation shim.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr lives in cli_test.go (T-043 unified it with captureStdout into
// one helper — this file used to carry a verbatim, independently-defective
// clone of the same function).

func TestInstallAgentFlag(t *testing.T) {
	payload := os.DirFS(repoRoot)

	t.Run("unknown agent is a usage error and writes nothing", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		var code int
		_ = captureStderr(t, func() {
			code = Run(payload, "test", []string{"install", "--project", "demo", "--agent", "zed"})
		})
		if code != exitUsage {
			t.Fatalf("exit = %d, want %d", code, exitUsage)
		}
		if _, err := os.Lstat(filepath.Join(root, "pickle.toml")); !os.IsNotExist(err) {
			t.Error("a rejected --agent still wrote pickle.toml")
		}
	})

	t.Run("multi-agent set lays down each slice", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		_ = captureStdout(t, func() {
			if code := Run(payload, "test", []string{"install", "--project", "demo", "--agent", "claude,opencode,pi"}); code != exitOK {
				t.Fatalf("exit = %d, want %d", code, exitOK)
			}
		})
		for _, f := range []string{
			"opencode.jsonc",
			".pi/extensions/docs-readability.ts",
			".pi/extensions/pickle-guardrails.ts",
			".claude/skills/ticket-flow",
			".agents/skills/ticket-flow/resources/docs-readability.prompt.md",
		} {
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(f))); err != nil {
				t.Errorf("expected %s after --agent claude,opencode,pi: %v", f, err)
			}
		}
	})

	t.Run("default is claude only", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		_ = captureStdout(t, func() {
			if code := Run(payload, "test", []string{"install", "--project", "demo"}); code != exitOK {
				t.Fatalf("exit = %d, want %d", code, exitOK)
			}
		})
		if _, err := os.Lstat(filepath.Join(root, ".claude/skills/ticket-flow")); err != nil {
			t.Errorf("default install missing the claude view: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(root, "opencode.jsonc")); !os.IsNotExist(err) {
			t.Error("default install wrote opencode.jsonc")
		}
		if _, err := os.Lstat(filepath.Join(root, ".pi")); !os.IsNotExist(err) {
			t.Error("default install wrote .pi/")
		}
	})

	t.Run("no-claude still works but warns deprecated", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		var stderr string
		_ = captureStdout(t, func() {
			stderr = captureStderr(t, func() {
				if code := Run(payload, "test", []string{"install", "--project", "demo", "--no-claude"}); code != exitOK {
					t.Fatalf("exit = %d, want %d", code, exitOK)
				}
			})
		})
		if !strings.Contains(stderr, "deprecated") || !strings.Contains(stderr, "--agent") {
			t.Errorf("expected a deprecation warning pointing at --agent, got: %q", stderr)
		}
		if _, err := os.Lstat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
			t.Error("--no-claude still installed the claude view")
		}
	})

	t.Run("existing opencode.jsonc surfaces the manual-merge note", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.WriteFile(filepath.Join(root, "opencode.jsonc"), []byte("// mine\n{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out := captureStdout(t, func() {
			if code := Run(payload, "test", []string{"install", "--project", "demo", "--agent", "opencode"}); code != exitOK {
				t.Fatalf("exit = %d, want %d", code, exitOK)
			}
		})
		if !strings.Contains(out, "never merges JSONC") {
			t.Errorf("expected the manual-merge note on stdout, got:\n%s", out)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "opencode.jsonc")); string(b) != "// mine\n{}\n" {
			t.Errorf("existing opencode.jsonc was rewritten:\n%s", b)
		}
	})
}
