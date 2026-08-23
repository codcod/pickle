package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubPickleAt writes a fake `pickle` into dir that answers `hooks run
// pre-commit` with rc, and `version` with a fixed string — a stand-in for a
// binary that is (rc 0) or is not (any other code) able to run the guard.
func stubPickleAt(t *testing.T, dir string, rc int) {
	t.Helper()
	body := fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\n  hooks) exit %d ;;\n  version) echo \"pickle 0.2.2\" ;;\n  *) exit %d ;;\nesac\n", rc, rc)
	if err := writeExecutable(filepath.Join(dir, "pickle"), body); err != nil {
		t.Fatal(err)
	}
}

func TestProbeCapablePickle(t *testing.T) {
	bin := t.TempDir()
	stubPickleAt(t, bin, 0)
	t.Setenv("PATH", bin)

	r := Probe()
	if !r.OK {
		t.Fatalf("Probe() = %+v, want OK", r)
	}
	if r.Self {
		t.Errorf("Probe() reported Self for an unrelated stub: %+v", r)
	}
	if r.Problem() != "" {
		t.Errorf("Problem() = %q, want empty for a capable pickle", r.Problem())
	}
	if !strings.HasSuffix(r.Path, "pickle") {
		t.Errorf("Path = %q, want it to resolve the stub", r.Path)
	}
}

// TestProbeCapableOnExitOne proves exit 1 — the guard ran and found a
// violation — reads as capable, not incapable: exit 1 is proof the verb
// dispatched, which is exactly what Probe is trying to answer (T-071 R3).
func TestProbeCapableOnExitOne(t *testing.T) {
	bin := t.TempDir()
	stubPickleAt(t, bin, 1)
	t.Setenv("PATH", bin)

	r := Probe()
	if !r.OK {
		t.Fatalf("Probe() = %+v, want OK for exit 1 (ran, found a violation)", r)
	}
	if r.Problem() != "" {
		t.Errorf("Problem() = %q, want empty for a capable pickle", r.Problem())
	}
}

func TestProbeIncapablePickle(t *testing.T) {
	bin := t.TempDir()
	stubPickleAt(t, bin, 2) // mirrors an older pickle's exit 2 on the (then) unknown `hooks` verb
	t.Setenv("PATH", bin)

	r := Probe()
	if r.OK {
		t.Fatalf("Probe() = %+v, want !OK", r)
	}
	if r.Version != "pickle 0.2.2" {
		t.Errorf("Version = %q, want the stub's `version` output", r.Version)
	}
	p := r.Problem()
	if !strings.Contains(p, "inert") || !strings.Contains(p, "pickle 0.2.2") || !strings.Contains(p, r.Path) {
		t.Errorf("Problem() = %q, want it to name both the version and the path", p)
	}
}

// TestProbeIncapableNoVersion covers a stub that fails even `version` (an
// exotic or broken binary) — Probe must still report incapable, falling back
// to "unknown version" rather than erroring out.
func TestProbeIncapableNoVersion(t *testing.T) {
	bin := t.TempDir()
	if err := writeExecutable(filepath.Join(bin, "pickle"), "#!/bin/sh\nexit 2\n"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	r := Probe()
	if r.OK {
		t.Fatalf("Probe() = %+v, want !OK", r)
	}
	if r.Version != "" {
		t.Errorf("Version = %q, want empty when `version` also fails", r.Version)
	}
	if !strings.Contains(r.Problem(), "unknown version") {
		t.Errorf("Problem() = %q, want the unknown-version fallback", r.Problem())
	}
}

func TestProbeNoPickleOnPATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty — nothing resolves
	r := Probe()
	if r.Path != "" || r.OK {
		t.Fatalf("Probe() = %+v, want an empty, unhealthy Reach", r)
	}
	if !strings.Contains(r.Problem(), "no pickle is on PATH") {
		t.Errorf("Problem() = %q, want it to say pickle is missing", r.Problem())
	}
}

// TestProbeTimesOut proves the probe does not hang on a stuck binary: a stub
// that sleeps well past probeTimeout must still return promptly and report
// incapable, never block the caller (hooks install/status, doctor) waiting on
// it. probeTimeout is shrunk for the duration of the test so this doesn't cost
// every `go test` run a real multi-second wait; the stub's PATH keeps the real
// PATH appended after it so its `sleep` resolves (the probe's own PATH is
// still restricted to the stub, since `pickle` is looked up before any exec).
func TestProbeTimesOut(t *testing.T) {
	old := probeTimeout
	probeTimeout = 100 * time.Millisecond
	t.Cleanup(func() { probeTimeout = old })

	bin := t.TempDir()
	if err := writeExecutable(filepath.Join(bin, "pickle"), "#!/bin/sh\nsleep 5\n"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	start := time.Now()
	r := Probe()
	elapsed := time.Since(start)

	if r.OK {
		t.Errorf("Probe() = %+v, want !OK for a binary that never answers", r)
	}
	if elapsed > 4*time.Second {
		t.Errorf("Probe() took %s, want it bounded by the shrunk probeTimeout, well under the stub's 5s sleep", elapsed)
	}
}

// TestProbeSelfShortCircuit proves the same-file case is answered without an
// exec: the fake `pickle` on PATH is a symlink to this very test binary, and
// the assertion that carries the proof is `Self` — it is set on exactly one
// path through Probe, the one that returns before any exec. (`OK` alone would
// prove nothing here: a Go test binary handed positional arguments generally
// still exits 0, so an actual exec of this symlink could also have produced
// OK.)
func TestProbeSelfShortCircuit(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("os.Executable unavailable")
	}
	bin := t.TempDir()
	link := filepath.Join(bin, "pickle")
	if err := os.Symlink(exe, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	t.Setenv("PATH", bin)

	r := Probe()
	if !r.Self {
		t.Errorf("Probe() = %+v, want Self for a PATH pickle that is this binary", r)
	}
	if !r.OK {
		t.Errorf("Probe() = %+v, want OK when Self", r)
	}
	if r.Problem() != "" {
		t.Errorf("Problem() = %q, want empty when Self", r.Problem())
	}
}
