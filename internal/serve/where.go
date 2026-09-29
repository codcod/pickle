package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// slugRE is the shape both a /where key and a ?from= label must have (T-134
// decisions 7 and 8): it is what keeps control characters and ANSI escapes off
// the operator's terminal, since both values can end up in the miss notice.
var slugRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// whereMatch is one (served root, child) answer to GET /where/{key} (T-134
// decision 5). Adding a field is compatible; renaming or removing one is not.
type whereMatch struct {
	Root             string `json:"root"`
	Slug             string `json:"slug"`
	Child            string `json:"child"`
	ChildPath        string `json:"child_path"`
	Prefix           string `json:"prefix"`
	Matched          string `json:"matched"` // "slug" | "child" | "prefix"
	Layout           string `json:"layout"`
	CheckedOutBranch string `json:"checked_out_branch"`
}

// whereHandler answers where a served project lives, so an agent in another
// project can go and read it (T-134). It returns a location only — never ticket
// data — and, like every other route, never writes.
type whereHandler struct {
	roots []NamedRoot
	local bool      // bound to loopback; false answers 404 to everything (decision 9)
	log   io.Writer // miss notices; nil discards

	mu   sync.Mutex
	seen map[string]bool // "from\x00key" already noticed, for once-per-asker printing
}

func (wh *whereHandler) where(w http.ResponseWriter, r *http.Request) {
	// An absolute path means nothing off this machine, and serve has no
	// authentication: off loopback the route reveals nothing, not even a notice.
	if !wh.local {
		http.NotFound(w, r)
		return
	}
	key := r.PathValue("key")
	if !slugRE.MatchString(key) {
		http.Error(w, "invalid key", http.StatusBadRequest)
		return
	}

	var matches []whereMatch
	for _, nr := range wh.roots {
		root, cfg := nr.Options.Root, nr.Options.Cfg
		slugHit := strings.EqualFold(key, nr.Slug)
		branch, probed := "", false
		add := func(m whereMatch) {
			if !probed { // one git probe per matched root, none for the rest
				branch, probed = staleBoardBranch(root, cfg), true
			}
			m.Root, m.Slug, m.Layout, m.CheckedOutBranch = root, nr.Slug, cfg.ResolvedLayout(), branch
			matches = append(matches, m)
		}
		// A plain install registers no child; it is still served, so still found.
		if slugHit && len(cfg.Projects) == 0 {
			add(whereMatch{Matched: "slug"})
		}
		for i := range cfg.Projects {
			p := &cfg.Projects[i]
			var matched string
			switch {
			case slugHit:
				matched = "slug"
			case strings.EqualFold(key, p.Name):
				matched = "child"
			case strings.EqualFold(key, p.Prefix()):
				matched = "prefix"
			default:
				continue
			}
			add(whereMatch{Child: p.Name, ChildPath: filepath.Join(root, p.Path), Prefix: p.Prefix(), Matched: matched})
		}
	}

	if len(matches) == 0 {
		wh.notice(r.URL.Query().Get("from"), key)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Matches []whereMatch `json:"matches"`
	}{matches})
}

// notice tells the operator someone looked for a project this serve does not
// have, once per (from, key) per process so a polling agent cannot flood the
// terminal. from is self-reported: a label for the human, never an identity.
func (wh *whereHandler) notice(from, key string) {
	if wh.log == nil {
		return
	}
	if !slugRE.MatchString(from) {
		from = "someone"
	}
	wh.mu.Lock()
	defer wh.mu.Unlock()
	if wh.seen == nil {
		wh.seen = map[string]bool{}
	}
	if k := from + "\x00" + key; !wh.seen[k] {
		wh.seen[k] = true
		fmt.Fprintf(wh.log, "pickle serve: %s wants to learn about %s (not served here)\n", from, key)
	}
}

// isLoopbackListener decides "local" from what was actually bound, never from
// the --addr string (T-134 decision 9).
func isLoopbackListener(ln net.Listener) bool {
	tcp, ok := ln.Addr().(*net.TCPAddr)
	return ok && tcp.IP.IsLoopback()
}
