# Fabrik E2E tests

Full workflow tests against a disposable copy of [`fixtures/sample-app`](fixtures/sample-app).

## Prerequisites

- `bd` on PATH (`.\install\setup.ps1`)
- `bun` on PATH
- `git` on PATH

## Run

```powershell
.\e2e\run-e2e.ps1
```

```bash
chmod +x e2e/run-e2e.sh
./e2e/run-e2e.sh
```

Or:

```bash
bun run test:e2e
bun run test:all    # unit + e2e
```

## What is covered

| Test | Verifies |
|------|----------|
| `init` | `bd init`, `.fabrik/config.yaml`, prompts, AGENTS.md workflow block |
| `orchestrator` | `claimNextTask` / `completeTask` / `getTaskStatus` against real Beads |
| `dependencies` | `bd dep add` gates `bd ready` |
| `backpressure` | fixture `bun test`, `lint`, `build` pass |
| `install assets` | `deps.json` pins, workflow template content |

Each test uses a temp git repo — nothing writes to `fixtures/sample-app`.
