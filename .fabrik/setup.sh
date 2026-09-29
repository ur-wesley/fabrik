#!/usr/bin/env bash
set -euo pipefail

echo "Fabrik per-repo setup..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INIT_SCRIPT="${REPO_ROOT}/install/init.sh"

if [[ ! -f "$INIT_SCRIPT" ]]; then
  echo "install/init.sh not found. Clone the full Fabrik repo."
  exit 1
fi

bash "$INIT_SCRIPT" "$REPO_ROOT"

if command -v npx >/dev/null 2>&1; then
  echo "Installing mattpocock/skills for OpenCode..."
  npx -y skills@latest add mattpocock/skills --agent opencode
fi

CONTEXT_FILE="${SCRIPT_DIR}/CONTEXT.md"
if [[ ! -f "$CONTEXT_FILE" ]]; then
  cat > "$CONTEXT_FILE" <<'EOF'
# Project Context & Dictionary

Domain terms and architecture rules for this project.

## Domain Language
*   **Term**: [Definition]

## Architecture Guidelines
*   Prefer deep modules over thin wrappers.
EOF
fi

mkdir -p "${SCRIPT_DIR}/styleguide"
if [[ ! -f "${SCRIPT_DIR}/styleguide/STYLEGUIDE.md" ]]; then
  echo '# Styleguide' > "${SCRIPT_DIR}/styleguide/STYLEGUIDE.md"
fi

chmod +x "${SCRIPT_DIR}/loop.sh" "${SCRIPT_DIR}/fabrik.sh" 2>/dev/null || true

echo "Repo ready. Run ./.fabrik/fabrik.sh or bd ready."
