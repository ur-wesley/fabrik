# fabrik setup

## Purpose

Install Fabrik on the machine, create **machine memory** (`~/.fabrik/memory.db`), register **Fabrik MCP** for selected apps, then [init.md](init.md) the repo.

Replaces `engram setup <app>` — memory tools come from Fabrik MCP, not Engram.

## Invocation

```text
fabrik setup [--apps cursor,pi,antigravity,opencode] [--repo path]
```

## Machine memory

On first setup:

1. Ensure `~/.fabrik/` exists (or `FABRIK_MEMORY_DIR`).
2. Apply [MEMORY.sql](MEMORY.sql) to `memory.db` if missing.

Memory is **not** created per `fabrik init`.

## MCP registration

For each selected app, write a server entry:

```json
{
  "fabrik": {
    "command": "fabrik",
    "args": ["mcp"]
  }
}
```

Exact file shape follows each app’s MCP config format. Remove Engram server entries for the same apps when present.

Optional env:

- `FABRIK_MCP_TOOLS` — comma list or `all` ([mcp.md](mcp.md)).

## Stored rows

Runs `init` on `--repo` (default cwd). Creates `.fabrik/dolt/` and `.fabrik/graph.db`.

Does not download `bd`, `engram`, or `graphify`.

## Text result

```text
memory /Users/me/.fabrik/memory.db
mcp registered cursor, opencode
dolt /src/app/.fabrik/dolt
graph /src/app/.fabrik/graph.db
```

## bd-neon

Assumes `fabrik` on PATH. No MCP.
