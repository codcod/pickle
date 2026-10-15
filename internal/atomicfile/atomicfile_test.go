package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFixture writes body to dir/name and returns its path.
func writeFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestWriteFileFollowsSymlink is WriteFile's own symlink-resolution
// regression: a symlinked target is followed rather than replaced, so the
// link itself survives and the real target picks up the update. Config's
// Save and SetPayloadVersionInPlace each carry their own integration-level
// version of this (config_test.go's TestSaveFollowsSymlink and
// TestSetPayloadVersionInPlaceFollowsSymlink) through their own callers; this
// one pins the behavior in the primitive itself.
func TestWriteFileFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := writeFixture(t, dir, "real.toml", "old\n")
	link := filepath.Join(dir, "link.toml")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := WriteFile(link, []byte("new\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("link.toml is no longer a symlink; WriteFile replaced the link instead of following it")
	}
	got, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Errorf("real.toml content = %q, want %q (the symlink target should have been updated)", got, "new\n")
	}
}

// TestWriteFileRewritesReadOnlyFile pins the other declined behaviour:
// create-temp-then-rename only needs a writable *directory*, so a read-only
// (0444) file is rewritten anyway — unlike the os.WriteFile it replaced,
// which would have failed on the file's own permission bit. This documents
// the decline (a successful rewrite), not a regression.
func TestWriteFileRewritesReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "a.toml", "old\n")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, []byte("updated\n")); err != nil {
		t.Fatalf("WriteFile: %v (expected to succeed even though the file is read-only)", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated\n" {
		t.Errorf("content = %q, want %q", got, "updated\n")
	}
}
