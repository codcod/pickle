package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/codcod/pickle/internal/flow"
)

const completionUsage = "usage: pickle completion bash|zsh|fish"

// cmdSpec is one node of the command tree the completion scripts are rendered
// from (T-151 decision 2). flags are spelled exactly as typed ("--reason",
// "-n"); args are the positional words offered wherever a non-flag word is
// typed under this node. completion_test.go keeps it from drifting from usage()
// and from each command's --help.
type cmdSpec struct {
	name  string
	flags []string
	args  []string
	subs  []cmdSpec
}

func completionSpec() cmdSpec {
	var statuses []string
	for _, s := range flow.Default().States() {
		_, bare, _ := strings.Cut(s.Dir, "-") // "3-in-development" → the bare token flow.ByToken accepts
		statuses = append(statuses, bare)
	}
	return cmdSpec{subs: []cmdSpec{
		{name: "install", flags: []string{"--agent", "--build", "--claude-symlink", "--docs", "--hooks", "--in-tree", "--lint", "--no-claude", "--path", "--project", "--test", "--ticket-prefix"}},
		{name: "upgrade"},
		{name: "doctor", flags: []string{"--verbose", "-v"}},
		{name: "uninstall", flags: []string{"--dry-run", "-n", "--purge"}},
		{name: "hooks", subs: []cmdSpec{
			{name: "install", flags: []string{"--force"}},
			{name: "uninstall", flags: []string{"--dry-run", "-n"}},
			{name: "status"},
			{name: "run", subs: []cmdSpec{{name: "pre-commit"}, {name: "pre-push"}}},
		}},
		{name: "project", subs: []cmdSpec{
			{name: "add", flags: []string{"--build", "--test", "--lint", "--docs", "--branch-prefix", "--ticket-prefix", "--wip-dev", "--wip-review"}},
			{name: "list"},
			{name: "remove"},
		}},
		{name: "flow", subs: []cmdSpec{{name: "show"}, {name: "list"}}},
		{name: "ticket", subs: []cmdSpec{
			{name: "new", flags: []string{"--project", "--impact", "--complexity", "--cost", "--spawned-by", "--family"}},
			{name: "move", flags: []string{"--reason"}, args: statuses},
			{name: "set", flags: []string{"--impact", "--complexity", "--cost", "--family", "--title"}},
		}},
		{name: "board", subs: []cmdSpec{
			{name: "audit"},
			{name: "sync", flags: []string{"--dry-run"}},
			{name: "state", flags: []string{"--json"}},
			{name: "decisions", flags: []string{"--project", "--status", "--grep", "--json"}},
			{name: "metrics", flags: []string{"--project", "--as-of", "--json"}},
		}},
		{name: "changelog", subs: []cmdSpec{
			{name: "check", flags: []string{"--since", "--until", "--changelog", "--section", "--show-excluded"}},
		}},
		{name: "retro", flags: []string{"--since", "--sessions"}},
		{name: "serve", flags: []string{"--addr", "-a", "--dir"}},
		{name: "scaffold", subs: []cmdSpec{
			{name: "release", flags: []string{"--project-name", "--force", "--dry-run"}},
		}},
		{name: "completion", subs: []cmdSpec{{name: "bash"}, {name: "zsh"}, {name: "fish"}}},
		{name: "version"},
		{name: "help"},
	}}
}

// walk calls fn for every node with its space-joined path ("" for the root).
func (c cmdSpec) walk(path string, fn func(path string, c cmdSpec)) {
	fn(path, c)
	for _, s := range c.subs {
		p := s.name
		if path != "" {
			p = path + " " + s.name
		}
		s.walk(p, fn)
	}
}

func (c cmdSpec) subNames() []string {
	var n []string
	for _, s := range c.subs {
		n = append(n, s.name)
	}
	return n
}

func runCompletion(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, completionUsage)
		return exitUsage
	}
	spec := completionSpec()
	switch args[0] {
	case "bash":
		renderBash(os.Stdout, spec)
	case "zsh":
		renderZsh(os.Stdout, spec)
	case "fish":
		renderFish(os.Stdout, spec)
	default:
		fmt.Fprintln(os.Stderr, completionUsage)
		return exitUsage
	}
	return exitOK
}

// renderShHelpers writes the lookup functions bash and zsh share: each maps a
// command path to its subcommands, flags or positional words. POSIX `case`, so
// it runs on macOS's bash 3.2 (T-151 decision 6).
func renderShHelpers(w io.Writer, spec cmdSpec) {
	for _, h := range []struct {
		fn   string
		list func(cmdSpec) []string
	}{
		{"__pickle_subs", cmdSpec.subNames},
		{"__pickle_flags", func(c cmdSpec) []string { return c.flags }},
		{"__pickle_args", func(c cmdSpec) []string { return c.args }},
	} {
		fmt.Fprintf(w, "%s() {\n  case \"$1\" in\n", h.fn)
		spec.walk("", func(path string, c cmdSpec) {
			if l := h.list(c); len(l) > 0 {
				fmt.Fprintf(w, "    %q) echo %q ;;\n", path, strings.Join(l, " "))
			}
		})
		fmt.Fprint(w, "  esac\n}\n\n")
	}
}

func renderBash(w io.Writer, spec cmdSpec) {
	fmt.Fprint(w, "# bash completion for pickle — eval \"$(pickle completion bash)\"\n\n")
	renderShHelpers(w, spec)
	fmt.Fprint(w, `_pickle() {
  local cur="${COMP_WORDS[COMP_CWORD]}" p="" i w words
  for ((i = 1; i < COMP_CWORD; i++)); do
    w="${COMP_WORDS[i]}"
    case " $(__pickle_subs "$p") " in *" $w "*) p="${p:+$p }$w" ;; esac
  done
  case "$cur" in
    -*) words="$(__pickle_flags "$p")" ;;
    *) words="$(__pickle_subs "$p") $(__pickle_args "$p")" ;;
  esac
  COMPREPLY=($(compgen -W "$words" -- "$cur"))
}

complete -o default -F _pickle pickle
`)
}

// renderZsh emits a native zsh function that works both from fpath (as _pickle,
// where the file body becomes the function) and via `source <(...)`.
func renderZsh(w io.Writer, spec cmdSpec) {
	fmt.Fprint(w, "#compdef pickle\n# zsh completion for pickle — source <(pickle completion zsh)\n\n")
	renderShHelpers(w, spec)
	fmt.Fprint(w, `_pickle() {
  local p="" w i
  local -a cands
  for ((i = 2; i < CURRENT; i++)); do
    w="${words[i]}"
    [[ " $(__pickle_subs "$p") " == *" $w "* ]] && p="${p:+$p }$w"
  done
  if [[ "${words[CURRENT]}" == -* ]]; then
    cands=(${=$(__pickle_flags "$p")})
  else
    cands=(${=$(__pickle_subs "$p")} ${=$(__pickle_args "$p")})
  fi
  compadd -- $cands || _files
}

if [[ "${funcstack[1]}" == "_pickle" ]]; then
  _pickle "$@"
else
  compdef _pickle pickle
fi
`)
}

func renderFish(w io.Writer, spec cmdSpec) {
	fmt.Fprint(w, "# fish completion for pickle — pickle completion fish | source\n\n")
	for _, h := range []struct {
		fn   string
		list func(cmdSpec) []string
	}{
		{"__pickle_subs", cmdSpec.subNames},
		{"__pickle_args", func(c cmdSpec) []string { return c.args }},
	} {
		fmt.Fprintf(w, "function %s\n    switch \"$argv[1]\"\n", h.fn)
		spec.walk("", func(path string, c cmdSpec) {
			if l := h.list(c); len(l) > 0 {
				fmt.Fprintf(w, "        case %s\n            printf '%%s\\n' %s\n", fishQuote(path), strings.Join(l, " "))
			}
		})
		fmt.Fprint(w, "    end\nend\n\n")
	}
	fmt.Fprint(w, `function __pickle_path
    set -l p ''
    for w in (commandline -opc)[2..-1]
        if contains -- $w (__pickle_subs "$p")
            set p (string trim -- "$p $w")
        end
    end
    echo "$p"
end

function __pickle_is
    set -l p (__pickle_path)
    test "$p" = "$argv[1]"
end

complete -c pickle -n 'not string match -q -- "-*" (commandline -opc)[-1]' -f
complete -c pickle -a '(__pickle_subs (__pickle_path); __pickle_args (__pickle_path))'
`)
	long := regexp.MustCompile(`^--(.+)$`)
	spec.walk("", func(path string, c cmdSpec) {
		for _, f := range c.flags {
			opt := "-s " + strings.TrimPrefix(f, "-")
			if m := long.FindStringSubmatch(f); m != nil {
				opt = "-l " + m[1]
			}
			fmt.Fprintf(w, "complete -c pickle -n %s %s\n", fishQuote("__pickle_is "+fishQuote(path)), opt)
		}
	})
}

// fishQuote single-quotes s for fish, escaping \ and '.
func fishQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}
