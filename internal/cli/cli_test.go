package cli

import "testing"

// The payload is not exercised by the skeleton tests; an empty FS is fine.
type emptyFS struct{}

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no args", nil, exitUsage},
		{"help", []string{"help"}, exitOK},
		{"version", []string{"version"}, exitOK},
		{"unknown command", []string{"frobnicate"}, exitUsage},
		{"install stub", []string{"install"}, exitUnimplement},
		{"board sync stub", []string{"board", "sync"}, exitUnimplement},
		{"board no subcommand", []string{"board"}, exitUsage},
		{"board unknown subcommand", []string{"board", "xyz"}, exitUsage},
		{"ticket new stub", []string{"ticket", "new"}, exitUnimplement},
		{"project no subcommand", []string{"project"}, exitUsage},
		{"project unknown subcommand", []string{"project", "xyz"}, exitUsage},
		{"project add missing args", []string{"project", "add"}, exitUsage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Run(nil, "test", tc.args); got != tc.want {
				t.Fatalf("Run(%v) = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}
