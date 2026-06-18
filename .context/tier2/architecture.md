# Architecture Notes

Notes Web is a small Go web server, not a SPA. The core decision is to keep the vault filesystem as the model, server-render HTML as the source of truth, and use vanilla JavaScript only for keyboard flow and fragment updates.

## Main runtime flow

1. `cmd/notes-web/main.go` calls `app.Main(args)`.
2. `app.Main` parses flags, creates `Vault`, reads the Basic Auth password env when configured, creates `Server`, then starts `http.ListenAndServe`.
3. `NewVault` canonicalizes the vault root and rejects non-directories.
4. `NewServer` wires `Renderer`, `Searcher`, auth credentials, and embedded Go templates.
5. `Server.ServeHTTP` runs auth first, serves `/_static/*`, dispatches internal routes, then falls back to vault path handling.

## Route families

- `/` — homepage dashboard from `Vault.BuildDashboardFor` and `HomepageView`.
- `/_search` — search page using `Searcher`; empty query shows recent notes.
- `/_api/palette` — JSON command-palette index built from favorites, notes, and tags.
- `/_tags`, `/_tags/<tag>` — tag index/detail from `VaultIndex.Tags`.
- `/_todo` — task board from Markdown task lines.
- `/_broken-links`, `/_orphans`, `/_dataview` — diagnostics over the index/scanners.
- `/_resolve`, `/_missing` — wikilink ambiguity/missing-note pages.
- Vault paths — folder listings, Markdown note pages, or non-Markdown files.

## Vault path model

- `Vault.Root` is the absolute root boundary.
- `ResolveURLPath` URL-decodes, joins under the root, canonicalizes, and rejects traversal.
- `ReadNote` repeats the root-boundary check for direct rel/abs reads.
- `HiddenRel` hides dot paths and configured `hidden` prefixes.
- Folder listings, `MarkdownFiles`, favorites, quick-jump links, normal pages, and diagnostics should all respect hidden paths.

## Index and metadata model

- `VaultIndex` is built by `BuildIndex` from `MarkdownFiles`.
- Cache key is `rel path + modtime + size` for each Markdown file; there is no filesystem watcher.
- `NoteMeta` carries rel path, URL, title, tags, frontmatter, outgoing wikilinks, outgoing occurrences, and mod time.
- `BuildIndex` also builds tag buckets and Dataview inlinks.
- Dashboards, tag pages, diagnostics, Dataview, and palette data should reuse the index rather than walking ad hoc unless they need note bodies.

## Markdown rendering pipeline

`Renderer.Render(note)` is the entry path:

1. `preprocess(note.Body)`
2. Goldmark render with GFM, footnotes, syntax highlighting, heading IDs, and raw HTML enabled.
3. Frontmatter panel rendering.
4. `normalizeRenderedHTML` decorations.

Preprocess order is intentionally: Dataview -> notes-map -> callouts -> Mermaid -> wikilinks. Callout preprocessing is currently a no-op; callout decoration happens after Goldmark.

## Generated HTML ownership

- Go templates and Go renderers own primary HTML.
- `internal/app/templates/*.html` define pages and shared layout.
- `internal/app/render_search.go`, `dataview*.go`, and `map.go` generate selected fragments.
- JavaScript replaces Dataview fragments but does not become the canonical renderer.
- App-generated values must be escaped before becoming HTML; trusted vault raw HTML is the exception.

## Static asset model

- `internal/app/ui.go` embeds templates, CSS, and JS with `//go:embed`.
- Runtime static routes are limited to `/_static/style.css` and `/_static/app.js`.
- A running binary will not see template/CSS/JS edits until rebuilt/restarted; Playwright `go run` does see fresh source on each server start.

## Major source map

- `app.go` — server, routes, vault basics, folders, files, CLI.
- `config.go` — `.notes-web.yaml`, homepage block config, favorites, quick-jump resolution.
- `dashboard.go` — homepage data, task board, broken/orphan diagnostics.
- `index.go` — `VaultIndex`, tags, outgoing/incoming link data.
- `links.go` — wikilink parsing and forward/backlink contexts.
- `render_search.go` — Markdown renderer, HTML normalization, search.
- `map.go` — fenced `notes-map` blocks and map payloads.
- `dataview*.go` — Dataview parser, evaluator, renderer, AJAX action, scanner, filters.
- `ui.go`, `templates/`, `static/` — embedded UI.

## Extension seams to respect

- Add new internal pages through `ServeHTTP` only after deciding whether they are public app routes or vault-local actions.
- Add vault-derived features through `VaultIndex` when possible; document when a body walk is required.
- Keep config as typed structs and explicit defaults; avoid generic maps for primary behavior.
- When adding note-local AJAX, route it through normal auth/path/hidden checks before action handling.
