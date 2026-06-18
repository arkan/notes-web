# Rule: testing

Rules for validation, fixtures, and test placement.

## Applies to

- `internal/app/*_test.go`
- `tests/e2e/**`
- `testdata/e2e-vault/**`
- `package*.json`
- `playwright.config.ts`
- `Makefile`

## Hard rules

1. Add or update tests for every behavior change at the correct level.
2. Use Go tests for parser, evaluator, route, config, path, security, and rendering-unit behavior.
3. Use Playwright for browser-visible rendering, keyboard behavior, navigation, AJAX, and template/CSS/JS integration.
4. Use `testdata/e2e-vault` as the canonical browser-test vault.
5. Never run tests against `/home/arkan/hermes`.
6. When adding supported Markdown or Dataview syntax, update fixture notes and E2E assertions in the same change.
7. Keep current E2E coverage areas protected: Dataview filters, Markdown rendering, internal pages, and Dataview gallery.
8. Run `make test-go` for Go validation; avoid `go test ./...` after npm dependencies are installed.
9. Run `make lint` for TypeScript/ESLint validation.
10. Run `make test-e2e` for Playwright validation when browser behavior changes.
11. Run `make test` before claiming multi-file behavior changes complete when feasible.
12. Run `git diff --check` before final response for any code or doc change.
13. If full validation is skipped, state exactly what was run and why.
14. Keep test fixtures small, explicit, and representative of real vault behavior.

## Patterns

```bash
make test-go
make lint
make test-e2e
make test
git diff --check
```

## Load on demand

- `tests/e2e/dataview-filters.spec.ts`
- `tests/e2e/markdown-rendering.spec.ts`
- `tests/e2e/internal-pages.spec.ts`
- `tests/e2e/dataview-gallery.spec.ts`
- `playwright.config.ts`
