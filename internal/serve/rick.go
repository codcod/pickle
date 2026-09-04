package serve

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/rickstatus"
	"github.com/codcod/pickle/internal/ticket"
)

// buildRickReports queries rickstatus once per rick-enabled child, keyed by
// project name (T-077 decision 3): the board renders every ticket in one
// request, so a per-row query would mean one exec.Command per ticket.
func buildRickReports(cfg *config.Config, root string) map[string]rickstatus.Report {
	reports := make(map[string]rickstatus.Report, len(cfg.Projects))
	for i := range cfg.Projects {
		p := &cfg.Projects[i]
		if !p.Rick {
			continue
		}
		reports[p.Name] = rickstatus.Query(root, p)
	}
	return reports
}

// rickPendingCount is the board row's compact badge count (decision 6): how
// many of a ticket's artifacts are not yet approved.
func rickPendingCount(artifacts []rickstatus.Artifact) int {
	n := 0
	for _, a := range artifacts {
		if a.Status != "approved" {
			n++
		}
	}
	return n
}

// ArtifactView is one rick artifact as the ticket page shows it: its own
// fields plus the effective-instance-rule flags (decisions 4-5).
type ArtifactView struct {
	Kind      string
	Status    string
	Name      string // basename of Artifact.Path — the route's whitelist key
	Href      string
	Date      string
	Effective bool
	Duplicate bool
	Mismatch  bool
}

// artifactDateRE pulls the tie-break's primary key out of a rick artifact
// filename (decision 4): "solution-design-2026-06-14-x.md" -> "2026-06-14".
var artifactDateRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// artifactTieBreakLess reports whether a sorts strictly before b under
// rick's own effective-instance rule (decision 4, reproduced best-effort —
// see the ticket's "Re-verified" note): filename-date descending, then
// Artifact.Date descending, then the full Path string descending — status-
// blind throughout.
func artifactTieBreakLess(a, b rickstatus.Artifact) bool {
	ad, bd := artifactDateRE.FindString(a.Path), artifactDateRE.FindString(b.Path)
	if ad != bd {
		return ad > bd
	}
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	return a.Path > b.Path
}

// buildArtifacts groups one ticket's rick artifacts by Kind and marks, per
// kind, which instance is Effective under rick's tie-break rule. A kind
// holding more than one instance is Duplicate; its Effective instance is
// also Mismatch when it is not yet approved — the signal the Description
// says earns this view its keep. This is UI-only and advisory (decision 5):
// it never feeds internal/audit.
func buildArtifacts(rep rickstatus.Report, ticketID, basePath string) []ArtifactView {
	artifacts := rep.For(ticketID)
	if len(artifacts) == 0 {
		return nil
	}

	byKind := make(map[string][]rickstatus.Artifact)
	var kinds []string
	for _, a := range artifacts {
		if _, ok := byKind[a.Kind]; !ok {
			kinds = append(kinds, a.Kind)
		}
		byKind[a.Kind] = append(byKind[a.Kind], a)
	}
	sort.Strings(kinds)

	var views []ArtifactView
	for _, kind := range kinds {
		group := byKind[kind]
		sort.SliceStable(group, func(i, j int) bool {
			return artifactTieBreakLess(group[i], group[j])
		})
		duplicate := len(group) > 1
		for i, a := range group {
			name := filepath.Base(a.Path)
			effective := i == 0
			views = append(views, ArtifactView{
				Kind:      a.Kind,
				Status:    a.Status,
				Name:      name,
				Href:      basePath + "/specs/" + ticketID + "/" + name,
				Date:      a.Date,
				Effective: effective,
				Duplicate: duplicate,
				Mismatch:  effective && duplicate && a.Status != "approved",
			})
		}
	}
	return views
}

// resolveArtifact implements the route's entire security model (decision 2):
// name is servable only when this request's own fresh rickstatus.Query
// reported it for key, under the child that key's ticket prefix names —
// nothing else is servable even if it physically exists on disk. A name
// containing '/' or '..' is rejected up front, belt-and-braces over the
// whitelist check that would already miss it.
func resolveArtifact(cfg *config.Config, root, key, name string) (path string, ok bool) {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", false
	}
	prefix, _, split := ticket.SplitID(key)
	if !split {
		return "", false
	}
	var proj *config.Project
	for i := range cfg.Projects {
		if cfg.Projects[i].Prefix() == prefix {
			proj = &cfg.Projects[i]
			break
		}
	}
	if proj == nil {
		return "", false
	}
	rep := rickstatus.Query(root, proj)
	for _, a := range rep.For(key) {
		if filepath.Base(a.Path) == name {
			return filepath.Join(root, proj.Path, a.Path), true
		}
	}
	return "", false
}
