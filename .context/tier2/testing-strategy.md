# Testing Strategy

Use the smallest test level that proves the behavior, then add browser coverage when the user-visible integration matters.

## Test levels

- Go tests in `internal/app/*_test.go`: parser, evaluator, renderer units, route handlers, config defaults, security boundaries, hidden paths, search, dashboards, diagnostics.
- Playwright tests in `tests/e2e/*.spec.ts`: rendered HTML, keyboard behavior, navigation, AJAX, local browser state, CSS/template/JS integration.
- Fixture vault in `testdata/e2e-vault`: browser-test source of truth; keep it small and explicit.

## Existing E2E coverage areas

- `markdown-rendering.spec.ts` — Markdown features, wikilinks, panels, copy buttons.
- `internal-pages.spec.ts` — Modern Workbench shell, home, folders, search, tags, diagnostics, missing/resolve, command palette, Settings, TODO/Tasks, Projects, Calendar, Maintenance, mobile overflow.
- `edit-mode.spec.ts` — edit CRUD, Inbox/capture, Trash/restore, action menus, write workflows.
- `dataview-gallery.spec.ts` — supported Dataview query types and diagnostics.
- `dataview-filters.spec.ts` — Dataview TABLE AJAX filters, sorting, debounce, pagination, keyboard behavior.

## Command matrix

```bash
make test-go     # Go tests: go test ./cmd/... ./internal/...
make lint        # ESLint/TypeScript validation
make test-e2e    # Playwright against testdata/e2e-vault
make test        # test-go -> lint -> test-e2e
git diff --check # whitespace check for any code/doc change
```

After `npm ci`, avoid plain `go test ./...` because `node_modules` may contain Go packages. Use `make test-go` or `go test ./cmd/... ./internal/...`.

## Change-to-test mapping

- Parser/evaluator/config/path/security change: Go tests first.
- Template/CSS/JS/keyboard/AJAX visible change: Playwright E2E plus any Go handler tests for server contract.
- Dataview syntax or semantics: Go parser/evaluator/render tests, fixture note updates, Playwright assertions, and `docs/dataview.md` updates.
- Dataview CSS/diagnostics changes: `dataview-gallery.spec.ts`, `dataview-filters.spec.ts`, and mobile overflow checks when tables/code diagnostics are affected.
- Markdown rendering behavior: Go renderer tests when possible, plus Playwright on `Syntax/All Syntaxes.md` when browser-visible.
- Homepage/config behavior: Go tests for typed config and view models; Playwright only for rendered block/interactivity regressions.
- Settings/palette/local preference behavior: Playwright for persistence/focus/mobile overflow, Go static contracts for embedded JS/CSS hooks.
- Write-path behavior: Go tests for auth/CSRF/path policy/collision/rollback and Playwright for visible workflows.
- Nix/Makefile/package tooling: command smoke checks and doc alignment; full E2E when Playwright config changes.

## Fixture vault rules

- Use `testdata/e2e-vault`, never `/home/arkan/hermes`, for automated tests.
- Prefer one fixture note per behavior cluster over a giant catch-all note, except `Syntax/All Syntaxes.md` which is intentionally broad.
- Keep Dataview fixtures representative of real vault frontmatter, tags, tasks, and folders.
- When adding hidden-path behavior, include both dot-hidden and configured-hidden cases in Go tests.

## Playwright mechanics

- `playwright.config.ts` starts `go run ./cmd/notes-web --vault ./testdata/e2e-vault --host 127.0.0.1 --port 18081`.
- Tests are not fully parallel; CI uses one worker and one retry.
- Use stable selectors from data attributes or semantic roles where possible.
- After AJAX DOM replacement, wait for `data-dataview-loading` to disappear and allow a short settle when needed.

## Review expectations

- For multi-file behavior changes, run `make test` when feasible.
- If full validation is too expensive or blocked, state exactly what passed and what was skipped.
- Embedded asset bugs must be verified against the process actually serving the page, not only the edited source.
- Any phase touching shared templates/static assets should end with `make test` and `git diff --check` before handoff.
