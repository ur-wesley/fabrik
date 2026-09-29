#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPS_PATH="$ROOT/deps.json"
TEMPLATE="$ROOT/templates/workflow-note.md"
BIN_DIR="${HOME}/.local/bin"
GRAPHIFY_SKILL="${HOME}/.agents/skills/graphify/SKILL.md"

SKIP_TOOL_INSTALL=false
SKIP_ENGRAM_SETUP=false
SKIP_PI_PACKAGES=false
REPO_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-tool-install) SKIP_TOOL_INSTALL=true; shift ;;
    --skip-engram-setup) SKIP_ENGRAM_SETUP=true; shift ;;
    --skip-pi-packages) SKIP_PI_PACKAGES=true; shift ;;
    --repo) REPO_PATH="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

step() { echo ""; echo "==> $1"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

platform_key() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    linux)
      case "$arch" in aarch64|arm64) echo linux_arm64 ;; *) echo linux_amd64 ;; esac
      ;;
    darwin)
      case "$arch" in arm64) echo darwin_arm64 ;; *) echo darwin_amd64 ;; esac
      ;;
    *) echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac
}

ensure_bin_dir() {
  mkdir -p "$BIN_DIR"
  case ":$PATH:" in *":$BIN_DIR:"*) ;; *)
    export PATH="$BIN_DIR:$PATH"
    ;;
  esac
}

json_get() {
  python3 - "$DEPS_PATH" "$1" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
path = sys.argv[2].split('.')
cur = data
for p in path:
    cur = cur[p]
print(cur)
PY
}

install_github_archive() {
  local repo="$1" tag="$2" asset="$3" binary="$4" label="$5"
  local dest="$BIN_DIR/$binary"
  if [[ -x "$dest" ]]; then
    echo "$label already at $dest"
    return
  fi
  if [[ "$SKIP_TOOL_INSTALL" == true ]]; then
    echo "$label missing and --skip-tool-install set" >&2
    exit 1
  fi
  step "Installing $label $tag"
  local url="https://github.com/$repo/releases/download/$tag/$asset"
  local archive
  archive="$(mktemp)"
  curl -fsSL "$url" -o "$archive"
  local extract
  extract="$(mktemp -d)"
  case "$asset" in
    *.zip) unzip -q "$archive" -d "$extract" ;;
    *.tar.gz) tar -xzf "$archive" -C "$extract" ;;
    *) echo "Unknown archive type: $asset" >&2; exit 1 ;;
  esac
  local found
  found="$(find "$extract" -type f -name "$binary" | head -n 1)"
  if [[ -z "$found" ]]; then
    echo "Could not find $binary inside $asset" >&2
    exit 1
  fi
  cp "$found" "$dest"
  chmod +x "$dest"
  rm -rf "$archive" "$extract"
  echo "Installed $dest"
}

install_bd() {
  if command_exists bd; then
    echo "bd already on PATH: $(bd version 2>&1 | head -n 1)"
    return
  fi
  local key asset repo tag binary
  key="$(platform_key)"
  repo="$(json_get beads.repo)"
  tag="$(json_get beads.tag)"
  binary="$(json_get beads.binary)"
  asset="$(python3 - "$DEPS_PATH" "beads.assets.$key" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
print(data["beads"]["assets"][sys.argv[2]])
PY
"$key")"
  install_github_archive "$repo" "$tag" "$asset" "$binary" "Beads"
}

install_engram() {
  if command_exists engram; then
    echo "engram already on PATH: $(engram version 2>&1 | head -n 1)"
    return
  fi
  if [[ "$SKIP_TOOL_INSTALL" == true ]]; then
    echo "engram missing and --skip-tool-install set" >&2
    exit 1
  fi
  if command_exists go; then
    step "Installing Engram $(json_get engram.tag) via go install"
    go install "$(json_get engram.goModule)@$(json_get engram.tag)"
    local go_bin
    go_bin="$(go env GOPATH)/bin/engram"
    if [[ -x "$go_bin" ]]; then
      cp "$go_bin" "$BIN_DIR/engram"
      chmod +x "$BIN_DIR/engram"
      echo "Copied engram to $BIN_DIR"
      return
    fi
  fi
  local key asset repo tag binary
  key="$(platform_key)"
  repo="$(json_get engram.repo)"
  tag="$(json_get engram.tag)"
  binary="$(json_get engram.binary)"
  asset="$(python3 - "$DEPS_PATH" "engram.assets.$key" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
print(data["engram"]["assets"][sys.argv[2]])
PY
"$key")"
  install_github_archive "$repo" "$tag" "$asset" "$binary" "Engram"
}

install_graphify() {
  step "Installing Graphify skill"
  mkdir -p "$(dirname "$GRAPHIFY_SKILL")"
  curl -fsSL "$(json_get graphify.skillUrl)" -o "$GRAPHIFY_SKILL"
  echo "Wrote $GRAPHIFY_SKILL"
  if command_exists graphify; then echo "graphify CLI already on PATH"; return; fi
  if [[ "$SKIP_TOOL_INSTALL" == true ]]; then return; fi
  if command_exists uv; then
    step "Installing $(json_get graphify.pypi)==$(json_get graphify.version) with uv"
    uv tool install "$(json_get graphify.pypi)==$(json_get graphify.version)"
    return
  fi
  echo "uv not found; install graphify CLI manually"
}

install_pi_packages() {
  if [[ "$SKIP_PI_PACKAGES" == true ]]; then return; fi
  if ! command_exists pi; then echo "pi not on PATH; skip pi package install"; return; fi
  step "Installing Pi MCP adapter"
  pi install "$(json_get pi.mcpAdapter)"
}

install_workflow_note() {
  step "Installing personal workflow note"
  local note
  note="$(cat "$TEMPLATE")"
  mkdir -p "${HOME}/.cursor/rules" "${HOME}/.config/opencode" "${HOME}/.pi/agent"
  cat > "${HOME}/.cursor/rules/ai-workflow.mdc" <<EOF
---
description: Fabrik workflow with Beads, Engram, and Graphify
alwaysApply: true
---
$note
EOF
  echo "Wrote ${HOME}/.cursor/rules/ai-workflow.mdc"
  printf '%s\n' "$note" > "${HOME}/.config/opencode/AGENTS.md"
  echo "Wrote ${HOME}/.config/opencode/AGENTS.md"
  printf '%s\n' "$note" > "${HOME}/.pi/agent/AGENTS.md"
  echo "Wrote ${HOME}/.pi/agent/AGENTS.md"
}

setup_engram_agents() {
  if [[ "$SKIP_ENGRAM_SETUP" == true ]]; then return; fi
  if ! command_exists engram; then echo "Skipping engram setup"; return; fi
  step "Running engram setup for Cursor, OpenCode, Pi"
  engram setup cursor
  engram setup opencode
  engram setup pi
}

setup_beads_cursor() {
  if ! command_exists bd; then return; fi
  step "Running bd setup cursor"
  bd setup cursor
}

echo "Fabrik machine setup"
echo "Deps: beads $(json_get beads.tag), engram $(json_get engram.tag), graphify $(json_get graphify.version)"

ensure_bin_dir
install_bd
install_engram
install_graphify
install_pi_packages
install_workflow_note
setup_engram_agents
setup_beads_cursor

if [[ -n "$REPO_PATH" ]]; then
  step "Initializing repo: $REPO_PATH"
  bash "$ROOT/init.sh" "$REPO_PATH"
fi

echo ""
echo "Done."
echo "Restart Cursor, OpenCode, and Pi so MCP and rules reload."
echo "Per repo: ./install/init.sh /path/to/your-repo"
