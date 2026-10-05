# Fabrik

One cross-platform AI workflow for **Cursor**, **OpenCode**, **Pi**, and **Antigravity**: grill → land in **Beads** → build with backpressure → remember in **Engram** → map with **Graphify**.

Models stay in each app's picker. All orchestration and hub content are handled by the standalone **Go CLI**.

## Quick start

```bash
npm i -g @ur-wesley/fabrik
# or: bun add -g @ur-wesley/fabrik
# or: go install github.com/ur-wesley/fabrik/cli/cmd/fabrik@latest
cd /path/to/your-repo && fabrik setup   # inits repo first; prompts for apps; --dry-run to preview
```

No postinstall script — `npm`/`bun` install the prebuilt binary for your
platform via `optionalDependencies` (`fabrik` on `PATH` comes from `bin/fabrik.js`).

Or download `fabrik-windows-amd64.exe` / `fabrik-linux-amd64` / `fabrik-linux-arm64` /
`fabrik-darwin-amd64` / `fabrik-darwin-arm64` from the GitHub release.
Version lives in `package.json` (mirrored to `cli/package.json` + `npm/*/package.json`
via `npm run versions:sync`); releases are tagged `v<version>`.

```bash
fabrik setup [--apps cursor,pi,antigravity,opencode] [--repo PATH]  # prompts for apps when omitted
fabrik init [PATH] [--apps ...]          # per-repo .fabrik hub + selective app wiring
fabrik check [--json]                   # probe bd, engram, graphify, pi, uv
fabrik update-deps --deps install/deps.json  # refresh pins from GitHub/PyPI
fabrik version
```

Dev: `cd cli && go test ./...`. Release targets: win32-x64, linux-x64, linux-arm64, darwin-x64, darwin-arm64.

## Per repo (Go CLI)

```bash
fabrik init [/path/to/your-repo]      # thin .fabrik hub + selective app wiring (Cursor/OpenCode/Pi/Antigravity)
fabrik show <key>                  # hub content: workflow, readme, prompt <plan|build>, skill <name>, agent <name>
fabrik list <skills|agents|all>    # what `fabrik show` can print
fabrik run [--auto -p "goals"]     # align → plan → build
fabrik loop <plan|build> [--max N] # plan once, or build until no ready issues
fabrik migrate --prune [--dry-run] # delete legacy generated dumps (keeps config.yaml, docs/, specs/)
```

`.fabrik/` on disk holds only `config.yaml` + `docs/` + `specs/`. All other hub
content is embedded in the CLI and printed via `fabrik show` — no shell scripts or external dependencies.

## Session loop (all apps)

1. **Orient** — Engram `mem_context` / `mem_search`. Query Graphify if `graphify-out/` exists.
2. **Align** — Grill (`/grill-with-docs`). Optional `/to-prd` → `.fabrik/docs/PRD.md`.
3. **Land** — `bd create`, `bd dep add` for blockers.
4. **Build** — `bd ready` → `bd update <id> --claim` → TDD/backpressure → `bd close <id>`.
5. **Remember** — Engram `mem_save` / `mem_session_summary`.

## AFK factory (OpenCode)

```bash
fabrik run --auto -p "your goals"  # grill → plan → build, non-interactive
fabrik loop plan                   # plan wave only, then review before build
fabrik loop build --max 5          # build wave, capped at 5 iterations
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

## Files

| Path | Role |
|------|------|
| `cli/` | Go CLI (`fabrik setup|init|check|update-deps|version|show|list|run|loop|migrate`), versioned via `cli/package.json` (synced from root) |
| `bin/fabrik.js` | npm/bun `fabrik` launcher (no postinstall, resolves platform package) |
| `npm/fabrik-*/` | Per-platform packages with prebuilt binary (`optionalDependencies`) |
| `scripts/sync-versions.mjs` | Keeps root + `cli/package.json` + `npm/*/package.json` in sync |
| `install/deps.json` | Pinned versions (embedded copy in CLI + sync test) |
| `install/templates/` | Template sources embedded in the CLI (`fabrik show`, `fabrik init`) |

See [PLAYBOOK.md](PLAYBOOK.md) for the full method.

## Tests

```bash
cd cli && go test ./...
cd cli && go build ./cmd/fabrik
```
