# Rule: tooling and docs

Rules for documentation, commands, packaging, and repo tooling.

## Applies to

- `AGENTS.md`
- `.context/**`
- `README.md`
- `PRODUCT.md`
- `DESIGN.md`
- `docs/**`
- `Makefile`
- `package*.json`
- `eslint.config.js`
- `tsconfig.json`
- `playwright.config.ts`
- `flake.nix`
- `nix/**`
- `.github/**`

## Hard rules

1. Keep project documentation in English unless explicitly requested otherwise.
2. Keep `PRODUCT.md` as the product purpose, audience, anti-references, and accessibility source.
3. Keep `DESIGN.md` as the visual system and interaction source.
4. Keep `README.md` user-facing: features, installation, config, commands, and operational notes.
5. Keep `docs/dataview.md` aligned with actual Dataview parser/eval/render behavior.
6. Do not document unsupported syntax as supported.
7. Keep NPM tooling dev/test-only; do not add runtime JS dependencies or bundling.
8. Keep Makefile commands aligned with README and CI workflows.
9. Keep Nix packaging and module changes aligned with the built Go binary and runtime flags.
10. Keep `.notes-web.yaml` docs aligned with `internal/app/config.go` defaults and validation.
11. Treat `homepage.order` and `homepage.blocks` as the configurable homepage model.
12. Preserve valid homepage block IDs: `today`, `calendar`, `todos`, `active_projects`, `selected_day`, `quick_jump`, `recent_notes`, `diagnostics`.
13. Use `daily_glob` for daily briefings / selected-day calendar data and `daily_notes_glob` for real daily note previews.
14. Update docs and tests together when commands, config, or public behavior changes.
15. Keep planning files under `docs/plans/YYYY-MM-DD-<feature>.md` for non-trivial work.

## Patterns

```text
Command docs must match Makefile targets and CI behavior.
```

```text
Config docs must match internal/app/config.go defaults and field names.
```

## Load on demand

- `README.md`
- `PRODUCT.md`
- `DESIGN.md`
- `docs/dataview.md`
- `internal/app/config.go`
