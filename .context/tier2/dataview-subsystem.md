# Dataview Subsystem

Notes Web implements a server-side, intentionally limited subset of Obsidian Dataview for trusted vault Markdown. `docs/dataview.md` is the user-facing syntax source; this file explains implementation seams and hazards.

## Source map

- `dataview.go` — query model, parser, evaluator, legacy renderers, diagnostics, expression helpers.
- `dataview_filter.go` — `FILTER` parser, modes, field validation, tag-field helpers.
- `dataview_eval.go` — TABLE interactive pipeline, filter states, selected values, q matching.
- `dataview_render.go` — TABLE controls/HTML, AJAX param parsing and validation helpers.
- `dataview_scan.go` — fenced block scanner, document-order TABLE indexing, full render replacement.
- `dataview_action.go` — note-local AJAX handler.
- `internal/app/static/app.js` — Dataview table state machine and multi-filter keyboard UI.

## Supported surface

- Fenced `dataview` code blocks only.
- Query kinds: `TABLE`, `LIST`, `TASK`, `CALENDAR`.
- Clauses: `FROM`, `WHERE`, `SORT`, `LIMIT`, `GROUP BY`, `FLATTEN`; `FILTER` is TABLE-only.
- Inline Dataview and `dataviewjs` are intentionally unsupported; diagnostics list `dataviewjs` as unsupported and do not execute it.

## Evaluation pipeline

General row evaluation starts with notes from `VaultIndex`:

1. Source match from `FROM`.
2. `WHERE` boolean expression.
3. `FLATTEN` expansion.
4. `GROUP BY` aggregation row creation.
5. Query `SORT`.
6. Query `LIMIT`.

Interactive TABLE rendering adjusts the order:

1. `FROM` / `WHERE`.
2. `FLATTEN`.
3. `GROUP BY`.
4. User sort if provided, otherwise query sort.
5. Compute filter options from visible column values.
6. Apply dropdown filters and text `q`.
7. Apply `LIMIT` after filters.
8. Render the table fragment.

This order is user-visible; changing it requires docs, Go tests, and E2E updates.

## TABLE filters

- Syntax: `FILTER <field> [DEFAULT <value-or-list>] [MODE single|multi] [CLEARABLE]`.
- Field must be a visible simple-field column expression, not a function or hidden expression.
- Different fields combine with AND; multi values within one field combine with OR.
- `DEFAULT` applies to full-page render only. During AJAX, omitted filter params mean `All`.
- Single mode rejects repeated AJAX values; multi mode deduplicates repeated values.
- Tag fields `tags`, `file.tags`, and `file.etags` display/filter with `#` prefixes; `project` must not match `#project`.

## AJAX contract

- URL shape: `/Some/Note.md?action=renderDataviewTable&table=1&filter.status=done`.
- The action is handled inside `Server.path`, after normal auth, URL path resolution, and hidden checks.
- `table=N` is 1-based and counts only TABLE Dataview blocks in document order.
- The response for valid requests is a full `.dataview-table-wrap` fragment.
- Invalid requests return a `.dataview-error` fragment with an appropriate status.
- A `path` query key is rejected even when empty; the note path must come from the URL path.

## JavaScript contract

- `initDataviewTables` enhances server-rendered wrappers.
- Text `q` uses a 200 ms debounce and AbortController to discard stale responses.
- Filter/sort changes replace the wrapper with the server fragment and reinitialize behavior.
- Rows pagination is client-side over the rows returned by the server and preserves page-size through replacement.
- Multi filters use a hidden `<select multiple>` as state source plus a keyboard-accessible checkbox menu.
- Dataview state is not stored in URL history, localStorage, or server session.

## Expressions and fields

- Built-in note fields include `file.link`, `file.name`, `file.path`, `file.folder`, `file.ext`, `file.mtime`, `file.ctime`, `file.tags`, `file.etags`, and `file.outlinks`.
- Heavy fields `file.content` and `file.inlinks` are populated only when an expression references them.
- Task fields include `text`, `completed`, dates, `priority`, `tags`, `line`, `path`, and `link`.
- Functions include `list`, `link`, `contains`, `default`, `date`, `dateformat`, `choice`, string prefix/suffix/regex helpers, length, and basic aggregations.

## Extension checklist

- Add parser support and evaluator behavior together; parser-only support creates misleading diagnostics.
- Keep full-page rendering and AJAX partial rendering on the same scanner/table indexing path.
- Update `docs/dataview.md`, Go tests, fixture notes, and Playwright assertions in the same change.
- Validate new AJAX params against the parsed query before evaluating rows.
- Escape all generated labels, values, error strings, and attributes.
