# Rule: Go backend

Rules for the Notes Web Go server and vault logic.

## Applies to

- `cmd/notes-web/**`
- `internal/app/**/*.go`

## Hard rules

1. Keep the runtime a small Go binary using standard-library HTTP and templates.
2. Keep `cmd/notes-web/main.go` as the thin CLI entrypoint into `internal/app`.
3. Keep vault path resolution centralized in existing vault/server helpers.
4. Reject traversal and hidden paths before reading, rendering, or mutating vault files.
5. Run auth before internal actions, file serving, and AJAX fragments.
6. Keep `Renderer.Render` and `Renderer.preprocess` as the Markdown rendering entry path.
7. Preserve preprocess order: Dataview, notes map, callouts, Mermaid, wikilinks.
8. Keep Goldmark configured for GFM, footnotes, syntax highlighting, heading IDs, and trusted raw HTML.
9. Escape all dynamic app-generated HTML with Go escaping helpers before writing fragments.
10. Keep config defaults and `.notes-web.yaml` parsing in `config.go` unless a plan says otherwise.
11. Split growing subsystems into focused files like the Dataview parser/eval/render/action split.
12. Avoid compatibility shims; remove obsolete paths cleanly with tests.
13. For bugs, verify the exact route, template, renderer, or handler path before changing adjacent code.
14. Do not add a new endpoint that bypasses note-local routing or vault boundaries without review.

## Patterns

```text
Renderer.preprocess order: Dataview -> notes map -> callouts -> Mermaid -> wikilinks
```

```text
Normal vault flow: auth -> path resolution -> hidden check -> read/index/render/action
```

## Load on demand

- `README.md` for public behavior and CLI/config expectations.
- `PRODUCT.md` for product intent.
- `docs/dataview.md` plus `.context/rules/dataview.md` for Dataview work.
