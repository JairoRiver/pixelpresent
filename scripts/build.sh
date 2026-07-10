#!/usr/bin/env bash
#
# Production build: compiles the Astro frontend and embeds it into a single
# static Go binary, ready to copy to the VPS. Output: bin/pixelpresent.
#
# The binary is fully static (CGO_ENABLED=0), stripped (-ldflags '-s -w') and
# reproducible (-trimpath), so it runs on the VPS with no libc/runtime deps.
#
# Cross-compiles to linux/amd64 by default: the usual VPS target, and it matches
# the dev host so the same binary also runs locally for verification. Override
# for another target, e.g. an arm64 VPS:
#
#   GOOS=linux GOARCH=arm64 scripts/build.sh
#
set -euo pipefail

# Resolve the repo root from this script's location so it works from any CWD.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
OUT="${OUT:-bin/pixelpresent}"

echo "==> Building frontend (pnpm build)"
( cd web && pnpm install --frozen-lockfile && pnpm build )

# Astro empties its outDir (internal/web/dist) on build, removing the tracked
# placeholder; restore it so the working tree stays clean and the //go:embed in
# internal/web/web.go still compiles on a checkout that hasn't built the front.
touch internal/web/dist/.gitkeep

echo "==> Compiling static Go binary (${GOOS}/${GOARCH}) -> ${OUT}"
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
  go build -trimpath -ldflags='-s -w' -o "$OUT" ./cmd/server

echo "==> Done: ${OUT} ($(du -h "$OUT" | cut -f1))"
