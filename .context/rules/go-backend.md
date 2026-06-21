# Rule: Go backend

Rules for the Notes Web Go server and vault logic.

## Applies to

- `cmd/notes-web/**`
- `internal/app/**/*.go`

## Hard rules

1. Keep the runtime a small Go binary using standard-library HTTP and templates.
2. Keep `cmd/notes-web/main.go` as the thin CLI entrypoint into `internal/app`.
3. Keep vault path resolution centralized in existing vault/server helpers.
4. Reject traversal before reading, rendering, or mutating vault files. Dot-prefixed paths are blocked for direct read/write and excluded from enumeration. Configured hidden paths are non-enumerated but direct-URL addressable. `_trash` subtree is non-enumerated and direct CRUD blocked. `_template.md` is direct-read/edit addressable, and non-enumerated when `editing.hide_templates` is true.
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
Normal vault flow: auth -> path resolution -> direct-read policy (dot/trash blocked, configured hidden/template allowed) -> read/index/render/action
Enumeration flow: auth -> path resolution -> enum-exclusion check (dot, configured hidden, trash, template) -> read/index/render/action
Edit API flow: auth -> path resolution -> editing-enabled check -> CSRF check -> path classification -> operation authorization -> read/write
```

## Load on demand

- `README.md` for public behavior and CLI/config expectations.
- `PRODUCT.md` for product intent.
- `docs/dataview.md` plus `.context/rules/dataview.md` for Dataview work.
