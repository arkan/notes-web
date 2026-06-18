# Rule: Dataview

Rules for the limited server-side Dataview implementation.

## Applies to

- `internal/app/dataview*.go`
- `docs/dataview.md`
- `tests/e2e/dataview-*.spec.ts`
- `testdata/e2e-vault/**/Dataview*.md`

## Hard rules

1. Support fenced `dataview` blocks only; inline Dataview and `dataviewjs` stay unsupported.
2. Keep supported query types to `TABLE`, `LIST`, `TASK`, and `CALENDAR` unless explicitly planned.
3. Keep `FILTER` valid for `TABLE` only; `LIST`, `TASK`, or `CALENDAR` with `FILTER` must render a visible Dataview error.
4. Keep `FILTER` syntax: `FILTER <field> [DEFAULT <value-or-list>] [MODE single|multi] [CLEARABLE]`.
5. Allow filtering and user sorting only on visible simple-field table columns.
6. Combine different filters with AND and multi filter values with OR.
7. Treat `DEFAULT` as initial full-page state only; omitted AJAX filter params mean `All`.
8. Display and filter `tags`, `file.tags`, and `file.etags` with a leading `#`.
9. Ensure `project` does not match `#project` for tag filters.
10. Keep the table AJAX action note-local: `?action=renderDataviewTable&table=N`.
11. Do not add a separate `/_dataview/table` endpoint or `path` query parameter.
12. Keep `table=N` 1-based and counted only across Dataview `TABLE` outputs in document order.
13. Return a `.dataview-table-wrap` fragment for valid AJAX table renders.
14. Return a `dataview-error` fragment with an appropriate status for invalid AJAX action requests.
15. Keep full-page and AJAX partial rendering on the same scanner/table-indexing path.
16. Preserve pipeline order: parse, FROM/WHERE, FLATTEN, GROUP BY, SORT, options, FILTER/q, LIMIT, render.
17. When changing syntax, update `docs/dataview.md`, fixture notes, Go tests, and Playwright assertions as needed.

## Patterns

```text
/Some/Page.md?action=renderDataviewTable&table=1&filter.status=done
```

```text
Pipeline: parse -> FROM/WHERE -> FLATTEN -> GROUP BY -> SORT -> options -> FILTER/q -> LIMIT -> render
```

## Load on demand

- `docs/dataview.md` for user-facing syntax.
- `tests/e2e/dataview-filters.spec.ts` for AJAX filter behavior.
- `tests/e2e/dataview-gallery.spec.ts` for query gallery coverage.
