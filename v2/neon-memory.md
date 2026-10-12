# bd-neon memory

## Purpose

Session and observations from **machine** `memory.db` via `fabrik mem`. Not tied to a single repo; optional filter to current project.

## What it runs

```text
fabrik mem context --json
fabrik mem search <query> --json
fabrik mem get <id> --json
```

When the board has a repo selected, UI may pass `--project <abs path>` on search only (user setting). Default context is machine-wide like Engram.

When `~/.fabrik/listen.json` exists, subscribe to SSE topic `memory` and refetch. Fallback watcher: `~/.fabrik/memory.db` mtime.
