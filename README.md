# Notes Web

Notes Web is a small, self-contained Go web server for browsing a Markdown/Obsidian-style vault from a browser. It renders Markdown files as HTML, preserves filesystem-style URLs, resolves Obsidian wikilinks, provides full-text search, and exposes a clean publish-style reading interface.

The application is designed for private use on a local network or over Tailscale. It does not publish anything to the public Internet by itself.

## Features

- **Filesystem URLs**: `/Areas/Daily%20Briefings/2026-05-22-briefing.md` maps directly to the matching file in the vault.
- **Markdown rendering** powered by Goldmark:
  - GitHub-flavored tables
  - task checkboxes
  - footnotes
  - heading anchors
  - syntax highlighting
  - raw HTML support for trusted local notes
- **Obsidian-style wikilinks**:
  - `[[Note]]`
  - `[[Note|Alias]]`
  - `[[Note#Heading]]`
  - ambiguous links show a chooser page
  - missing links show a dedicated not-found page
- **Frontmatter display** with `title:` used as the page title when present.
- **Sidebar vault tree** with collapsible folders and state persisted in `localStorage`.
- **Home page** with favorites, latest daily note, and recently modified notes.
- **Backlinks** computed on demand.
- **Full-text search** using `ripgrep` when available, with a native Go fallback.
- **TODO task IDs**: comments such as `<!-- tid:1c496356 -->` are rendered as copyable task ID badges.
- **Inline non-Markdown files** served with the browser's native MIME handling.
- **Optional Basic Auth** for private access over Tailscale/LAN.

## Requirements

- Go 1.25+ recommended.
- `ripgrep` (`rg`) is optional but recommended for faster search.
- A Markdown vault directory, for example `/home/arkan/hermes`.

## Build

```bash
go build -o ./bin/notes-web ./cmd/notes-web
```

## Run locally

```bash
./bin/notes-web \
  --vault /home/arkan/hermes \
  --host 127.0.0.1 \
  --port 8080
```

Open:

```text
http://127.0.0.1:8080/
```

## Run over Tailscale or LAN with Basic Auth

```bash
export NOTES_WEB_PASSWORD='change-this-password'

./bin/notes-web \
  --vault /home/arkan/hermes \
  --host 0.0.0.0 \
  --port 8080 \
  --user arkan \
  --password-env NOTES_WEB_PASSWORD
```

Then open:

```text
http://<tailscale-hostname-or-ip>:8080/
```

Example note URL:

```text
http://<tailscale-hostname-or-ip>:8080/Areas/Daily%20Briefings/2026-05-22-briefing.md
```

## Configuration

Notes Web can read an optional `.notes-web.yaml` file at the vault root.

Example:

```yaml
favorites:
  - Areas/Daily Briefings
  - Areas/TODO.md
daily_glob: "Areas/Daily Briefings/*-briefing.md"
```

### `favorites`

A list of vault-relative files or folders displayed on the home page and in the sidebar.

### `daily_glob`

A glob used to find the latest daily note shown on the home page.

## Command-line options

```text
-host string
      HTTP bind host (default "127.0.0.1")
-password-env string
      environment variable containing the Basic Auth password
-port int
      HTTP bind port (default 8080)
-user string
      Basic Auth username. If empty, authentication is disabled.
-vault string
      vault path (default "/home/arkan/hermes")
```

## Security model

Notes Web is intended for trusted private networks, typically Tailscale.

Important details:

- Basic Auth is optional but recommended when binding to `0.0.0.0`.
- TLS is intentionally not handled by the app; use Tailscale or a reverse proxy if HTTPS is required.
- The server rejects path traversal attempts that escape the vault root.
- Raw HTML rendering is enabled because the vault is assumed to be trusted local content.
- Do not expose this service directly to the public Internet without a proper reverse proxy, TLS, and authentication.

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
go build -o ./bin/notes-web ./cmd/notes-web
```

Run a local smoke test manually:

```bash
./bin/notes-web --vault /home/arkan/hermes --host 127.0.0.1 --port 18080
```

Then visit:

```text
http://127.0.0.1:18080/
http://127.0.0.1:18080/_search?q=briefing
```

## Project layout

```text
cmd/notes-web/main.go      CLI entry point
internal/app/app.go        vault access, rendering, search, HTTP handlers
internal/app/ui.go         embedded templates, CSS, and browser JavaScript
internal/app/app_test.go   unit and integration-style tests
```

## License

Private project. No license is currently granted.
