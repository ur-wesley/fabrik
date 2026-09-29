# Fabrik

One cross-platform AI workflow for **Cursor**, **OpenCode**, and **Pi**: grill → land in **Beads** → build with backpressure → remember in **Engram** → map with **Graphify**.

Pi extension + OpenCode plugin + shell orchestrators. Models stay in each app's picker.

## Quick start

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
| `install/setup.ps1` / `setup.sh` | Machine install (bd, engram, graphify, MCP) |
| `install/init.ps1` / `init.sh` | Per-repo `bd init` + AGENTS.md |
| `install/deps.json` | Pinned versions |
| `install/update-deps.*` | Refresh pins from GitHub/PyPI |
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
```

E2E uses [`e2e/fixtures/sample-app`](e2e/fixtures/sample-app). See [e2e/README.md](e2e/README.md).
