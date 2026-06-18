# Dataview

Notes Web supports server-side rendering of Obsidian-style Dataview queries in
fenced `dataview` code blocks.

## Supported query types

- `TABLE` — render columns from frontmatter or computed expressions.
- `LIST` — render matching notes as a list.
- `TASK` — render task lists from matching notes.

## TABLE syntax

Basic table query:

```dataview
TABLE status, file.name AS "Name", file.folder AS "Folder"
FROM "Projects"
SORT file.name
```

### FILTER clause

Add interactive column dropdown filters to any `TABLE` query:

```dataview
FILTER <field> [DEFAULT <value-or-list>] [MODE single|multi] [CLEARABLE]
```

- **`<field>`** — a visible column source field (e.g., `status`, `tags`, `file.name`).
  Dots are allowed for nested fields like `file.tags`.
- **`DEFAULT`** — initial value(s) selected on page load. Optional.
  - Single mode: exactly one scalar value. List syntax (`[a, b]`) is an error.
  - Multi mode: a list wrapped in brackets (`[a, b]`). A scalar default is an error.
  - Values with spaces must be quoted.
- **`MODE`** — `single` (native `<select>`, default) or `multi` (checkbox dropdown).
  Case-insensitive.
- **`CLEARABLE`** — exposes an `All` option that clears the filter.
  Without `CLEARABLE` and without a `DEFAULT`, a disabled placeholder is shown.

Multiple filters combine with AND. Multi-mode values combine with OR within the
same field. Filters apply server-side, before `LIMIT`.

#### Single-mode examples

```dataview
TABLE status, file.name AS "Name"
FROM "Projects"
SORT file.name
FILTER status DEFAULT "active" CLEARABLE
FILTER priority MODE single
```

#### Multi-mode example

```dataview
TABLE tags, file.name AS "Name"
FROM "Projects"
SORT file.name
FILTER tags MODE multi
```

### Tag fields

Fields named `tags`, `file.tags`, and `file.etags` automatically display values
with a leading `#` prefix. For example, a frontmatter value `project` appears
as `#project` in the filter dropdown.

Defaults and AJAX parameters for tag fields must also use the `#` prefix.

### Text filter

Each Dataview table includes a `Filter table…` search input. It filters rows by
case-insensitive text containment. The text filter combines with dropdown
filters using AND. When you type, a 200 ms debounce prevents excessive requests.

### Sorting

Click any column header whose source is a simple field path to sort by that
column:

- First click: descending.
- Subsequent clicks: toggle descending ↔ ascending.
- `aria-sort` on the active header reflects the current direction.
- Complex expression columns (e.g., `dateformat(...)`) are not sortable.

### Pagination

The `Rows` dropdown controls how many rows appear per page. Pagination is
client-side and resets to page 1 whenever you change a filter, text query, or
sort. The `Rows` value is preserved across AJAX updates.

### No results

When no rows match the current filter combination, the table header remains
visible and a `No matching rows` message is shown below it.

### AJAX action URL (debugging)

All filter, sort, and text interactions are performed via AJAX. The server
returns a partial HTML fragment. For debugging or direct access, the same URL
format works in the browser:

```text
/Projects/Index.md?action=renderDataviewTable&table=1&filter.status=done
```

Parameters:

| Parameter       | Required | Description                                          |
|-----------------|----------|------------------------------------------------------|
| `action`        | yes      | Must be `renderDataviewTable`.                       |
| `table`         | yes      | 1-based table index in document order.               |
| `filter.<field>`| no       | Filter value for the given field. Repeat for multi.  |
| `q`             | no       | Text search query (case-insensitive contains).        |
| `sort`          | no       | Field to sort by (requires `dir`).                   |
| `dir`           | no       | Sort direction: `asc` or `desc` (requires `sort`).   |

Omitted filter params mean "All" (no filter active). The response is the full
`.dataview-table-wrap` HTML fragment including controls, table, and pager.

If the request is invalid (wrong method, unknown action, missing table,
undeclared filter, invalid sort/dir), the response is an HTML error fragment
with class `dataview-error` and an appropriate HTTP status code.

### Full example

```dataview
TABLE status, tags, file.link AS "Name"
FROM "Projects"
SORT file.name
FILTER status DEFAULT "active" CLEARABLE
FILTER tags MODE multi
```

This renders a project table with:

- A single-select `status` dropdown defaulting to `active` with an `All` option.
- A multi-select `tags` dropdown with checkboxes and `#`-prefixed values.
- Server-side text search via `Filter table…`.
- Clickable column headers for server-side sorting.
- Client-side pagination via the `Rows` dropdown.

## LIST and TASK queries

LIST and TASK queries render without interactive controls. They display matching
notes and tasks as formatted lists. `FILTER` is not supported in non-TABLE
queries and produces a visible Dataview error.
