package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/install"
)

// replaceView swaps installFixture's Claude view symlink for body: a copy of
// the skill dir when body is "", otherwise a placeholder file holding body.
func replaceView(t *testing.T, root, body string) {
	t.Helper()
	view := filepath.Join(root, filepath.FromSlash(install.ClaudeSkillLink))
	if err := os.Remove(view); err != nil {
		t.Fatal(err)
	}
	if body == "" {
		if err := os.CopyFS(view, os.DirFS(filepath.Join(root, filepath.FromSlash(install.SkillDir)))); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(view, install.ClaudeViewSentinel), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.WriteFile(view, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// T-143: a copy view is a supported state; a placeholder is an error.
func TestCheckClaudeViewCopyAndPlaceholder(t *testing.T) {
	root := installFixture(t)
	replaceView(t, root, "")
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Fatalf("copy view: errors %v", res.Errors)
	}
	want := "claude view present (" + install.ClaudeSkillLink + ", copy)"
	found := false
	for _, p := range res.Passed {
		found = found || p == want
	}
	if !found {
		t.Errorf("missing passed line %q in %v", want, res.Passed)
	}

	root = installFixture(t)
	replaceView(t, root, install.ClaudeSkillTarget)
	res = Check(root, "test-ver", os.DirFS(payloadRoot()))
	if !hasErrContaining(res.Errors, "symlink placeholder") {
		t.Errorf("placeholder view: errors %v", res.Errors)
	}
}

// T-143: a CLAUDE.md placeholder warns instead of erroring on missing markers.
func TestCheckClaudeMDPlaceholderWarns(t *testing.T) {
	root := installFixture(t)
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(install.ClaudeMDTarget), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Check(root, "test-ver", os.DirFS(payloadRoot()))
	if len(res.Errors) != 0 {
		t.Errorf("errors %v", res.Errors)
	}
	found := false
	for _, w := range res.Warnings {
		found = found || strings.Contains(w, "CLAUDE.md is a symlink placeholder")
	}
	if !found {
		t.Errorf("no placeholder warning in %v", res.Warnings)
	}
}
