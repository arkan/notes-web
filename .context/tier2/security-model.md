# Security Model

Notes Web assumes a trusted local Markdown vault and a private network deployment. It is not designed as a public multi-user publishing platform.

## Trust boundaries

- Trusted: the local vault owner and vault-authored Markdown/HTML.
- Untrusted: URL paths, query parameters, HTTP headers, app-generated labels/errors, browser-sent AJAX params, and arbitrary file names/frontmatter values.
- Semi-trusted: `.notes-web.yaml`; it comes from the vault owner but still must not bypass path/hidden rules.

## Authentication and exposure

- Basic Auth is optional; empty `-user` disables auth.
- When auth is enabled, `Server.ServeHTTP` checks it before static, internal, and vault routes.
- Username/password comparison uses constant-time comparison.
- TLS is intentionally out of app scope; use Tailscale or a reverse proxy for HTTPS.
- Binding to `0.0.0.0` without auth is not safe for shared networks.

## Vault path safety

- All vault URL paths must pass through `Vault.ResolveURLPath` or an equivalent root-boundary helper.
- `ResolveURLPath` URL-decodes, joins under `Vault.Root`, canonicalizes, and rejects paths escaping the root.
- `ReadNote` repeats the escape check for rel/abs note reads.
- Never add a route that accepts a free `path` query parameter to read vault files unless it goes through the same model and has a plan.

## Hidden path semantics

- Hidden means any dot-prefixed path segment or configured `hidden` prefix.
- Hidden paths should be absent from `MarkdownFiles`, folder listings, normal note/file routes, favorites, quick-jump links, diagnostics, and AJAX actions.
- If a new feature lists vault content, test both dot-hidden and configured-hidden paths.

## Markdown and HTML output

- Goldmark raw HTML is enabled because the vault is trusted local content.
- App-generated HTML is never trusted; use Go HTML escaping helpers for text, attributes, URLs, labels, and error messages.
- Frontmatter display escapes keys and values.
- Wikilinks, Dataview cells, Dataview errors, notes-map payloads, and task metadata must escape user/vault-derived text unless intentionally rendering a trusted link structure.

## Internal actions and AJAX

- `renderDataviewTable` is note-local: `/Note.md?action=renderDataviewTable&table=N`.
- It runs inside `Server.path`, after auth, path resolution, and hidden checks.
- It rejects non-GET, non-Markdown files, missing/invalid table indexes, undeclared filters, invalid sort params, and any `path` query key.
- Invalid action responses should be HTML `dataview-error` fragments, not plaintext pages, so browser replacement stays predictable.

## File serving

- Non-Markdown files are served only after normal path resolution and hidden checks.
- MIME type is set from extension when known, then `http.ServeFile` handles response mechanics.
- Do not add directory traversal shortcuts for assets outside the vault or embedded static files.

## Browser-side privacy gotchas

- UI preferences, sidebar state, panel state, and TODO filters use `localStorage`; do not store secrets there.
- `notes-map` renders OpenStreetMap tile images from `https://tile.openstreetmap.org/...`; tile requests can reveal viewed map areas to that external service.
- The command palette fetches `/_api/palette` from the same origin and contains note titles/paths/tags; keep it behind the same auth boundary.

## Security review checklist

- Does this change read, list, render, mutate, or expose a vault path?
- Does it honor auth before work and hidden checks before content exposure?
- Are all query-derived values validated and escaped?
- Could a public URL, AJAX param, or config value bypass the vault root?
- Does a new browser feature persist sensitive state or call an external service?
