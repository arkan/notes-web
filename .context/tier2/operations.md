# Operations and Tooling Notes

Notes Web is optimized for a small local/self-hosted deployment: one Go binary, optional Basic Auth, optional Nix packaging, and a vault on disk.

## Build and run

```bash
make build
./bin/notes-web --vault /path/to/vault --host 127.0.0.1 --port 8080
```

Development alternatives:

```bash
go run ./cmd/notes-web --vault ./testdata/e2e-vault --host 127.0.0.1 --port 18081
make run
```

`make run` uses the default CLI flags, including the default vault path, so prefer explicit flags for smoke tests.

## Live verification gotchas

- Templates, CSS, and JS are embedded in the Go binary; rebuild/restart before judging a live `./bin/notes-web` process.
- Playwright starts `go run`, so it reflects source changes at server start.
- If a browser fix appears not to work, confirm which process and binary are serving the URL before changing adjacent code.

## Dependencies

- Go module target is declared in `go.mod`.
- Runtime JS has no npm dependencies.
- npm dependencies are dev/test only: ESLint, TypeScript support, and Playwright.
- `make deps` runs `npm ci` and installs the Chromium browser.
- `make deps-ci` also installs system dependencies for headless Chromium.

## Testing commands

```bash
make test-go
make lint
make test-e2e
make test
git diff --check
```

After `npm ci`, do not use plain `go test ./...`; use `make test-go` because `node_modules` can contain Go packages.

## Configuration operations

- Optional vault config is `.notes-web.yaml` at the vault root.
- `daily_glob` drives daily briefings/calendar/selected-day summary.
- `daily_notes_glob` drives real daily note previews on the homepage today block.
- `hidden` hides paths across normal pages, listings, diagnostics, and actions.
- `hidden_blocks` is legacy-compatible for selected UI blocks, but homepage blocks have typed `homepage.blocks.<id>.visible` flags.
- Homepage block IDs are explicit: `today`, `calendar`, `todos`, `active_projects`, `selected_day`, `quick_jump`, `recent_notes`, `diagnostics`.

## Nix and NixOS

- `flake.nix` exposes packages/apps/devShells for Linux and Darwin systems.
- The package uses `buildGoModule`; `vendorHash = null` is a first-build placeholder that may need replacement for reproducible Nix builds.
- `nix/nixos-module.nix` supports system and systemd user service modes.
- Service assertions enforce absolute vault paths and paired auth user/password env config.
- System service mode can create a dedicated user and uses read-only vault paths.
- Use `environmentFile` plus `auth.passwordEnv` for Basic Auth secrets.

## Real vault boundary

- `/home/arkan/hermes` is the user's real vault, not test data.
- Do not edit it without explicit fresh confirmation of the intended diff.
- Use `testdata/e2e-vault` for tests and reproducible browser scenarios.

## Release/user docs alignment

- README command examples should match `Makefile`, CLI flags, Nix module options, and config defaults.
- If public behavior changes, update README or the relevant docs in the same change.
- If Dataview syntax changes, update `docs/dataview.md` and fixture coverage.
