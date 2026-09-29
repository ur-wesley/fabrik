#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPS_PATH="$ROOT/deps.json"

beads_tag="$(curl -fsSL "https://api.github.com/repos/gastownhall/beads/releases/latest" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')"
engram_tag="$(curl -fsSL "https://api.github.com/repos/Gentleman-Programming/engram/releases/latest" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')"
graphify_version="$(curl -fsSL "https://pypi.org/pypi/graphifyy/json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["info"]["version"])')"

beads_ver="${beads_tag#v}"
engram_ver="${engram_tag#v}"

python3 - "$DEPS_PATH" "$beads_tag" "$beads_ver" "$engram_tag" "$engram_ver" "$graphify_version" <<'PY'
import json, sys
path, beads_tag, beads_ver, engram_tag, engram_ver, graphify_version = sys.argv[1:]
data = json.load(open(path))
data["beads"]["version"] = beads_ver
data["beads"]["tag"] = beads_tag
data["beads"]["assets"] = {
    "windows_amd64": f"beads_{beads_ver}_windows_amd64.zip",
    "windows_arm64": f"beads_{beads_ver}_windows_arm64.zip",
    "linux_amd64": f"beads_{beads_ver}_linux_amd64.tar.gz",
    "linux_arm64": f"beads_{beads_ver}_linux_arm64.tar.gz",
    "darwin_amd64": f"beads_{beads_ver}_darwin_amd64.tar.gz",
    "darwin_arm64": f"beads_{beads_ver}_darwin_arm64.tar.gz",
}
data["engram"]["version"] = engram_ver
data["engram"]["tag"] = engram_tag
data["engram"]["assets"] = {
    "windows_amd64": f"engram_{engram_ver}_windows_amd64.zip",
    "windows_arm64": f"engram_{engram_ver}_windows_arm64.zip",
    "linux_amd64": f"engram_{engram_ver}_linux_amd64.tar.gz",
    "linux_arm64": f"engram_{engram_ver}_linux_arm64.tar.gz",
    "darwin_amd64": f"engram_{engram_ver}_darwin_amd64.tar.gz",
    "darwin_arm64": f"engram_{engram_ver}_darwin_arm64.tar.gz",
}
data["graphify"]["version"] = graphify_version
json.dump(data, open(path, "w"), indent=2)
print(f"Updated deps.json:\n  beads    {beads_tag}\n  engram   {engram_tag}\n  graphify {graphify_version}")
PY

echo ""
echo "Run ./install/setup.sh to install the new pins."
