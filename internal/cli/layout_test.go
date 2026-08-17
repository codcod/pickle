// T-108: `pickle install`'s layout contract at the CLI boundary. The umbrella
// layout is the default and never registers a child; --in-tree is the only
// way to select the in-tree layout, and the layout is always recorded
// explicitly rather than inferred (decision 1).

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

func TestInstallLayoutDefaultIsUmbrellaWithNoChild(t *testing.T) {
	payload := os.DirFS(repoRoot)
	root := t.TempDir()
	t.Chdir(root)

	out := captureStdout(t, func() {
		if code := Run(payload, "test", []string{"install"}); code != exitOK {
			t.Fatalf("install = %d, want %d", code, exitOK)
		}
	})
	if !strings.Contains(out, "no child registered yet") {
		t.Errorf("expected the no-child message on stdout, got:\n%s", out)
	}

	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Layout != config.LayoutUmbrella {
		t.Errorf("Layout = %q, want %q", cfg.Layout, config.LayoutUmbrella)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("Projects = %v, want none", cfg.Projects)
	}
}

func TestInstallInTreeRegistersTheSoleChildAtRoot(t *testing.T) {
	payload := os.DirFS(repoRoot)
	root := t.TempDir()
	t.Chdir(root)

	captureStdout(t, func() {
		if code := Run(payload, "test", []string{"install", "--in-tree", "--project", "demo"}); code != exitOK {
			t.Fatalf("install --in-tree = %d, want %d", code, exitOK)
		}
	})

	cfg, err := config.Load(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Layout != config.LayoutInTree {
		t.Errorf("Layout = %q, want %q", cfg.Layout, config.LayoutInTree)
	}
	if len(cfg.Projects) != 1 || cfg.Projects[0].Path != "." {
		t.Errorf("Projects = %v, want exactly one at \".\"", cfg.Projects)
	}
}

func TestInstallInTreeConflictsWithAnExplicitPath(t *testing.T) {
	payload := os.DirFS(repoRoot)
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t, func() {
		if code := Run(payload, "test", []string{"install", "--in-tree", "--path", "sub"}); code != exitUsage {
			t.Fatalf("install --in-tree --path sub = %d, want %d", code, exitUsage)
		}
	})
	if !strings.Contains(stderr, "sub") {
		t.Errorf("expected the conflict message to name the offending path, got: %q", stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, config.FileName)); !os.IsNotExist(err) {
		t.Error("a refused install still wrote pickle.toml")
	}
}

func TestInstallBarePathDotIsRefused(t *testing.T) {
	payload := os.DirFS(repoRoot)
	root := t.TempDir()
	t.Chdir(root)

	stderr := captureStderr(t, func() {
		if code := Run(payload, "test", []string{"install", "--path", "."}); code != exitUsage {
			t.Fatalf("install --path . = %d, want %d", code, exitUsage)
		}
	})
	if !strings.Contains(stderr, "--in-tree") {
		t.Errorf("expected the refusal to point at --in-tree, got: %q", stderr)
	}
}

// TestInstallChildFlagsWithoutPathAreRefused pins the gap T-108 review found:
// --project (or --build/--test/--lint/--docs) alone, with neither --path nor
// --in-tree, used to silently install the umbrella layout while the final
// message still claimed a child had been registered. Silently discarding an
// explicit flag is exactly the guess this ticket removes elsewhere, so it is
// refused instead.
func TestInstallChildFlagsWithoutPathAreRefused(t *testing.T) {
	payload := os.DirFS(repoRoot)
	root := t.TempDir()
	t.Chdir(root)

	stderr := captureStderr(t, func() {
		if code := Run(payload, "test", []string{"install", "--project", "demo"}); code != exitUsage {
			t.Fatalf("install --project demo (no --path/--in-tree) = %d, want %d", code, exitUsage)
		}
	})
	if !strings.Contains(stderr, "--path") || !strings.Contains(stderr, "--in-tree") {
		t.Errorf("expected the refusal to name both --path and --in-tree, got: %q", stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, config.FileName)); !os.IsNotExist(err) {
		t.Error("a refused install still wrote pickle.toml")
	}
}
