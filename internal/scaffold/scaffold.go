// Package scaffold implements the `pickle scaffold` verb group: `release`
// writes exactly two skeleton files — a Keep a Changelog CHANGELOG.md and a
// headings-only RELEASING.md — prescribing no release tooling of any kind: no
// workflow, no justfile recipes, no language detection, no shell-out, no
// command named in either file.
//
// Deliberately unrelated to brine (T-113): `pickle install` continues to
// scaffold the ticket flow only, and nothing this package writes is ever read
// back by `pickle doctor`/`pickle board audit` or any other pickle command —
// it is a one-shot scaffold, not an ongoing invariant pickle owns.
package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// projectNameToken is substituted, verbatim, for opts.ProjectName in every
// template file. Plain string replacement — not text/template — because a
// single token across two skeleton files does not need a template engine, and
// the token stays legible in the checked-in templates themselves.
const projectNameToken = "__PROJECT_NAME__"

// templateFile pairs an embedded template asset with the path it is written
// to, relative to the target root.
type templateFile struct {
	Asset     string
	Installed string
}

// releaseTemplateFiles is the fixed set of files `pickle scaffold release`
// writes — exactly two, no language branch (decision 8, T-113).
var releaseTemplateFiles = []templateFile{
	{"scaffold/release-template/CHANGELOG.md", "CHANGELOG.md"},
	{"scaffold/release-template/RELEASING.md", "RELEASING.md"},
}

// Options configures a single scaffold run.
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
	// Notes carries longer, human-directed guidance — currently just the
	// dry-run previews writeTemplates emits.
	Notes []string
}

func (r *Result) created(f string) { r.Created = append(r.Created, f) }
func (r *Result) skipped(f string) { r.Skipped = append(r.Skipped, f) }
func (r *Result) note(n string)    { r.Notes = append(r.Notes, n) }

// Release performs a scaffold release run into root using the embedded
// payload FS (the binary's "scaffold/release-template" tree). It writes
// exactly two files — a Keep a Changelog CHANGELOG.md and a headings-only
// RELEASING.md — and nothing else: no justfile recipes, no GitHub Actions
// workflow, no shell-out to any release tool, no pickle.toml read, no
// doctor/board-audit integration (decision 8, T-113).
func Release(payload fs.FS, root string, opts Options) (Result, error) {
	var res Result

	if err := writeTemplates(payload, root, releaseTemplateFiles, opts.ProjectName, opts, &res); err != nil {
		return res, err
	}

	return res, nil
}

// writeTemplates writes each of files into root, substituting name for
// projectNameToken in every template's content. An existing destination file
// is left untouched unless opts.Force is set, and opts.DryRun reports what
// would happen without writing anything.
func writeTemplates(payload fs.FS, root string, files []templateFile, name string, opts Options, res *Result) error {
	if name == "" {
		name = filepath.Base(root)
	}
	token := []byte(projectNameToken)
	replacement := []byte(name)

	for _, tf := range files {
		data, err := fs.ReadFile(payload, tf.Asset)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", tf.Asset, err)
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
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		res.created(tf.Installed)
	}

	return nil
}
