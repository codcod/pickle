# Board

The maintained index of every ticket in `tickets/`, across **all registered child-projects**.
**Update this file on every ticket move** — see `tickets/README.md` §6 (board rule). A move
that doesn't touch this file is a bug.

Within each status section, tickets are **sub-grouped by child-project** under a `### <child>`
heading (each child is registered with `pickle project add`; a ticket's target is its
`project:` frontmatter). TO DO / READY are ordered by descending impact **inside** each child's
group. There is one board — no per-child boards.

**WIP limits are per child-project:** `3-in-development/` ≤ 1 · `4-in-review/` ≤ 1 (tune per
project). The count next to each child heading in those two sections is that child's own count.

> The `<child-project>` headings below are placeholders written at install time. `pickle
> install` seeds the first child; `pickle project add` adds more. Delete groups with no
> tickets, or keep them as empty placeholders — your choice.

Last updated: <YYYY-MM-DD>

---

## IN DEVELOPMENT

### <child-project> (0/1)

| id | title | branch | depends-on |
|---|---|---|---|

## IN REVIEW

### <child-project> (0/1)

| id | title | branch | depends-on |
|---|---|---|---|

## REWORK

### <child-project>

| id | title | branch | open findings |
|---|---|---|---|

## READY (impact order, per child)

### <child-project>

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|

## TO DO (impact order, per child)

### <child-project>

| id | title | impact | complexity | cost | depends-on |
|---|---|---|---|---|---|

## DONE

### <child-project>

| id | title | merged |
|---|---|---|

## DROPPED

### <child-project>

| id | title | reason |
|---|---|---|

---

## Known soft couplings (cross-referenced in ticket Descriptions, not `depends-on`)

*(none yet)*
