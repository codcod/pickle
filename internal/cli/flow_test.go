package cli

import (
	"strings"
	"testing"
)

// TestFlowShowAndList (T-073): a freshly installed project has no explicit
// flow key, so both subcommands report the default ("brine") — `list`
// printing exactly one entry today, by design (a second flow is out of scope
// until one exists).
func TestFlowShowAndList(t *testing.T) {
	newProject(t)

	for _, sub := range []string{"show", "list"} {
		out := captureStdout(t, func() {
			if got := Run(nil, "test", []string{"flow", sub}); got != exitOK {
				t.Fatalf("flow %s = %d, want %d", sub, got, exitOK)
			}
		})
		got := strings.TrimSpace(out)
		if got != "brine" {
			t.Errorf("flow %s = %q, want %q", sub, got, "brine")
		}
	}
}

// TestFlowUnknownSubcommand mirrors runProject's default case: an unknown
// verb is a usage error, not a runtime one.
func TestFlowUnknownSubcommand(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"flow", "bogus"}); got != exitUsage {
		t.Errorf("flow bogus = %d, want %d (exitUsage)", got, exitUsage)
	}
}

// TestFlowNoSubcommand mirrors runProject's no-args usage error.
func TestFlowNoSubcommand(t *testing.T) {
	newProject(t)
	if got := Run(nil, "test", []string{"flow"}); got != exitUsage {
		t.Errorf("flow (no args) = %d, want %d (exitUsage)", got, exitUsage)
	}
}
