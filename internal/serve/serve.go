// Package serve implements `pickle serve`: a local, read-only web view of the
// board. It renders the same ground truth the board is generated from — the
// ticket files plus pickle.toml — so it can never disagree with tickets/BOARD.md,
// and it never writes: no handler creates, moves, rewrites or regenerates
// anything, not even a stale board. The CLI (`ticket new|move`, `board sync`)
// remains the only writer in the flow.
//
// The stack is deliberately small: stdlib net/http and html/template, assets
// embedded with go:embed (so the single binary needs no network and no asset
// directory), goldmark for ticket bodies, and one vendored copy of htmx
// (v2.0.4, from https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js, with its
// 0BSD licence beside it in static/) to poll two fragment routes. No framework,
// no router, no bundler, no CDN.
//
// Freshness is by construction rather than by cache invalidation: every request
// re-reads the ticket tree, so a page always shows the tree as it is now, and the
// htmx polling only saves the human from pressing reload.
package serve

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
	"github.com/codcod/pickle/internal/ticket"
)

//go:embed templates static
var assets embed.FS

// Options configures a dashboard: which project root to read, with which config.
type Options struct {
	Root string         // overarching project root (the directory holding tickets/)
	Cfg  *config.Config // registered children, WIP limits
}

// Handler builds the read-only dashboard's mux. Templates are parsed once, here,
// so a broken template is a startup error the user sees immediately instead of a
// 500 on some page they happen to visit later.
func Handler(opts Options) (http.Handler, error) {
	if opts.Cfg == nil {
		return nil, errors.New("serve: nil config")
	}
	tmpls, err := template.New("").Funcs(funcs).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("opening embedded static assets: %w", err)
	}
	h := &handler{opts: opts, tmpls: tmpls}

	mux := http.NewServeMux()
	// Method-qualified patterns: anything but GET/HEAD gets 405 from the mux, so
	// no write-shaped request ever reaches a handler.
	mux.HandleFunc("GET /", h.board)
	mux.HandleFunc("GET /activity", h.activity)
	mux.HandleFunc("GET /t/{id}", h.ticket)
	mux.HandleFunc("GET /fragments/board", h.boardFragment)
	mux.HandleFunc("GET /fragments/activity", h.activityFragment)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "ok")
	})
	// fs.Sub above is why this is not "/static/static/...": the embed is rooted at
	// the package directory, so the FileServer needs the subtree, and StripPrefix
	// removes the URL prefix the mux matched on.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	return mux, nil
}

// Serve runs the dashboard on an already-open listener until ctx is cancelled.
// Taking a net.Listener rather than an address keeps the caller in charge of
// binding (the CLI reports a bind failure with its own exit code; tests bind port
// 0 and never contend for a fixed port).
func Serve(ctx context.Context, ln net.Listener, opts Options) error {
	h, err := Handler(opts)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       15 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		// A fresh context: the shutdown must outlive the cancelled one, but not
		// hang forever on a wedged connection.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	err = srv.Serve(ln)
	<-done
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

type handler struct {
	opts  Options
	tmpls *template.Template
}

// load re-reads the ticket tree for one request. There is no cache to invalidate:
// re-reading is what makes the dashboard live, and a board-sized tree is a few
// hundred small files.
//
// Structural load problems (a bad filename, a missing frontmatter block) are
// deliberately not an error here: one malformed file must not blank the whole
// dashboard. The audit banner reports them, because audit.Audit re-runs the same
// load and lists each one.
func (h *handler) load() []*ticket.Ticket {
	def := flow.ForName(h.opts.Cfg.FlowName())
	tickets, _ := ticket.LoadAll(def, h.opts.Root)
	return tickets
}

func (h *handler) render(w http.ResponseWriter, name string, data page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpls.ExecuteTemplate(w, name, data); err != nil {
		// Templates stream, so by the time one fails the status line and part of
		// the body are usually already on the wire: this http.Error then appends
		// its text to a half-rendered page and the stdlib logs a "superfluous
		// WriteHeader" — which is also what a client that hangs up mid-response
		// produces. Both are cosmetic (nothing is written to disk, and a template
		// that parses at startup rarely fails at execution); rendering into a
		// bytes.Buffer first is the fix if it ever stops being cosmetic.
		// Recorded as finding N3 of the T-053 review.
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// newPage assembles the shared parts of every page: the header label and the
// health banner.
func (h *handler) newPage(title string, tickets []*ticket.Ticket) page {
	def := flow.ForName(h.opts.Cfg.FlowName())
	return page{
		Title:   title,
		Project: projectName(h.opts.Root),
		Health:  buildHealth(def, h.opts.Root, tickets, h.opts.Cfg),
	}
}

func (h *handler) board(w http.ResponseWriter, r *http.Request) {
	// "GET /" is a catch-all pattern: without this guard, /favicon.ico and every
	// mistyped path would render the board with a 200.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tickets := h.load()
	p := h.newPage("Board", tickets)
	p.Board = buildBoard(flow.ForName(h.opts.Cfg.FlowName()), tickets, h.opts.Cfg)
	h.render(w, "board.html", p)
}

func (h *handler) activity(w http.ResponseWriter, _ *http.Request) {
	tickets := h.load()
	p := h.newPage("Activity", tickets)
	p.Activity = buildActivity(flow.ForName(h.opts.Cfg.FlowName()), tickets)
	h.render(w, "activity.html", p)
}

func (h *handler) ticket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !ticket.ValidID(id) {
		http.NotFound(w, r)
		return
	}
	tickets := h.load()
	view, ok := buildTicket(flow.ForName(h.opts.Cfg.FlowName()), tickets, id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p := h.newPage(view.ID, tickets)
	p.Ticket = view
	h.render(w, "ticket.html", p)
}

// The fragment routes exist for htmx polling. They execute the very same template
// blocks the full pages embed, so a poll can never drift from a reload.
func (h *handler) boardFragment(w http.ResponseWriter, _ *http.Request) {
	tickets := h.load()
	p := h.newPage("Board", tickets)
	p.Board = buildBoard(flow.ForName(h.opts.Cfg.FlowName()), tickets, h.opts.Cfg)
	h.render(w, "board-fragment", p)
}

func (h *handler) activityFragment(w http.ResponseWriter, _ *http.Request) {
	tickets := h.load()
	p := h.newPage("Activity", tickets)
	p.Activity = buildActivity(flow.ForName(h.opts.Cfg.FlowName()), tickets)
	h.render(w, "activity-fragment", p)
}
