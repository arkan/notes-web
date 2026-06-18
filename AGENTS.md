# AGENTS.md

OpenCode entrypoint for AI agents working in this repository. This file is the canonical `.context` Tier 1 routing table.

## Project identity

Notes Web is a small, self-contained Go web server for browsing and maintaining a trusted Markdown / Obsidian-style vault from a browser.

It should feel like a calm, keyboard-first web workbench: fast retrieval, comfortable reading, actionable task review, and visible vault maintenance without a heavy app stack.

Primary references: `PRODUCT.md`, `DESIGN.md`, `README.md`, `docs/dataview.md`.

## Required Tier 1 context

Always load:

- `.context/ai-rules.md` — project-wide hard rules.
- `.context/glossary.md` — project terms.

Then load path-scoped `.context/rules/*.md` files for touched files:

| Touching | Load |
|---|---|
| `AGENTS.md`, `.context/**` | `.context/rules/tooling-docs.md` |
| `cmd/notes-web/**`, `internal/app/**/*.go` | `.context/rules/go-backend.md` |
| `internal/app/dataview*.go`, `docs/dataview.md`, `tests/e2e/dataview-*.spec.ts`, `testdata/e2e-vault/**/Dataview*.md` | `.context/rules/dataview.md` |
| `internal/app/static/**`, `internal/app/templates/**`, `DESIGN.md` | `.context/rules/ui.md` |
| `internal/app/*_test.go`, `tests/e2e/**`, `testdata/e2e-vault/**`, `package*.json`, `playwright.config.ts`, `Makefile` | `.context/rules/testing.md` |
| `README.md`, `PRODUCT.md`, `DESIGN.md`, `docs/**`, `Makefile`, `package*.json`, `eslint.config.js`, `tsconfig.json`, `playwright.config.ts`, `flake.nix`, `nix/**`, `.github/**` | `.context/rules/tooling-docs.md` |

## Commands

```bash
make deps       # npm ci + Playwright browser install
make deps-ci    # CI/headless browser dependencies
make test       # Go tests, lint, Playwright E2E
make test-go    # go test ./cmd/... ./internal/...
make lint       # npm run lint
make test-e2e   # Playwright against testdata/e2e-vault
make build      # build bin/notes-web
git diff --check
```

Do not use plain `go test ./...` after `npm ci`; `node_modules` may contain Go packages.

## Boundaries

- Do not commit, amend, push, or create PRs unless explicitly requested.
- Do not modify `/home/arkan/hermes` unless explicitly requested and freshly confirmed.
- Use `testdata/e2e-vault`, never the real Hermes vault, for tests.
- For non-trivial work, write a concise plan under `docs/plans/YYYY-MM-DD-<feature>.md` and keep `tasks/todo.md` current.

## Load on demand

- Tier 2 context index: `.context/tier2/README.md`.
- Product/design depth: `PRODUCT.md`, `DESIGN.md`.
- User behavior and configuration: `README.md`.
- Dataview syntax and filters: `docs/dataview.md`.
- Current work and lessons: `tasks/todo.md`, `tasks/lessons.md`.
