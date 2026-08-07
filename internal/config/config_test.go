package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
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
	if p.TicketPrefix != DefaultTicketPrefix || p.Prefix() != DefaultTicketPrefix {
		t.Errorf("ticket_prefix default = %q (Prefix()=%q), want %q", p.TicketPrefix, p.Prefix(), DefaultTicketPrefix)
	}
	if p.WIPInDevelopment != 1 || p.WIPInReview != 1 {
		t.Errorf("WIP defaults = %d/%d", p.WIPInDevelopment, p.WIPInReview)
	}
	if c.Root() != dir {
		t.Errorf("Root = %q, want %q", c.Root(), dir)
	}
}

// TestLoadDefaultsZeroWIP (T-043 item 3): the "zero wip" case in
// TestLoadErrors actually asserted -1, an error case renamed to "negative
// wip" alongside this addition. An *omitted or explicit* wip_in_review = 0
// must load successfully and default to 1 — applyDefaults treats the Go zero
// value as "unset", the same rule TestLoadValidAndDefaults already pins for a
// wholly-omitted field; this pins the explicit-zero shape too.
func TestLoadDefaultsZeroWIP(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, `payload_version = "1"
[[project]]
name = "a"
path = "."
wip_in_development = 0
wip_in_review = 0
`)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v (explicit wip_in_review = 0 must default, not error)", err)
	}
	p := c.Projects[0]
	if p.WIPInDevelopment != DefaultWIPInDevelopment || p.WIPInReview != DefaultWIPInReview {
		t.Errorf("WIP = %d/%d, want the defaults %d/%d", p.WIPInDevelopment, p.WIPInReview, DefaultWIPInDevelopment, DefaultWIPInReview)
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
		"negative wip": `payload_version = "1"
[[project]]
name = "a"
path = "."
wip_in_review = -1
`,
		"bad prefix shape (lowercase)": `payload_version = "1"
[[project]]
name = "a"
path = "."
ticket_prefix = "rick"
`,
		"bad prefix shape (too long)": `payload_version = "1"
[[project]]
name = "a"
path = "."
ticket_prefix = "TOOLONGXX"
`,
		"bad prefix shape (leading digit)": `payload_version = "1"
[[project]]
name = "a"
path = "."
ticket_prefix = "1X"
`,
		"duplicate explicit prefix": `payload_version = "1"
[[project]]
name = "a"
path = "."
ticket_prefix = "RICK"
[[project]]
name = "b"
path = "."
ticket_prefix = "RICK"
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

func TestTicketPrefixBackCompat(t *testing.T) {
	// Two children that both omit ticket_prefix both fall back to "T" and share
	// the one legacy global namespace — this must stay legal so pre-prefix
	// multi-child workspaces still load (T-058 D2/D4).
	body := `payload_version = "1"
[[project]]
name = "a"
path = "."
[[project]]
name = "b"
path = "."
`
	dir := t.TempDir()
	path := writeCfg(t, dir, body)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("two default-T children must be legal, got: %v", err)
	}
	for _, p := range c.Projects {
		if p.Prefix() != DefaultTicketPrefix {
			t.Errorf("project %q Prefix() = %q, want %q", p.Name, p.Prefix(), DefaultTicketPrefix)
		}
	}
	// A distinct explicit prefix is likewise fine and round-trips through Render.
	c.Projects[1].TicketPrefix = "SB"
	if err := c.Validate(); err != nil {
		t.Fatalf("distinct prefixes must be legal: %v", err)
	}
	if !strings.Contains(c.Render(), `ticket_prefix = "SB"`) {
		t.Errorf("Render did not emit the explicit prefix:\n%s", c.Render())
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

// TestRenderEscaping pins tomlQuote's round-trip table against the exact
// shapes T-069 measured %q getting wrong: control bytes with a TOML short
// escape, ones without (BEL, VT), a DEL byte, and ordinary multibyte text.
// Each must render as a legal TOML basic string that decodes back to the
// original Go string, byte for byte.
func TestRenderEscaping(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"control byte 0x01", "a\x01b"},
		{"tab", "a\tb"},
		{"newline", "a\nb"},
		{"multibyte text", "h\u00e9llo\u2192"},
		{"NEL u0085", "a\u0085b"},
		{"DEL 0x7f", "a\x7fb"},
		{"BEL 0x07 (no TOML equivalent; %q emitted \\a)", "a\x07b"},
		{"VT 0x0b (no TOML equivalent; %q emitted \\v)", "a\x0bb"},
		{"backspace", "a\bb"},
		{"form feed", "a\fb"},
		{"carriage return", "a\rb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			quoted := tomlQuote(c.value)
			var probe map[string]any
			if _, err := toml.Decode("v = "+quoted+"\n", &probe); err != nil {
				t.Fatalf("tomlQuote(%q) = %s, which does not parse as TOML: %v", c.value, quoted, err)
			}
			got, _ := probe["v"].(string)
			if got != c.value {
				t.Errorf("round-trip mismatch: got %q, want %q (rendered %s)", got, c.value, quoted)
			}
		})
	}
}

// TestSaveEscapingRoundTrip exercises the same shapes through the full
// AddProject -> Save -> Load path (a project name carrying a tab and a BEL
// byte), not just tomlQuote in isolation.
func TestSaveEscapingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	name := "a\tb\x07c" // tab + BEL: %q's \a rendering of the BEL byte used to be invalid TOML.
	if err := c.AddProject(Project{Name: name, Path: "sub"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if err := c.Save(""); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatalf("reload after escaping round-trip: %v", err)
	}
	if _, ok := c2.Project(name); !ok {
		t.Errorf("project name did not round-trip through Save/Load; got %+v", c2.Projects)
	}
}

// TestSaveFollowsSymlink is Save's counterpart to
// TestSetPayloadVersionInPlaceFollowsSymlink: since T-069, Save also goes
// through writePreservingMode, so it must not turn a symlinked pickle.toml
// into a regular file either.
func TestSaveFollowsSymlink(t *testing.T) {
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

	c, err := Load(link)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c.PayloadVersion = "7.7.7"
	if err := c.Save(""); err != nil {
		t.Fatalf("Save: %v", err)
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

// TestSaveUnwritableParentNamesTheRealFile is Save's counterpart to
// TestSetPayloadVersionInPlaceUnwritableParentNamesTheRealFile: it now shares
// writePreservingMode, so it must report the same actionable, real-path-first
// failure (T-026 D5) rather than os.WriteFile's bare errno.
func TestSaveUnwritableParentNamesTheRealFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) // let t.TempDir clean up

	err = c.Save("")
	if err == nil {
		t.Fatal("expected a failure writing into a read-only directory")
	}
	msg := err.Error()
	wantPath := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		wantPath = resolved
	}
	if !strings.HasPrefix(msg, wantPath+":") {
		t.Errorf("error does not lead with the real file %q: %v", wantPath, err)
	}
	if !strings.Contains(msg, "writable") {
		t.Errorf("error gives no actionable cause: %v", err)
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

// TestValidateRejectsInvalidUTF8AtTopLevel covers the same UTF-8 check for the
// two top-level string fields Render also quotes, alongside the per-project
// coverage in TestAddProjectRejectsInvalidUTF8.
func TestValidateRejectsInvalidUTF8AtTopLevel(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	base, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	c := *base
	c.PayloadVersion = "u\xffb"
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "payload_version") {
		t.Errorf("payload_version: Validate() = %v, want a payload_version UTF-8 error", err)
	}

	c = *base
	c.ReviewAddendum = "u\xffb"
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "review_addendum") {
		t.Errorf("review_addendum: Validate() = %v, want a review_addendum UTF-8 error", err)
	}
}

// TestAddProjectRejectsInvalidUTF8 pins the ticket's second reachability repro
// (`pk project add $'u\xffb' sub`): a name that is not valid UTF-8 used to be
// written to pickle.toml as-is and silently mutate on the next load (%q's
// \xff escaping reads back as a *different*, valid string). AddProject must
// now refuse it outright, and leave the registry exactly as it was — not with
// a half-appended project sitting behind the error.
func TestAddProjectRejectsInvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := len(c.Projects)
	badName := "u\xffb" // the ticket's exact repro: not valid UTF-8.
	if err := c.AddProject(Project{Name: badName, Path: "."}); err == nil {
		t.Fatal("expected AddProject to reject a name that is not valid UTF-8")
	} else if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("error does not mention UTF-8: %v", err)
	}
	if len(c.Projects) != before {
		t.Errorf("AddProject left a partial append behind: %d projects, want %d", len(c.Projects), before)
	}
	if _, ok := c.Project(badName); ok {
		t.Error("the invalid project is still registered after the rejected AddProject")
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
	// treeEqual, not reflect.DeepEqual: a legal `nan` value must not make this
	// independent check fail the same way it would make the writer's own
	// safety gate fail — NaN != NaN is not "something changed".
	if !treeEqual(inTree, outTree) {
		t.Errorf("values other than payload_version changed:\n%#v\n%#v", inTree, outTree)
	}
	if strings.HasPrefix(in, bom) != strings.HasPrefix(out, bom) {
		t.Errorf("byte-order mark was added or dropped:\n%q", out)
	}
	// Line endings are content too: a CRLF file must not come back with one
	// lone LF line where the key was rewritten. This is only asserted when the
	// input's own last line was itself newline-terminated: an input whose last
	// line never had a terminator at all (T-069 3(b)) establishes no CRLF style
	// for that line to preserve, and the fix's own contract is "stop refusing",
	// not "invent a line ending the file never had" — see
	// TestSetPayloadVersionCRLFUnterminatedLastLine for that shape pinned
	// directly instead.
	if crlfLines(in) && strings.HasSuffix(in, "\n") && !crlfLines(out) {
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

	// T-026: the scanner now tracks multi-line-string state and array bracket
	// depth, so these three shapes are read correctly regardless of position —
	// position (a scan that stops at the key) is no longer what decides it.
	{"quoted key", "\"payload_version\" = \"v1\"\n", true, ""},
	{"multi-line string with a table-looking line", "note = \"\"\"\n[warning]\nkeep\n\"\"\"\npayload_version = \"v1\"\n", true, ""},
	{"multi-line string containing the key", "note = \"\"\"\npayload_version = \"DO NOT TOUCH\"\n\"\"\"\npayload_version = \"v1\"\n", true, ""},
	// A quoted key elsewhere in the file that is *not* the real key must not
	// be mistaken for it, mirroring the bare-key "prefixed key" case above.
	{"quoted prefixed key", "\"payload_version_note\" = \"x\"\npayload_version = \"v1\"\n", true, ""},
	// A legal `nan` anywhere in the file used to make the safety gate
	// unpassable by construction (NaN != NaN under reflect.DeepEqual), even
	// though nothing about payload_version changed. treeEqual fixes this.
	{"nan elsewhere in the file", "threshold = nan\npayload_version = \"v1\"\n", true, ""},
	// The insert path inside a multi-line array used to be misread as a table
	// header on the array's continuation line, wedging the file. The key
	// must land after the array, at the top level, not inside it.
	{"multi-line array, key absent", "matrix = [\n[1, 2],\n]\n", true, ""},
	// A '[' inside a single-line string must not be counted as a bracket or
	// mistaken for a table header.
	{"table-looking text inside a single-line string", "note = \"[not a header]\"\npayload_version = \"v1\"\n", true, ""},
	// A literal ('''-delimited) multi-line string has no escapes at all, unlike
	// its basic ("""-delimited) counterpart above — this exercises that this
	// separate code path also keeps scanning across lines rather than
	// misreading the table-looking line inside it as a header.
	{"table-looking line inside a literal multi-line string", "note = '''\n[not a header]\n'''\npayload_version = \"v1\"\n", true, ""},

	// Shapes the line scanner cannot read correctly. It must refuse, not guess.
	{"multi-line value on the key itself", "payload_version = \"\"\"\nv1\n\"\"\"\n", false, "multi-line string; set it by hand"},
	{"array value with a space", "payload_version = [\"a\", \"b\"]\n", false, "unparseable"},
	{"already duplicated key", "payload_version = \"a\"\npayload_version = \"b\"\n", false, "does not parse"},

	// T-069 3(a): an escaped `\"""` inside a multi-line basic string used to be
	// misread as the closing delimiter one byte early, so the `[x]` line after
	// it was taken for a table header and payload_version got inserted inside
	// the string instead of rewritten at top level.
	{"escaped triple-quote inside a multi-line string", `note = """
a \""" b
[x]
"""
payload_version = "v1"
`, true, ""},
	// T-069 3(b): a CRLF file whose last line is never newline-terminated used
	// to have a bare trailing \r appended to the inserted key (nothing follows
	// it to be consistent with), which the parse-back gate then refused as
	// unparseable control-character content. See also
	// TestSetPayloadVersionCRLFUnterminatedLastLine, which pins the exact byte
	// shape of the original defect directly.
	{"crlf, unterminated last line", "\r\n#", true, ""},
	{"crlf, unterminated last line with a real key", "a = 1\r\n# tail", true, ""},
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
		rt := payloadVersionKey + " = " + tomlQuote(version) + "\n"
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

// TestSetPayloadVersionInPlaceUnwritableParentNamesTheRealFile pins T-026's
// D5: a failure inside writePreservingMode must be reported against the
// file the user owns, with an actionable cause — never a bare errno naming
// the throwaway temp file it tried to create beside it.
func TestSetPayloadVersionInPlaceUnwritableParentNamesTheRealFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) // let t.TempDir clean up

	err := SetPayloadVersionInPlace(path, "9.9.9")
	if err == nil {
		t.Fatal("expected a failure writing into a read-only directory")
	}
	msg := err.Error()
	// writePreservingMode resolves symlinks in its own path before reporting
	// (e.g. macOS's /var -> /private/var), so compare against that same
	// resolution rather than the literal t.TempDir() path.
	wantPath := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		wantPath = resolved
	}
	// The real path the user owns must be named, and named first — not
	// merely present somewhere inside the wrapped syscall detail, which is
	// exactly the "raw errno naming a file the user never created" failure
	// mode this task fixes.
	if !strings.HasPrefix(msg, wantPath+":") {
		t.Errorf("error does not lead with the real file %q: %v", wantPath, err)
	}
	if !strings.Contains(msg, "writable") {
		t.Errorf("error gives no actionable cause: %v", err)
	}
}

// hardlinkCount returns fi's link count, or 0 if the platform's os.FileInfo
// does not expose one (e.g. Windows), in which case the caller should skip
// rather than assert anything.
func hardlinkCount(t *testing.T, fi os.FileInfo) uint64 {
	t.Helper()
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return uint64(st.Nlink)
}

// TestWritePreservingModeSeversHardlink pins one of T-069 D3's documented and
// declined behaviours instead of leaving it to be re-found: os.Rename cannot
// preserve a hardlink's identity, so a second name pointing at the same inode
// is left behind at the old contents once the first is atomically replaced.
func TestWritePreservingModeSeversHardlink(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	other := filepath.Join(dir, "other-name.toml")
	if err := os.Link(path, other); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}

	if err := writePreservingMode(path, []byte("updated content\n")); err != nil {
		t.Fatalf("writePreservingMode: %v", err)
	}

	fi, err := os.Stat(other)
	if err != nil {
		t.Fatal(err)
	}
	if nlink := hardlinkCount(t, fi); nlink == 0 {
		t.Skip("platform does not report a usable link count")
	} else if nlink != 1 {
		t.Errorf("other-name.toml nlink = %d, want 1 (severed from the renamed-over path)", nlink)
	}
	got, err := os.ReadFile(other)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "updated content\n" {
		t.Error("the second hardlinked name picked up the update; writePreservingMode is documented to sever hardlinks, not share them")
	}
}

// TestWritePreservingModeRewritesReadOnlyFile pins the other declined
// behaviour: create-temp-then-rename only needs a writable *directory*, so a
// read-only (0444) file is rewritten anyway — unlike the os.WriteFile it
// replaced, which would have failed on the file's own permission bit. This
// documents the decline (a successful rewrite), not a regression.
func TestWritePreservingModeRewritesReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}

	if err := writePreservingMode(path, []byte("updated\n")); err != nil {
		t.Fatalf("writePreservingMode: %v (expected to succeed even though the file is read-only)", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated\n" {
		t.Errorf("content = %q, want %q", got, "updated\n")
	}
}

// TestWritePreservingModeCreateRespectsUmask pins the T-069 rework fix
// (finding F1): when the target does not exist yet, writePreservingMode must
// not hard-code a mode via Chmod (which ignores the umask) the way it does
// for an existing file's rename-over path — it must behave like plain
// os.WriteFile(path, data, 0o644) and let the umask narrow that, exactly as
// pickle install's other generated files do.
func TestWritePreservingModeCreateRespectsUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	dir := t.TempDir()
	path := filepath.Join(dir, "new.toml")
	if err := writePreservingMode(path, []byte("x\n")); err != nil {
		t.Fatalf("writePreservingMode: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 600 (umask 077 applied to the requested 0644)", got)
	}
}

// TestWritePreservingModeRewriteIgnoresUmask is
// TestWritePreservingModeCreateRespectsUmask's counterpart for the *existing*
// path: once a file is there, its own mode is preserved regardless of the
// umask in effect at rewrite time — the umask only ever governs creation.
func TestWritePreservingModeRewriteIgnoresUmask(t *testing.T) {
	dir := t.TempDir()
	path := writeCfg(t, dir, oneProject)
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	if err := writePreservingMode(path, []byte("updated\n")); err != nil {
		t.Fatalf("writePreservingMode: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o640 {
		t.Errorf("file mode = %o, want 640 (preserved despite umask 077 at rewrite time)", got)
	}
}

// TestSaveCreatesNewConfigRespectingUmask is Save's own regression test for
// F1, exercised through the caller shape install.go's writeConfig uses
// (Save on a path that does not exist yet), not just writePreservingMode in
// isolation.
func TestSaveCreatesNewConfigRespectingUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Config{}
	if err := c.AddProject(Project{Name: "demo", Path: "sub"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	path := filepath.Join(dir, FileName)
	if err := c.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("new pickle.toml mode = %o, want 600 (umask 077 applied), matching every other file pickle install creates", got)
	}
}

// TestSetPayloadVersionCRLFUnterminatedLastLine pins T-069 3(b): a CRLF file
// whose last line was never newline-terminated used to be misjudged as fully
// CRLF by usesCRLF, so insertPayloadVersion appended a trailing \r to what
// became the file's own last line — a bare \r with no \n after it, which the
// parse-back gate then refused as unparseable. It must now succeed instead.
//
// This is deliberately not folded into payloadVersionFixtures /
// checkPayloadVersionInvariant: that shared check also insists a CRLF input
// comes back fully CRLF, which this pathological input cannot promise — its
// last line was never terminated at all, so there is nothing already-CRLF for
// the newly-terminated line to match. The ticket's fix is "stop refusing", not
// "invent a line ending the file never had".
func TestSetPayloadVersionCRLFUnterminatedLastLine(t *testing.T) {
	cases := []string{
		"\r\n#",
		"a = 1\r\n# tail",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			out, err := setPayloadVersion(in, "9.9.9")
			if err != nil {
				t.Fatalf("setPayloadVersion(%q): %v", in, err)
			}
			tree, err := decodeTree(out)
			if err != nil {
				t.Fatalf("result does not parse: %v\n%q", err, out)
			}
			if got, _ := tree[payloadVersionKey].(string); got != "9.9.9" {
				t.Errorf("payload_version = %q, want 9.9.9\n%q", got, out)
			}
			// No bare, unpaired \r anywhere in the output — the exact defect this
			// fix removes.
			for i := 0; i < len(out); i++ {
				if out[i] == '\r' && (i+1 >= len(out) || out[i+1] != '\n') {
					t.Errorf("unpaired \\r at byte %d:\n%q", i, out)
				}
			}
		})
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
