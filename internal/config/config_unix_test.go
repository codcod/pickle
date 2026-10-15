//go:build unix

package config

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestSaveCreatesNewConfigRespectingUmask is Save's own regression test for
// F1, exercised through the caller shape install.go's writeConfig uses
// (Save on a path that does not exist yet), not just atomicfile.WriteFile in
// isolation.
func TestSaveCreatesNewConfigRespectingUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Config{}
	if err := c.AddProject(Project{Name: "demo", Path: "sub"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	path := filepath.Join(dir, FileName)
	if err := c.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("new pickle.toml mode = %o, want 600 (umask 077 applied), matching every other file pickle install creates", got)
	}
}
