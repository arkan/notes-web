# Glossary

Short project terminology for Notes Web agents.

## Product terms

- **Notes Web**: The Go web app in this repository.
- **Vault**: A trusted local Markdown / Obsidian-style directory served by Notes Web.
- **Hermes**: The user's real vault at `/home/arkan/hermes`; external user data, not test data.
- **Fixture vault**: `testdata/e2e-vault`, the canonical vault for Playwright tests.
- **Note**: A Markdown file in the vault rendered as a web page.
- **Folder page**: A route that lists vault directories and notes.
- **Dashboard**: Home/TODO/project/calendar surfaces built from vault metadata.
- **Diagnostics**: Maintenance pages for broken links, orphans, and Dataview errors.

## Markdown and rendering terms

- **Renderer**: The Go Markdown renderer centered in `internal/app/render_search.go`.
- **Preprocess**: The ordered transformation before Goldmark rendering.
- **GFM**: GitHub Flavored Markdown tables and task lists.
- **TOC**: Generated table of contents for headings.
- **Callout**: Obsidian-style blockquote converted before Markdown render.
- **Mermaid block**: Fenced Mermaid code rendered as `<pre class="mermaid">`.
- **Wikilink**: Obsidian-style `[[target]]`, `[[target|alias]]`, or heading link.
- **Raw HTML**: Trusted vault-authored HTML that Goldmark passes through.

## Dataview terms

- **Dataview block**: Fenced `dataview` code block supported server-side.
- **TABLE/LIST/TASK/CALENDAR**: Supported Dataview query types.
- **FILTER**: TABLE-only dropdown clause: `FILTER <field> [DEFAULT ...] [MODE single|multi] [CLEARABLE]`.
- **Flatten**: Dataview row expansion before grouping and filtering.
- **Group by**: Dataview aggregation step after flattening.
- **Table AJAX action**: Note-local `?action=renderDataviewTable&table=N` fragment render.

## Implementation terms

- **VaultIndex**: In-memory index built from vault files and metadata.
- **NoteMeta**: Parsed note metadata used by search, dashboards, links, and Dataview.
- **Hidden path**: File or directory excluded from normal pages, actions, and diagnostics.
- **Embedded asset**: Template, CSS, or JS included in the Go binary with `//go:embed`.
- **Command palette**: Keyboard-first browser navigation and action UI.
- **Task ID**: Copyable task marker encoded as `<!-- tid:... -->` in Markdown.

## Context method terms

- **Tier 1**: Tiny always/path-loaded working-memory files: `AGENTS.md`, `.context/ai-rules.md`, `.context/glossary.md`, and `.context/rules/*.md`.
- **Tier 2**: On-demand deeper references such as `.context/tier2/*`, `PRODUCT.md`, `DESIGN.md`, `README.md`, and `docs/dataview.md`.
- **Path-scoped rule**: A short rule file loaded only when its glob paths are touched.
