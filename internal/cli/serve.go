package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/serve"
)

// `pickle serve` starts the read-only board dashboard. It is the only long-running
// command, and the only one that reads the whole ticket tree per request; it writes
// nothing, so it is safe to leave running while an agent moves tickets.

const serveUsage = "usage: pickle serve [--addr|-a host:port] [--dir [name=]path ...]"

// defaultAddr binds loopback only: the dashboard has no authentication and is a
// local tool. A user who wants otherwise says so with --addr and gets warned.
const defaultAddr = "127.0.0.1:8745"

// dirArg is one repeated --dir value, split on its optional "name=" prefix
// (T-127). Name is "" when the caller did not pin a slug explicitly; runServe
// then defaults it to filepath.Base of the resolved root.
type dirArg struct {
	Name, Path string
}

// parseServeArgs validates argv and resolves the listen address plus every
// repeated --dir. Kept separate from runServe so the flag contract is
// testable without binding a port or touching a filesystem (T-127: dirArg
// values are not resolved to a root here — that needs config.Find, which
// touches disk, so it stays in runServe).
func parseServeArgs(args []string) (addr string, dirs []dirArg, code int) {
	addr = defaultAddr
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--addr" || a == "-a":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pickle serve: --addr needs a value\n%s\n", serveUsage)
				return "", nil, exitUsage
			}
			i++
			addr = args[i]
		case strings.HasPrefix(a, "--addr="):
			addr = strings.TrimPrefix(a, "--addr=")
		case a == "--dir":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pickle serve: --dir needs a value\n%s\n", serveUsage)
				return "", nil, exitUsage
			}
			i++
			d, ok := parseDirArg(args[i])
			if !ok {
				fmt.Fprintf(os.Stderr, "pickle serve: --dir needs a value\n%s\n", serveUsage)
				return "", nil, exitUsage
			}
			dirs = append(dirs, d)
		case strings.HasPrefix(a, "--dir="):
			d, ok := parseDirArg(strings.TrimPrefix(a, "--dir="))
			if !ok {
				fmt.Fprintf(os.Stderr, "pickle serve: --dir needs a value\n%s\n", serveUsage)
				return "", nil, exitUsage
			}
			dirs = append(dirs, d)
		default:
			fmt.Fprintf(os.Stderr, "pickle serve: unknown argument %q\n%s\n", a, serveUsage)
			return "", nil, exitUsage
		}
	}
	if strings.TrimSpace(addr) == "" {
		fmt.Fprintf(os.Stderr, "pickle serve: --addr needs a value\n%s\n", serveUsage)
		return "", nil, exitUsage
	}
	return addr, dirs, exitOK
}

// parseDirArg splits one --dir value on its first "=" into an explicit slug
// and a path ("name=path"), or leaves Name empty for a bare path. ok is false
// for an empty value, or a "name=" with nothing after the "=" — both are the
// same "needs a value" usage error as a bare --dir.
func parseDirArg(v string) (d dirArg, ok bool) {
	if strings.TrimSpace(v) == "" {
		return dirArg{}, false
	}
	if name, path, found := strings.Cut(v, "="); found {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(path) == "" {
			return dirArg{}, false
		}
		return dirArg{Name: name, Path: path}, true
	}
	return dirArg{Path: v}, true
}

func runServe(args []string) int {
	addr, dirs, code := parseServeArgs(args)
	if code != exitOK {
		return code
	}

	if len(dirs) == 0 {
		return runServeSingle(addr)
	}
	return runServeMulti(addr, dirs)
}

// runServeSingle is byte-for-byte the pre-T-127 behaviour: cwd resolves one
// pickle.toml, one root, classic (unprefixed) routes. Never touched by --dir.
func runServeSingle(addr string) int {
	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errf("cannot listen on %s: %v", addr, err)
	}
	warnIfNotLoopback(addr, false)
	// T-108: name the resolved layout on the one line most likely to be seen —
	// in-tree is the layout that can show a stale ticket status, so saying so
	// up front costs nothing, and the in-page banner (staleBoardBranch) covers
	// the case this line cannot: a long-running serve whose terminal has
	// scrolled away by the time the status actually goes stale.
	fmt.Printf("pickle serve: board at http://%s (%s layout; read-only; Ctrl-C to stop)\n",
		ln.Addr().String(), cfg.ResolvedLayout())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve.Serve(ctx, ln, serve.Options{Root: cfg.Root(), Cfg: cfg}); err != nil {
		return errf("%v", err)
	}
	return exitOK
}

// runServeMulti resolves every --dir into a serve.NamedRoot and serves them
// all from one listener (T-127). Any --dir, even exactly one, takes this path
// — an accepted, opt-in URL change (decision 3): routes become "/p/{slug}/"
// and "/" becomes an index, which never happens to someone who passes no
// --dir at all.
func runServeMulti(addr string, dirs []dirArg) int {
	roots, code := resolveNamedRoots(dirs)
	if code != exitOK {
		return code
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errf("cannot listen on %s: %v", addr, err)
	}
	warnIfNotLoopback(addr, true)

	fmt.Printf("pickle serve: %d boards at http://%s (Ctrl-C to stop)\n", len(roots), ln.Addr().String())
	for _, r := range roots {
		fmt.Printf("pickle serve:   %s at http://%s/p/%s/ (%s layout)\n",
			r.Slug, ln.Addr().String(), r.Slug, r.Options.Cfg.ResolvedLayout())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serve.ServeMulti(ctx, ln, roots); err != nil {
		return errf("%v", err)
	}
	return exitOK
}

// resolveNamedRoots turns every --dir into a serve.NamedRoot: each path
// resolves to its nearest pickle.toml exactly as loadConfig resolves cwd
// today (config.Find walks upward), so --dir may point at a subdirectory of
// a project (decision 4). A duplicate slug — default or explicit — is a
// startup error before any listener is opened (decision 5), never a silent
// overwrite.
func resolveNamedRoots(dirs []dirArg) ([]serve.NamedRoot, int) {
	roots := make([]serve.NamedRoot, 0, len(dirs))
	slugRoot := make(map[string]string, len(dirs)) // slug -> the root it came from, for the collision message
	for _, d := range dirs {
		abs, err := filepath.Abs(d.Path)
		if err != nil {
			return nil, errf("--dir %s: %v", d.Path, err)
		}
		cfgPath, err := config.Find(abs)
		if err != nil {
			return nil, errf("--dir %s: %v", d.Path, err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return nil, errf("--dir %s: %v", d.Path, err)
		}
		root := cfg.Root()
		slug := d.Name
		if slug == "" {
			slug = filepath.Base(root)
		}
		if prev, dup := slugRoot[slug]; dup {
			fmt.Fprintf(os.Stderr,
				"pickle serve: duplicate project name %q (from %s and %s) — use --dir name=path to disambiguate\n",
				slug, prev, root)
			return nil, exitUsage
		}
		slugRoot[slug] = root
		roots = append(roots, serve.NamedRoot{Slug: slug, Options: serve.Options{Root: root, Cfg: cfg}})
	}
	return roots, exitOK
}

// warnIfNotLoopback is isLoopback's caller-facing half: the message's scope
// ("the project" vs "every served project") tracks runServeMulti's own mode
// switch (decision 8) — tied to whether any --dir was passed, never to a
// dirs count threshold, so the wording and the routing mode can never say two
// different things.
func warnIfNotLoopback(addr string, multi bool) {
	if isLoopback(addr) {
		return
	}
	scope := "every ticket in the project"
	if multi {
		scope = "every ticket in every served project"
	}
	fmt.Fprintf(os.Stderr,
		"pickle serve: WARNING — %s is not loopback: the dashboard has no authentication,\n"+
			"  and anyone who can reach this port can read %s.\n", addr, scope)
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
