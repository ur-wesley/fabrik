# Fabrik

One cross-platform AI workflow for **Cursor**, **OpenCode**, and **Pi**: grill → land in **Beads** → build with backpressure → remember in **Engram** → map with **Graphify**.

Pi extension + OpenCode plugin + shell orchestrators. Models stay in each app's picker.

## Quick start (Go CLI, recommended)

```bash
go install github.com/ur-wesley/fabrik/cli/cmd/fabrik@latest
fabrik setup --repo /path/to/your-repo   # interactive; --yes for CI, --dry-run to preview
```

Or download `fabrik-windows-amd64.exe` / `fabrik-linux-amd64` / `fabrik-darwin-arm64`
from the GitHub release. Version lives in `cli/package.json`; releases are tagged `v<version>`.

```bash
fabrik setup [--skip-tool-install] [--skip-engram-setup] [--skip-pi-packages] [--repo PATH]
fabrik init [PATH] [--skip-checks]   # per-repo .fabrik hub + 3-app wiring
fabrik check [--json]                # probe bd, engram, graphify, bun, pi, uv
fabrik update-deps --deps install/deps.json  # refresh pins from GitHub/PyPI
fabrik version
```

Dev: `cd cli && go test ./...`. Release targets: windows/amd64, linux/amd64, darwin/arm64.

## Quick start (shell scripts)

Shell scripts in `install/` remain as fallback:

### Machine (once)

**Windows:**
```powershell
cd D:\projects\fabrik
.\install\setup.ps1
```

**macOS / Linux:**
```bash
cd /path/to/fabrik
chmod +x install/setup.sh install/init.sh
./install/setup.sh
```

Restart Cursor, OpenCode, and Pi.

### Per repo

**Windows:**
```powershell
.\install\init.ps1 D:\path\to\your-repo
```

**macOS / Linux:**
```bash
./install/init.sh /path/to/your-repo
```

Or from inside a repo with Fabrik checked in:
```bash
./.fabrik/setup.sh    # or setup.ps1 on Windows
```

## Session loop (all apps)

1. **Orient** — Engram `mem_context` / `mem_search`. Query Graphify if `graphify-out/` exists.
2. **Align** — Grill (`/grill-with-docs`). Optional `/to-prd` → `.fabrik/docs/PRD.md`.
3. **Land** — `bd create`, `bd dep add` for blockers.
4. **Build** — `bd ready` → `bd update <id> --claim` → TDD/backpressure → `bd close <id>`.
5. **Remember** — Engram `mem_save` / `mem_session_summary`.

## AFK factory (OpenCode)

```powershell
.\.fabrik\fabrik.ps1          # Windows: grill → plan → build
./.fabrik/fabrik.sh           # Unix
```

## Stack

Pinned in [`install/deps.json`](install/deps.json):

| Layer | Tool |
|-------|------|
| Issues | [Beads](https://github.com/gastownhall/beads) (`bd`) |
| Memory | [Engram](https://github.com/Gentleman-Programming/engram) |
| Code map | [Graphify](https://github.com/safishamsi/graphify) (`graphifyy`) |
| Pi MCP | `@piarium/pi-mcp-adapter` |

Beads owns tasks. Engram owns memory. Do not use `bd remember` for the same facts.

## Pi package

```bash
pi install git:https://github.com/ur-wesley/fabrik.git
```

| Command / Tool | Description |
|----------------|-------------|
| `/fabrik-init` | `bd init` + `.fabrik/config.yaml` |
| `/fabrik-status` | Beads counts + RTK/model info |
| `/fabrik-plan` | Land PRD gaps as `bd create` issues |
| `fabrik_next_task` | `bd ready --claim` |
| `fabrik_complete_task` | `bd close` + optional commit |

## Files

| Path | Role |
|------|------|
| `cli/` | Go setup CLI (`fabrik setup|init|check|update-deps|version`), versioned via `cli/package.json` |
| `install/setup.ps1` / `setup.sh` | Machine install fallback (bd, engram, graphify, MCP) |
| `install/init.ps1` / `init.sh` | Per-repo fallback (`bd init` + AGENTS.md) |
| `install/deps.json` | Pinned versions (embedded copy in CLI + sync test) |
| `install/update-deps.*` | Refresh pins fallback (or `fabrik update-deps`) |
| `.fabrik/fabrik.ps1` / `fabrik.sh` | Master orchestrator |
| `.fabrik/loop.ps1` / `loop.sh` | Plan/build OpenCode loops |
| `src/` | Pi extension (TypeScript) |
| `.opencode/plugins/fabrik.ts` | Wave runner + batched commits |

See [PLAYBOOK.md](PLAYBOOK.md) for the full method.

## Tests

```bash
bun run test          # unit
bun run test:e2e      # full workflow (needs bd on PATH)
bun run test:all
bun run cli:test      # Go CLI (cd cli && go test ./...)
```

E2E uses [`e2e/fixtures/sample-app`](e2e/fixtures/sample-app). See [e2e/README.md](e2e/README.md).
