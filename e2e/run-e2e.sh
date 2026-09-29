#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

command -v bd >/dev/null 2>&1 || { echo "bd not on PATH. Run ./install/setup.sh first." >&2; exit 1; }
command -v bun >/dev/null 2>&1 || { echo "bun not on PATH." >&2; exit 1; }

cd "$ROOT"
echo "==> Fabrik E2E"
bun test ./e2e/*.e2e.test.ts
echo "E2E passed."
