# Stack

## Monorepo

See [REPO.md](REPO.md). One Go binary in `cli/`. GUI in `gui/` (Rust). Spec in `v2/`.

## Dolt store (issues only)

| Piece | Library | Why |
| --- | --- | --- |
| Versioned SQL | `github.com/dolthub/driver` | Embedded Dolt. `database/sql`, auto-commit per write, push/pull on `refs/dolt/data`. |

Pin the driver to the Dolt **v2.2.0** line. Dolt 2.3.x can break `DOLT_RESET('--hard')`, which pull uses when a merge is abandoned.

Embedded Dolt links `facebook/zstd` through **cgo**. Release builds set `CGO_ENABLED=1` and use a C toolchain per target. On macOS, the driver may also need ICU (`brew install icu4c`; set `CGO_CPPFLAGS` / `CGO_LDFLAGS` to `icu4c` paths in CI).

Migrations: [MIGRATE.md](MIGRATE.md), `cli/internal/sqlmigrate/sql/dolt/`.

## Machine memory

| Piece | Library | Why |
| --- | --- | --- |
| Sessions + observations + FTS | `modernc.org/sqlite` | `~/.fabrik/memory.db` ([MEMORY.sql](MEMORY.sql)). WAL. Not in Dolt. |

Override path with `FABRIK_MEMORY_DIR`. Migrations: `sqlmigrate/sql/memory/`.

## Repo graph index

| Piece | Library | Why |
| --- | --- | --- |
| Code graph | `modernc.org/sqlite` | `.fabrik/graph.db` ([GRAPH.sql](GRAPH.sql)). |
| Code parsing | `github.com/odvcencio/gotreesitter` | **All** grammar blobs embedded (~20MB). v1 parses Go, Rust, TS, TSX, JS, JSX, Python only. |

## MCP

| Piece | Library |
| --- | --- |
| Stdio server | `github.com/modelcontextprotocol/go-sdk` |

`fabrik mcp` subcommand. [setup.md](setup.md) registers it; removes Engram MCP entries.

## Shell

| Piece | Library |
| --- | --- |
| CLI | `github.com/spf13/cobra` |
| Human UI | eframe / egui in `gui/` |

SQL is hand-written. Graph BFS and shortest path are small loops over edge rows.

## Subscribe

Loopback SSE: [listen.md](listen.md). `net/http` in stdlib.

## Do not use

| Library | Why not |
| --- | --- |
| Beads / `bd` (runtime) | v2 issues are native Dolt. |
| Engram / Graphify binaries | Replaced by `fabrik mcp` + backends. |
| Gorm, ent, sqlc | Raw SQL + sqlmigrate. |
| `mark3labs/mcp-go` | Prefer official MCP Go SDK for new code. |

## Processes

CLI: short-lived commands; `fabrik mcp` long-lived stdio. GUI: CLI JSON + optional listen SSE.
