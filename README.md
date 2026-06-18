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
- **Server-side Dataview rendering** for common Obsidian `TABLE`, `LIST`, and `TASK` queries; see [`docs/dataview.md`](docs/dataview.md).
- **Sidebar vault tree** with collapsible folders and state persisted in `localStorage`.
- **Home page** with favorites, latest daily note, and recently modified notes.
- **Backlinks** computed on demand.
- **Full-text search** using `ripgrep` when available, with a native Go fallback.
- **TODO task IDs**: comments such as `<!-- tid:1c496356 -->` are rendered as copyable task ID badges.
- **Inline non-Markdown files** served with the browser's native MIME handling.
- **Optional Basic Auth** for private access over Tailscale/LAN.

## Requirements

- A Markdown vault directory, for example `/home/arkan/hermes`.
- `ripgrep` (`rg`) is optional but recommended for faster search.
- Go 1.25+ is only required when building from source.

## Install from a prebuilt release archive

This is the preferred installation method for regular use.

Download the archive matching your platform from the
[latest GitHub Release](https://github.com/arkan/notes-web/releases/latest):

- `notes-web-<version>-linux-amd64.tar.gz`
- `notes-web-<version>-linux-arm64.tar.gz`
- `notes-web-<version>-darwin-amd64.tar.gz`
- `notes-web-<version>-darwin-arm64.tar.gz`

Example for Linux amd64:

```bash
version=v0.1.0
archive="notes-web-${version}-linux-amd64.tar.gz"
url="https://github.com/arkan/notes-web/releases/download/${version}/${archive}"
curl -L "${url}" -o notes-web.tar.gz

tar -xzf notes-web.tar.gz
install -D notes-web ~/.local/bin/notes-web
```

Run it:

```bash
notes-web \
  --vault /home/arkan/hermes \
  --host 127.0.0.1 \
  --port 8080
```

Open:

```text
http://127.0.0.1:8080/
```

Optional: verify the archive checksum with the `checksums-sha256.txt` file
attached to the same release.

## Build from source

```bash
go build -o ./bin/notes-web ./cmd/notes-web
```

## Run locally from a source checkout

```bash
./bin/notes-web \
  --vault /home/arkan/hermes \
  --host 127.0.0.1 \
  --port 8080
```

## Install/run with Nix

From a local checkout:

```bash
nix run . -- --vault /path/to/vault
nix profile install .
```

From GitHub:

```bash
nix run github:arkan/notes-web -- --vault /path/to/vault
nix profile install github:arkan/notes-web
```

### NixOS system service

```nix
{
  inputs.notes-web.url = "github:arkan/notes-web";

  outputs = { nixpkgs, notes-web, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        notes-web.nixosModules.default
        {
          services.notes-web = {
            enable = true;
            mode = "system";
            vault = "/home/alice/Notes";
            host = "127.0.0.1";
            port = 8080;

            # If the vault is in an existing user's home, usually run as that user.
            user = "alice";
            group = "users";
            createUser = false;
          };
        }
      ];
    };
  };
}
```

### NixOS systemd user service

```nix
{
  services.notes-web = {
    enable = true;
    mode = "user";
    user = "alice";
    createUser = false;
    vault = "/home/alice/Notes";
    host = "127.0.0.1";
    port = 8080;
  };
}
```

For Basic Auth in a NixOS service, set `auth.user`, `auth.passwordEnv`, and provide the secret through `environmentFile`.

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
daily_glob: "Areas/Daily Briefings/*-briefing.md"
daily_notes_glob: "Daily Notes/*/*/*.md"
folder_sort: name_asc
sidebar:
  favorites:
    visible: true
    items:
      - path: Areas/Daily Briefings
        label: Daily Briefings
      - path: _todo
        label: TODOs
      - path: Projects
        label: Projects
  explore:
    visible: false
homepage:
  order:
    - today
    - calendar
    - todos
    - active_projects
    - selected_day
    - quick_jump
    - recent_notes
    - diagnostics
  blocks:
    today:
      visible: true
    quick_jump:
      visible: true
      items:
        - label: Today
          path: /
        - label: TODO
          path: /_todo
        - label: Search
          path: /_search
        - label: Daily Briefings
          path: Areas/Daily Briefings
    todos:
      visible: true
    active_projects:
      visible: true
      limit: 20
    calendar:
      visible: true
    selected_day:
      visible: true
    recent_notes:
      visible: true
      limit: 10
    diagnostics:
      visible: true
```

### `sidebar.favorites.items`

A list of vault-relative files or folders displayed in the sidebar and command palette. Each entry must define:

- `path`: vault-relative file, folder, or internal route such as `_todo`.
- `label`: display label used in the UI.

Set `sidebar.favorites.visible` to `false` to hide favorites from the sidebar and command palette. The home page quick-jump block is configured separately with `homepage.blocks.quick_jump.items`.

### `daily_glob`

A glob pattern for daily briefing files, used by the sidebar calendar and the selected-day summary. Default: `Areas/Daily Briefings/*-briefing.md`.

### `daily_notes_glob`

A separate glob pattern for real daily note files, used by the homepage today block to render the selected date's daily note preview. Default: `Daily Notes/*/*/*.md`. This split lets you keep generated briefings in one location and hand-written daily notes in another; the homepage today block uses `daily_notes_glob` while the calendar and selected-day summary continue to use `daily_glob`.

### `folder_sort`

Default sort for folder pages. Accepted values: `name_asc` (default), `name_desc`, `modified_desc`, `modified_asc`.

### `sidebar`

Controls sidebar UI sections while keeping the underlying internal routes available.

- `sidebar.explore.visible`: set to `false` to hide the sidebar Explore section.
- `sidebar.favorites.visible`: set to `false` to hide the sidebar Favorites section and command palette favorites.
- `sidebar.favorites.items`: a list of `{path, label}` entries for favorites shown in the sidebar and command palette.

### `homepage`

Controls home page blocks while keeping the underlying internal routes available.

- `homepage.order`: optional list of block IDs for homepage ordering. Unknown IDs are ignored; missing valid blocks are appended at the end in default order. Default: `today, calendar, todos, active_projects, selected_day, quick_jump, recent_notes, diagnostics`.
- `homepage.blocks.<id>.visible`: set to `false` to hide a specific homepage block. All blocks visible by default.
- `homepage.blocks.active_projects.limit`: max active projects from the `Projects/` folder shown on homepage (default `20`). A project is active when its frontmatter has `status: active`.
- `homepage.blocks.recent_notes.limit`: max recent notes shown on homepage (default `10`).
- `homepage.blocks.quick_jump.items`: list of `{label, path}` entries for the quick-jump block. Internal routes like `/_todo` and vault paths like `Areas/...` are supported. Absent `items` uses defaults; explicit `items: []` shows no links.
- `homepage.blocks.todos.visible`: set to `false` to hide the home page TODO summary.
- `homepage.blocks.calendar.visible`: set to `false` to hide the home page calendar card. Does not affect `selected_day`.
- `homepage.blocks.selected_day.visible`: set to `false` to hide the selected day notes. Does not affect `calendar`.
- `homepage.blocks.diagnostics.visible`: set to `false` to hide the broken-links and orphan-notes diagnostic block.

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

Notes Web is licensed under the [MIT License](./LICENSE).
