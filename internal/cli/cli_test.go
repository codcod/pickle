package cli

import "testing"

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
		{"upgrade stub", []string{"upgrade"}, exitUnimplement},
		{"doctor stub", []string{"doctor"}, exitUnimplement},
		{"uninstall stub", []string{"uninstall"}, exitUnimplement},
		{"board sync bad flag", []string{"board", "sync", "--bogus"}, exitUsage},
		{"board no subcommand", []string{"board"}, exitUsage},
		{"board unknown subcommand", []string{"board", "xyz"}, exitUsage},
		{"ticket new missing title", []string{"ticket", "new"}, exitUsage},
		{"ticket move missing args", []string{"ticket", "move"}, exitUsage},
		{"ticket move missing status", []string{"ticket", "move", "T-001"}, exitUsage},
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
