# Tier 2 Context Index

OpenCode-compatible on-demand context for Notes Web. Load Tier 1 first, then load only the Tier 2 files that match the current problem.

## Existing canonical references

- `PRODUCT.md` — product purpose, audience, anti-references, accessibility principles.
- `DESIGN.md` — visual system, tokens, component feel, UI anti-patterns.
- `README.md` — user-facing install, config, commands, security, development notes.
- `docs/dataview.md` — user-facing Dataview syntax and behavior.

## Deep context files

- `.context/tier2/architecture.md` — request flow, vault/index model, rendering pipeline, subsystem map.
- `.context/tier2/security-model.md` — trusted-vault threat model, auth, path safety, hidden paths, raw HTML, AJAX actions.
- `.context/tier2/testing-strategy.md` — test placement, fixture vault strategy, validation matrix, command gotchas.
- `.context/tier2/dataview-subsystem.md` — parser/evaluator/render/AJAX pipeline and extension hazards.
- `.context/tier2/ui-implementation.md` — templates, embedded assets, vanilla JS state, keyboard interactions, notes-map.
- `.context/tier2/operations.md` — build/run/Nix/service/config gotchas and live-binary verification.

## Loading shortcuts

- Backend route, vault, rendering, index, or config work: load `architecture.md`, then `security-model.md` if paths/auth/output are involved.
- Dataview work: load `dataview-subsystem.md`, `docs/dataview.md`, and `testing-strategy.md`.
- Template/CSS/JS work: load `ui-implementation.md`, `DESIGN.md`, and `testing-strategy.md`.
- Tests, fixtures, CI, Playwright, Makefile, or npm tooling: load `testing-strategy.md` and `operations.md`.
- Packaging, local runs, NixOS service, or live debugging: load `operations.md` and `README.md`.

## Maintenance rules for Tier 2

- Keep durable knowledge here; keep active task state in `tasks/todo.md`.
- Do not duplicate Tier 1 hard rules unless the deeper explanation needs them locally.
- Prefer links to canonical docs over copying long product, design, or syntax tables.
- Do not store secrets, real vault content, or unconfirmed personal workflow assumptions.
