// Package scaffold implements `pickle scaffold docs`: an entirely optional,
// standalone command that lays down a minimal AsciiDoc docs skeleton, a
// best-effort `snowball init` shell-out, additive justfile recipes, and a
// standalone GitHub Action that attaches the built manual to a release.
//
// This is deliberately unrelated to brine (T-110): `pickle install` continues
// to scaffold the ticket flow only, and nothing this package writes is ever
// read back by `pickle doctor`/`pickle board audit` or any other pickle
// command — it is a one-shot scaffold, not an ongoing invariant pickle owns.
package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// projectNameToken is substituted, verbatim, for opts.ProjectName in every
// template file. Plain string replacement — not text/template — because the
// GitHub Actions template contains `${{ }}` expressions of its own, which
// would collide with Go's default template delimiters.
const projectNameToken = "__PROJECT_NAME__"

// templateFile pairs an embedded template asset with the path it is written
// to, relative to the target root.
type templateFile struct {
	Asset     string
	Installed string
}

// templateFiles is the fixed set of files `pickle scaffold docs` writes.
var templateFiles = []templateFile{
	{"scaffold/docs-template/attributes.adoc", "docs/attributes.adoc"},
	{"scaffold/docs-template/user-manual.adoc", "docs/user-manual.adoc"},
	{"scaffold/docs-template/user-manual/introduction.adoc", "docs/user-manual/introduction.adoc"},
	{"scaffold/docs-template/workflows/docs-release.yml", ".github/workflows/docs-release.yml"},
}

// Options configures a single `scaffold docs` run.
type Options struct {
	// ProjectName substitutes projectNameToken in every template file.
	// Defaults to filepath.Base(root) when empty (Run fills this in).
	ProjectName string
	// Force overwrites files that already exist. Without it, an existing
	// file is left untouched and reported in Skipped.
	Force bool
	// DryRun reports what would happen without writing anything.
	DryRun bool
}

// Result records what a run created, left in place, or wants the caller to
// know about, for the CLI summary — same shape as internal/install's Result.
type Result struct {
	Created []string
	Skipped []string
	// Notes carries longer, human-directed guidance (dry-run previews,
	// snowball-init follow-up instructions, a missing-snowball warning).
	Notes []string
}

func (r *Result) created(f string) { r.Created = append(r.Created, f) }
func (r *Result) skipped(f string) { r.Skipped = append(r.Skipped, f) }
func (r *Result) note(n string)    { r.Notes = append(r.Notes, n) }

// Docs performs a scaffold docs run into root using the embedded payload FS
// (the binary's "scaffold/docs-template" tree).
func Docs(payload fs.FS, root string, opts Options) (Result, error) {
	var res Result

	name := opts.ProjectName
	if name == "" {
		name = filepath.Base(root)
	}
	token := []byte(projectNameToken)
	replacement := []byte(name)

	for _, tf := range templateFiles {
		data, err := fs.ReadFile(payload, tf.Asset)
		if err != nil {
			return res, fmt.Errorf("read embedded %s: %w", tf.Asset, err)
		}
		data = bytes.ReplaceAll(data, token, replacement)

		dst := filepath.Join(root, filepath.FromSlash(tf.Installed))
		if _, err := os.Lstat(dst); err == nil {
			if !opts.Force {
				res.skipped(tf.Installed + " (exists — pass --force to overwrite)")
				continue
			}
			if opts.DryRun {
				res.note(tf.Installed + " (dry-run) would overwrite")
				continue
			}
		} else if opts.DryRun {
			res.note(tf.Installed + " (dry-run) would create")
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return res, err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return res, err
		}
		res.created(tf.Installed)
	}

	if err := scaffoldJustfile(root, &res, opts.DryRun); err != nil {
		return res, err
	}

	scaffoldSnowballConfig(root, &res, opts.DryRun)

	return res, nil
}

// justfileRecipe is one docs recipe scaffoldJustfile may append. Each body
// matches this repo's own `snowball` invocation for that recipe (T-048).
// pickle's own docs-check additionally runs a repo-local Go test that guards
// its manual's cross-references (T-067); that has no scaffolded equivalent, so
// the bodies here are no longer character-for-character identical to this
// repo's justfile.
type justfileRecipe struct {
	Name    string // e.g. "docs-check" — matched as a "<name>:" line prefix
	Comment string
	Body    string
}

var justfileRecipes = []justfileRecipe{
	{
		Name:    "docs-check",
		Comment: "# Validate the AsciiDoc manual via snowball (broken includes/xrefs fail the check)",
		Body:    "snowball check",
	},
	{
		Name:    "docs-build",
		Comment: "# Render the user manual to PDF + EPUB into dist/docs/ (never committed)",
		Body:    "snowball build -o dist/docs",
	},
}

// scaffoldJustfile appends docs-check/docs-build recipes to an existing
// justfile — additive only. It never creates a justfile (decision 6, T-110):
// a repo with no task runner does not get one invented for it.
func scaffoldJustfile(root string, res *Result, dryRun bool) error {
	path := filepath.Join(root, "justfile")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			res.skipped("justfile (does not exist — skipped; pickle never creates a task runner)")
			return nil
		}
		return err
	}

	var toAppend []justfileRecipe
	for _, rec := range justfileRecipes {
		if hasRecipe(string(data), rec.Name) {
			res.skipped(fmt.Sprintf("justfile: %s (already defined)", rec.Name))
			continue
		}
		toAppend = append(toAppend, rec)
	}
	if len(toAppend) == 0 {
		return nil
	}
	if dryRun {
		for _, rec := range toAppend {
			res.note(fmt.Sprintf("justfile: %s (dry-run) would append", rec.Name))
		}
		return nil
	}

	var b strings.Builder
	b.WriteString(string(data))
	if !strings.HasSuffix(string(data), "\n") {
		b.WriteString("\n")
	}
	for _, rec := range toAppend {
		fmt.Fprintf(&b, "\n%s\n%s:\n    %s\n", rec.Comment, rec.Name, rec.Body)
		res.created(fmt.Sprintf("justfile: %s recipe appended", rec.Name))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// hasRecipe reports whether justfile content already defines a recipe named
// name — matched as a line starting with "<name>:", the same shape `just`
// itself parses a recipe header as.
func hasRecipe(content, name string) bool {
	prefix := name + ":"
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// scaffoldSnowballConfig shells out to `snowball init` when the binary is on
// PATH (best-effort: never fatal — a missing or failing snowball is recorded
// as a Note, not a returned error) and always leaves follow-up guidance for
// pointing the generated config at the scaffolded book (decision 5, T-110).
func scaffoldSnowballConfig(root string, res *Result, dryRun bool) {
	const guidance = "snowball.yaml: point it at the scaffolded manual — " +
		"set `src: docs/user-manual.adoc` and `out: <project-name>-user-manual`."

	bin, err := exec.LookPath("snowball")
	if err != nil {
		res.note("snowball not found on PATH — install it (see https://github.com/codcod/snowball) " +
			"and run `snowball init` yourself. " + guidance)
		return
	}
	if dryRun {
		res.note("snowball found on PATH — (dry-run) would run `snowball init`. " + guidance)
		return
	}

	cmd := exec.Command(bin, "init")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		res.note(fmt.Sprintf("`snowball init` failed (non-fatal): %v\n%s", err, strings.TrimSpace(string(out))))
	} else {
		res.created("snowball.yaml (via `snowball init`)")
	}
	res.note(guidance)
}
