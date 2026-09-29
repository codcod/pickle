package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
)

func whereCfg() *config.Config {
	return &config.Config{Layout: config.LayoutUmbrella, Projects: []config.Project{
		{Name: "porth", Path: "porth", TicketPrefix: "POR", WIPInDevelopment: 1, WIPInReview: 1},
	}}
}

func decodeWhere(t *testing.T, rec *httptest.ResponseRecorder) []whereMatch {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var body struct{ Matches []whereMatch }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Matches
}

func TestWhereSingleRoot(t *testing.T) {
	root := newTree(t)
	h, err := Handler(Options{Root: root, Cfg: whereCfg(), Local: true})
	if err != nil {
		t.Fatal(err)
	}
	for key, matched := range map[string]string{"por": "prefix", "PORTH": "child", strings.ToUpper(filepath.Base(root)): "slug"} {
		m := decodeWhere(t, get(t, h, "/where/"+key))
		want := whereMatch{
			Root: root, Slug: filepath.Base(root), Child: "porth", ChildPath: filepath.Join(root, "porth"),
			Prefix: "POR", Matched: matched, Layout: config.LayoutUmbrella,
		}
		if len(m) != 1 || m[0] != want {
			t.Errorf("/where/%s = %+v, want [%+v]", key, m, want)
		}
		if !filepath.IsAbs(m[0].ChildPath) {
			t.Errorf("child_path %q not absolute", m[0].ChildPath)
		}
	}
}

func TestWhereMultiRoot(t *testing.T) {
	a, b := newTree(t), newTree(t)
	h, err := MultiHandler([]NamedRoot{
		{Slug: "a", Options: Options{Root: a, Cfg: testCfg()}},
		{Slug: "b", Options: Options{Root: b, Cfg: testCfg()}},
		{Slug: "bare", Options: Options{Root: a, Cfg: &config.Config{Layout: config.LayoutUmbrella}}},
	}, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := decodeWhere(t, get(t, h, "/where/t"))
	if len(m) != 2 || m[0].Slug != "a" || m[1].Slug != "b" || m[0].Matched != "prefix" {
		t.Errorf("/where/t = %+v, want a then b by prefix", m)
	}
	// A served root with no registered child still answers to its slug.
	m = decodeWhere(t, get(t, h, "/where/bare"))
	if len(m) != 1 || m[0].Child != "" || m[0].ChildPath != "" || m[0].Prefix != "" || m[0].Matched != "slug" {
		t.Errorf("/where/bare = %+v, want one child-less slug match", m)
	}
	if rec := get(t, h, "/p/a/where/t"); rec.Code != http.StatusNotFound {
		t.Errorf("/p/a/where/t = %d, want 404", rec.Code)
	}
}

func TestWhereMissNotice(t *testing.T) {
	var log bytes.Buffer
	h, err := Handler(Options{Root: newTree(t), Cfg: whereCfg(), Local: true, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"/where/messgr?from=porth", "/where/messgr?from=porth", "/where/messgr?from=other",
		"/where/x?from=a%1Bb", "/where/y?from=%1B%5B31m", "/where/z",
	} {
		if rec := get(t, h, p); rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", p, rec.Code)
		}
	}
	want := "pickle serve: porth wants to learn about messgr (not served here)\n" +
		"pickle serve: other wants to learn about messgr (not served here)\n" +
		"pickle serve: someone wants to learn about x (not served here)\n" +
		"pickle serve: someone wants to learn about y (not served here)\n" +
		"pickle serve: someone wants to learn about z (not served here)\n"
	if log.String() != want {
		t.Errorf("notices =\n%s\nwant\n%s", log.String(), want)
	}
}

func TestWhereRejects(t *testing.T) {
	var log bytes.Buffer
	root := newTree(t)
	h, err := Handler(Options{Root: root, Cfg: whereCfg(), Local: true, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	// Escaped forms: a literal "/where/.." or "/where/a/b" never reaches {key}.
	for _, k := range []string{"%2E%2E", strings.Repeat("a", 65), "a%2Fb", "-x"} {
		if rec := get(t, h, "/where/"+k); rec.Code != http.StatusBadRequest {
			t.Errorf("/where/%s = %d, want 400", k, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/where/por", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /where/por = %d, want 405", rec.Code)
	}

	// Not bound to loopback: nothing is revealed, not even a notice.
	remote, err := Handler(Options{Root: root, Cfg: whereCfg(), Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/where/por", "/where/nope"} {
		if rec := get(t, remote, p); rec.Code != http.StatusNotFound {
			t.Errorf("non-local %s = %d, want 404", p, rec.Code)
		}
	}
	if log.Len() != 0 {
		t.Errorf("notices = %q, want none", log.String())
	}
}
