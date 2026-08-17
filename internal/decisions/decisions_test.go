package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codcod/pickle/internal/config"
	"github.com/codcod/pickle/internal/flow"
)

func TestExtract(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []Decision
	}{
		{
			name: "structured form: leading bold run is the statement",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **The hook performs no network I/O.** No `git fetch` — a stale ref can only\n" +
				"   ever be stale, never wrong.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "The hook performs no network I/O."},
			},
		},
		{
			name: "unstructured item: no leading bold run, raw first line carried instead",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. Use the existing tabwriter, matching runProjectList.\n",
			want: []Decision{
				{Number: 1, Structured: false, Raw: "Use the existing tabwriter, matching runProjectList."},
			},
		},
		{
			name: "a gap in the numbering is reported as written, never recounted",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **First.** Rationale.\n" +
				"2. **Second.** Rationale.\n" +
				"4. **Fourth, because 3 was dropped.** Rationale.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "First."},
				{Number: 2, Structured: true, Statement: "Second."},
				{Number: 4, Structured: true, Statement: "Fourth, because 3 was dropped."},
			},
		},
		{
			name: "a nested, indented ordinal is not a decision of its own",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **Outer decision.** With a nested illustration:\n" +
				"   1. not a decision\n" +
				"   2. still not a decision\n" +
				"2. **Next decision.** Rationale.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "Outer decision."},
				{Number: 2, Structured: true, Statement: "Next decision."},
			},
		},
		{
			name: "a numbered line inside a fenced code block is not a decision",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **Real decision.** See the example:\n" +
				"   ```\n" +
				"   1. this is example code, not a decision\n" +
				"   2. neither is this\n" +
				"   ```\n" +
				"2. **Second real decision.** Rationale.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "Real decision."},
				{Number: 2, Structured: true, Statement: "Second real decision."},
			},
		},
		{
			// Measured against this repo's own corpus (T-054 decision 1, T-090
			// decision 1, and a dozen others): the bold statement itself routinely
			// wraps onto a second physical line before its closing "**" — an
			// ordinary markdown soft wrap, not a second paragraph. A parser that
			// only checked the first physical line misread every one of these as
			// unstructured.
			name: "a leading bold run that wraps onto a second physical line is still structured",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **Mechanism: a block that overrides the custom\n" +
				"   properties.** Not `light-dark()`. The dark values stay byte-identical.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "Mechanism: a block that overrides the custom properties."},
			},
		},
		{
			name: "a multi-line item is one decision, not several",
			text: "## Implementation Plan\n\n" +
				"### Confirmed design decisions (do not deviate without asking)\n\n" +
				"1. **One decision, three lines.** The rationale continues here\n" +
				"   and wraps again on a third physical line before the next\n" +
				"   item begins.\n" +
				"2. **Second.** Short.\n",
			want: []Decision{
				{Number: 1, Structured: true, Statement: "One decision, three lines."},
				{Number: 2, Structured: true, Statement: "Second."},
			},
		},
		{
			name: "a ticket with no Implementation Plan section yields nothing",
			text: "## Description\n\nSome prose.\n\n## Review\n\nfindings\n",
			want: nil,
		},
		{
			name: "a plan with no Confirmed design decisions subsection yields nothing",
			text: "## Implementation Plan\n\n### Feature branch (mandatory)\n\ndetail\n\n" +
				"### Tasks\n\n#### Task 1\n\ndetail\n",
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Extract(c.text)
			if len(got) != len(c.want) {
				t.Fatalf("Extract() = %d decisions, want %d (%+v)", len(got), len(c.want), got)
			}
			for i := range got {
				if got[i].Number != c.want[i].Number ||
					got[i].Structured != c.want[i].Structured ||
					got[i].Statement != c.want[i].Statement ||
					got[i].Raw != c.want[i].Raw {
					t.Errorf("Extract()[%d] = %+v, want Number=%d Structured=%v Statement=%q Raw=%q",
						i, got[i], c.want[i].Number, c.want[i].Structured, c.want[i].Statement, c.want[i].Raw)
				}
			}
		})
	}
}

// TestExtractGrepTargetsWholeItem confirms the unexported `full` field (the
// only channel Query has for T-105 decision 9: --grep must search the whole
// item, not only the projected statement) actually carries the rationale,
// not just the first line.
func TestExtractGrepTargetsWholeItem(t *testing.T) {
	text := "## Implementation Plan\n\n" +
		"### Confirmed design decisions (do not deviate without asking)\n\n" +
		"1. **Short statement.** The word zephyr only appears down here, in the\n" +
		"   rationale, never in the statement itself.\n"
	got := Extract(text)
	if len(got) != 1 {
		t.Fatalf("Extract() = %d decisions, want 1", len(got))
	}
	if !strings.Contains(got[0].full, "zephyr") {
		t.Errorf("Extract()[0].full = %q, want it to contain the rationale text", got[0].full)
	}
	if strings.Contains(got[0].Statement, "zephyr") {
		t.Errorf("Extract()[0].Statement = %q, must not leak rationale text", got[0].Statement)
	}
}

// newTestTree builds a minimal on-disk tickets/ tree under a temp dir, with
// one project registered in a *config.Config, for Query's integration tests.
// It writes ticket files directly rather than going through the CLI's
// scaffolding, since only frontmatter + a Confirmed design decisions
// subsection matter here.
func newTestTree(t *testing.T) (root string, def *flow.Definition, cfg *config.Config) {
	t.Helper()
	root = t.TempDir()
	def = flow.ForName("brine")
	for _, s := range def.States() {
		if err := os.MkdirAll(filepath.Join(root, "tickets", s.Dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg = &config.Config{
		Projects: []config.Project{{Name: "alpha", Path: "."}, {Name: "beta", Path: "./beta"}},
	}

	write := func(dir, name, project, plan string) {
		text := "---\nid: " + name[:strings.LastIndexByte(name, '-')] + "\ntitle: t\nproject: " + project + "\n" +
			"depends-on: []\nspawned-by: []\nimpact: low\ncomplexity: low\ncost: S\n---\n\n" +
			"# " + name + "\n\n## Outcome\n\nsomething.\n\n## Description\n\nprose.\n\n" +
			plan + "\n\n## Review\n\n## History\n\n- 2026-08-16 — created (TO DO). source: chat: test\n"
		p := filepath.Join(root, "tickets", dir, name+".md")
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("6-done", "T-001-alpha-one", "alpha",
		"## Implementation Plan\n\n### Confirmed design decisions (do not deviate without asking)\n\n"+
			"1. **Alpha decision one.** About widgets.\n")
	write("6-done", "T-002-alpha-two", "alpha",
		"## Implementation Plan\n\n### Confirmed design decisions (do not deviate without asking)\n\n"+
			"1. **Alpha decision two.** About gadgets.\n")
	write("1-to-do", "T-003-alpha-todo", "alpha",
		"## Implementation Plan\n\n<!-- empty until refined -->\n")
	write("6-done", "T-004-beta-one", "beta",
		"## Implementation Plan\n\n### Confirmed design decisions (do not deviate without asking)\n\n"+
			"1. **Beta decision one.** About sprockets.\n")

	return root, def, cfg
}

func TestQuery(t *testing.T) {
	root, def, cfg := newTestTree(t)

	t.Run("no filters returns every decision, sorted by ticket then ordinal", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Decisions) != 3 {
			t.Fatalf("got %d decisions, want 3: %+v", len(res.Decisions), res.Decisions)
		}
		var ids []string
		for _, d := range res.Decisions {
			ids = append(ids, d.Citation)
		}
		want := []string{"T-001 decision 1", "T-002 decision 1", "T-004 decision 1"}
		for i, w := range want {
			if ids[i] != w {
				t.Errorf("Decisions[%d].Citation = %q, want %q (all: %v)", i, ids[i], w, ids)
			}
		}
	})

	t.Run("project filter narrows to that child only", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{Project: "beta"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Decisions) != 1 || res.Decisions[0].TicketID != "T-004" {
			t.Fatalf("got %+v, want exactly T-004's one decision", res.Decisions)
		}
	})

	t.Run("unregistered project is an error", func(t *testing.T) {
		if _, err := Query(def, root, cfg, Filter{Project: "no-such-child"}); err == nil {
			t.Fatal("want an error for an unregistered project, got nil")
		}
	})

	t.Run("status filter: TO DO holds no decisions — empty, not an error", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{Status: "1-to-do"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Decisions) != 0 {
			t.Fatalf("got %d decisions in 1-to-do, want 0", len(res.Decisions))
		}
	})

	t.Run("unknown status directory is an error naming the legal values", func(t *testing.T) {
		_, err := Query(def, root, cfg, Filter{Status: "9-nowhere"})
		if err == nil {
			t.Fatal("want an error for an unknown status, got nil")
		}
		if !strings.Contains(err.Error(), "1-to-do") {
			t.Errorf("error %q does not name the legal status directories", err.Error())
		}
	})

	t.Run("grep matches the rationale, not only the statement", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{Grep: "sprockets"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Decisions) != 1 || res.Decisions[0].TicketID != "T-004" {
			t.Fatalf("got %+v, want exactly T-004's decision (its rationale mentions sprockets)", res.Decisions)
		}
	})

	t.Run("grep with no matches is empty, not an error", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{Grep: "zzz-no-such-topic-zzz"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Decisions) != 0 {
			t.Fatalf("got %d decisions, want 0", len(res.Decisions))
		}
	})

	t.Run("uncompilable grep pattern is an error", func(t *testing.T) {
		if _, err := Query(def, root, cfg, Filter{Grep: "(["}); err == nil {
			t.Fatal("want an error for an invalid regex, got nil")
		}
	})

	t.Run("Decisions is never nil, even when empty", func(t *testing.T) {
		res, err := Query(def, root, cfg, Filter{Status: "1-to-do"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Decisions == nil {
			t.Fatal("Decisions is nil, want a non-nil empty slice")
		}
	})
}
