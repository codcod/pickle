package serve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/ticket"
)

// --- fixtures ---------------------------------------------------------------

func testCfg() *config.Config {
	return &config.Config{Projects: []config.Project{
		{Name: "demo", Path: ".", WIPInDevelopment: 1, WIPInReview: 1},
	}}
}

type fixture struct {
	dir, id, title, impact string
	depends, spawned       string
	history                []string
	body                   string
}

func (f fixture) text() string {
	dep, spawn := f.depends, f.spawned
	if dep == "" {
		dep = "[]"
	}
	if spawn == "" {
		spawn = "[]"
	}
	hist := strings.Join(f.history, "\n")
	return fmt.Sprintf(`---
id: %s
title: %s
project: demo
depends-on: %s
spawned-by: %s
impact: %s
complexity: medium
cost: M
---

# %s — %s

## Description

%s

## History

%s
`, f.id, f.title, dep, spawn, f.impact, f.id, f.title, f.body, hist)
}

// newTree writes a fixture tickets/ tree plus a *rendered* BOARD.md.
//
// The board matters: audit.Audit (behind the health banner) compares BOARD.md to a
// fresh render, so a fixture without one reports a board error on every page and
// every "an audit error is shown" assertion would pass for the wrong reason.
func newTree(t *testing.T, fixtures ...fixture) string {
	t.Helper()
	root := t.TempDir()
	for _, s := range ticket.Statuses {
		if err := os.MkdirAll(filepath.Join(root, "tickets", s.Dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range fixtures {
		p := filepath.Join(root, "tickets", f.dir, f.id+"-slug.md")
		if err := os.WriteFile(p, []byte(f.text()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tickets, issues := ticket.LoadAll(root)
	if len(issues) > 0 {
		t.Fatalf("fixture load issues: %v", issues)
	}
	text := board.Render(tickets, testCfg(), time.Now().Format("2006-01-02"))
	if err := os.WriteFile(filepath.Join(root, "tickets", "BOARD.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// standardTree is the tree most tests use: a dependency pair, a spawned ticket, a
// dropped one with a reason, and a done one with a merge line.
func standardTree(t *testing.T) string {
	t.Helper()
	return newTree(t,
		fixture{dir: "1-to-do", id: "T-001", title: "low impact idea", impact: "low",
			history: []string{"- 2026-07-20 — created (TO DO). source: test"}},
		fixture{dir: "1-to-do", id: "T-002", title: "critical idea", impact: "critical",
			depends: "[T-004]", body: "## Sub-heading\n\n- a list item\n",
			history: []string{"- 2026-07-21 — created (TO DO). source: test"}},
		fixture{dir: "1-to-do", id: "T-003", title: "medium idea", impact: "medium",
			spawned: "[T-002]",
			history: []string{"- 2026-07-22 — created (TO DO). source: review of T-002"}},
		fixture{dir: "3-in-development", id: "T-004", title: "being built", impact: "high",
			history: []string{
				"- 2026-07-19 — created (TO DO). source: test",
				"- 2026-07-23 — TO DO → READY: plan complete",
				"- 2026-07-24 — READY → IN DEVELOPMENT: picked up",
			}},
		fixture{dir: "6-done", id: "T-005", title: "shipped", impact: "medium",
			history: []string{
				"- 2026-07-18 — created (TO DO). source: test",
				"- 2026-07-25 — IN REVIEW → DONE: review clean",
				"- 2026-07-26 — merged to main (abc1234)",
			}},
	)
}

func newHandler(t *testing.T, root string) http.Handler {
	t.Helper()
	h, err := Handler(Options{Root: root, Cfg: testCfg()})
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	return h
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// --- routes -----------------------------------------------------------------

func TestBoardPage(t *testing.T) {
	h := newHandler(t, standardTree(t))
	rec := get(t, h, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"T-001", "T-002", "T-003", "T-004", "T-005", "IN DEVELOPMENT", "TO DO", "demo"} {
		if !strings.Contains(body, want) {
			t.Errorf("GET / is missing %q", want)
		}
	}
	// TO DO is impact-ordered, exactly as BOARD.md renders it.
	if got := indexOrder(body, "T-002", "T-003", "T-001"); !got {
		t.Error("TO DO tickets are not in impact order (want T-002 critical, T-003 medium, T-001 low)")
	}
	// WIP badge for the one in-development ticket, at its limit of 1.
	if !strings.Contains(body, "1/1") {
		t.Error("board is missing the in-development WIP badge 1/1")
	}
	// A merged DONE ticket shows its merge line.
	if !strings.Contains(body, "merged to main (abc1234)") {
		t.Error("board is missing the DONE merge line")
	}
}

func TestTicketPage(t *testing.T) {
	h := newHandler(t, standardTree(t))
	rec := get(t, h, "/t/T-002")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /t/T-002 = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// goldmark rendered the body's markdown.
	if !strings.Contains(body, "<h2") || !strings.Contains(body, "Sub-heading") {
		t.Error("ticket body markdown was not rendered to HTML")
	}
	if !strings.Contains(body, "<li>a list item</li>") {
		t.Error("ticket body list was not rendered")
	}
	// Forward edge (depends-on) as a link, and the frontmatter is not dumped as prose.
	if !strings.Contains(body, `href="/t/T-004"`) {
		t.Error("depends-on T-004 is not a link")
	}
	if strings.Contains(body, "complexity: medium") {
		t.Error("frontmatter block leaked into the rendered body (it should be stripped)")
	}
	// Reverse edges: T-003 was spawned by T-002.
	if !strings.Contains(body, `href="/t/T-003"`) {
		t.Error("reverse lineage edge (spawned T-003) is missing")
	}
}

func TestTicketPageReverseBlocksEdge(t *testing.T) {
	h := newHandler(t, standardTree(t))
	body := get(t, h, "/t/T-004").Body.String()
	// T-002 depends on T-004, so T-004 "blocks" T-002 — a fact no ticket file states.
	if !strings.Contains(body, "blocks") || !strings.Contains(body, `href="/t/T-002"`) {
		t.Error("T-004 page does not show that it blocks T-002")
	}
}

func TestActivityPage(t *testing.T) {
	h := newHandler(t, standardTree(t))
	rec := get(t, h, "/activity")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /activity = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	// Newest first: the 07-26 merge precedes the 07-18 creation.
	if !indexOrder(body, "2026-07-26", "2026-07-24", "2026-07-18") {
		t.Error("activity timeline is not newest-first")
	}
	if !strings.Contains(body, "merged to main (abc1234)") {
		t.Error("activity timeline dropped the merge line")
	}
	if !strings.Contains(body, "review of T-002") {
		t.Error("activity timeline dropped a created line's source")
	}
}

func TestFragmentsMatchPagesAndAreNotWholeDocuments(t *testing.T) {
	h := newHandler(t, standardTree(t))
	for _, tc := range []struct{ frag, page string }{
		{"/fragments/board", "/"},
		{"/fragments/activity", "/activity"},
	} {
		rec := get(t, h, tc.frag)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", tc.frag, rec.Code)
		}
		frag := rec.Body.String()
		if strings.Contains(frag, "<html") || strings.Contains(frag, "<!DOCTYPE") {
			t.Errorf("%s returned a whole document, not a fragment", tc.frag)
		}
		if !strings.Contains(frag, "hx-get="+`"`+tc.frag+`"`) {
			t.Errorf("%s does not re-arm its own polling", tc.frag)
		}
		// The fragment is the same block the page embeds: every id on the page's
		// fragment area must appear in the fragment too.
		for _, id := range []string{"T-001", "T-005"} {
			if !strings.Contains(frag, id) {
				t.Errorf("%s is missing %s (page and fragment have drifted)", tc.frag, id)
			}
		}
	}
}

func TestStaticAssetsAndHealthz(t *testing.T) {
	h := newHandler(t, standardTree(t))
	for _, tc := range []struct {
		path, wantSubstr string
	}{
		{"/static/styles.css", "--accent"},
		{"/static/htmx.min.js", "htmx"},
		{"/healthz", "ok"},
		// The color-scheme meta is what paints browser chrome correctly before
		// the stylesheet loads, and what survives it failing to load — so it
		// needs its own assertion, not just the stylesheet's.
		{"/", `name="color-scheme"`},
	} {
		rec := get(t, h, tc.path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", tc.path, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), tc.wantSubstr) {
			t.Errorf("GET %s does not contain %q", tc.path, tc.wantSubstr)
		}
	}
}

// --- stylesheet palettes (T-054) --------------------------------------------

// cssComment matches a /* … */ comment. Every scan below runs on a
// comment-stripped copy: a comment naming the media query, or one containing a
// brace or a "--foo:" token, otherwise steers the extraction and makes this
// test report confident nonsense. The file's own header comment already
// discusses theming, so this is not hypothetical.
var cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// cssVarDecl matches a custom-property declaration, capturing the name and the
// value up to the terminating ";" or "}". Custom properties are case-sensitive
// and uppercase is legal, so the name class is not lowercase-only.
var cssVarDecl = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:\s*([^;}]*)`)

// lightMedia matches the light palette's query without pinning its formatting:
// "@media(prefers-color-scheme:light)" and "@media screen and (…)" are both
// legitimate CSS and must not read as "there is no light palette".
var lightMedia = regexp.MustCompile(`@media[^{]*prefers-color-scheme\s*:\s*light`)

// colorSchemeDecl captures a color-scheme value, up to ";" or the end of its
// block — so a value may legally omit its final semicolon.
var colorSchemeDecl = regexp.MustCompile(`color-scheme\s*:\s*([^;}]+)`)

// cssVars maps every custom property declared in a region to its value.
func cssVars(region string) map[string]string {
	out := map[string]string{}
	for _, m := range cssVarDecl.FindAllStringSubmatch(region, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

// braceBlock returns the text between the first "{" at or after start and its
// matching "}", counting nesting: the light palette wraps a :root block, so a
// scan to the first "}" would stop one level too early. Callers must pass
// comment-stripped CSS. A brace inside a string value (`content: "}"`) would
// still fool it; this stylesheet has no content property, and a CSS parser
// would be a disproportionate dependency for one hand-written sheet.
func braceBlock(s string, start int) string {
	open := strings.Index(s[start:], "{")
	if open < 0 {
		return ""
	}
	open += start
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return s[open+1 : i]
			}
		}
	}
	return ""
}

// rootBlocksAtDepth returns the bodies of every ":root" rule sitting at the
// given brace depth in comment-stripped CSS. Depth is what separates a top-level
// palette (depth 0) from one nested in an at-rule (depth 1), so the caller can
// ask precisely for the one it means.
func rootBlocksAtDepth(css string, want int) []string {
	var out []string
	depth := 0
	for i := 0; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ':':
			if depth == want && strings.HasPrefix(css[i:], ":root") {
				if body := braceBlock(css, i); body != "" {
					out = append(out, body)
				}
			}
		}
	}
	return out
}

// TestStylesheetServesBothPalettes is T-054's regression guard. There is no
// browser here, so it asserts what can be checked mechanically and would
// plausibly break: that the served sheet carries a light palette behind
// prefers-color-scheme, that color-scheme is declared in step with it, and that
// neither palette has drifted a variable the other lacks.
func TestStylesheetServesBothPalettes(t *testing.T) {
	rec := get(t, newHandler(t, standardTree(t)), "/static/styles.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/styles.css = %d, want 200", rec.Code)
	}
	css := cssComment.ReplaceAllString(rec.Body.String(), " ")

	loc := lightMedia.FindStringIndex(css)
	if loc == nil {
		t.Fatal("stylesheet declares no prefers-color-scheme: light block — the dashboard is hardcoded to one palette")
	}
	light := braceBlock(css, loc[1])
	if light == "" {
		t.Fatal("the light palette's @media block is unterminated")
	}
	// The base palette is every *top-level* :root block — plural, and not merely
	// "the text before the light query": a :root added after it would otherwise
	// escape the parity check while still winning the cascade. Depth-0 only, so a
	// :root scoped to some other at-rule (@media print, say) is correctly ignored
	// rather than demanding a light-mode counterpart.
	base := strings.Join(rootBlocksAtDepth(css, 0), "\n")
	if base == "" {
		t.Fatal("stylesheet declares no top-level :root block")
	}
	// The light palette must target :root too: a block styling something else
	// would leave the variables at their dark values everywhere that matters.
	if !strings.Contains(light, ":root") {
		t.Error("the light @media block does not target :root, so it overrides nothing document-wide")
	}

	// color-scheme must track the palette exactly. An exact-value comparison is
	// the point: "dark light" would let the browser paint dark chrome under the
	// light palette, which is what decision 2 forbids.
	for _, tc := range []struct{ region, want, where string }{
		{base, "dark", ":root"},
		{light, "light", "the light @media block"},
	} {
		m := colorSchemeDecl.FindStringSubmatch(tc.region)
		switch {
		case m == nil:
			t.Errorf("%s declares no color-scheme — browser-painted chrome (scrollbars, the <details> marker) will not match the palette", tc.where)
		case strings.TrimSpace(m[1]) != tc.want:
			t.Errorf("%s declares color-scheme: %q, want exactly %q — a two-value list lets the browser paint the wrong scheme's chrome",
				tc.where, strings.TrimSpace(m[1]), tc.want)
		}
	}

	// Parity: --mono is a font stack, not a colour, and is deliberately declared
	// once. Every other variable must exist in both palettes.
	baseVars, lightVars := cssVars(base), cssVars(light)
	delete(baseVars, "--mono")
	for name := range baseVars {
		if _, ok := lightVars[name]; !ok {
			t.Errorf("%s is declared in :root but not in the light palette — light mode will inherit a dark value", name)
		}
	}
	for name := range lightVars {
		if name == "--mono" {
			t.Error("--mono is a font stack, not a colour; it should not be overridden per palette")
			continue
		}
		if _, ok := baseVars[name]; !ok {
			t.Errorf("%s is declared only in the light palette — it has no dark-mode value at all", name)
		}
	}

	// A light palette that merely repeats the dark values is not a light
	// palette. --bg is the page itself: if that matches, nothing was re-authored.
	if baseVars["--bg"] != "" && baseVars["--bg"] == lightVars["--bg"] {
		t.Errorf("both palettes set --bg to %s — the light block is a copy, not a light palette", baseVars["--bg"])
	}
}

func TestNotFoundAndMethodGuards(t *testing.T) {
	h := newHandler(t, standardTree(t))
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/t/T-999", http.StatusNotFound},    // well-formed but unknown
		{http.MethodGet, "/t/nonsense", http.StatusNotFound}, // not an id shape
		{http.MethodGet, "/nope", http.StatusNotFound},       // "GET /" is a catch-all pattern
		{http.MethodGet, "/favicon.ico", http.StatusNotFound},
		{http.MethodPost, "/", http.StatusMethodNotAllowed},
		{http.MethodPut, "/t/T-001", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/activity", http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
}

// --- health banner ----------------------------------------------------------

func TestHealthBannerReportsAuditErrors(t *testing.T) {
	root := newTree(t, fixture{dir: "1-to-do", id: "T-001", title: "bad grade", impact: "spicy",
		history: []string{"- 2026-07-20 — created (TO DO). source: test"}})
	body := get(t, newHandler(t, root), "/").Body.String()
	if !strings.Contains(body, "illegal impact value") {
		t.Errorf("health banner does not report the specific audit error\n%s", body)
	}
}

func TestHealthBannerIsCleanForACleanTree(t *testing.T) {
	body := get(t, newHandler(t, standardTree(t)), "/").Body.String()
	if !strings.Contains(body, "board audit clean") {
		t.Errorf("health banner is not clean for a clean fixture — the banner does not discriminate\n%s", body)
	}
	if strings.Contains(body, "BOARD.md is stale") {
		t.Error("fixture board was reported stale; newTree must render BOARD.md")
	}
}

// --- guarantees -------------------------------------------------------------

// TestServeNeverWrites is decision 1's proof: hit every route, then assert the
// tree is byte-for-byte what it was, with no file added or removed.
func TestServeNeverWrites(t *testing.T) {
	root := standardTree(t)
	h := newHandler(t, root)
	before := snapshot(t, root)

	for _, p := range []string{
		"/", "/activity", "/t/T-001", "/t/T-002", "/t/T-004", "/t/T-005", "/t/T-999",
		"/nope", "/fragments/board", "/fragments/activity", "/healthz",
		"/static/styles.css", "/static/htmx.min.js",
	} {
		get(t, h, p)
	}
	// Non-GET verbs too: the mux must reject them without a handler ever running.
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(m, "/", nil))
	}

	after := snapshot(t, root)
	if len(before) != len(after) {
		t.Fatalf("file count changed: %d → %d", len(before), len(after))
	}
	for path, sum := range before {
		got, ok := after[path]
		if !ok {
			t.Errorf("%s disappeared", path)
			continue
		}
		if got != sum {
			t.Errorf("%s was modified — serve must be read-only", path)
		}
	}
}

// snapshot maps every file under root to a hash of its contents.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return out
}

// TestEscapingIsTheTemplatesJob guards decision 4: the board's markdown-table
// sanitisation (pipes → ¦, the 120-rune cap) is not applied to HTML, so the
// escaping has to be real.
func TestEscapingIsTheTemplatesJob(t *testing.T) {
	long := strings.Repeat("x", 200)
	root := newTree(t, fixture{
		dir: "1-to-do", id: "T-001",
		title:   `<script>alert(1)</script> pipe | and ` + long,
		impact:  "high",
		history: []string{"- 2026-07-20 — created (TO DO). source: test"},
	})
	for _, path := range []string{"/", "/t/T-001"} {
		body := get(t, newHandler(t, root), path).Body.String()
		if strings.Contains(body, "<script>alert(1)</script>") {
			t.Errorf("%s emitted a raw <script> from a ticket title", path)
		}
		if !strings.Contains(body, "&lt;script&gt;") {
			t.Errorf("%s did not escape the title's markup", path)
		}
		if strings.Contains(body, "¦") {
			t.Errorf("%s applied the board's pipe substitution to HTML", path)
		}
		if !strings.Contains(body, long) {
			t.Errorf("%s truncated a long title; the 120-rune cap is a markdown-table rule only", path)
		}
	}
}

func TestMarkdownDoesNotRenderRawHTML(t *testing.T) {
	root := newTree(t, fixture{
		dir: "1-to-do", id: "T-001", title: "raw html body", impact: "low",
		body:    "<img src=x onerror=alert(1)>\n",
		history: []string{"- 2026-07-20 — created (TO DO). source: test"},
	})
	body := get(t, newHandler(t, root), "/t/T-001").Body.String()
	if strings.Contains(body, "onerror=alert(1)") && !strings.Contains(body, "&lt;img") {
		t.Error("goldmark rendered raw HTML from a ticket body (WithUnsafe must stay off)")
	}
}

func TestActivityCapReportsTruncation(t *testing.T) {
	var fixtures []fixture
	for i := 1; i <= 60; i++ {
		var hist []string
		for d := 1; d <= 5; d++ {
			hist = append(hist, fmt.Sprintf("- 2026-07-%02d — created (TO DO). source: test %d", d, d))
		}
		fixtures = append(fixtures, fixture{
			dir: "1-to-do", id: fmt.Sprintf("T-%03d", i), title: fmt.Sprintf("t %d", i),
			impact: "low", history: hist,
		})
	}
	root := newTree(t, fixtures...)
	tickets, _ := ticket.LoadAll(root)
	view := buildActivity(tickets)
	if view.Total != 300 {
		t.Errorf("Total = %d, want 300", view.Total)
	}
	if len(view.Events) != activityCap || !view.Truncated {
		t.Errorf("got %d events (truncated=%v), want %d and truncated", len(view.Events), view.Truncated, activityCap)
	}
	body := get(t, newHandler(t, root), "/activity").Body.String()
	if !strings.Contains(body, "most recent") {
		t.Error("activity page does not say it is truncated")
	}
}

// --- end to end -------------------------------------------------------------

// TestServeOnRealListener exercises the actual server: bind port 0 (never a fixed
// port), serve, request, then cancel and require a clean return.
func TestServeOnRealListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- Serve(ctx, ln, Options{Root: standardTree(t), Cfg: testCfg()}) }()

	base := "http://" + ln.Addr().String()
	var body string
	for i := 0; i < 50; i++ {
		resp, err := http.Get(base + "/healthz") //nolint:noctx // local test request
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			body = string(b)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if strings.TrimSpace(body) != "ok" {
		t.Fatalf("/healthz body = %q, want ok", body)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Serve returned %v, want nil after context cancellation", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Serve did not return after the context was cancelled")
	}
}

func TestHandlerRejectsNilConfig(t *testing.T) {
	if _, err := Handler(Options{Root: t.TempDir()}); err == nil {
		t.Error("Handler(nil cfg) = nil error, want a startup error")
	}
}

// --- helpers ----------------------------------------------------------------

// indexOrder reports whether the substrings appear in the given order.
func indexOrder(body string, subs ...string) bool {
	prev := -1
	for _, s := range subs {
		i := strings.Index(body, s)
		if i < 0 || i < prev {
			return false
		}
		prev = i
	}
	return true
}
