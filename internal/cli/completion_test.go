package cli

import (
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// usageCommandPaths parses usage()'s command lines ("  hooks run pre-commit …")
// into space-joined command paths: the leading run of lower-case words, with
// "a|b|c" alternatives expanded. Every prefix of a path counts too, so
// "ticket new" also yields "ticket".
func usageCommandPaths(t *testing.T) []string {
	t.Helper()
	out := captureStdout(t, func() { usage(os.Stdout) })
	word := regexp.MustCompile(`^[a-z][a-z|-]*$`)
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue // not a command line (continuation/description lines are indented further)
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == "pickle" { // "  pickle <command> [arguments]"
			continue
		}
		prefixes := []string{""}
		for _, f := range fields {
			if !word.MatchString(f) {
				break
			}
			var next []string
			for _, p := range prefixes {
				for _, alt := range strings.Split(f, "|") {
					q := strings.TrimSpace(p + " " + alt)
					seen[q] = true
					next = append(next, q)
				}
			}
			prefixes = next
		}
	}
	var paths []string
	for p := range seen {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return paths
}

// TestCompletionSpecMatchesUsage is T-151 decision 4(a).
func TestCompletionSpecMatchesUsage(t *testing.T) {
	var spec []string
	completionSpec().walk("", func(path string, _ cmdSpec) {
		if path != "" {
			spec = append(spec, path)
		}
	})
	slices.Sort(spec)
	if got := usageCommandPaths(t); !slices.Equal(got, spec) {
		t.Errorf("completion spec and usage() disagree on commands:\nusage: %v\nspec:  %v", got, spec)
	}
}

// TestCompletionSpecMatchesHelpFlags is T-151 decision 4(b): every command's
// spec'd flags equal those its --help prints — flag-package "  -name" lines, or
// the --name/-x tokens of a one-line usage string.
func TestCompletionSpecMatchesHelpFlags(t *testing.T) {
	exempt := map[string]string{
		"project add":    "usage says only [flags]",
		"project list":   "needs pickle.toml before reading --help",
		"project remove": "needs pickle.toml before reading --help",
		"board audit":    "needs pickle.toml before reading --help",
		"hooks run":      "positional hook name only",
		"flow show":      "takes no flags",
		"flow list":      "takes no flags",
		"version":        "takes no flags",
		"help":           "takes no flags",
	}
	pkgFlag := regexp.MustCompile(`(?m)^  -([a-z][a-z-]*)`)
	usageFlag := regexp.MustCompile(`(?:^|[\s\[(|])--?([a-z][a-z-]*)`)
	completionSpec().walk("", func(path string, c cmdSpec) {
		if path == "" || len(c.subs) > 0 || exempt[path] != "" || strings.HasPrefix(path, "completion") {
			return
		}
		args := append(strings.Fields(path), "--help")
		var stdout string
		out := captureStderr(t, func() { stdout = captureStdout(t, func() { Run(nil, "", args) }) }) + stdout
		re := usageFlag
		if strings.Contains(out, "Usage of ") {
			re = pkgFlag
		}
		got := map[string]bool{}
		for _, m := range re.FindAllStringSubmatch(out, -1) {
			got[m[1]] = true
		}
		want := map[string]bool{}
		for _, f := range c.flags {
			want[strings.TrimLeft(f, "-")] = true
		}
		var g, w []string
		for f := range got {
			g = append(g, f)
		}
		for f := range want {
			w = append(w, f)
		}
		slices.Sort(g)
		slices.Sort(w)
		if !slices.Equal(g, w) {
			t.Errorf("pickle %s: --help flags %v, completion spec %v\n--help output:\n%s", path, g, w, out)
		}
	})
}

func TestCompletionUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"completion"}, {"completion", "pwsh"}, {"completion", "bash", "zsh"}} {
		var code int
		captureStderr(t, func() { code = Run(nil, "", args) })
		if code != exitUsage {
			t.Errorf("%v: exit %d, want %d", args, code, exitUsage)
		}
	}
}

func completionScript(t *testing.T, shell string) string {
	t.Helper()
	var code int
	out := captureStdout(t, func() { code = Run(nil, "", []string{"completion", shell}) })
	if code != exitOK || out == "" {
		t.Fatalf("completion %s: exit %d, %d bytes", shell, code, len(out))
	}
	return out
}

// shells returns each interpreter for name on this machine: the PATH one, plus
// macOS's /bin/bash (3.2, T-151 decision 6) when it differs.
func shells(t *testing.T, name string) []string {
	t.Helper()
	var found []string
	if p, err := exec.LookPath(name); err == nil {
		found = append(found, p)
	}
	if name == "bash" && !slices.Contains(found, "/bin/bash") {
		if _, err := os.Stat("/bin/bash"); err == nil {
			found = append(found, "/bin/bash")
		}
	}
	if len(found) == 0 {
		t.Skipf("%s not on PATH", name)
	}
	return found
}

func TestCompletionScriptsParse(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish"} {
		t.Run(sh, func(t *testing.T) {
			script := completionScript(t, sh)
			for _, bin := range shells(t, sh) {
				cmd := exec.Command(bin, "-n")
				cmd.Stdin = strings.NewReader(script)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("%s -n: %v\n%s", bin, err, out)
				}
			}
		})
	}
}

// TestCompletionFallsBackToFiles pins review F1: where the spec offers nothing
// (a flag's value, say `install --path <TAB>`), each script hands over to the
// shell's filename completion instead of offering nothing.
func TestCompletionFallsBackToFiles(t *testing.T) {
	for sh, want := range map[string]string{
		"bash": "complete -o default -F _pickle pickle",
		"zsh":  "compadd -- $cands || _files",
		"fish": `complete -c pickle -n 'not string match -q -- "-*" (commandline -opc)[-1]' -f`,
	} {
		if !strings.Contains(completionScript(t, sh), want) {
			t.Errorf("%s script lacks the filename fallback %q", sh, want)
		}
	}
	for _, bin := range shells(t, "bash") {
		out, err := exec.Command(bin, "-c", completionScript(t, "bash")+"\ncomplete -p pickle").CombinedOutput()
		if err != nil || !strings.Contains(string(out), "-o default") {
			t.Errorf("%s: complete -p pickle = %q (err %v), want -o default", bin, out, err)
		}
	}
}

func TestCompletionBashBehaviour(t *testing.T) {
	script := completionScript(t, "bash")
	cases := []struct{ words, cword, want string }{
		{"pickle tic", "1", "ticket"},
		{"pickle ticket mo", "2", "move"},
		{"pickle ticket move T-1 in-", "4", "in-development in-review"},
		{"pickle ticket move --re", "3", "--reason"},
		{"pickle hooks run pre-", "3", "pre-commit pre-push"},
	}
	for _, bin := range shells(t, "bash") {
		for _, c := range cases {
			cmd := exec.Command(bin, "-c", script+"\nCOMP_WORDS=("+c.words+"); COMP_CWORD="+c.cword+"; _pickle; echo \"${COMPREPLY[*]}\"")
			out, err := cmd.CombinedOutput()
			if got := strings.TrimSpace(string(out)); err != nil || got != c.want {
				t.Errorf("%s: %q → %q (err %v), want %q", bin, c.words, got, err, c.want)
			}
		}
	}
}
