#!/usr/bin/env bash
# Builds the static, offline-capable (PWA) version of the web UI into a
# directory that any static host can serve, e.g. GitHub Pages:
#
#   scripts/build-site.sh [outdir]     # default: dist
#
# It's the same UI that `chordpro serve` embeds, but songs are rendered in
# the browser by the Go renderers compiled to WebAssembly, and a service
# worker caches the app so it works with no network. Preview locally with
#
#   python3 -m http.server -d dist 8000
#
# (service workers require https or localhost).
set -euo pipefail

cd "$(dirname "$0")/.."
out="${1:-dist}"
static=internal/server/static

rm -rf "$out"
mkdir -p "$out/static"
cp "$static"/*.css "$static"/*.js "$static"/*.png "$static"/*.webmanifest "$out/static/"
sed 's/<meta name="chordpro-mode" content="server">/<meta name="chordpro-mode" content="static">/' \
  "$static/index.html" > "$out/index.html"
grep -q 'content="static"' "$out/index.html" || { echo "build-site: failed to set static mode" >&2; exit 1; }

GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$out/static/chordpro.wasm" ./cmd/chordpro-wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/static/"

# Version the service worker by the build's content, so any change ships a
# new cache and an unchanged rebuild doesn't force a re-download.
version="$(cd "$out" && find . -type f | LC_ALL=C sort | xargs shasum | shasum | cut -c1-12)"
sed "s/__VERSION__/$version/" site/sw.js > "$out/sw.js"

echo "built $out (version $version)"
