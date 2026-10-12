# fabrik graph build

## Purpose

Parse the repo and fill `graph_nodes`, `graph_edges`, and `graph_files`. Query, path, and explain read that index. They do not parse on the fly.

## Who calls it

A person or an agent once per repo, and again after a large edit. bd-neon Lookup shows an empty state with this command until the index exists. The build is not run on a timer.

## Invocation

```text
fabrik [-C repo] graph build [--json]
```

No language flag and no path argument. The repo root is the store's directory.

## Inputs

The filesystem under the repo root.

Skipped directories, anywhere in the tree: `.git`, `.fabrik`, `vendor`, `node_modules`, `dist`, `build`, `target`, `graphify-out`.

Parsed files, by extension:

| Extension | Lang | Grammar |
| --- | --- | --- |
| `.go` | `go` | Go |
| `.rs` | `rust` | Rust |
| `.ts` | `typescript` | TypeScript |
| `.tsx` | `tsx` | TSX |
| `.js` | `javascript` | TypeScript grammar |
| `.jsx` | `jsx` | TypeScript grammar |
| `.py` | `python` | Python |
| `.java` | `java` | Java |
| `.kt`, `.kts` | `kotlin` | Kotlin |
| `.cs` | `c_sharp` | C# |

Other files are ignored. A file over 1 MiB is skipped and recorded in `graph_files` with `ok` 0 and empty lang.

## Stored rows

Writes only to `.fabrik/graph.db`. No Dolt commit.

For each parsed file whose mtime differs from `graph_files`, or that is not in `graph_files`:

1. Parse with gotreesitter.
2. Delete nodes whose `path` is that file, and edges whose `src` or `dst` is one of those nodes.
3. Insert a `file:` node.
4. Insert a `sym:` node for each function, method, and type definition the grammar query returns. `line` is the 1-based start line.
5. Insert `contains` edges from the file node to those symbols.
6. Insert `imports` edges from the file node to another `file:` node when the import path resolves to a file in this build. Unresolved imports are kept with `dst` empty and `label` set to the import string.
7. Insert `calls` edges using import-aware resolution ([graph-explore.md](graph-explore.md)): `confidence=extracted` when callee resolves uniquely in scope; else `inferred` with `dst` empty and `label` = callee name, optional `candidates` in MCP responses only.
8. Upsert `graph_files` with the mtime, lang, and `ok` 1. `ok` is 0 when the tree has a parse error. Nodes from a partial tree are still stored.

After the walk, paths in `graph_files` that no longer exist on disk are deleted, along with their nodes and edges.

Symbol identity is `sym:<path>:<name>`. Two definitions with the same name in one file share one node. The line is the first definition.

## JSON result

```json
{
  "files": 42,
  "parsed": 3,
  "skipped": 1,
  "nodes": 400,
  "edges": 900
}
```

`files` is how many source files were considered. `parsed` is how many were reparsed. `skipped` counts oversized files and files with `ok` 0. `nodes` and `edges` are totals after the build, not the delta.

## Text result

```text
parsed 3 of 42 files, skipped 1
400 nodes, 900 edges
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| No store | 1 | `fabrik: no store (run fabrik init)` |
| Repo root unreadable | 1 | `fabrik: graph: …` |

A single file that fails to read is counted in `skipped` and does not fail the command.

## Examples

```text
fabrik graph build
fabrik graph build --json
```

## bd-neon

Lookup does not run build on a timer. The person may click **Build graph** / **Rebuild graph** in the pane or toolbar; the app runs `fabrik graph build --json` in the background. When the index is missing, the pane shows the build card (not only the CLI hint text).
