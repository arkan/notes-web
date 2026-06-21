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

## Path classification semantics

The vault distinguishes four path categories for access policy:

### Dot-prefixed (always blocked)
- Any path segment starting with `.` (e.g. `.git`, `.obsidian`, `.hidden/Note.md`).
- Blocked for direct read/write and excluded from all enumeration (MarkdownFiles, folder listings, tree, sidebar, favorites, quick-jump, search, palette, backlinks, Dataview, diagnostics).

### Configured hidden (non-enumerated, direct-URL addressable)
- Paths listed in the `hidden:` YAML key.
- Excluded from all enumeration surfaces (MarkdownFiles, folder listings, tree, sidebar, favorites, quick-jump, search, palette, backlinks, Dataview, diagnostics).
- Accessible by direct URL (note/folder route). Shows a Hidden badge in reading context.
- Full CRUD support when editing is enabled.

### Trash subtree (non-enumerated, direct CRUD blocked)
- Paths under the configured `editing.trash_path` (default `_trash`).
- Excluded from all enumeration surfaces.
- Blocked for direct read/write via note/folder route. Only accessible through the dedicated Trash view/API.

### Template files (non-enumerated, direct-read addressable)
- Files matching the configured `editing.template_name` (default `_template.md`).
- Excluded from enumeration when `editing.hide_templates` is true (default).
- Accessible by direct URL for read and edit. Shows a Template badge in reading context.
- Not created, renamed, moved to trash, or scanned for link rewrites in v1.

### New features listing vault content

- If a new feature lists vault content, test all categories: dot-prefixed, configured-hidden, trash, and template paths.

## Markdown and HTML output

- Goldmark raw HTML is enabled because the vault is trusted local content.
- App-generated HTML is never trusted; use Go HTML escaping helpers for text, attributes, URLs, labels, and error messages.
- Frontmatter display escapes keys and values.
- Wikilinks, Dataview cells, Dataview errors, notes-map payloads, and task metadata must escape user/vault-derived text unless intentionally rendering a trusted link structure.

## Internal actions and AJAX

- `renderDataviewTable` is note-local: `/Note.md?action=renderDataviewTable&table=N`.
- It runs inside `Server.path`, after auth, path resolution, and direct-read path policy checks.
- It rejects non-GET, non-Markdown files, missing/invalid table indexes, undeclared filters, invalid sort params, and any `path` query key.
- Invalid action responses should be HTML `dataview-error` fragments, not plaintext pages, so browser replacement stays predictable.

## File serving

- Non-Markdown files are served only after normal path resolution and direct-read path policy checks.
- MIME type is set from extension when known, then `http.ServeFile` handles response mechanics.
- Do not add directory traversal shortcuts for assets outside the vault or embedded static files.

## Browser-side privacy gotchas

- UI preferences, sidebar state, panel state, and TODO filters use `localStorage`; do not store secrets there.
- `notes-map` renders OpenStreetMap tile images from `https://tile.openstreetmap.org/...`; tile requests can reveal viewed map areas to that external service.
- The command palette fetches `/_api/palette` from the same origin and contains note titles/paths/tags; keep it behind the same auth boundary.

## Security review checklist

- Does this change read, list, render, mutate, or expose a vault path?
- Does it honor auth before work and path policy checks (dot-blocked, trash, configured-hidden, template) before content exposure?
- Are all query-derived values validated and escaped?
- Could a public URL, AJAX param, or config value bypass the vault root?
- Does a new browser feature persist sensitive state or call an external service?
