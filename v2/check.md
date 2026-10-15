# fabrik check

## Purpose

Report repo store health and machine memory.

## Invocation

```text
fabrik [-C repo] check [--json]
fabrik mem check [--json]
```

## JSON result (repo)

```json
{
  "dolt": "ok",
  "dolt_path": "/src/app/.fabrik/dolt",
  "graph": "ok",
  "graph_path": "/src/app/.fabrik/graph.db",
  "schema_dolt": 1,
  "schema_graph": 1,
  "issues": 4,
  "graph_files": 40
}
```

When Dolt is missing, `"dolt": "missing"`, counts are zero, exit 1. Graph missing is `"graph": "missing"` but exit 0 if Dolt is ok.

## JSON result (mem check)

```json
{
  "memory": "ok",
  "memory_path": "/Users/me/.fabrik/memory.db",
  "schema_memory": 1,
  "observations": 42,
  "sessions": 3,
  "projects": ["/src/app", "/other/repo"]
}
```

Missing `memory.db` → `"memory": "missing"`, exit 0 (lazy create on first mem write).

## Text result

```text
dolt ok  schema 1  issues 4
graph ok  graph_files 40
memory ok  observations 42  projects 2
```

## bd-neon

Footer may call `fabrik check --json` and `fabrik mem check --json`.
