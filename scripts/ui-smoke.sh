#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VAULT="${NOTES_WEB_VAULT:-/home/arkan/hermes}"
HOST="${NOTES_WEB_HOST:-127.0.0.1}"
PORT="${NOTES_WEB_PORT:-18092}"
BASE="http://${HOST}:${PORT}"

export PATH=/home/arkan/.nix-profile/bin:$PATH
cd "$ROOT"

go build -o ./bin/notes-web ./cmd/notes-web
./bin/notes-web --vault "$VAULT" --host "$HOST" --port "$PORT" > /tmp/notes-web-ui-smoke.log 2>&1 &
pid=$!
cleanup() { kill "$pid" >/dev/null 2>&1 || true; }
trap cleanup EXIT

for _ in $(seq 1 80); do
  if curl -fsS "$BASE/" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

check() {
  local path="$1"
  local needle="$2"
  local html
  html="$(curl -fsS "$BASE$path")"
  if ! grep -Fq -- "$needle" <<<"$html"; then
    echo "Missing '$needle' in $path" >&2
    exit 1
  fi
}

check "/" "Notes Web"
check "/_search?q=tag%3Afp" "Search syntax"
check "/_tags" "Popular tags"
check "/_todo" "Task dashboard"
check "/_broken-links" "distinct targets"
check "/_orphans" "orphan notes"
check "/_static/style.css" "--surface-raised"
check "/_static/app.js" "paletteSelectedIndex"

echo "UI smoke OK: $BASE"
