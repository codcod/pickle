package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codcod/pickle/internal/board"
	"github.com/codcod/pickle/internal/move"
	"github.com/codcod/pickle/internal/ticket"
)

// Ticket mechanics. `ticket new` lands in P1 (id allocation + template + board
// row); `ticket move` in P3 (state machine, per-child WIP, cross-child merge gate).

func runTicket(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pickle ticket <new|move> ...")
		return exitUsage
	}
	switch args[0] {
	case "new":
		return runTicketNew(args[1:])
	case "move":
		return runTicketMove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pickle ticket: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

const (
	ticketNewUsage  = `usage: pickle ticket new "<title>" --project <name> [--impact V --complexity V --cost V] [--spawned-by "T-NNN[,T-MMM]"]`
	ticketMoveUsage = `usage: pickle ticket move <T-NNN> <status> [--reason "<why>"]`
)

func runTicketMove(args []string) int {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") || strings.HasPrefix(args[1], "-") {
		fmt.Fprintln(os.Stderr, ticketMoveUsage)
		return exitUsage
	}
	id, status := args[0], args[1]

	fs := flag.NewFlagSet("ticket move", flag.ContinueOnError)
	reason := fs.String("reason", "", "why the move is happening (required for backward/rework/drop moves)")
	if err := fs.Parse(args[2:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, ticketMoveUsage)
		return exitUsage
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	res, err := move.Move(cfg.Root(), cfg, id, status, *reason)
	if err != nil {
		return errf("%v", err)
	}
	fmt.Printf("moved %s: %s → %s  (%s)\n", id, res.From, res.To, res.Path)
	for _, w := range res.Warnings {
		fmt.Printf("  warning: %s\n", w)
	}
	return exitOK
}

func runTicketNew(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, ticketNewUsage)
		return exitUsage
	}
	title := args[0]

	fs := flag.NewFlagSet("ticket new", flag.ContinueOnError)
	project := fs.String("project", "", "target child-project (required)")
	impact := fs.String("impact", "medium", "impact grade")
	complexity := fs.String("complexity", "medium", "complexity grade")
	cost := fs.String("cost", "M", "cost grade")
	spawnedBy := fs.String("spawned-by", "", "lineage: ticket id(s) this one was born from, comma-separated (non-gating)")
	if err := fs.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, ticketNewUsage)
		return exitUsage
	}
	if *project == "" {
		return errf("--project is required")
	}
	for _, g := range []struct{ kind, v string }{{"impact", *impact}, {"complexity", *complexity}, {"cost", *cost}} {
		if !ticket.ValidGrade(g.kind, g.v) {
			return errf("illegal %s value %q (legal: single values or adjacent-pair ranges)", g.kind, g.v)
		}
	}
	if err := validateTitle(title); err != nil {
		return errf("%v", err)
	}
	// Shape-checked and de-duplicated here, before anything is written. Whether the
	// ids *exist* is still the audit's job: --spawned-by T-999 for a ticket not yet
	// filed is legal input, and `pickle board audit` is what flags it.
	lineage, err := ticket.ParseIDList(*spawnedBy)
	if err != nil {
		return errf("--spawned-by: %v", err)
	}

	cfg, code := loadConfig()
	if code != exitOK {
		return code
	}
	if _, ok := cfg.Project(*project); !ok {
		return errf("project %q is not a registered child", *project)
	}

	root := cfg.Root()
	id := fmt.Sprintf("T-%03d", ticket.NextNum(root))
	slug := ticket.Slugify(title)
	rel := filepath.Join("tickets", "1-to-do", id+"-"+slug+".md")
	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); err == nil {
		return errf("%s already exists", rel)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errf("%v", err)
	}
	if err := os.WriteFile(path, []byte(ticket.Scaffold(id, title, *project, *impact, *complexity, *cost, lineage)), 0o644); err != nil {
		return errf("%v", err)
	}
	if err := board.AddTODORow(filepath.Join(root, "tickets", "BOARD.md"), *project, id, title, *impact, *complexity, *cost); err != nil {
		return errf("%v", err)
	}
	fmt.Printf("created %s  (%s)\n", id, rel)
	return exitOK
}

// validateTitle rejects titles that cannot be rendered safely. A newline is the
// load-bearing case: ticket.Scaffold interpolates the title into the frontmatter
// block, so a newline injects extra keys, leaks the remainder into the document
// body below the H1, and splits the ticket's BOARD.md row across two physical
// lines — none of which `pickle board audit` currently notices.
//
// Deliberately a rejection, not a sanitisation like move.sanitizeReason: a
// --reason is free text, but a title becomes the filename, the H1 and a board
// cell, so quietly rewriting it hands back a ticket nobody asked for (T-030
// decision 1).
//
// Not a character whitelist, and not a defence against markdown-breaking cells:
// a '|' still reaches the board row, because escaping belongs at the render
// boundary (T-014), not here.
func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		// Otherwise ticket.Slugify's "untitled" fallback names the file after
		// nothing at all: T-00N-untitled.md. Only whitespace is caught here —
		// an unsluggable title like "###" still reaches that fallback, by design.
		return errors.New("title is empty")
	}
	if strings.ContainsAny(title, "\n\r") {
		return errors.New("title may not contain newlines (it is written into the frontmatter, the heading and a board cell)")
	}
	// A "---" title cannot actually truncate the frontmatter — Scaffold writes it
	// after a literal "title: " prefix, and ParseFrontmatter's terminator needs a
	// line that trims to exactly "---". It is rejected as a degenerate title, not
	// as an injection defence. (Reachable only when padded: bare "---" is caught
	// by the "-" prefix guard in runTicketNew and exits as a usage error.)
	if strings.TrimSpace(title) == "---" {
		return errors.New(`title may not be "---"`)
	}
	return nil
}
