# Rule: UI

Rules for templates, CSS, browser JavaScript, and visual behavior.

## Applies to

- `internal/app/templates/**`
- `internal/app/static/**`
- `DESIGN.md`

## Hard rules

1. Keep browser code vanilla; do not introduce a runtime frontend framework.
2. Keep server-rendered templates and Go renderers as the source of truth for HTML.
3. Use JavaScript to enhance keyboard flow, dialogs, sidebar behavior, Dataview controls, and fragments.
4. Preserve keyboard navigation, focus visibility, command palette behavior, shortcuts, and predictable tab order.
5. Follow `DESIGN.md`: Modern Workbench graphite shell, electric-blue action thread, flat premium layers, quiet cards, visible focus.
6. Do not use cold wiki styling, generic SaaS dashboards, glassmorphism, gradients, ornamental motion, or marketing-style decoration.
7. Use electric blue only for orientation, action, selected filters, active navigation, command flow, and focus.
8. Do not use pure black or pure white in new UI surfaces.
9. Keep diagnostics clear and calm; do not turn broken links, orphans, or Dataview errors into alarm walls.
10. Keep Dataview table controls server-backed via AJAX; do not reintroduce client-only filtering/sorting as primary behavior.
11. Keep Dataview `Rows` pagination client-side over rows returned by the server.
12. Do not store Dataview filter state in URL history, localStorage, or server session unless explicitly planned.
13. Verify the exact route/template/CSS path before fixing similar-looking visual bugs.
14. Remember templates, CSS, and JS are embedded; rebuild and restart to verify a running binary.
15. For visual or interaction polish, use a UI/design review rather than flattening intentional design details.
16. Modern Workbench CSS cleanup must be evidence-backed: verify template/JS/test references before deleting selectors, and keep risky legacy token removal for a dedicated visual QA pass.
17. Local UI preferences stay browser-local. Palette recents are browser-local history, but only URL-backed items still present in the current `/_api/palette` payload may render or navigate; treat `localStorage` as untrusted cache.
18. Density controls must not make already-spacious key surfaces smaller; “comfortable” must preserve or increase reading/card spacing.
19. Dataview diagnostics, code blocks, tables, and controls must not cause page-level horizontal overflow on mobile; use contained scrolling/wrapping.

## Patterns

```text
Design north star: Modern Workbench graphite shell, calm compact density, keyboard-native flow.
```

```text
Dataview controls: browser event -> note-local AJAX request -> server HTML fragment -> replace table wrap.
```

```text
Palette recents: localStorage cache -> filter against current /_api/palette URL-backed items -> render safe current items only.
```

## Load on demand

- `DESIGN.md` for visual system details.
- `PRODUCT.md` for product personality and accessibility principles.
- `internal/app/templates/layout.html` for shared app shell behavior.
- `.context/tier2/ui-implementation.md` for Modern Workbench shell, preferences, and palette details.
