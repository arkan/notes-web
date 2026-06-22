# UI Implementation Notes

The UI is server-rendered HTML with vanilla JavaScript enhancements. `DESIGN.md` defines the visual system; this file covers implementation mechanics.

## Server-rendered structure

- Shared page shell lives in `internal/app/templates/layout.html`.
- Page templates live in `internal/app/templates/*.html` and receive maps from `Server` handlers.
- Go fragment renderers exist for Markdown normalization, Dataview, task text, notes-map, and diagnostics.
- Keep templates as the source of page structure; JS may enhance, filter, or replace server fragments.

## Embedded assets

- `ui.go` embeds `templates/*.html`, `static/style.css`, and `static/app.js`.
- `/_static/style.css` and `/_static/app.js` are served from embedded strings.
- A built binary must be rebuilt/restarted after template/CSS/JS changes.

## JavaScript responsibilities

- Command palette: fetches `/_api/palette`, filters client-side, supports mouse and keyboard selection, and renders browser-local recents only after hydrating them from current server palette items.
- Settings controls: theme, font size, density, reading focus, and palette recent clearing.
- Sidebar/panels: restore open state for tree folders and details panels.
- Modern Workbench shell: right pane toggle, mobile sidebar drawer open/close, Escape handling, and modal isolation.
- List/tag/home project filters: local filtering over already-rendered content.
- TODO controls: action menus and local filter persistence.
- Copy buttons: task IDs, code blocks, and current path.
- Dataview tables: AJAX q/filter/sort plus client-side row pagination.
- Notes maps: renders server-provided point payloads with OpenStreetMap tile images.

## Browser state keys

- `notes-web:sidebar-open` — open tree folders.
- `notes-web:panel-open` — details panel open/closed state.
- `notes-web:theme` — `auto`, light, dark, sepia modes.
- `notes-web:font-size` — reading font size.
- `notes-web:reading-focus` — reading focus mode.
- `notes-web:right-pane-open` — right context pane open/closed state.
- `notes-web:density` — compact or comfortable density.
- `notes-web:palette-recent` — browser-local command palette recents, filtered against current `/_api/palette` items before rendering.
- `notes-web:todo-filters` — TODO page filters.

Do not store secrets or server-derived credentials in localStorage.
Treat `notes-web:palette-recent` as an untrusted cache: do not render or navigate a recent unless its URL is present in the current `/_api/palette` payload.

## Keyboard and accessibility expectations

- Preserve `/` and `Ctrl/Cmd+K` command palette shortcuts unless a field has focus.
- Preserve `Ctrl/Cmd+B` reading focus toggle.
- Reading focus must keep Settings and command palette reachable, and must not leave inert mobile sidebar controls visible.
- Escape closes palette, settings modal, sidebar, and Dataview multi menus where relevant.
- Interactive table headers must be keyboard-operable and keep `aria-sort` accurate.
- Dataview multi filters must support Arrow keys, Home/End, Enter/Space, Tab close, and Escape close with focus return.
- Focus rings and selected state cannot rely on color alone.

## Notes-map specifics

- Fenced `notes-map` blocks are parsed as YAML config in `map.go`.
- Required config: `from`; defaults include `lat: latitude`, `lon: longitude`, `title: title`, `color: status`.
- Rows are Markdown notes under `from` whose frontmatter matches `where` and includes numeric coordinates.
- Server emits escaped JSON in `data-notes-map`; browser JS renders markers and popups.
- Map tiles are external OpenStreetMap images; mention privacy/network implications when exposing map behavior.

## Safe UI change workflow

- Identify the exact route and template or renderer before changing CSS.
- For cleanup, remove selectors only with evidence that templates/JS/tests no longer reference them or that cascade/computed behavior proves the selector is dead.
- If JS replaces server HTML, preserve data attributes used for reinitialization.
- If CSS changes affect embedded assets, verify against a rebuilt/restarted binary or Playwright server.
- Use `DESIGN.md` for visual decisions instead of introducing new tokens ad hoc.
- Comfortable density must preserve or increase spacing on already-spacious key surfaces.
- Dataview diagnostics/code/table surfaces must contain overflow internally instead of widening the page.
- Add Playwright coverage for browser-visible keyboard, AJAX, or stateful behavior.

## Anti-patterns

- Runtime frameworks, bundlers, and runtime npm dependencies.
- Client-only Dataview filtering as the source of truth.
- Duplicated desktop/mobile DOM for the same homepage block.
- Decorative motion, gradients, glassmorphism, or generic dashboard chrome.
