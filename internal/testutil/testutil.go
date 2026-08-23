// Package testutil holds tiny helpers shared by other packages' tests. It
// exists so that a fact tests need to agree on lives in exactly one place,
// rather than in one copy per package (T-042 item 3).
package testutil

import (
	"path/filepath"
	"runtime"
)

// RepoRoot returns the module root — the directory whose os.DirFS exposes
// "skill/SKILL.md" and the other payload paths at the same locations the
// binary's embedded payload uses.
//
// It is computed from this very file's own compile-time source path via
// runtime.Caller, three filepath.Dir calls up from
// internal/testutil/testutil.go to the module root — correct by construction
// regardless of the test process's current working directory, unlike a
// relative guess such as filepath.Join("..", "..") (which breaks the moment a
// test process changes its CWD, e.g. internal/cli's TestMain sandbox).
//
// Caveat, recorded rather than solved: runtime.Caller returns a non-absolute
// path when the binary was built with -trimpath. This project's own test
// entry points (`just test`, CI) never pass -trimpath to `go test`, so this is
// a documented assumption, not a defect guarded against here.
func RepoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file is .../internal/testutil/testutil.go; three Dir calls reach the
	// module root: testutil/ -> internal/ -> root.
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}
