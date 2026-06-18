# Path-scoped rule index

Non-normative index for Notes Web Tier 1 path-scoped rules. Normative loading rules live in `AGENTS.md` and `.context/ai-rules.md`.

## Always-loaded companions

- `.context/ai-rules.md`
- `.context/glossary.md`

## Maintenance notes

- Keep each rule file focused on one job and under about 80 lines.
- Put hard rules near the top of each path-scoped rule file.
- Move rationale and long examples to Tier 2 references.
- Add a path-scoped rule only when it prevents repeated mistakes.
- Keep routing globs in `AGENTS.md` aligned with each rule file's `Applies to` section.

## Rule files

- `go-backend.md` — Go server, routing, rendering, vault model, config.
- `dataview.md` — Dataview parser/eval/render/AJAX/docs.
- `ui.md` — Templates, CSS, vanilla JS, design, accessibility.
- `testing.md` — Go tests, Playwright, fixture vault, validation commands.
- `tooling-docs.md` — README, product docs, Makefile, JS tooling, Nix, CI.

## Tier 2 references

- `.context/tier2/README.md`
- `PRODUCT.md`
- `DESIGN.md`
- `README.md`
- `docs/dataview.md`
