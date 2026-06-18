# AI Rules

Project-wide hard rules for Notes Web. Keep this file short; put path-specific rules in `.context/rules/`.

## Hard rules

1. Write code, comments, variable names, and project documentation in English unless explicitly requested otherwise.
2. Preserve the product goal: calm, keyboard-first retrieval and vault maintenance before decoration.
3. Ask targeted questions before deciding ambiguous product behavior, syntax, persistence, security, API shape, UI behavior, or external-vault writes.
4. Do not add runtime frontend frameworks, bundling, or runtime npm dependencies.
5. Keep Go boring: standard-library HTTP, Go templates, explicit small functions, and clear subsystem files.
6. Keep server renderers as the source of truth for HTML; JavaScript enhances and requests fragments.
7. Escape all app-generated HTML text, attributes, URLs, labels, errors, and query-derived values.
8. Reject path traversal outside the vault and keep hidden paths hidden across pages, files, actions, and diagnostics.
9. Run internal AJAX actions only after auth and normal path resolution / hidden-path checks.
10. Keep raw HTML enabled for trusted vault Markdown, but never treat app-generated output as trusted.
11. Do not create endpoints that bypass the normal vault path model without an explicit plan and review.
12. Do not modify `/home/arkan/hermes` unless the user explicitly asks and confirms the exact intended diff.
13. Use `testdata/e2e-vault` for browser tests and fixtures; never use the real Hermes vault for tests.
14. Add or update tests for every behavior change at the correct level.
15. Use Go tests for parser, evaluator, routing, security, and config behavior.
16. Use Playwright E2E tests for browser-visible rendering, interaction, navigation, and AJAX behavior.
17. After `npm ci`, use `make test-go` or `go test ./cmd/... ./internal/...`, not `go test ./...`.
18. Remember embedded static assets: changes to templates, CSS, or JS require rebuilding/restarting the binary to affect a running app.
19. For non-trivial work, write a concise plan under `docs/plans/YYYY-MM-DD-<feature>.md` and keep `tasks/todo.md` current.
20. Run `git diff --check` before final response for code or documentation changes.
21. Use specialist review for UI/UX, security, architecture, or high-risk multi-file changes.
22. Keep Tier 1 concise; move explanations and rationale to on-demand references.
23. For touched paths, load the matching path-scoped rule files from `AGENTS.md`.
24. When changing path-scoped rules, keep `AGENTS.md` and each rule file's `Applies to` section aligned.

## Load on demand

- Product rationale: `PRODUCT.md`.
- Visual system: `DESIGN.md`.
- User-facing behavior: `README.md`.
- Dataview syntax: `docs/dataview.md`.
