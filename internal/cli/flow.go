package cli

import (
	"fmt"
	"os"
)

// runFlow dispatches `pickle flow <show|list>`. Both subcommands take no
// positional args and print the configured flow name: `show` because it is
// the one thing to report, `list` because exactly one flow exists today
// (rules for a future second flow belong to whatever introduces it, not
// here).
func runFlow(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle flow <show|list>")
		return exitUsage
	}
	switch args[0] {
	case "show":
		return runFlowShow(args[1:])
	case "list":
		return runFlowList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle flow: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

func runFlowShow(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle flow show")
		return exitUsage
	}
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	fmt.Println(cfg.FlowName())
	return exitOK
}

func runFlowList(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle flow list")
		return exitUsage
	}
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	fmt.Println(cfg.FlowName())
	return exitOK
}
