package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// writeCfg writes a pickle.toml into dir and returns its path.
func writeCfg(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	return path
}

const oneProject = `payload_version = "1.2.3"
[commit]
overarching_auto = true
child_publish_gated = true
[[project]]
name = "pickle"
path = "."
build = "just build"
`

func TestLoadValidAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.PayloadVersion != "1.2.3" {
		t.Errorf("payload_version = %q", c.PayloadVersion)
	}
	if len(c.Projects) != 1 || c.Projects[0].Name != "pickle" {
		t.Fatalf("projects = %+v", c.Projects)
	}
	p := c.Projects[0]
	if p.BranchPrefix != DefaultBranchPrefix {
		t.Errorf("branch_prefix default = %q", p.BranchPrefix)
	}
	if p.WIPInDevelopment != 1 || p.WIPInReview != 1 {
		t.Errorf("WIP defaults = %d/%d", p.WIPInDevelopment, p.WIPInReview)
	}
	if c.Root() != dir {
		t.Errorf("Root = %q, want %q", c.Root(), dir)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]string{
		"no project": `payload_version = "1"
[commit]
`,
		"empty name": `payload_version = "1"
[[project]]
name = ""
path = "."
`,
		"empty path": `payload_version = "1"
[[project]]
name = "a"
path = ""
`,
		"bad path": `payload_version = "1"
[[project]]
name = "a"
path = "does-not-exist"
`,
		"duplicate name": `payload_version = "1"
[[project]]
name = "dup"
path = "."
[[project]]
name = "dup"
path = "."
`,
		"zero wip": `payload_version = "1"
[[project]]
name = "a"
path = "."
wip_in_review = -1
`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeCfg(t, dir, body)
			if _, err := Load(path); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.Save(""); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c2.Render() != c.Render() {
		t.Errorf("round-trip not stable:\n--- first ---\n%s\n--- second ---\n%s", c.Render(), c2.Render())
	}
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	writeCfg(t, dir, oneProject)
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(nested)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != filepath.Join(dir, FileName) {
		t.Errorf("Find = %q", got)
	}
	if _, err := Find(t.TempDir()); err == nil {
		t.Error("expected Find to fail with no pickle.toml")
	}
}

func TestAddRemove(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.AddProject(Project{Name: "pickle", Path: "."}); err == nil {
		t.Error("expected duplicate-name error")
	}
	if err := c.AddProject(Project{Name: "web", Path: "web"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if p, ok := c.Project("web"); !ok || p.WIPInDevelopment != 1 || p.BranchPrefix != DefaultBranchPrefix {
		t.Errorf("added project missing defaults: %+v ok=%v", p, ok)
	}
	if err := c.RemoveProject("nope"); err == nil {
		t.Error("expected remove-unknown error")
	}
	if err := c.RemoveProject("web"); err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}
	if _, ok := c.Project("web"); ok {
		t.Error("web still present after remove")
	}
}

// TestLoadRepoConfig loads this repository's own pickle.toml (../../ from here).
func TestLoadRepoConfig(t *testing.T) {
	repoCfg, err := filepath.Abs(filepath.Join("..", "..", FileName))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(repoCfg); err != nil {
		t.Skipf("repo pickle.toml not found: %v", err)
	}
	c, err := Load(repoCfg)
	if err != nil {
		t.Fatalf("Load repo config: %v", err)
	}
	if p, ok := c.Project("pickle"); !ok || p.Path != "." {
		t.Errorf("expected sole child 'pickle' at '.', got %+v ok=%v", c.Projects, ok)
	}
}

// commentedCfg exercises everything SetPayloadVersionInPlace must not disturb:
// leading comments, blank lines, an inline comment on the target line, a
// commented-out payload_version, and a payload_version *inside* a table.
const commentedCfg = `# pickle configuration.
# Second comment line.

# payload_version = "decoy-commented-out"
payload_version = "1.2.3"   # inline comment

[commit]
overarching_auto = true
child_publish_gated = true

# --- children ---
[[project]]
name = "pickle"
path = "."
payload_version = "decoy-in-table"
build = "just build"
`

func TestSetPayloadVersionInPlacePreservesEverythingElse(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, commentedCfg)

	if err := SetPayloadVersionInPlace(path, "9.9.9"); err != nil {
		t.Fatalf("SetPayloadVersionInPlace: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Byte-identical except the single real payload_version line.
	before := strings.Split(commentedCfg, "\n")
	after := strings.Split(string(got), "\n")
	if len(before) != len(after) {
		t.Fatalf("line count changed: %d -> %d\n%s", len(before), len(after), got)
	}
	var changed []int
	for i := range before {
		if before[i] != after[i] {
			changed = append(changed, i)
		}
	}
	if len(changed) != 1 {
		t.Fatalf("expected exactly 1 changed line, got %v\n%s", changed, got)
	}
	if want := `payload_version = "9.9.9"   # inline comment`; after[changed[0]] != want {
		t.Errorf("rewritten line = %q, want %q", after[changed[0]], want)
	}

	// The decoys must be untouched, and the file must still parse.
	if !strings.Contains(string(got), `# payload_version = "decoy-commented-out"`) {
		t.Error("commented-out payload_version was rewritten")
	}
	if !strings.Contains(string(got), `payload_version = "decoy-in-table"`) {
		t.Error("payload_version inside [[project]] was rewritten")
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load after in-place edit: %v", err)
	}
	if c.PayloadVersion != "9.9.9" {
		t.Errorf("payload_version = %q, want 9.9.9", c.PayloadVersion)
	}
}

func TestSetPayloadVersionInPlaceInsertsWhenAbsent(t *testing.T) {
	// No top-level key, but one inside a table: the table key must be ignored
	// rather than hijacked, and the real key inserted above [commit].
	const noVersion = `# a comment worth keeping

[commit]
overarching_auto = true
child_publish_gated = true

[[project]]
name = "pickle"
path = "."
payload_version = "decoy-in-table"
`
	dir := t.TempDir()
	path := writeCfg(t, dir, noVersion)

	if err := SetPayloadVersionInPlace(path, "4.5.6"); err != nil {
		t.Fatalf("SetPayloadVersionInPlace: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Inserted, not re-rendered: every original line survives in order.
	for _, line := range strings.Split(strings.TrimSuffix(noVersion, "\n"), "\n") {
		if !strings.Contains(string(got), line) {
			t.Errorf("original line %q lost", line)
		}
	}
	if !strings.Contains(string(got), `payload_version = "4.5.6"`) {
		t.Fatalf("payload_version not inserted:\n%s", got)
	}
	if !strings.Contains(string(got), `payload_version = "decoy-in-table"`) {
		t.Errorf("the [[project]] payload_version was hijacked instead of inserting:\n%s", got)
	}
	// It must land at top level, above the first table header.
	if i, j := strings.Index(string(got), "payload_version"), strings.Index(string(got), "[commit]"); i > j {
		t.Errorf("payload_version inserted below [commit] (%d > %d):\n%s", i, j, got)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load after insert: %v", err)
	}
	if c.PayloadVersion != "4.5.6" {
		t.Errorf("payload_version = %q, want 4.5.6", c.PayloadVersion)
	}
}

func TestSetPayloadVersionInPlaceEscapesAndKeepsMode(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	// 0o640 deliberately: 0o600 is os.CreateTemp's own default, so asserting it
	// would pass even if the mode were never preserved at all.
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}

	weird := `v1 "quoted" \ backslash`
	if err := SetPayloadVersionInPlace(path, weird); err != nil {
		t.Fatalf("SetPayloadVersionInPlace: %v", err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load after escaping: %v", err)
	}
	if c.PayloadVersion != weird {
		t.Errorf("payload_version = %q, want %q", c.PayloadVersion, weird)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o640 {
		t.Errorf("file mode = %o, want 640", got)
	}
}

func TestSetPayloadVersionInPlaceKeyPrefixIsNotAMatch(t *testing.T) {
	const decoy = `payload_version_note = "not the key"
payload_version = "1.0.0"

[commit]
overarching_auto = true
child_publish_gated = true

[[project]]
name = "pickle"
path = "."
`
	dir := t.TempDir()
	path := writeCfg(t, dir, decoy)
	if err := SetPayloadVersionInPlace(path, "2.0.0"); err != nil {
		t.Fatalf("SetPayloadVersionInPlace: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `payload_version_note = "not the key"`) {
		t.Errorf("a key merely prefixed with payload_version was rewritten:\n%s", got)
	}
	if !strings.Contains(string(got), `payload_version = "2.0.0"`) {
		t.Errorf("real key not updated:\n%s", got)
	}
}

// --- The in-place writer's invariant -----------------------------------------
//
// One rule covers every input, so the tests do not have to anticipate the exact
// shape of a file the line scanner might mis-read:
//
//	Either the call refuses and changes nothing, or the result parses, carries
//	the new version, decodes to the same values in every other respect, and is
//	byte-identical line-for-line apart from the one line rewritten or inserted.
//
// checkPayloadVersionInvariant asserts that for a single input. It returns
// whether the call succeeded so table cases can also pin down which outcome they
// expect — a fixture flipping from "rewritten cleanly" to "refused" is a
// regression even though both satisfy the invariant.
func checkPayloadVersionInvariant(t *testing.T, in, version string) bool {
	t.Helper()

	out, err := setPayloadVersion(in, version)
	if err != nil {
		if out != "" {
			t.Errorf("refused but still returned text: %q", out)
		}
		return false
	}

	inTree, inErr := decodeTree(strings.TrimPrefix(in, bom))
	outTree, outErr := decodeTree(strings.TrimPrefix(out, bom))
	if inErr != nil {
		t.Errorf("accepted an input that does not parse (%v)", inErr)
		return true
	}
	if outErr != nil {
		t.Errorf("produced unparseable output (%v):\n%s", outErr, out)
		return true
	}
	if got, _ := outTree[payloadVersionKey].(string); got != version {
		t.Errorf("payload_version = %q, want %q:\n%s", got, version, out)
	}
	delete(inTree, payloadVersionKey)
	delete(outTree, payloadVersionKey)
	if !reflect.DeepEqual(inTree, outTree) {
		t.Errorf("values other than payload_version changed:\n%#v\n%#v", inTree, outTree)
	}
	if strings.HasPrefix(in, bom) != strings.HasPrefix(out, bom) {
		t.Errorf("byte-order mark was added or dropped:\n%q", out)
	}
	// Line endings are content too: a CRLF file must not come back with one
	// lone LF line where the key was rewritten.
	if crlfLines(in) && !crlfLines(out) {
		t.Errorf("CRLF line endings were not preserved:\n%q", out)
	}

	// Every other line must be untouched: either one line was rewritten, or one
	// was inserted.
	before := strings.Split(strings.TrimPrefix(in, bom), "\n")
	after := strings.Split(strings.TrimPrefix(out, bom), "\n")
	switch len(after) - len(before) {
	case 0:
		var changed []int
		for i := range before {
			if before[i] != after[i] {
				changed = append(changed, i)
			}
		}
		if len(changed) > 1 {
			t.Errorf("%d lines changed, want at most 1 (%v):\n%s", len(changed), changed, out)
		}
		for _, i := range changed {
			if !strings.Contains(after[i], payloadVersionKey) {
				t.Errorf("rewrote an unrelated line %d: %q", i, after[i])
			}
		}
	case 1:
		found := false
		for i := range after {
			rest := append(append([]string{}, after[:i]...), after[i+1:]...)
			if reflect.DeepEqual(rest, before) {
				found = true
				if !strings.Contains(after[i], payloadVersionKey) {
					t.Errorf("inserted an unrelated line %d: %q", i, after[i])
				}
				break
			}
		}
		if !found {
			t.Errorf("output is not the input plus one line:\n--- in\n%s\n--- out\n%s", in, out)
		}
	default:
		t.Errorf("line count %d -> %d, want +0 or +1:\n%s", len(before), len(after), out)
	}
	return true
}

// payloadVersionFixtures are the adversarial shapes the line scanner has to
// survive. They double as the fuzz seed corpus.
var payloadVersionFixtures = []struct {
	name string
	in   string
	// ok is whether the writer is expected to make the edit; false means it must
	// refuse rather than guess.
	ok bool
	// wantErr is a substring of the refusal message, so that each refusal is
	// pinned to its own reason rather than to "some error happened".
	wantErr string
}{
	{"canonical", oneProject, true, ""},
	{"empty", "", true, ""},
	{"comments and blank lines", commentedCfg, true, ""},
	{"bom before a comment", bom + "# hi\npayload_version = \"1\"\n", true, ""},
	{"bom on the key's own line", bom + "payload_version = \"1\"\n[commit]\n", true, ""},
	{"crlf", "payload_version = \"1\"\r\n[commit]\r\noverarching_auto = true\r\n", true, ""},
	// The insert path on a CRLF file: the `crlf` fixture above already has the
	// key, so it only ever exercises the replace path. The fuzzer found the
	// inserted line going out with a lone LF; this pins it as a named case.
	{"crlf, key absent", "# note\r\n[commit]\r\noverarching_auto = true\r\n", true, ""},
	{"no trailing newline", "payload_version = \"1\"", true, ""},
	{"literal string value", "payload_version = 'v1'\n", true, ""},
	{"no spaces around =", "payload_version=\"v1\"\n", true, ""},
	{"aligned columns", "payload_version  = \"v1\"\nreview_addendum = \"x\"\n", true, ""},
	{"inline comment", "payload_version = \"v1\" # keep me\n", true, ""},
	{"tab indented", "\tpayload_version = \"v1\"\n", true, ""},
	{"absent, table below", "# note\n\n[commit]\noverarching_auto = true\n", true, ""},
	{"absent entirely", "# just a comment\n", true, ""},
	{"key only in a table", "[[project]]\nname = \"p\"\npayload_version = \"decoy\"\n", true, ""},
	{"prefixed key", "payload_version_note = \"x\"\npayload_version = \"v1\"\n", true, ""},
	// A multi-line string above the key is read correctly as long as none of its
	// lines looks like a table header or like the key: this is the accepted case
	// the refusals below are carved out of, and the docs say so.
	{"benign multi-line string above the key", "note = \"\"\"\nhello\nworld\n\"\"\"\npayload_version = \"v1\"\n", true, ""},
	// The mirror of the last two refusals: the scan stops at the key, so a
	// multi-line string *below* it is never read and cannot make the writer
	// refuse. What decides the refusal is position, not the shape — the README
	// says so, so both directions are pinned here.
	{"table-looking multi-line string below the key", "payload_version = \"v1\"\nnote = \"\"\"\n[warning]\nkeep\n\"\"\"\n", true, ""},
	{"multi-line string containing the key, below the key", "payload_version = \"v1\"\nnote = \"\"\"\npayload_version = \"DO NOT TOUCH\"\n\"\"\"\n", true, ""},

	// Shapes the line scanner cannot read correctly. It must refuse, not guess.
	{"quoted key", "\"payload_version\" = \"v1\"\n", false, "unparseable"},
	{"multi-line string with a table-looking line", "note = \"\"\"\n[warning]\nkeep\n\"\"\"\npayload_version = \"v1\"\n", false, "could not set payload_version"},
	{"multi-line string containing the key", "note = \"\"\"\npayload_version = \"DO NOT TOUCH\"\n\"\"\"\npayload_version = \"v1\"\n", false, "could not set payload_version"},
	{"multi-line value on the key itself", "payload_version = \"\"\"\nv1\n\"\"\"\n", false, "multi-line string; set it by hand"},
	{"array value with a space", "payload_version = [\"a\", \"b\"]\n", false, "unparseable"},
	{"already duplicated key", "payload_version = \"a\"\npayload_version = \"b\"\n", false, "does not parse"},
}

func TestSetPayloadVersionInvariant(t *testing.T) {
	for _, f := range payloadVersionFixtures {
		t.Run(f.name, func(t *testing.T) {
			if got := checkPayloadVersionInvariant(t, f.in, "9.9.9"); got != f.ok {
				verb := map[bool]string{true: "made the edit", false: "refused"}
				t.Fatalf("writer %s, want %s", verb[got], verb[f.ok])
			}
			if f.ok {
				return
			}
			_, err := setPayloadVersion(f.in, "9.9.9")
			if err == nil {
				t.Fatal("expected a refusal")
			}
			if !strings.Contains(err.Error(), f.wantErr) {
				t.Errorf("refusal = %q, want it to mention %q", err, f.wantErr)
			}
		})
	}
}

// TestSetPayloadVersionIsIdempotent pins the no-op path: stamping the version a
// file already carries must not perturb a single byte.
func TestSetPayloadVersionIsIdempotent(t *testing.T) {
	for _, f := range payloadVersionFixtures {
		if !f.ok {
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			once, err := setPayloadVersion(f.in, "9.9.9")
			if err != nil {
				t.Fatalf("first pass: %v", err)
			}
			twice, err := setPayloadVersion(once, "9.9.9")
			if err != nil {
				t.Fatalf("second pass: %v", err)
			}
			if once != twice {
				t.Errorf("second pass changed the file:\n--- once\n%s\n--- twice\n%s", once, twice)
			}
		})
	}
}

func FuzzSetPayloadVersion(f *testing.F) {
	for _, fx := range payloadVersionFixtures {
		f.Add(fx.in, "9.9.9")
	}
	f.Fuzz(func(t *testing.T, in, version string) {
		// The invariant only binds for inputs that parse to begin with; a file
		// pickle could never have loaded is not this function's problem.
		if _, err := decodeTree(strings.TrimPrefix(in, bom)); err != nil {
			return
		}
		// A version string that cannot survive a TOML round-trip (control
		// characters, invalid UTF-8) is out of scope in the same way.
		var probe map[string]any
		rt := payloadVersionKey + " = " + fmt.Sprintf("%q", version) + "\n"
		if _, err := toml.Decode(rt, &probe); err != nil {
			return
		}
		if got, _ := probe[payloadVersionKey].(string); got != version {
			return
		}
		checkPayloadVersionInvariant(t, in, version)
	})
}

// TestSetPayloadVersionInPlaceFollowsSymlink guards the atomic write: renaming
// onto a symlinked pickle.toml would turn it into a regular file and leave the
// real one stale.
func TestSetPayloadVersionInPlaceFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := writeCfg(t, realDir, oneProject)
	link := filepath.Join(dir, FileName)
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := SetPayloadVersionInPlace(link, "7.7.7"); err != nil {
		t.Fatalf("SetPayloadVersionInPlace: %v", err)
	}

	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced by a regular file")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `payload_version = "7.7.7"`) {
		t.Errorf("the symlink target kept the old version:\n%s", got)
	}
}

// crlfLines reports whether every line that has a terminator uses CRLF.
func crlfLines(text string) bool {
	lines := strings.Split(text, "\n")
	for _, l := range lines[:max(len(lines)-1, 0)] {
		if !strings.HasSuffix(l, "\r") {
			return false
		}
	}
	return len(lines) > 1
}
