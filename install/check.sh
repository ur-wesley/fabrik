#!/usr/bin/env bash
set -uo pipefail

# Fabrik tool check (mac + linux; Windows uses check.ps1).
# Exit 0 = ok, 1 = missing. Use --json for machine output.

JSON=false
if [[ "${1:-}" == "--json" ]]; then JSON=true; fi

have() { command -v "$1" >/dev/null 2>&1; }

declare -A out=()
fail=0

check() {
  if have "$1"; then out["$1"]="ok"; else out["$1"]="missing"; fail=1; fi
}

check bd
check engram
check graphify
check bun
check pi
check uv

if have bd; then out["bd_version"]="$(bd version 2>&1 | head -n 1)"; fi
if have engram; then out["engram_version"]="$(engram version 2>&1 | head -n 1)"; fi
if have graphify; then out["graphify_version"]="$(graphify --version 2>&1 | head -n 1)"; fi

if [[ "$JSON" == true ]]; then
  printf '{'
  first=true
  for k in bd engram graphify bun pi uv; do
    [[ "$first" == true ]] || printf ','
    first=false
    printf '"%s":"%s"' "$k" "${out[$k]}"
  done
  printf ',"ok":%s}\n' "$([[ $fail -eq 0 ]] && echo true || echo false)"
else
  for k in bd engram graphify bun pi uv; do
    echo "$k: ${out[$k]}"
  done
  if [[ $fail -ne 0 ]]; then
    echo ""
    echo "Missing tools. Run ./install/setup.sh (mac/linux) or .\\install\\setup.ps1 (windows)."
  fi
fi

exit $fail
