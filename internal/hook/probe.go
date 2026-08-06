package hook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// probeTimeout bounds how long Probe waits for the pickle on PATH to answer.
// A hung or misbehaving binary must not hang `pickle doctor`/`hooks install`;
// 3s is generous for a process that does nothing but fail config discovery.
// A var, not a const, so TestProbeTimesOut can shrink it instead of either
// sleeping out a real 3s in every test run or faking the deadline.
var probeTimeout = 3 * time.Second

// maxVersionRunes caps the best-effort version string quoted in a warning —
// it is diagnostic text, not something that needs to be exact or unbounded.
const maxVersionRunes = 60

// Reach describes the pickle binary the installed shim will actually execute
// — the answer to "is the guard that `pickle hooks install` just reported as
// current actually reachable and capable?" (T-068). It is deliberately not
// part of State/Status: probing execs a process, and Status must stay a cheap
// filesystem read that every hooks command pays for, capability is worth the
// cost only where the guard is armed (callers gate this themselves).
type Reach struct {
	Path    string // resolved `pickle` on PATH; "" when there is none
	Self    bool   // Path is this very binary — resolved without exec'ing it
	OK      bool   // it can run `hooks run pre-commit`
	Version string // best-effort `<Path> version` line, filled only when !OK
}

// Problem renders one sentence describing why r is not healthy, or "" when it
// is. It is the single source of the wording so `hooks install`, `hooks
// status` and `doctor` cannot drift apart; each caller prepends its own
// context (e.g. "hooks: <path> is installed and current, but ...").
func (r Reach) Problem() string {
	switch {
	case r.Path == "":
		return "no pickle is on PATH — the guard is inert (the installed shim resolves `pickle` from PATH)"
	case r.OK:
		return ""
	case r.Version != "":
		return fmt.Sprintf("the pickle on PATH cannot run the guard — it is inert (%s at %s)", r.Version, r.Path)
	default:
		return fmt.Sprintf("the pickle on PATH cannot run the guard — it is inert (unknown version at %s)", r.Path)
	}
}

// Probe answers the question the installed shim can't ask itself: can the
// `pickle` first on PATH actually run the guard?
//
// The check is literally the shim's own call — `pickle hooks run
// pre-commit` — run with its working directory in a freshly created, empty
// temp directory. That is the invariant the whole design rests on: a
// hooks-aware binary finds no pickle.toml there (config.Find walks up from an
// empty dir and fails), so `hooks run` degrades and exits 0 by design
// (internal/cli/hooks.go, runHooksRun); a pre-hooks binary exits 2 on the
// unknown `hooks` verb. Measured both ways (T-068 refinement) before this was
// written: 0 on a build carrying this ticket's own predecessor, 2 on a
// released 0.2.2. Do not "simplify" this into a dedicated `pickle hooks
// probe` verb or a bare `--help`/`version` check — a new verb would report
// every hooks-aware build made before it shipped as incapable (`hooks` itself
// has never been in a tagged release), and `--help`/`version` answer a
// different question (the binary runs at all) than the one that matters here
// (it can run *this* guard without violating the shim's exit-code contract).
func Probe() Reach {
	path, err := exec.LookPath("pickle")
	if err != nil {
		return Reach{}
	}
	if self, err := sameExecutable(path); err == nil && self {
		// No exec needed: this binary obviously can run its own guard, and a
		// path *difference* alone is never a finding (a build-from-source next
		// to a Homebrew copy of the same version is a normal, healthy setup).
		return Reach{Path: path, Self: true, OK: true}
	}
	if probeCapable(path) {
		return Reach{Path: path, OK: true}
	}
	return Reach{Path: path, Version: probeVersion(path)}
}

// sameExecutable reports whether path resolves to the same file as the
// running binary.
func sameExecutable(path string) (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	a, err := os.Stat(exe)
	if err != nil {
		return false, err
	}
	b, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return os.SameFile(a, b), nil
}

// probeCapable runs `<path> hooks run pre-commit` in a throwaway empty
// directory and reports whether it exited 0. Output is discarded: an old
// binary's unknown-verb usage dump (~40 lines) must never reach the user —
// only Problem()'s one sentence should.
func probeCapable(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "pickle-hook-probe-*")
	if err != nil {
		return false // can't even set up the probe; report incapable, not a crash
	}
	defer os.RemoveAll(dir)

	cmd := exec.CommandContext(ctx, path, "hooks", "run", HookName)
	cmd.Dir = dir
	// Strip repo-pinning variables (finding F8's lesson, reused here): an
	// inherited GIT_DIR/GIT_INDEX_FILE from the calling hook must not point
	// the probed binary at a real repository — the probe dir is meant to hold
	// no pickle.toml and no git state at all.
	cmd.Env = withoutRepoEnv(os.Environ())
	return cmd.Run() == nil
}

// probeVersion best-effort reads `<path> version`'s first line, for the
// warning message only. All failures are swallowed: a probe that cannot even
// report its own version is still correctly reported as incapable by
// probeCapable, and Problem() falls back to "unknown version".
//
// Stdout is captured through a real temp **file**, not `cmd.Output()`'s
// in-memory buffer. `Output()` routes the child's stdout through an
// OS pipe drained by a goroutine that blocks until every holder of the
// pipe's write end closes it — and a version-manager shim (mise, asdf,
// direnv-style wrappers are common on PATH) that runs the real binary
// *without* `exec`-replacing itself leaves that binary as a live grandchild
// holding the pipe open long after `ctx`'s deadline kills only the direct
// child. The probe would then block for however long that grandchild takes,
// not for probeTimeout — defeating the one guarantee this package exists to
// give doctor/install/status. A plain file has no such wait: the kernel
// writes into it directly, so Wait() completing (the direct child reaped) is
// enough regardless of what any orphaned descendant still has open.
func probeVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	f, err := os.CreateTemp("", "pickle-hook-probe-version-*")
	if err != nil {
		return ""
	}
	defer os.Remove(f.Name())
	defer f.Close()

	cmd := exec.CommandContext(ctx, path, "version")
	cmd.Stdout = f
	if cmd.Run() != nil {
		return ""
	}
	out, err := os.ReadFile(f.Name())
	if err != nil {
		return ""
	}
	line := out
	if i := bytes.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	s := strings.TrimSpace(string(line))
	if r := []rune(s); len(r) > maxVersionRunes {
		s = string(r[:maxVersionRunes])
	}
	return s
}
