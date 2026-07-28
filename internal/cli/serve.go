package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codcod/pickle/internal/serve"
)

// `pickle serve` starts the read-only board dashboard. It is the only long-running
// command, and the only one that reads the whole ticket tree per request; it writes
// nothing, so it is safe to leave running while an agent moves tickets.

const serveUsage = "usage: pickle serve [--addr|-a host:port]"

// defaultAddr binds loopback only: the dashboard has no authentication and is a
// local tool. A user who wants otherwise says so with --addr and gets warned.
const defaultAddr = "127.0.0.1:8745"

// parseServeArgs validates argv and resolves the listen address. Kept separate from
// runServe so the flag contract is testable without binding a port.
func parseServeArgs(args []string) (string, int) {
	addr := defaultAddr
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--addr" || a == "-a":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pickle serve: --addr needs a value\n%s\n", serveUsage)
				return "", exitUsage
			}
			i++
			addr = args[i]
		case strings.HasPrefix(a, "--addr="):
			addr = strings.TrimPrefix(a, "--addr=")
		default:
			fmt.Fprintf(os.Stderr, "pickle serve: unknown argument %q\n%s\n", a, serveUsage)
			return "", exitUsage
		}
	}
	if strings.TrimSpace(addr) == "" {
		fmt.Fprintf(os.Stderr, "pickle serve: --addr needs a value\n%s\n", serveUsage)
		return "", exitUsage
	}
	return addr, exitOK
}

func runServe(args []string) int {
	addr, code := parseServeArgs(args)
	if code != exitOK {
		return code
	}
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errf("cannot listen on %s: %v", addr, err)
	}
	if !isLoopback(addr) {
		fmt.Fprintf(os.Stderr,
			"pickle serve: WARNING — %s is not loopback: the dashboard has no authentication,\n"+
				"  and anyone who can reach this port can read every ticket in the project.\n", addr)
	}
	fmt.Printf("pickle serve: board at http://%s (read-only; Ctrl-C to stop)\n", ln.Addr().String())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve.Serve(ctx, ln, serve.Options{Root: cfg.Root(), Cfg: cfg}); err != nil {
		return errf("%v", err)
	}
	return exitOK
}

// isLoopback reports whether addr's host is a loopback address or the empty host.
// An empty host means "all interfaces", which is the least loopback thing there
// is, so it warns. A hostname that is not an IP literal (e.g. "localhost") is
// resolved; anything unresolvable is treated as non-loopback, because warning
// about a safe address is cheap and staying silent about an exposed one is not.
func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !ip.IsLoopback() {
			return false
		}
	}
	return true
}
